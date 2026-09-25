package httppersonal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// Rutas de las acciones propias del candidato. Identidad, candidato y perfil
// proceden siempre de la frontera; el cuerpo solo trae la bolsa y los datos
// de la acción.
const (
	RutaMiBolsaSolicitudes = "/api/vec/bolsa/mi-bolsa/solicitudes"
	RutaMiBolsaRespuestas  = "/api/vec/bolsa/mi-bolsa/respuestas"
	maximoCuerpoPortal     = 4096
)

// EsRutaPortal indica si la ruta pertenece a «Mi bolsa» del candidato.
func EsRutaPortal(ruta string) bool {
	return ruta == RutaMiBolsa || ruta == RutaMiBolsaSolicitudes || ruta == RutaMiBolsaRespuestas || ruta == RutaMiBolsaDisposiciones
}

// AccionPortalEn devuelve las acciones que admite cada ruta y método.
func AccionPortalEn(metodo, ruta string) []string {
	switch {
	case metodo == http.MethodGet && ruta == RutaMiBolsa:
		return []string{puertosbolsa.AccionConsultarMiBolsa}
	case metodo == http.MethodPost && ruta == RutaMiBolsaSolicitudes:
		return []string{puertosbolsa.AccionSolicitarPausaPropia, puertosbolsa.AccionSolicitarReactivacionPropia}
	case metodo == http.MethodPost && ruta == RutaMiBolsaRespuestas:
		return []string{puertosbolsa.AccionResponderLlamamientoPropio}
	case metodo == http.MethodPost && ruta == RutaMiBolsaDisposiciones:
		return []string{puertosbolsa.AccionManifestarDisposicionPropia}
	}
	return nil
}

type EjecutorPortal interface {
	SolicitarPausa(context.Context, mibolsa.Orden, string, time.Time, string) (puertosbolsa.ReciboSolicitudPortal, error)
	SolicitarReactivacion(context.Context, mibolsa.Orden, string, string) (puertosbolsa.ReciboSolicitudPortal, error)
	Responder(context.Context, mibolsa.Orden, mibolsa.ComandoRespuestaPortal) (puertosbolsa.ReciboRespuestaPortal, error)
}

type HandlerPortal struct {
	ruta       string
	preparador Preparador
	ejecutor   EjecutorPortal
}

func NuevoPortal(ruta string, preparador Preparador, ejecutor EjecutorPortal) (http.Handler, error) {
	if (ruta != RutaMiBolsaSolicitudes && ruta != RutaMiBolsaRespuestas) || nula(preparador) || nula(ejecutor) {
		return nil, ErrDependenciaNoDisponible
	}
	return &HandlerPortal{ruta: ruta, preparador: preparador, ejecutor: ejecutor}, nil
}

type entradaSolicitud struct {
	Tipo       string  `json:"tipo"`
	Bolsa      string  `json:"bolsa"`
	PausaHasta *string `json:"pausa_hasta"`
	Clave      string  `json:"clave"`
}

type entradaRespuesta struct {
	Bolsa              string `json:"bolsa"`
	Respuesta          string `json:"respuesta"`
	Causa              string `json:"causa"`
	JustificanteRef    string `json:"justificante_ref"`
	JustificanteSHA256 string `json:"justificante_sha256"`
	Clave              string `json:"clave"`
}

type reciboPortal struct {
	Data struct {
		Esquema      string  `json:"esquema"`
		Referencia   string  `json:"referencia"`
		Recibo       string  `json:"recibo"`
		RegistradaEn string  `json:"registrada_en"`
		Estado       string  `json:"estado"`
		Repetida     bool    `json:"repetida"`
		VenceAntesDe *string `json:"vence_antes_de,omitempty"`
	} `json:"data"`
}

func (h *HandlerPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || r == nil || nula(h.preparador) || nula(h.ejecutor) {
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
		return
	}
	if r.URL == nil || r.URL.Path != h.ruta || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		responder(w, 404, errorRespuesta{"recurso_no_encontrado"})
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responder(w, 405, errorRespuesta{"metodo_no_permitido"})
		return
	}
	// Un tipo de contenido no simple obliga al navegador a una comprobación
	// previa que ningún origen ajeno supera; Sec-Fetch-Site, si llega, debe
	// ser del mismo origen.
	if cabeceraProhibida(r.Header) || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" ||
		(r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin") ||
		r.ContentLength < 2 || r.ContentLength > maximoCuerpoPortal || len(r.TransferEncoding) != 0 || r.Body == nil {
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
		return
	}
	contenido, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maximoCuerpoPortal))
	if err != nil || int64(len(contenido)) != r.ContentLength {
		responder(w, 400, errorRespuesta{"peticion_no_permitida"})
		return
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if h.ruta == RutaMiBolsaSolicitudes {
		var entrada entradaSolicitud
		if decodificador.Decode(&entrada) != nil || decodificador.More() {
			responder(w, 400, errorRespuesta{"datos_no_validos"})
			return
		}
		h.solicitar(w, r, entrada)
		return
	}
	var entrada entradaRespuesta
	if decodificador.Decode(&entrada) != nil || decodificador.More() {
		responder(w, 400, errorRespuesta{"datos_no_validos"})
		return
	}
	h.responder(w, r, entrada)
}

