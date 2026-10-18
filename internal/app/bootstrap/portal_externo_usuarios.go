package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
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
	// Cartera es una capacidad independiente. Una configuración ausente o una
	// precondición incompleta deja sus rutas en 503 sin desmontar Preferencias.
	cerrarCartera, falloCartera := montarCarteraPortalExterno(ctx, cfg, autoridad, derivador, preflight)
	if falloCartera != nil {
		slog.Warn("cartera_meritos_portal_externo_no_disponible", "causa", falloCartera.Error())
	} else {
		cerrarPreferencias := autoridad.cerrar
		autoridad.cerrar = func() { cerrarCartera(); cerrarPreferencias() }
	}
	return autoridad, nil
}

// autoridadExactasPortalExterno autoriza solo las rutas exactas que compone
// el proceso externo, con el contexto que fijó su propia frontera.
type autoridadExactasPortalExterno struct {
	preferencias *autoridadPreferenciasUsuariosDesarrollo
}

func (a autoridadExactasPortalExterno) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if ctx == nil || a.preferencias == nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	seleccionada := a.preferencias
	switch ruta {
	case usuarioshttp.RutaMisPreferenciasAreaPersonal:
	case meritoshttp.RutaCartera:
		seleccionada = a.preferencias.cartera
	case meritoshttp.RutaRetiradasCartera:
		seleccionada = a.preferencias.retirada
	case meritoshttp.RutaDeclaracionesCartera:
		seleccionada = a.preferencias.declaracion
	default:
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	c, ok := ctx.Value(claveContextoPreferenciasUsuarios{}).(contextoPreferenciasUsuarios)
	if !ok || seleccionada == nil || c.autoridad != seleccionada || c.resultado.Validar() != nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if c.vinculo.ValidarPara(c.resultado) != nil || !c.vinculo.VigenteEn(seleccionada.reloj.Ahora(), c.resultado) {
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
	rutas := []vechttp.RutaExacta{{Ruta: usuarioshttp.RutaMisPreferenciasAreaPersonal, Manejador: preferencias.manejador}}
	frontera := registradorFronterasConUsuariosPreferencias{externa: preferencias.registrador}
	carteraLista := preferencias.cartera != nil && preferencias.retirada != nil && preferencias.declaracion != nil &&
		preferencias.cartera.manejador != nil && preferencias.retirada.manejador != nil &&
		preferencias.cartera.registrador != nil && preferencias.retirada.registrador != nil
	if carteraLista {
		rutas = append(rutas,
			vechttp.RutaExacta{Ruta: meritoshttp.RutaCartera, Manejador: preferencias.cartera.manejador},
			vechttp.RutaExacta{Ruta: meritoshttp.RutaRetiradasCartera, Manejador: preferencias.retirada.manejador},
			vechttp.RutaExacta{Ruta: meritoshttp.RutaDeclaracionesCartera, Manejador: preferencias.declaracion.manejador})
		frontera.meritos = preferencias.cartera.registrador
	}
	manejador, err := vechttp.NewHandlerWithOptions(servicio, vechttp.HandlerOptions{
		AllowDemoIdentity:                        true,
		DemoIdentityResolver:                     identidad,
		RutasExactas:                             rutas,
		AutoridadRutasExactas:                    autoridadExactasPortalExterno{preferencias: preferencias},
		RegistradorAuditoriaFronteraRutasExactas: frontera,
		EmisorIncidenciasTecnicas:                incidencias,
	})
	if err != nil {
		return nil, err
	}
	protegido := preferencias.proteger(manejador)
	if carteraLista {
		return preferencias.cartera.proteger(preferencias.retirada.proteger(preferencias.declaracion.proteger(protegido))), nil
	}
	cerrada := meritoshttp.NuevoHandlerCarteraNoDisponible()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil &&
			(r.URL.Path == meritoshttp.RutaCartera || r.URL.Path == meritoshttp.RutaRetiradasCartera || r.URL.Path == meritoshttp.RutaDeclaracionesCartera) {
			cerrada.ServeHTTP(w, r)
			return
		}
		protegido.ServeHTTP(w, r)
	}), nil
}
