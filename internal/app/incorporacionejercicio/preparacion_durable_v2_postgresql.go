package incorporacionejercicio

import (
	"github.com/jackc/pgx/v5/pgxpool"
	hist "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/historiaincorporacion"
	pgct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	appct "vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
	pgpersonal "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
)

// PoolsPreparacionDurableV2 conserva las fronteras propietarias existentes.
// CT78, CT81, Personal6 y Personal5 respectivamente; Historia mantiene los cinco
// lectores originales. La composición de runtime verifica LOGIN/ACL/codecs y
// conserva la propiedad y cierre de los pools. Punteros distintos no prueban ACL.
type PoolsPreparacionDurableV2 struct {
	InicialCT           *pgxpool.Pool
	LocalizadorCT       *pgxpool.Pool
	LocalizadorPersonal *pgxpool.Pool
	LecturaPersonal     *pgxpool.Pool
	Historia            pgct.PoolsHistoriaIncorporacionV2
}

type ConfiguracionPreparacionDurableV2PostgreSQL struct {
	Autoridad      *AutoridadAplicacion
	Detalle        *appct.ServicioConsultaDetalleRRHH
	Planes         []byte
	TernaPlanes    ct.ReferenciaVersionadaPersonalRPT
	FuentePersonal []byte
	TernaPersonal  fuenteejercicio.TernaEsperada
	Pools          PoolsPreparacionDurableV2
	Reloj          ct.Reloj
}

// NuevoPreparadorDurableV2PostgreSQL ensambla la única implementación del
// proveedor V2 con sus adaptadores reales. No efectúa IO, no consulta el reloj,
// no obtiene identidad/autoridad ni concede permisos. Autoridad y Detalle ya
// pertenecen a la misma petición autenticada y al contexto RRHH del servidor.
// No admite callbacks de alta, confirmadores, nueva SQL o rutas alternativas.
func NuevoPreparadorDurableV2PostgreSQL(c ConfiguracionPreparacionDurableV2PostgreSQL) (*PreparadorDurableV2, error) {
	f := ct.ErrComposicionIncorporacionAplicacion
	if c.Autoridad == nil || validarConfiguracionPreparacionV2PostgreSQL(c) != nil {
		return nil, f
	}
	planes, err := NuevaFuentePlanesPreparacionV2(c.Planes, c.TernaPlanes)
	if err != nil {
		return nil, f
	}
	inicial, err := pgct.NuevoResolverRaizIncorporacionV2PostgreSQL(c.Pools.InicialCT, c.Reloj)
	if err != nil {
		return nil, f
	}
	localCT, err := pgct.NuevoLocalizadorIncorporacionOriginalV2PostgreSQL(c.Pools.LocalizadorCT, c.Reloj)
	if err != nil {
		return nil, f
	}
	restaurador, err := pgct.NuevoRestauradorHistoriaIncorporacionV2PostgreSQL(c.Pools.Historia, c.Reloj)
	if err != nil {
		return nil, f
	}
	localPersonal, err := pgpersonal.NuevoLocalizadorSolicitudAltaEjercicioPostgreSQL(c.Pools.LocalizadorPersonal, c.Reloj)
	if err != nil {
		return nil, f
	}
	txLectura, err := pgpersonal.NuevaTransaccionLecturaIncorporacionV2PostgreSQL(c.Pools.LecturaPersonal, c.Reloj)
	if err != nil {
		return nil, f
	}
	lectura, err := lector.NuevoV2(c.Autoridad, txLectura, c.Reloj)
	if err != nil {
		return nil, f
	}
	return NuevoPreparadorDurableV2(ConfiguracionPreparacionDurableV2{Autoridad: c.Autoridad, Detalle: c.Detalle,
		Planes: planes, FuentePersonal: c.FuentePersonal, TernaPersonal: c.TernaPersonal, Inicial: inicial,
		LocalizadorCT: localCT, Restaurador: restaurador, LocalizadorPersonal: localPersonal, LectorPersonal: lectura, Reloj: c.Reloj})
}

func validarConfiguracionPreparacionV2PostgreSQL(c ConfiguracionPreparacionDurableV2PostgreSQL) error {
	f := ct.ErrComposicionIncorporacionAplicacion
	if c.Detalle == nil || nulo(c.Reloj) {
		return f
	}
	vistos := make(map[*pgxpool.Pool]bool, 9)
	for _, p := range []*pgxpool.Pool{c.Pools.InicialCT, c.Pools.LocalizadorCT, c.Pools.LocalizadorPersonal, c.Pools.LecturaPersonal,
		c.Pools.Historia.RegistroCT, c.Pools.Historia.Autenticacion, c.Pools.Historia.Contexto, c.Pools.Historia.Evaluacion, c.Pools.Historia.Concesion} {
		if p == nil || vistos[p] {
			return f
		}
		vistos[p] = true
	}
	return nil
}

var (
	_ LectorInicialPreparacionV2       = (*pgct.ResolverRaizIncorporacionV2PostgreSQL)(nil)
	_ LocalizadorOriginalPreparacionV2 = (*pgct.LocalizadorIncorporacionOriginalV2PostgreSQL)(nil)
	_ LocalizadorPersonalPreparacionV2 = (*pgpersonal.LocalizadorSolicitudAltaEjercicioPostgreSQL)(nil)
	_ RestauradorOriginalPreparacionV2 = (*hist.Restaurador)(nil)
	_ LectorPersonalPreparacionV2      = (*lector.ConsumidorV2)(nil)
	_ ConsultaDetallePreparacionV2     = (*appct.ServicioConsultaDetalleRRHH)(nil)
)
