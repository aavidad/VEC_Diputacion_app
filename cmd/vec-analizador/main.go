package main

import (
	"context"
	"errors"
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

var errArranque = errors.New("analizador_no_disponible")

func main() {
	if ejecutar() != nil {
		log.Fatal("analizador_no_disponible")
	}
}

func ejecutar() (resultado error) {
	escucha := os.Getenv("VEC_ANALIZADOR_ESCUCHA")
	host, _, err := net.SplitHostPort(escucha)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || !ip.IsLoopback() {
		return errArranque
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	control, err := analisiscontenido.IniciarClamdControlado(ctx, analisiscontenido.ConfiguracionClamdControlado{
		Bwrap: "/usr/bin/bwrap", ClamdBin: os.Getenv("VEC_ANALIZADOR_CLAMD_BIN"), Prlimit: "/usr/bin/prlimit",
		RutaConfiguracion: os.Getenv("VEC_ANALIZADOR_CLAMD_CONFIG"), DBDir: os.Getenv("VEC_ANALIZADOR_CLAMD_DB_DIR"),
		SocketDir: os.Getenv("VEC_ANALIZADOR_CLAMD_SOCKET_DIR"), MaxEdadFirmas: 48 * time.Hour,
	})
	if err != nil {
		return errArranque
	}
	defer func() {
		if control.Cerrar() != nil {
			resultado = errors.Join(resultado, errArranque)
		}
	}()
	servicio, err := analisiscontenido.NuevoServicio(analisiscontenido.ConfiguracionServicio{
		ConectorID: os.Getenv("VEC_ANALIZADOR_CONECTOR_ID"), ClamdSocket: control.Socket(), Control: control,
		Bwrap: "/usr/bin/bwrap", QPDF: "/usr/bin/qpdf", Prlimit: "/usr/bin/prlimit",
		IdentidadesCliente: strings.Split(os.Getenv("VEC_ANALIZADOR_CLIENTES_URI"), ","), MaxEdadFirmas: 48 * time.Hour,
	})
	if err != nil {
		return errArranque
	}
	tlsConfig, err := servicio.TLSConfig(os.Getenv("VEC_ANALIZADOR_TLS_CERT"), os.Getenv("VEC_ANALIZADOR_TLS_KEY"), os.Getenv("VEC_ANALIZADOR_CLIENTE_CA"))
	if err != nil {
		return errArranque
	}
	servidor := &http.Server{
		Addr: escucha, Handler: servicio.Handler(), TLSConfig: tlsConfig,
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 45 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 10 * time.Second,
		MaxHeaderBytes: 8 << 10, ErrorLog: log.New(io.Discard, "", 0),
	}
	terminado := make(chan struct{})
	go func() {
		<-ctx.Done()
		limite, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = servidor.Shutdown(limite)
		close(terminado)
	}()
	err = servidor.ListenAndServeTLS("", "")
	if ctx.Err() != nil {
		<-terminado
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errArranque
	}
	return nil
}
