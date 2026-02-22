package xraygen

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func Generate(mainCfg *config.File, routingCfg *config.Routing, customRules []map[string]any) error {
	if routingCfg == nil {
		routingCfg = &config.Routing{}
	}

	b, err := os.ReadFile(mainCfg.Xray.BaseConfig)
	if err != nil {
		return fmt.Errorf("read xray base config: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("parse xray base config: %w", err)
	}

	outbounds := ensureArray(doc, "outbounds")
	existingTags := collectOutboundTags(outbounds)
	for _, c := range mainCfg.Cores {
		if c.Active {
			continue
		}
		tag := strings.TrimSpace(c.OutboundTag)
		if tag == "" {
			tag = strings.TrimSpace(c.Alias)
		}
		if tag == "" {
			continue
		}
		if _, exists := existingTags[tag]; exists {
			continue
		}
		outbounds = append(outbounds, map[string]any{
			"tag":      tag,
			"protocol": "socks",
			"settings": map[string]any{
				"servers": []any{map[string]any{
					"address": c.Listen.Host,
					"port":    c.Listen.Port,
				}},
			},
		})
		existingTags[tag] = struct{}{}
	}
	doc["outbounds"] = outbounds

	routing := ensureObject(doc, "routing")
	baseRules := ensureArrayFromObject(routing, "rules")
	rules := make([]any, 0, len(customRules)+len(routingCfg.Rules)+len(baseRules))
	for _, rule := range customRules {
		rules = append(rules, rule)
	}
	for _, r := range routingCfg.Rules {
		rules = append(rules, map[string]any{
			"type":        "field",
			"domain":      r.Domain,
			"outboundTag": r.OutboundTag,
		})
	}
	rules = append(rules, baseRules...)
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

func ensureArrayFromObject(doc map[string]any, key string) []any {
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

func collectOutboundTags(outbounds []any) map[string]struct{} {
	tags := make(map[string]struct{}, len(outbounds))
	for _, outbound := range outbounds {
		m, ok := outbound.(map[string]any)
		if !ok {
			continue
		}
		tag, ok := m["tag"].(string)
		if !ok {
			continue
		}
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		tags[tag] = struct{}{}
	}
	return tags
}
