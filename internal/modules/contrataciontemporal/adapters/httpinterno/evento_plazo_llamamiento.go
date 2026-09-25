package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaEventoPlazoLlamamiento = "/api/vec/contratacion-temporal/llamamientos/plazos/eventos"
const EsquemaEventoPlazoLlamamiento = "vec.contratacion-temporal.evento-plazo-llamamiento.v1"

type EjecutorEventoPlazoLlamamiento interface {
	Registrar(context.Context, ports.SolicitudRegistrarEventoPlazoLlamamiento) (ports.EventoPlazoLlamamientoRegistrado, error)
}

type eventoPlazoJSON struct {
	ClaveIdempotencia           string `json:"clave_idempotencia"`
	OrganizacionRef             string `json:"organizacion_ref"`
	ExpedienteRef               string `json:"expediente_ref"`
	LlamamientoRef              string `json:"llamamiento_ref"`
	ComunicacionRef             string `json:"comunicacion_ref"`
	VersionComunicacionEsperada uint64 `json:"version_comunicacion_esperada"`
	Tipo                        string `json:"tipo"`
	InstanteEn                  string `json:"instante_en"`
	PruebaRef                   string `json:"prueba_ref"`
}

// plazoRespuestaJSON publica el vencimiento y las reglas que lo gobiernan.
// La propuesta de expiración se deriva de él: al vencer sin respuesta, VEC
// propone la no aceptación y el siguiente candidato y confirma quien indique
// confirmacion_expiracion. No incluye datos de la persona candidata.
type plazoRespuestaJSON struct {
	RespuestaHasta          string `json:"respuesta_hasta"`
	UltimoDia               string `json:"ultimo_dia"`
	PoliticaRef             string `json:"politica_ref"`
	TratamientoFueraDePlazo string `json:"tratamiento_fuera_de_plazo"`
	ConfirmacionExpiracion  string `json:"confirmacion_expiracion"`
	CriterioRespuestaRef    string `json:"criterio_respuesta_ref"`
	CriterioExpiracionRef   string `json:"criterio_expiracion_ref"`
	ReglaEjemplo            bool   `json:"regla_ejemplo"`
}

type eventoPlazoSalidaJSON struct {
	eventoPlazoJSON
	Esquema      string              `json:"esquema"`
	EventoRef    string              `json:"evento_ref"`
	ReciboRef    string              `json:"recibo_ref"`
	AuditoriaRef string              `json:"auditoria_ref"`
	RegistradoEn string              `json:"registrado_en"`
	Estado       string              `json:"estado"`
	Plazo        *plazoRespuestaJSON `json:"plazo,omitempty"`
}

