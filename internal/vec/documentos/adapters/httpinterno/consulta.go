package httpinterno

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
)

const (
	RutaConsultaExpediente = "/api/vec/documentos/expedientes/consultas"
	RutaDescargaOriginal   = "/api/vec/documentos/originales/descargas"
	maxCuerpo              = 4096
	maxListado             = 100
	maxOriginal            = 20 * 1024 * 1024
)

var ErrManejadorInvalido = errors.New("documentos http: dependencias invalidas")

// La autoridad obtiene la concesion V3 del contexto autenticado por el canal.
// El servicio resuelve el contexto de custodia despues de su lectura autorizada.
// La referencia del cuerpo solo identifica el recurso; no concede autoridad.
type AutoridadContextoConsulta interface {
	ResolverConsultaExpediente(context.Context, string) (ports.AutorizacionV3, error)
	ResolverDescargaOriginal(context.Context, string, uint64) (ports.AutorizacionV3, error)
}

type ServicioLectura interface {
	ListarExpediente(context.Context, ports.ConsultaExpediente) (ports.PaginaDocumentos, error)
	DescargarOriginal(context.Context, ports.ConsultaDocumento) (ports.Original, error)
}

type manejador struct {
	servicio  ServicioLectura
	autoridad AutoridadContextoConsulta
	descarga  bool
}

// NuevasRutasExactas entrega los dos manejadores al unico dispatcher de la raiz.
// La raiz debe proporcionar tambien su AutoridadRutasExactas independiente.
func NuevasRutasExactas(servicio ServicioLectura, autoridad AutoridadContextoConsulta) ([]httpapi.RutaExacta, error) {
	if nula(servicio) || nula(autoridad) {
		return nil, ErrManejadorInvalido
	}
	return []httpapi.RutaExacta{
		{Ruta: RutaConsultaExpediente, Manejador: &manejador{servicio: servicio, autoridad: autoridad}},
		{Ruta: RutaDescargaOriginal, Manejador: &manejador{servicio: servicio, autoridad: autoridad, descarga: true}},
	}, nil
}
func nula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	}
	return false
}

