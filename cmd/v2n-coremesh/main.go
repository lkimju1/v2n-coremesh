package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lkimju1/v2n-coremesh/internal/config"
	"github.com/lkimju1/v2n-coremesh/internal/runner"
	"github.com/lkimju1/v2n-coremesh/internal/v2raynimport"
	"github.com/lkimju1/v2n-coremesh/internal/validate"
	"github.com/lkimju1/v2n-coremesh/internal/xraygen"
)

func main() {
	cfgPath := flag.String("config", "./config.yaml", "main config file path")
	v2raynHome := flag.String("v2rayn-home", "", "v2rayN home path (auto read guiConfigs/guiNConfig.json and guiNDB.db)")
	dryRun := flag.Bool("dry-run", false, "only validate and generate config")
	flag.Parse()

	var (
		mainCfg    *config.File
		routingCfg *config.Routing
		err        error
	)
	if *v2raynHome != "" {
		mainCfg, routingCfg, err = v2raynimport.LoadFromHome(*v2raynHome)
	} else {
		mainCfg, err = config.LoadMain(*cfgPath)
		if err == nil {
			routingCfg, err = config.LoadRouting(mainCfg.RoutingRulesFile)
		}
	}
	if err != nil {
		exitErr(err)
	}
	if err := validate.Main(mainCfg, routingCfg); err != nil {
		exitErr(err)
	}
	if err := xraygen.Generate(mainCfg, routingCfg); err != nil {
		exitErr(err)
	}
	fmt.Printf("generated xray config: %s\n", mainCfg.App.GeneratedXrayConfig)
	if *dryRun {
		return
	}
	if err := runner.Run(mainCfg); err != nil {
		exitErr(err)
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
