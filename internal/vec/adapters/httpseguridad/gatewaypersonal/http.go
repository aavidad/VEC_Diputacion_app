package gatewaypersonal

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const nombreCookie = "__Host-vec_session"

// ServicioSesion expresa el puerto real sin filtrar identidad hacia HTTP.
type ServicioSesion interface {
	Abrir(ctx context.Context, cuenta, huella string) (string, time.Time, error)
	Activa(ctx context.Context, token string) (bool, time.Time, error)
	Cerrar(ctx context.Context, token string) error
}

type Handler struct {
	sesiones ServicioSesion
	origen   string
	cuentas  map[string]string
	crl      *x509.RevocationList
	agora    func() time.Time
	raizWeb  string
}

func NuevoHandler(s ServicioSesion, origen string, cuentas map[string]string, crl *x509.RevocationList, ahora func() time.Time) *Handler {
	if ahora == nil {
		ahora = time.Now
	}
	clon := make(map[string]string, len(cuentas))
	for h, c := range cuentas {
		clon[strings.ToLower(h)] = c
	}
	return &Handler{sesiones: s, origen: origen, cuentas: clon, crl: crl, agora: ahora}
}

func (h *Handler) Rutas() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /acceso/", h.activo)
	mux.HandleFunc("HEAD /acceso/", h.activo)
	mux.HandleFunc("GET /api/acceso/sesion", h.sesion)
	mux.HandleFunc("POST /api/acceso/certificado", h.certificado)
	mux.HandleFunc("POST /api/acceso/cerrar", h.cerrar)
	return noStore(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/acceso/") && !rutaActivaPermitida(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		mux.ServeHTTP(w, r)
	}))
}

func (h *Handler) ConfigurarActivos(raiz string) error {
	if h == nil || !filepath.IsAbs(raiz) {
		return ErrAlmacen
	}
	i, err := os.Lstat(raiz)
	if err != nil || !i.IsDir() || i.Mode()&os.ModeSymlink != 0 {
		return ErrAlmacen
	}
	canon, err := filepath.EvalSymlinks(raiz)
	if err != nil || canon != filepath.Clean(raiz) {
		return ErrAlmacen
	}
	h.raizWeb = canon
	return nil
}
func (h *Handler) activo(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.raizWeb == "" {
		http.NotFound(w, r)
		return
	}
	permitidos := map[string]struct{ nombre, tipo string }{
		"/acceso/": {"index.html", "text/html; charset=utf-8"}, "/acceso/acceso.js": {"acceso.js", "text/javascript; charset=utf-8"}, "/acceso/gateway.css": {"gateway.css", "text/css; charset=utf-8"}, "/acceso/locales/es.json": {"locales/es.json", "application/json; charset=utf-8"},
	}
	activo, ok := permitidos[r.URL.Path]
	if !ok || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	ruta := filepath.Join(h.raizWeb, activo.nombre)
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", activo.tipo)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	if r.Method == http.MethodHead {
		return
	}
	f, err := os.Open(ruta)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	_, _ = io.Copy(w, f)
}
func rutaActivaPermitida(ruta string) bool {
	switch ruta {
	case "/acceso/", "/acceso/acceso.js", "/acceso/gateway.css", "/acceso/locales/es.json":
		return true
	}
	return false
}

func (h *Handler) sesion(w http.ResponseWriter, r *http.Request) {
	token := cookie(r)
	ok := false
	var expira time.Time
	if token != "" {
		ok, expira, _ = h.sesiones.Activa(r.Context(), token)
	}
	respuesta(w, ok, expira)
}
func (h *Handler) certificado(w http.ResponseWriter, r *http.Request) {
	if !h.postValido(r) {
		denegar(w)
		return
	}
	huella, cuenta, ok := h.certificadoValido(r)
	if !ok {
		denegar(w)
		return
	}
	token, expira, err := h.sesiones.Abrir(r.Context(), cuenta, huella)
	if err != nil {
		denegar(w)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: nombreCookie, Value: token, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: expira, MaxAge: int(time.Until(expira).Seconds())})
	respuesta(w, true, expira)
}
func (h *Handler) cerrar(w http.ResponseWriter, r *http.Request) {
	if !h.postValido(r) {
		denegar(w)
		return
	}
	if token := cookie(r); token != "" {
		if err := h.sesiones.Cerrar(r.Context(), token); err != nil {
			denegar(w)
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: nombreCookie, Value: "", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) certificadoValido(r *http.Request) (string, string, bool) {
	if h == nil || h.crl == nil || r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || len(r.TLS.VerifiedChains) == 0 {
		return "", "", false
	}
	ahora := h.agora().UTC()
	if ahora.Before(h.crl.ThisUpdate) || !ahora.Before(h.crl.NextUpdate) {
		return "", "", false
	}
	leaf := r.TLS.PeerCertificates[0]
	if !contieneCadenaVerificada(r.TLS.VerifiedChains, leaf) || !admiteCliente(leaf) || ahora.Before(leaf.NotBefore) || !ahora.Before(leaf.NotAfter) {
		return "", "", false
	}
	verificada := false
	for _, cadena := range r.TLS.VerifiedChains {
		if len(cadena) > 1 && h.crl.CheckSignatureFrom(cadena[1]) == nil {
			verificada = true
			break
		}
	}
	if !verificada {
		return "", "", false
	}
	for _, entrada := range h.crl.RevokedCertificateEntries {
		if leaf.SerialNumber.Cmp(entrada.SerialNumber) == 0 {
			return "", "", false
		}
	}
	suma := sha256.Sum256(leaf.Raw)
	huella := hex.EncodeToString(suma[:])
	cuenta, ok := h.cuentas[huella]
	return huella, cuenta, ok && cuenta != ""
}
func admiteCliente(c *x509.Certificate) bool {
	for _, uso := range c.ExtKeyUsage {
		if uso == x509.ExtKeyUsageClientAuth {
			return true
		}
	}
	return false
}
func contieneCadenaVerificada(cadenas [][]*x509.Certificate, leaf *x509.Certificate) bool {
	for _, cadena := range cadenas {
		if len(cadena) != 0 && string(cadena[0].Raw) == string(leaf.Raw) {
			return true
		}
	}
	return false
}
func (h *Handler) postValido(r *http.Request) bool {
	if h == nil || r.Host == "" || r.Header.Get("Sec-Fetch-Site") != "same-origin" || r.Header.Get("Sec-Fetch-Mode") != "same-origin" {
		return false
	}
	u, err := url.Parse(h.origen)
	if err != nil || u.Host != r.Host || r.Header.Get("Origin") != h.origen {
		return false
	}
	return true
}
func cookie(r *http.Request) string {
	c, err := r.Cookie(nombreCookie)
	if err != nil {
		return ""
	}
	return c.Value
}
func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Pragma", "no-cache")
		next.ServeHTTP(w, r)
	})
}
func denegar(w http.ResponseWriter) { http.Error(w, "acceso denegado", http.StatusForbidden) }
func respuesta(w http.ResponseWriter, autenticada bool, expira time.Time) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	d := struct {
		Autenticada bool    `json:"autenticada"`
		Expira      *string `json:"expira_en"`
	}{Autenticada: autenticada}
	if autenticada {
		valor := expira.UTC().Format(time.RFC3339)
		d.Expira = &valor
	}
	_ = json.NewEncoder(w).Encode(struct {
		Data any `json:"data"`
	}{d})
}
