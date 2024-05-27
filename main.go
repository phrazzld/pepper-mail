package main

import (
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/joho/godotenv"
	"log"
	"net/mail"
	"os"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	emailUser := os.Getenv("PROTONMAIL_EMAIL")
	emailPass := os.Getenv("PROTONMAIL_PASSWORD")

	// fetch emails
	emails, err := fetchEmails(emailUser, emailPass)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("fetched emails:", emails)
}

func fetchEmails(username, password string) ([]string, error) {
	/* imapClient, err := client.DialTLS("127.0.0.1:1143", &tls.Config{ */
	/* 	InsecureSkipVerify: true, */
	/* }) */
	imapClient, err := client.Dial("127.0.0.1:1143")
	if err != nil {
		return nil, err
	}
	defer imapClient.Logout()

	// manually start tls session
	if err = imapClient.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		return nil, err
	}

	// login
	if err := imapClient.Login(username, password); err != nil {
		return nil, err
	}

	// select inbox
	_, err = imapClient.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	// search unseen emails
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	ids, err := imapClient.Search(criteria)
	if err != nil {
		return nil, err
	}

	var emails []string
	if len(ids) > 0 {
		seqset := new(imap.SeqSet)
		seqset.AddNum(ids...)
		section := &imap.BodySectionName{}
		items := []imap.FetchItem{section.FetchItem()}

		messages := make(chan *imap.Message, 10)
		go func() {
			if err := imapClient.Fetch(seqset, items, messages); err != nil {
				log.Fatal(err)
			}
		}()

		for msg := range messages {
			r := msg.GetBody(section)
			if r == nil {
				continue
			}

			mailMessage, _ := mail.ReadMessage(r)
			body := make([]byte, 4096)
			_, err := mailMessage.Body.Read(body)
			if err != nil {
				log.Println(err)
				continue
			}
			emails = append(emails, string(body))
		}
	}

	return emails, nil
}
