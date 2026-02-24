package bindmode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func TestPrepareForBindAll(t *testing.T) {
	tmp := t.TempDir()
	xrayPath := filepath.Join(tmp, "xray.generated.json")
	corePath := filepath.Join(tmp, "core.json")

	writeJSONFile(t, xrayPath, `{
  "inbounds": [
    {"tag":"in-socks","listen":"127.0.0.1","port":10808},
    {"tag":"in-http","port":10809}
  ],
  "outbounds": [{"tag":"proxy","protocol":"socks"}],
  "routing": {"rules":[]}
}`)
	writeJSONFile(t, corePath, `{
  "inbounds":[{"listen":"127.0.0.1","port":1081}],
  "listen":"socks://127.0.0.1:1081",
  "nested":{"listen":"127.0.0.1:1082"}
}`)

	cfg := &config.File{
		App:  config.App{GeneratedXrayConfig: xrayPath},
		Xray: config.Xray{Bin: "/tmp/xray", Args: []string{"run", "-c", "{{config}}"}},
		Cores: []config.Core{
			{
				Name:   "core-a",
				Alias:  "core-a",
				Config: corePath,
				Listen: config.Listen{Host: "127.0.0.1", Port: 1081},
			},
		},
	}

	out, err := PrepareForBindAll(cfg, tmp)
	if err != nil {
		t.Fatalf("prepare bind-all failed: %v", err)
	}
	if out.App.GeneratedXrayConfig == xrayPath {
		t.Fatal("xray config path should be rewritten")
	}
	if out.Cores[0].Config == corePath {
		t.Fatal("core config path should be rewritten")
	}
	if out.Cores[0].Listen.Host != bindAllHost {
		t.Fatalf("core listen host not rewritten: %#v", out.Cores[0].Listen)
	}
	if cfg.Cores[0].Config != corePath {
		t.Fatal("input config should not be mutated")
	}

	xrayDoc := readJSONDoc(t, out.App.GeneratedXrayConfig)
	inbounds, ok := xrayDoc["inbounds"].([]any)
	if !ok || len(inbounds) != 2 {
		t.Fatalf("unexpected xray inbounds: %#v", xrayDoc["inbounds"])
	}
	for _, inbound := range inbounds {
		m := inbound.(map[string]any)
		if m["listen"] != bindAllHost {
			t.Fatalf("xray inbound listen should be %s: %#v", bindAllHost, m)
		}
	}

	coreDoc := readJSONDoc(t, out.Cores[0].Config)
	if coreDoc["listen"] != "socks://0.0.0.0:1081" {
		t.Fatalf("unexpected core top-level listen: %#v", coreDoc["listen"])
	}
	nested := coreDoc["nested"].(map[string]any)
	if nested["listen"] != "0.0.0.0:1082" {
		t.Fatalf("unexpected core nested listen: %#v", nested["listen"])
	}
}

func TestPrepareForBindAllRejectsNonJSONCoreConfig(t *testing.T) {
	tmp := t.TempDir()
	xrayPath := filepath.Join(tmp, "xray.generated.json")
	corePath := filepath.Join(tmp, "core.txt")

	writeJSONFile(t, xrayPath, `{
  "inbounds": [{"tag":"in-socks","listen":"127.0.0.1","port":10808}],
  "outbounds": [{"tag":"proxy","protocol":"socks"}],
  "routing": {"rules":[]}
}`)
	if err := os.WriteFile(corePath, []byte("not-json"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.File{
		App:  config.App{GeneratedXrayConfig: xrayPath},
		Xray: config.Xray{Bin: "/tmp/xray", Args: []string{"run", "-c", "{{config}}"}},
		Cores: []config.Core{
			{
				Name:   "core-a",
				Alias:  "core-a",
				Config: corePath,
				Listen: config.Listen{Host: "127.0.0.1", Port: 1081},
			},
		},
	}

	if _, err := PrepareForBindAll(cfg, tmp); err == nil {
		t.Fatal("expected error for non-json core config")
	}
}

func writeJSONFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readJSONDoc(t *testing.T, path string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}
