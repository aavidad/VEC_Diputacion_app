package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaContextoOriginalV2 = `SELECT registro_contexto_ref,representacion_canonica,huella_sha256,manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva,resuelto_en FROM vec_contexto_actor_v1.leer_contexto_original_v2($1,$2,$3)`
const ajustesContextoOriginalV2 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

type RelojLecturaContextoOriginalV2 interface{ Ahora() time.Time }
type iniciadorLecturaContextoOriginalV2 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type LectorContextoOriginalPostgreSQLV2 struct {
	pool  iniciadorLecturaContextoOriginalV2
	reloj RelojLecturaContextoOriginalV2
}

var _ ports.LectorContextoOriginalV2 = (*LectorContextoOriginalPostgreSQLV2)(nil)

// Pool dedicado al LOGIN histórico simple. Instalar la fachada y configurar
// timestamptz con ScanLocation=time.UTC corresponde a composición, no al lector.
func NuevoLectorContextoOriginalPostgreSQLV2(pool *pgxpool.Pool, reloj RelojLecturaContextoOriginalV2) (*LectorContextoOriginalPostgreSQLV2, error) {
	return nuevoLectorContextoOriginalPostgreSQLV2(pool, reloj)
}
func nuloHistorico(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}
func nuevoLectorContextoOriginalPostgreSQLV2(pool iniciadorLecturaContextoOriginalV2, reloj RelojLecturaContextoOriginalV2) (*LectorContextoOriginalPostgreSQLV2, error) {
	if nuloHistorico(pool) || nuloHistorico(reloj) {
		return nil, ports.ErrLecturaContextoOriginalV2
	}
	return &LectorContextoOriginalPostgreSQLV2{pool, reloj}, nil
}
func instanteHistorico(t time.Time) bool {
	return !t.IsZero() && t.Location() == time.UTC && t.Year() >= 1 && t.Year() <= 9999 && t.Nanosecond()%1000 == 0
}
func errorHistorico(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrLecturaContextoOriginalV2
}
func (l *LectorContextoOriginalPostgreSQLV2) LeerContextoOriginalV2(ctx context.Context, s ports.SolicitudLecturaContextoOriginalV2) (domain.ResultadoContextoActorRegistradoV2, error) {
	var cero domain.ResultadoContextoActorRegistradoV2
	if ctx == nil || l == nil || nuloHistorico(l.pool) || nuloHistorico(l.reloj) || s.Validar() != nil {
		return cero, errorHistorico(ctx)
	}
	var ultimo time.Time
	validar := func() error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		t := l.reloj.Ahora()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !instanteHistorico(t) || t.Before(ultimo) {
			return ports.ErrLecturaContextoOriginalV2
		}
		ultimo = t
		return ctx.Err()
	}
	if e := validar(); e != nil {
		return cero, e
	}
	tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !nuloHistorico(tx) {
		defer func() {
			if !confirmado {
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(c)
			}
		}()
	}
	if e != nil || nuloHistorico(tx) {
		return cero, errorHistorico(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if _, e = tx.Exec(ctx, ajustesContextoOriginalV2); e != nil {
		return cero, errorHistorico(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	rows, e := tx.Query(ctx, consultaContextoOriginalV2, s.RegistroContextoRef, s.HuellaSHA256, s.ManifiestoProcedenciaHuellaSHA256)
	if !nuloHistorico(rows) {
		defer rows.Close()
	}
	if e != nil || nuloHistorico(rows) {
		return cero, errorHistorico(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if !rows.Next() {
		if rows.Err() != nil {
			return cero, errorHistorico(ctx)
		}
		return cero, errorHistorico(ctx)
	}
	var r domain.ResultadoContextoActorRegistradoV2
	var autoridad string
	if e = rows.Scan(&r.RegistroContextoRef, &r.RepresentacionCanonica, &r.HuellaSHA256, &r.ManifiestoProcedenciaCanonico, &r.ManifiestoProcedenciaHuellaSHA256, &autoridad, &r.ResueltoEnAutoritativo); e != nil {
		return cero, errorHistorico(ctx)
	}
	// Scanner produce bytea; comprobar cotas antes de cualquier copia o parser.
	if len(r.RepresentacionCanonica) == 0 || len(r.RepresentacionCanonica) > ports.MaximoBytesContextoOriginalV2 || len(r.ManifiestoProcedenciaCanonico) == 0 || len(r.ManifiestoProcedenciaCanonico) > ports.MaximoBytesContextoOriginalV2 || !instanteHistorico(r.ResueltoEnAutoritativo) {
		return cero, errorHistorico(ctx)
	}
	r.RepresentacionCanonica = append([]byte(nil), r.RepresentacionCanonica...)
	r.ManifiestoProcedenciaCanonico = append([]byte(nil), r.ManifiestoProcedenciaCanonico...)
	r.AutoridadEfectiva = domain.AutoridadProcedenciaContextoActorV1(autoridad)
	if rows.Next() || rows.Err() != nil {
		return cero, errorHistorico(ctx)
	}
	rows.Close()
	if e = validar(); e != nil {
		return cero, e
	}
	r.Contexto, e = domain.RehidratarContextoActorVinculadoV2(r.RepresentacionCanonica)
	if e != nil || s.ValidarResultado(r) != nil || r.ResueltoEnAutoritativo.After(ultimo) {
		return cero, errorHistorico(ctx)
	}
	copia, e := r.Clonar()
	if e != nil {
		return cero, errorHistorico(ctx)
	}
	if e = validar(); e != nil {
		return cero, e
	}
	if e = tx.Commit(ctx); e != nil {
		return cero, errorHistorico(ctx)
	}
	confirmado = true
	if e = validar(); e != nil {
		return cero, e
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	return copia, nil
}
