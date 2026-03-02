package main

import (
	"log"

	"vpnBot/vpnbot/telegram"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	telegram.Run()
	return nil
}

