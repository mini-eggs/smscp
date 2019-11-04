package main

import (
	"log"

	"smscp.xyz/pkg/builder"
)

func main() {
	server, err := builder.Build()
	if err != nil {
		log.Fatal(err)
		return
	}
	if err = server.Run(); err != nil {
		log.Fatal(err)
		return
	}
}
