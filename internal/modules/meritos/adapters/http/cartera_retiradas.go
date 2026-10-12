package meritoshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const RutaRetiradasCartera = "/api/meritos/cartera/retiradas"

type ProveedorSolicitudRetiradaCartera interface {
	PrepararSolicitudRetiradaCartera(*http.Request) (application.SolicitudRetiradaCartera, error)
}

type RetiradorCartera interface {
	Retirar(context.Context, application.SolicitudRetiradaCartera) (ports.ConfirmacionRetiradaCartera, error)
}

type HandlerRetiradasCartera struct {
	proveedor ProveedorSolicitudRetiradaCartera
	servicio  RetiradorCartera
}

var _ http.Handler = (*HandlerRetiradasCartera)(nil)
var _ RetiradorCartera = (*application.ServicioRetiradaCartera)(nil)

func NuevoHandlerRetiradasCartera(p ProveedorSolicitudRetiradaCartera,
	s RetiradorCartera) (*HandlerRetiradasCartera, error) {
	if dependenciaConsultaNula(p) || dependenciaConsultaNula(s) {
		return nil, ports.ErrCarteraNoDisponible
	}
	return &HandlerRetiradasCartera{p, s}, nil
}

func (h *HandlerRetiradasCartera) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r == nil || r.URL == nil || r.URL.Path != RutaRetiradasCartera ||
		r.URL.EscapedPath() != RutaRetiradasCartera || r.URL.ForceQuery {
		responderErrorCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderErrorCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
		return
	}
	if h == nil || dependenciaConsultaNula(h.proveedor) || dependenciaConsultaNula(h.servicio) ||
		r.Context().Err() != nil {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	q, err := h.proveedor.PrepararSolicitudRetiradaCartera(r)
	if err != nil {
		responderFalloRetiradaCartera(w, err)
		return
	}
	if r.URL.RawQuery != "" || r.Body == nil || r.Header.Get("Content-Encoding") != "" ||
		r.ContentLength > 4096 || !contenidoJSONCartera(r.Header.Get("Content-Type")) {
		h.responderEntradaRetiradaInvalida(w, r, q)
		return
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 4097))
	if err != nil || len(b) == 0 || len(b) > 4096 {
		h.responderEntradaRetiradaInvalida(w, r, q)
		return
	}
	defer borrarBytesCartera(b)
	entrada, err := decodificarRetiradaHTTP(b)
	if err != nil {
		h.responderEntradaRetiradaInvalida(w, r, q)
		return
	}
	q.HechoRef, q.HechoVersionEsperada = entrada.HechoRef, entrada.HechoVersionEsperada
	q.DisponibilidadVersionEsperada = entrada.DisponibilidadVersionEsperada
	q.MotivoRetirada, q.ClaveIdempotencia = entrada.Motivo, entrada.ClaveIdempotencia
	confirmacion, err := h.servicio.Retirar(r.Context(), q)
	if err != nil {
		responderFalloRetiradaCartera(w, err)
		return
	}
	if r.Context().Err() != nil || confirmacion.Recibo.Referencia == "" ||
		confirmacion.AuditoriaAccesoRef == "" {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	contenido, err := json.Marshal(struct {
		Data ports.ConfirmacionRetiradaCartera `json:"data"`
	}{confirmacion})
	if err != nil || len(contenido) > 16384 {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	w.Header().Set("X-Audit-Ref", confirmacion.AuditoriaAccesoRef)
	estado := http.StatusCreated
	if confirmacion.Repetida {
		estado = http.StatusOK
	}
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

func (h *HandlerRetiradasCartera) responderEntradaRetiradaInvalida(w http.ResponseWriter,
	r *http.Request, q application.SolicitudRetiradaCartera) {
	q.HechoRef, q.HechoVersionEsperada, q.DisponibilidadVersionEsperada = "", 0, 0
	q.MotivoRetirada, q.ClaveIdempotencia = vec.ReferenciaEntradaCatalogo{}, ""
	_, err := h.servicio.Retirar(r.Context(), q) // Entrada inválida: solo asiento de intento.
	if !errors.Is(err, ports.ErrCarteraEntradaInvalida) {
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
		return
	}
	responderErrorCartera(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
}

func contenidoJSONCartera(valor string) bool {
	tipo, parametros, err := mime.ParseMediaType(valor)
	return err == nil && tipo == "application/json" &&
		(len(parametros) == 0 || len(parametros) == 1 && strings.EqualFold(parametros["charset"], "utf-8"))
}

type retiradaHTTP struct {
	HechoRef                      string
	HechoVersionEsperada          int
	DisponibilidadVersionEsperada uint64
	Motivo                        vec.ReferenciaEntradaCatalogo
	ClaveIdempotencia             string
}

func decodificarRetiradaHTTP(b []byte) (retiradaHTTP, error) {
	var vacia retiradaHTTP
	o, err := objetoCarteraJSON(b, []string{"hecho_ref", "hecho_version_esperada",
		"disponibilidad_version_esperada", "motivo", "clave_idempotencia"})
	if err != nil {
		return vacia, err
	}
	if _, err := objetoCarteraJSON(o["motivo"], []string{"catalogo_id", "catalogo_version",
		"catalogo_huella_sha256", "entrada_clave"}); err != nil {
		return vacia, err
	}
	if json.Unmarshal(o["hecho_ref"], &vacia.HechoRef) != nil ||
		json.Unmarshal(o["hecho_version_esperada"], &vacia.HechoVersionEsperada) != nil ||
		json.Unmarshal(o["disponibilidad_version_esperada"], &vacia.DisponibilidadVersionEsperada) != nil ||
		json.Unmarshal(o["motivo"], &vacia.Motivo) != nil ||
		json.Unmarshal(o["clave_idempotencia"], &vacia.ClaveIdempotencia) != nil {
		return retiradaHTTP{}, ports.ErrCarteraEntradaInvalida
	}
	return vacia, nil
}

func objetoCarteraJSON(raw []byte, esperadas []string) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	inicio, err := d.Token()
	if err != nil || inicio != json.Delim('{') {
		return nil, ports.ErrCarteraEntradaInvalida
	}
	permitidas := make(map[string]bool, len(esperadas))
	for _, clave := range esperadas {
		permitidas[clave] = true
	}
	vistas := make(map[string]json.RawMessage, len(esperadas))
	for d.More() {
		claveToken, err := d.Token()
		clave, ok := claveToken.(string)
		if err != nil || !ok || !permitidas[clave] || vistas[clave] != nil {
			return nil, ports.ErrCarteraEntradaInvalida
		}
		var valor json.RawMessage
		if err := d.Decode(&valor); err != nil {
			return nil, errors.Join(ports.ErrCarteraEntradaInvalida, err)
		}
		vistas[clave] = valor
	}
	fin, err := d.Token()
	if err != nil || fin != json.Delim('}') || len(vistas) != len(esperadas) ||
		d.Decode(new(any)) != io.EOF {
		return nil, ports.ErrCarteraEntradaInvalida
	}
	return vistas, nil
}

func borrarBytesCartera(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func responderFalloRetiradaCartera(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ports.ErrCarteraSinSesion):
		responderErrorCartera(w, http.StatusUnauthorized, "meritos.error.sesion_necesaria")
	case errors.Is(err, ports.ErrCarteraDenegada):
		responderErrorCartera(w, http.StatusForbidden, "meritos.error.cartera_denegada")
	case errors.Is(err, ports.ErrCarteraEntradaInvalida):
		responderErrorCartera(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
	case errors.Is(err, ports.ErrCarteraConflicto):
		responderErrorCartera(w, http.StatusConflict, "meritos.error.cartera_conflicto")
	default:
		responderErrorCartera(w, http.StatusServiceUnavailable, "meritos.error.cartera_no_disponible")
	}
}
