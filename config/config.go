package config

import (
	"encoding/json"
	"os"
)

type Zone struct {
	Records map[string]string `json:"records"`
}

type Config struct {
	DNS struct {
		Port       int               `json:"port"`
		DefaultTTL uint32            `json:"default_ttl"`
		Records    map[string]string `json:"records"`
		Zones      map[string]Zone   `json:"zones"`
	} `json:"dns"`
	WebUI struct {
		Enabled bool `json:"enabled"`
		Port    int  `json:"port"`
	} `json:"web_ui"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
