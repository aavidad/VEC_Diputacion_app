package meritoshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const RutaConsultaPropia = "/api/meritos/hecho-propio/consulta"

// ProveedorSolicitudConsultaPropia obtiene la identidad y contexto registrados
// desde una frontera confiable. No recibe HTTP ni acepta identidad del cliente.
type ProveedorSolicitudConsultaPropia interface {
	SolicitudConsultaPropia(context.Context) (application.SolicitudConsultaPropia, error)
}

type LectorConsultaPropia interface {
	ConsultarActual(context.Context, application.SolicitudConsultaPropia) (ports.ResultadoConsultaPropia, error)
	RegistrarFalloConsulta(context.Context, application.SolicitudConsultaPropia, error) error
}

type ConsultaPropia struct {
	proveedor ProveedorSolicitudConsultaPropia
	lector    LectorConsultaPropia
}

var _ http.Handler = (*ConsultaPropia)(nil)
var _ LectorConsultaPropia = (*application.ServicioConsultaPropia)(nil)

// NuevaConsultaPropia conserva una ruta cerrada cuando falta una dependencia.
// La composición elige el proveedor confiable y el servicio real; este handler
// no crea identidad, permisos ni material V3.
func NuevaConsultaPropia(p ProveedorSolicitudConsultaPropia, l LectorConsultaPropia) *ConsultaPropia {
	return &ConsultaPropia{proveedor: p, lector: l}
}

func (h *ConsultaPropia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r == nil {
		respuestaErrorConsulta(w, http.StatusServiceUnavailable, "meritos.error.consulta_no_disponible")
		return
	}
	if r.URL == nil || r.URL.Path != RutaConsultaPropia || r.URL.EscapedPath() != RutaConsultaPropia {
		respuestaErrorConsulta(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		respuestaErrorConsulta(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
		return
	}
	if h == nil || dependenciaConsultaNula(h.proveedor) || dependenciaConsultaNula(h.lector) || r.Context().Err() != nil {
		respuestaErrorConsulta(w, http.StatusServiceUnavailable, "meritos.error.consulta_no_disponible")
		return
	}
	solicitud, err := h.proveedor.SolicitudConsultaPropia(r.Context())
	if err != nil {
		responderFalloConsulta(w, err)
		return
	}
	falloEntrada := func(causa error) {
		err := h.lector.RegistrarFalloConsulta(r.Context(), solicitud, causa)
		if err != application.ErrSolicitud {
			responderFalloConsulta(w, err)
			return
		}
		respuestaErrorConsulta(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
	}
	if err := r.Context().Err(); err != nil {
		falloEntrada(err)
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery {
		falloEntrada(application.ErrSolicitud)
		return
	}
	hecho, err := leerSelectorConsulta(r)
	if err != nil {
		falloEntrada(err)
		return
	}
	solicitud.HechoRef = hecho
	resultado, err := h.lector.ConsultarActual(r.Context(), solicitud)
	if err != nil {
		responderFalloConsulta(w, err)
		return
	}
	if err := r.Context().Err(); err != nil {
		responderFalloConsulta(w, h.lector.RegistrarFalloConsulta(r.Context(), solicitud, err))
		return
	}
	if resultado.Codigo == "denegada" {
		responderFalloConsulta(w, h.lector.RegistrarFalloConsulta(r.Context(), solicitud, vec.ErrAutorizacionDenegada))
		return
	}
	if !resultadoConsultaPublicable(hecho, resultado) {
		responderFalloConsulta(w, h.lector.RegistrarFalloConsulta(r.Context(), solicitud, ports.ErrConsultaNoDisponible))
		return
	}
	raw, err := json.Marshal(resultado)
	if err != nil || len(raw) > 65536 || r.Context().Err() != nil {
		responderFalloConsulta(w, h.lector.RegistrarFalloConsulta(r.Context(), solicitud, ports.ErrConsultaNoDisponible))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func leerSelectorConsulta(r *http.Request) (string, error) {
	tipo, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "application/json" || len(params) > 1 || params["charset"] != "" && !strings.EqualFold(params["charset"], "utf-8") ||
		len(params) == 1 && params["charset"] == "" || r.Header.Get("Content-Encoding") != "" || r.Body == nil || r.ContentLength > 4096 {
		return "", application.ErrSolicitud
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4097))
	if err != nil || len(raw) == 0 || len(raw) > 4096 {
		return "", application.ErrSolicitud
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	// La secuencia cerrada también rechaza claves duplicadas: exactamente una
	// propiedad de texto dentro de un objeto, seguida del final del cuerpo.
	inicio, err := decoder.Token()
	if err != nil || inicio != json.Delim('{') {
		return "", application.ErrSolicitud
	}
	clave, err := decoder.Token()
	if err != nil || clave != "hecho_ref" {
		return "", application.ErrSolicitud
	}
	var hecho string
	if decoder.Decode(&hecho) != nil || !domain.ReferenciaValida(hecho) {
		return "", application.ErrSolicitud
	}
	fin, err := decoder.Token()
	if err != nil || fin != json.Delim('}') || decoder.Decode(new(any)) != io.EOF {
		return "", application.ErrSolicitud
	}
	return hecho, nil
}

func resultadoConsultaPublicable(hecho string, r ports.ResultadoConsultaPropia) bool {
	if r.ReciboConsulta == nil || r.ReciboConsulta.HechoRef != hecho {
		return false
	}
	switch r.Codigo {
	case "obtenida":
		return r.HechoActual != nil && r.HechoActual.Referencia == hecho && r.HechoActual.Version > 0 && r.HechoActual.Version == r.ReciboConsulta.VersionConsultada
	case "no_encontrada":
		return r.HechoActual == nil && r.ReciboConsulta.VersionConsultada == 0
	default:
		return false
	}
}

func responderFalloConsulta(w http.ResponseWriter, err error) {
	if application.DenegacionConsultaReal(err) {
		respuestaErrorConsulta(w, http.StatusForbidden, "meritos.error.autorizacion_denegada")
		return
	}
	respuestaErrorConsulta(w, http.StatusServiceUnavailable, "meritos.error.consulta_no_disponible")
}

func respuestaErrorConsulta(w http.ResponseWriter, estado int, codigo string) {
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(struct {
		Codigo string `json:"codigo"`
	}{codigo})
}

func dependenciaConsultaNula(x any) bool {
	if x == nil {
		return true
	}
	v := reflect.ValueOf(x)
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	}
	return false
}
