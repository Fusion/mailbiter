package main

import (
	"fmt"
	"log"
	"strings"

	. "github.com/Soft/iter"
	"github.com/antonmedv/expr"
	"github.com/davecgh/go-spew/spew"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/fusion/mailbiter/clients"
	"github.com/fusion/mailbiter/config"
	"github.com/fusion/mailbiter/exprhelpers"
	"github.com/fusion/mailbiter/messageinfo"
	"github.com/fusion/mailbiter/secret"
	"github.com/hydronica/toml"
)

type Action int

const (
	SET Action = iota
	CLEAR
)

func main() {
	cfg := getConfig()
	validateConfig(cfg)

	for _, profile := range cfg.Profile {
		profileWork(&profile)
	}

	//getMailboxes(client)
	//getInbox(client, "Clutter")
	//moveMsg(client, "INBOX", "Clutter")
	//moveMsg(client, "Clutter", "INBOX")
}

func profileWork(profile *config.Profile) {

	fmt.Println("Profile:", profile.Settings.SecretName)
	readClient := login(profile)
	defer (*readClient).Logout()
	writeClient := login(profile)
	defer (*writeClient).Logout()
	clients := &clients.Clients{readClient, writeClient}

	processFolder(profile, clients, "INBOX", profile.Settings.MaxProcessed)
}

func getConfig() *config.Config {
	var config config.Config
	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		log.Fatal(err)
	}
	var secret secret.Secret
	if _, err := toml.DecodeFile("secret.toml", &secret); err != nil {
		log.Fatal(err)
	}
	for idx, _ := range config.Profile {
		account, ok := secret.Account[config.Profile[idx].Settings.SecretName]
		if !ok {
			// TODO does not validate
		}
		config.Profile[idx].Account = account
	}
	return &config
}

func validateConfig(cfg *config.Config) {
	for _, profile := range cfg.Profile {
		for _, rule := range profile.RowRule {
			for _, actionname := range rule.ActionNames {
				_, ok := profile.Actions[actionname]
				if !ok {
					// TODO does not validate
				}
			}
		}
	}
}

func processFolder(cfg *config.Profile, c *clients.Clients, folderName string, cutoff uint32) {
	mbox, err := c.Read.Select(folderName, false)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Flags for ", folderName, ":", mbox.Flags)

	// Get the last 4 messages
	from := uint32(1)
	to := mbox.Messages
	if mbox.Messages > cutoff-1 {
		// We're using unsigned integers here, only subtract if the result is > 0
		from = mbox.Messages - (cutoff - 1)
	}
	seqset := new(imap.SeqSet)
	seqset.AddRange(from, to)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Read.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	}()

	log.Println("Max ", cutoff, " messages:")
	for msg := range messages {

		sender := ""
		var senders []string
		if len(msg.Envelope.From) > 0 {
			sender = makeEmailAddress(msg.Envelope.From[0])
			if len(msg.Envelope.From) < 2 {
				senders = []string{sender}
			} else {
				senders = ToSlice(Map(
					Slice(msg.Envelope.From),
					func(address *imap.Address) string {
						return makeEmailAddress(address)
					}))
			}
		}

		recipient := ""
		var recipients []string
		if len(msg.Envelope.To) > 0 {
			recipient = makeEmailAddress(msg.Envelope.To[0])
			if len(msg.Envelope.To) < 2 {
				recipients = []string{recipient}
			} else {
				recipients = ToSlice(Map(
					Slice(msg.Envelope.To),
					func(address *imap.Address) string {
						return makeEmailAddress(address)
					}))
			}
		}

		cc := ToSlice(Map(
			Slice(msg.Envelope.Cc),
			func(address *imap.Address) string {
				return makeEmailAddress(address)
			}))

		bcc := ToSlice(Map(
			Slice(msg.Envelope.Bcc),
			func(address *imap.Address) string {
				return makeEmailAddress(address)
			}))

		flags := ToSlice(Map(
			Slice(msg.Flags),
			func(flag string) string {
				return strings.Replace(flag, "\\", "", -1)
			}))

		messageInfo := messageinfo.MessageInfo{
			Uid:        msg.Uid,
			Sender:     sender,
			Senders:    senders,
			Recipient:  recipient,
			Recipients: recipients,
			Cc:         cc,
			Bcc:        bcc,
			Date:       msg.Envelope.Date.Unix(),
			Day:        msg.Envelope.Date.Day(),
			Month:      int(msg.Envelope.Date.Month()),
			Monthname:  msg.Envelope.Date.Month().String(),
			Year:       msg.Envelope.Date.Year(),
			Subject:    msg.Envelope.Subject,
			Flags:      flags,
		}
		processMessage(cfg, c.Write, folderName, messageInfo)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	log.Println("Done!")

}

func makeEmailAddress(address *imap.Address) string {
	if address.PersonalName != "" {
		return fmt.Sprintf("%s <%s@%s>",
			address.PersonalName,
			address.MailboxName,
			address.HostName)
	}
	return fmt.Sprintf("%s@%s",
		address.MailboxName,
		address.HostName)
}

