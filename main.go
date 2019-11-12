package main

import (
	"log"

	// only used in dev
	// but dep is used to minify web/html/*
	// before deploy
	_ "github.com/tdewolff/minify/cmd/minify"
	"smscp.xyz/pkg/builder"
	"smscp.xyz/pkg/mode"
)

func main() {
	server, err := builder.Build(mode.Dev)
	if err != nil {
		log.Fatal(err)
		return
	}
	if err = server.Run(); err != nil {
		log.Fatal(err)
		return
	}
}
