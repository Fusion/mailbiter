package service

import (
	"log"
	"time"

	"github.com/fusion/mailbiter/config"
	"github.com/fusion/mailbiter/core"
	"github.com/kardianos/service"
	"github.com/sevlyar/go-daemon"
)

type mbService struct {
	debugLevel uint8
}

func RunService(cfg *config.Config) {
	cntxt := &daemon.Context{
		PidFileName: cfg.Service.PidFileName,
		PidFilePerm: 0644,
		LogFileName: cfg.Global.LogFileName,
		LogFilePerm: 0640,
		WorkDir:     "./",
		Umask:       027,
		Args:        []string{"[mailbiter daemon]"},
	}

	_, err := cntxt.Reborn()
	if err != nil {
		log.Fatal("Unable to run: ", err)
	}
	defer cntxt.Release()

	go func() {
		for {
			core := core.Core{}
			core.Execute(cfg)

			time.Sleep(time.Duration(cfg.Service.Polling) * time.Second)
		}
	}()

	err = daemon.ServeSignals()
	if err != nil {
		log.Printf("Error: %s", err.Error())
	}
}

func (mb mbService) Start(_ service.Service) error {
	if mb.debugLevel > 0 {
		log.Println("Starting mb service")
	}
	return nil
}

func (mb mbService) Stop(_ service.Service) error {
	if mb.debugLevel > 0 {
		log.Println("Stopping mb service")
	}
	return nil
}
