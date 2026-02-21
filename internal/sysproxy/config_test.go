package sysproxy

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestDetectProxyEndpoint(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "xray.generated.json")
	content := `{
  "inbounds": [
    {"tag":"in-socks","protocol":"socks","listen":"0.0.0.0","port":10808},
    {"tag":"in-http","protocol":"http","listen":"","port":10809}
  ]
}`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	ep, err := DetectProxyEndpoint(cfgPath)
	if err != nil {
		t.Fatalf("detect endpoint: %v", err)
	}
	if ep.Protocol != "http" || ep.Host != "127.0.0.1" || ep.Port != 10809 {
		t.Fatalf("unexpected endpoint: %#v", ep)
	}
}

func TestMergeBypass(t *testing.T) {
	existing := "localhost;example.com;127.*"
	required := "localhost;10.*;127.*;192.168.*"

	got := mergeBypass(existing, required)
	normalized := sortedLowerSet(got)
	expect := []string{"10.*", "127.*", "192.168.*", "example.com", "localhost"}
	if len(normalized) != len(expect) {
		t.Fatalf("unexpected merged size: got=%v expect=%v", normalized, expect)
	}
	for i := range expect {
		if normalized[i] != expect[i] {
			t.Fatalf("unexpected merged list: got=%v expect=%v", normalized, expect)
		}
	}
}

func sortedLowerSet(v string) []string {
	set := make(map[string]struct{})
	for _, item := range splitBypass(v) {
		set[strings.ToLower(item)] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
