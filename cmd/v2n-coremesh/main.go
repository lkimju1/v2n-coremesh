package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/lkimju1/v2n-coremesh/internal/applog"
	"github.com/lkimju1/v2n-coremesh/internal/assets"
	"github.com/lkimju1/v2n-coremesh/internal/bindmode"
	"github.com/lkimju1/v2n-coremesh/internal/config"
	"github.com/lkimju1/v2n-coremesh/internal/runner"
	"github.com/lkimju1/v2n-coremesh/internal/state"
	"github.com/lkimju1/v2n-coremesh/internal/v2raynimport"
	"github.com/lkimju1/v2n-coremesh/internal/validate"
	"github.com/lkimju1/v2n-coremesh/internal/xraygen"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "v2n-coremesh",
		Usage: "parse v2rayN and run custom cores with xray",
		Commands: []*cli.Command{
			{
				Name:    "parse",
				Aliases: []string{"p"},
				Usage:   "parse v2rayN and generate runtime files",
				Action:  runParse,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "conf-dir",
						Aliases: []string{"c"},
						Usage:   "config directory",
						Value:   defaultConfDir(),
					},
					&cli.StringFlag{
						Name:     "v2rayn-home",
						Aliases:  []string{"v"},
						Usage:    "v2rayN home path",
						Required: true,
					},
				},
			},
			{
				Name:    "run",
				Aliases: []string{"r"},
				Usage:   "run all cores and xray",
				Action:  runRun,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "conf-dir",
						Aliases: []string{"c"},
						Usage:   "config directory",
						Value:   defaultConfDir(),
					},
					&cli.BoolFlag{
						Name:    "bind-all",
						Aliases: []string{"a"},
						Usage:   "bind xray and core listen addresses to 0.0.0.0 for LAN access",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		exitErr(err)
	}
}

func runParse(c *cli.Context) error {
	confDir := strings.TrimSpace(c.String("conf-dir"))
	v2raynHome := strings.TrimSpace(c.String("v2rayn-home"))

	logger, err := applog.New(confDir)
	if err != nil {
		return err
	}
	defer logger.Close()
	logger.Printf("command=parse conf_dir=%s v2rayn_home=%s", confDir, v2raynHome)

	mainCfg, routingCfg, err := v2raynimport.LoadFromHome(v2raynHome)
	if err != nil {
		logger.Printf("parse failed: %v", err)
		return err
	}

	mainCfg.App.WorkDir = confDir
	mainCfg.App.GeneratedXrayConfig = filepath.Join(confDir, "xray.generated.json")

	customRules, err := config.LoadCustomRules(filepath.Join(confDir, "custom_rules.yaml"))
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
	if err := state.Save(confDir, state.New(v2raynHome, mainCfg)); err != nil {
		logger.Printf("save state failed: %v", err)
		return err
	}
	logger.Printf("generated xray config: %s", mainCfg.App.GeneratedXrayConfig)
	logger.Printf("state file: %s", state.Path(confDir))
	logger.Printf("parsed cores: %d", len(mainCfg.Cores))
	return nil
}

func runRun(c *cli.Context) error {
	confDir := strings.TrimSpace(c.String("conf-dir"))
	bindAll := c.Bool("bind-all")

	logger, err := applog.New(confDir)
	if err != nil {
		return err
	}
	defer logger.Close()
	logger.Printf("command=run conf_dir=%s bind_all=%t", confDir, bindAll)

	stateFile, err := state.Load(confDir)
	if err != nil {
		logger.Printf("load state failed: %v", err)
		return err
	}
	cfg := &stateFile.Config
	cfg.App.WorkDir = confDir

	if bindAll {
		cfg, err = bindmode.PrepareForBindAll(cfg, confDir)
		if err != nil {
			logger.Printf("prepare bind-all runtime config failed: %v", err)
			return err
		}
		logger.Printf("bind-all enabled, runtime xray config: %s", cfg.App.GeneratedXrayConfig)
	}

	if err := assets.EnsureGeoFiles(confDir, time.Now()); err != nil {
		logger.Printf("ensure geo files failed: %v", err)
		return err
	}
	if err := validate.ForRun(cfg); err != nil {
		logger.Printf("validate run config failed: %v", err)
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return runner.RunWithAssetDir(ctx, cfg, confDir, logger)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}

func defaultConfDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ".v2n_coremesh"
	}
	return filepath.Join(home, ".v2n_coremesh")
}
