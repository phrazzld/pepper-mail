package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/joho/godotenv"
	"log"
	"net/mail"
	"os"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	emailUser := os.Getenv("PROTONMAIL_EMAIL")
	emailPass := os.Getenv("PROTONMAIL_PASSWORD")
	imapHost := os.Getenv("PROTONMAIL_IMAP_HOST")
	imapPort := os.Getenv("PROTONMAIL_IMAP_PORT")

	if emailUser == "" || emailPass == "" || imapHost == "" || imapPort == "" {
		log.Fatal("missing email credentials")
	}

	// fetch emails
	emails, ids, err := fetchEmails(emailUser, emailPass, imapHost, imapPort)

	fmt.Println("fetched emails:", emails)
	fmt.Println("ids:", ids)

	if err := saveDraft(imapHost, imapPort, emailUser, emailPass, emailUser, "phrazzld@pm.me", "Hello there!", "It's Pepper!"); err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}
}

func fetchEmails(username, password, host, port string) ([]string, []uint32, error) {
	fmt.Println("fetching emails...")
	imapAddress := fmt.Sprintf("%s:%s", host, port)
	imapClient, err := client.Dial(imapAddress)
	fmt.Println("connected to imap server")
	if err != nil {
		return nil, nil, err
	}
	defer imapClient.Logout()

	// manually start tls session
	fmt.Println("starting tls session...")
	if err = imapClient.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		return nil, nil, err
	}

	// login
	fmt.Println("logging in...")
	if err := imapClient.Login(username, password); err != nil {
		return nil, nil, err
	}

	// select inbox
	fmt.Println("selecting inbox...")
	_, err = imapClient.Select("INBOX", false)
	if err != nil {
		return nil, nil, err
	}

	// search unseen emails
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{"\\Seen"}
	ids, err := imapClient.Search(criteria)
	fmt.Println("searched emails")
	fmt.Println("ids:", ids)
	if err != nil {
		return nil, nil, err
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

	return emails, ids, nil
}

func saveDraft(host, port, username, password, from, to, subject, body string) error {
	imapAddress := fmt.Sprintf("%s:%s", host, port)
	imapClient, err := client.Dial(imapAddress)
	if err != nil {
		return err
	}
	defer imapClient.Logout()

	if err = imapClient.StartTLS(&tls.Config{InsecureSkipVerify: true}); err != nil {
		return err
	}

	if err = imapClient.Login(username, password); err != nil {
		return err
	}

	// Create a new MIME message
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", from, to, subject, body)
	msgData := []byte(msg)

	// Use current time for the date parameter
	date := time.Now()
	// Specify flags
	flags := []string{"\\Draft"} // Adjust this based on your IMAP server capabilities

	// Append the message to the "Drafts" folder
	return imapClient.Append("Drafts", flags, date, bytes.NewReader(msgData))
}
