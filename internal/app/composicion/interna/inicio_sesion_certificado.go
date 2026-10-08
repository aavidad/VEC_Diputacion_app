package interna

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

const rutaInicioSesionCertificado = "/api/vec/session/start"

var errInicioSesionCertificadoNoDisponible = errors.New("composicion interna: inicio de sesion certificado no disponible")

// El acto START no acepta identidad ni modo del cliente. La fachada consume
// una capacidad C4 de la misma petición y sólo retorna tras la confirmación
// durable de la presentación; las peticiones ordinarias usan reanudación.
type iniciadorSesionCertificado interface {
	IniciarPresentacionYVincular(context.Context, []byte) (context.Context, error)
}

type manejadorInicioSesionCertificado struct {
	extractor extractorAsercionInstitucional
	iniciador iniciadorSesionCertificado
	origen    string
	host      string
}

// origenHTTPS procede de material privado gobernado y debe coincidir con el
// nombre acreditado por TLS. Request.Host sólo se coteja, nunca lo selecciona.
func nuevoManejadorInicioSesionCertificado(extractor extractorAsercionInstitucional,
	iniciador iniciadorSesionCertificado, origenHTTPS, nombreTLS string,
) (http.Handler, error) {
	if interfazNulaIdentidadOffline(extractor) || interfazNulaIdentidadOffline(iniciador) ||
		origenHTTPS == "" || origenHTTPS != strings.TrimSpace(origenHTTPS) ||
		nombreTLS == "" {
		return nil, errInicioSesionCertificadoNoDisponible
	}
	origen, err := url.Parse(origenHTTPS)
	if err != nil || origen.Scheme != "https" || origen.Host == "" ||
		origen.Hostname() != nombreTLS || origen.User != nil || origen.Path != "" ||
		origen.RawPath != "" || origen.RawQuery != "" || origen.ForceQuery ||
		origen.Fragment != "" || origen.Opaque != "" || origen.String() != origenHTTPS {
		return nil, errInicioSesionCertificadoNoDisponible
	}
	return &manejadorInicioSesionCertificado{extractor: extractor, iniciador: iniciador,
		origen: origenHTTPS, host: origen.Host}, nil
}

func (h *manejadorInicioSesionCertificado) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if h == nil || r == nil || r.URL == nil || interfazNulaIdentidadOffline(h.extractor) ||
		interfazNulaIdentidadOffline(h.iniciador) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path != rutaInicioSesionCertificado || r.URL.RawPath != "" ||
		r.URL.EscapedPath() != r.URL.Path || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.Scheme != "" || r.URL.Host != "" || r.URL.User != nil ||
		r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.RequestURI != rutaInicioSesionCertificado {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if r.Context().Err() != nil || r.ContentLength != 0 ||
		(r.Body != nil && reflect.TypeOf(r.Body) != reflect.TypeOf(http.NoBody)) ||
		len(r.TransferEncoding) != 0 || len(r.Header.Values("Transfer-Encoding")) != 0 ||
		len(r.Header.Values("Content-Type")) != 0 ||
		len(r.Header.Values("Content-Length")) > 1 ||
		(len(r.Header.Values("Content-Length")) == 1 && r.Header.Get("Content-Length") != "0") {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !origenInicioSesionCertificadoValido(r, h.origen, h.host) ||
		!cabecerasCertificadoPersonalValidas(r.Header) {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	if r.TLS == nil || !r.TLS.HandshakeComplete || r.TLS.Version != tls.VersionTLS13 ||
		len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) < 2 ||
		len(r.TLS.PeerCertificates) == 0 {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	asercion, err := h.extractor.ExtraerAsercionProtegida(r)
	if err != nil || len(asercion) == 0 {
		clear(asercion)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer clear(asercion)
	vinculado, err := h.iniciador.IniciarPresentacionYVincular(r.Context(), asercion)
	if err != nil || vinculado == nil || vinculado.Err() != nil || r.Context().Err() != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func origenInicioSesionCertificadoValido(r *http.Request, esperado, host string) bool {
	if r == nil || r.Host != host || len(r.Header.Values("Origin")) != 1 ||
		r.Header.Get("Origin") != esperado || len(r.Header.Values("Referer")) > 1 ||
		len(r.Header.Values("Sec-Fetch-Site")) > 1 {
		return false
	}
	if sitio := r.Header.Get("Sec-Fetch-Site"); sitio != "" && sitio != "same-origin" {
		return false
	}
	if referencia := r.Header.Get("Referer"); referencia != "" {
		u, err := url.Parse(referencia)
		if err != nil || u.Scheme != "https" || u.Host != host || u.User != nil || u.Fragment != "" {
			return false
		}
	}
	return true
}
