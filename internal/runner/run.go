package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lkimju1/v2n-coremesh/internal/config"
	"github.com/lkimju1/v2n-coremesh/internal/sysproxy"
)

type Process struct {
	name   string
	cmd    *exec.Cmd
	doneCh chan error
}

func Run(cfg *config.File) error {
	return RunWithAssetDir(cfg, "")
}

func RunWithAssetDir(cfg *config.File, assetDir string) error {
	started := make([]Process, 0, len(cfg.Cores)+1)
	restoreProxy := func() error { return nil }
	proxyChanged := false
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if err := restoreProxy(); err != nil {
				fmt.Fprintf(os.Stderr, "[sysproxy] restore failed: %v\n", err)
			} else if proxyChanged {
				fmt.Printf("[sysproxy] restored previous system proxy settings\n")
			}
			for i := len(started) - 1; i >= 0; i-- {
				p := started[i]
				if p.cmd.Process != nil {
					_ = p.cmd.Process.Kill()
				}
				select {
				case <-p.doneCh:
				case <-time.After(2 * time.Second):
				}
			}
		})
	}

	for _, c := range cfg.Cores {
		args := replaceConfigPlaceholder(c.Args, c.Config)
		cmd := exec.Command(c.Bin, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		doneCh := make(chan error, 1)
		fmt.Printf("[core] starting %s: %s %s\n", c.Name, c.Bin, strings.Join(args, " "))
		if err := cmd.Start(); err != nil {
			cleanup()
			return fmt.Errorf("start core %s failed: %w", c.Name, err)
		}
		go func() {
			doneCh <- cmd.Wait()
		}()
		started = append(started, Process{name: c.Name, cmd: cmd, doneCh: doneCh})
		if err := waitHealthy(doneCh, 600*time.Millisecond); err != nil {
			cleanup()
			return fmt.Errorf("core %s exited early: %w", c.Name, err)
		}
		fmt.Printf("[core] started %s\n", c.Name)
	}

	xrayArgs := replaceConfigPlaceholder(cfg.Xray.Args, cfg.App.GeneratedXrayConfig)
	xrayCmd := exec.Command(cfg.Xray.Bin, xrayArgs...)
	xrayCmd.Stdout = os.Stdout
	xrayCmd.Stderr = os.Stderr
	if assetDir == "" {
		assetDir = inferXrayAssetDir(cfg.Xray.Bin)
	}
	xrayCmd.Env = append(os.Environ(),
		"XRAY_LOCATION_ASSET="+assetDir,
		"XRAY_LOCATION_CERT="+assetDir,
	)
	xrayDone := make(chan error, 1)
	fmt.Printf("[xray] starting: %s %s\n", cfg.Xray.Bin, strings.Join(xrayArgs, " "))
	if err := xrayCmd.Start(); err != nil {
		cleanup()
		return fmt.Errorf("start xray failed: %w", err)
	}
	go func() {
		xrayDone <- xrayCmd.Wait()
	}()
	started = append(started, Process{name: "xray", cmd: xrayCmd, doneCh: xrayDone})
	if err := waitHealthy(xrayDone, 600*time.Millisecond); err != nil {
		cleanup()
		return fmt.Errorf("xray exited early: %w", err)
	}
	fmt.Printf("[xray] started\n")

	proxyRestore, changed, err := sysproxy.ConfigureForRun(cfg.App.GeneratedXrayConfig)
	if err != nil {
		cleanup()
		return fmt.Errorf("configure system proxy: %w", err)
	}
	restoreProxy = proxyRestore
	proxyChanged = changed
	if changed {
		fmt.Printf("[sysproxy] enabled and pointed to xray inbound\n")
	} else {
		fmt.Printf("[sysproxy] unchanged (already configured or unsupported platform)\n")
	}

	if err := <-xrayDone; err != nil {
		cleanup()
		return fmt.Errorf("xray exited with error: %w", err)
	}
	cleanup()
	return nil
}

func waitHealthy(doneCh <-chan error, grace time.Duration) error {
	select {
	case err := <-doneCh:
		if err != nil {
			return err
		}
		return fmt.Errorf("process exited")
	case <-time.After(grace):
		return nil
	}
}

func replaceConfigPlaceholder(args []string, configPath string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		out = append(out, strings.ReplaceAll(a, "{{config}}", configPath))
	}
	if len(out) == 0 {
		return []string{configPath}
	}
	return out
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
