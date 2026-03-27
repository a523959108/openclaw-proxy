package config

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ListenAddr:         "0.0.0.0:9090",
		Subscriptions:      []*Subscription{},
		Rules:              []*Rule{},
		EnableAutoSelect:   true,
		AutoSelectInterval: 30, // 30 minutes
		SelectedGroup:      "auto",
	}
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	// Implementation will be added
	cfg := DefaultConfig()
	return cfg, nil
}
