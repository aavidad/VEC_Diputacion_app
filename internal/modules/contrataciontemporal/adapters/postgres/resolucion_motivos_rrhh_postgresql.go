package postgres

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const consultaResolucionMotivoCuadroRRHHPostgreSQL = `
WITH resultado AS MATERIALIZED (
	SELECT catalogo_id, catalogo_version, catalogo_huella_sha256, entrada_clave
	FROM vec_autorizacion.resolver_motivo_cuadro_rrhh_v1($1::timestamptz)
	LIMIT 2
)
SELECT pg_catalog.count(*)::bigint,
	COALESCE(pg_catalog.max(catalogo_id), '')::text,
	COALESCE(pg_catalog.max(catalogo_version), 0)::bigint,
	COALESCE(pg_catalog.max(catalogo_huella_sha256), '')::text,
	COALESCE(pg_catalog.max(entrada_clave), '')::text
FROM resultado`

const consultaResolucionMotivoDetalleRRHHPostgreSQL = `
WITH resultado AS MATERIALIZED (
	SELECT catalogo_id, catalogo_version, catalogo_huella_sha256, entrada_clave
	FROM vec_autorizacion.resolver_motivo_detalle_rrhh_v1($1::timestamptz)
	LIMIT 2
)
SELECT pg_catalog.count(*)::bigint,
	COALESCE(pg_catalog.max(catalogo_id), '')::text,
	COALESCE(pg_catalog.max(catalogo_version), 0)::bigint,
	COALESCE(pg_catalog.max(catalogo_huella_sha256), '')::text,
	COALESCE(pg_catalog.max(entrada_clave), '')::text
FROM resultado`

type origenResolutorMotivosRRHHPostgreSQL interface {
	adquirirOperacion(context.Context) (conexionPoolResolucionMotivosRRHH, error)
	reacreditar(context.Context, transaccionAcreditacionResolucionMotivosRRHH) error
}

// ResolutorMotivosRRHHPostgreSQL resuelve exclusivamente los dos motivos
// nominales de consulta RRHH. No admite selectores libres ni expone el pool.
type ResolutorMotivosRRHHPostgreSQL struct {
	origen origenResolutorMotivosRRHHPostgreSQL
}

var _ ports.ResolutorMotivoConsultaRRHH = (*ResolutorMotivosRRHHPostgreSQL)(nil)

// NuevoResolutorMotivoConsultaRRHHPostgreSQL solo admite el pool nominal M2.1
// previamente acreditado. Los dobles quedan confinados al constructor privado.
func NuevoResolutorMotivoConsultaRRHHPostgreSQL(
	pool *PoolResolucionMotivosRRHHPostgreSQL,
) (*ResolutorMotivosRRHHPostgreSQL, error) {
	if pool == nil || !selloPoolResolucionMotivosRRHHValido(
		pool.sello, pool, true,
	) {
		return nil, ports.ErrMotivoConsultaRRHHNoDisponible
	}
	return nuevoResolutorMotivosRRHHPostgreSQL(pool)
}

func nuevoResolutorMotivosRRHHPostgreSQL(
	origen origenResolutorMotivosRRHHPostgreSQL,
) (*ResolutorMotivosRRHHPostgreSQL, error) {
	if dependenciaNula(origen) {
		return nil, ports.ErrMotivoConsultaRRHHNoDisponible
	}
	return &ResolutorMotivosRRHHPostgreSQL{origen: origen}, nil
}

func (r *ResolutorMotivosRRHHPostgreSQL) ResolverMotivoCuadroRRHH(
	ctx context.Context,
	instante time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	return r.resolver(ctx, instante, consultaResolucionMotivoCuadroRRHHPostgreSQL)
}

func (r *ResolutorMotivosRRHHPostgreSQL) ResolverMotivoDetalleRRHH(
	ctx context.Context,
	instante time.Time,
) (dominiovec.ReferenciaEntradaCatalogo, error) {
	return r.resolver(ctx, instante, consultaResolucionMotivoDetalleRRHHPostgreSQL)
}

func (r *ResolutorMotivosRRHHPostgreSQL) resolver(
	ctx context.Context,
	instante time.Time,
	consulta string,
) (
	resultado dominiovec.ReferenciaEntradaCatalogo,
	errResultado error,
) {
	var conexion conexionPoolResolucionMotivosRRHH
	var transaccion transaccionPoolResolucionMotivosRRHH
	confirmada := false
	defer func() {
		panico := recover()
		if !confirmada && !dependenciaNula(transaccion) {
			revertirTransaccionResolucionMotivosRRHH(transaccion)
		}
		falloLiberacion := liberarConexionResolucionMotivosRRHH(conexion)
		if panico != nil || falloLiberacion {
			resultado = dominiovec.ReferenciaEntradaCatalogo{}
			errResultado = errorPoolResolucionMotivosRRHH(ctx)
		}
	}()
	if dependenciaNula(ctx) || r == nil || dependenciaNula(r.origen) ||
		!instanteResolucionMotivosRRHHValido(instante) {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	if err := ctx.Err(); err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	var err error
	conexion, err = r.origen.adquirirOperacion(ctx)
	if err != nil || dependenciaNula(conexion) {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	transaccion, err = conexion.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil || dependenciaNula(transaccion) ||
		conexion.Sello() == nil || transaccion.Sello() != conexion.Sello() {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	if err = r.origen.reacreditar(ctx, transaccion); err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	var cardinalidad, version int64
	err = transaccion.QueryRow(ctx, consulta, instante).Scan(
		&cardinalidad, &resultado.CatalogoID, &version,
		&resultado.CatalogoHuellaSHA256, &resultado.EntradaClave,
	)
	if err != nil || cardinalidad != 1 || version < 1 ||
		version > math.MaxInt32 {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	resultado.CatalogoVersion = int(version)
	if !dominiovec.ReferenciaMotivoAutorizacionV2Valida(resultado) {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	if err = transaccion.Commit(ctx); err != nil {
		return dominiovec.ReferenciaEntradaCatalogo{},
			errorPoolResolucionMotivosRRHH(ctx)
	}
	confirmada = true
	return resultado, nil
}

func instanteResolucionMotivosRRHHValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func revertirTransaccionResolucionMotivosRRHH(
	transaccion transaccionPoolResolucionMotivosRRHH,
) {
	defer func() { _ = recover() }()
	if !dependenciaNula(transaccion) {
		ctxLimpieza, cancelar := context.WithTimeout(
			context.Background(), 2*time.Second,
		)
		defer cancelar()
		_ = transaccion.Rollback(ctxLimpieza)
	}
}
