package handler

import (
	"log"
	"net/http"

	"github.com/mini-eggs/smscp/pkg/builder"
	"github.com/mini-eggs/smscp/pkg/mode"
)

func H(w http.ResponseWriter, r *http.Request) {
	server, err := builder.Build(mode.Prod)
	if err != nil {
		log.Fatal(err)
		return
	}
	server.ServeHTTP(w, r)
}
