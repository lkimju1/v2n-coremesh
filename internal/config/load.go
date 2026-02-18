package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadMain(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read main config: %w", err)
	}
	var cfg File
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse main config: %w", err)
	}
	return &cfg, nil
}

func LoadRouting(path string) (*Routing, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read routing config: %w", err)
	}
	var cfg Routing
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse routing config: %w", err)
	}
	return &cfg, nil
}
