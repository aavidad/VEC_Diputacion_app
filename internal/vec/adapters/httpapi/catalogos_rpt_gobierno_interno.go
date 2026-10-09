package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	RutaProponerGobiernoCategoriaRPT  = "/api/vec/interno/catalogos/rpt/gobierno/proponer"
	RutaAprobarGobiernoCategoriaRPT   = "/api/vec/interno/catalogos/rpt/gobierno/aprobar"
	RutaConfirmarGobiernoCategoriaRPT = "/api/vec/interno/catalogos/rpt/gobierno/confirmar"
	maximoEntradaGobiernoCategoriaRPT = 8 << 20
)

var (
	ErrHandlerGobiernoCategoriaRPTInvalido = errors.New("vec http: gobierno RPT no disponible")
	ErrSesionGobiernoCategoriaRPTRequerida = errors.New("vec http: sesion interna requerida")
	ErrSesionGobiernoCategoriaRPTDenegada  = errors.New("vec http: sesion interna denegada")
	patronClaveGobiernoCategoriaRPT        = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// La raíz liga este puerto a una sesión breve y revalidada de certificado/PDP.
// Ningún dato de identidad procede de la petición HTTP.
type AutoridadGobiernoCategoriaRPTInterno interface {
	ResolverCredenciales(context.Context) (application.CredencialesGobiernoCategoriaRPT, error)
}

type OperadorGobiernoCategoriaRPTInterno interface {
	Proponer(context.Context, application.OrdenProponerGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error)
	Aprobar(context.Context, application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error)
	Confirmar(context.Context, application.OrdenAvanzarGobiernoCategoriaRPT) (ports.ResultadoGobiernoCategoriaRPT, error)
}

type AuditorRechazoGobiernoCategoriaRPTInterno interface {
	RegistrarRechazoGobiernoCategoriaRPT(context.Context, RechazoGobiernoCategoriaRPTInterno) error
}

type RechazoGobiernoCategoriaRPTInterno struct {
	Ruta           string
	Codigo         string
	CorrelacionRef string
}

type handlerGobiernoCategoriaRPTInterno struct {
	autoridad AutoridadGobiernoCategoriaRPTInterno
	operador  OperadorGobiernoCategoriaRPTInterno
	auditor   AuditorRechazoGobiernoCategoriaRPTInterno
}

var _ OperadorGobiernoCategoriaRPTInterno = (*application.ServicioGobiernoCategoriaRPT)(nil)

func NuevoHandlerGobiernoCategoriaRPTInterno(a AutoridadGobiernoCategoriaRPTInterno, o OperadorGobiernoCategoriaRPTInterno, audit AuditorRechazoGobiernoCategoriaRPTInterno) (http.Handler, error) {
	if nuloGobiernoCategoriaRPTHTTP(a) || nuloGobiernoCategoriaRPTHTTP(o) || nuloGobiernoCategoriaRPTHTTP(audit) {
		return nil, ErrHandlerGobiernoCategoriaRPTInvalido
	}
	return &handlerGobiernoCategoriaRPTInterno{a, o, audit}, nil
}

func nuloGobiernoCategoriaRPTHTTP(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}

func (h *handlerGobiernoCategoriaRPTInterno) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || nuloGobiernoCategoriaRPTHTTP(h.autoridad) || nuloGobiernoCategoriaRPTHTTP(h.operador) || nuloGobiernoCategoriaRPTHTTP(h.auditor) {
		responderGobiernoCategoriaRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	if r == nil || r.URL == nil || r.URL.RawQuery != "" || r.URL.ForceQuery || r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path || (r.RequestURI != "" && r.RequestURI != r.URL.Path) {
		responderGobiernoCategoriaRPT(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	ruta := r.URL.Path
	if ruta != RutaProponerGobiernoCategoriaRPT && ruta != RutaAprobarGobiernoCategoriaRPT && ruta != RutaConfirmarGobiernoCategoriaRPT {
		responderGobiernoCategoriaRPT(w, http.StatusNotFound, "recurso_no_encontrado", nil)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		responderGobiernoCategoriaRPT(w, http.StatusMethodNotAllowed, "metodo_no_permitido", nil)
		return
	}
	if cabeceraGobiernoCategoriaRPTPresente(r.Header, "Cookie") || cabeceraGobiernoCategoriaRPTPresente(r.Header, "Proxy-Authorization") {
		h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	cred, err := h.autoridad.ResolverCredenciales(r.Context())
	if err != nil {
		switch {
		case errors.Is(err, ErrSesionGobiernoCategoriaRPTRequerida):
			h.rechazar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
		case errors.Is(err, ErrSesionGobiernoCategoriaRPTDenegada):
			h.rechazar(w, r, http.StatusForbidden, "acceso_denegado")
		default:
			h.rechazar(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		}
		return
	}
	clave, ok := cabeceraGobiernoCategoriaRPTUnica(r.Header, "Idempotency-Key")
	if !ok || !patronClaveGobiernoCategoriaRPT.MatchString(clave) || !cabeceraGobiernoCategoriaRPTExacta(r.Header, "Content-Type", "application/json") ||
		cabeceraGobiernoCategoriaRPTPresente(r.Header, "Content-Encoding") || r.Body == nil || r.Body == http.NoBody || r.ContentLength == 0 || r.ContentLength > maximoEntradaGobiernoCategoriaRPT || len(r.TransferEncoding) != 0 || len(r.Trailer) != 0 {
		h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	defer r.Body.Close()
	cuerpo, err := io.ReadAll(io.LimitReader(http.MaxBytesReader(w, r.Body, maximoEntradaGobiernoCategoriaRPT+1), maximoEntradaGobiernoCategoriaRPT+1))
	if err != nil || len(cuerpo) == 0 || len(cuerpo) > maximoEntradaGobiernoCategoriaRPT || !utf8.Valid(cuerpo) || validarClavesUnicasGobiernoCategoriaRPT(cuerpo) != nil {
		h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
		return
	}
	recibo := reciboGobiernoCategoriaRPTHTTP(ruta, clave)
	var resultado ports.ResultadoGobiernoCategoriaRPT
	if ruta == RutaProponerGobiernoCategoriaRPT {
		var entrada ports.EntradaPropuestaGobiernoCategoriaRPT
		if decodificarGobiernoCategoriaRPT(cuerpo, &entrada) != nil || entrada.PropuestaRef == "" || entrada.FuenteRef == "" {
			h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
			return
		}
		contenido := domain.ContenidoGobiernoCategoriaRPT{Accion: entrada.Accion, CatalogoID: entrada.CatalogoID, ModuloID: entrada.ModuloID, Version: entrada.Version, DocumentoCanonico: entrada.DocumentoCanonico, PreimagenesControl: entrada.PreimagenesControl, CategoriaID: entrada.CategoriaID, RevisionEsperada: entrada.RevisionEsperada, MotivoRef: cred.Motivo.Referencia(), FuenteRef: entrada.FuenteRef}
		if !contenido.TamanoBorradorValido() {
			h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
			return
		}
		resultado, err = h.operador.Proponer(r.Context(), application.OrdenProponerGobiernoCategoriaRPT{Credenciales: cred, Borrador: ports.BorradorPropuestaGobiernoCategoriaRPT{PropuestaRef: entrada.PropuestaRef, ReciboRef: recibo, Contenido: contenido}})
	} else {
		var entrada ports.EntradaAvanceGobiernoCategoriaRPT
		if decodificarGobiernoCategoriaRPT(cuerpo, &entrada) != nil || entrada.PropuestaRef == "" || entrada.CatalogoID == "" || entrada.ModuloID == "" || entrada.HuellaSHA256 == "" || (ruta == RutaAprobarGobiernoCategoriaRPT && entrada.RevisionEsperada != 1) || (ruta == RutaConfirmarGobiernoCategoriaRPT && entrada.RevisionEsperada != 2) {
			h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
			return
		}
		orden := application.OrdenAvanzarGobiernoCategoriaRPT{Credenciales: cred, Material: ports.MaterialAvanceGobiernoCategoriaRPT{PropuestaRef: entrada.PropuestaRef, CatalogoID: entrada.CatalogoID, ModuloID: entrada.ModuloID, HuellaSHA256: entrada.HuellaSHA256, RevisionEsperada: entrada.RevisionEsperada, ReciboRef: recibo}}
		if ruta == RutaAprobarGobiernoCategoriaRPT {
			resultado, err = h.operador.Aprobar(r.Context(), orden)
		} else {
			resultado, err = h.operador.Confirmar(r.Context(), orden)
		}
	}
	if err != nil {
		switch {
		case errors.Is(err, application.ErrOrdenGobiernoCategoriaRPTInvalida), errors.Is(err, ports.ErrGobiernoCategoriaRPTInvalido):
			h.rechazar(w, r, http.StatusBadRequest, "peticion_no_valida")
		case errors.Is(err, ports.ErrGobiernoCategoriaRPTDenegado):
			h.rechazar(w, r, http.StatusForbidden, "acceso_denegado")
		case errors.Is(err, ports.ErrGobiernoCategoriaRPTConflicto):
			h.rechazar(w, r, http.StatusConflict, "conflicto")
		default:
			h.rechazar(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		}
		return
	}
	if resultado.ReciboRef != recibo || resultado.PropuestaRef == "" || resultado.Evidencia.DecisionRef == "" || resultado.Evidencia.AuditoriaRef == "" || resultado.Evidencia.ConsumoHuellaSHA256 == "" {
		h.rechazar(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	responderGobiernoCategoriaRPT(w, http.StatusOK, "", ports.RespuestaGobiernoCategoriaRPT{Data: resultado})
}

func decodificarGobiernoCategoriaRPT(cuerpo []byte, destino any) error {
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return ErrHandlerGobiernoCategoriaRPTInvalido
	}
	return nil
}

func validarClavesUnicasGobiernoCategoriaRPT(cuerpo []byte) error {
	d := json.NewDecoder(bytes.NewReader(cuerpo))
	if err := valorJSONGobiernoCategoriaRPT(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrHandlerGobiernoCategoriaRPTInvalido
	}
	return nil
}
func valorJSONGobiernoCategoriaRPT(d *json.Decoder, depth int) error {
	if depth > 24 {
		return ErrHandlerGobiernoCategoriaRPTInvalido
	}
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := k.(string)
			if !ok || keys[s] || len(s) > 160 {
				return ErrHandlerGobiernoCategoriaRPTInvalido
			}
			keys[s] = true
			if err := valorJSONGobiernoCategoriaRPT(d, depth+1); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return ErrHandlerGobiernoCategoriaRPTInvalido
		}
	case '[':
		for d.More() {
			if err := valorJSONGobiernoCategoriaRPT(d, depth+1); err != nil {
				return err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return ErrHandlerGobiernoCategoriaRPTInvalido
		}
	default:
		return ErrHandlerGobiernoCategoriaRPTInvalido
	}
	return nil
}

func cabeceraGobiernoCategoriaRPTUnica(h http.Header, nombre string) (string, bool) {
	var valores []string
	for clave, lista := range h {
		if strings.EqualFold(clave, nombre) {
			valores = append(valores, lista...)
		}
	}
	if len(valores) != 1 || valores[0] == "" || strings.TrimSpace(valores[0]) != valores[0] {
		return "", false
	}
	return valores[0], true
}
func cabeceraGobiernoCategoriaRPTExacta(h http.Header, nombre, esperado string) bool {
	valor, ok := cabeceraGobiernoCategoriaRPTUnica(h, nombre)
	return ok && valor == esperado
}
func cabeceraGobiernoCategoriaRPTPresente(h http.Header, nombre string) bool {
	for clave := range h {
		if strings.EqualFold(clave, nombre) {
			return true
		}
	}
	return false
}

func reciboGobiernoCategoriaRPTHTTP(ruta, clave string) string {
	suma := sha256.Sum256([]byte(ruta + "\x00" + clave))
	return "recibo:" + hex.EncodeToString(suma[:])
}
func (h *handlerGobiernoCategoriaRPTInterno) rechazar(w http.ResponseWriter, r *http.Request, estado int, codigo string) {
	suma := sha256.Sum256([]byte(r.URL.Path + "\x00" + time.Now().UTC().Format(time.RFC3339Nano)))
	orden := RechazoGobiernoCategoriaRPTInterno{Ruta: r.URL.Path, Codigo: codigo, CorrelacionRef: "corr_" + hex.EncodeToString(suma[:16])}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancel()
	if h.auditor.RegistrarRechazoGobiernoCategoriaRPT(ctx, orden) != nil {
		responderGobiernoCategoriaRPT(w, http.StatusServiceUnavailable, "servicio_no_disponible", nil)
		return
	}
	responderGobiernoCategoriaRPT(w, estado, codigo, nil)
}
func responderGobiernoCategoriaRPT(w http.ResponseWriter, estado int, codigo string, data any) {
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	if data == nil {
		data = map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.vec.catalogos.rpt.gobierno.error." + codigo}}
	}
	b, err := json.Marshal(data)
	if err != nil {
		estado = http.StatusServiceUnavailable
		b = []byte(`{"error":{"codigo":"servicio_no_disponible"}}`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(estado)
	_, _ = w.Write(b)
}

func _unusedGobiernoRPT(s string) bool { return strings.TrimSpace(s) == s }
