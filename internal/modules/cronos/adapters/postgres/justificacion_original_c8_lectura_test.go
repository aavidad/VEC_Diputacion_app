package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestOriginalC8LecturaRechazaRecursoAjenoYOrdenConTitularDelCliente(t *testing.T) {
	const ref = "permiso:cronos:solicitud:abcdefgh"
	orden, _ := json.Marshal(map[string]any{"accion": "cronos.justificacion.propia.consultar",
		"recurso_ref": ref, "correlacion_ref": "correlacion:cronos:uno"})
	c := ports.CapturaLecturaOriginalC8{ContextoCanonico: []byte("{}"), VinculoCanonico: []byte("{}"),
		Orden: orden, ValidaHasta: time.Now().Add(time.Minute)}
	if _, ok := capturaOriginalC8Valida(c, "cronos.justificacion.propia.consultar", ref); !ok {
		t.Fatal("rechazó la captura ligada al recurso")
	}
	if _, ok := capturaOriginalC8Valida(c, "cronos.justificacion.propia.consultar", "permiso:cronos:solicitud:otro1234"); ok {
		t.Fatal("aceptó recurso distinto")
	}
	c.Orden, _ = json.Marshal(map[string]any{"accion": "cronos.justificacion.propia.consultar",
		"recurso_ref": ref, "correlacion_ref": "correlacion:cronos:uno", "titular_persona_ref": "per_ajena"})
	if _, ok := capturaOriginalC8Valida(c, "cronos.justificacion.propia.consultar", ref); ok {
		t.Fatal("aceptó titular declarado en la orden")
	}
}

func TestOriginalC8ErroresNoRevelanSolicitudAjena(t *testing.T) {
	if !errors.Is(errorOriginalC8(context.Background(), &pgconn.PgError{Code: "PC012"}), ports.ErrResolucionNoCompetente) {
		t.Fatal("una solicitud ajena no conserva la denegación nominal")
	}
	if !errors.Is(errorOriginalC8(context.Background(), &pgconn.PgError{Code: "PC013"}), ports.ErrBandejaOriginalC8DemasiadoGrande) {
		t.Fatal("una bandeja grande no puede truncarse sin avisar")
	}
}
