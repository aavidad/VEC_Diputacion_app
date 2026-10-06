package server

import (
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

// En HTTP/2 net/http entrega siempre un Body propio, también en un GET sin
// cuerpo; en HTTP/1.1 entrega http.NoBody. Tras la barrera de trailers, el
// manejador debe ver lo mismo en los dos protocolos: http.NoBody si no hay
// cuerpo y el cuerpo íntegro si lo hay. Prueba con un servidor TLS real.
func TestCuerpoVacioIgualEnHTTP1YHTTP2(t *testing.T) {
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.Body != http.NoBody || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
				http.Error(w, "cuerpo inesperado", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		cuerpo, err := io.ReadAll(r.Body)
		if err != nil || string(cuerpo) != `{"consulta":"sintetica"}` || r.Body == http.NoBody {
			http.Error(w, "cuerpo alterado", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})
	for nombre, constructor := range map[string]func(config.Config, http.Handler) http.Handler{
		"interna":              NewHandlerInternoWithConfig,
		"integrada_desarrollo": NewHandlerWithConfig,
	} {
		t.Run(nombre, func(t *testing.T) {
			servidor := httptest.NewUnstartedServer(constructor(config.Config{}, api))
			servidor.EnableHTTP2 = true
			servidor.StartTLS()
			defer servidor.Close()
			for _, http2 := range []bool{true, false} {
				cliente := servidor.Client()
				if !http2 {
					transporte := cliente.Transport.(*http.Transport).Clone()
					transporte.ForceAttemptHTTP2 = false
					transporte.TLSClientConfig = transporte.TLSClientConfig.Clone()
					transporte.TLSClientConfig.NextProtos = []string{"http/1.1"}
					transporte.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
					cliente = &http.Client{Transport: transporte}
				}
				for _, caso := range []struct {
					metodo, cuerpo string
					estado         int
				}{
					{http.MethodGet, "", http.StatusOK},
					{http.MethodPost, `{"consulta":"sintetica"}`, http.StatusCreated},
				} {
					var cuerpo io.Reader
					if caso.cuerpo != "" {
						cuerpo = strings.NewReader(caso.cuerpo)
					}
					peticion, err := http.NewRequest(caso.metodo, servidor.URL+"/api/vec/bolsa/llamamientos/plazo-respuesta", cuerpo)
					if err != nil {
						t.Fatal(err)
					}
					respuesta, err := cliente.Do(peticion)
					if err != nil {
						t.Fatal(err)
					}
					_, _ = io.Copy(io.Discard, respuesta.Body)
					_ = respuesta.Body.Close()
					if (respuesta.ProtoMajor == 2) != http2 || respuesta.StatusCode != caso.estado {
						t.Fatalf("http2=%v %s: protocolo=%d estado=%d, esperado %d", http2, caso.metodo, respuesta.ProtoMajor, respuesta.StatusCode, caso.estado)
					}
				}
			}
		})
	}
}
