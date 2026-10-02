// vec-baremador-web es una herramienta local sin efectos administrativos.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	os.Exit(ejecutar(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func ejecutar(ctx context.Context, args []string, salida, diagnostico io.Writer) int {
	fs := flag.NewFlagSet("vec-baremador-web", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	puerto := fs.Int("puerto", 0, "")
	dir := fs.String("web-dir", "web/static", "")
	if fs.Parse(args) != nil || fs.NArg() != 0 || *puerto < 0 || *puerto > 65535 {
		fmt.Fprintln(diagnostico, "argumentos_invalidos")
		return 2
	}
	assets, err := cargarRecursos(*dir)
	if err != nil {
		fmt.Fprintln(diagnostico, "recursos_no_disponibles", codigoCausa(err))
		return 2
	}
	ln, err := net.Listen("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(*puerto)))
	if err != nil {
		fmt.Fprintln(diagnostico, "escucha_no_disponible", codigoCausa(err))
		return 2
	}
	defer ln.Close()
	host := ln.Addr().String()
	servidor := &http.Server{Handler: nuevoHandler(host, assets), ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 * 1024}
	fin := make(chan struct{})
	defer close(fin)
	go func() {
		select {
		case <-ctx.Done():
			apagar, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancelar()
			_ = servidor.Shutdown(apagar)
		case <-fin:
		}
	}()
	fmt.Fprintln(salida, "http://"+host+entradaWeb)
	fmt.Fprintln(salida, "http://"+host+entradaProvision)
	fmt.Fprintln(salida, "http://"+host+entradaSeleccion)
	if err := servidor.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(diagnostico, "servidor_fallido")
		return 2
	}
	return 0
}

// La causa nominal permite distinguir un fallo de arranque sin revelar el
// mensaje del SO, que puede contener rutas o configuración del operador.
func codigoCausa(err error) string {
	switch {
	case errors.Is(err, os.ErrNotExist):
		return "recurso_ausente"
	case errors.Is(err, os.ErrPermission):
		return "permiso_denegado"
	case errors.Is(err, syscall.EADDRINUSE):
		return "puerto_ocupado"
	default:
		return "dependencia_no_disponible"
	}
}
