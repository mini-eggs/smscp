package handler

import (
	"log"
	"net/http"

	"smscp.xyz/pkg/builder"
	"smscp.xyz/pkg/mode"
)

func H(w http.ResponseWriter, r *http.Request) {
	server, err := builder.Build(mode.ModeProd)
	if err != nil {
		log.Fatal(err)
		return
	}
	server.ServeHTTP(w, r)
}
