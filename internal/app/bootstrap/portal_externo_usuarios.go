package bootstrap

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecmemory "vec-diputacion-granada/internal/vec/adapters/memory"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// ErrUsuariosPortalExternoNoDisponible: el proceso externo tiene encendidas
// las preferencias pero le falta su material, su conexión de preflight o su
// configuración. No arranca a medias.
var ErrUsuariosPortalExternoNoDisponible = errors.New("bootstrap: preferencias del portal externo no disponibles")

// nuevasPreferenciasPortalExterno compone «Mis preferencias» del Área
// personal en el proceso externo. Reutiliza el montaje por superficie de la
// composición combinada, pero con lo que solo tiene el externo: sus LOGIN
// (identidad/usuarios-preferencias-externa.json), su idempotencia y las
// claves derivadas de sus dos audiencias, cotejadas con el gobierno mediante
// el rol de preflight externo.
func nuevasPreferenciasPortalExterno(ctx context.Context, cfg config.Config, identidad *resolvedorIdentidadDesarrollo,
	derivador *derivadorIdentidadOperacionDesarrollo, incidencias vecports.EmisorIncidenciasTecnicas, preflight *pgxpool.Pool,
) (*autoridadPreferenciasUsuariosDesarrollo, error) {
	if ctx == nil || identidad == nil || derivador == nil || !derivador.valido() || incidencias == nil || preflight == nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight, "usuarios_preferencias", relojContratacionTemporalDesarrollo{})
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	consulta := proveedores[audienciaConsultaPreferenciasUsuariosExterna]
	actualizacion := proveedores[audienciaActualizacionPreferenciasUsuariosExterna]
	if consulta == nil || actualizacion == nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	// Todas las conexiones del externo deben ir a la misma instancia que su
	// preflight; es la misma comprobación que la combinada hace con gobierno.
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, preflight)
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	autoridad, err := nuevaRutaUsuariosPreferenciasSuperficieDesarrollo(cfg, identidad, derivador, incidencias, topologia,
		core.SuperficieAutenticacionExternaPersonalV1, usuarioshttp.RutaMisPreferenciasAreaPersonal, consulta, actualizacion, nil, nil, nil)
	if err != nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	return autoridad, nil
}

// autoridadExactasPortalExterno autoriza solo las rutas exactas que compone
// el proceso externo, con el contexto que fijó su propia frontera.
type autoridadExactasPortalExterno struct {
	preferencias *autoridadPreferenciasUsuariosDesarrollo
}

func (a autoridadExactasPortalExterno) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if ctx == nil || a.preferencias == nil || ruta != usuarioshttp.RutaMisPreferenciasAreaPersonal {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || c.autoridad != a.preferencias || c.resultado.Validar() != nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(a.preferencias.reloj.Ahora(), c.resultado) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

// nuevaAPIPersonalPortalExterno monta las rutas exactas del Área personal en
// un despachador VEC sin módulos internos. La identidad sale del certificado
// ya filtrado en el saludo TLS; el filtro por portal sigue delante.
func nuevaAPIPersonalPortalExterno(identidad *resolvedorIdentidadDesarrollo, incidencias vecports.EmisorIncidenciasTecnicas,
	preferencias *autoridadPreferenciasUsuariosDesarrollo,
) (http.Handler, error) {
	if identidad == nil || incidencias == nil || preferencias == nil || preferencias.manejador == nil {
		return nil, ErrUsuariosPortalExternoNoDisponible
	}
	almacen := vecmemory.NewStore()
	servicio, _, err := vecapp.NewServiceWithInternalOperations(almacen, almacen, almacen)
	if err != nil {
		return nil, err
	}
	manejador, err := vechttp.NewHandlerWithOptions(servicio, vechttp.HandlerOptions{
		AllowDemoIdentity:                        true,
		DemoIdentityResolver:                     identidad,
		RutasExactas:                             []vechttp.RutaExacta{{Ruta: usuarioshttp.RutaMisPreferenciasAreaPersonal, Manejador: preferencias.manejador}},
		AutoridadRutasExactas:                    autoridadExactasPortalExterno{preferencias: preferencias},
		RegistradorAuditoriaFronteraRutasExactas: registradorFronterasConUsuariosPreferencias{externa: preferencias.registrador},
		EmisorIncidenciasTecnicas:                incidencias,
	})
	if err != nil {
		return nil, err
	}
	return preferencias.proteger(manejador), nil
}