func (h *manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cabeceras(w)
	if h == nil || nula(h.servicio) || nula(h.autoridad) || r == nil || r.URL == nil {
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	ruta := RutaConsultaExpediente
	if h.descarga {
		ruta = RutaDescargaOriginal
	}
	if !rutaExacta(r, ruta) {
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Context().Err() != nil {
		responderError(w, http.StatusRequestTimeout, "peticion_cancelada")
		return
	}
	if !tipoContenido(r) {
		responderError(w, http.StatusUnsupportedMediaType, "tipo_no_admitido")
		return
	}
	if h.descarga {
		h.servirOriginal(w, r)
	} else {
		h.servirLista(w, r)
	}
}
func cabeceras(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
}
func rutaExacta(r *http.Request, ruta string) bool {
	u := r.URL
	return u.Path == ruta && u.EscapedPath() == ruta && u.RawPath == "" && u.RawQuery == "" && !u.ForceQuery && u.Scheme == "" && u.Host == "" && u.User == nil && u.Fragment == "" && u.Opaque == ""
}
func tipoContenido(r *http.Request) bool {
	if len(r.Header.Values("Content-Type")) != 1 || len(r.Header.Values("Accept")) > 1 {
		return false
	}
	return r.Header.Get("Content-Type") == "application/json"
}
func decodificar(w http.ResponseWriter, r *http.Request, destino any) error {
	if r.Body == nil {
		return ports.ErrSolicitudInvalida
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxCuerpo))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return ports.ErrSolicitudInvalida
	}
	return nil
}
func (h *manejador) servirLista(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		ExpedienteRef string `json:"expediente_ref"`
		Cursor        string `json:"cursor"`
		Limite        uint32 `json:"limite"`
	}
	if err := decodificar(w, r, &entrada); err != nil || !domain.ReferenciaOpacaValida(entrada.ExpedienteRef) || entrada.Limite == 0 || entrada.Limite > maxListado || (entrada.Cursor != "" && !domain.ReferenciaValida(entrada.Cursor)) {
		responderError(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	autorizacion, err := h.autoridad.ResolverConsultaExpediente(r.Context(), entrada.ExpedienteRef)
	if err != nil {
		responderError(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	pagina, err := h.servicio.ListarExpediente(r.Context(), ports.ConsultaExpediente{ExpedienteRef: entrada.ExpedienteRef, Cursor: entrada.Cursor, Limite: entrada.Limite, Autorizacion: autorizacion})
	if err != nil {
		responderServicio(w, err)
		return
	}
	if len(pagina.Items) > int(entrada.Limite) ||
		(pagina.SiguienteCursor != "" && (!domain.ReferenciaValida(pagina.SiguienteCursor) || len(pagina.Items) == 0 || pagina.SiguienteCursor == entrada.Cursor)) {
		responderError(w, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	salida := make([]any, 0, len(pagina.Items))
	for _, d := range pagina.Items {
		if d.Validar() != nil || d.ExpedienteRef != entrada.ExpedienteRef {
			responderError(w, http.StatusBadGateway, "resultado_no_confiable")
			return
		}
		descargable := d.MIME == "application/pdf" || d.MIME == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		salida = append(salida, struct {
			Ref         string `json:"ref"`
			Numero      string `json:"numero_vec"`
			Tipo        string `json:"tipo"`
			Version     uint64 `json:"version"`
			EstadoFirma string `json:"estado_firma"`
			Huella      string `json:"huella"`
			MIME        string `json:"mime"`
			Descargable bool   `json:"descargable"`
		}{d.ID, d.NumeroVEC, d.TipoRef, d.Version, "pendiente_firma", d.HuellaSHA256, d.MIME, descargable})
	}
	estado := "disponible"
	if len(salida) == 0 {
		estado = "vacio"
	}
	responderJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"estado": estado, "documentos": salida, "siguiente_cursor": pagina.SiguienteCursor}})
}
func (h *manejador) servirOriginal(w http.ResponseWriter, r *http.Request) {
	var entrada struct {
		DocumentoRef string `json:"documento_ref"`
		Version      uint64 `json:"version"`
	}
	if err := decodificar(w, r, &entrada); err != nil || !domain.ReferenciaOpacaValida(entrada.DocumentoRef) || entrada.Version == 0 {
		responderError(w, http.StatusUnprocessableEntity, "contenido_no_valido")
		return
	}
	autorizacion, err := h.autoridad.ResolverDescargaOriginal(r.Context(), entrada.DocumentoRef, entrada.Version)
	if err != nil {
		responderError(w, http.StatusForbidden, "acceso_denegado")
		return
	}
	original, err := h.servicio.DescargarOriginal(r.Context(), ports.ConsultaDocumento{DocumentoID: entrada.DocumentoRef, Version: entrada.Version, Autorizacion: autorizacion})
	if err != nil {
		responderServicio(w, err)
		return
	}
	if len(original.Contenido) == 0 || len(original.Contenido) > maxOriginal || !domain.HuellaValida(original.HuellaSHA256) {
		responderError(w, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	suma := sha256.Sum256(original.Contenido)
	if !strings.EqualFold(hex.EncodeToString(suma[:]), original.HuellaSHA256) {
		responderError(w, http.StatusBadGateway, "resultado_no_confiable")
		return
	}
	extension := ""
	switch original.MIME {
	case "application/pdf":
		extension = "pdf"
		if !bytes.HasPrefix(original.Contenido, []byte("%PDF-")) {
			responderError(w, http.StatusBadGateway, "resultado_no_confiable")
			return
		}
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		extension = "docx"
		if !bytes.HasPrefix(original.Contenido, []byte("PK\x03\x04")) {
			responderError(w, http.StatusBadGateway, "resultado_no_confiable")
			return
		}
	default:
		responderError(w, http.StatusNotAcceptable, "formato_no_admitido")
		return
	}
	nombre := nombreArchivo(entrada.DocumentoRef, extension)
	w.Header().Set("Content-Type", original.MIME)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", nombre))
	w.Header().Set("X-Content-SHA256", strings.ToLower(original.HuellaSHA256))
	w.Header().Set("Content-Length", strconv.Itoa(len(original.Contenido)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(original.Contenido)
}
func nombreArchivo(ref, ext string) string {
	var b strings.Builder
	b.WriteString("documento-")
	for _, r := range ref {
		if b.Len() >= 90 {
			break
		}
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	b.WriteByte('.')
	b.WriteString(ext)
	return b.String()
}
func responderServicio(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, context.Canceled):
		responderError(w, http.StatusRequestTimeout, "peticion_cancelada")
	case errors.Is(err, context.DeadlineExceeded):
		responderError(w, http.StatusGatewayTimeout, "plazo_agotado")
	case errors.Is(err, ports.ErrOriginalNoDisponible):
		responderError(w, http.StatusNotFound, "recurso_no_encontrado")
	case errors.Is(err, ports.ErrSolicitudInvalida):
		responderError(w, http.StatusForbidden, "acceso_denegado")
	default:
		responderError(w, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}
func responderError(w http.ResponseWriter, status int, codigo string) {
	responderJSON(w, status, map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.documentos.error." + codigo}})
}
func responderJSON(w http.ResponseWriter, status int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		status = http.StatusInternalServerError
		data = []byte(`{"error":{"codigo":"error_interno"}}`)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
