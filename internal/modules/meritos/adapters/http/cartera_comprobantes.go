package meritoshttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	documentosports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const RutaComprobantesCartera = "/api/vec/meritos/cartera/comprobantes"
const RutaEstadoComprobantesCartera = RutaComprobantesCartera + "/estado"
const maximoComprobanteCartera = 16 << 20
const maximoCuerpoComprobanteCartera = maximoComprobanteCartera + 64<<10

// El proveedor obtiene actor, vínculo y correlación de esta petición antes de
// leer el archivo. La resolución del recurso y su reserva queda en Documentos.
type ProveedorComprobanteCartera interface {
	PrepararComprobanteCartera(*http.Request) (IdentidadComprobanteCartera, error)
}

type IdentidadComprobanteCartera struct {
	ActorRegistradoV2 vec.ResultadoContextoActorRegistradoV2
	VinculoV2         vec.VinculoAutenticacionActorV2
	CorrelacionRef    string
}

type ResultadoCargaComprobanteCartera struct {
	ComprobanteRef string    `json:"comprobante_ref"`
	CargaID        string    `json:"carga_id"`
	RegistradoEn   time.Time `json:"registrado_en"`
	Estado         string    `json:"estado"`
}

type EstadoComprobanteCartera struct {
	CargaID        string     `json:"carga_id"`
	Estado         string     `json:"estado"`
	ReciboRef      string     `json:"recibo_ref,omitempty"`
	ComprobanteRef string     `json:"comprobante_ref,omitempty"`
	RegistradoEn   *time.Time `json:"registrado_en,omitempty"`
	CodigoRechazo  string     `json:"codigo_rechazo,omitempty"`
}

var ErrEstadoComprobanteNoEncontrado = errors.New("meritos.error.comprobante_no_encontrado")

// CargadorComprobanteCartera deriva en el servidor una correlación estable
// por persona y clave de declaración y delega el ciclo completo de custodia
// al servicio común. Solo devuelve un resultado tras admisión y registro real.
type CargadorComprobanteCartera interface {
	SubirComprobante(context.Context, IdentidadComprobanteCartera, string, []byte) (ResultadoCargaComprobanteCartera, error)
	ConsultarEstadoComprobante(context.Context, IdentidadComprobanteCartera, string) (EstadoComprobanteCartera, error)
	AbrirOriginalComprobante(context.Context, IdentidadComprobanteCartera, string) (io.ReadCloser, error)
}

type HandlerComprobantesCartera struct {
	proveedor ProveedorComprobanteCartera
	cargador  CargadorComprobanteCartera
}

var _ http.Handler = (*HandlerComprobantesCartera)(nil)

func NuevoHandlerComprobantesCartera(p ProveedorComprobanteCartera, c CargadorComprobanteCartera) (*HandlerComprobantesCartera, error) {
	if dependenciaConsultaNula(p) || dependenciaConsultaNula(c) {
		return nil, ports.ErrComprobanteMeritoNoDisponible
	}
	return &HandlerComprobantesCartera{proveedor: p, cargador: c}, nil
}

func (h *HandlerComprobantesCartera) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	if r == nil || r.URL == nil || r.URL.ForceQuery {
		falloComprobanteCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	if h == nil || dependenciaConsultaNula(h.proveedor) || dependenciaConsultaNula(h.cargador) || r.Context().Err() != nil {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	if r.URL.Path == RutaComprobantesCartera && r.URL.EscapedPath() == RutaComprobantesCartera {
		if r.URL.RawQuery != "" {
			falloComprobanteCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			falloComprobanteCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
			return
		}
		h.subir(w, r)
		return
	}
	if r.URL.Path == RutaEstadoComprobantesCartera && r.URL.EscapedPath() == RutaEstadoComprobantesCartera {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			falloComprobanteCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
			return
		}
		h.consultarEstado(w, r)
		return
	}
	if r.URL.RawQuery != "" {
		falloComprobanteCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		falloComprobanteCartera(w, http.StatusMethodNotAllowed, "meritos.error.metodo_no_admitido")
		return
	}
	ref, ok := referenciaOriginalComprobante(r.URL)
	if !ok {
		falloComprobanteCartera(w, http.StatusNotFound, "meritos.error.ruta_no_encontrada")
		return
	}
	h.descargar(w, r, ref)
}

