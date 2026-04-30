package main

import (
	"fmt"
	"log"

	"mail-assistant/internal/client"
)

func main() {
	c := client.New()
	if err := c.ConnectByXOAUTH2("<imap_server_address>", "<email>", "<token>"); err != nil {
		log.Fatal("failed to connect to the mail server: ", err)
	}
	defer c.Close()

	letters, _ := c.GetNewLetters("INBOX", 29)
	fmt.Println(len(letters))
	for i := range letters {
		fmt.Println(i, " ", letters[i].Body)
	}
}
