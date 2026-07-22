package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

type servidorInternoFalso struct {
	iniciado      chan struct{}
	detener       chan struct{}
	terminado     chan struct{}
	errorEscucha  error
	errorApagado  error
	iniciarUnaVez sync.Once
	detenerUnaVez sync.Once
}

func nuevoServidorInternoFalso() *servidorInternoFalso {
	return &servidorInternoFalso{
		iniciado:  make(chan struct{}),
		detener:   make(chan struct{}),
		terminado: make(chan struct{}),
	}
}

func (s *servidorInternoFalso) EscucharYServir() error {
	s.iniciarUnaVez.Do(func() { close(s.iniciado) })
	defer close(s.terminado)
	if s.errorEscucha != nil {
		return s.errorEscucha
	}
	<-s.detener
	return nil
}

func (s *servidorInternoFalso) Apagar(context.Context) error {
	if s.errorApagado != nil {
		return s.errorApagado
	}
	s.detenerUnaVez.Do(func() { close(s.detener) })
	return nil
}

func TestServirHastaApagadoCancelaYEsperaEscucha(t *testing.T) {
	for _, cancelarAntes := range []bool{true, false} {
		t.Run(map[bool]string{true: "antes", false: "durante"}[cancelarAntes], func(t *testing.T) {
			servidor := nuevoServidorInternoFalso()
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			if cancelarAntes {
				cancelar()
			}
			resultado := make(chan error, 1)
			go func() { resultado <- servirHastaApagado(ctx, servidor) }()
			if !cancelarAntes {
				select {
				case <-servidor.iniciado:
				case <-time.After(time.Second):
					t.Fatal("escucha falsa no arranco")
				}
				cancelar()
			}
			select {
			case err := <-resultado:
				if err != nil {
					t.Fatalf("apagado = %v", err)
				}
			case <-time.After(time.Second):
				t.Fatal("servirHastaApagado no termino")
			}
			select {
			case <-servidor.terminado:
			default:
				t.Fatal("retorno antes de esperar goroutine de escucha")
			}
		})
	}
}

func TestServirHastaApagadoSaneaErrores(t *testing.T) {
	const marcador = "MARCADOR_PRIVADO_10.7.15.40_/run/secrets"
	t.Run("escucha", func(t *testing.T) {
		servidor := nuevoServidorInternoFalso()
		servidor.errorEscucha = errors.New(marcador)
		err := servirHastaApagado(context.Background(), servidor)
		if !errors.Is(err, errArranqueEscucha) || strings.Contains(err.Error(), marcador) {
			t.Fatalf("error = %v", err)
		}
	})
	t.Run("apagado", func(t *testing.T) {
		servidor := nuevoServidorInternoFalso()
		servidor.errorApagado = errors.New(marcador)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		err := servirHastaApagado(ctx, servidor)
		if !errors.Is(err, errArranqueEscucha) || strings.Contains(err.Error(), marcador) {
			t.Fatalf("error = %v", err)
		}
		servidor.detenerUnaVez.Do(func() { close(servidor.detener) })
		<-servidor.terminado
	})
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