func processMessage(cfg *config.Profile, c *client.Client, folderName string, message messageinfo.MessageInfo) {
	env := map[string]interface{}{
		"lower":    exprhelpers.Lower,
		"calendar": exprhelpers.Date,
		"now":      exprhelpers.Now,
		"duration": exprhelpers.Duration,

		"sender":     message.Sender,
		"senders":    message.Senders,
		"recipient":  message.Recipient,
		"recipients": message.Recipients,
		"cc":         message.Cc,
		"bcc":        message.Bcc,
		"date":       message.Date,
		"day":        message.Day,
		"weekday":    message.Weekday,
		"month":      message.Month,
		"monthname":  message.Monthname,
		"year":       message.Year,
		"subject":    message.Subject,
		"flags":      message.Flags,
	}

	for _, rule := range cfg.RowRule {
		out, err := expr.Eval(rule.Condition, env)
		if err != nil {
			panic(err)
		}
		if out == false {
			continue
		}
		// So, we matched a rule.
		for _, action := range rule.Actions {
			performAction(cfg, c, folderName, message, action)
		}
		for _, actionname := range rule.ActionNames {
			action := cfg.Actions[actionname]
			for _, disp := range action.Disp {
				performAction(cfg, c, folderName, message, disp)
			}
		}
	}
}

func performAction(cfg *config.Profile, c *client.Client, folderName string, message messageinfo.MessageInfo, action string) {
	log.Println(message.Uid, message.Subject, "->", action)
	if action == "info" {
		return
	}
	if action == "delete" {
		deleteMsg(cfg, c, folderName, message.Uid)
		return
	}
	if action == "inspect" {
		alertMsg(cfg, message)
		return
	}
	if strings.HasPrefix(action, "move to ") {
		dest := unquote(strings.TrimPrefix(action, "move to "))
		moveMsg(cfg, c, folderName, message.Uid, dest)
		return
	}
	if strings.HasPrefix(action, "copy to ") {
		dest := unquote(strings.TrimPrefix(action, "copy to "))
		copyMsg(cfg, c, folderName, message.Uid, dest)
		return
	}
	if strings.HasPrefix(action, "set flag ") {
		flagName := unquote(strings.TrimPrefix(action, "flag "))
		toggleMsgFlag(cfg, c, folderName, message.Uid, SET, flagName)
		return
	}
	if strings.HasPrefix(action, "run ") {
		script := unquote(strings.TrimPrefix(action, "run "))
		log.Println(script)
		return
	}
	log.Fatal("Unknown action:", action)
}

func unquote(str string) string {
	if len(str) > 0 && str[0] == '\'' {
		str = str[1:]
	}
	if len(str) > 0 && str[len(str)-1] == '\'' {
		str = str[:len(str)-1]
	}
	return str
}

func alertMsg(cfg *config.Profile, message messageinfo.MessageInfo) {
	spew.Dump(message)
}

func moveMsg(cfg *config.Profile, c *client.Client, sourceFolder string, uid uint32, destFolder string) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := c.Select(sourceFolder, false); err != nil {
		log.Fatal(err)
	}
	if err := c.UidMove(seqset, destFolder); err != nil {
		log.Fatal(err)
	}
}

func copyMsg(cfg *config.Profile, c *client.Client, sourceFolder string, uid uint32, destFolder string) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := c.Select(sourceFolder, false); err != nil {
		log.Fatal(err)
	}
	if err := c.UidCopy(seqset, destFolder); err != nil {
		log.Fatal(err)
	}
}

func deleteMsg(cfg *config.Profile, c *client.Client, workFolder string, uid uint32) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := c.Select(workFolder, false); err != nil {
		log.Fatal(err)
	}
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.DeletedFlag}
	if err := c.UidStore(seqset, item, flags, nil); err != nil {
		log.Fatal(err)
	}
	// Delete immediately. Surely expuging only once per session would be an improvement.
	if err := c.Expunge(nil); err != nil {
		log.Fatal(err)
	}
}

// DO NOT DOCUMENT.
// For the time being, this will be considered an easter egg more than anything else.
// If a flag does not exist, even when arbitrary flags are allowed, this will fail.
func toggleMsgFlag(cfg *config.Profile, c *client.Client, workFolder string, uid uint32, action Action, flagName string) {
	seqset := new(imap.SeqSet)
	seqset.AddNum(uid)
	if _, err := c.Select(workFolder, false); err != nil {
		log.Fatal(err)
	}
	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{fmt.Sprintf("\\%s", flagName)}
	if err := c.UidStore(seqset, item, flags, nil); err != nil {
		log.Fatal(err)
	}
}

func login(cfg *config.Profile) *client.Client {
	fmt.Print("Connecting to server...")
	c, err := client.DialTLS(
		fmt.Sprintf("%s:%d", cfg.Account.Host, cfg.Account.Port),
		nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Connected...")
	if err := c.Login(
		cfg.Account.Username,
		cfg.Account.Password); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Logged in")
	return c
}

func getMailboxes(c *client.Client) {
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	log.Println("Mailboxes:")
	for m := range mailboxes {
		log.Println("* " + m.Name)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}
}

func getInbox(c *client.Client, folder string) {
	mbox, err := c.Select(folder, false)
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
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
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
