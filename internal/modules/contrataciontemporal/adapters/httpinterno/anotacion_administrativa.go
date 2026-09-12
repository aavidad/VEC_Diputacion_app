package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaAnotacionesAdministrativas = "/api/vec/contratacion-temporal/expedientes/anotaciones-administrativas"
const RutaRecuperacionAnotacionesAdministrativas = RutaAnotacionesAdministrativas + "/recuperacion"

type ContextoCanalAnotacionAdministrativa struct{ AutenticacionRef, SesionRef, PerfilRef, OrganizacionRef string }

func (c ContextoCanalAnotacionAdministrativa) valido() bool {
	return (ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef}).Validar() == nil && domain.ReferenciaOpacaValida(c.OrganizacionRef)
}

type AutoridadContextoCanalAnotacionAdministrativa interface {
	ResolverContextoCanalAnotacionAdministrativa(context.Context) (ContextoCanalAnotacionAdministrativa, error)
}
type EjecutorAnotacionAdministrativa interface {
	RegistrarAnotacionAdministrativa(context.Context, application.SolicitudRegistrarAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error)
	RecuperarAnotacionAdministrativa(context.Context, string, string, ContextoCanalAnotacionAdministrativa) (ports.ReciboAnotacionAdministrativa, error)
}
type manejadorAnotacionAdministrativa struct {
	autoridad AutoridadContextoCanalAnotacionAdministrativa
	ejecutor  EjecutorAnotacionAdministrativa
}

