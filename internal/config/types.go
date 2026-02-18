package config

type App struct {
	WorkDir             string `yaml:"work_dir"`
	GeneratedXrayConfig string `yaml:"generated_xray_config"`
}

type Xray struct {
	Bin        string   `yaml:"bin"`
	BaseConfig string   `yaml:"base_config"`
	Args       []string `yaml:"args"`
}

type Listen struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type Core struct {
	Name        string   `yaml:"name"`
	Type        string   `yaml:"type"`
	Bin         string   `yaml:"bin"`
	Config      string   `yaml:"config"`
	Listen      Listen   `yaml:"listen"`
	Args        []string `yaml:"args"`
	OutboundTag string   `yaml:"outbound_tag"`
}

type File struct {
	App              App    `yaml:"app"`
	Xray             Xray   `yaml:"xray"`
	Cores            []Core `yaml:"cores"`
	RoutingRulesFile string `yaml:"routing_rules_file"`
}

type RoutingRule struct {
	Name        string   `yaml:"name"`
	Domain      []string `yaml:"domain"`
	OutboundTag string   `yaml:"outbound_tag"`
}

type Routing struct {
	Rules              []RoutingRule `yaml:"rules"`
	DefaultOutboundTag string        `yaml:"default_outbound_tag"`
}
