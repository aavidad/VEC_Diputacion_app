package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"
)

var errEscucha = errors.New("escucha del servidor terminada")

// ejecutarServidor espera al listener o a la señal. Los recursos se cierran
// después de Shutdown, nunca desde sus callbacks concurrentes.
func ejecutarServidor(ctx context.Context, srv *http.Server, servir func() error, cerrar func(context.Context) error, plazo time.Duration) error {
	terminado := make(chan error, 1)
	go func() { terminado <- servir() }()
	select {
	case err := <-terminado:
		log.Print("vec servidor: inicio del cierre")
		_ = srv.Close()
		ctxCierre, cancelar := context.WithTimeout(context.Background(), plazo)
		defer cancelar()
		cierreErr := cerrarRecursos(ctxCierre, cerrar)
		log.Print("vec servidor: fin del cierre")
		if errors.Is(err, http.ErrServerClosed) {
			return cierreErr
		}
		return errors.Join(errEscucha, err, cierreErr)
	case <-ctx.Done():
		log.Print("vec servidor: inicio del cierre")
		limite := time.Now().Add(plazo)
		// Se reserva tiempo para vaciar el emisor y cerrar los pools dentro
		// del mismo plazo total, también si una petición agota Shutdown.
		reserva := min(time.Second, plazo/2)
		ctxHTTP, cancelarHTTP := context.WithDeadline(context.Background(), limite.Add(-reserva))
		err := srv.Shutdown(ctxHTTP)
		cancelarHTTP()
		if err != nil {
			_ = srv.Close()
		}
		<-terminado
		ctxCierre, cancelarCierre := context.WithDeadline(context.Background(), limite)
		defer cancelarCierre()
		cierreErr := cerrarRecursos(ctxCierre, cerrar)
		log.Print("vec servidor: fin del cierre")
		return errors.Join(err, cierreErr)
	}
}

func cerrarRecursos(ctx context.Context, cerrar func(context.Context) error) error {
	if cerrar == nil {
		return nil
	}
	resultado := make(chan error, 1)
	go func() { resultado <- cerrar(ctx) }()
	select {
	case err := <-resultado:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
