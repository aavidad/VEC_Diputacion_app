package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type filaLoginAmbitoRRHHPrueba struct {
	valido bool
	err    error
}

func (f filaLoginAmbitoRRHHPrueba) Scan(dst ...any) error {
	if f.err != nil {
		return f.err
	}
	*dst[0].(*bool) = f.valido
	return nil
}

type lectorLoginAmbitoRRHHPrueba struct {
	valido   bool
	consulta string
	login    string
}

func (l *lectorLoginAmbitoRRHHPrueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	l.consulta = sql
	l.login = args[0].(string)
	return filaLoginAmbitoRRHHPrueba{valido: l.valido}
}

func TestAcreditarLoginNominalAmbitoRRHHExigeMembresiaUnica(t *testing.T) {
	lector := &lectorLoginAmbitoRRHHPrueba{valido: true}
	if err := acreditarLoginNominalAmbitoRRHH(context.Background(), lector, "ct_lector_nuevo"); err != nil {
		t.Fatal(err)
	}
	if lector.login != "ct_lector_nuevo" || !strings.Contains(lector.consulta, "count(*)") ||
		!strings.Contains(lector.consulta, "vec_contratacion_temporal_consultor_rrhh_ambito") ||
		!strings.Contains(lector.consulta, "NOT m.set_option") {
		t.Fatal("contrato de membresia nominal incompleto")
	}
	lector.valido = false
	if err := acreditarLoginNominalAmbitoRRHH(context.Background(), lector, "ct_lector_nuevo"); !errors.Is(err, ports.ErrConsultaRRHHNoDisponible) {
		t.Fatalf("login sin membresia aceptado: %v", err)
	}
}
