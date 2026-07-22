package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"strings"
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

func TestMensajesArranqueNoRegistranDireccionNiErrorCrudo(t *testing.T) {
	const marcador = "MARCADOR_PRIVADO_10.7.15.40_/run/secrets"
	for _, err := range []error{errArranqueComposicion, errArranqueEscucha} {
		var salida bytes.Buffer
		log.New(&salida, "", 0).Print(err)
		texto := salida.String()
		if strings.Contains(texto, marcador) || strings.Contains(texto, "10.7.15.40") ||
			strings.Contains(texto, "/run/secrets") {
			t.Fatalf("mensaje revelo detalle: %q", texto)
		}
	}
	errCrudo := errors.New(marcador)
	registrable := errArranqueEscucha
	if strings.Contains(registrable.Error(), errCrudo.Error()) {
		t.Fatalf("error de escucha reflejo causa cruda: %q", registrable)
	}
}
