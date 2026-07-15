// Command vec-emisor-capacidad-v4 ejecuta exclusivamente el verificador COSE
// y emisor de capacidades V4. No importa la fachada ejecutora y no debe recibir
// la credencial PostgreSQL del portal.
package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	confianza "vec-diputacion-granada/internal/vec/adapters/postgres/confianzadocumental"
)

func main() {
	if err := ejecutar(); err != nil {
		log.Fatal(err)
	}
}

func ejecutar() error {
	dsn := os.Getenv("VEC_V4_EMISOR_DATABASE_URL")
	rutaSocket := os.Getenv("VEC_V4_EMISOR_SOCKET")
	if dsn == "" || rutaSocket == "" || os.Getenv("VEC_V4_EJECUTOR_DATABASE_URL") != "" {
		return errors.New("configuracion aislada del emisor V4 invalida")
	}
	if _, err := os.Lstat(rutaSocket); !errors.Is(err, os.ErrNotExist) {
		return errors.New("el socket del emisor V4 ya existe o no puede verificarse")
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return errors.New("no se pudo configurar PostgreSQL para el emisor V4")
	}
	defer pool.Close()
	manejador, err := confianza.NuevoManejadorHTTPEmisorCapacidadDocumentalV4(ctx, pool)
	if err != nil {
		return errors.New("no se pudo inicializar el emisor V4")
	}
	mascaraAnterior := syscall.Umask(0o007)
	escucha, err := net.Listen("unix", rutaSocket)
	syscall.Umask(mascaraAnterior)
	if err != nil {
		return errors.New("no se pudo abrir el socket del emisor V4")
	}
	informacionSocket, err := os.Lstat(rutaSocket)
	if err != nil || informacionSocket.Mode()&os.ModeSocket == 0 {
		_ = escucha.Close()
		return errors.New("no se pudo verificar el socket del emisor V4")
	}
	defer cerrarYRetirarSocket(escucha, rutaSocket, informacionSocket)
	if err = os.Chmod(rutaSocket, 0o660); err != nil {
		return errors.New("no se pudo restringir el socket del emisor V4")
	}
	servidor := &http.Server{
		Handler: manejador, ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout: 6 * time.Second, WriteTimeout: 8 * time.Second,
		IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024,
	}
	fin := make(chan error, 1)
	go func() { fin <- servidor.Serve(escucha) }()
	select {
	case <-ctx.Done():
		ctxCierre, cancelarCierre := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelarCierre()
		if err = servidor.Shutdown(ctxCierre); err != nil {
			return errors.New("no se pudo cerrar el emisor V4 de forma ordenada")
		}
	case err = <-fin:
		if !errors.Is(err, http.ErrServerClosed) {
			return errors.New("el emisor V4 se detuvo de forma inesperada")
		}
	}
	return nil
}

// cerrarYRetirarSocket evita que un reinicio quede bloqueado por un socket
// obsoleto, pero nunca borra un fichero que otro proceso haya colocado en la
// misma ruta despues de abrir la escucha.
func cerrarYRetirarSocket(escucha net.Listener, ruta string, propio os.FileInfo) {
	// net.UnixListener elimina por defecto la ruta al cerrar. Se desactiva para
	// poder comprobar antes el inode y no retirar un reemplazo ajeno.
	if escuchaUnix, ok := escucha.(*net.UnixListener); ok {
		escuchaUnix.SetUnlinkOnClose(false)
	}
	_ = escucha.Close()
	actual, err := os.Lstat(ruta)
	if err == nil && propio != nil && os.SameFile(propio, actual) {
		_ = os.Remove(ruta)
	}
}
