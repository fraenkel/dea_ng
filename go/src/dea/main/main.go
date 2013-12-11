package main

import (
	"dea/boot"
	cfg "dea/config"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config path>\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]
	config, err := cfg.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config not found at '%s'\n%s", configPath, err.Error())
		os.Exit(1)
	}

	b := boot.NewBootstrap(&config)
	err = b.Setup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed\n%s", configPath, err.Error())
		os.Exit(1)
	}

	b.Start()
}
