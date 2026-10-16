package httpinterno

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

var claveConfirmacionDirecta = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,256}$`)

type OperadorConfirmacionDirecta interface {
	ConfirmarAceptacionDirecta(context.Context, ports.SolicitudConfirmarAceptacionDirecta) (ports.RegistroConfirmacionAceptacionDirecta, error)
}

type HandlerConfirmacionDirecta struct {
	preparador PreparadorSituacionParticipacion
	operador   OperadorConfirmacionDirecta
}

func NuevoHandlerConfirmacionDirecta(p PreparadorSituacionParticipacion, o OperadorConfirmacionDirecta) (http.Handler, error) {
	if dependenciaNula(p) || dependenciaNula(o) {
		return nil, ports.ErrSituacionParticipacionNoDisponible
	}
	return &HandlerConfirmacionDirecta{p, o}, nil
}

func (h *HandlerConfirmacionDirecta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, llamamiento, ok := ReferenciasRutaConfirmacionDirecta(r)
	if !ok {
		responderConfirmacionDirecta(w, 404, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderConfirmacionDirecta(w, 405, "metodo_no_permitido")
		return
	}
	if h == nil || dependenciaNula(h.preparador) || dependenciaNula(h.operador) ||
		r.Body == nil || r.Body == http.NoBody || r.ContentLength < 2 || r.ContentLength > 16*1024 ||
		len(r.TransferEncoding) != 0 || !cabeceraExactaLlamamiento(r.Header, "Accept", "application/json") ||
		!cabeceraExactaLlamamiento(r.Header, "Content-Type", "application/json") ||
		cabeceraEntornoNoConfiableLlamamientoPresente(r.Header) ||
		cabeceraPresente(r.Header, "Cookie") || cabeceraPresente(r.Header, "Authorization") ||
		cabeceraPresente(r.Header, "Proxy-Authorization") || cabeceraPresente(r.Header, "Content-Encoding") ||
		cabeceraPresente(r.Header, "Trailer") || cabeceraPresente(r.Header, "If-Match") ||
		cabeceraPresente(r.Header, "If-None-Match") ||
		(cabeceraPresente(r.Header, "Sec-Fetch-Site") &&
			!cabeceraExactaLlamamiento(r.Header, "Sec-Fetch-Site", "same-origin")) {
		responderConfirmacionDirecta(w, 400, "solicitud_invalida")
		return
	}
	clave, err := cabeceraUnicaLlamamiento(r.Header, "Idempotency-Key")
	if err != nil || !claveConfirmacionDirecta.MatchString(clave) {
		responderConfirmacionDirecta(w, 400, "solicitud_invalida")
		return
	}
	var cuerpo struct {
		Motivo                 string `json:"motivo"`
		SituacionEsperadaDesde string `json:"situacion_desde_esperada"`
	}
	if decodificarCuerpoLlamamiento(w, r, &cuerpo, 16*1024) != nil || cuerpo.Motivo == "" ||
		strings.TrimSpace(cuerpo.Motivo) != cuerpo.Motivo || len(cuerpo.Motivo) > 1000 {
		responderConfirmacionDirecta(w, 400, "solicitud_invalida")
		return
	}
	esperada, err := time.Parse(time.RFC3339Nano, cuerpo.SituacionEsperadaDesde)
	if err != nil || esperada.IsZero() || esperada.Nanosecond()%1_000 != 0 {
		responderConfirmacionDirecta(w, 400, "solicitud_invalida")
		return
	}
	entrada := EntradaCambiarSituacionParticipacion{BolsaRef: bolsa, ParticipacionRef: participacion,
		Destino: domain.SituacionPendienteIncorporacion, Motivo: cuerpo.Motivo, ClaveIdempotencia: clave}
	base, err := h.preparador.PrepararSolicitudCambiarSituacion(r.Context(), entrada)
	if err != nil {
		responderErrorSituacion(w, err)
		return
	}
	if base.BolsaRef != bolsa || base.ParticipacionRef != participacion ||
		base.Destino != entrada.Destino || base.Motivo != cuerpo.Motivo ||
		base.ClaveIdempotencia != clave || base.FechaDisponible != nil {
		responderConfirmacionDirecta(w, 503, "servicio_no_disponible")
		return
	}
	q := ports.SolicitudConfirmarAceptacionDirecta{SolicitudCambiarSituacionParticipacion: base,
		LlamamientoRef: llamamiento, SituacionEsperadaDesde: esperada.UTC()}
	registro, err := h.operador.ConfirmarAceptacionDirecta(r.Context(), q)
	if err != nil {
		responderErrorSituacion(w, err)
		return
	}
	if registro.RespuestaRef == "" || registro.ResolucionRef == "" || registro.ReciboRef == "" ||
		registro.ResueltaEn == nil || registro.ParticipacionRef != participacion ||
		registro.BolsaRef != bolsa || registro.LlamamientoRef != llamamiento ||
		registro.Situacion != domain.SituacionPendienteIncorporacion ||
		registro.Motivo != cuerpo.Motivo || registro.Desde.IsZero() {
		responderConfirmacionDirecta(w, 503, "servicio_no_disponible")
		return
	}
	estado := 201
	if registro.Reutilizada {
		estado = 200
	}
	responderSituacion(w, estado, map[string]any{"data": map[string]any{
		"respuesta_ref": registro.RespuestaRef, "resolucion_ref": registro.ResolucionRef,
		"participacion_ref": registro.ParticipacionRef, "situacion": registro.Situacion,
		"desde": registro.Desde.UTC().Format(time.RFC3339Nano), "motivo": registro.Motivo,
		"recibo_ref": registro.ReciboRef, "resuelta_en": registro.ResueltaEn.UTC().Format(time.RFC3339Nano),
		"reutilizada": registro.Reutilizada,
	}})
}

func ReferenciasRutaConfirmacionDirecta(r *http.Request) (string, string, string, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		r.RequestURI != r.URL.Path || strings.Contains(r.URL.EscapedPath(), "%") ||
		!strings.HasPrefix(r.URL.Path, RutaBolsasGestion+"/") {
		return "", "", "", false
	}
	partes := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(partes) != 6 || partes[1] != "candidatos" || partes[3] != "respuestas" || partes[5] != "confirmacion" {
		return "", "", "", false
	}
	for _, ref := range []string{partes[0], partes[2], partes[4]} {
		if !ports.ReferenciaOpacaLlamamientoValida(ref) ||
			strings.ContainsAny(ref, "?# \\%") {
			return "", "", "", false
		}
	}
	return partes[0], partes[2], partes[4], true
}

func responderConfirmacionDirecta(w http.ResponseWriter, estado int, codigo string) {
	responderSituacion(w, estado, map[string]any{"error": map[string]string{"codigo": codigo}})
}
