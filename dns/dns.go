package dns

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

type Zone struct {
	Records map[string]string
}

type Server struct {
	Port       int
	DefaultTTL uint32
	Records    map[string]string
	Zones      map[string]Zone
}

func (s *Server) Start() error {
	addr := net.UDPAddr{
		Port: s.Port,
		IP:   net.ParseIP("0.0.0.0"),
	}
	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return fmt.Errorf("failed to start DNS server: %v", err)
	}
	defer conn.Close()

	log.Printf("[INFO] DNS server started on port %d", s.Port)
	buffer := make([]byte, 512)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("[ERROR] Failed to read UDP packet: %v", err)
			continue
		}

		log.Printf("[INFO] Received DNS request from %s:%d", clientAddr.IP, clientAddr.Port)

		go s.handleDNSRequest(buffer[:n], clientAddr, conn)
	}
}

func (s *Server) handleDNSRequest(data []byte, clientAddr *net.UDPAddr, conn *net.UDPConn) {
	request, err := parseDNSRequest(data)
	if err != nil {
		log.Printf("[ERROR] Failed to parse DNS request from %s: %v", clientAddr.IP, err)
		return
	}

	log.Printf("[INFO] Handling DNS request from %s:%d - ID: %d, Questions: %d",
		clientAddr.IP, clientAddr.Port, request.ID, len(request.Questions))

	for _, question := range request.Questions {
		log.Printf("[INFO] Question: Name=%s, Type=%d, Class=%d",
			question.Name, question.Type, question.Class)
	}

	response := createDNSResponse(request, s.resolveQuery)

	log.Printf("[INFO] Sending DNS response to %s:%d - ID: %d, Answers: %d",
		clientAddr.IP, clientAddr.Port, request.ID, len(request.Questions))

	conn.WriteToUDP(response, clientAddr)
}

func (s *Server) resolveQuery(name string) (string, uint32, error) {
	log.Printf("[INFO] Resolving query for %s", name)

	if ip, found := s.Records[name]; found {
		log.Printf("[INFO] Found global record: %s -> %s", name, ip)
		return ip, s.DefaultTTL, nil
	}

	for zoneName, zone := range s.Zones {
		if strings.HasSuffix(name, zoneName) {
			if ip, found := zone.Records[name]; found {
				log.Printf("[INFO] Found record in zone %s: %s -> %s", zoneName, name, ip)
				return ip, s.DefaultTTL, nil
			}
		}
	}

	log.Printf("[WARNING] Record not found for %s", name)
	return "", 0, errors.New("record not found")
}

func parseDNSRequest(data []byte) (*DNSRequest, error) {
	if len(data) < 12 {
		return nil, errors.New("invalid DNS packet")
	}

	request := &DNSRequest{}
	request.ID = binary.BigEndian.Uint16(data[:2])
	request.Flags = binary.BigEndian.Uint16(data[2:4])
	questionCount := binary.BigEndian.Uint16(data[4:6])

	offset := 12
	for i := 0; i < int(questionCount); i++ {
		qName, n := parseQName(data[offset:])
		offset += n
		qType := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2
		qClass := binary.BigEndian.Uint16(data[offset : offset+2])
		offset += 2

		request.Questions = append(request.Questions, DNSQuestion{
			Name:  qName,
			Type:  qType,
			Class: qClass,
		})
	}

	log.Printf("[INFO] Parsed DNS request - ID: %d, Questions: %d", request.ID, len(request.Questions))
	return request, nil
}

func parseQName(data []byte) (string, int) {
	var nameParts []string
	offset := 0

	for {
		length := int(data[offset])
		if length == 0 {
			break
		}
		offset++
		nameParts = append(nameParts, string(data[offset:offset+length]))
		offset += length
	}

	return strings.Join(nameParts, "."), offset + 1
}

func createDNSResponse(request *DNSRequest, resolver func(string) (string, uint32, error)) []byte {
	var buffer bytes.Buffer

	binary.Write(&buffer, binary.BigEndian, request.ID)
	binary.Write(&buffer, binary.BigEndian, uint16(0x8180))
	binary.Write(&buffer, binary.BigEndian, uint16(len(request.Questions)))
	binary.Write(&buffer, binary.BigEndian, uint16(len(request.Questions)))
	binary.Write(&buffer, binary.BigEndian, uint16(0))
	binary.Write(&buffer, binary.BigEndian, uint16(0))

	for _, question := range request.Questions {
		writeQName(&buffer, question.Name)
		binary.Write(&buffer, binary.BigEndian, question.Type)
		binary.Write(&buffer, binary.BigEndian, question.Class)

		ip, ttl, err := resolver(question.Name)
		if err == nil {
			writeQName(&buffer, question.Name)
			binary.Write(&buffer, binary.BigEndian, question.Type)
			binary.Write(&buffer, binary.BigEndian, question.Class)
			binary.Write(&buffer, binary.BigEndian, ttl) // TTL
			binary.Write(&buffer, binary.BigEndian, uint16(4))
			buffer.Write(net.ParseIP(ip).To4())
		} else {
			log.Printf("[WARNING] Failed to resolve query for %s", question.Name)
		}
	}

	return buffer.Bytes()
}

func writeQName(buffer *bytes.Buffer, name string) {
	parts := strings.Split(name, ".")
	for _, part := range parts {
		buffer.WriteByte(byte(len(part)))
		buffer.WriteString(part)
	}
	buffer.WriteByte(0)
}

type DNSRequest struct {
	ID        uint16
	Flags     uint16
	Questions []DNSQuestion
}

type DNSQuestion struct {
	Name  string
	Type  uint16
	Class uint16
}
