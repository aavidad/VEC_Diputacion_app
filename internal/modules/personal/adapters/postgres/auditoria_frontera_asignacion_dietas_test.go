package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

type filaBoolAuditoriaAsignacion struct {
	valor bool
	err   error
}

func (f filaBoolAuditoriaAsignacion) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*bool) = f.valor
	return nil
}

type consultaAuditoriaAsignacionPrueba struct {
	fila  pgx.Row
	query string
	args  []any
}

func (c *consultaAuditoriaAsignacionPrueba) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	c.query, c.args = query, args
	return c.fila
}

func TestAuditoriaFronteraAsignacionContratoNominal(t *testing.T) {
	c := &consultaAuditoriaAsignacionPrueba{fila: filaBoolAuditoriaAsignacion{valor: true}}
	r, err := nuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(c)
	if err != nil || r.Preflight(context.Background()) != nil || c.query != preflightAuditoriaFronteraAsignacionSQL {
		t.Fatalf("preflight: err=%v query=%s", err, c.query)
	}
	o := personalports.OrdenAuditoriaFronteraAsignacionDietas{
		CorrelacionRef: "corr_no_disponible", Motivo: personalports.MotivoFronteraPersonalAutenticacion,
		Ruta: personalports.RutaFronteraAsignacionDetalle, Accion: "consultar", EstadoHTTP: 401,
	}
	if err := r.RegistrarAuditoriaFronteraAsignacionDietas(context.Background(), o); err != nil || c.query != registrarAuditoriaFronteraAsignacionSQL || len(c.args) != 8 || c.args[2] != personalports.SuperficieFronteraAsignacionDietas || c.args[5] != "" || c.args[6] != "" || c.args[7] != int16(401) {
		t.Fatalf("registro: err=%v query=%s args=%v", err, c.query, c.args)
	}
}

func TestAuditoriaFronteraAsignacionFallaCerrada(t *testing.T) {
	c := &consultaAuditoriaAsignacionPrueba{fila: filaBoolAuditoriaAsignacion{valor: false}}
	r, err := nuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(c)
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(r.Preflight(context.Background()), ErrAuditoriaFronteraAsignacionNoDisponible) {
		t.Fatal("preflight acepto funcion sin permiso")
	}
	o := personalports.OrdenAuditoriaFronteraAsignacionDietas{CorrelacionRef: "corr_no_disponible", Motivo: personalports.MotivoFronteraPersonalDenegado, Ruta: personalports.RutaFronteraAsignacionGrupo, Accion: "grupo_corregir", RecursoRef: "rel_aaaaaaaaaaaaaaaaaaaaaa", EstadoHTTP: 403}
	if !errors.Is(r.RegistrarAuditoriaFronteraAsignacionDietas(context.Background(), o), ErrAuditoriaFronteraAsignacionNoDisponible) {
		t.Fatal("registro falso tratado como durable")
	}
}
