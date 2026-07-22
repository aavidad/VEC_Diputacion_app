package interna

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManejadorInternoRechazaEstadoTLSAusenteOIncoherente(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	llamadas := 0
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		llamadas++
		w.WriteHeader(http.StatusNoContent)
	}))
	if err != nil {
		t.Fatal(err)
	}
	pruebas := []struct {
		nombre string
		mutar  func(*http.Request)
	}{
		{"sin TLS", func(r *http.Request) { r.TLS = nil }},
		{"handshake incompleto", func(r *http.Request) { r.TLS.HandshakeComplete = false }},
		{"TLS 1.2", func(r *http.Request) { r.TLS.Version = tls.VersionTLS12 }},
		{"sin ALPN", func(r *http.Request) { r.TLS.NegotiatedProtocol = "" }},
		{"ALPN no mutuo", func(r *http.Request) { r.TLS.NegotiatedProtocolIsMutual = false }},
		{"reanudada", func(r *http.Request) { r.TLS.DidResume = true }},
		{"SNI ajeno", func(r *http.Request) { r.TLS.ServerName = "ajeno.test" }},
		{"cifrado incoherente", func(r *http.Request) { r.TLS.CipherSuite = tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 }},
		{"sin curva", func(r *http.Request) { r.TLS.CurveID = 0 }},
		{"sin certificado cliente", func(r *http.Request) { r.TLS.PeerCertificates = nil }},
		{"sin cadenas verificadas", func(r *http.Request) { r.TLS.VerifiedChains = nil }},
		{"HTTP2", func(r *http.Request) { r.ProtoMajor, r.ProtoMinor = 2, 0 }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, "/api/vec/prueba", nil)
			peticion.RemoteAddr = "127.0.0.2:50000"
			peticion.TLS = estadoTLSMutuoValidoPrueba(t, material)
			prueba.mutar(peticion)
			respuesta := httptest.NewRecorder()
			servidor.Handler.ServeHTTP(respuesta, peticion)
			if respuesta.Code != http.StatusBadRequest {
				t.Fatalf("estado incoherente = %d", respuesta.Code)
			}
		})
	}
	if llamadas != 0 {
		t.Fatalf("estado TLS invalido alcanzo API: %d", llamadas)
	}
}
