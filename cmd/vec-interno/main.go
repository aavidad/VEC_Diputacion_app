package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/interna"
)

const tiempoMaximoApagado = 10 * time.Second

type servidorInternoEjecutable interface {
	EscucharYServir() error
	Apagar(context.Context) error
}

var (
	errArranqueComposicion = errors.New("servidor interno: composicion no disponible")
	errArranqueEscucha     = errors.New("servidor interno: escucha TLS no disponible")
)

func main() {
	if err := ejecutar(); err != nil {
		log.Fatal(err)
	}
}

func ejecutar() error {
	// Esta raíz no compone reglas de ejemplo: declarar cualquier catálogo
	// impide arrancar. Fuera de la doble llave de desarrollo se rechaza como en
	// el resto de raíces; con ella tampoco se ignora, porque aquí no se usaría.
	if err := config.Load().RechazarReglasEjemploSinComposicion(); err != nil {
		return err
	}
	cfg := interna.CargarConfiguracion()
	aplicacion, err := interna.NuevaAplicacion(context.Background(), cfg)
	if err != nil {
		return errArranqueComposicion
	}
	defer aplicacion.Cerrar()
	log.Print("servidor interno VEC iniciando escucha TLS mutua")
	ctx, detenerSenales := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer detenerSenales()
	return servirHastaApagado(ctx, aplicacion)
}

func servirHastaApagado(ctx context.Context, servidor servidorInternoEjecutable) error {
	if ctx == nil || servidor == nil {
		return errArranqueComposicion
	}
	terminado := make(chan error, 1)
	go func() { terminado <- servidor.EscucharYServir() }()
	select {
	case err := <-terminado:
		if err != nil {
			return errArranqueEscucha
		}
		return nil
	case <-ctx.Done():
		ctxApagado, cancelar := context.WithTimeout(context.Background(), tiempoMaximoApagado)
		defer cancelar()
		if err := servidor.Apagar(ctxApagado); err != nil {
			return errArranqueEscucha
		}
		select {
		case <-terminado:
			return nil
		case <-ctxApagado.Done():
			return errArranqueEscucha
		}
	}
}
