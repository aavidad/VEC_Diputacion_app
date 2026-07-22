package main

import (
	"crypto/tls"
	"errors"
	"net/http"
	"testing"

	"vec-diputacion-granada/internal/app/composicion/interna"
)

func TestValidarServidorParaEscuchaRechazaServidorIncompleto(t *testing.T) {
	for _, prueba := range []struct {
		nombre   string
		servidor *http.Server
		error    error
	}{
		{nombre: "nulo", servidor: nil, error: interna.ErrServidorInternoInvalido},
		{nombre: "sin manejador", servidor: &http.Server{}, error: interna.ErrServidorInternoInvalido},
		{
			nombre: "TLS sin verificacion de cliente",
			servidor: &http.Server{
				Handler:   http.NotFoundHandler(),
				TLSConfig: &tls.Config{MinVersion: tls.VersionTLS13, ClientAuth: tls.NoClientCert},
			},
			error: interna.ErrTLSMutuoNoVerificado,
		},
	} {
		t.Run(prueba.nombre, func(t *testing.T) {
			if err := validarServidorParaEscucha(prueba.servidor); !errors.Is(err, prueba.error) {
				t.Fatalf("error = %v; se esperaba %v", err, prueba.error)
			}
		})
	}
}
