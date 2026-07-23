package ports

import (
	"context"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestImporteMaximoSeAplicaAntesYDespuesDeLaFuente(t *testing.T) {
	inicio := instanteFuenteAnalisisPrueba()
	preparacion := preparacionValidarRCPrueba()
	preparacion.Declaracion.Importe.Centimos = maximoCentimosFuente
	solicitudRC, err := nuevaSolicitudValidarRCOrquestadaPrueba(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacion,
	)
	if err != nil {
		t.Fatalf("máximo de entrada RC rechazado: %v", err)
	}
	validacion := validacionRCPrueba(t, solicitudRC, inicio.Add(time.Second))
	validacion.Importe.Centimos = maximoCentimosFuente
	metadatosRC := metadatosRespuestaPrueba(
		validacion.FuenteRef,
		validacion.ReciboRef,
		inicio,
	)
	if _, err := NuevaPreimagenRespuestaValidacionRC(
		solicitudRC,
		validacion,
		MotivoFuenteAnalisis{},
		metadatosRC,
	); err != nil {
		t.Fatalf("máximo de salida RC rechazado: %v", err)
	}
	preparacion.Declaracion.Importe.Centimos++
	if _, err := nuevaSolicitudValidarRCOrquestadaPrueba(
		context.Background(),
		generadorFijoFuenteAnalisis("pet_0123456789abcdefghijklmn"),
		selladorHMACFuenteAnalisisPrueba(),
		relojFijoFuenteAnalisis(inicio),
		preparacion,
	); err == nil {
		t.Fatal("entrada RC superior al máximo aceptada")
	}

	solicitudCoste := solicitudCalcularCostePrueba(t, inicio)
	metadatosCoste := metadatosRespuestaPrueba(
		"tabla_retributiva_2026_v3",
		"recibo_coste_0123456789",
		inicio,
	)
	if _, err := NuevaPreimagenRespuestaCalculoCoste(
		solicitudCoste,
		metadatosCoste.AutoridadRef,
		metadatosCoste.ReciboRef,
		domain.Importe{Centimos: maximoCentimosFuente, Moneda: "EUR"},
		inicio.Add(time.Second),
		metadatosCoste,
	); err != nil {
		t.Fatalf("máximo de coste rechazado: %v", err)
	}
	if _, err := NuevaPreimagenRespuestaCalculoCoste(
		solicitudCoste,
		metadatosCoste.AutoridadRef,
		metadatosCoste.ReciboRef,
		domain.Importe{Centimos: maximoCentimosFuente + 1, Moneda: "EUR"},
		inicio.Add(time.Second),
		metadatosCoste,
	); err == nil {
		t.Fatal("coste superior al máximo aceptado")
	}
}
