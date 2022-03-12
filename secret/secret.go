package secret

type Account struct {
	Username string
	Password string
	Host     string
	Port     int
}

type Secret struct {
	Account map[string]*Account
}
