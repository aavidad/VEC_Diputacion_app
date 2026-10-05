package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type txPublicacionRechazoPrueba struct {
	txPrueba
	queryError error
	queryRaw   []byte
	consultas  int
}

func (t *txPublicacionRechazoPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	t.consultas++
	return filaPublicacionRechazoPrueba{t.queryError, t.queryRaw}
}

type filaPublicacionRechazoPrueba struct {
	err error
	raw []byte
}

func (f filaPublicacionRechazoPrueba) Scan(dest ...any) error {
	if f.err != nil {
		return f.err
	}
	*(dest[0].(*[]byte)) = append([]byte(nil), f.raw...)
	return nil
}

type poolPublicacionRechazoPrueba struct{ tx *txPublicacionRechazoPrueba }

func (p poolPublicacionRechazoPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return p.tx, nil
}

// Dobles del transporte: no emiten capacidades ni acreditan permisos SQL.
// La fase donde aparece 42501 debe determinar el resultado de auditoría.
func TestPublicacionDistingueRechazoSQLDeCommitNoConfirmado(t *testing.T) {
	for _, caso := range []string{"cuerpo_42501", "commit_42501"} {
		t.Run(caso, func(t *testing.T) {
			rechazado := &pgconn.PgError{Code: "42501", Message: "mensaje_sql_privado_no_exponer"}
			tx := &txPublicacionRechazoPrueba{queryRaw: []byte(`{"persona_ref":"per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","procedencia_ref":"prc_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","sobre_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","auditoria_ref":"aud_v3_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version":1}`)}
			esperado := ErrNoDisponible
			commits := 1
			if caso == "cuerpo_42501" {
				tx.queryError = fmt.Errorf("consulta: %w", rechazado)
				esperado = domain.ErrAutorizacionDenegada
				commits = 0
			} else {
				tx.commitError = rechazado
			}
			a := &Adaptador{pool: poolPublicacionRechazoPrueba{tx}}
			var provisional reciboJSON
			fallo := a.transaccion(context.Background(), func(actual pgx.Tx) error {
				var raw []byte
				if err := actual.QueryRow(context.Background(), "SELECT publicacion_sintetica").Scan(&raw); err != nil {
					return err
				}
				return decodificar(raw, &provisional)
			})
			recibo, err := resultadoPublicacion(provisional, fallo)
			if !errors.Is(err, esperado) || err != esperado || recibo != (ports.ReciboDenominacionPersona{}) || tx.consultas != 1 || tx.commits != commits || tx.rollbacks != 1 {
				t.Fatal("clasificacion_o_recibo_no_cerrados")
			}
			if caso == "commit_42501" && (provisional.PersonaRef == "" || errors.Is(err, domain.ErrAutorizacionDenegada)) {
				t.Fatal("commit_incierto_clasificado_como_denegacion")
			}
		})
	}
}
