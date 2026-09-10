package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"reflect"
	"time"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const consultaHistoriaIncorporacionV2 = `SELECT vec_contratacion_temporal.leer_historia_incorporacion_original_v2($1,$2,$3)::text`
const ajustesHistoriaIncorporacionV2 = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','5s',true)`

type iniciadorHistoriaIncorporacionV2 interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}
type LectorHistoriaIncorporacionV2PostgreSQL struct {
	pool  iniciadorHistoriaIncorporacionV2
	reloj ct.Reloj
}

var _ hist.LectorRegistro = (*LectorHistoriaIncorporacionV2PostgreSQL)(nil)

// Pool propietario dedicado al LOGIN simple del grupo histórico CT77. No SELECT
// cruzado, resolución de raíz, registro, retry ni autorización actual.
func NuevoLectorHistoriaIncorporacionV2PostgreSQL(p *pgxpool.Pool, r ct.Reloj) (*LectorHistoriaIncorporacionV2PostgreSQL, error) {
	return nuevoLectorHistoriaIncorporacionV2(p, r)
}
func nuloLectorHistoriaV2(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Slice, reflect.Map, reflect.Chan:
		return x.IsNil()
	}
	return false
}
func nuevoLectorHistoriaIncorporacionV2(p iniciadorHistoriaIncorporacionV2, r ct.Reloj) (*LectorHistoriaIncorporacionV2PostgreSQL, error) {
	if nuloLectorHistoriaV2(p) || nuloLectorHistoriaV2(r) {
		return nil, hist.ErrHistoria
	}
	return &LectorHistoriaIncorporacionV2PostgreSQL{p, r}, nil
}
func errorLectorHistoriaV2(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return hist.ErrHistoria
}
func (l *LectorHistoriaIncorporacionV2PostgreSQL) LeerRegistroOriginal(ctx context.Context, s hist.Selector) ([]byte, error) {
	if ctx == nil || l == nil || nuloLectorHistoriaV2(l.pool) || nuloLectorHistoriaV2(l.reloj) {
		return nil, errorLectorHistoriaV2(ctx)
	}
	if !dom.ReferenciaOpacaValida(s.ReciboRef) {
		return nil, errorLectorHistoriaV2(ctx)
	}
	for _, h := range []string{s.MaterialSHA256, s.IntencionSHA256} {
		b, e := hex.DecodeString(h)
		if e != nil || len(b) != 32 || hex.EncodeToString(b) != h {
			return nil, errorLectorHistoriaV2(ctx)
		}
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
		if !dom.InstanteUTCCanonico(t) || t.Before(ultimo) {
			return hist.ErrHistoria
		}
		ultimo = t
		return ctx.Err()
	}
	if e := validar(); e != nil {
		return nil, e
	}
	tx, e := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !nuloLectorHistoriaV2(tx) {
		defer func() {
			if !confirmado {
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(c)
			}
		}()
	}
	if e != nil || nuloLectorHistoriaV2(tx) {
		return nil, errorLectorHistoriaV2(ctx)
	}
	if e = validar(); e != nil {
		return nil, e
	}
	if _, e = tx.Exec(ctx, ajustesHistoriaIncorporacionV2); e != nil {
		return nil, errorLectorHistoriaV2(ctx)
	}
	if e = validar(); e != nil {
		return nil, e
	}
	rows, e := tx.Query(ctx, consultaHistoriaIncorporacionV2, s.ReciboRef, s.MaterialSHA256, s.IntencionSHA256)
	if !nuloLectorHistoriaV2(rows) {
		defer rows.Close()
	}
	if e != nil || nuloLectorHistoriaV2(rows) {
		return nil, errorLectorHistoriaV2(ctx)
	}
	if e = validar(); e != nil {
		return nil, e
	}
	if !rows.Next() {
		if rows.Err() != nil {
			return nil, errorLectorHistoriaV2(ctx)
		}
		return nil, errorLectorHistoriaV2(ctx)
	}
	var raw []byte
	if e = rows.Scan(&raw); e != nil {
		return nil, errorLectorHistoriaV2(ctx)
	}
	if len(raw) == 0 || len(raw) > hist.MaximoBytesDocumento {
		return nil, errorLectorHistoriaV2(ctx)
	}
	// Copiar antes de Next/Close: no retener buffers propiedad del driver.
	salida := bytes.Clone(raw)
	if rows.Next() || rows.Err() != nil {
		return nil, errorLectorHistoriaV2(ctx)
	}
	rows.Close()
	if e = validar(); e != nil {
		return nil, e
	}
	if e = hist.ValidarDocumentoRegistroOriginal(ctx, salida, s, ultimo); e != nil {
		return nil, errorLectorHistoriaV2(ctx)
	}
	if e = validar(); e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, errorLectorHistoriaV2(ctx)
	}
	confirmado = true
	if e = validar(); e != nil {
		return nil, e
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return salida, nil
}
