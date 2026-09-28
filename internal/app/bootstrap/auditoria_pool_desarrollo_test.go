package bootstrap

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/auditoria"
)

type filaPoolAuditoriaCTPrueba struct {
	login  string
	valido bool
}

func (f filaPoolAuditoriaCTPrueba) Scan(destino ...any) error {
	*destino[0].(*string) = f.login
	*destino[1].(*bool) = f.valido
	return nil
}

type consultadorPoolAuditoriaCTPrueba struct {
	fila filaPoolAuditoriaCTPrueba
	args []any
}

func (q *consultadorPoolAuditoriaCTPrueba) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	q.args = args
	return q.fila
}

func TestPoolAuditoriaCTExigeLoginNominalYFuncionPropia(t *testing.T) {
	const login = "login_ct_consultor_auditoria"
	q := &consultadorPoolAuditoriaCTPrueba{fila: filaPoolAuditoriaCTPrueba{login: login, valido: true}}
	if err := comprobarPoolConsultaAuditoriaCTDesarrollo(t.Context(), q, login); err != nil || len(q.args) != 3 ||
		q.args[0] != rolConsultaAuditoriaCTDesarrollo || q.args[1] != funcionConsultaAuditoriaCTDesarrollo ||
		q.args[2] != funcionConsultaAuditoriaBolsaDesarrollo {
		t.Fatalf("login/funciones nominales rechazados: error=%v args=%v", err, q.args)
	}
	for _, caso := range []filaPoolAuditoriaCTPrueba{{login: login, valido: false}, {login: "login_ajeno", valido: true}} {
		q.fila = caso
		if err := comprobarPoolConsultaAuditoriaCTDesarrollo(t.Context(), q, login); !errors.Is(err, auditoria.ErrNoDisponible) {
			t.Fatalf("identidad ajena admitida: %+v, %v", caso, err)
		}
	}
}
