package mail

import (
	"fmt"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/fusion/mailbiter/config"
)

type Action int

const (
	SET Action = iota
	CLEAR
)

func Login(debugLevel uint8, cfg *config.Profile) *client.Client {
	if debugLevel > 0 {
		fmt.Print("Connecting to server...")
	}
	c, err := client.DialTLS(
		fmt.Sprintf("%s:%d", cfg.Account.Host, cfg.Account.Port),
		nil)
	if err != nil {
		log.Fatal(err)
	}
	if debugLevel > 1 {
		fmt.Print("Connected...")
	}
	if err := c.Login(
		cfg.Account.Username,
		cfg.Account.Password); err != nil {
		log.Fatal(err)
	}
	if debugLevel > 1 {
		fmt.Println("Logged in")
	}
	return c
}

func MoveMsg(cfg *config.Profile, client *client.Client, sourceFolder string, uid uint32, destFolder string) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := client.Select(sourceFolder, false); err != nil {
		log.Fatal(err)
	}
	if err := client.UidMove(seqset, destFolder); err != nil {
		log.Fatal(err)
	}
}

func CopyMsg(cfg *config.Profile, client *client.Client, sourceFolder string, uid uint32, destFolder string) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := client.Select(sourceFolder, false); err != nil {
		log.Fatal(err)
	}
	if err := client.UidCopy(seqset, destFolder); err != nil {
		log.Fatal(err)
	}
}

func DeleteMsg(cfg *config.Profile, client *client.Client, workFolder string, uid uint32) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := client.Select(workFolder, false); err != nil {
		log.Fatal(err)
	}
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := client.UidStore(seqset, item, flags, nil); err != nil {
		log.Fatal(err)
	}
	// Delete immediately. Surely expuging only once per session would be an improvement.
	if err := client.Expunge(nil); err != nil {
		log.Fatal(err)
	}
}

// DO NOT DOCUMENT.
// For the time being, this will be considered an easter egg more than anything else.
// If a flag does not exist, even when arbitrary flags are allowed, this will fail.
func ToggleMsgFlag(cfg *config.Profile, client *client.Client, workFolder string, uid uint32, action Action, flagName string) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := client.Select(workFolder, false); err != nil {
		log.Fatal(err)
	}
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{fmt.Sprintf("\\%s", flagName)}
	if err := client.UidStore(seqset, item, flags, nil); err != nil {
		log.Fatal(err)
	}
}

/*
 * BELOW: UNUSED -- FOR TESTING PURPOSE ONLY
 */

func getMailboxes(client *client.Client) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}
}

func getInbox(client *client.Client, folder string) {
	mbox, err := client.Select(folder, false)
	if err != nil {
		log.Fatal(err)
	}
	//log.Println("Flags for", folder, ":", mbox.Flags)

	maxMsg := uint32(1000)

	// Get the last 4 messages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > maxMsg-1 {
		// We're using unsigned integers here, only subtract if the result is > 0
		from = mbox.Messages - (maxMsg - 1)
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- client.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	}()

	log.Println("Last ", maxMsg, " messages:")
	for msg := range messages {
		log.Println("* " + msg.Envelope.Subject)
		spew.Dump(msg.Flags)
		spew.Dump(msg.Uid)
		spew.Dump(msg.SeqNum)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")
}
