package postgres

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestRespuestaCancelacionConOfertaPendienteConservaRechazoDeNegocio(t *testing.T) {
	respuesta := []byte(`{"esquema":"vec.contratacion-temporal.resultado-cancelacion.v1","resultado":"seleccion_oferta_pendiente"}`)
	_, err := decodificarRespuestaSeguimiento(respuesta, "vec.contratacion-temporal.resultado-cancelacion.v1")
	if !errors.Is(err, ports.ErrCancelacionConOfertaPendiente) {
		t.Fatalf("la respuesta CT223 no se reconoce como conflicto de negocio: %v", err)
	}
}
