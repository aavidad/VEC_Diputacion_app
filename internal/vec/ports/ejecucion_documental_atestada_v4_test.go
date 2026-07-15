package ports_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

func TestResultadoConectorEjecucionDocumentalAtestadaV4EsOpacoYExacto(t *testing.T) {
	instante := time.Date(2026, time.July, 15, 18, 0, 0, 123456000, time.UTC)
	huellaA := strings.Repeat("a", 64)
	huellaB := strings.Repeat("b", 64)
	resultado, err := ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
		"efecto:documental:v4:prueba", "pendiente_generacion",
		"auditoria:documental:v4:"+huellaA,
		"evento:documental:v4:"+huellaB,
		instante,
	)
	if err != nil {
		t.Fatalf("crear resultado neutral: %v", err)
	}
	orden, errOrden := resultado.OrdenRef()
	estado, errEstado := resultado.Estado()
	auditoria, errAuditoria := resultado.AuditoriaRef()
	evento, errEvento := resultado.EventoOutboxRef()
	registradaEn, errRegistro := resultado.RegistradaEn()
	if errOrden != nil || errEstado != nil || errAuditoria != nil ||
		errEvento != nil || errRegistro != nil ||
		orden != "efecto:documental:v4:prueba" || estado != "pendiente_generacion" ||
		auditoria != "auditoria:documental:v4:"+huellaA ||
		evento != "evento:documental:v4:"+huellaB || !registradaEn.Equal(instante) {
		t.Fatal("el resultado opaco no conservo la confirmacion exacta")
	}
	if texto := fmt.Sprintf("%v %#v", resultado, resultado); strings.Contains(
		texto, orden,
	) || strings.Contains(texto, huellaA) || strings.Contains(texto, huellaB) {
		t.Fatalf("el resultado filtro referencias al formatearse: %s", texto)
	}
	if _, err := json.Marshal(resultado); !errors.Is(
		err, ports.ErrEjecucionDocumentalAtestadaV4NoDisponible,
	) {
		t.Fatalf("serializacion general no bloqueada: %v", err)
	}
}

func TestResultadoConectorEjecucionDocumentalAtestadaV4FallaCerrado(t *testing.T) {
	instante := time.Date(2026, time.July, 15, 18, 0, 0, 0, time.UTC)
	huella := strings.Repeat("a", 64)
	casos := []struct {
		nombre            string
		orden, estado     string
		auditoria, evento string
		registradaEn      time.Time
	}{
		{"cero", "", "", "", "", time.Time{}},
		{"estado", "efecto:1", "ejecutada", "auditoria:documental:v4:" + huella, "evento:documental:v4:" + strings.Repeat("b", 64), instante},
		{"auditoria", "efecto:1", "pendiente_generacion", "auditoria:documental:v4:no-huella", "evento:documental:v4:" + strings.Repeat("b", 64), instante},
		{"evento", "efecto:1", "pendiente_generacion", "auditoria:documental:v4:" + huella, "evento:documental:v4:no-huella", instante},
		{"zona", "efecto:1", "pendiente_generacion", "auditoria:documental:v4:" + huella, "evento:documental:v4:" + strings.Repeat("b", 64), instante.In(time.FixedZone("CET", 3600))},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			resultado, err := ports.NuevoResultadoConectorEjecucionDocumentalAtestadaV4(
				caso.orden, caso.estado, caso.auditoria, caso.evento, caso.registradaEn,
			)
			if !errors.Is(err, ports.ErrEjecucionDocumentalAtestadaV4NoDisponible) ||
				resultado.Validar() == nil {
				t.Fatalf("resultado invalido aceptado: %v", err)
			}
		})
	}
}