func (h *HandlerComprobantesCartera) subir(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q, err := h.proveedor.PrepararComprobanteCartera(r)
	if err != nil {
		responderFalloCartera(w, err)
		return
	}
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	clave, archivo, err := leerMultipartComprobante(w, r)
	if err != nil {
		falloComprobanteCartera(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
		return
	}
	defer borrarBytesCartera(archivo)
	qConsulta := q
	// La correlación efímera de la petición no identifica una carga recuperable.
	q.CorrelacionRef = ""
	if http.DetectContentType(archivo[:min(len(archivo), 512)]) != "application/pdf" {
		falloComprobanteCartera(w, http.StatusUnprocessableEntity, "meritos.error.solicitud_invalida")
		return
	}
	resultado, err := h.cargador.SubirComprobante(r.Context(), q, clave, archivo)
	if errors.Is(err, documentosports.ErrSubidaLocalDenegada) {
		falloComprobanteCartera(w, http.StatusForbidden, "meritos.error.cartera_denegada")
		return
	}
	if err != nil || resultado.Estado != "admitido" {
		estado, fallo := h.cargador.ConsultarEstadoComprobante(r.Context(), qConsulta, clave)
		if fallo == nil && r.Context().Err() == nil && estadoComprobanteCarteraValido(estado) {
			switch estado.Estado {
			case "admitido":
				h.responderCargaAdmitida(w, ResultadoCargaComprobanteCartera{
					ComprobanteRef: estado.ComprobanteRef, CargaID: estado.CargaID,
					RegistradoEn: *estado.RegistradoEn, Estado: "admitido"})
			case "rechazado":
				responderEstadoComprobante(w, http.StatusUnprocessableEntity, estado)
			default:
				responderEstadoComprobante(w, http.StatusAccepted, estado)
			}
			return
		}
		if errors.Is(err, documentosports.ErrSubidaLocalPendiente) {
			falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_pendiente")
		} else {
			falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		}
		return
	}
	if err != nil || r.Context().Err() != nil ||
		!referenciaRespuestaComprobante(resultado.ComprobanteRef) || !referenciaRespuestaComprobante(resultado.CargaID) ||
		resultado.RegistradoEn.IsZero() || resultado.Estado != "admitido" {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	h.responderCargaAdmitida(w, resultado)
}

func (h *HandlerComprobantesCartera) responderCargaAdmitida(w http.ResponseWriter, resultado ResultadoCargaComprobanteCartera) {
	contenido, err := json.Marshal(struct {
		Data ResultadoCargaComprobanteCartera `json:"data"`
	}{resultado})
	if err != nil {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contenido)
}

func (h *HandlerComprobantesCartera) consultarEstado(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q, err := h.proveedor.PrepararComprobanteCartera(r)
	if err != nil {
		responderFalloCartera(w, err)
		return
	}
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil ||
		q.CorrelacionRef == "" ||
		r.Body != nil && r.Body != http.NoBody && r.ContentLength != 0 {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	clave, err := claveEstadoComprobante(r.URL.RawQuery)
	if err != nil {
		falloComprobanteCartera(w, http.StatusBadRequest, "meritos.error.solicitud_invalida")
		return
	}
	estado, err := h.cargador.ConsultarEstadoComprobante(r.Context(), q, clave)
	if errors.Is(err, documentosports.ErrSubidaLocalDenegada) {
		falloComprobanteCartera(w, http.StatusForbidden, "meritos.error.cartera_denegada")
		return
	}
	if errors.Is(err, ErrEstadoComprobanteNoEncontrado) {
		falloComprobanteCartera(w, http.StatusNotFound, "meritos.error.comprobante_no_encontrado")
		return
	}
	if err != nil || r.Context().Err() != nil || !estadoComprobanteCarteraValido(estado) {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	responderEstadoComprobante(w, http.StatusOK, estado)
}

func claveEstadoComprobante(raw string) (string, error) {
	if raw == "" || len(raw) > 1024 {
		return "", ports.ErrCarteraEntradaInvalida
	}
	valores, err := url.ParseQuery(raw)
	claves := valores["clave_idempotencia"]
	if err != nil || len(valores) != 1 || len(claves) != 1 || !domain.ReferenciaValida(claves[0]) {
		return "", ports.ErrCarteraEntradaInvalida
	}
	return claves[0], nil
}

func estadoComprobanteCarteraValido(e EstadoComprobanteCartera) bool {
	if !referenciaRespuestaComprobante(e.CargaID) ||
		e.ReciboRef != "" && !domain.ReferenciaValida(e.ReciboRef) {
		return false
	}
	switch e.Estado {
	case "reservada", "recibido", "analizando":
		return e.ComprobanteRef == "" && e.RegistradoEn == nil && e.CodigoRechazo == ""
	case "admitido":
		return referenciaRespuestaComprobante(e.ComprobanteRef) && e.RegistradoEn != nil &&
			!e.RegistradoEn.IsZero() && e.CodigoRechazo == ""
	case "rechazado":
		return e.ComprobanteRef == "" && e.RegistradoEn == nil &&
			(e.CodigoRechazo == "pdf_no_valido" || e.CodigoRechazo == "contenido_no_permitido" ||
				e.CodigoRechazo == "analisis_no_concluyente")
	default:
		return false
	}
}

func responderEstadoComprobante(w http.ResponseWriter, estadoHTTP int, estado EstadoComprobanteCartera) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	contenido, err := json.Marshal(struct {
		Data EstadoComprobanteCartera `json:"data"`
	}{estado})
	if err != nil {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	w.WriteHeader(estadoHTTP)
	_, _ = w.Write(contenido)
}

func leerMultipartComprobante(w http.ResponseWriter, r *http.Request) (string, []byte, error) {
	if r.Body == nil || r.Header.Get("Content-Encoding") != "" ||
		r.ContentLength > maximoCuerpoComprobanteCartera {
		return "", nil, ports.ErrCarteraEntradaInvalida
	}
	tipo, parametros, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || tipo != "multipart/form-data" || len(parametros) != 1 || parametros["boundary"] == "" {
		return "", nil, ports.ErrCarteraEntradaInvalida
	}
	r.Body = http.MaxBytesReader(w, r.Body, maximoCuerpoComprobanteCartera)
	defer r.Body.Close()
	lector, err := r.MultipartReader()
	if err != nil {
		return "", nil, ports.ErrCarteraEntradaInvalida
	}
	var clave string
	var archivo []byte
	for i := 0; i < 3; i++ {
		parte, err := lector.NextPart()
		if err == io.EOF {
			if clave == "" || len(archivo) == 0 {
				return "", nil, ports.ErrCarteraEntradaInvalida
			}
			return clave, archivo, nil
		}
		if err != nil {
			return "", nil, ports.ErrCarteraEntradaInvalida
		}
		switch parte.FormName() {
		case "clave_idempotencia":
			if clave != "" || parte.FileName() != "" {
				return "", nil, ports.ErrCarteraEntradaInvalida
			}
			b, err := io.ReadAll(io.LimitReader(parte, 257))
			if err != nil || len(b) == 0 || len(b) > 256 || !domain.ReferenciaValida(string(b)) {
				return "", nil, ports.ErrCarteraEntradaInvalida
			}
			clave = string(b)
		case "documento":
			if archivo != nil || parte.FileName() == "" {
				return "", nil, ports.ErrCarteraEntradaInvalida
			}
			archivo, err = io.ReadAll(io.LimitReader(parte, maximoComprobanteCartera+1))
			if err != nil || len(archivo) == 0 || len(archivo) > maximoComprobanteCartera {
				return "", nil, ports.ErrCarteraEntradaInvalida
			}
		default:
			return "", nil, ports.ErrCarteraEntradaInvalida
		}
	}
	return "", nil, ports.ErrCarteraEntradaInvalida
}

func referenciaOriginalComprobante(u *url.URL) (string, bool) {
	const prefijo = RutaComprobantesCartera + "/"
	const sufijo = "/original"
	escapada := u.EscapedPath()
	if !strings.HasPrefix(escapada, prefijo) || !strings.HasSuffix(escapada, sufijo) {
		return "", false
	}
	segmento := strings.TrimSuffix(strings.TrimPrefix(escapada, prefijo), sufijo)
	if segmento == "" || strings.Contains(segmento, "/") || len(segmento) > 768 {
		return "", false
	}
	ref, err := url.PathUnescape(segmento)
	return ref, err == nil && referenciaRespuestaComprobante(ref) &&
		u.Path == prefijo+ref+sufijo
}

func (h *HandlerComprobantesCartera) descargar(w http.ResponseWriter, r *http.Request, ref string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q, err := h.proveedor.PrepararComprobanteCartera(r)
	if err != nil {
		responderFalloCartera(w, err)
		return
	}
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil ||
		q.CorrelacionRef == "" || r.Body != nil && r.Body != http.NoBody && r.ContentLength != 0 {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	original, err := h.cargador.AbrirOriginalComprobante(r.Context(), q, ref)
	if errors.Is(err, documentosports.ErrSubidaLocalDenegada) {
		falloComprobanteCartera(w, http.StatusForbidden, "meritos.error.cartera_denegada")
		return
	}
	if err != nil || original == nil {
		if original != nil {
			_ = original.Close()
		}
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	contenido, err := io.ReadAll(io.LimitReader(original, maximoComprobanteCartera+1))
	cierreErr := original.Close()
	if err != nil || cierreErr != nil || r.Context().Err() != nil || len(contenido) == 0 ||
		len(contenido) > maximoComprobanteCartera || http.DetectContentType(contenido[:min(len(contenido), 512)]) != "application/pdf" {
		falloComprobanteCartera(w, http.StatusServiceUnavailable, "meritos.error.comprobante_no_disponible")
		return
	}
	defer borrarBytesCartera(contenido)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment")
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(contenido)
}

func referenciaRespuestaComprobante(ref string) bool {
	return strings.HasPrefix(ref, "ref:") && domain.ReferenciaValida(ref) &&
		!strings.ContainsAny(ref, ".")
}

func falloComprobanteCartera(w http.ResponseWriter, estado int, codigo string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	responderErrorCartera(w, estado, codigo)
}
