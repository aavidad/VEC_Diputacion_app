package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestRegistroEmpleadoB2ConfirmaSoloRespuestaValidada(t *testing.T) {
	autorizacion := ordenP(t).Autorizacion
	tx := &txP{fila: filaP{vals: []any{[]byte(`{"ok":true}`)}}}
	pool := &poolP{tx: tx}
	resultado, err := ejecutarRegistroEmpleadoB2(context.Background(), pool, consultaFichaEmpleadoB2SQL, []byte(`{"esquema":"prueba"}`), autorizacion, 100, func(b []byte) (string, error) {
		return string(b), nil
	})
	if err != nil || resultado != `{"ok":true}` || pool.o.IsoLevel != pgx.Serializable || pool.o.AccessMode != pgx.ReadWrite || tx.commits != 1 || tx.rollbacks != 0 {
		t.Fatal("no confirmó consulta nominal validada", err)
	}
	if len(tx.q) != 2 || tx.q[1] != consultaFichaEmpleadoB2SQL || len(tx.a) != 1 || len(tx.a[0]) != 11 || tx.a[0][0] != `{"esquema":"prueba"}` || strings.Contains(strings.ToLower(tx.q[1]), " from ") {
		t.Fatal("consulta no nominal o argumentos incompletos")
	}
}

func TestRegistroEmpleadoB2RevierteRespuestaAlterada(t *testing.T) {
	autorizacion := ordenP(t).Autorizacion
	casos := []struct {
		nombre    string
		bruto     []byte
		validador func([]byte) (string, error)
	}{
		{"decodificacion", []byte(`{"ok":false}`), func([]byte) (string, error) { return "", errors.New("incompatible") }},
		{"excesivo", []byte(`{"ok":true}`), func(b []byte) (string, error) { return string(b), nil }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			tx := &txP{fila: filaP{vals: []any{caso.bruto}}}
			pool := &poolP{tx: tx}
			limite := 100
			if caso.nombre == "excesivo" {
				limite = 2
			}
			_, err := ejecutarRegistroEmpleadoB2(context.Background(), pool, consultaFichaEmpleadoB2SQL, []byte(`{}`), autorizacion, limite, caso.validador)
			if !errors.Is(err, errRegistroEmpleadoB2NoDisponible) || tx.commits != 0 || tx.rollbacks != 1 {
				t.Fatal("respuesta inválida confirmada", err)
			}
		})
	}
}

func TestRegistroEmpleadoB2DenegacionSQLNoFiltraDetalle(t *testing.T) {
	tx := &txP{errQ: &pgconn.PgError{Code: "42501", Message: "detalle privado"}}
	pool := &poolP{tx: tx}
	_, err := ejecutarRegistroEmpleadoB2(context.Background(), pool, consultaFichaEmpleadoB2SQL, []byte(`{}`), ordenP(t).Autorizacion, 100, func([]byte) (string, error) { return "", nil })
	if !errors.Is(err, errRegistroEmpleadoB2Denegado) || strings.Contains(err.Error(), "privado") || tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatal("denegación incorrecta", err)
	}
}
