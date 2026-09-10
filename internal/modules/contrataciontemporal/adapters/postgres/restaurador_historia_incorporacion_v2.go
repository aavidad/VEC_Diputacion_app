package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	autorizacionpg "vec-diputacion-granada/internal/vec/adapters/postgres"
)

// PoolsHistoriaIncorporacionV2 son conexiones propietarias independientes:
// CT77, Identidad histórica, Contexto5, Auth12 y Auth11 respectivamente.
// El llamador configura y verifica LOGIN/ACL históricos y ScanLocation=time.UTC
// para timestamptz antes de construirlos. No usar pools CT runtime ni fuentes
// vivas. Auth11 usa la composición propietaria de registro existente.
// Cinco punteros distintos evitan reutilización accidental, pero NO acreditan
// que sus identidades/configuraciones sean distintas ni que tengan autoridad.
// La propiedad/cierre de los pools permanece siempre en el llamador.
type PoolsHistoriaIncorporacionV2 struct {
	RegistroCT    *pgxpool.Pool
	Autenticacion *pgxpool.Pool
	Contexto      *pgxpool.Pool
	Evaluacion    *pgxpool.Pool
	Concesion     *pgxpool.Pool
}

var _ RestauradorOriginalIncorporacionV2 = (*hist.Restaurador)(nil)

// NuevoRestauradorHistoriaIncorporacionV2PostgreSQL compone lectores existentes
// y sus fronteras anticorrupción. No efectúa IO, no cambia pools/roles/codecs,
// no registra concesiones ni fabrica autoridad u órdenes desde DTO. El retorno
// sirve directamente como dependencia histórica de CT23; el permiso actual y
// la transacción de escritura siguen siendo independientes de estas lecturas.
func NuevoRestauradorHistoriaIncorporacionV2PostgreSQL(p PoolsHistoriaIncorporacionV2, reloj ct.Reloj) (*hist.Restaurador, error) {
	if nuloRegistroTX(reloj) {
		return nil, hist.ErrHistoria
	}
	vistos := make(map[*pgxpool.Pool]bool, 5)
	for _, pool := range []*pgxpool.Pool{p.RegistroCT, p.Autenticacion, p.Contexto, p.Evaluacion, p.Concesion} {
		if pool == nil || vistos[pool] {
			return nil, hist.ErrHistoria
		}
		vistos[pool] = true
	}
	registros, e := NuevoLectorHistoriaIncorporacionV2PostgreSQL(p.RegistroCT, reloj)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	autenticaciones, e := identidadpg.NuevoLectorAutenticacionOriginalPostgreSQLV1(p.Autenticacion, reloj)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	a, e := hist.NuevoAutenticacionPropietaria(autenticaciones)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	contextos, e := contextopg.NuevoLectorContextoOriginalPostgreSQLV2(p.Contexto, reloj)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	c, e := hist.NuevoContextoPropietario(contextos)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	evaluaciones, e := autorizacionpg.NuevoLectorEvaluacionOriginalPostgreSQLV3(p.Evaluacion, reloj)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	evaluador, e := hist.NuevoEvaluacionPropietaria(evaluaciones)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	concesiones, e := autorizacionpg.NuevoAlmacenAutorizacion(p.Concesion)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	r, e := hist.NuevoRestaurador(registros, a, c, evaluador, concesiones, reloj)
	if e != nil {
		return nil, hist.ErrHistoria
	}
	return r, nil
}
