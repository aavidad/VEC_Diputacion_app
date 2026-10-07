package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestSeleccionLocalExponeTresConfiguracionesYSoloEnsayo(t *testing.T) {
	w := httptest.NewRecorder()
	configurarSeleccionLocal(w)
	if w.Code != http.StatusOK {
		t.Fatalf("configuración: %d %s", w.Code, w.Body.String())
	}
	var catalogo struct {
		Ejemplos []simulacion.Ejemplo `json:"ejemplos"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &catalogo); err != nil || len(catalogo.Ejemplos) != 3 {
		t.Fatalf("configuración inesperada: %v %s", err, w.Body.String())
	}
	for _, ejemplo := range catalogo.Ejemplos {
		datos, err := json.Marshal(struct {
			EjemploRef    string               `json:"ejemplo_ref"`
			Configuracion domain.Configuracion `json:"configuracion"`
		}{ejemplo.Referencia, ejemplo.Configuracion})
		if err != nil {
			t.Fatal(err)
		}
		w = httptest.NewRecorder()
		simularSeleccionLocal(w, datos)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", ejemplo.Referencia, w.Code, w.Body.String())
		}
		var resultado domain.Resultado
		if err := json.Unmarshal(w.Body.Bytes(), &resultado); err != nil || resultado.Alcance != "ensayo_sintetico" || resultado.Modalidad != ejemplo.Configuracion.Modalidad {
			t.Fatalf("%s: respuesta no explicable: %v %s", ejemplo.Referencia, err, w.Body.String())
		}
	}
}

func TestSeleccionLocalRechazaHechosDeClienteYEjemploAjeno(t *testing.T) {
	for _, caso := range []struct {
		datos  string
		codigo int
	}{
		{`{"ejemplo_ref":"seleccion_oposicion_v1","configuracion":{},"solicitudes":[{"nombre":"Persona ajena"}]}`, http.StatusBadRequest},
		{`{"ejemplo_ref":"ajeno","configuracion":{"version":1}}`, http.StatusBadRequest},
	} {
		w := httptest.NewRecorder()
		simularSeleccionLocal(w, []byte(caso.datos))
		if w.Code != caso.codigo {
			t.Fatalf("%s: %d %s", caso.datos, w.Code, w.Body.String())
		}
	}
}
