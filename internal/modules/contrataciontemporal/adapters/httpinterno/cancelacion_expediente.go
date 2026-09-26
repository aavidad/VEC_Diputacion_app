package httpinterno

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Rutas de la cancelación del expediente antes de la fiscalización (CT122):
// el efecto y la consulta de opciones y de la cancelación registrada. Cada
// canal (RRHH y centro) se compone con su propia autoridad y su servicio.
const (
	RutaCancelacionesExpediente     = "/api/vec/contratacion-temporal/cancelaciones-expediente"
	RutaCancelacionExpediente       = "/api/vec/contratacion-temporal/cancelacion-expediente"
	maximoCuerpoCancelacion         = 8 * 1024
	esquemaConsultaCancelacionHTTP  = "vec.contratacion-temporal.cancelacion-expediente.v1"
	esquemaReciboCancelacionHTTP    = "vec.contratacion-temporal.recibo-cancelacion.v1"
	prefijoI18nErrorCancelacionHTTP = "api.contratacion_temporal.cancelacion.error."
)

// EjecutorCancelacionExpediente es el servicio de aplicación de un canal.
type EjecutorCancelacionExpediente interface {
	CancelarExpediente(context.Context, application.SolicitudCancelarExpediente) (ports.ReciboOperacionSeguimiento, error)
	Opciones(context.Context) (application.OpcionesCancelacion, error)
	Estado(context.Context, string, string) (ports.EstadoCancelacionExpediente, error)
}

type manejadorCancelacion struct {
	ruta      string
	autoridad AutoridadCanalSeguimiento
	lectura   AutorizadorLecturaSeguimiento
	ejecutor  EjecutorCancelacionExpediente
}

// NuevosManejadoresCancelacion devuelve el manejador de cada ruta. La
// identidad, la organización y el canal proceden de la autoridad compuesta,
// nunca del cuerpo.
func NuevosManejadoresCancelacion(a AutoridadCanalSeguimiento, l AutorizadorLecturaSeguimiento, e EjecutorCancelacionExpediente) (map[string]http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(l) || dependenciaNula(e) {
		return nil, errors.New("contratacion temporal http: cancelacion no disponible")
	}
	return map[string]http.Handler{
		RutaCancelacionesExpediente: &manejadorCancelacion{ruta: RutaCancelacionesExpediente, autoridad: a, lectura: l, ejecutor: e},
		RutaCancelacionExpediente:   &manejadorCancelacion{ruta: RutaCancelacionExpediente, autoridad: a, lectura: l, ejecutor: e},
	}, nil
}

