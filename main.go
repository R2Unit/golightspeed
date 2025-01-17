package main

import (
	"log"
	"os"

	"github.com/r2unit/golightspeed/config"
	"github.com/r2unit/golightspeed/dns"
	"github.com/r2unit/golightspeed/web"
)

func main() {
	configPath := os.Getenv("CONFIG_FILE")
	if configPath == "" {
		log.Fatal("CONFIG_FILE environment variable is not set")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	dnsZones := make(map[string]dns.Zone)
	for zoneName, zone := range cfg.DNS.Zones {
		dnsZones[zoneName] = dns.Zone{
			Records: zone.Records,
		}
	}

	webZones := make(map[string]map[string]string)
	for zoneName, zone := range cfg.DNS.Zones {
		webZones[zoneName] = zone.Records
	}

	dnsServer := &dns.Server{
		Port:       cfg.DNS.Port,
		DefaultTTL: cfg.DNS.DefaultTTL,
		Records:    cfg.DNS.Records,
		Zones:      dnsZones,
	}

	go func() {
		if err := dnsServer.Start(); err != nil {
			log.Fatalf("DNS server failed: %v", err)
		}
	}()

	if cfg.WebUI.Enabled {
		go func() {
			web.StartWebUI(cfg.WebUI.Port, cfg.DNS.Records, webZones)
		}()
	}

	select {}
}
