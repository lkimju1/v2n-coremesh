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
	base := `{"outbounds":[{"protocol":"freedom","tag":"direct"}]}`
	if err := os.WriteFile(basePath, []byte(base), 0o644); err != nil {
		t.Fatal(err)
	}

	mainCfg := &config.File{
		App:  config.App{GeneratedXrayConfig: outPath},
		Xray: config.Xray{BaseConfig: basePath},
		Cores: []config.Core{{
			Name: "c1", OutboundTag: "core-a", Listen: config.Listen{Host: "127.0.0.1", Port: 11080},
		}},
	}
	routingCfg := &config.Routing{
		Rules:              []config.RoutingRule{{Name: "r1", Domain: []string{"domain:example.com"}, OutboundTag: "core-a"}},
		DefaultOutboundTag: "direct",
	}

	if err := Generate(mainCfg, routingCfg); err != nil {
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
	if !ok || len(outbounds) != 2 {
		t.Fatalf("unexpected outbounds: %#v", doc["outbounds"])
	}
	routing, ok := doc["routing"].(map[string]any)
	if !ok {
		t.Fatalf("routing missing: %#v", doc["routing"])
	}
	rules, ok := routing["rules"].([]any)
	if !ok || len(rules) != 2 {
		t.Fatalf("unexpected rules: %#v", routing["rules"])
	}
}