// La composición protege la ruta con identidad y autorización. HTTP recibe la
// declaración de RRHH; nunca el vencimiento, la regla, el actor ni el perfil.
func NuevoManejadorEventoPlazoLlamamiento(e EjecutorEventoPlazoLlamamiento) (http.Handler, error) {
	if dependenciaNula(e) {
		return nil, errors.New("contratacion temporal http: ejecutor de plazo no disponible")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responderFallo := func(estado int, codigo string, causas ...error) {
			responderErrorComunicacionLlamamiento(w, r, errorPublicoCobertura{estado: estado, codigo: codigo,
				claveI18n: "api.contratacion_temporal.plazo_llamamiento.error." + codigo}, causas...)
		}
		if r == nil || r.URL == nil || r.URL.Path != RutaEventoPlazoLlamamiento ||
			r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.Scheme != "" ||
			r.URL.Host != "" || r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
			responderFallo(http.StatusNotFound, "recurso_no_encontrado")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			responderFallo(http.StatusMethodNotAllowed, "metodo_no_permitido")
			return
		}
		if p := validarMetadatosComunicacionLlamamiento(r); p != nil {
			responderFallo(p.estado, p.codigo)
			return
		}
		var entrada eventoPlazoJSON
		if err := decodificarComunicacionLlamamiento(w, r, &entrada); err != nil {
			p := errorEntradaComunicacionLlamamiento(err)
			responderFallo(p.estado, p.codigo)
			return
		}
		instante, err := time.Parse(time.RFC3339Nano, entrada.InstanteEn)
		s := ports.SolicitudRegistrarEventoPlazoLlamamiento{
			ClaveIdempotencia: entrada.ClaveIdempotencia, OrganizacionRef: entrada.OrganizacionRef,
			ExpedienteRef: entrada.ExpedienteRef, LlamamientoRef: entrada.LlamamientoRef,
			ComunicacionRef: entrada.ComunicacionRef, VersionComunicacionEsperada: entrada.VersionComunicacionEsperada,
			Tipo: ports.TipoEventoPlazoLlamamiento(entrada.Tipo), InstanteEn: instante, PruebaRef: entrada.PruebaRef,
		}
		if err != nil || s.Validar() != nil {
			responderFallo(http.StatusUnprocessableEntity, "contenido_no_valido")
			return
		}
		resultado, err := e.Registrar(r.Context(), s)
		if r.Context().Err() != nil {
			err = r.Context().Err()
		}
		if err != nil {
			estado, codigo := errorEventoPlazoHTTP(err)
			responderFallo(estado, codigo, err)
			return
		}
		if resultado.ValidarPara(s) != nil {
			responderFallo(http.StatusBadGateway, "resultado_no_confiable")
			return
		}
		entrada.InstanteEn = resultado.Solicitud.InstanteEn.Format(time.RFC3339Nano)
		salida := eventoPlazoSalidaJSON{eventoPlazoJSON: entrada, Esquema: EsquemaEventoPlazoLlamamiento,
			EventoRef: resultado.EventoRef, ReciboRef: resultado.ReciboRef, AuditoriaRef: resultado.AuditoriaRef,
			RegistradoEn: resultado.RegistradoEn.Format(time.RFC3339Nano), Estado: resultado.Estado}
		if p := resultado.Plazo; p != nil {
			salida.Plazo = &plazoRespuestaJSON{
				RespuestaHasta: p.RespuestaHasta.Format(time.RFC3339Nano), UltimoDia: p.UltimoDia,
				PoliticaRef: p.Politica.Referencia, TratamientoFueraDePlazo: string(p.TratamientoFueraDePlazo),
				ConfirmacionExpiracion: string(p.ConfirmacionExpiracion),
				CriterioRespuestaRef:   p.CriterioRespuesta.Referencia, CriterioExpiracionRef: p.CriterioExpiracion.Referencia,
				ReglaEjemplo: p.ReglaEjemplo,
			}
		}
		estado := http.StatusCreated
		if resultado.EsReplay() {
			estado = http.StatusOK
		}
		responderJSONCobertura(w, r, estado, struct {
			Data eventoPlazoSalidaJSON `json:"data"`
		}{salida})
	}), nil
}

func errorEventoPlazoHTTP(err error) (int, string) {
	switch {
	case errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout, "peticion_cancelada"
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout, "plazo_agotado"
	case errors.Is(err, application.ErrSolicitudEventoPlazoInvalida):
		return http.StatusUnprocessableEntity, "contenido_no_valido"
	case errors.Is(err, application.ErrEventoPlazoDenegado):
		return http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, application.ErrClaveEventoPlazoEnColision):
		return http.StatusConflict, "clave_idempotencia_reutilizada"
	case errors.Is(err, application.ErrEventoPlazoEnConflicto):
		return http.StatusConflict, "evento_en_conflicto"
	case errors.Is(err, application.ErrResultadoEventoPlazoNoConfiable):
		return http.StatusBadGateway, "resultado_no_confiable"
	case errors.Is(err, application.ErrReglasPlazoLlamamientoNoDisponibles):
		return http.StatusServiceUnavailable, "reglas_no_disponibles"
	default:
		return http.StatusServiceUnavailable, "servicio_no_disponible"
	}
}
