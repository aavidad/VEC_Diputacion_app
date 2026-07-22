package server

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/limiteshttp"
)

func TestServidorPublicoAcotaEscrituraAunqueConfiguracionSoliciteSesentaSegundos(t *testing.T) {
	servidor, err := NewHTTPServerPublico(config.Config{
		Address:          "127.0.0.1:0",
		HTTPAllowedCIDRs: []string{"127.0.0.1/32"},
		WriteTimeout:     60 * time.Second,
	}, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	if servidor.WriteTimeout != limiteshttp.DuracionMaximaPeticionPublica {
		t.Fatalf("WriteTimeout público = %s", servidor.WriteTimeout)
	}
}

func TestServidorPublicoCortaClienteQueNoLeeDentroDelPresupuesto(t *testing.T) {
	inicioEscritura := make(chan struct{})
	errorEscritura := make(chan error, 1)
	api := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(inicioEscritura)
		bloque := make([]byte, 1<<20)
		for {
			if _, err := w.Write(bloque); err != nil {
				errorEscritura <- err
				return
			}
			if escritor, ok := w.(http.Flusher); ok {
				escritor.Flush()
			}
		}
	})
	servidor, err := NewHTTPServerPublico(config.Config{
		Address:          "127.0.0.1:0",
		HTTPAllowedCIDRs: []string{"127.0.0.1/32"},
		WriteTimeout:     60 * time.Second,
	}, api)
	if err != nil {
		t.Fatal(err)
	}
	escucha, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer escucha.Close()
	finServidor := make(chan error, 1)
	go func() { finServidor <- servidor.Serve(escucha) }()
	defer func() {
		_ = servidor.Close()
		<-finServidor
	}()

	cliente, err := net.Dial("tcp", escucha.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer cliente.Close()
	if _, err := fmt.Fprintf(cliente, "GET /api/publico/dato HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	<-inicioEscritura
	inicio := time.Now()
	select {
	case err := <-errorEscritura:
		transcurrido := time.Since(inicio)
		if err == nil || transcurrido < 6*time.Second || transcurrido > 11*time.Second {
			t.Fatalf("corte de escritura en %s con error %v", transcurrido, err)
		}
	case <-time.After(11 * time.Second):
		t.Fatal("el servidor público no cortó al cliente lento")
	}
}
