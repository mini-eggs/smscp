package main

import (
	"log"

	// only used in dev
	// but dep is used to minify web/html/*
	// before deploy
	_ "github.com/tdewolff/minify"
	"github.com/mini-eggs/smscp/pkg/builder"
	"github.com/mini-eggs/smscp/pkg/mode"
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
