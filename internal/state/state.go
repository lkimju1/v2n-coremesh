package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lkimju1/v2n-coremesh/internal/config"
)

const (
	CurrentVersion = 1
	FileName       = "coremesh.state.json"
)

type File struct {
	Version    int         `json:"version"`
	V2rayNHome string      `json:"v2rayn_home"`
	ParsedAt   time.Time   `json:"parsed_at"`
	Config     config.File `json:"config"`
}

func New(v2raynHome string, cfg *config.File) *File {
	return &File{
		Version:    CurrentVersion,
		V2rayNHome: v2raynHome,
		ParsedAt:   time.Now().UTC(),
		Config:     *cfg,
	}
}

func Path(confDir string) string {
	return filepath.Join(confDir, FileName)
}

func Save(confDir string, stateFile *File) error {
	if stateFile == nil {
		return fmt.Errorf("state is nil")
	}
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		return fmt.Errorf("create conf dir: %w", err)
	}

	content, err := json.MarshalIndent(stateFile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	target := Path(confDir)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return fmt.Errorf("write state temp file: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("rename state file: %w", err)
	}
	return nil
}

func Load(confDir string) (*File, error) {
	target := Path(confDir)
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var stateFile File
	if err := json.Unmarshal(content, &stateFile); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	if stateFile.Version <= 0 {
		return nil, fmt.Errorf("invalid state version: %d", stateFile.Version)
	}
	return &stateFile, nil
}