func NuevoManejadorAnotacionAdministrativa(a AutoridadContextoCanalAnotacionAdministrativa, e EjecutorAnotacionAdministrativa) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(e) {
		return nil, errors.New("contratacion temporal http: anotacion no disponible")
	}
	return &manejadorAnotacionAdministrativa{a, e}, nil
}
func (h *manejadorAnotacionAdministrativa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil {
		errorAnotacion(w, 400, "peticion_no_valida")
		return
	}
	recuperacion := r.URL.Path == RutaRecuperacionAnotacionesAdministrativas
	if (!recuperacion && r.URL.Path != RutaAnotacionesAdministrativas) || (recuperacion && r.Method != http.MethodGet) || (!recuperacion && r.Method != http.MethodPost) {
		errorAnotacion(w, 404, "recurso_no_encontrado")
		return
	}
	if !recuperacion && (r.URL.RawQuery != "" || r.URL.ForceQuery) {
		errorAnotacion(w, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	c, err := h.autoridad.ResolverContextoCanalAnotacionAdministrativa(r.Context())
	if err != nil {
		responderErrorAnotacion(w, clasificarAnotacion(err))
		return
	}
	if !c.valido() {
		responderErrorAnotacion(w, errorAccesoAnotacionDenegado)
		return
	}
	if recuperacion {
		exp, key, valida := consultaRecuperacionAnotacion(r)
		if !valida {
			errorAnotacion(w, 400, "peticion_no_valida")
			return
		}
		recibo, err := h.ejecutor.RecuperarAnotacionAdministrativa(r.Context(), exp, key, c)
		if err != nil {
			responderErrorAnotacion(w, clasificarAnotacion(err))
			return
		}
		if !reciboAnotacionValidoParaContexto(recibo, c, exp, 0) {
			responderErrorAnotacion(w, errorServicioAnotacionNoDisponible)
			return
		}
		responderReciboAnotacion(w, http.StatusOK, recibo)
		return
	}
	if !tipoContenidoJSON(r.Header) {
		errorAnotacion(w, 422, "contenido_no_valido")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoAnotacionAdministrativaBytes+1))
	if err != nil {
		errorAnotacion(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	in, err := decodificarEntradaAnotacionAdministrativa(contenido)
	if err != nil {
		errorAnotacion(w, 422, "contenido_no_valido")
		return
	}
	recibo, err := h.ejecutor.RegistrarAnotacionAdministrativa(r.Context(), application.SolicitudRegistrarAnotacionAdministrativa{AutenticacionRef: c.AutenticacionRef, SesionRef: c.SesionRef, PerfilRef: c.PerfilRef, OrganizacionRef: c.OrganizacionRef, ExpedienteRef: in.ExpedienteRef, VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, Observaciones: in.Observaciones})
	if err != nil {
		responderErrorAnotacion(w, clasificarAnotacion(err))
		return
	}
	if !reciboAnotacionValidoParaContexto(recibo, c, in.ExpedienteRef, in.VersionEsperada) {
		responderErrorAnotacion(w, errorServicioAnotacionNoDisponible)
		return
	}
	responderReciboAnotacion(w, http.StatusCreated, recibo)
}

func consultaRecuperacionAnotacion(r *http.Request) (string, string, bool) {
	if r == nil || r.URL == nil || len(r.URL.RawQuery) > 600 || r.URL.ForceQuery || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		return "", "", false
	}
	q, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(q) != 2 || len(q["expediente_ref"]) != 1 || len(q["clave_idempotencia"]) != 1 {
		return "", "", false
	}
	exp, key := q.Get("expediente_ref"), q.Get("clave_idempotencia")
	return exp, key, domain.ReferenciaOpacaValida(exp) && ports.ClaveIdempotenciaValida(key)
}

func reciboAnotacionValidoParaContexto(r ports.ReciboAnotacionAdministrativa, c ContextoCanalAnotacionAdministrativa, expediente string, version uint64) bool {
	// La anotación conserva la fase y el estado del expediente. El dominio
	// permite fases publicadas, pero nunca registra sobre completado/cancelado.
	// Acotar ambas versiones antes de sumar evita 0->1 y overflow en GET.
	if r.VersionAnterior == 0 || r.VersionAnterior >= ports.MaximoEnteroSeguroOperacionAnalisis ||
		r.VersionResultante > ports.MaximoEnteroSeguroOperacionAnalisis ||
		!r.FaseResultante.Valida() || !r.EstadoResultante.Valido() ||
		r.EstadoResultante == domain.EstadoCompletado || r.EstadoResultante == domain.EstadoCancelado {
		return false
	}
	return r.Operacion == ports.OperacionRegistrarAnotacionAdministrativa &&
		r.OrganizacionRef == c.OrganizacionRef && r.ExpedienteRef == expediente &&
		r.VersionResultante == r.VersionAnterior+1 && (version == 0 || r.VersionAnterior == version) &&
		r.SeguimientoOriginal.Validar() == nil && domain.ReferenciaOpacaValida(r.ReciboRef) &&
		domain.ReferenciaOpacaValida(r.AuditoriaRef) && domain.ReferenciaOpacaValida(r.EventoRef) &&
		domain.ReferenciaOpacaValida(r.ActorRef) && domain.InstanteUTCCanonico(r.RegistradaEn)
}

var (
	errorAutenticacionAnotacionRequerida = nuevoErrorAnotacion(http.StatusUnauthorized, "autenticacion_requerida")
	errorAccesoAnotacionDenegado         = nuevoErrorAnotacion(http.StatusForbidden, "acceso_denegado")
	errorConflictoAnotacion              = nuevoErrorAnotacion(http.StatusConflict, "conflicto")
	errorServicioAnotacionNoDisponible   = nuevoErrorAnotacion(http.StatusServiceUnavailable, "servicio_no_disponible")
)

func nuevoErrorAnotacion(estado int, codigo string) errorPublicoCobertura {
	return errorPublicoCobertura{estado: estado, codigo: codigo, claveI18n: "api.contratacion_temporal.anotacion_administrativa.error." + codigo}
}

func clasificarAnotacion(e error) errorPublicoCobertura {
	switch {
	case errors.Is(e, ErrContextoCanalAusente), errors.Is(e, ErrContextoCanalCaducado):
		return errorAutenticacionAnotacionRequerida
	case errors.Is(e, ErrContextoCanalOrganizacionDenegada),
		errors.Is(e, application.ErrAnotacionAdministrativaDenegada),
		errors.Is(e, ports.ErrAutorizacionDenegada):
		return errorAccesoAnotacionDenegado
	case errors.Is(e, ports.ErrClaveIdempotenciaUsada),
		errors.Is(e, domain.ErrVersionEnConflicto),
		errors.Is(e, domain.ErrTransicionInvalida):
		return errorConflictoAnotacion
	default:
		// El contrato no publica causas de infraestructura ni de validación interna.
		return errorServicioAnotacionNoDisponible
	}
}

func errorAnotacion(w http.ResponseWriter, estado int, codigo string) {
	responderErrorAnotacion(w, nuevoErrorAnotacion(estado, codigo))
}

func responderErrorAnotacion(w http.ResponseWriter, problema errorPublicoCobertura) {
	responderJSONCobertura(w, problema.estado, envoltorioErrorCobertura{Error: detalleErrorCobertura{
		Codigo: problema.codigo, ClaveI18n: problema.claveI18n, CorrelacionRef: nuevaCorrelacionCobertura(),
	}})
}
func responderReciboAnotacion(w http.ResponseWriter, s int, r ports.ReciboAnotacionAdministrativa) {
	responderJSONCobertura(w, s, map[string]any{"data": r})
}
