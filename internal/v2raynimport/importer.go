package v2raynimport

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/lkimju1/v2n-coremesh/internal/config"

	_ "modernc.org/sqlite"
)

const (
	configTypeCustom = 2
	coreTypeXray     = 2
)

type guiConfig struct {
	IndexID string `json:"IndexId"`
}

type profileRow struct {
	IndexID    string
	ConfigType int
	CoreType   sql.NullInt64
	Remarks    sql.NullString
	Address    sql.NullString
}

type routingRow struct {
	RuleSet sql.NullString
}

type ruleItem struct {
	OutboundTag string   `json:"OutboundTag"`
	Domain      []string `json:"Domain"`
	Enabled     bool     `json:"Enabled"`
}

func LoadFromHome(home string) (*config.File, *config.Routing, error) {
	home = filepath.Clean(home)
	guiConfigPath := filepath.Join(home, "guiConfigs", "guiNConfig.json")
	dbPath := filepath.Join(home, "guiConfigs", "guiNDB.db")
	if _, err := os.Stat(guiConfigPath); err != nil {
		return nil, nil, fmt.Errorf("v2rayN gui config not found: %s: %w", guiConfigPath, err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, nil, fmt.Errorf("v2rayN db not found: %s: %w", dbPath, err)
	}

	guiCfg, err := readGuiConfig(guiConfigPath)
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open v2rayN db: %w", err)
	}
	defer db.Close()

	profiles, err := readProfiles(db)
	if err != nil {
		return nil, nil, err
	}
	if len(profiles) == 0 {
		return nil, nil, fmt.Errorf("no profiles found in v2rayN db")
	}

	xrayBase, err := detectXrayBaseConfig(home)
	if err != nil {
		return nil, nil, err
	}
	xrayBin, err := findXrayBin(home)
	if err != nil {
		return nil, nil, err
	}

	customProfiles := filterCustomProfiles(profiles)
	cores := make([]config.Core, 0, len(customProfiles))
	remarkToTag := make(map[string]string)
	aliasUseCount := make(map[string]int)
	for _, p := range customProfiles {
		coreType := int64(coreTypeXray)
		if p.CoreType.Valid {
			coreType = p.CoreType.Int64
		}
		binPath, args, err := findCoreBinAndArgs(home, coreType)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve core bin for profile %s: %w", p.IndexID, err)
		}
		cfgRel := strings.TrimSpace(p.Address.String)
		cfgPath := cfgRel
		if !filepath.IsAbs(cfgPath) {
			cfgPath = filepath.Join(home, "guiConfigs", cfgPath)
		}
		listenHost, listenPort, err := inferListenFromConfig(cfgPath)
		if err != nil {
			return nil, nil, fmt.Errorf("infer listen for profile %s(%s): %w", p.IndexID, strings.TrimSpace(p.Remarks.String), err)
		}
		name := strings.TrimSpace(p.Remarks.String)
		if name == "" {
			name = p.IndexID
		}
		alias := uniqueAlias(aliasUseCount, sanitizeTag(name))
		isActive := guiCfg != nil && strings.EqualFold(strings.TrimSpace(guiCfg.IndexID), strings.TrimSpace(p.IndexID))
		cores = append(cores, config.Core{
			ProfileID:   p.IndexID,
			Name:        name,
			Alias:       alias,
			Type:        strconv.FormatInt(coreType, 10),
			Bin:         binPath,
			Config:      cfgPath,
			Listen:      config.Listen{Host: listenHost, Port: listenPort},
			Args:        args,
			OutboundTag: alias,
			Active:      isActive,
		})
		remark := strings.TrimSpace(p.Remarks.String)
		if remark != "" {
			// v2rayN routing references remarks, keep the first match as the stable mapping.
			if _, ok := remarkToTag[remark]; !ok {
				remarkToTag[remark] = alias
			}
		}
	}

	if len(cores) == 0 {
		return nil, nil, fmt.Errorf("no custom core profiles found in v2rayN db")
	}

	routing, err := readRouting(db, remarkToTag)
	if err != nil {
		return nil, nil, err
	}

	mainCfg := &config.File{
		App: config.App{
			WorkDir:             filepath.Join(home, "guiTemps", "create_exe"),
			GeneratedXrayConfig: filepath.Join(home, "guiTemps", "create_exe", "xray.generated.json"),
		},
		Xray: config.Xray{
			Bin:        xrayBin,
			BaseConfig: xrayBase,
			Args:       []string{"run", "-c", "{{config}}"},
		},
		Cores: cores,
	}
	return mainCfg, routing, nil
}

