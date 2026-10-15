package bootstrap

import (
	"context"
	"errors"
	"os"
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

// La conexión nominal de una copia debe poder abrir Auditoría sin recibir
// USAGE de Bolsa. No crea objetos ni cambia permisos del clon.
func TestPoolAuditoriaCTSinAccesoAlEsquemaBolsaPostgreSQL18(t *testing.T) {
	dsn := os.Getenv("VEC_AUDITORIA_POOL_CT_PG18_DSN")
	if dsn == "" {
		t.Skip("sin conexión nominal CT de una copia PostgreSQL 18")
	}
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("conexión nominal CT no disponible (%T)", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(context.Background()); err != nil {
			t.Errorf("cierre de conexión CT: %T", err)
		}
	})
	var version int
	var accesoBolsa bool
	if err := conn.QueryRow(ctx, `SELECT current_setting('server_version_num')::integer/10000,
		has_schema_privilege(session_user,'vec_bolsa_llamamientos','USAGE')`).Scan(&version, &accesoBolsa); err != nil {
		t.Fatalf("preimagen del clon no disponible (%T)", err)
	}
	if version != 18 || accesoBolsa {
		t.Fatal("se requiere PostgreSQL 18 y un LOGIN CT sin USAGE de Bolsa")
	}
	pool, err := abrirPoolConsultaAuditoriaCTDesarrollo(ctx, dsn)
	if err != nil {
		t.Fatalf("el aislamiento de Bolsa impide abrir la consulta de auditoría: %v", err)
	}
	t.Cleanup(pool.Close)
}
