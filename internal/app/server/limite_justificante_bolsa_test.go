package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

func TestJustificanteBolsaTransportaPDFMayorDeDosMiBSinAmpliarOtrasRutas(t *testing.T) {
	for _, ruta := range []string{"/api/vec/bolsa/mi-bolsa/justificantes", "/api/vec/bolsa/mi-bolsa/respuestas"} {
		t.Run(ruta, func(t *testing.T) {
			contenido := bytes.Repeat([]byte("p"), 3<<20)
			llamadas := 0
			api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				llamadas++
				b, err := io.ReadAll(r.Body)
				if err != nil || !bytes.Equal(b, contenido) {
					t.Errorf("cuerpo alterado o truncado: %v", err)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			h := prohibirCookiesYAutorizacionProxyConLimite(limitRequestBody(api, config.DefaultMaxRequestBodyBytes), config.DefaultMaxRequestBodyBytes)
			r := httptest.NewRequest(http.MethodPost, "https://localhost:18445"+ruta, bytes.NewReader(contenido))
			r.ProtoMajor = 2
			r.Header.Set("Origin", "https://localhost:18445")
			r.Header.Set("Sec-Fetch-Site", "same-origin")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if ruta == "/api/vec/bolsa/mi-bolsa/justificantes" {
				if llamadas != 1 || w.Code != http.StatusNoContent {
					t.Fatalf("PDF válido bloqueado por transporte: %d/%d", llamadas, w.Code)
				}
			} else if llamadas != 0 || w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("se amplió otra ruta: %d/%d", llamadas, w.Code)
			}
		})
	}
}
