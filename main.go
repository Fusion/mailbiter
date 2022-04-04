package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fusion/mailbiter/config"
	"github.com/fusion/mailbiter/core"
	"github.com/fusion/mailbiter/service"
)

var Version = "unspecified"

func main() {
	var configFile string
	var secretFile string
	var serviceRequested bool
	var logFile string
	var pidFile string
	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
	fs.StringVar(&configFile, "config", "config.toml", "Path to config file")
	fs.StringVar(&secretFile, "secret", "secret.toml", "Path to secret file")
	fs.StringVar(&logFile, "log", "mailbiter.toml", "Path to log file")
	fs.StringVar(&pidFile, "pid", "mailbiter.pid", "Path to pid file")
	fs.BoolVar(&serviceRequested, "service", false, "Run as a service")
	fs.Usage = func() {
		w := flag.CommandLine.Output()
		fmt.Fprintf(w, "\n%s version %s\n", filepath.Base(os.Args[0]), Version)
		fmt.Fprintf(w, "Usage:\n\n")
		fs.PrintDefaults()
	}
	err := fs.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	cfg := config.GetConfig(configFile, secretFile)
	config.ValidateConfig(cfg)

	if serviceRequested {
		service.RunService(cfg)
	} else {
		core := core.Core{}
		core.Execute(cfg)
	}
}
