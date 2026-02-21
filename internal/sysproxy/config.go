package sysproxy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const RequiredBypassList = "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*;127.0.0.1"

type InboundEndpoint struct {
	Protocol string
	Host     string
	Port     int
}

type xrayConfig struct {
	Inbounds []xrayInbound `json:"inbounds"`
}

type xrayInbound struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"`
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
}

func DetectProxyEndpoint(xrayConfigPath string) (*InboundEndpoint, error) {
	b, err := os.ReadFile(xrayConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read xray config: %w", err)
	}

	var cfg xrayConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse xray config: %w", err)
	}
	if len(cfg.Inbounds) == 0 {
		return nil, fmt.Errorf("xray config has no inbounds")
	}

	prefer := []string{"http", "mixed", "socks"}
	for _, protocol := range prefer {
		for _, inbound := range cfg.Inbounds {
			if !strings.EqualFold(strings.TrimSpace(inbound.Protocol), protocol) {
				continue
			}
			if inbound.Port <= 0 {
				continue
			}
			return &InboundEndpoint{
				Protocol: protocol,
				Host:     normalizeHost(inbound.Listen),
				Port:     inbound.Port,
			}, nil
		}
	}

	for _, inbound := range cfg.Inbounds {
		if inbound.Port <= 0 {
			continue
		}
		return &InboundEndpoint{
			Protocol: strings.ToLower(strings.TrimSpace(inbound.Protocol)),
			Host:     normalizeHost(inbound.Listen),
			Port:     inbound.Port,
		}, nil
	}
	return nil, fmt.Errorf("xray config has no valid inbound endpoint")
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	switch host {
	case "", "0.0.0.0", "::", "::0":
		return "127.0.0.1"
	default:
		return host
	}
}

func mergeBypass(existing, required string) string {
	seen := make(map[string]struct{})
	out := make([]string, 0)

	appendUnique := func(list string) {
		for _, item := range splitBypass(list) {
			key := strings.ToLower(item)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, item)
		}
	}
	appendUnique(existing)
	appendUnique(required)
	return strings.Join(out, ";")
}

func splitBypass(v string) []string {
	f := func(r rune) bool {
		return r == ';' || r == ','
	}
	raw := strings.FieldsFunc(v, f)
	out := make([]string, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
