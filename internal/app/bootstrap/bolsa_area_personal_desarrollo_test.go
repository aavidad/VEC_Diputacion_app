package bootstrap

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/config"
)

func TestAreaPersonalBolsaDemostracionContratoYDisponibilidadIdempotente(t *testing.T) {
	a, err := nuevaAreaPersonalBolsaDesarrollo("../../../data/demo/bolsa/v1.bolsas-demo.json")
	if err != nil {
		t.Fatal(err)
	}
	panel := httptest.NewRecorder()
	a.ServeHTTP(panel, httptest.NewRequest(http.MethodGet, rutaAreaPersonalBolsaDesarrollo, nil))
	if panel.Code != 200 {
		t.Fatalf("panel=%d", panel.Code)
	}
	var salida struct {
		Data struct {
			Meta struct {
				Presentacion bool `json:"presentacion"`
			}
			Sesion struct {
				Metodo string `json:"metodo"`
			}
			Disponibilidad struct {
				Disponible bool `json:"disponible"`
			} `json:"disponibilidad"`
		} `json:"data"`
	}
	if err := json.Unmarshal(panel.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if salida.Data.Meta.Presentacion || salida.Data.Sesion.Metodo != "demostración sin identidad de candidato" || !salida.Data.Disponibilidad.Disponible {
		t.Fatalf("contrato=%s", panel.Body.String())
	}
	cuerpo := []byte(`{"data":{"esquema":"vec.bolsa.area-personal.accion.v1","accion":"cambiar_disponibilidad","confirmacion":true,"payload":{"disponible":false}}}`)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		a.ServeHTTP(w, httptest.NewRequest(http.MethodPost, rutaDisponibilidadBolsaDesarrollo, bytes.NewReader(cuerpo)))
		if w.Code != 200 {
			t.Fatalf("post=%d cuerpo=%s", w.Code, w.Body.String())
		}
	}
	panel = httptest.NewRecorder()
	a.ServeHTTP(panel, httptest.NewRequest(http.MethodGet, rutaAreaPersonalBolsaDesarrollo, nil))
	if err := json.Unmarshal(panel.Body.Bytes(), &salida); err != nil {
		t.Fatal(err)
	}
	if salida.Data.Disponibilidad.Disponible {
		t.Fatal("la disponibilidad no se conservó en el proceso")
	}
}

func TestAreaPersonalBolsaSoloSeComponeConDatasetYEntraEnElGuardian(t *testing.T) {
	if rutas, err := nuevasRutasAreaPersonalBolsaDesarrollo(config.Config{}); err == nil || rutas != nil {
		t.Fatalf("dataset ausente: rutas=%v error=%v", rutas, err)
	}
	for _, ruta := range []string{rutaAreaPersonalBolsaDesarrollo, rutaDisponibilidadBolsaDesarrollo} {
		if !esRutaContratacionTemporalDesarrollo(httptest.NewRequest(http.MethodGet, ruta, nil)) {
			t.Fatalf("la ruta %s elude el guardián", ruta)
		}
	}
}
