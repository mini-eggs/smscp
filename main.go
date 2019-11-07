package main

import (
	"log"

	"smscp.xyz/pkg/builder"
	"smscp.xyz/pkg/mode"
)

func main() {
	server, err := builder.Build(mode.MODE_DEV)
	if err != nil {
		log.Fatal(err)
		return
	}
	if err = server.Run(); err != nil {
		log.Fatal(err)
		return
	}
}
