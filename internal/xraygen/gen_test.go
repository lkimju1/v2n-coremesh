package xraygen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func TestGenerateInjectsOutboundsAndRules(t *testing.T) {
	tmp := t.TempDir()
	basePath := filepath.Join(tmp, "base.json")
	outPath := filepath.Join(tmp, "out.json")
	base := `{
  "outbounds":[
    {"protocol":"freedom","tag":"direct"},
    {"protocol":"socks","tag":"proxy"}
  ],
  "routing":{
    "rules":[
      {"type":"field","domain":["domain:old.example.com"],"outboundTag":"proxy"}
    ]
  }
}`
	if err := os.WriteFile(basePath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	mainCfg := &config.File{
		App:  config.App{GeneratedXrayConfig: outPath},
		Xray: config.Xray{BaseConfig: basePath},
		Cores: []config.Core{
			{
				Name: "active", Alias: "active", OutboundTag: "active",
				Listen: config.Listen{Host: "127.0.0.1", Port: 10001}, Active: true,
			},
			{
				Name: "c1", Alias: "core-a", OutboundTag: "core-a",
				Listen: config.Listen{Host: "127.0.0.1", Port: 11080},
			},
		},
	}
	routingCfg := &config.Routing{
		Rules: []config.RoutingRule{{Name: "r1", Domain: []string{"domain:example.com"}, OutboundTag: "core-a"}},
	}
	customRules := []map[string]any{
		{
			"type":        "field",
			"domain":      []string{"domain:custom.example.com"},
			"outboundTag": "core-a",
		},
	}

	if err := Generate(mainCfg, routingCfg, customRules); err != nil {
		t.Fatalf("generate error: %v", err)
	}

	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	outbounds, ok := doc["outbounds"].([]any)
	if !ok || len(outbounds) != 3 {
		t.Fatalf("unexpected outbounds: %#v", doc["outbounds"])
	}
	if hasTag(outbounds, "active") {
		t.Fatalf("active core should not be injected when proxy outbound exists: %#v", outbounds)
	}
	if !hasTag(outbounds, "core-a") {
		t.Fatalf("core-a outbound missing: %#v", outbounds)
	}

	routing, ok := doc["routing"].(map[string]any)
	if !ok {
		t.Fatalf("routing missing: %#v", doc["routing"])
	}
	rules, ok := routing["rules"].([]any)
	if !ok || len(rules) != 3 {
		t.Fatalf("unexpected rules: %#v", routing["rules"])
	}
	first := rules[0].(map[string]any)
	if first["outboundTag"] != "core-a" {
		t.Fatalf("first rule should be custom rule: %#v", first)
	}
	for _, rule := range rules {
		m, ok := rule.(map[string]any)
		if !ok {
			continue
		}
		if m["network"] == "tcp,udp" {
			t.Fatalf("unexpected catch-all rule appended: %#v", m)
		}
	}
}

func hasTag(outbounds []any, tag string) bool {
	for _, outbound := range outbounds {
		m, ok := outbound.(map[string]any)
		if !ok {
			continue
		}
		if v, ok := m["tag"].(string); ok && v == tag {
			return true
		}
	}
	return false
}
