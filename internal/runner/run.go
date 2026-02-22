package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lkimju1/v2n-coremesh/internal/applog"
	"github.com/lkimju1/v2n-coremesh/internal/config"
	"github.com/lkimju1/v2n-coremesh/internal/sysproxy"
)

type Process struct {
	name   string
	cmd    *exec.Cmd
	doneCh chan error
}

func Run(cfg *config.File) error {
	return RunWithAssetDir(context.Background(), cfg, "", nil)
}

func RunWithAssetDir(ctx context.Context, cfg *config.File, assetDir string, logger *applog.Logger) error {
	if ctx == nil {
		ctx = context.Background()
	}
	workDir := strings.TrimSpace(cfg.App.WorkDir)
	if workDir == "" {
		workDir = "."
	}
	started := make([]Process, 0, len(cfg.Cores)+1)
	restoreProxy := func() error { return nil }
	proxyChanged := false
	xrayLogPath := filepath.Join(workDir, applog.XrayLogFileName)
	xrayLog, err := applog.OpenXrayLog(workDir)
	if err != nil {
		return fmt.Errorf("open xray log: %w", err)
	}
	defer xrayLog.Close()

	logf := func(format string, args ...any) {
		if logger == nil {
			return
		}
		logger.Printf(format, args...)
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if err := restoreProxy(); err != nil {
				logf("[sysproxy] restore failed: %v", err)
			} else if proxyChanged {
				logf("[sysproxy] restored previous system proxy settings")
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
	defer cleanup()

	for _, c := range cfg.Cores {
		args := replaceConfigPlaceholder(c.Args, c.Config)
		cmd := exec.Command(c.Bin, args...)
		if logger != nil && logger.Writer() != nil {
			cmd.Stdout = logger.Writer()
			cmd.Stderr = logger.Writer()
		}
		doneCh := make(chan error, 1)
		logf("[core] starting %s: %s %s", c.Name, c.Bin, strings.Join(args, " "))
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start core %s failed: %w", c.Name, err)
		}
		go func() {
			doneCh <- cmd.Wait()
		}()
		started = append(started, Process{name: c.Name, cmd: cmd, doneCh: doneCh})
		if err := waitHealthy(doneCh, 600*time.Millisecond); err != nil {
			return fmt.Errorf("core %s exited early: %w", c.Name, err)
		}
		logf("[core] started %s", c.Name)
	}

	xrayArgs := replaceConfigPlaceholder(cfg.Xray.Args, cfg.App.GeneratedXrayConfig)
	xrayCmd := exec.Command(cfg.Xray.Bin, xrayArgs...)
	xrayCmd.Stdout = xrayLog
	xrayCmd.Stderr = xrayLog
	if assetDir == "" {
		assetDir = inferXrayAssetDir(cfg.Xray.Bin)
	}
	xrayCmd.Env = append(os.Environ(),
		"XRAY_LOCATION_ASSET="+assetDir,
		"XRAY_LOCATION_CERT="+assetDir,
	)
	xrayDone := make(chan error, 1)
	logf("[xray] starting: %s %s", cfg.Xray.Bin, strings.Join(xrayArgs, " "))
	logf("[xray] log file: %s", xrayLogPath)
	if err := xrayCmd.Start(); err != nil {
		return fmt.Errorf("start xray failed: %w", err)
	}
	go func() {
		xrayDone <- xrayCmd.Wait()
	}()
	started = append(started, Process{name: "xray", cmd: xrayCmd, doneCh: xrayDone})
	if err := waitHealthy(xrayDone, 600*time.Millisecond); err != nil {
		return fmt.Errorf("xray exited early: %w", err)
	}
	logf("[xray] started")

	proxyRestore, changed, err := sysproxy.ConfigureForRun(cfg.App.GeneratedXrayConfig)
	if err != nil {
		return fmt.Errorf("configure system proxy: %w", err)
	}
	restoreProxy = proxyRestore
	proxyChanged = changed
	if changed {
		logf("[sysproxy] enabled and pointed to xray inbound")
	} else {
		logf("[sysproxy] unchanged (already configured or unsupported platform)")
	}

	select {
	case err := <-xrayDone:
		if err != nil {
			return fmt.Errorf("xray exited with error: %w", err)
		}
		logf("[xray] exited")
		return nil
	case <-ctx.Done():
		logf("[run] shutdown requested: %v", ctx.Err())
		return nil
	}
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
