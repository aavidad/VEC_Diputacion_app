package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const correlacionVECReservadaPrueba = "correlacion_0123456789abcdef0123456789abcdef"

func TestGeneradorCorrelacionVECReservadaAcunaReferenciaUnaVez(t *testing.T) {
	generador, err := nuevoGeneradorCorrelacionVECReservada(
		correlacionVECReservadaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(
		context.Background(),
		generador,
	)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := capacidad.ValorCanonico()
	if err != nil || obtenida != correlacionVECReservadaPrueba {
		t.Fatalf("correlacion inesperada: %q, %v", obtenida, err)
	}
	if obtenida, err = generador.NuevaReferenciaCorrelacionAutorizacionV2(
		context.Background(),
	); obtenida != "" ||
		!errors.Is(err, errCorrelacionVECReservadaNoDisponible) {
		t.Fatalf("la correlacion se entrego dos veces: %q, %v", obtenida, err)
	}
}

func TestGeneradorCorrelacionVECReservadaFallaCerrado(t *testing.T) {
	for _, referencia := range []string{
		"",
		"correlacion_0123456789abcdef",
		"correlacion_0123456789ABCDEF0123456789ABCDEF",
		"correlacion_0123456789abcdef0123456789abcdeg",
		strings.Repeat("a", 64),
	} {
		generador, err := nuevoGeneradorCorrelacionVECReservada(referencia)
		if generador != nil ||
			!errors.Is(err, errCorrelacionVECReservadaNoDisponible) {
			t.Fatalf("correlacion invalida aceptada: %q, %v", referencia, err)
		}
	}
	var generador *generadorCorrelacionVECReservada
	if valor, err := generador.NuevaReferenciaCorrelacionAutorizacionV2(
		context.Background(),
	); valor != "" ||
		!errors.Is(err, errCorrelacionVECReservadaNoDisponible) {
		t.Fatalf("receptor nulo aceptado: %q, %v", valor, err)
	}
	valido, err := nuevoGeneradorCorrelacionVECReservada(
		correlacionVECReservadaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if valor, err := valido.NuevaReferenciaCorrelacionAutorizacionV2(cancelado); valor != "" || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelacion ignorada: %q, %v", valor, err)
	}
	if valor, err := valido.NuevaReferenciaCorrelacionAutorizacionV2(
		context.Background(),
	); valor != correlacionVECReservadaPrueba || err != nil {
		t.Fatalf("cancelar consumio la referencia: %q, %v", valor, err)
	}
}

func TestGeneradorCorrelacionVECReservadaSoloTieneUnGanadorConcurrente(
	t *testing.T,
) {
	generador, err := nuevoGeneradorCorrelacionVECReservada(
		correlacionVECReservadaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	const participantes = 64
	var entregas atomic.Uint32
	var rechazos atomic.Uint32
	var espera sync.WaitGroup
	espera.Add(participantes)
	for range participantes {
		go func() {
			defer espera.Done()
			valor, err := generador.NuevaReferenciaCorrelacionAutorizacionV2(
				context.Background(),
			)
			if err == nil && valor == correlacionVECReservadaPrueba {
				entregas.Add(1)
				return
			}
			if valor == "" &&
				errors.Is(err, errCorrelacionVECReservadaNoDisponible) {
				rechazos.Add(1)
			}
		}()
	}
	espera.Wait()
	if entregas.Load() != 1 || rechazos.Load() != participantes-1 {
		t.Fatalf(
			"resultado concurrente inseguro: entregas=%d rechazos=%d",
			entregas.Load(),
			rechazos.Load(),
		)
	}
}

func TestGeneradorCorrelacionVECReservadaRedactaFormatosYLogs(t *testing.T) {
	generador, err := nuevoGeneradorCorrelacionVECReservada(
		correlacionVECReservadaPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, representacion := range []string{
		fmt.Sprintf("%v", generador),
		fmt.Sprintf("%+v", generador),
		fmt.Sprintf("%#v", generador),
		fmt.Sprintf("%s", generador),
		fmt.Sprintf("%q", generador),
	} {
		if strings.Contains(representacion, correlacionVECReservadaPrueba) ||
			!strings.Contains(representacion, "REDACTADO") {
			t.Fatalf("formato no redactado: %q", representacion)
		}
	}
	var salida bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&salida, nil))
	logger.Info("prueba", slog.Any("generador", generador))
	if strings.Contains(salida.String(), correlacionVECReservadaPrueba) ||
		!strings.Contains(salida.String(), "REDACTADO") {
		t.Fatalf("log no redactado: %s", salida.String())
	}
}
