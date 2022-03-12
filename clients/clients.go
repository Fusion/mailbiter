package clients

import (
	"github.com/emersion/go-imap/client"
)

type Clients struct {
	Read  *client.Client
	Write *client.Client
}
