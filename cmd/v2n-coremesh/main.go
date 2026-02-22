package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/lkimju1/v2n-coremesh/internal/applog"
	"github.com/lkimju1/v2n-coremesh/internal/assets"
	"github.com/lkimju1/v2n-coremesh/internal/config"
	"github.com/lkimju1/v2n-coremesh/internal/runner"
	"github.com/lkimju1/v2n-coremesh/internal/state"
	"github.com/lkimju1/v2n-coremesh/internal/v2raynimport"
	"github.com/lkimju1/v2n-coremesh/internal/validate"
	"github.com/lkimju1/v2n-coremesh/internal/xraygen"
)

func main() {
	if len(os.Args) < 2 {
		exitErr(fmt.Errorf("missing subcommand"))
	}

	cmd := strings.ToLower(strings.TrimSpace(os.Args[1]))
	var err error
	switch cmd {
	case "parse":
		err = runParse(os.Args[2:])
	case "run":
		err = runRun(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
		return
	default:
		err = fmt.Errorf("unknown subcommand %q", os.Args[1])
	}
	if err != nil {
		exitErr(err)
	}
}

func runParse(args []string) error {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	confDir := fs.String("conf-dir", defaultConfDir(), "config directory")
	v2raynHome := fs.String("v2rayn-home", "", "v2rayN home path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*v2raynHome) == "" {
		return fmt.Errorf("-v2rayn-home is required")
	}
	logger, err := applog.New(*confDir)
	if err != nil {
		return err
	}
	defer logger.Close()
	logger.Printf("command=parse conf_dir=%s v2rayn_home=%s", *confDir, *v2raynHome)

	mainCfg, routingCfg, err := v2raynimport.LoadFromHome(*v2raynHome)
	if err != nil {
		logger.Printf("parse failed: %v", err)
		return err
	}

	mainCfg.App.WorkDir = *confDir
	mainCfg.App.GeneratedXrayConfig = filepath.Join(*confDir, "xray.generated.json")

	customRules, err := config.LoadCustomRules(filepath.Join(*confDir, "custom_rules.yaml"))
	if err != nil {
		logger.Printf("load custom rules failed: %v", err)
		return err
	}
	if err := validate.Main(mainCfg, routingCfg); err != nil {
		logger.Printf("validate failed: %v", err)
		return err
	}
	if err := xraygen.Generate(mainCfg, routingCfg, customRules); err != nil {
		logger.Printf("generate xray config failed: %v", err)
		return err
	}
	if err := state.Save(*confDir, state.New(*v2raynHome, mainCfg)); err != nil {
		logger.Printf("save state failed: %v", err)
		return err
	}
	logger.Printf("generated xray config: %s", mainCfg.App.GeneratedXrayConfig)
	logger.Printf("state file: %s", state.Path(*confDir))
	logger.Printf("parsed cores: %d", len(mainCfg.Cores))
	return nil
}

func runRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	confDir := fs.String("conf-dir", defaultConfDir(), "config directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	logger, err := applog.New(*confDir)
	if err != nil {
		return err
	}
	defer logger.Close()
	logger.Printf("command=run conf_dir=%s", *confDir)

	stateFile, err := state.Load(*confDir)
	if err != nil {
		logger.Printf("load state failed: %v", err)
		return err
	}
	cfg := &stateFile.Config
	cfg.App.WorkDir = *confDir

	if err := assets.EnsureGeoFiles(*confDir, time.Now()); err != nil {
		logger.Printf("ensure geo files failed: %v", err)
		return err
	}
	if err := validate.ForRun(cfg); err != nil {
		logger.Printf("validate run config failed: %v", err)
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return runner.RunWithAssetDir(ctx, cfg, *confDir, logger)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	printUsage()
	os.Exit(1)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  v2n-coremesh parse -v2rayn-home <path> [-conf-dir <path>]")
	fmt.Fprintln(os.Stderr, "  v2n-coremesh run [-conf-dir <path>]")
}

func defaultConfDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".v2n_coremesh"
	}
	return filepath.Join(home, ".v2n_coremesh")
}
