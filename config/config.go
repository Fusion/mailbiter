package config

import (
	"log"

	"github.com/fusion/mailbiter/secret"
	"github.com/hydronica/toml"
)

type Global struct {
	DebugLevel  uint8
	LogFileName string
}

type Service struct {
	Polling     uint32
	PidFileName string
}

type Actions struct {
	Disp []string
}

type Rule struct {
	Condition   string
	ActionNames []string
	Actions     []string
}

type Settings struct {
	MaxProcessed uint32
	SecretName   string
}

type Profile struct {
	Actions  map[string]Actions
	RowRule  []Rule
	SetRule  []Rule
	Account  *secret.Account
	Settings Settings
}

type Config struct {
	Global  Global
	Service Service
	Profile []Profile
}

func GetConfig(configFile string, secretFile string) *Config {
	var config Config

	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Fatal(err)
	}
	var secret secret.Secret
	if _, err := toml.DecodeFile(secretFile, &secret); err != nil {
		log.Fatal(err)
	}
	for idx, _ := range config.Profile {
		account, ok := secret.Account[config.Profile[idx].Settings.SecretName]
		if !ok {
			log.Fatal("Configuration error -- Unknown secret for account:", config.Profile[idx].Settings.SecretName)
		}
		config.Profile[idx].Account = account
	}

	if config.Global.LogFileName == "" {
		config.Global.LogFileName = "mailbiter.log"
	}
	if config.Service.Polling == 0 {
		config.Service.Polling = 60
	}
	if config.Service.PidFileName == "" {
		config.Service.PidFileName = "mailbiter.pid"
	}

	return &config
}

func ValidateConfig(cfg *Config) {
	for _, profile := range cfg.Profile {
		for _, rule := range profile.RowRule {
			for _, actionname := range rule.ActionNames {
				_, ok := profile.Actions[actionname]
				if !ok {
					log.Fatal("Configuration error -- Unknown action:", actionname)
				}
			}
		}
	}
}
