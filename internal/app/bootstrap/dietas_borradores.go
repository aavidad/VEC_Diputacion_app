package bootstrap

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	dietascomp "vec-diputacion-granada/internal/modules/dietas/adapters/composicion"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietaspostgres "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	personalcomp "vec-diputacion-granada/internal/modules/personal/adapters/composicion"
	personalpostgres "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionBorradoresDietasNoDisponible = errors.New("bootstrap: borradores de Dietas no disponibles")

// dependenciasBorradoresDietas recibe pools de roles distintos y el emisor V3
// de la misma política común. La raíz conserva la propiedad y el cierre de los
// pools; este montaje no acepta DSN, selectores de actor ni permisos del HTTP.
type dependenciasBorradoresDietas struct {
	personal       *pgxpool.Pool
	dietas         *pgxpool.Pool
	seguridad      resolutorContextoPersonalDietas
	reloj          vecports.Reloj
	emisorPersonal personalcomp.EmisorMaterialRelacionDietasV3
	emisorDietas   dietascomp.EmisorMaterialBorradorV3
	motivoPersonal vecdomain.ReferenciaEntradaCatalogo
	motivoDietas   vecdomain.ReferenciaEntradaCatalogo
}

func componerBorradoresDietas(d dependenciasBorradoresDietas) ([]vechttp.RutaExacta, []vechttp.RutaColeccion, error) {
	if d.personal == nil || d.dietas == nil || dependenciaDietasNula(d.seguridad) || dependenciaDietasNula(d.reloj) || dependenciaDietasNula(d.emisorPersonal) || dependenciaDietasNula(d.emisorDietas) {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	identidad, err := nuevaIdentidadPersonalDietas(d.seguridad, d.reloj)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	autorizadorPersonal, err := personalcomp.NuevoProveedorAutorizacionRelacionDietas(identidad, d.emisorPersonal, d.motivoPersonal)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	repositorioPersonal, err := personalpostgres.NuevoRepositorioRelacionEmpleadoPostgreSQL(d.personal)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	personal, err := personalapp.NuevoServicioConsultaRelacionEmpleado(autorizadorPersonal, repositorioPersonal)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	autorizadorDietas, err := dietascomp.NuevoEmisorAutorizacionBorrador(d.emisorDietas, d.motivoDietas)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	identidadDietas, err := dietascomp.NuevoResolutorIdentidadEfectivaBorrador(identidad, personal, autorizadorDietas)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	repositorioDietas, err := dietaspostgres.NuevoRepositorioBorradorComisionPostgreSQL(d.dietas)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	servicio, err := dietasapp.NuevoServicioBorradorComision(repositorioDietas)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	manejador, err := dietashttp.NuevoManejadorBorradores(identidadDietas, servicio)
	if err != nil {
		return nil, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	// Las dos consultas de identidad comparten el mismo resultado revalidado
	// durante la petición; una segunda identidad no puede sustituir la primera.
	ruta := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), claveCacheSeguridadComunDesarrollo{}, &cacheSeguridadComunDesarrollo{})
		manejador.ServeHTTP(w, r.WithContext(ctx))
	})
	return []vechttp.RutaExacta{{Ruta: dietashttp.RutaBorradores, Manejador: ruta}},
		[]vechttp.RutaColeccion{{Prefijo: dietashttp.RutaBorradores, Manejador: ruta}}, nil
}
