package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	RutaCerrarAdministrativamente = "" +
		"/api/vec/contratacion-temporal/seguimiento/cerrar"
	RutaReabrirExcepcionalmente = "" +
		"/api/vec/contratacion-temporal/seguimiento/reabrir-excepcionalmente"
)

var ErrManejadorCierreAdministrativoInvalido = errors.New(
	"contratacion temporal http: manejador de cierre administrativo invalido",
)

// AutoridadServidorCierreAdministrativo obtiene la organizacion desde una
// frontera confiable. No recibe peticion HTTP, URL, cabeceras ni cuerpo.
type AutoridadServidorCierreAdministrativo interface {
	ResolverOrganizacionCierreAdministrativo(context.Context) (string, error)
}

// EjecutorCierreAdministrativo limita HTTP al caso de uso comun. Actor,
// perfil, unidad y autorizacion permanecen dentro de su transaccion.
type EjecutorCierreAdministrativo interface {
	Cerrar(
		context.Context,
		application.SolicitudCerrarAdministrativamente,
	) (ports.ResultadoCierreAdministrativo, error)
	ReabrirExcepcionalmente(
		context.Context,
		application.SolicitudReabrirExcepcionalmente,
	) (ports.ResultadoCierreAdministrativo, error)
}

type manejadorCierreAdministrativo struct {
	autoridad AutoridadServidorCierreAdministrativo
	ejecutor  EjecutorCierreAdministrativo
}

var _ http.Handler = (*manejadorCierreAdministrativo)(nil)
var _ EjecutorCierreAdministrativo = (*application.ServicioCierreAdministrativo)(nil)

// NuevoManejadorCierreAdministrativo no registra rutas, compone identidad ni
// crea otra autoridad. La composicion exterior conserva esas responsabilidades.
func NuevoManejadorCierreAdministrativo(
	autoridad AutoridadServidorCierreAdministrativo,
	ejecutor EjecutorCierreAdministrativo,
) (http.Handler, error) {
	if dependenciaNula(autoridad) || dependenciaNula(ejecutor) {
		return nil, ErrManejadorCierreAdministrativoInvalido
	}
	return &manejadorCierreAdministrativo{
		autoridad: autoridad,
		ejecutor:  ejecutor,
	}, nil
}

func (h *manejadorCierreAdministrativo) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	if h == nil || dependenciaNula(h.autoridad) || dependenciaNula(h.ejecutor) {
		responderErrorCierreAdministrativo(
			w,
			errorServicioCierreAdministrativoNoDisponible,
		)
		return
	}
	operacion, rutaValida := operacionCierreAdministrativoHTTP(r)
	if !rutaValida {
		responderErrorCierreAdministrativo(
			w,
			errorRecursoCierreAdministrativoNoEncontrado,
		)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorCierreAdministrativo(
			w,
			errorMetodoCierreAdministrativoNoPermitido,
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(err),
		)
		return
	}
	if problema := validarMetadatosCierreAdministrativo(r); problema != nil {
		responderErrorCierreAdministrativo(w, *problema)
		return
	}

	entrada, err := cierreAdministrativoDesdePeticion(w, r)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorCierreAdministrativo(
			w,
			errorEntradaCierreAdministrativo(err),
		)
		return
	}
	if !entrada.valida() {
		responderErrorCierreAdministrativo(
			w,
			errorContenidoCierreAdministrativoInvalido,
		)
		return
	}

	organizacionRef, err := h.autoridad.
		ResolverOrganizacionCierreAdministrativo(r.Context())
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(err),
		)
		return
	}
	solicitudPuerto := entrada.solicitudPuerto(organizacionRef, operacion)
	if solicitudPuerto.Validar() != nil {
		responderErrorCierreAdministrativo(
			w,
			errorServicioCierreAdministrativoNoDisponible,
		)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(err),
		)
		return
	}

	resultado, err := ejecutarCierreAdministrativoHTTP(
		r.Context(),
		h.ejecutor,
		operacion,
		entrada,
		organizacionRef,
	)
	if errContexto := r.Context().Err(); errContexto != nil {
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(errContexto),
		)
		return
	}
	if err != nil {
		if resultado != (ports.ResultadoCierreAdministrativo{}) {
			responderErrorCierreAdministrativo(
				w,
				errorResultadoCierreAdministrativoNoConfiable,
			)
			return
		}
		responderErrorCierreAdministrativo(
			w,
			clasificarErrorCierreAdministrativoHTTP(err),
		)
		return
	}
	salida, estadoHTTP, valida := proyectarCierreAdministrativo(
		solicitudPuerto,
		resultado,
	)
	if !valida {
		responderErrorCierreAdministrativo(
			w,
			errorResultadoCierreAdministrativoNoConfiable,
		)
		return
	}
	responderJSONCobertura(
		w,
		estadoHTTP,
		envoltorioCierreAdministrativo{Data: salida},
	)
}

func operacionCierreAdministrativoHTTP(
	r *http.Request,
) (ports.OperacionCierreAdministrativo, bool) {
	if r == nil || r.URL == nil || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
		r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" ||
		r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.EscapedPath() != r.URL.Path || strings.Contains(r.URL.Path, "%") {
		return "", false
	}
	switch r.URL.Path {
	case RutaCerrarAdministrativamente:
		return ports.OperacionCerrarAdministrativamente, true
	case RutaReabrirExcepcionalmente:
		return ports.OperacionReabrirExcepcionalmente, true
	default:
		return "", false
	}
}

func ejecutarCierreAdministrativoHTTP(
	ctx context.Context,
	ejecutor EjecutorCierreAdministrativo,
	operacion ports.OperacionCierreAdministrativo,
	entrada cierreAdministrativoEntradaJSON,
	organizacionRef string,
) (ports.ResultadoCierreAdministrativo, error) {
	switch operacion {
	case ports.OperacionCerrarAdministrativamente:
		return ejecutor.Cerrar(ctx, application.SolicitudCerrarAdministrativamente{
			OrganizacionRef:   organizacionRef,
			ExpedienteRef:     entrada.ExpedienteRef,
			SeguimientoRef:    entrada.SeguimientoRef,
			VersionEsperada:   *entrada.VersionEsperada,
			ClaveIdempotencia: entrada.ClaveIdempotencia,
			TransicionClave:   domain.ClaveCatalogo(entrada.TransicionClave),
			MotivoClave:       domain.ClaveCatalogo(entrada.MotivoClave),
		})
	case ports.OperacionReabrirExcepcionalmente:
		return ejecutor.ReabrirExcepcionalmente(
			ctx,
			application.SolicitudReabrirExcepcionalmente{
				OrganizacionRef:   organizacionRef,
				ExpedienteRef:     entrada.ExpedienteRef,
				SeguimientoRef:    entrada.SeguimientoRef,
				VersionEsperada:   *entrada.VersionEsperada,
				ClaveIdempotencia: entrada.ClaveIdempotencia,
				TransicionClave:   domain.ClaveCatalogo(entrada.TransicionClave),
				MotivoClave:       domain.ClaveCatalogo(entrada.MotivoClave),
			},
		)
	default:
		return ports.ResultadoCierreAdministrativo{},
			application.ErrSolicitudCierreAdministrativoInvalida
	}
}
