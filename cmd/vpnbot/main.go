package main

import (
	"log"

	vpnbot "vpnBot/vpnbot"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	vpnbot.Run()
	return nil
}

