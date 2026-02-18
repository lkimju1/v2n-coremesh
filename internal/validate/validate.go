package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

func Main(cfg *config.File, routing *config.Routing) error {
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
	nameSet := make(map[string]struct{})
	tagSet := make(map[string]struct{})
	listenSet := make(map[string]struct{})
	for i, c := range cfg.Cores {
		idx := fmt.Sprintf("cores[%d]", i)
		if c.Name == "" || c.OutboundTag == "" {
			return fmt.Errorf("%s.name and %s.outbound_tag are required", idx, idx)
		}
		if err := checkFile(c.Bin, idx+".bin"); err != nil {
			return err
		}
		if err := checkFile(c.Config, idx+".config"); err != nil {
			return err
		}
		if _, ok := nameSet[c.Name]; ok {
			return fmt.Errorf("duplicate core name: %s", c.Name)
		}
		nameSet[c.Name] = struct{}{}
		if _, ok := tagSet[c.OutboundTag]; ok {
			return fmt.Errorf("duplicate outbound_tag: %s", c.OutboundTag)
		}
		tagSet[c.OutboundTag] = struct{}{}
		key := fmt.Sprintf("%s:%d", c.Listen.Host, c.Listen.Port)
		if _, ok := listenSet[key]; ok {
			return fmt.Errorf("duplicate listen endpoint: %s", key)
		}
		listenSet[key] = struct{}{}
	}
	for _, r := range routing.Rules {
		if _, ok := tagSet[r.OutboundTag]; !ok {
			return fmt.Errorf("routing rule %q references unknown outbound_tag %q", r.Name, r.OutboundTag)
		}
	}
	if routing.DefaultOutboundTag != "" {
		if routing.DefaultOutboundTag != "direct" {
			if _, ok := tagSet[routing.DefaultOutboundTag]; !ok {
				return fmt.Errorf("default_outbound_tag %q not found", routing.DefaultOutboundTag)
			}
		}
	}
	if err := checkXrayAssetFiles(cfg, routing); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(cfg.App.GeneratedXrayConfig), 0o755)
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

func checkXrayAssetFiles(cfg *config.File, routing *config.Routing) error {
	needGeosite := false
	for _, r := range routing.Rules {
		for _, d := range r.Domain {
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(d)), "geosite:") {
				needGeosite = true
				break
			}
		}
		if needGeosite {
			break
		}
	}
	if !needGeosite {
		return nil
	}
	assetDir := inferXrayAssetDir(cfg.Xray.Bin)
	geositePath := filepath.Join(assetDir, "geosite.dat")
	if _, err := os.Stat(geositePath); err != nil {
		return fmt.Errorf("geosite rule detected but geosite.dat not found at %s", geositePath)
	}
	return nil
}

func inferXrayAssetDir(xrayBin string) string {
	// v2rayN layout: <home>/bin/xray/xray, assets under <home>/bin
	dir := filepath.Dir(xrayBin)
	parent := filepath.Dir(dir)
	if strings.EqualFold(filepath.Base(dir), "xray") {
		return parent
	}
	return dir
}
