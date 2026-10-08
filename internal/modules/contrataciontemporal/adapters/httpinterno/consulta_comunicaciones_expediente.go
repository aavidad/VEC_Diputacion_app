package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const RutaConsultaComunicacionesExpediente = "/api/vec/contratacion-temporal/expedientes/comunicaciones"

type AutoridadCanalConsultaComunicacionesExpediente interface {
	ResolverContextoConsultaComunicacionesExpediente(context.Context) error
}

type ConsultorComunicacionesExpediente interface {
	ConsultarComunicacionesExpediente(context.Context, ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error)
}

// La composición registra y protege esta ruta exacta. El proveedor de
// persistencia obtiene el ámbito de organización del contexto sellado.
func NuevoManejadorConsultaComunicacionesExpediente(a AutoridadCanalConsultaComunicacionesExpediente, c ConsultorComunicacionesExpediente) (http.Handler, error) {
	if dependenciaNula(a) || dependenciaNula(c) {
		return nil, ports.ErrConsultaComunicacionesExpedienteNoDisponible
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || r.URL.Path != RutaConsultaComunicacionesExpediente ||
			r.URL.RawPath != "" || r.URL.Scheme != "" || r.URL.Host != "" ||
			r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" ||
			r.URL.RawFragment != "" || r.URL.ForceQuery || r.URL.EscapedPath() != r.URL.Path {
			responderErrorComunicacionLlamamiento(w, r, errorRecursoComunicacionLlamamientoNoEncontrado)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			responderErrorComunicacionLlamamiento(w, r, errorMetodoComunicacionLlamamientoNoPermitido)
			return
		}
		if r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Type") != "" ||
			!acceptCompatibleJSON(r.Header) {
			responderErrorComunicacionLlamamiento(w, r, errorPeticionComunicacionLlamamientoNoValida)
			return
		}
		peticion, err := leerConsultaComunicacionesExpediente(r.URL.RawQuery)
		if err != nil {
			responderErrorComunicacionLlamamiento(w, r, errorPeticionComunicacionLlamamientoNoValida)
			return
		}
		if err = r.Context().Err(); err != nil {
			responderErrorComunicacionLlamamiento(w, r, clasificarErrorConsultaComunicacionesExpediente(err))
			return
		}
		if err = a.ResolverContextoConsultaComunicacionesExpediente(r.Context()); err != nil {
			responderErrorComunicacionLlamamiento(w, r, clasificarErrorConsultaComunicacionesExpediente(err))
			return
		}
		pagina, err := c.ConsultarComunicacionesExpediente(r.Context(), peticion)
		adjuntarAcuseAuditoriaLectura(w, err)
		if errContexto := r.Context().Err(); errContexto != nil {
			responderErrorComunicacionLlamamiento(w, r, clasificarErrorConsultaComunicacionesExpediente(errContexto))
			return
		}
		if err != nil {
			responderErrorComunicacionLlamamiento(w, r, clasificarErrorConsultaComunicacionesExpediente(err))
			return
		}
		if pagina.ValidarPara(peticion) != nil {
			responderErrorComunicacionLlamamiento(w, r, errorResultadoComunicacionLlamamientoNoConfiable)
			return
		}
		responderJSONCobertura(w, r, http.StatusOK, struct {
			Data ports.PaginaComunicacionesExpediente `json:"data"`
		}{pagina})
	}), nil
}

func leerConsultaComunicacionesExpediente(raw string) (ports.ConsultaComunicacionesExpediente, error) {
	v, e := url.ParseQuery(raw)
	if e != nil || len(v) < 1 || len(v) > 3 {
		return ports.ConsultaComunicacionesExpediente{}, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	for k, a := range v {
		if (k != "expediente_ref" && k != "limite" && k != "cursor") || len(a) != 1 {
			return ports.ConsultaComunicacionesExpediente{}, ports.ErrConsultaComunicacionesExpedienteInvalida
		}
	}
	c := ports.ConsultaComunicacionesExpediente{ExpedienteRef: v.Get("expediente_ref"), Limite: 10, Cursor: v.Get("cursor")}
	if _, ok := v["limite"]; ok {
		if v.Get("limite") == "" || len(v.Get("limite")) > 2 {
			return ports.ConsultaComunicacionesExpediente{}, ports.ErrConsultaComunicacionesExpedienteInvalida
		}
		c.Limite, e = strconv.Atoi(v.Get("limite"))
		if e != nil {
			return ports.ConsultaComunicacionesExpediente{}, ports.ErrConsultaComunicacionesExpedienteInvalida
		}
	}
	if c.Validar() != nil {
		return ports.ConsultaComunicacionesExpediente{}, ports.ErrConsultaComunicacionesExpedienteInvalida
	}
	return c, nil
}

func clasificarErrorConsultaComunicacionesExpediente(err error) errorPublicoCobertura {
	switch {
	case errors.Is(err, ErrContextoCanalAusente), errors.Is(err, ErrContextoCanalCaducado):
		return nuevoErrorComunicacionLlamamiento(http.StatusUnauthorized, "autenticacion_requerida")
	case errors.Is(err, ports.ErrConsultaComunicacionesExpedienteDenegada):
		return errorAccesoComunicacionLlamamientoDenegado
	case errors.Is(err, ports.ErrConsultaComunicacionesExpedienteNoEncontrado):
		return errorRecursoComunicacionLlamamientoNoEncontrado
	case errors.Is(err, ports.ErrConsultaComunicacionesExpedienteInvalida):
		return errorPeticionComunicacionLlamamientoNoValida
	case errors.Is(err, context.Canceled):
		return errorCancelacionComunicacionLlamamiento
	case errors.Is(err, context.DeadlineExceeded):
		return errorPlazoComunicacionLlamamiento
	default:
		return errorServicioComunicacionLlamamientoNoDisponible
	}
}
