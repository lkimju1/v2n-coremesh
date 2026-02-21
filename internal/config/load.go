package config

import (
	"errors"
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

func LoadCustomRules(path string) ([]map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read custom rules: %w", err)
	}
	var rules []map[string]any
	if err := yaml.Unmarshal(b, &rules); err != nil {
		return nil, fmt.Errorf("parse custom rules: %w", err)
	}
	return rules, nil
}
