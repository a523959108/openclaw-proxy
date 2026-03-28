package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"

	"gopkg.in/yaml.v3"
)

// generateRandomPassword generates a secure random 16-character password
func generateRandomPassword() string {
	b := make([]byte, 12)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:                 "0.0.0.0:9090",
		Subscriptions:              []*Subscription{},
		Groups:                     []*Group{},
		Rules:                      []*Rule{},
		RuleStrings:                []string{},
		EnableAuth:                 false,
		Username:                   "admin",
		Password:                   generateRandomPassword(),
		EnableAutoSelect:           true,
		AutoSelectInterval:         30, // 30 minutes
		SubscriptionUpdateInterval: 24, // 24 hours
		SelectedGroup:              "auto",
		SelectionStrategy:          "latency", // default to latency-based selection
		EnableHTTPS:                false,
		DNS: &DNSConfig{
			Enable:         false,
			TrustedDNS:     []string{"223.5.5.5:53", "114.114.114.114:53", "8.8.8.8:53", "1.1.1.1:53"},
			CacheTTL:       30,
			Timeout:        5,
			CheckPollution: true,
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cfg := DefaultConfig()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// SaveConfig saves configuration to file
func SaveConfig(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
