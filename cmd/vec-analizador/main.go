package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/analisiscontenido"
)

func main() {
	escucha := os.Getenv("VEC_ANALIZADOR_ESCUCHA")
	host, _, err := net.SplitHostPort(escucha)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		log.Fatal("configuracion_invalida")
	}
	identidades := strings.Split(os.Getenv("VEC_ANALIZADOR_CLIENTES_URI"), ",")
	servicio, err := analisiscontenido.NuevoServicio(analisiscontenido.ConfiguracionServicio{
		ConectorID: os.Getenv("VEC_ANALIZADOR_CONECTOR_ID"), ClamdSocket: os.Getenv("VEC_ANALIZADOR_CLAMD_SOCKET"),
		Bwrap: "/usr/bin/bwrap", QPDF: "/usr/bin/qpdf", Prlimit: "/usr/bin/prlimit",
		IdentidadesCliente: identidades, MaxEdadFirmas: 48 * time.Hour,
	})
	if err != nil {
		log.Fatal("configuracion_invalida")
	}
	tlsConfig, err := servicio.TLSConfig(os.Getenv("VEC_ANALIZADOR_TLS_CERT"), os.Getenv("VEC_ANALIZADOR_TLS_KEY"), os.Getenv("VEC_ANALIZADOR_CLIENTE_CA"))
	if err != nil {
		log.Fatal("tls_invalido")
	}
	servidor := &http.Server{
		Addr: escucha, Handler: servicio.Handler(), TLSConfig: tlsConfig,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 10 * time.Second,
		MaxHeaderBytes: 8 << 10, ErrorLog: log.New(io.Discard, "", 0),
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	go func() {
		<-ctx.Done()
		limite, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = servidor.Shutdown(limite)
	}()
	if err = servidor.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		log.Fatal("servicio_no_disponible")
	}
}
