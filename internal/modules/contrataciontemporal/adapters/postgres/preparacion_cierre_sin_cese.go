package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// LectorPreparacionCierreAdministrativoPostgreSQL es una dependencia interna de
// la frontera que ya autorizó ConsultaDetalleRRHHV3. Nunca se expone directamente
// como handler ni sustituye la concesión de escritura del cierre.
type LectorPreparacionCierreAdministrativoPostgreSQL struct {
	pool iniciadorRegistroIncorporacionV2
}

var _ ports.LectorPreparacionCierreAdministrativo = (*LectorPreparacionCierreAdministrativoPostgreSQL)(nil)

func NuevoLectorPreparacionCierreAdministrativoPostgreSQL(pool *pgxpool.Pool) (*LectorPreparacionCierreAdministrativoPostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ports.ErrConsultaPreparacionCierreAdministrativoNoDisponible
	}
	return &LectorPreparacionCierreAdministrativoPostgreSQL{pool: pool}, nil
}

const consultarPreparacionCierreSQL87 = `SELECT vec_contratacion_temporal.consultar_preparacion_cierre_administrativo_sin_cese_v1($1,$2,$3)::text`

func (l *LectorPreparacionCierreAdministrativoPostgreSQL) ConsultarPreparacionCierreAdministrativo(ctx context.Context, s ports.SolicitudPreparacionCierreAdministrativo) (ports.PreparacionCierreAdministrativo, error) {
	cero := ports.PreparacionCierreAdministrativo{}
	if ctx == nil || s.Validar() != nil {
		return cero, ports.ErrConsultaPreparacionCierreAdministrativoInvalida
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	if l == nil || nuloRegistroTX(l.pool) {
		return cero, ports.ErrConsultaPreparacionCierreAdministrativoNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil || nuloRegistroTX(tx) {
		return cero, errorConsultaPreparacionCierre87(ctx)
	}
	defer func() {
		c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(c)
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorConsultaPreparacionCierre87(ctx)
	}
	var b []byte
	if err = tx.QueryRow(ctx, consultarPreparacionCierreSQL87, s.OrganizacionRef, s.ExpedienteRef, s.SeguimientoRef).Scan(&b); err != nil {
		return cero, errorConsultaPreparacionCierre87(ctx)
	}
	defer clear(b)
	var p ports.PreparacionCierreAdministrativo
	if decodificarCierre87(ctx, b, &p) != nil || p.ValidarPara(s) != nil {
		return cero, errorConsultaPreparacionCierre87(ctx)
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	// READ ONLY: no negocio, preparación, recibo, consumo ni evento por confirmar.
	// Rollback cierra la instantánea y también cubre todos los rechazos anteriores.
	return p, nil
}
func errorConsultaPreparacionCierre87(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrConsultaPreparacionCierreAdministrativoNoDisponible
}
