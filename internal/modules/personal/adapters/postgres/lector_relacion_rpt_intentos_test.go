package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
)

type poolIntentosRPTPrueba struct {
	calls int
	q     string
	args  []any
	ok    bool
	err   error
}

func (p *poolIntentosRPTPrueba) QueryRow(_ context.Context, q string, a ...any) pgx.Row {
	p.calls++
	p.q = q
	p.args = a
	return filaBoolIntentoRPTPrueba{ok: p.ok, err: p.err}
}

type filaBoolIntentoRPTPrueba struct {
	ok  bool
	err error
}

func (f filaBoolIntentoRPTPrueba) Scan(d ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(d) != 1 {
		return errors.New("scan invalido")
	}
	p, ok := d[0].(*bool)
	if !ok {
		return errors.New("scan no bool")
	}
	*p = f.ok
	return nil
}
func TestLectorRPTRegistroIntentosNominalSegregado(t *testing.T) {
	p := &poolIntentosRPTPrueba{ok: true}
	r := &RegistroIntentosLectorRPTPostgreSQL{pool: p}
	e := ports.EventoIntentoLectorRelacionRPT{CorrelacionRef: "correlacion_" + strings.Repeat("a", 32), Motivo: "denegado", RelacionRef: "rel_" + strings.Repeat("b", 24)}
	if err := r.RegistrarEventoRelacionRPT(context.Background(), e); err != nil || p.calls != 1 || p.q != registrarIntentoLectorRPTSQL || len(p.args) != 4 || p.args[2] != "" {
		t.Fatal("registro nominal fallido", err)
	}
	if err := r.VerificarDestinoRelacionRPT(context.Background()); err != nil || p.calls != 2 {
		t.Fatal("preflight fallido", err)
	}
}
func TestLectorRPTRegistroIntentosNoCopiaDetalleNiCamposLibres(t *testing.T) {
	p := &poolIntentosRPTPrueba{err: errors.New("detalle privado")}
	r := &RegistroIntentosLectorRPTPostgreSQL{pool: p}
	base := ports.EventoIntentoLectorRelacionRPT{CorrelacionRef: "correlacion_" + strings.Repeat("a", 32), Motivo: "no_disponible"}
	if err := r.RegistrarEventoRelacionRPT(context.Background(), base); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || strings.Contains(err.Error(), "privado") {
		t.Fatal("detalle filtrado", err)
	}
	base.Motivo = "mensaje libre con datos"
	previo := p.calls
	if err := r.RegistrarEventoRelacionRPT(context.Background(), base); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) || p.calls != previo {
		t.Fatal("motivo libre llegó a SQL", err)
	}
}
