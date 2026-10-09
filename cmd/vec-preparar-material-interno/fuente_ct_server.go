package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/bootstrap"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

const errMotivosRRHHDSN = errorPropio("motivos RRHH: DSN privado 0600 ausente o identidad nominal no disponible")

// leerDatosCTServidor toma la misma generación HMAC que el publicador de
// vec-server. El resultado no contiene ni deriva las capacidades del vec-interno
// legado: B2 puro sólo necesita raíz, emisor y resolutor de motivo de detalle.
func leerDatosCTServidor(directorioIdempotencia string, ahora time.Time) (datosCT, string, error) {
	c, err := bootstrap.DerivarCoordenadasCTPreparacion(directorioIdempotencia, ahora)
	if err != nil || c.RaizID == "" || c.HuellaSPKI == "" || c.EmisorID == "" {
		return datosCT{}, "", errIdempotencia
	}
	return datosCT{emisor: c.EmisorID, raizID: c.RaizID, audiencia: c.Audiencia}, c.HuellaSPKI, nil
}

// resolverMotivoDetalleCTServidor conserva la fábrica nominal existente de
// RRHH y su transacción SERIALIZABLE. Ningún motivo se toma de la DSN.
func resolverMotivoDetalleCTServidor(ctx context.Context, archivoDSN string, ahora time.Time) (core.ReferenciaEntradaCatalogo, error) {
	vacio := core.ReferenciaEntradaCatalogo{}
	dsn, err := leerDSN(archivoDSN, "")
	if err != nil {
		return vacio, errMotivosRRHHDSN
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c == nil || c.ConnConfig == nil || c.ConnConfig.User == "" {
		return vacio, errMotivosRRHHDSN
	}
	p, err := pgct.NuevoPoolResolucionMotivosRRHHPostgreSQL(ctx, dsn, c.ConnConfig.User)
	if err != nil {
		return vacio, errMotivosRRHHDSN
	}
	defer p.Cerrar()
	r, err := pgct.NuevoResolutorMotivoConsultaRRHHPostgreSQL(p)
	if err != nil {
		return vacio, errMotivosRRHHDSN
	}
	return r.ResolverMotivoDetalleRRHH(ctx, ahora.UTC().Truncate(time.Microsecond))
}
