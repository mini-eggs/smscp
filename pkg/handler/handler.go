package handler

import (
	"log"
	"net/http"

	"smscp.xyz/pkg/builder"
)

func H(w http.ResponseWriter, r *http.Request) {
	server, err := builder.Build()
	if err != nil {
		log.Fatal(err)
		return
	}
	server.ServeHTTP(w, r)
}
