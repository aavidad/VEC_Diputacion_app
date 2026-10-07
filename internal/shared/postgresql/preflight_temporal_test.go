package postgresql

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type filaTEMPPrueba struct {
	publico bool
	login   bool
	err     error
}

func TestNuevoPoolConPreflightTEMPRechazaConexionFallida(t *testing.T) {
	const dsn = "host=/nonexistent/vec-preflight-temp user=vec_prueba dbname=db sslmode=disable"
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancelar()
	pool, err := NuevoPoolConPreflightTEMP(ctx, cfg)
	if pool != nil || !errors.Is(err, ErrTEMPArranque) || strings.Contains(err.Error(), "/nonexistent") {
		t.Fatalf("conexión fallida no cerrada u opaca: pool=%v err=%v", pool, err)
	}
}

func (f filaTEMPPrueba) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	*destinos[0].(*bool) = f.publico
	*destinos[1].(*bool) = f.login
	return nil
}

type consultaTEMPPrueba struct{ fila filaTEMPPrueba }

func (c consultaTEMPPrueba) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	if !strings.Contains(sql, "acldefault('d', base.datdba)") ||
		!strings.Contains(sql, "acl.grantee = 0") ||
		!strings.Contains(sql, "has_database_privilege(rol.oid, base.oid, 'TEMPORARY')") ||
		!strings.Contains(sql, "rol.rolname = session_user") {
		return filaTEMPPrueba{err: errors.New("consulta de privilegios incompleta")}
	}
	return c.fila
}

func TestComprobarTEMPArranque(t *testing.T) {
	casos := []struct {
		nombre string
		fila   filaTEMPPrueba
		clave  string
	}{
		{"base_cerrada", filaTEMPPrueba{}, ""},
		{"public_temp", filaTEMPPrueba{publico: true}, "public_temp"},
		{"login_temp_directo_o_heredado", filaTEMPPrueba{login: true}, "login_app_temp"},
		{"sin_metadatos", filaTEMPPrueba{err: pgx.ErrNoRows}, "lectura_acl"},
		{"lectura_fallida", filaTEMPPrueba{err: errors.New("DSN secreto")}, "lectura_acl"},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			err := ComprobarTEMPArranque(context.Background(), consultaTEMPPrueba{caso.fila})
			if caso.clave == "" {
				if err != nil {
					t.Fatalf("base cerrada rechazada: %v", err)
				}
				return
			}
			if !errors.Is(err, ErrTEMPArranque) || !strings.Contains(err.Error(), "clave="+caso.clave) ||
				strings.Contains(err.Error(), "DSN secreto") {
				t.Fatalf("fallo no cerrado u opaco: %v", err)
			}
		})
	}
	if err := ComprobarTEMPArranque(context.Background(), nil); !errors.Is(err, ErrTEMPArranque) {
		t.Fatalf("consulta ausente: %v", err)
	}
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	if err := ComprobarTEMPArranque(cancelado, consultaTEMPPrueba{}); !errors.Is(err, ErrTEMPArranque) {
		t.Fatalf("contexto cancelado: %v", err)
	}
}
