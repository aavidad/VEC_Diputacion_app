package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	textos "vec-diputacion-granada/web"
)

func TestBolsasPublicasErroresUsanCatalogosReales(t *testing.T) {
	catalogo, err := textos.CatalogoBolsasPublicas()
	if err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre, metodo, ruta, codigo string
		fuente                       fuentePublicaPrueba
		estado                       int
	}{
		{"metodo", http.MethodPost, RutaBolsasPublicas, "metodo_no_permitido", fuentePruebaConPosiciones(1), http.StatusMethodNotAllowed},
		{"ruta", http.MethodGet, RutaBolsasPublicas + "/%2f/lista", "ruta_invalida", fuentePruebaConPosiciones(1), http.StatusBadRequest},
		{"consulta", http.MethodGet, RutaBolsasPublicas + "?desconocido=1", "consulta_invalida", fuentePruebaConPosiciones(1), http.StatusBadRequest},
		{"recurso", http.MethodGet, RutaBolsasPublicas + "/inexistente", "recurso_no_encontrado", fuentePruebaConPosiciones(1), http.StatusNotFound},
		{"bolsa", http.MethodGet, RutaBolsasPublicas + "/bolsa:inexistente/lista", "bolsa_no_encontrada", fuentePruebaConPosiciones(1), http.StatusNotFound},
		{"fuente", http.MethodGet, RutaBolsasPublicas, "servicio_no_disponible", fuentePublicaPrueba{err: ErrFuenteBolsasPublicasRequerida}, http.StatusServiceUnavailable},
		{"timeout", http.MethodGet, RutaBolsasPublicas, "tiempo_operacion_agotado", fuentePublicaPrueba{err: context.DeadlineExceeded}, http.StatusGatewayTimeout},
		{"cancelacion", http.MethodGet, RutaBolsasPublicas, "peticion_cancelada", fuentePublicaPrueba{err: context.Canceled}, http.StatusRequestTimeout},
		{"datos", http.MethodGet, RutaBolsasPublicas, "error_interno", fuentePublicaPrueba{bolsas: []BolsaPublica{{BolsaRef: "invalida"}}}, http.StatusInternalServerError},
	}
	for _, idioma := range catalogo.Locales() {
		for _, caso := range casos {
			t.Run(idioma+"/"+caso.nombre, func(t *testing.T) {
				h, err := NuevoManejadorBolsasPublicas(caso.fuente)
				if err != nil {
					t.Fatal(err)
				}
				r := httptest.NewRequest(caso.metodo, caso.ruta, nil)
				r.Header.Set("Accept-Language", idioma+"-GB,"+idioma+";q=0.8")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				var salida respuestaError
				if err := json.Unmarshal(w.Body.Bytes(), &salida); err != nil {
					t.Fatal(err)
				}
				esperado, ok := catalogo.Message(idioma, caso.codigo)
				if !ok || esperado == "" || salida.Error.Mensaje != esperado || salida.Error.Codigo != caso.codigo || w.Code != caso.estado || w.Header().Get("Content-Language") != idioma {
					t.Fatalf("error sin localizar: %d %s %v", w.Code, w.Body.String(), w.Header())
				}
				if w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Set-Cookie") != "" {
					t.Fatal("cabeceras publicas invalidas")
				}
			})
		}
	}
}

func TestBolsasPublicasIdiomaDesconocidoYHead(t *testing.T) {
	h, err := NuevoManejadorBolsasPublicas(fuentePruebaConPosiciones(1))
	if err != nil {
		t.Fatal(err)
	}
	catalogo, err := textos.CatalogoBolsasPublicas()
	if err != nil {
		t.Fatal(err)
	}
	for _, preferencia := range []string{"", "xx-XX", "%%%"} {
		r := httptest.NewRequest(http.MethodHead, RutaBolsasPublicas+"?invalido=1", nil)
		r.Header.Set("Accept-Language", preferencia)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || w.Body.Len() != 0 || w.Header().Get("Content-Language") != catalogo.DefaultLocale() {
			t.Fatalf("HEAD/fallback: %d %s %v", w.Code, w.Body.String(), w.Header())
		}
	}
}