func (h *manejadorCancelacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r == nil || r.URL == nil || r.URL.Path != h.ruta || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responderErrorCancelacion(w, r, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		responderErrorCancelacion(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if !tipoContenidoJSON(r.Header) || cabeceraCoberturaProhibida(r.Header) {
		responderErrorCancelacion(w, r, http.StatusBadRequest, "peticion_no_permitida")
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoCancelacion+1))
	if err != nil || len(contenido) == 0 || len(contenido) > maximoCuerpoCancelacion {
		responderErrorCancelacion(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	canal, err := h.autoridad.ResolverContextoCanalSeguimiento(r.Context())
	if err != nil || !canal.Valido() {
		responderErrorCancelacion(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	if h.ruta == RutaCancelacionExpediente {
		h.consultar(w, r, canal, contenido)
		return
	}
	var in struct {
		ExpedienteRef     string `json:"expediente_ref"`
		VersionEsperada   uint64 `json:"version_esperada"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		MotivoClave       string `json:"motivo_clave"`
		Observaciones     string `json:"observaciones"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil {
		responderErrorCancelacion(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	recibo, err := h.ejecutor.CancelarExpediente(r.Context(), application.SolicitudCancelarExpediente{Canal: canal, ExpedienteRef: in.ExpedienteRef,
		VersionEsperada: in.VersionEsperada, ClaveIdempotencia: in.ClaveIdempotencia, MotivoClave: domain.ClaveCatalogo(in.MotivoClave),
		Observaciones: in.Observaciones})
	if err != nil {
		estado, codigo := estadoErrorCancelacion(err)
		responderErrorCancelacion(w, r, estado, codigo, err)
		return
	}
	if recibo.OrganizacionRef != canal.OrganizacionRef || recibo.ExpedienteRef != in.ExpedienteRef ||
		recibo.EstadoResultante != domain.EstadoCancelado || !domain.InstanteUTCCanonico(recibo.RegistradaEn) {
		responderErrorCancelacion(w, r, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	responderJSONCobertura(w, r, http.StatusCreated, map[string]any{"data": map[string]any{"esquema": esquemaReciboCancelacionHTTP,
		"operacion": recibo.Operacion, "expediente_ref": recibo.ExpedienteRef, "version_anterior": recibo.VersionAnterior,
		"version_resultante": recibo.VersionResultante, "fase_resultante": string(recibo.FaseResultante),
		"estado_resultante": string(recibo.EstadoResultante), "motivo_clave": string(recibo.MotivoClave), "recibo_ref": recibo.ReciboRef,
		"auditoria_ref": recibo.AuditoriaRef, "evento_ref": recibo.EventoRef, "registrada_en": recibo.RegistradaEn.UTC().Format(time.RFC3339Nano)}})
}

// consultar devuelve las fases y los motivos del canal y, tras acreditar la
// lectura del expediente exacto, su cancelación registrada.
func (h *manejadorCancelacion) consultar(w http.ResponseWriter, r *http.Request, canal application.ContextoCanalSeguimiento, contenido []byte) {
	var in struct {
		ExpedienteRef string `json:"expediente_ref"`
	}
	if decodificarCuerpoSeguimiento(contenido, &in) != nil || !domain.ReferenciaOpacaValida(in.ExpedienteRef) {
		responderErrorCancelacion(w, r, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	if err := h.lectura.AutorizarLecturaSeguimiento(r.Context(), canal.OrganizacionRef, in.ExpedienteRef); err != nil {
		responderErrorCancelacion(w, r, http.StatusForbidden, "acceso_denegado", err)
		return
	}
	opciones, err := h.ejecutor.Opciones(r.Context())
	if err != nil {
		responderErrorCancelacion(w, r, http.StatusServiceUnavailable, "servicio_no_disponible", err)
		return
	}
	estado, err := h.ejecutor.Estado(r.Context(), canal.OrganizacionRef, in.ExpedienteRef)
	if err != nil {
		codigo, clave := estadoErrorCancelacion(err)
		responderErrorCancelacion(w, r, codigo, clave, err)
		return
	}
	fases := make([]string, 0, len(opciones.Fases))
	for _, f := range opciones.Fases {
		fases = append(fases, string(f))
	}
	motivos := make([]map[string]string, 0, len(opciones.Motivos))
	for _, m := range opciones.Motivos {
		motivos = append(motivos, map[string]string{"clave": string(m.Clave), "etiqueta": m.Etiqueta, "clave_i18n": m.ClaveI18n})
	}
	var cancelacion any
	if c := estado.Cancelacion; c != nil {
		cancelacion = map[string]string{"canal": string(c.Canal), "motivo_clave": c.MotivoClave, "fase_previa": c.FasePrevia,
			"observaciones": c.Observaciones, "recibo_ref": c.ReciboRef, "registrada_en": c.RegistradaEn.UTC().Format(time.RFC3339Nano)}
	}
	responderJSONCobertura(w, r, http.StatusOK, map[string]any{"data": map[string]any{"esquema": esquemaConsultaCancelacionHTTP,
		"expediente_ref": in.ExpedienteRef, "fases_admitidas": fases, "motivos": motivos, "cancelacion": cancelacion}})
}

func estadoErrorCancelacion(err error) (int, string) {
	switch {
	case errors.Is(err, ports.ErrCancelacionTrasFiscalizacion):
		return http.StatusConflict, "tras_fiscalizacion"
	case errors.Is(err, ports.ErrCancelacionNoAdmitida):
		return http.StatusConflict, "fase_no_admitida"
	case errors.Is(err, ports.ErrCancelacionYaRegistrada):
		return http.StatusConflict, "cancelacion_existente"
	case errors.Is(err, application.ErrServicioCancelacionInvalido):
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
	return estadoErrorSeguimiento(err)
}

func responderErrorCancelacion(w http.ResponseWriter, peticion *http.Request, estado int, codigo string, causas ...error) {
	responderJSONCobertura(w, peticion, estado, envoltorioErrorCobertura{Error: detalleErrorCobertura{Codigo: codigo,
		ClaveI18n: prefijoI18nErrorCancelacionHTTP + codigo, CorrelacionRef: nuevaCorrelacionCobertura()}}, causas...)
}
