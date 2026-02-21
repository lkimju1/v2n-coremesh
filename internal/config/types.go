package config

type App struct {
	WorkDir             string `yaml:"work_dir" json:"work_dir"`
	GeneratedXrayConfig string `yaml:"generated_xray_config" json:"generated_xray_config"`
}

type Xray struct {
	Bin        string   `yaml:"bin" json:"bin"`
	BaseConfig string   `yaml:"base_config" json:"base_config"`
	Args       []string `yaml:"args" json:"args"`
}

type Listen struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type Core struct {
	ProfileID   string   `yaml:"profile_id,omitempty" json:"profile_id,omitempty"`
	Name        string   `yaml:"name" json:"name"`
	Alias       string   `yaml:"alias" json:"alias"`
	Type        string   `yaml:"type" json:"type"`
	Bin         string   `yaml:"bin" json:"bin"`
	Config      string   `yaml:"config" json:"config"`
	Listen      Listen   `yaml:"listen" json:"listen"`
	Args        []string `yaml:"args" json:"args"`
	OutboundTag string   `yaml:"outbound_tag" json:"outbound_tag"`
	Active      bool     `yaml:"active" json:"active"`
}

type File struct {
	App              App    `yaml:"app" json:"app"`
	Xray             Xray   `yaml:"xray" json:"xray"`
	Cores            []Core `yaml:"cores" json:"cores"`
	RoutingRulesFile string `yaml:"routing_rules_file,omitempty" json:"routing_rules_file,omitempty"`
}

type RoutingRule struct {
	Name        string   `yaml:"name" json:"name"`
	Domain      []string `yaml:"domain" json:"domain"`
	OutboundTag string   `yaml:"outbound_tag" json:"outbound_tag"`
}

type Routing struct {
	Rules              []RoutingRule `yaml:"rules" json:"rules"`
	DefaultOutboundTag string        `yaml:"default_outbound_tag" json:"default_outbound_tag"`
}
