package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type filaAsignacionP struct {
	valores []any
	err     error
}

func (f filaAsignacionP) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	for i, destino := range destinos {
		switch p := destino.(type) {
		case *string:
			*p = f.valores[i].(string)
		case *time.Time:
			*p = f.valores[i].(time.Time)
		case *int16:
			*p = f.valores[i].(int16)
		case *int64:
			*p = f.valores[i].(int64)
		default:
			return errors.New("destino no previsto")
		}
	}
	return nil
}

func ordenAsignacionP(t *testing.T) personalports.OrdenAsignacionDietas {
	t.Helper()
	base := ordenP(t)
	fecha, err := personaldomain.NuevaFechaCivil("2026-09-20")
	if err != nil {
		t.Fatal(err)
	}
	s := personaldomain.SolicitudAsignacionDietas{Actor: base.Material.Solicitud().Actor, Operacion: personaldomain.ConsultarAsignacionDietas, RelacionRef: "rel_" + strings.Repeat("a", 24), UnidadRef: "unidad:x", FechaReferencia: fecha}
	m, err := personaldomain.NuevoMaterialAsignacionDietas(s)
	if err != nil {
		t.Fatal(err)
	}
	return personalports.OrdenAsignacionDietas{Material: m, Autorizacion: base.Autorizacion}
}

func filaAsignacionCorrecta(o personalports.OrdenAsignacionDietas) filaAsignacionP {
	s := o.Material.Solicitud()
	a := o.Autorizacion.ResumenCapacidad()
	return filaAsignacionP{valores: []any{
		"rad_" + strings.Repeat("a", 32), a.DecisionRef(), a.EfectoRef(), strings.Repeat("a", 64), "auditoria:uno", a.EmitidaEn().Add(time.Microsecond), "consultada",
		"ads_" + strings.Repeat("a", 24), s.RelacionRef, s.PersonaRef, s.UnidadRef, "centro:uno", "per_" + strings.Repeat("b", 24), "per_" + strings.Repeat("c", 24), int16(2), "2026-09-01", int64(1),
	}}
}

func TestRepositorioAsignacionUsaFachadaYTransaccion(t *testing.T) {
	o := ordenAsignacionP(t)
	tx := &txP{fila: filaAsignacionCorrecta(o)}
	pool := &poolP{tx: tx}
	repo, err := nuevoRepositorioAsignacionDietasPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	r, err := repo.EjecutarAsignacionDietas(context.Background(), o)
	if err != nil || r.EstadoLocal != "consultada" || tx.commits != 1 || tx.rollbacks != 0 || pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite || len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.q[1] != consultaAsignacionDietasSQL {
		t.Fatalf("fachada: err=%v estado=%s commits=%d rollback=%d query=%v", err, r.EstadoLocal, tx.commits, tx.rollbacks, tx.q)
	}
}

func TestRepositorioAsignacionRevierteRespuestaNoConfiable(t *testing.T) {
	o := ordenAsignacionP(t)
	f := filaAsignacionCorrecta(o)
	f.valores[9] = "per_" + strings.Repeat("f", 24)
	tx := &txP{fila: f}
	repo, err := nuevoRepositorioAsignacionDietasPostgreSQL(&poolP{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.EjecutarAsignacionDietas(context.Background(), o)
	if !errors.Is(err, personalports.ErrAsignacionDietasNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("resultado ajeno: err=%v commits=%d rollback=%d", err, tx.commits, tx.rollbacks)
	}
}

func TestErroresAsignacionNoFiltranPostgres(t *testing.T) {
	for _, tc := range []struct {
		codigo string
		want   error
	}{
		{"P7201", personalports.ErrAsignacionDietasDenegada},
		{"P7203", personalports.ErrVersionAsignacionDietas},
		{"P7204", personalports.ErrClaveAsignacionDietas},
		{"42501", personalports.ErrAsignacionDietasDenegada},
		{"22023", personalports.ErrAsignacionDietasNoDisponible},
	} {
		got := normalizarErrorAsignacionDietas(context.Background(), &pgconn.PgError{Code: tc.codigo})
		if !errors.Is(got, tc.want) {
			t.Errorf("SQLSTATE %s: %v", tc.codigo, got)
		}
	}
}
