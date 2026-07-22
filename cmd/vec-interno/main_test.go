package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

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

func TestErrorLogHTTPEmiteSoloEvidenciaFijaConRateLimit(t *testing.T) {
	const marcador = "MARCADOR_CRUDO_10.7.15.40:8443_handshake"
	var salida bytes.Buffer
	instante := time.Unix(100, 0)
	escritor := nuevoEscritorEventosHTTPSaneados(&salida)
	escritor.ahora = func() time.Time { return instante }
	registro := log.New(escritor, "", 0)
	registro.Printf("error desde %s: %s", marcador, errors.New(marcador))
	registro.Printf("repetido %s", marcador)
	if texto := salida.String(); texto != mensajeEventoHTTPSaneado || strings.Contains(texto, marcador) {
		t.Fatalf("evidencia inicial = %q", texto)
	}
	instante = instante.Add(intervaloEventoHTTPSaneado)
	registro.Printf("nuevo intervalo %s", marcador)
	if texto := salida.String(); texto != mensajeEventoHTTPSaneado+mensajeEventoHTTPSaneado || strings.Contains(texto, marcador) {
		t.Fatalf("evidencia tras intervalo = %q", texto)
	}
}

func TestErrorLogHTTPSaneadoEsSeguroEnConcurrencia(t *testing.T) {
	var salida bytes.Buffer
	escritor := nuevoEscritorEventosHTTPSaneados(&salida)
	escritor.ahora = func() time.Time { return time.Unix(100, 0) }
	var grupo sync.WaitGroup
	for indice := 0; indice < 32; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			_, _ = escritor.Write([]byte("MARCADOR_CRUDO"))
		}()
	}
	grupo.Wait()
	if salida.String() != mensajeEventoHTTPSaneado {
		t.Fatalf("evidencia concurrente = %q", salida.String())
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
