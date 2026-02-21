package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func Main(cfg *config.File, routing *config.Routing) error {
	if routing == nil {
		routing = &config.Routing{}
	}
	if cfg.App.GeneratedXrayConfig == "" {
		return fmt.Errorf("app.generated_xray_config is required")
	}
	if cfg.Xray.Bin == "" || cfg.Xray.BaseConfig == "" {
		return fmt.Errorf("xray.bin and xray.base_config are required")
	}
	if err := checkFile(cfg.Xray.Bin, "xray.bin"); err != nil {
		return err
	}
	if err := checkFile(cfg.Xray.BaseConfig, "xray.base_config"); err != nil {
		return err
	}
	tagSet := make(map[string]struct{})
	listenSet := make(map[string]struct{})
	for i, c := range cfg.Cores {
		idx := fmt.Sprintf("cores[%d]", i)
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("%s.name is required", idx)
		}
		tag := strings.TrimSpace(c.OutboundTag)
		if tag == "" {
			tag = strings.TrimSpace(c.Alias)
		}
		if tag == "" {
			return fmt.Errorf("%s.outbound_tag or %s.alias is required", idx, idx)
		}
		if err := checkFile(c.Bin, idx+".bin"); err != nil {
			return err
		}
		if err := checkFile(c.Config, idx+".config"); err != nil {
			return err
		}
		if _, ok := tagSet[tag]; ok {
			return fmt.Errorf("duplicate outbound_tag: %s", tag)
		}
		tagSet[tag] = struct{}{}
		key := fmt.Sprintf("%s:%d", c.Listen.Host, c.Listen.Port)
		if _, ok := listenSet[key]; ok {
			return fmt.Errorf("duplicate listen endpoint: %s", key)
		}
		listenSet[key] = struct{}{}
	}
	for _, r := range routing.Rules {
		if isBuiltinOutboundTag(r.OutboundTag) {
			continue
		}
		if _, ok := tagSet[r.OutboundTag]; !ok {
			return fmt.Errorf("routing rule %q references unknown outbound_tag %q", r.Name, r.OutboundTag)
		}
	}
	if routing.DefaultOutboundTag != "" {
		if !isBuiltinOutboundTag(routing.DefaultOutboundTag) {
			if _, ok := tagSet[routing.DefaultOutboundTag]; !ok {
				return fmt.Errorf("default_outbound_tag %q not found", routing.DefaultOutboundTag)
			}
		}
	}
	return os.MkdirAll(filepath.Dir(cfg.App.GeneratedXrayConfig), 0o755)
}

func ForRun(cfg *config.File) error {
	if cfg.App.GeneratedXrayConfig == "" {
		return fmt.Errorf("app.generated_xray_config is required")
	}
	if cfg.Xray.Bin == "" {
		return fmt.Errorf("xray.bin is required")
	}
	if err := checkFile(cfg.Xray.Bin, "xray.bin"); err != nil {
		return err
	}
	if err := checkFile(cfg.App.GeneratedXrayConfig, "app.generated_xray_config"); err != nil {
		return err
	}
	for i, c := range cfg.Cores {
		idx := fmt.Sprintf("cores[%d]", i)
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("%s.name is required", idx)
		}
		if err := checkFile(c.Bin, idx+".bin"); err != nil {
			return err
		}
		if err := checkFile(c.Config, idx+".config"); err != nil {
			return err
		}
	}
	return nil
}

func checkFile(path, field string) error {
	if path == "" {
		return fmt.Errorf("%s is required", field)
	}
	st, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s invalid: %w", field, err)
	}
	if st.IsDir() {
		return fmt.Errorf("%s points to directory: %s", field, path)
	}
	return nil
}

func isBuiltinOutboundTag(tag string) bool {
	switch strings.ToLower(strings.TrimSpace(tag)) {
	case "direct", "block", "proxy":
		return true
	default:
		return false
	}
}
