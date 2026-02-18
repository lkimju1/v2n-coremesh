package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func TestMainSuccess(t *testing.T) {
	tmp := t.TempDir()
	xrayBin := touchFile(t, tmp, "xray")
	xrayBase := touchFile(t, tmp, "xray.base.json")
	coreBin := touchFile(t, tmp, "core")
	coreCfg := touchFile(t, tmp, "core.json")

	cfg := &config.File{
		App:  config.App{GeneratedXrayConfig: filepath.Join(tmp, "runtime", "xray.generated.json")},
		Xray: config.Xray{Bin: xrayBin, BaseConfig: xrayBase},
		Cores: []config.Core{{
			Name: "c1", Bin: coreBin, Config: coreCfg,
			Listen: config.Listen{Host: "127.0.0.1", Port: 10001}, OutboundTag: "o1",
		}},
	}
	routing := &config.Routing{Rules: []config.RoutingRule{{Name: "r1", OutboundTag: "o1"}}, DefaultOutboundTag: "direct"}

	if err := Main(cfg, routing); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMainDuplicatePort(t *testing.T) {
	tmp := t.TempDir()
	xrayBin := touchFile(t, tmp, "xray")
	xrayBase := touchFile(t, tmp, "xray.base.json")
	coreBin := touchFile(t, tmp, "core")
	coreCfg := touchFile(t, tmp, "core.json")

	cfg := &config.File{
		App:  config.App{GeneratedXrayConfig: filepath.Join(tmp, "runtime", "xray.generated.json")},
		Xray: config.Xray{Bin: xrayBin, BaseConfig: xrayBase},
		Cores: []config.Core{
			{Name: "c1", Bin: coreBin, Config: coreCfg, Listen: config.Listen{Host: "127.0.0.1", Port: 10001}, OutboundTag: "o1"},
			{Name: "c2", Bin: coreBin, Config: coreCfg, Listen: config.Listen{Host: "127.0.0.1", Port: 10001}, OutboundTag: "o2"},
		},
	}
	routing := &config.Routing{}

	if err := Main(cfg, routing); err == nil {
		t.Fatal("expected duplicate endpoint error")
	}
}

func touchFile(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write file %s: %v", p, err)
	}
	return p
}
