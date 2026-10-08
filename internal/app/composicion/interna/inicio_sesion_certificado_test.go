package interna

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type extractorInicioSesionPrueba struct{ llamadas int }

func (e *extractorInicioSesionPrueba) ExtraerAsercionProtegida(*http.Request) ([]byte, error) {
	e.llamadas++
	return []byte("asercion-protegida-prueba"), nil
}

type iniciadorSesionPrueba struct {
	llamadas int
	err      error
}

func (i *iniciadorSesionPrueba) IniciarPresentacionYVincular(ctx context.Context, asercion []byte) (context.Context, error) {
	i.llamadas++
	if string(asercion) != "asercion-protegida-prueba" || i.err != nil {
		return nil, i.err
	}
	return ctx, nil
}

func peticionInicioSesionPrueba() *http.Request {
	r := httptest.NewRequest(http.MethodPost, rutaInicioSesionCertificado, nil)
	r.Host = "servidor.interna.test:8443"
	r.Header.Set("Origin", "https://servidor.interna.test:8443")
	certificado, ca := &x509.Certificate{}, &x509.Certificate{}
	r.TLS = &tls.ConnectionState{HandshakeComplete: true, Version: tls.VersionTLS13,
		PeerCertificates: []*x509.Certificate{certificado},
		VerifiedChains:   [][]*x509.Certificate{{certificado, ca}}}
	return r
}

func TestInicioSesionCertificadoSoloPostVacioConOrigenGobernado(t *testing.T) {
	extractor, iniciador := &extractorInicioSesionPrueba{}, &iniciadorSesionPrueba{}
	h, err := nuevoManejadorInicioSesionCertificado(extractor, iniciador,
		"https://servidor.interna.test:8443", "servidor.interna.test")
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, peticionInicioSesionPrueba())
	if w.Code != http.StatusNoContent || w.Body.Len() != 0 ||
		w.Header().Get("Cache-Control") != "no-store" || extractor.llamadas != 1 || iniciador.llamadas != 1 {
		t.Fatalf("START exacto: estado=%d extractor=%d inicio=%d", w.Code, extractor.llamadas, iniciador.llamadas)
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(*http.Request)
		estado int
	}{
		{"otro origen", func(r *http.Request) { r.Header.Set("Origin", "https://otro.interna.test:8443") }, http.StatusForbidden},
		{"origen repetido", func(r *http.Request) { r.Header.Add("Origin", "https://servidor.interna.test:8443") }, http.StatusForbidden},
		{"sin origen", func(r *http.Request) { r.Header.Del("Origin") }, http.StatusForbidden},
		{"host distinto", func(r *http.Request) { r.Host = "otro.interna.test:8443" }, http.StatusForbidden},
		{"sitio cruzado", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }, http.StatusForbidden},
		{"método GET", func(r *http.Request) { r.Method = http.MethodGet }, http.StatusMethodNotAllowed},
		{"query", func(r *http.Request) { r.URL.RawQuery = "modo=alta" }, http.StatusNotFound},
		{"identidad ambiental", func(r *http.Request) { r.Header.Set("Authorization", "Bearer hostil") }, http.StatusForbidden},
		{"transferencia", func(r *http.Request) { r.TransferEncoding = []string{"chunked"} }, http.StatusBadRequest},
		{"cuerpo", func(r *http.Request) {
			r.Body = http.NoBody
			r.ContentLength = 1
		}, http.StatusBadRequest},
		{"sin TLS", func(r *http.Request) { r.TLS = nil }, http.StatusUnauthorized},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e, i := &extractorInicioSesionPrueba{}, &iniciadorSesionPrueba{}
			h, _ := nuevoManejadorInicioSesionCertificado(e, i,
				"https://servidor.interna.test:8443", "servidor.interna.test")
			r := peticionInicioSesionPrueba()
			caso.mutar(r)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != caso.estado || e.llamadas != 0 || i.llamadas != 0 {
				t.Fatalf("rechazo temprano: estado=%d extractor=%d inicio=%d", w.Code, e.llamadas, i.llamadas)
			}
		})
	}
	iniciador.err = errors.New("registro indisponible")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, peticionInicioSesionPrueba())
	if w.Code != http.StatusServiceUnavailable || w.Body.Len() != 0 ||
		extractor.llamadas != 2 || iniciador.llamadas != 2 {
		t.Fatal("fallo de registro aparentó inicio", w.Code)
	}
	for _, origen := range []string{"http://servidor.interna.test:8443", "https://otro.interna.test:8443",
		"https://servidor.interna.test:8443/", "https://servidor.interna.test:8443?modo=alta",
		" https://servidor.interna.test:8443 "} {
		if _, err := nuevoManejadorInicioSesionCertificado(extractor, iniciador, origen, "servidor.interna.test"); err == nil {
			t.Fatal("origen de configuración inválido admitido", origen)
		}
	}
}
