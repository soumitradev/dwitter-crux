package subscriptions

import (
	"fmt"
	"os"

	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/soumitradev/Dwitter/backend/common"
	"github.com/soumitradev/Dwitter/backend/prisma/db"
)

// TODO: Make the email look better
func SendEmail(title string, body string, recipient string) error {
	from := mail.NewEmail("Dwitter", os.Getenv("SENDGRID_SENDER_EMAIL_ADDR"))
	to := mail.NewEmail("Recipient", recipient)
	message := mail.NewSingleEmail(from, title, to, body, body)
	_, err := common.SendgridClient.Send(message)
	return err
}

func NotifyDweetSubscribers(event string, dweet db.DweetModel, subscribers []string) error {
	for _, subscriber := range subscribers {
		err := SendEmail("New reply to dweet on dwitter", fmt.Sprintf("The dweet with ID %s that you subscribed to has a new reply!", dweet.ID), subscriber)
		if err != nil {
			return err
		}
	}
	return nil
}

func NotifyUserSubscribersDweet(event string, dweet db.DweetModel, subscribers []string) error {
	for _, subscriber := range subscribers {
		err := SendEmail("New dweet by user on dwitter", fmt.Sprintf("The user %s that you subscribed to has a new dweet!", dweet.AuthorID), subscriber)
		if err != nil {
			return err
		}
	}
	return nil
}

func NotifyUserSubscribersRedweet(event string, redweet db.RedweetModel, subscribers []string) error {
	for _, subscriber := range subscribers {
		err := SendEmail("New redweet by user on dwitter", fmt.Sprintf("The user %s that you subscribed to has a new redweet!", redweet.AuthorID), subscriber)
		if err != nil {
			return err
		}
	}
	return nil
}
