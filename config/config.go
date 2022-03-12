package config

import (
	"github.com/fusion/mailbiter/secret"
)

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
	Profile []Profile
}
