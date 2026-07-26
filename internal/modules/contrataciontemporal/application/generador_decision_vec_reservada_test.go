package application

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

func TestGeneradorDecisionVECReservadaEntregaReferenciaUnaVez(t *testing.T) {
	const esperada = "decision_vec_reservada_01"
	generador, err := nuevoGeneradorDecisionVECReservada(esperada)
	if err != nil {
		t.Fatal(err)
	}
	obtenida, err := generador.NuevaReferenciaDecisionAutorizacion()
	if err != nil || obtenida != esperada {
		t.Fatalf("primera entrega inesperada: %q, %v", obtenida, err)
	}
	if obtenida, err = generador.NuevaReferenciaDecisionAutorizacion(); obtenida != "" ||
		!errors.Is(err, errReferenciaDecisionVECReservadaNoDisponible) {
		t.Fatalf("la referencia se entrego dos veces: %q, %v", obtenida, err)
	}
}

func TestGeneradorDecisionVECReservadaRechazaReferenciasInvalidas(t *testing.T) {
	for _, referencia := range []string{
		"",
		" con_espacio",
		"con_espacio ",
		"con*comodin",
		string([]byte{0x1f}),
		string(make([]byte, 513)),
	} {
		if generador, err := nuevoGeneradorDecisionVECReservada(referencia); generador != nil ||
			!errors.Is(err, errReferenciaDecisionVECReservadaNoDisponible) {
			t.Fatalf("referencia invalida aceptada: %q, %v", referencia, err)
		}
	}
	var generador *generadorDecisionVECReservada
	if referencia, err := generador.NuevaReferenciaDecisionAutorizacion(); referencia != "" ||
		!errors.Is(err, errReferenciaDecisionVECReservadaNoDisponible) {
		t.Fatalf("receptor nulo aceptado: %q, %v", referencia, err)
	}
}

func TestGeneradorDecisionVECReservadaSoloTieneUnGanadorConcurrente(
	t *testing.T,
) {
	const esperada = "decision_vec_reservada_concurrente_01"
	generador, err := nuevoGeneradorDecisionVECReservada(esperada)
	if err != nil {
		t.Fatal(err)
	}
	const participantes = 64
	var entregas atomic.Uint32
	var errores atomic.Uint32
	var espera sync.WaitGroup
	espera.Add(participantes)
	for range participantes {
		go func() {
			defer espera.Done()
			referencia, err := generador.NuevaReferenciaDecisionAutorizacion()
			if err == nil && referencia == esperada {
				entregas.Add(1)
				return
			}
			if referencia == "" &&
				errors.Is(err, errReferenciaDecisionVECReservadaNoDisponible) {
				errores.Add(1)
			}
		}()
	}
	espera.Wait()
	if entregas.Load() != 1 || errores.Load() != participantes-1 {
		t.Fatalf(
			"resultado concurrente inseguro: entregas=%d errores=%d",
			entregas.Load(),
			errores.Load(),
		)
	}
}

func TestGeneradorDecisionVECReservadaRedactaFormatosYLogs(t *testing.T) {
	const referencia = "decision_vec_reservada_confidencial_01"
	generador, err := nuevoGeneradorDecisionVECReservada(referencia)
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
		if strings.Contains(representacion, referencia) ||
			!strings.Contains(representacion, "REDACTADO") {
			t.Fatalf("formato no redactado: %q", representacion)
		}
	}
	var salida bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&salida, nil))
	logger.Info("prueba", slog.Any("generador", generador))
	if strings.Contains(salida.String(), referencia) ||
		!strings.Contains(salida.String(), "REDACTADO") {
		t.Fatalf("log no redactado: %s", salida.String())
	}
}
