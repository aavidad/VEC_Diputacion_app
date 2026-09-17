package bootstrap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestBolsasRRHHDesarrolloExponeContratoCerradoYPaginaCandidatos(t *testing.T) {
	rutas, colecciones, err := nuevasRutasBolsasRRHHDesarrollo(config.Config{BolsaDemoPath: "../../../data/demo/bolsa/v1.bolsas-demo.json"})
	if err != nil || len(rutas) != 1 || len(colecciones) != 1 {
		t.Fatalf("rutas C20: exactas=%d colecciones=%d error=%v", len(rutas), len(colecciones), err)
	}
	lista := httptest.NewRecorder()
	rutas[0].Manejador.ServeHTTP(lista, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo, nil))
	if lista.Code != http.StatusOK || strings.Contains(strings.ToLower(lista.Body.String()), "correo") || strings.Contains(strings.ToLower(lista.Body.String()), "telefono") {
		t.Fatalf("lista C20: status=%d body=%s", lista.Code, lista.Body.String())
	}
	var salida struct {
		Data struct {
			Esquema string `json:"esquema"`
			Bolsas  []struct {
				Referencia string         `json:"bolsa_ref"`
				Estados    map[string]int `json:"por_estado"`
			} `json:"bolsas"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lista.Body.Bytes(), &salida); err != nil || salida.Data.Esquema != "vec.bolsa.rrhh.bolsas.v1" || len(salida.Data.Bolsas) != 12 || len(salida.Data.Bolsas[0].Estados) != 5 {
		t.Fatalf("contrato bolsas: %#v err=%v", salida, err)
	}
	candidatos := httptest.NewRecorder()
	colecciones[0].Manejador.ServeHTTP(candidatos, httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:demo:administrativo/candidatos?estado=disponible&limite=1", nil))
	if candidatos.Code != http.StatusOK {
		t.Fatalf("candidatos=%d body=%s", candidatos.Code, candidatos.Body.String())
	}
	var pagina struct {
		Data struct {
			Esquema    string `json:"esquema"`
			Candidatos []struct {
				Documento string `json:"documento_enmascarado"`
				Estado    string `json:"estado_clave"`
			} `json:"candidatos"`
			HayMas bool    `json:"hay_mas"`
			Cursor *string `json:"cursor_siguiente"`
		} `json:"data"`
	}
	if err := json.Unmarshal(candidatos.Body.Bytes(), &pagina); err != nil || pagina.Data.Esquema != "vec.bolsa.rrhh.candidatos.v1" || len(pagina.Data.Candidatos) != 1 || pagina.Data.Candidatos[0].Estado != "disponible" || !strings.HasPrefix(pagina.Data.Candidatos[0].Documento, "***") || !pagina.Data.HayMas || pagina.Data.Cursor == nil {
		t.Fatalf("contrato candidatos: %#v err=%v", pagina, err)
	}
	if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:demo:administrativo/candidatos", nil)) {
		t.Fatal("la colección C20 no pasa por el guardián mTLS")
	}
}

func TestBolsasRRHHDesarrolloRechazaConsultaNoCanonica(t *testing.T) {
	rutas, colecciones, err := nuevasRutasBolsasRRHHDesarrollo(config.Config{BolsaDemoPath: "../../../data/demo/bolsa/v1.bolsas-demo.json"})
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []*http.Request{
		httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"?limite=1", nil),
		httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:demo:administrativo/candidatos?estado=trabajando", nil),
		httptest.NewRequest(http.MethodGet, rutaBolsasRRHHDesarrollo+"/bolsa:demo:administrativo/candidatos?limite=101", nil),
	} {
		w := httptest.NewRecorder()
		if caso.URL.Path == rutaBolsasRRHHDesarrollo {
			rutas[0].Manejador.ServeHTTP(w, caso)
		} else {
			colecciones[0].Manejador.ServeHTTP(w, caso)
		}
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status=%d para %s", w.Code, caso.URL.String())
		}
	}
}
