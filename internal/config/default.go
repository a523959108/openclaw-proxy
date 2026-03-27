package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

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
		Password:                   "admin",
		EnableAutoSelect:           true,
		AutoSelectInterval:         30, // 30 minutes
		SubscriptionUpdateInterval: 24, // 24 hours
		SelectedGroup:              "auto",
		SelectionStrategy:          "latency", // default to latency-based selection
		EnableHTTPS:                false,
		DNS: &DNSConfig{
			Enable:          false,
			TrustedDNS:      []string{"8.8.8.8:53", "1.1.1.1:53"},
			CacheTTL:        30,
			Timeout:         5,
			CheckPollution:  true,
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

	return os.WriteFile(path, data, 0644)
}
