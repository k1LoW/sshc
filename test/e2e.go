package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/k1LoW/sshc"
)

func main() {
	log.Println("Test ssh to bastion.")
	err := sshToBastion()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Test ssh to server by ProxyCommand.")
	err = sshToServer()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Successful ssh connection test.")
}

func sshToBastion() error {
	return ssh("bastion")
}

func sshToServer() error {
	return ssh("server")
}

func ssh(dest string) error {
	client, err := sshc.NewClient(dest)
	if err != nil {
		log.Fatal(err)
	}

	session, _ := client.NewSession()
	defer session.Close()

	var stdout = &bytes.Buffer{}
	session.Stdout = stdout
	err = session.Run("hostname")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	if stdout.String() != fmt.Sprintf("%s\n", dest) {
		return errors.New(fmt.Sprintf("Failed to exec `hostname`: expected: %s, actual: %s", dest, stdout.String()))
	}

	return nil
}