func readGuiConfig(path string) (*guiConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read gui config: %w", err)
	}
	var cfg guiConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse gui config: %w", err)
	}
	return &cfg, nil
}

func readProfiles(db *sql.DB) ([]profileRow, error) {
	rows, err := db.Query(`SELECT IndexId, ConfigType, CoreType, Remarks, Address FROM ProfileItem`)
	if err != nil {
		return nil, fmt.Errorf("query ProfileItem: %w", err)
	}
	defer rows.Close()
	out := make([]profileRow, 0)
	for rows.Next() {
		var r profileRow
		if err := rows.Scan(&r.IndexID, &r.ConfigType, &r.CoreType, &r.Remarks, &r.Address); err != nil {
			return nil, fmt.Errorf("scan ProfileItem: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func filterCustomProfiles(profiles []profileRow) []profileRow {
	out := make([]profileRow, 0)
	for _, p := range profiles {
		if p.ConfigType == configTypeCustom {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].IndexID < out[j].IndexID })
	return out
}

func detectXrayBaseConfig(home string) (string, error) {
	// configPre.json is generated for pre-service xray/sing-box. create_exe 2.0 requires this file explicitly.
	runtimePreConfig := filepath.Join(home, "binConfigs", "configPre.json")
	stat, err := os.Stat(runtimePreConfig)
	if err != nil {
		return "", fmt.Errorf("required xray base config not found: %s: %w", runtimePreConfig, err)
	}
	if stat.IsDir() {
		return "", fmt.Errorf("required xray base config is a directory: %s", runtimePreConfig)
	}
	ok, err := looksLikeXrayConfig(runtimePreConfig)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("required xray base config is not valid xray format: %s", runtimePreConfig)
	}
	return runtimePreConfig, nil
}

func looksLikeXrayConfig(path string) (bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read xray config candidate %s: %w", path, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		return false, fmt.Errorf("parse xray config candidate %s: %w", path, err)
	}

	outbounds, hasOutbounds := doc["outbounds"].([]any)
	routing, hasRouting := doc["routing"].(map[string]any)
	if !hasOutbounds || !hasRouting {
		return false, nil
	}
	if len(outbounds) == 0 {
		return false, nil
	}
	if _, ok := routing["rules"].([]any); !ok {
		return false, nil
	}
	return true, nil
}

func findXrayBin(home string) (string, error) {
	base := filepath.Join(home, "bin", "xray")
	for _, name := range candidates("xray") {
		p := filepath.Join(base, name)
		if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
			return p, nil
		}
	}
	return "", fmt.Errorf("xray executable not found under %s", base)
}

func findCoreBinAndArgs(home string, coreType int64) (string, []string, error) {
	type meta struct {
		dir   string
		names []string
		args  []string
	}
	m := map[int64]meta{
		2:  {dir: "xray", names: []string{"xray"}, args: []string{"run", "-c", "{{config}}"}},
		22: {dir: "naiveproxy", names: []string{"naive", "naiveproxy"}, args: []string{"{{config}}"}},
		24: {dir: "sing_box", names: []string{"sing-box-client", "sing-box"}, args: []string{"run", "-c", "{{config}}", "--disable-color"}},
		13: {dir: "mihomo", names: []string{"clash", "mihomo"}, args: []string{"-f", "{{config}}"}},
		23: {dir: "tuic", names: []string{"tuic-client", "tuic"}, args: []string{"-c", "{{config}}"}},
	}
	v, ok := m[coreType]
	if !ok {
		return "", nil, fmt.Errorf("unsupported coreType %d", coreType)
	}
	base := filepath.Join(home, "bin", v.dir)
	for _, n := range v.names {
		for _, c := range candidates(n) {
			p := filepath.Join(base, c)
			if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
				return p, v.args, nil
			}
		}
	}
	return "", nil, fmt.Errorf("executable not found under %s", base)
}

func candidates(name string) []string {
	// Keep cross-platform tolerance for parsing v2rayN home from another OS.
	if runtime.GOOS == "windows" {
		return []string{name + ".exe", name}
	}
	return []string{name, name + ".exe"}
}

