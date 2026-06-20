package main

import (
	"errors"
	"log"
	"net/http"

	"vec-diputacion-granada/internal/app/bootstrap"
)

func main() {
	srv, err := bootstrap.NewHTTPServer()
	if err != nil {
		log.Fatalf("bootstrap server: %v", err)
	}
	log.Printf("vec server listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve: %v", err)
	}
}
