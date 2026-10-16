package postgres

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestFalloSelectorPerfilConservaCausaYAcotaDiagnostico(t *testing.T) {
	pg := &pgconn.PgError{Code: "42883", Message: "postgres://clave-secreta@local"}
	f := falloSeleccionPerfilJustificante{etapa: "consulta", causa: pg}
	if !errors.Is(f, ports.ErrFuenteAutorizacionNoDisponible) || !errors.Is(f, pg) {
		t.Fatal("se perdió la causa técnica")
	}
	etapa, codigo := f.DiagnosticoSelectorPerfilJustificante()
	if etapa != "consulta" || codigo != "sqlstate_42883" || strings.Contains(f.Error(), "clave-secreta") {
		t.Fatal("el diagnóstico expuso detalles de conexión o perdió SQLSTATE")
	}
}