func inferListenFromConfig(path string) (string, int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", 0, fmt.Errorf("read core config: %w", err)
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		return "", 0, fmt.Errorf("parse json core config: %w", err)
	}

	if m, ok := doc.(map[string]any); ok {
		if inbounds, ok := m["inbounds"].([]any); ok {
			for _, in := range inbounds {
				if mm, ok := in.(map[string]any); ok {
					h, p, ok := parseListenMap(mm)
					if ok {
						return h, p, nil
					}
				}
			}
		}
		if listenRaw, ok := m["listen"].(string); ok {
			h, p, ok := parseListenString(listenRaw)
			if ok {
				return h, p, nil
			}
		}
		if listenArr, ok := m["listen"].([]any); ok {
			h, p, ok := parseListenArray(listenArr)
			if ok {
				return h, p, nil
			}
		}
	}
	return "", 0, fmt.Errorf("cannot infer listen host:port from %s", path)
}

func parseListenMap(mm map[string]any) (string, int, bool) {
	var host string
	var port int
	if v, ok := mm["listen"].(string); ok && v != "" {
		host = v
	}
	if v, ok := mm["port"].(float64); ok && v > 0 {
		port = int(v)
	}
	if host != "" && port > 0 {
		return host, port, true
	}
	if s, ok := mm["listen"].(string); ok {
		h, p, ok := parseListenString(s)
		if ok {
			return h, p, true
		}
	}
	return "", 0, false
}

func parseListenString(s string) (string, int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, false
	}
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil {
			return "", 0, false
		}
		h := normalizeHost(u.Hostname())
		p, _ := strconv.Atoi(u.Port())
		if h != "" && p > 0 {
			return h, p, true
		}
		return "", 0, false
	}
	h, pStr, err := net.SplitHostPort(s)
	if err == nil {
		h = normalizeHost(h)
		p, _ := strconv.Atoi(pStr)
		if h != "" && p > 0 {
			return h, p, true
		}
	}
	return "", 0, false
}

func parseListenArray(listenArr []any) (string, int, bool) {
	// Prefer socks endpoint when multiple protocols are exposed.
	for _, it := range listenArr {
		s, ok := it.(string)
		if !ok {
			continue
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(s)), "socks://") {
			h, p, ok := parseListenString(s)
			if ok {
				return h, p, true
			}
		}
	}
	for _, it := range listenArr {
		s, ok := it.(string)
		if !ok {
			continue
		}
		h, p, ok := parseListenString(s)
		if ok {
			return h, p, true
		}
	}
	return "", 0, false
}

func normalizeHost(host string) string {
	h := strings.TrimSpace(strings.Trim(host, "[]"))
	switch h {
	case "", "0.0.0.0", "::":
		return "127.0.0.1"
	default:
		return h
	}
}

func readRouting(db *sql.DB, remarkToTag map[string]string) (*config.Routing, error) {
	rows, err := db.Query(`SELECT RuleSet FROM RoutingItem WHERE IsActive = 1 LIMIT 1`)
	if err != nil {
		return nil, fmt.Errorf("query active routing: %w", err)
	}
	defer rows.Close()
	var rr routingRow
	if !rows.Next() {
		return &config.Routing{DefaultOutboundTag: "direct"}, nil
	}
	if err := rows.Scan(&rr.RuleSet); err != nil {
		return nil, fmt.Errorf("scan active routing: %w", err)
	}
	if !rr.RuleSet.Valid || strings.TrimSpace(rr.RuleSet.String) == "" {
		return &config.Routing{DefaultOutboundTag: "direct"}, nil
	}
	var items []ruleItem
	if err := json.Unmarshal([]byte(rr.RuleSet.String), &items); err != nil {
		return nil, fmt.Errorf("parse routing ruleset: %w", err)
	}
	out := &config.Routing{Rules: make([]config.RoutingRule, 0), DefaultOutboundTag: "direct"}
	for _, it := range items {
		if !it.Enabled || len(it.Domain) == 0 {
			continue
		}
		tag := strings.TrimSpace(it.OutboundTag)
		if tag == "" || tag == "proxy" || tag == "block" {
			continue
		}
		if tag == "direct" {
			out.Rules = append(out.Rules, config.RoutingRule{Name: "direct", Domain: it.Domain, OutboundTag: "direct"})
			continue
		}
		mapped, ok := remarkToTag[tag]
		if !ok {
			continue
		}
		out.Rules = append(out.Rules, config.RoutingRule{Name: tag, Domain: it.Domain, OutboundTag: mapped})
	}
	return out, nil
}

func sanitizeTag(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ReplaceAll(s, " ", "-")
	if s == "" {
		return "unknown"
	}
	return s
}

func uniqueAlias(useCount map[string]int, base string) string {
	if base == "" {
		base = "unknown"
	}
	count := useCount[base]
	useCount[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, count+1)
}
