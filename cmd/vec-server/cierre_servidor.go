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
		cierreErr := cerrarRecursos(cerrar, plazo)
		log.Print("vec servidor: fin del cierre")
		if errors.Is(err, http.ErrServerClosed) {
			return cierreErr
		}
		return errors.Join(errEscucha, err, cierreErr)
	case <-ctx.Done():
		log.Print("vec servidor: inicio del cierre")
		ctxCierre, cancelar := context.WithTimeout(context.Background(), plazo)
		err := srv.Shutdown(ctxCierre)
		cancelar()
		if err != nil {
			_ = srv.Close()
		}
		<-terminado
		cierreErr := cerrarRecursos(cerrar, plazo)
		log.Print("vec servidor: fin del cierre")
		return errors.Join(err, cierreErr)
	}
}

func cerrarRecursos(cerrar func(context.Context) error, plazo time.Duration) error {
	if cerrar == nil {
		return nil
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazo)
	defer cancelar()
	resultado := make(chan error, 1)
	go func() { resultado <- cerrar(ctx) }()
	select {
	case err := <-resultado:
		return errors.Join(err, ctx.Err())
	case <-ctx.Done():
		return ctx.Err()
	}
}
