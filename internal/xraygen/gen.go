package xraygen

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func Generate(mainCfg *config.File, routingCfg *config.Routing) error {
	b, err := os.ReadFile(mainCfg.Xray.BaseConfig)
	if err != nil {
		return fmt.Errorf("read xray base config: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("parse xray base config: %w", err)
	}

	outbounds := ensureArray(doc, "outbounds")
	for _, c := range mainCfg.Cores {
		outbounds = append(outbounds, map[string]any{
			"tag":      c.OutboundTag,
			"protocol": "socks",
			"settings": map[string]any{
				"servers": []any{map[string]any{
					"address": c.Listen.Host,
					"port":    c.Listen.Port,
				}},
			},
		})
	}
	doc["outbounds"] = outbounds

	routing := ensureObject(doc, "routing")
	rules := make([]any, 0, len(routingCfg.Rules)+1)
	for _, r := range routingCfg.Rules {
		rules = append(rules, map[string]any{
			"type":        "field",
			"domain":      r.Domain,
			"outboundTag": r.OutboundTag,
		})
	}
	if routingCfg.DefaultOutboundTag != "" {
		rules = append(rules, map[string]any{
			"type":        "field",
			"network":     "tcp,udp",
			"outboundTag": routingCfg.DefaultOutboundTag,
		})
	}
	routing["rules"] = rules
	doc["routing"] = routing

	result, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal generated xray config: %w", err)
	}
	if err := os.WriteFile(mainCfg.App.GeneratedXrayConfig, result, 0o644); err != nil {
		return fmt.Errorf("write generated xray config: %w", err)
	}
	return nil
}

func ensureArray(doc map[string]any, key string) []any {
	if v, ok := doc[key]; ok {
		if vv, ok := v.([]any); ok {
			return vv
		}
	}
	return []any{}
}

func ensureObject(doc map[string]any, key string) map[string]any {
	if v, ok := doc[key]; ok {
		if vv, ok := v.(map[string]any); ok {
			return vv
		}
	}
	return map[string]any{}
}
