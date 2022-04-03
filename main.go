package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fusion/mailbiter/config"
	"github.com/fusion/mailbiter/core"
)

var Version = "unspecified"

func main() {
	var configFile string
	fs := flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
	fs.StringVar(&configFile, "c", "config.toml", "Path to config file")
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

	cfg := config.GetConfig(configFile)
	config.ValidateConfig(cfg)

	core.Execute(cfg)
}
