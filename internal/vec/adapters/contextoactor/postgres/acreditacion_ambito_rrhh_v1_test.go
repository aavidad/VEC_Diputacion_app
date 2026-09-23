package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/ports"
)

type filaSelectorAmbitoRRHHV1Prueba struct{ valido bool }

func (f filaSelectorAmbitoRRHHV1Prueba) Scan(destinos ...any) error {
	*destinos[0].(*bool) = f.valido
	return nil
}

type lectorSelectorAmbitoRRHHV1Prueba struct {
	valido bool
	sql    string
	login  string
}

func (l *lectorSelectorAmbitoRRHHV1Prueba) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	l.sql = sql
	l.login = args[0].(string)
	return filaSelectorAmbitoRRHHV1Prueba{valido: l.valido}
}

func TestSelectorAmbitoRRHHV1RechazaLoginNoExclusivo(t *testing.T) {
	l := &lectorSelectorAmbitoRRHHV1Prueba{valido: true}
	if err := acreditarSelectorAmbitoRRHHV1(context.Background(), l, "ca_selector_nominal"); err != nil {
		t.Fatal(err)
	}
	if l.login != "ca_selector_nominal" || !strings.Contains(l.sql, "vec_contexto_actor_corporativo_rrhh_selector") ||
		!strings.Contains(l.sql, "count(*)") || !strings.Contains(l.sql, "NOT m.set_option") {
		t.Fatal("frontera selector no acredita grupo nominal unico")
	}
	l.valido = false
	if err := acreditarSelectorAmbitoRRHHV1(context.Background(), l, "ca_selector_nominal"); !errors.Is(err, ports.ErrComprobanteAmbitoCorporativoRRHHV1Invalido) {
		t.Fatalf("selector sin grupo aceptado: %v", err)
	}
}
