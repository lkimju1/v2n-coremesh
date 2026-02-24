package bindmode

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

const bindAllHost = "0.0.0.0"

func PrepareForBindAll(cfg *config.File, confDir string) (*config.File, error) {
	if cfg == nil {
		return nil, fmt.Errorf("nil config")
	}
	out := cloneConfig(cfg)
	runtimeDir := filepath.Join(confDir, "runtime_bind_all")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, fmt.Errorf("create bind-all runtime dir: %w", err)
	}

	xrayOutPath := filepath.Join(runtimeDir, "xray.generated.bindall.json")
	if err := patchListenConfigFile(out.App.GeneratedXrayConfig, xrayOutPath, bindAllHost, true); err != nil {
		return nil, fmt.Errorf("patch xray generated config: %w", err)
	}
	out.App.GeneratedXrayConfig = xrayOutPath

	for i := range out.Cores {
		coreName := sanitizeName(out.Cores[i].Alias)
		if coreName == "" {
			coreName = sanitizeName(out.Cores[i].Name)
		}
		if coreName == "" {
			coreName = fmt.Sprintf("core-%d", i+1)
		}
		coreOutPath := filepath.Join(runtimeDir, fmt.Sprintf("%02d-%s.bindall.json", i+1, coreName))
		if err := patchListenConfigFile(out.Cores[i].Config, coreOutPath, bindAllHost, false); err != nil {
			return nil, fmt.Errorf("patch core %q config: %w", out.Cores[i].Name, err)
		}
		out.Cores[i].Config = coreOutPath
		out.Cores[i].Listen.Host = bindAllHost
	}

	return out, nil
}

func cloneConfig(cfg *config.File) *config.File {
	cp := *cfg
	cp.Xray.Args = append([]string(nil), cfg.Xray.Args...)
	cp.Cores = append([]config.Core(nil), cfg.Cores...)
	for i := range cp.Cores {
		cp.Cores[i].Args = append([]string(nil), cfg.Cores[i].Args...)
	}
	return &cp
}

func patchListenConfigFile(src, dst, host string, ensureInboundListen bool) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source config: %w", err)
	}

	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return fmt.Errorf("parse json config %s: %w", src, err)
	}

	updated, _ := rewriteListenAny(doc, host)
	if ensureInboundListen {
		updated, _ = ensureInboundListenHost(updated, host)
	}

	out, err := json.MarshalIndent(updated, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal patched config: %w", err)
	}
	if err := os.WriteFile(dst, out, 0o644); err != nil {
		return fmt.Errorf("write patched config: %w", err)
	}
	return nil
}

func ensureInboundListenHost(v any, host string) (any, bool) {
	m, ok := v.(map[string]any)
	if !ok {
		return v, false
	}
	inbounds, ok := m["inbounds"].([]any)
	if !ok {
		return v, false
	}
	changed := false
	for i, raw := range inbounds {
		inbound, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		listenVal, hasListen := inbound["listen"]
		if !hasListen {
			inbound["listen"] = host
			inbounds[i] = inbound
			changed = true
			continue
		}
		rewritten, itemChanged := rewriteListenValue(listenVal, host)
		if itemChanged {
			inbound["listen"] = rewritten
			inbounds[i] = inbound
			changed = true
		}
	}
	if changed {
		m["inbounds"] = inbounds
		return m, true
	}
	return v, false
}

func rewriteListenAny(v any, host string) (any, bool) {
	switch val := v.(type) {
	case map[string]any:
		changed := false
		for k, raw := range val {
			if strings.EqualFold(k, "listen") {
				rewritten, itemChanged := rewriteListenValue(raw, host)
				if itemChanged {
					val[k] = rewritten
					changed = true
				}
				continue
			}
			rewritten, itemChanged := rewriteListenAny(raw, host)
			if itemChanged {
				val[k] = rewritten
				changed = true
			}
		}
		return val, changed
	case []any:
		changed := false
		out := make([]any, len(val))
		copy(out, val)
		for i, item := range out {
			rewritten, itemChanged := rewriteListenAny(item, host)
			if itemChanged {
				out[i] = rewritten
				changed = true
			}
		}
		return out, changed
	default:
		return v, false
	}
}

func rewriteListenValue(v any, host string) (any, bool) {
	switch val := v.(type) {
	case string:
		rewritten, changed := rewriteListenString(val, host)
		if !changed {
			return v, false
		}
		return rewritten, true
	case []any:
		changed := false
		out := make([]any, len(val))
		copy(out, val)
		for i, item := range out {
			s, ok := item.(string)
			if !ok {
				continue
			}
			rewritten, itemChanged := rewriteListenString(s, host)
			if itemChanged {
				out[i] = rewritten
				changed = true
			}
		}
		if !changed {
			return v, false
		}
		return out, true
	default:
		return v, false
	}
}

func rewriteListenString(s, host string) (string, bool) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return host, true
	}

	if strings.Contains(trimmed, "://") {
		u, err := url.Parse(trimmed)
		if err != nil {
			return s, false
		}
		port := u.Port()
		if port != "" {
			u.Host = net.JoinHostPort(host, port)
		} else {
			u.Host = host
		}
		rewritten := u.String()
		return rewritten, rewritten != s
	}

	_, port, err := net.SplitHostPort(trimmed)
	if err == nil && port != "" {
		rewritten := net.JoinHostPort(host, port)
		return rewritten, rewritten != s
	}

	lower := strings.ToLower(trimmed)
	if lower == "localhost" || net.ParseIP(trimmed) != nil {
		return host, trimmed != host
	}
	return s, false
}

func sanitizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		".", "-",
		"_", "-",
	)
	s = replacer.Replace(s)
	builder := strings.Builder{}
	builder.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	return strings.Trim(builder.String(), "-")
}
