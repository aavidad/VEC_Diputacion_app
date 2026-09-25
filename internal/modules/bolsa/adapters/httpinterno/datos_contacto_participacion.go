package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// B4: subrecurso `datos-contacto` de un candidato de la bolsa. POST registra una
// versión nueva (correo y hasta dos teléfonos) y GET devuelve la vigente. La
// respuesta trae siempre la forma enmascarada; el claro solo se incluye cuando
// el llamador lo pide expresamente con `?ver=completo`, que el caso de uso
// autoriza igual que la escritura.
type EntradaRegistrarDatosContactoParticipacion struct {
	BolsaRef, ParticipacionRef, Motivo, ClaveIdempotencia string
	Datos                                                 dominiobolsa.DatosContactoParticipacion
	// Origen es vacío (contacto propio) o «convoca» (duda 45).
	Origen string
}

type PreparadorDatosContactoParticipacion interface {
	PrepararSolicitudRegistrarDatosContacto(context.Context, EntradaRegistrarDatosContactoParticipacion) (puertosbolsa.SolicitudRegistrarDatosContactoParticipacion, error)
	PrepararSolicitudConsultarDatosContacto(context.Context, string, string) (puertosbolsa.SolicitudConsultarDatosContactoParticipacion, error)
}

type OperadorDatosContactoParticipacion interface {
	Registrar(context.Context, puertosbolsa.SolicitudRegistrarDatosContactoParticipacion) (puertosbolsa.RegistroDatosContactoParticipacion, error)
	Consultar(context.Context, puertosbolsa.SolicitudConsultarDatosContactoParticipacion) (puertosbolsa.DatosContactoParticipacionLeidos, error)
}

type HandlerDatosContactoParticipacion struct {
	preparador PreparadorDatosContactoParticipacion
	operador   OperadorDatosContactoParticipacion
}

func NuevoHandlerDatosContactoParticipacion(p PreparadorDatosContactoParticipacion, o OperadorDatosContactoParticipacion) (http.Handler, error) {
	if p == nil || o == nil {
		return nil, errors.New("bolsa http interno: datos de contacto no disponibles")
	}
	return &HandlerDatosContactoParticipacion{p, o}, nil
}

func (h *HandlerDatosContactoParticipacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bolsa, participacion, completo, ok := ReferenciasRutaDatosContactoParticipacion(r)
	if !ok {
		responderSituacion(w, http.StatusNotFound, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
		return
	}
	switch r.Method {
	case http.MethodGet:
		h.consultar(w, r, bolsa, participacion, completo)
	case http.MethodPost:
		h.registrar(w, r, bolsa, participacion)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		responderSituacion(w, http.StatusMethodNotAllowed, map[string]any{"error": map[string]string{"codigo": "metodo_no_permitido"}})
	}
}

func (h *HandlerDatosContactoParticipacion) consultar(w http.ResponseWriter, r *http.Request, bolsa, participacion string, completo bool) {
	if r.Body != nil && r.Body != http.NoBody && r.ContentLength > 0 {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	solicitud, err := h.preparador.PrepararSolicitudConsultarDatosContacto(r.Context(), bolsa, participacion)
	if err != nil {
		responderErrorDatosContacto(w, err)
		return
	}
	leidos, err := h.operador.Consultar(r.Context(), solicitud)
	if err != nil {
		responderErrorDatosContacto(w, err)
		return
	}
	datos := map[string]any{
		"participacion_ref": leidos.ParticipacionRef,
		"version":           leidos.Version,
		"registrada_en":     leidos.RegistradaEn.UTC().Format(time.RFC3339Nano),
		"enmascarados":      leidos.Enmascarados,
		"origen":            origenDatosContactoRespuesta(leidos.Origen, leidos.EstadoOrigen),
	}
	if completo {
		datos["correo"] = leidos.Datos.Correo
		datos["telefono_1"] = leidos.Datos.Telefono1
		datos["telefono_2"] = leidos.Datos.Telefono2
	}
	responderSituacion(w, http.StatusOK, map[string]any{"data": datos})
}

func (h *HandlerDatosContactoParticipacion) registrar(w http.ResponseWriter, r *http.Request, bolsa, participacion string) {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength <= 0 || r.ContentLength > 4096 || len(r.TransferEncoding) != 0 ||
		len(r.Header.Values("Content-Type")) != 1 || len(r.Header.Values("Accept")) != 1 ||
		r.Header.Get("Content-Type") != "application/json" || r.Header.Get("Accept") != "application/json" {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	clave := r.Header.Get("Idempotency-Key")
	if len(r.Header.Values("Idempotency-Key")) != 1 || clave == "" || strings.TrimSpace(clave) != clave || len(clave) > 256 {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	var cuerpo struct {
		Correo    string `json:"correo"`
		Telefono1 string `json:"telefono_1"`
		Telefono2 string `json:"telefono_2"`
		Motivo    string `json:"motivo"`
		Origen    string `json:"origen"`
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4097))
	dec.DisallowUnknownFields()
	if dec.Decode(&cuerpo) != nil || dec.Decode(&struct{}{}) != io.EOF ||
		strings.TrimSpace(cuerpo.Motivo) != cuerpo.Motivo || cuerpo.Motivo == "" || len(cuerpo.Motivo) > 1000 ||
		!dominiobolsa.OrigenDatosContactoAdmitido(cuerpo.Origen) {
		responderSituacion(w, http.StatusBadRequest, map[string]any{"error": map[string]string{"codigo": "solicitud_invalida"}})
		return
	}
	entrada := EntradaRegistrarDatosContactoParticipacion{
		BolsaRef: bolsa, ParticipacionRef: participacion, Motivo: cuerpo.Motivo, ClaveIdempotencia: clave, Origen: cuerpo.Origen,
		Datos: dominiobolsa.DatosContactoParticipacion{ParticipacionRef: participacion, Correo: cuerpo.Correo, Telefono1: cuerpo.Telefono1, Telefono2: cuerpo.Telefono2},
	}
	solicitud, err := h.preparador.PrepararSolicitudRegistrarDatosContacto(r.Context(), entrada)
	if err != nil {
		responderErrorDatosContacto(w, err)
		return
	}
	resultado, err := h.operador.Registrar(r.Context(), solicitud)
	if err != nil {
		responderErrorDatosContacto(w, err)
		return
	}
	estado := http.StatusCreated
	if resultado.Reutilizada {
		estado = http.StatusOK
	}
	responderSituacion(w, estado, map[string]any{"data": map[string]any{
		"participacion_ref": resultado.ParticipacionRef,
		"version":           resultado.Version,
		"registrada_en":     resultado.RegistradaEn.UTC().Format(time.RFC3339Nano),
		"recibo_ref":        resultado.ReciboRef,
		"reutilizada":       resultado.Reutilizada,
		"origen":            origenDatosContactoRespuesta(resultado.Origen, ""),
	}})
}

// origenDatosContactoRespuesta expone la marca de origen CONVOCA con la regla
// que la fijó; nil para un contacto propio. El estado solo acompaña a la
// lectura, que lo calcula a su hora.
func origenDatosContactoRespuesta(marca *dominiobolsa.MarcaOrigenDatosContacto, estado string) any {
	if marca == nil {
		return nil
	}
	origen := map[string]any{
		"origen":        marca.Origen,
		"vigente_hasta": marca.VigenteHasta.UTC().Format(time.RFC3339Nano),
		"ultimo_dia":    marca.UltimoDia,
		"regla_ref":     marca.ReglaRef,
	}
	if estado != "" {
		origen["estado"] = estado
	}
	return origen
}

// ReferenciasRutaDatosContactoParticipacion acepta
// /api/vec/bolsa/bolsas/{bolsa}/candidatos/{participacion}/datos-contacto,
// con la única consulta admitida `?ver=completo`.
func ReferenciasRutaDatosContactoParticipacion(r *http.Request) (string, string, bool, bool) {
	if r == nil || r.URL == nil || r.URL.RawPath != "" || strings.Contains(r.URL.EscapedPath(), "%") {
		return "", "", false, false
	}
	completo := false
	switch r.URL.RawQuery {
	case "":
	case "ver=completo":
		completo = true
	default:
		return "", "", false, false
	}
	if r.RequestURI != r.URL.Path && r.RequestURI != r.URL.Path+"?"+r.URL.RawQuery {
		return "", "", false, false
	}
	segmentos := strings.Split(strings.TrimPrefix(r.URL.Path, RutaBolsasGestion+"/"), "/")
	if len(segmentos) != 4 || segmentos[0] == "" || segmentos[1] != "candidatos" || segmentos[2] == "" || segmentos[3] != "datos-contacto" {
		return "", "", false, false
	}
	for _, v := range []string{segmentos[0], segmentos[2]} {
		if _, err := url.PathUnescape(v); err != nil || strings.ContainsAny(v, "?# \\%") || len(v) > 512 {
			return "", "", false, false
		}
	}
	return segmentos[0], segmentos[2], completo, true
}

func responderErrorDatosContacto(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dominiovec.ErrAutorizacionDenegada), errors.Is(err, dominiovec.ErrPermissionDenied):
		responderSituacion(w, http.StatusForbidden, map[string]any{"error": map[string]string{"codigo": "acceso_denegado"}})
	case errors.Is(err, dominiobolsa.ErrDatosContactoParticipacionInvalidos):
		responderSituacion(w, http.StatusConflict, map[string]any{"error": map[string]string{"codigo": "datos_contacto_en_conflicto"}})
	case errors.Is(err, puertosbolsa.ErrOrigenDatosContactoNoConfigurado):
		responderSituacion(w, http.StatusUnprocessableEntity, map[string]any{"error": map[string]string{"codigo": "origen_contacto_no_disponible"}})
	case errors.Is(err, puertosbolsa.ErrDatosContactoParticipacionNoEncontrados):
		responderSituacion(w, http.StatusNotFound, map[string]any{"error": map[string]string{"codigo": "recurso_no_encontrado"}})
	default:
		responderSituacion(w, http.StatusServiceUnavailable, map[string]any{"error": map[string]string{"codigo": "servicio_no_disponible"}})
	}
}
