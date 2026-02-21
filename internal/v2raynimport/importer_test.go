package v2raynimport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeXrayConfig(t *testing.T) {
	tmp := t.TempDir()

	valid := filepath.Join(tmp, "valid.json")
	if err := os.WriteFile(valid, []byte(`{
  "inbounds":[{"protocol":"mixed","listen":"127.0.0.1","port":10808}],
  "outbounds":[{"tag":"proxy","protocol":"socks"}],
  "routing":{"rules":[{"type":"field","outboundTag":"proxy"}]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err := looksLikeXrayConfig(valid)
	if err != nil {
		t.Fatalf("looksLikeXrayConfig(valid) error: %v", err)
	}
	if !ok {
		t.Fatal("valid xray config should be recognized")
	}

	invalid := filepath.Join(tmp, "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{"listen":"socks://127.0.0.1:1080","proxy":"https://example.com"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ok, err = looksLikeXrayConfig(invalid)
	if err != nil {
		t.Fatalf("looksLikeXrayConfig(invalid) error: %v", err)
	}
	if ok {
		t.Fatal("invalid xray config should not be recognized")
	}
}

func TestDetectXrayBaseConfigPrefersConfigPre(t *testing.T) {
	tmp := t.TempDir()
	binConfigs := filepath.Join(tmp, "binConfigs")
	if err := os.MkdirAll(binConfigs, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(binConfigs, "config.json"), []byte(`{"listen":"socks://127.0.0.1:1080"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binConfigs, "configPre.json"), []byte(`{
  "inbounds":[{"protocol":"mixed","listen":"127.0.0.1","port":10808}],
  "outbounds":[{"tag":"proxy","protocol":"socks"}],
  "routing":{"rules":[{"type":"field","outboundTag":"proxy"}]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := detectXrayBaseConfig(tmp)
	if err != nil {
		t.Fatalf("detectXrayBaseConfig error: %v", err)
	}
	expect := filepath.Join(binConfigs, "configPre.json")
	if got != expect {
		t.Fatalf("expected %s, got %s", expect, got)
	}
}

func TestDetectXrayBaseConfigErrorWhenConfigPreMissing(t *testing.T) {
	tmp := t.TempDir()
	binConfigs := filepath.Join(tmp, "binConfigs")
	if err := os.MkdirAll(binConfigs, 0o755); err != nil {
		t.Fatal(err)
	}
	// Even if config.json exists, configPre.json is mandatory.
	if err := os.WriteFile(filepath.Join(binConfigs, "config.json"), []byte(`{
  "inbounds":[{"protocol":"mixed","listen":"127.0.0.1","port":10808}],
  "outbounds":[{"tag":"proxy","protocol":"socks"}],
  "routing":{"rules":[{"type":"field","outboundTag":"proxy"}]}
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := detectXrayBaseConfig(tmp); err == nil {
		t.Fatal("expected error when configPre.json is missing")
	}
}
