package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func TestAdjudicacionLocalConsumeValoracionesServidor(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	p, err := simulacion.EjemploAdjudicacion()
	if err != nil {
		t.Fatal(err)
	}
	c, err := leerCatalogoEnsayos()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(struct {
		EjemploRef    string                           `json:"ejemplo_ref"`
		Configuracion domain.ConfiguracionAdjudicacion `json:"configuracion"`
	}{c.AdjudicacionRef, p.Configuracion})
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := application.SimularAdjudicacion(p.Configuracion, p.Entrada)
	if err != nil {
		t.Fatal(err)
	}
	bytesEsperados, err := json.Marshal(esperado)
	if err != nil {
		t.Fatal(err)
	}
	w := request(h, http.MethodPost, rutaSimularAdjudicacion, string(b), nil)
	if w.Code != http.StatusOK || !bytes.Equal(bytes.TrimSpace(w.Body.Bytes()), bytesEsperados) {
		t.Fatalf("ensayo y motor difieren: %d %s", w.Code, w.Body.String())
	}
	if len(esperado.Asignaciones) != 2 {
		t.Fatal("fixture sin propuesta global")
	}
	for _, campo := range []string{`,"entrada":{}`, `,"persona_ref":"otra"`, `,"valoraciones":[]`} {
		body := strings.TrimSuffix(string(b), "}") + campo + "}"
		w := request(h, http.MethodPost, rutaSimularAdjudicacion, body, nil)
		if w.Code != http.StatusBadRequest {
			t.Fatal("hechos cliente admitidos", w.Code)
		}
	}
}

func TestAdjudicacionLocalFrontera(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	if w := request(h, http.MethodGet, rutaAdjudicacionesLocales, "", nil); w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(h, http.MethodGet, rutaAdjudicacionesLocales+"?persona=ajena", "", nil); w.Code != http.StatusBadRequest {
		t.Fatal(w.Code)
	}
	if w := request(h, http.MethodPost, rutaSimularAdjudicacion, "{}", func(r *http.Request) { r.Header.Del("Origin") }); w.Code != http.StatusForbidden {
		t.Fatal(w.Code)
	}
	if w := request(h, http.MethodGet, rutaSimularAdjudicacion, "", nil); w.Code != http.StatusMethodNotAllowed {
		t.Fatal(w.Code)
	}
}
