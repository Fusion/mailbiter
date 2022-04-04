package core

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
	"github.com/fusion/mailbiter/mail"
	"github.com/fusion/mailbiter/messageinfo"
)

type Core struct {
	debugLevel uint8
}

func (core Core) Execute(cfg *config.Config) {
	core.debugLevel = cfg.DebugLevel

	for _, profile := range cfg.Profile {
		core.profileWork(&profile)
	}

}

func (core Core) profileWork(profile *config.Profile) {

	log.Println("Profile:", profile.Settings.SecretName)
	readClient := mail.Login(core.debugLevel, profile)
	defer (*readClient).Logout()
	writeClient := mail.Login(core.debugLevel, profile)
	defer (*writeClient).Logout()
	clients := &clients.Clients{
		Read:  readClient,
		Write: writeClient}

	core.processFolder(profile, clients, "INBOX", profile.Settings.MaxProcessed)
}

func (core Core) processFolder(cfg *config.Profile, clients *clients.Clients, folderName string, cutoff uint32) {
	mbox, err := clients.Read.Select(folderName, false)
	if err != nil {
		log.Fatal(err)
	}
	if core.debugLevel > 1 {
		log.Println("Flags for ", folderName, ":", mbox.Flags)
	}

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
		done <- clients.Read.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchUid}, messages)
	}()

	if core.debugLevel > 1 {
		log.Println("Max ", cutoff, " messages:")
	}
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
		core.processMessage(cfg, clients.Write, folderName, messageInfo)
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	if core.debugLevel > 0 {
		log.Println("Done!")
	}

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

func (core Core) processMessage(cfg *config.Profile, client *client.Client, folderName string, message messageinfo.MessageInfo) {
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
			log.Fatal(err)
		}
		if out == false {
			continue
		}
		// So, we matched a rule.
		for _, action := range rule.Actions {
			core.performAction(cfg, client, folderName, message, action)
		}
		for _, actionname := range rule.ActionNames {
			action := cfg.Actions[actionname]
			for _, disp := range action.Disp {
				core.performAction(cfg, client, folderName, message, disp)
			}
		}
	}
}

func (core Core) performAction(cfg *config.Profile, client *client.Client, folderName string, message messageinfo.MessageInfo, action string) {
	if core.debugLevel > 1 {
		log.Println(message.Uid, message.Subject, "->", action)
	}
	if action == "info" {
		return
	}
	if action == "delete" {
		mail.DeleteMsg(cfg, client, folderName, message.Uid)
		return
	}
	if action == "inspect" {
		alertMsg(cfg, message)
		return
	}
	if strings.HasPrefix(action, "move to ") {
		dest := unquote(strings.TrimPrefix(action, "move to "))
		mail.MoveMsg(cfg, client, folderName, message.Uid, dest)
		return
	}
	if strings.HasPrefix(action, "copy to ") {
		dest := unquote(strings.TrimPrefix(action, "copy to "))
		mail.CopyMsg(cfg, client, folderName, message.Uid, dest)
		return
	}
	if strings.HasPrefix(action, "set flag ") {
		flagName := unquote(strings.TrimPrefix(action, "flag "))
		mail.ToggleMsgFlag(cfg, client, folderName, message.Uid, mail.SET, flagName)
		return
	}
	if strings.HasPrefix(action, "run ") {
		// TODO
		script := unquote(strings.TrimPrefix(action, "run "))
		if core.debugLevel < 1 {
			log.Println(script)
		}
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