func (h *HandlerPortal) solicitar(w http.ResponseWriter, r *http.Request, e entradaSolicitud) {
	var hasta time.Time
	switch e.Tipo {
	case puertosbolsa.SolicitudPortalPausa:
		var err error
		if e.PausaHasta == nil {
			responder(w, 400, errorRespuesta{"datos_no_validos"})
			return
		}
		if hasta, err = time.Parse(time.RFC3339Nano, *e.PausaHasta); err != nil {
			responder(w, 400, errorRespuesta{"datos_no_validos"})
			return
		}
	case puertosbolsa.SolicitudPortalReactivacion:
		if e.PausaHasta != nil {
			responder(w, 400, errorRespuesta{"datos_no_validos"})
			return
		}
	default:
		responder(w, 400, errorRespuesta{"datos_no_validos"})
		return
	}
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderErrorPortal(w, err)
		return
	}
	var recibo puertosbolsa.ReciboSolicitudPortal
	if e.Tipo == puertosbolsa.SolicitudPortalPausa {
		recibo, err = h.ejecutor.SolicitarPausa(r.Context(), orden, e.Bolsa, hasta, e.Clave)
	} else {
		recibo, err = h.ejecutor.SolicitarReactivacion(r.Context(), orden, e.Bolsa, e.Clave)
	}
	if err != nil {
		responderErrorPortal(w, err)
		return
	}
	var salida reciboPortal
	salida.Data.Esquema = "vec.bolsa.mi-bolsa.solicitud.v1"
	salida.Data.Referencia, salida.Data.Recibo, salida.Data.Repetida = recibo.SolicitudRef, recibo.ReciboRef, recibo.Reutilizada
	salida.Data.RegistradaEn = recibo.RegistradaEn.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
	salida.Data.Estado = "pendiente_rrhh"
	responder(w, estadoRecibo(recibo.Reutilizada), salida)
}

func (h *HandlerPortal) responder(w http.ResponseWriter, r *http.Request, e entradaRespuesta) {
	orden, err := h.preparador.PrepararMiBolsa(r)
	if err != nil {
		responderErrorPortal(w, err)
		return
	}
	recibo, err := h.ejecutor.Responder(r.Context(), orden, mibolsa.ComandoRespuestaPortal{
		Bolsa: e.Bolsa, Respuesta: e.Respuesta, Causa: e.Causa,
		JustificanteRef: e.JustificanteRef, JustificanteHash: e.JustificanteSHA256, Clave: e.Clave,
	})
	if err != nil {
		responderErrorPortal(w, err)
		return
	}
	var salida reciboPortal
	salida.Data.Esquema = "vec.bolsa.mi-bolsa.respuesta.v1"
	salida.Data.Referencia, salida.Data.Recibo, salida.Data.Repetida = recibo.RespuestaRef, recibo.ReciboRef, recibo.Reutilizada
	salida.Data.RegistradaEn = recibo.RespondidaEn.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
	salida.Data.Estado = recibo.Modo
	if !recibo.VenceAntesDe.IsZero() {
		vence := recibo.VenceAntesDe.UTC().Format("2006-01-02T15:04:05.000000Z07:00")
		salida.Data.VenceAntesDe = &vence
	}
	responder(w, estadoRecibo(recibo.Reutilizada), salida)
}

func estadoRecibo(repetida bool) int {
	if repetida {
		return http.StatusOK
	}
	return http.StatusCreated
}

