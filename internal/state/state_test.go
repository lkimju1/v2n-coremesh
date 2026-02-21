package state

import (
	"path/filepath"
	"testing"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	cfg := &config.File{
		App:  config.App{GeneratedXrayConfig: filepath.Join(tmp, "xray.generated.json")},
		Xray: config.Xray{Bin: "/tmp/xray", Args: []string{"run", "-c", "{{config}}"}},
		Cores: []config.Core{
			{Name: "c1", Alias: "c1", OutboundTag: "c1", Bin: "/tmp/core", Config: "/tmp/core.json"},
		},
	}
	stateFile := New("/tmp/v2rayN", cfg)
	if err := Save(tmp, stateFile); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(tmp)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Version != CurrentVersion {
		t.Fatalf("unexpected version: %d", loaded.Version)
	}
	if loaded.Config.App.GeneratedXrayConfig != cfg.App.GeneratedXrayConfig {
		t.Fatalf("unexpected generated config: %s", loaded.Config.App.GeneratedXrayConfig)
	}
	if len(loaded.Config.Cores) != 1 || loaded.Config.Cores[0].Alias != "c1" {
		t.Fatalf("unexpected cores: %#v", loaded.Config.Cores)
	}
}
