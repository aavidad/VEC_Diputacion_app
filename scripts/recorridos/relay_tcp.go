// relay_tcp transporta bytes TLS sin interpretarlos por stdin/stdout.
// Uso exclusivo del lanzador local: /relay PORT o /relay --sha256.
// El lanzador fija PORT; nunca debe obtenerlo de entradas del navegador.
package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	plazoConexion            = 3 * time.Second
	plazoTotal               = 90 * time.Second
	maxBytesSentido    int64 = 64 << 20
	maxBytesEjecutable int64 = 256 << 20
)

var errLimite = errors.New("relay_byte_limit")

func main() {
	os.Exit(ejecutar(os.Args[1:]))
}

// No escribir diagnósticos: stdout es exclusivamente el canal TLS (o la huella)
// y los errores de red pueden contener datos del entorno. El host usa el código.
func ejecutar(args []string) int {
	if len(args) != 1 {
		return 2
	}
	if args[0] == "--sha256" {
		ruta, err := os.Executable()
		if err != nil || huellaEjecutable(ruta, os.Stdout) != nil {
			return 1
		}
		return 0
	}
	puerto, err := validarPuerto(args[0])
	if err != nil {
		return 2
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancelar()
	ctx, cancelarPlazo := context.WithTimeout(ctx, plazoTotal)
	defer cancelarPlazo()
	if conectar(ctx, puerto, os.Stdin, os.Stdout) != nil {
		return 1
	}
	return 0
}

func validarPuerto(valor string) (int, error) {
	puerto, err := strconv.Atoi(valor)
	if err != nil || puerto < 1024 || puerto > 65535 || strconv.Itoa(puerto) != valor {
		return 0, errors.New("relay_invalid_port")
	}
	return puerto, nil
}

func conectar(ctx context.Context, puerto int, entrada io.ReadCloser, salida io.WriteCloser) error {
	// La función no recibe ni resuelve un host; tcp4 impide otras familias.
	direccion := net.JoinHostPort("127.0.0.1", strconv.Itoa(puerto))
	dialer := net.Dialer{Timeout: plazoConexion}
	conexion, err := dialer.DialContext(ctx, "tcp4", direccion)
	if err != nil {
		return err
	}
	tcp, ok := conexion.(*net.TCPConn)
	if !ok {
		_ = conexion.Close()
		return errors.New("relay_invalid_connection")
	}
	return transportar(ctx, tcp, entrada, salida, maxBytesSentido)
}

func transportar(ctx context.Context, tcp *net.TCPConn, entrada io.ReadCloser, salida io.WriteCloser, limite int64) error {
	ctx, cancelar := context.WithTimeout(ctx, plazoTotal)
	defer cancelar()
	cerrar := func() {
		_ = tcp.Close()
		_ = entrada.Close()
		_ = salida.Close()
	}
	if limite <= 0 {
		cerrar()
		return errLimite
	}
	if plazo, ok := ctx.Deadline(); ok {
		if err := tcp.SetDeadline(plazo); err != nil {
			cerrar()
			return err
		}
	}
	// También cierra pipes bloqueados: un deadline TCP solo no cancela stdio.
	detener := context.AfterFunc(ctx, cerrar)
	var grupo sync.WaitGroup
	defer func() {
		detener()
		cerrar()
		grupo.Wait()
	}()
	type resultado struct {
		entrada bool
		err     error
	}
	resultados := make(chan resultado, 2)
	grupo.Add(2)
	go func() {
		defer grupo.Done()
		err := copiarAcotado(tcp, entrada, limite)
		if err == nil {
			// EOF del cliente conserva el sentido TCP -> stdout para la respuesta.
			err = tcp.CloseWrite()
		}
		resultados <- resultado{entrada: true, err: err}
	}()
	go func() {
		defer grupo.Done()
		resultados <- resultado{err: copiarAcotado(salida, tcp, limite)}
	}()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-resultados:
			if r.err != nil || !r.entrada {
				return r.err
			}
		}
	}
}

func copiarAcotado(destino io.Writer, origen io.Reader, limite int64) error {
	bytes, err := io.Copy(destino, io.LimitReader(origen, limite))
	if err != nil {
		return err
	}
	// Alcanzar el límite termina el canal sin copiar un byte adicional.
	if bytes == limite {
		return errLimite
	}
	return nil
}

func huellaEjecutable(ruta string, salida io.Writer) error {
	// ruta procede de os.Executable, nunca del cliente. O_NOFOLLOW evita enlaces.
	// #nosec G304 -- solo se abre la ruta del ejecutable propio, sin entrada externa.
	archivo, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer func() { _ = archivo.Close() }()
	info, err := archivo.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() >= maxBytesEjecutable {
		return errors.New("relay_invalid_executable")
	}
	huella := sha256.New()
	bytes, err := io.Copy(huella, io.LimitReader(archivo, maxBytesEjecutable))
	if err != nil {
		return err
	}
	posterior, err := archivo.Stat()
	if err != nil {
		return err
	}
	if bytes != info.Size() || posterior.Size() != info.Size() || !posterior.ModTime().Equal(info.ModTime()) {
		return errors.New("relay_changed_executable")
	}
	_, err = fmt.Fprintf(salida, "%x\n", huella.Sum(nil))
	return err
}