func responderErrorPortal(w http.ResponseWriter, e error) {
	for _, conflicto := range []struct {
		err    error
		codigo string
	}{
		{puertosbolsa.ErrPortalClaveReutilizada, "clave_reutilizada"},
		{puertosbolsa.ErrPortalSolicitudPendiente, "solicitud_pendiente"},
		{puertosbolsa.ErrPortalSituacionNoAdmite, "situacion_no_admite"},
		{puertosbolsa.ErrPortalSinLlamamientoAbierto, "sin_llamamiento_abierto"},
		{puertosbolsa.ErrPortalRespuestaFueraDePlazo, "fuera_de_plazo"},
		{puertosbolsa.ErrPortalCausaNoAdmitida, "causa_no_admitida"},
		{puertosbolsa.ErrPortalPausaFueraDeLimite, "pausa_fuera_de_limite"},
	} {
		if errors.Is(e, conflicto.err) {
			responder(w, 409, errorRespuesta{conflicto.codigo})
			return
		}
	}
	switch {
	case errors.Is(e, puertosbolsa.ErrPortalCandidatoNoDisponible):
		responder(w, 503, errorRespuesta{"servicio_no_disponible"})
	case errors.Is(e, dominiovec.ErrAutorizacionDenegada), errors.Is(e, dominiovec.ErrPermissionDenied):
		responder(w, 403, errorRespuesta{"acceso_denegado"})
	case errors.Is(e, puertosbolsa.ErrPortalCandidatoInvalido):
		responder(w, 400, errorRespuesta{"datos_no_validos"})
	default:
		responderError(w, e)
	}
}

const formatoInstantePortal = "2006-01-02T15:04:05.000000Z07:00"

type estadoPortal struct {
	Bolsa              string `json:"bolsa"`
	LlamamientoAbierto *struct {
		ContactoEn   string  `json:"contacto_en"`
		VenceAntesDe *string `json:"vence_antes_de"`
	} `json:"llamamiento_abierto"`
	SolicitudPendiente *struct {
		Tipo         string  `json:"tipo"`
		Recibo       string  `json:"recibo"`
		RegistradaEn string  `json:"registrada_en"`
		PausaHasta   *string `json:"pausa_hasta"`
	} `json:"solicitud_pendiente"`
	UltimaRespuesta *struct {
		Respuesta    string `json:"respuesta"`
		Modo         string `json:"modo"`
		Recibo       string `json:"recibo"`
		RespondidaEn string `json:"respondida_en"`
	} `json:"ultima_respuesta"`
}

type accionesPortal struct {
	CausasRenuncia []string `json:"causas_renuncia"`
	PausaMaxima    string   `json:"pausa_maxima"`
	ModoRespuesta  string   `json:"modo_respuesta"`
}

func instantePortal(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := t.UTC().Format(formatoInstantePortal)
	return &v
}

func respuestaPortal(i puertosbolsa.InstantaneaMiBolsa) ([]estadoPortal, *accionesPortal) {
	if i.ReglasPortal == nil {
		return nil, nil
	}
	acciones := &accionesPortal{
		CausasRenuncia: append([]string{}, i.ReglasPortal.CausasRenuncia...),
		PausaMaxima:    i.ReglasPortal.PausaMaxima.UTC().Format(formatoInstantePortal),
		ModoRespuesta:  i.ReglasPortal.ModoRespuesta,
	}
	estados := make([]estadoPortal, 0, len(i.Portal))
	for _, p := range i.Portal {
		e := estadoPortal{Bolsa: p.Bolsa}
		if a := p.LlamamientoAbierto; a != nil {
			e.LlamamientoAbierto = &struct {
				ContactoEn   string  `json:"contacto_en"`
				VenceAntesDe *string `json:"vence_antes_de"`
			}{ContactoEn: a.ContactoEn.UTC().Format(formatoInstantePortal), VenceAntesDe: instantePortal(a.VenceAntesDe)}
		}
		if s := p.SolicitudPendiente; s != nil {
			e.SolicitudPendiente = &struct {
				Tipo         string  `json:"tipo"`
				Recibo       string  `json:"recibo"`
				RegistradaEn string  `json:"registrada_en"`
				PausaHasta   *string `json:"pausa_hasta"`
			}{Tipo: s.Tipo, Recibo: s.Recibo, RegistradaEn: s.RegistradaEn.UTC().Format(formatoInstantePortal), PausaHasta: instantePortal(s.PausaHasta)}
		}
		if u := p.UltimaRespuesta; u != nil {
			e.UltimaRespuesta = &struct {
				Respuesta    string `json:"respuesta"`
				Modo         string `json:"modo"`
				Recibo       string `json:"recibo"`
				RespondidaEn string `json:"respondida_en"`
			}{Respuesta: u.Respuesta, Modo: u.Modo, Recibo: u.Recibo, RespondidaEn: u.RespondidaEn.UTC().Format(formatoInstantePortal)}
		}
		estados = append(estados, e)
	}
	return estados, acciones
}
