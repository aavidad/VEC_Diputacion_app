package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaConsultaReciboRespuesta = "/api/vec/contratacion-temporal/llamamientos/respuestas/recibo"
const EsquemaConsultaReciboRespuesta = "vec.contratacion-temporal.recibo-respuesta-llamamiento.v1"

type EjecutorConsultaReciboRespuesta interface {
	Consultar(context.Context, ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error)
}

type reciboRespuestaConsultadoJSON struct {
	Esquema         string `json:"esquema"`
	OrganizacionRef string `json:"organizacion_ref"`
	ExpedienteRef   string `json:"expediente_ref"`
	ComunicacionRef string `json:"comunicacion_ref"`
	Respuesta       string `json:"respuesta"`
	JustificanteRef string `json:"justificante_ref"`
	ReciboRef       string `json:"recibo_ref"`
	AuditoriaRef    string `json:"auditoria_ref"`
	RegistradaEn    string `json:"registrada_en"`
	Estado          string `json:"estado"`
}

func NuevoManejadorConsultaReciboRespuesta(e EjecutorConsultaReciboRespuesta) (http.Handler, error) {
	if dependenciaNula(e) {
		return nil, ports.ErrConsultaReciboRespuestaFallo
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallo := func(estado int, codigo string, causa error) {
			responderErrorComunicacionLlamamiento(w, r, errorPublicoCobertura{
				estado: estado, codigo: codigo, claveI18n: "api.contratacion_temporal.respuesta_recibida.error." + codigo,
			}, causa)
		}
		if r == nil || r.URL == nil || r.URL.Path != RutaConsultaReciboRespuesta || r.URL.RawPath != "" ||
			r.URL.ForceQuery || r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil ||
			r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
			fallo(http.StatusNotFound, "recurso_no_encontrado", nil)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			fallo(http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
			return
		}
		if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 || len(r.URL.RawQuery) > 2048 {
			fallo(http.StatusBadRequest, "peticion_no_permitida", nil)
			return
		}
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil || len(q) != 3 || len(q["organizacion_ref"]) != 1 || len(q["expediente_ref"]) != 1 || len(q["comunicacion_ref"]) != 1 {
			fallo(http.StatusBadRequest, "peticion_no_permitida", err)
			return
		}
		s := ports.SolicitudConsultaReciboRespuesta{OrganizacionRef: q.Get("organizacion_ref"), ExpedienteRef: q.Get("expediente_ref"), ComunicacionRef: q.Get("comunicacion_ref")}
		if s.Validar() != nil {
			fallo(http.StatusBadRequest, "peticion_no_permitida", nil)
			return
		}
		resultado, err := e.Consultar(r.Context(), s)
		adjuntarAcuseAuditoriaLectura(w, err)
		if e := r.Context().Err(); e != nil {
			err = e
		}
		if err != nil {
			switch {
			case errors.Is(err, ports.ErrReciboRespuestaNoEncontrado):
				fallo(http.StatusNotFound, "recurso_no_encontrado", err)
			case errors.Is(err, ports.ErrConsultaReciboRespuestaDenegada):
				fallo(http.StatusForbidden, "acceso_denegado", err)
			case errors.Is(err, ports.ErrConsultaReciboRespuestaInvalida):
				fallo(http.StatusBadRequest, "peticion_no_permitida", err)
			case errors.Is(err, context.Canceled):
				fallo(http.StatusRequestTimeout, "peticion_cancelada", err)
			case errors.Is(err, context.DeadlineExceeded):
				fallo(http.StatusGatewayTimeout, "plazo_agotado", err)
			case errors.Is(err, ports.ErrReciboRespuestaNoConfiable):
				fallo(http.StatusBadGateway, "resultado_no_confiable", err)
			default:
				fallo(http.StatusServiceUnavailable, "servicio_no_disponible", err)
			}
			return
		}
		if resultado.ValidarPara(s) != nil {
			fallo(http.StatusBadGateway, "resultado_no_confiable", nil)
			return
		}
		salida := reciboRespuestaConsultadoJSON{
			Esquema: EsquemaConsultaReciboRespuesta, OrganizacionRef: resultado.OrganizacionRef, ExpedienteRef: resultado.ExpedienteRef,
			ComunicacionRef: resultado.ComunicacionRef, Respuesta: string(resultado.Respuesta),
			JustificanteRef: resultado.JustificanteRef, ReciboRef: resultado.ReciboRef,
			AuditoriaRef: resultado.AuditoriaRef, RegistradaEn: resultado.RegistradaEn.Format(time.RFC3339Nano),
			Estado: resultado.Estado,
		}
		responderJSONCobertura(w, r, http.StatusOK, struct {
			Data reciboRespuestaConsultadoJSON `json:"data"`
		}{salida})
	}), nil
}
