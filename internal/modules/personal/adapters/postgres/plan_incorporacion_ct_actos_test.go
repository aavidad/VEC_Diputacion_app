package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"testing"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

func TestPlanCTActoUsaFachadaMismaTransaccion(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	tx := &txP{fila: filaP{vals: []any{reciboAltaB2Prueba(t, a)}}}
	r, _ := nuevoRepositorioActosPlanCT(&poolP{tx: tx})
	rec, e := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a})
	if e != nil || rec.Recibo.Tipo != "alta" || tx.commits != 1 || tx.q[1] != registrarAltaPlanCTSQL {
		t.Fatal("acto fuera fachada propietaria", e)
	}
}
func TestPlanCTActoFuenteDivergenteRevierteTodo(t *testing.T) {
	m := materialAltaB2Prueba(t)
	a := atestacionActoB2Prueba(t, m, domain.AccionAltaEmpleadoB2, domain.AudienciaAltaEmpleadoB2)
	tx := &txP{fila: filaP{err: &pgconn.PgError{Code: "23505"}}}
	r, _ := nuevoRepositorioActosPlanCT(&poolP{tx: tx})
	if _, e := r.RegistrarEmpleadoRRHH(context.Background(), ports.OrdenAltaEmpleadoB2{Material: m, Autorizacion: a}); !errors.Is(e, domain.ErrRegistroEmpleadoB2Conflicto) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("conflicto confirma acto", e)
	}
}
