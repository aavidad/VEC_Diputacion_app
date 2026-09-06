package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestContratacionHTTP2NormalizaSoloAnuncioDeRespuesta(t *testing.T) {
	for nombreConstructor, constructor := range map[string]func(config.Config, http.Handler) http.Handler{
		"interna":              NewHandlerInternoWithConfig,
		"integrada_desarrollo": NewHandlerWithConfig,
	} {
		t.Run(nombreConstructor, func(t *testing.T) {
			for _, caso := range []struct {
				nombre    string
				protocolo int
				ruta      string
				cabeceras http.Header
				trailer   http.Header
				estado    int
			}{
				{"firefox", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}}, nil, 200},
				{"sin anuncio", 2, "/api/vec/contratacion-temporal/cuadro/consultas", nil, nil, 200},
				{"HTTP1 no cambia", 1, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}}, nil, 400},
				{"otro modulo no cambia", 2, "/api/vec/bolsa/panel", http.Header{"Te": {"trailers"}}, nil, 400},
				{"otro valor", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"gzip"}}, nil, 400},
				{"lista", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers, gzip"}}, nil, 400},
				{"duplicado", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers", "trailers"}}, nil, 400},
				{"alias duplicado", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}, "te": {"trailers"}}, nil, 400},
				{"cookie prohibida", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}, "Cookie": {"sesion=forjada"}}, nil, 400},
				{"autorizacion prohibida", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}, "Authorization": {"forjada"}}, nil, 400},
				{"trailer anunciado", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}, "Trailer": {"X-Vec-Subject"}}, nil, 400},
				{"trailer efectivo", 2, "/api/vec/contratacion-temporal/cuadro/consultas", http.Header{"Te": {"trailers"}}, http.Header{"X-Vec-Subject": {"forjado"}}, 400},
			} {
				t.Run(caso.nombre, func(t *testing.T) {
					api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if contieneCabecera(r.Header, "TE") || contieneCabecera(r.Header, "Authorization") {
							w.WriteHeader(400)
							return
						}
						cuerpo, err := io.ReadAll(r.Body)
						if err != nil || string(cuerpo) != `{"consulta":"sintetica"}` {
							t.Fatalf("cuerpo alterado: %q, %v", cuerpo, err)
						}
						if len(r.Trailer) != 0 {
							t.Fatal("trailer llegó a aplicación")
						}
						w.WriteHeader(200)
					})
					r := peticionServidorPrueba(http.MethodPost, caso.ruta, strings.NewReader(`{"consulta":"sintetica"}`))
					r.ProtoMajor = caso.protocolo
					for nombre, valores := range caso.cabeceras {
						r.Header[nombre] = valores
					}
					r.Trailer = caso.trailer
					rec := httptest.NewRecorder()
					constructor(config.Config{}, api).ServeHTTP(rec, r)
					if rec.Code != caso.estado {
						t.Fatalf("estado=%d, esperado=%d", rec.Code, caso.estado)
					}
					if caso.cabeceras.Get("Te") == "trailers" && r.Header.Get("Te") != "trailers" {
						t.Fatal("cabecera original alterada")
					}
				})
			}
		})
	}
}
