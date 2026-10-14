package httpinterno

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"vec-diputacion-granada/internal/modules/dietas/adapters/documentos"
	"vec-diputacion-granada/internal/modules/dietas/ports"
)

// ManejadorDocumentoLiquidacion reutiliza la misma lectura autorizada y
// auditada que la consulta JSON, antes de renderizar un solo byte.
type ManejadorDocumentoLiquidacion struct {
	captura ports.CapturadorLecturaLiquidacionEconomica
	lector  ports.LectorLiquidacionEconomica
	textos  map[string]documentos.TextosLiquidacionDocumento
}

func NuevoManejadorDocumentoLiquidacion(c ports.CapturadorLecturaLiquidacionEconomica, l ports.LectorLiquidacionEconomica, textos map[string]documentos.TextosLiquidacionDocumento) (*ManejadorDocumentoLiquidacion, error) {
	if dependenciaNula(c) || dependenciaNula(l) || textos["es"].Idioma != "es" || textos["en"].Idioma != "en" {
		return nil, ErrManejadorNoDisponible
	}
	return &ManejadorDocumentoLiquidacion{captura: c, lector: l, textos: textos}, nil
}

func RutaDocumentoLiquidacion(path string) (string, uint64, bool) {
	for _, prefijo := range []string{RutaLiquidacionesCircuito, RutaLiquidaciones} {
		resto, ok := strings.CutPrefix(path, prefijo)
		if !ok {
			continue
		}
		p := strings.Split(resto, "/")
		if len(p) != 4 || !referenciaCircuitoHTTP.MatchString(p[0]) || p[1] != "liquidaciones" || p[3] != "documento" {
			return "", 0, false
		}
		if p[2] == "actual" {
			return p[0], 0, true
		}
		v, valido := versionLiquidacionHTTP(p[2])
		return p[0], v, valido && v >= 2
	}
	return "", 0, false
}

func (m *ManejadorDocumentoLiquidacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Vary", "Accept-Language")
	if m == nil || r == nil || r.URL == nil || dependenciaNula(m.captura) || dependenciaNula(m.lector) {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" || r.Header.Get("X-Vec-Actor") != "" || r.Header.Get("X-Vec-Persona") != "" || r.Header.Get("X-Vec-Perfil") != "" || r.URL.EscapedPath() != r.URL.Path {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	ref, version, ok := RutaDocumentoLiquidacion(r.URL.Path)
	if !ok {
		responderError(w, http.StatusNotFound, "no_encontrada")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		responderError(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	valores, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || r.URL.ForceQuery || len(valores) != 1 || len(valores["formato"]) != 1 || r.ContentLength != 0 || len(r.TransferEncoding) != 0 || (r.Body != nil && r.Body != http.NoBody) || r.Header.Get("Content-Type") != "" {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	formato := valores.Get("formato")
	if formato != "pdf" && formato != "docx" {
		responderError(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	accept := "application/pdf"
	if formato == "docx" {
		accept = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	}
	if r.Header.Get("Accept") != accept {
		responderError(w, http.StatusNotAcceptable, "representacion_no_disponible")
		return
	}
	idioma := idiomaDocumentoLiquidacion(r.Header.Get("Accept-Language"))
	textos, ok := m.textos[idioma]
	if !ok {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	captura, err := m.captura.CapturarLecturaLiquidacionEconomica(r.Context(), ref)
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	resultado, err := m.lector.Consultar(r.Context(), captura, ports.ConsultaLiquidacionEconomica{Referencia: ref, Version: version})
	if err != nil {
		responderErrorLiquidacion(w, err)
		return
	}
	s := resultado.Liquidacion
	if s.Economia.ComisionRef != ref || (version != 0 && s.Constancia.VersionResultante != version) || resultado.Auditoria.Referencia == "" {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	contenido, tipoContenido, err := documentos.RenderizarLiquidacionEconomica(r.Context(), s, formato, textos)
	if err != nil {
		responderError(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	huella := sha256.Sum256(contenido)
	w.Header().Set("Content-Type", tipoContenido)
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(s.Constancia.NumeroDocumento+"."+formato))
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.Header().Set("X-Content-SHA256", hex.EncodeToString(huella[:]))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, bytes.NewReader(contenido)); err != nil {
		slog.Warn("dietas: descarga de documento interrumpida")
	}
}

func idiomaDocumentoLiquidacion(aceptado string) string {
	for _, preferencia := range strings.Split(aceptado, ",") {
		clave := strings.TrimSpace(strings.SplitN(preferencia, ";", 2)[0])
		if clave == "en" || strings.HasPrefix(clave, "en-") {
			return "en"
		}
		if clave == "es" || strings.HasPrefix(clave, "es-") {
			return "es"
		}
	}
	return "es"
}

var _ http.Handler = (*ManejadorDocumentoLiquidacion)(nil)
