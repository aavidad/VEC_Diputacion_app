package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const consultaLocalizadorIncorporacionAplicacionV2 = `SELECT recibo_ref, material_sha256, intencion_sha256 FROM vec_contratacion_temporal.localizar_incorporacion_original_v2($1::text,$2::text,$3::text)`

var ErrLocalizadorIncorporacionNoDisponible = errors.New("contratacion temporal: localizacion original no disponible")

// LocalizadorIncorporacionOriginalV2PostgreSQL sólo descubre el selector CT77.
// La composición debe autorizar la lectura actual antes de llamarlo y restaurar
// las cinco fuentes originales después. Un selector no es recibo ni permiso.
// El pool corresponde al LOGIN dedicado del localizador, sin SELECT de tablas.
type LocalizadorIncorporacionOriginalV2PostgreSQL struct {
	pool  iniciadorHistoriaIncorporacionV2
	reloj ct.Reloj
}

func NuevoLocalizadorIncorporacionOriginalV2PostgreSQL(p *pgxpool.Pool, r ct.Reloj) (*LocalizadorIncorporacionOriginalV2PostgreSQL, error) {
	return nuevoLocalizadorIncorporacionOriginalV2(p, r)
}

func nuevoLocalizadorIncorporacionOriginalV2(p iniciadorHistoriaIncorporacionV2, r ct.Reloj) (*LocalizadorIncorporacionOriginalV2PostgreSQL, error) {
	if nuloLectorHistoriaV2(p) || nuloLectorHistoriaV2(r) {
		return nil, ErrLocalizadorIncorporacionNoDisponible
	}
	return &LocalizadorIncorporacionOriginalV2PostgreSQL{pool: p, reloj: r}, nil
}

func errorLocalizadorIncorporacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var estado interface{ SQLState() string }
	if errors.As(err, &estado) && estado.SQLState() == "P1102" {
		return ct.ErrConflictoIncorporacionAplicacion
	}
	return ErrLocalizadorIncorporacionNoDisponible
}

// Localizar devuelve encontrado=false sólo después de una consulta vacía y un
// commit de lectura confirmado. Fallo, duplicidad o cancelación devuelven cero.
// No hay retry, búsqueda latest, renovación de historia ni efectos de negocio.
func (l *LocalizadorIncorporacionOriginalV2PostgreSQL) Localizar(ctx context.Context, organizacion, expediente, solicitud string) (hist.Selector, bool, error) {
	var cero hist.Selector
	if ctx == nil || l == nil || nuloLectorHistoriaV2(l.pool) || nuloLectorHistoriaV2(l.reloj) {
		return cero, false, errorLocalizadorIncorporacion(ctx, nil)
	}
	for _, ref := range []string{organizacion, expediente, solicitud} {
		if !dom.ReferenciaOpacaValida(ref) {
			return cero, false, errorLocalizadorIncorporacion(ctx, nil)
		}
	}
	var ultimo time.Time
	validar := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ahora := l.reloj.Ahora()
		if err := ctx.Err(); err != nil {
			return err
		}
		if !dom.InstanteUTCCanonico(ahora) || ahora.Before(ultimo) {
			return ErrLocalizadorIncorporacionNoDisponible
		}
		ultimo = ahora
		return ctx.Err()
	}
	if err := validar(); err != nil {
		return cero, false, err
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !nuloLectorHistoriaV2(tx) {
		defer func() {
			if !confirmado {
				limite, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = tx.Rollback(limite)
			}
		}()
	}
	if err != nil || nuloLectorHistoriaV2(tx) {
		return cero, false, errorLocalizadorIncorporacion(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, false, err
	}
	if _, err = tx.Exec(ctx, ajustesHistoriaIncorporacionV2); err != nil {
		return cero, false, errorLocalizadorIncorporacion(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, false, err
	}
	rows, err := tx.Query(ctx, consultaLocalizadorIncorporacionAplicacionV2, organizacion, expediente, solicitud)
	if !nuloLectorHistoriaV2(rows) {
		defer rows.Close()
	}
	if err != nil || nuloLectorHistoriaV2(rows) {
		return cero, false, errorLocalizadorIncorporacion(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, false, err
	}
	var resultado hist.Selector
	encontrado := rows.Next()
	if encontrado {
		if err = rows.Scan(&resultado.ReciboRef, &resultado.MaterialSHA256, &resultado.IntencionSHA256); err != nil {
			return cero, false, errorLocalizadorIncorporacion(ctx, err)
		}
		if err = validar(); err != nil {
			return cero, false, err
		}
		if !dom.ReferenciaOpacaValida(resultado.ReciboRef) {
			return cero, false, ErrLocalizadorIncorporacionNoDisponible
		}
		for _, h := range []string{resultado.MaterialSHA256, resultado.IntencionSHA256} {
			b, e := hex.DecodeString(h)
			if e != nil || len(b) != 32 || hex.EncodeToString(b) != h {
				return cero, false, ErrLocalizadorIncorporacionNoDisponible
			}
		}
		if rows.Next() {
			return cero, false, ct.ErrConflictoIncorporacionAplicacion
		}
	}
	if err = rows.Err(); err != nil {
		return cero, false, errorLocalizadorIncorporacion(ctx, err)
	}
	rows.Close()
	if err = validar(); err != nil {
		return cero, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, false, errorLocalizadorIncorporacion(ctx, err)
	}
	confirmado = true
	if err = validar(); err != nil {
		return cero, false, err
	}
	return resultado, encontrado, nil
}
