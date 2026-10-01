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

func solicitudProcesoPrueba(t *testing.T) string {
	t.Helper()
	p, err := simulacion.EjemploProceso()
	if err != nil {
		t.Fatal(err)
	}
	proyeccion, err := leerPresentacionProceso()
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(struct {
		EjemploRef    string                        `json:"ejemplo_ref"`
		Configuracion domain.Configuracion          `json:"configuracion"`
		Preferencias  []domain.PreferenciaProvision `json:"preferencias"`
	}{proyeccion.EjemploRef, p.Proceso.Configuracion, p.Solicitud.Preferencias})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestProcesoLocalUsaMotorYConservaPreferencias(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	w := request(h, http.MethodGet, rutaProcesosLocales, "", nil)
	if w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, campo := range []string{`"empleado_ref"`, `"instantanea"`, `"entrada"`, `"condicion_interna"`} {
		if bytes.Contains(w.Body.Bytes(), []byte(campo)) {
			t.Fatalf("proyección de oferta publica hechos: %s", campo)
		}
	}
	w = request(h, http.MethodPost, rutaSimulacionProceso, solicitudProcesoPrueba(t), nil)
	p, err := simulacion.EjemploProceso()
	if err != nil {
		t.Fatal(err)
	}
	esperado, err := application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(esperado)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || !bytes.Equal(bytes.TrimSpace(w.Body.Bytes()), b) {
		t.Fatalf("HTTP y aplicación difieren: %d %s", w.Code, w.Body.String())
	}
	repetida := request(h, http.MethodPost, rutaSimulacionProceso, solicitudProcesoPrueba(t), nil)
	if !bytes.Equal(w.Body.Bytes(), repetida.Body.Bytes()) {
		t.Fatal("ensayo idéntico cambió la salida")
	}
	if w.Header().Get("Set-Cookie") != "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("ensayo crea sesión o permite caché")
	}
}

func TestProcesoLocalCierraHechosYPreferenciasInvalidas(t *testing.T) {
	h := nuevoHandler(hostPrueba, nil)
	base := solicitudProcesoPrueba(t)
	for _, caso := range []struct {
		nombre, cuerpo string
		estado         int
	}{
		{"hechos_cliente", strings.TrimSuffix(base, "}") + `,"solicitud":{}}`, 400},
		{"ejemplo_ajeno", strings.Replace(base, "provision_proceso_sintetico_v1", "ajeno", 1), 400},
		{"puesto_ajeno", strings.Replace(base, "puesto:sintetico:2", "puesto:ajeno", 1), 422},
		{"preferencia_duplicada", strings.Replace(base, "puesto:sintetico:2", "puesto:sintetico:1", 1), 422},
		{"orden_duplicado", strings.Replace(base, `"orden":2`, `"orden":1`, 1), 422},
		{"convocatoria_ajena", strings.Replace(base, "convocatoria:sintetica:concursos:v1", "convocatoria:ajena", 1), 422},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			w := request(h, http.MethodPost, rutaSimulacionProceso, caso.cuerpo, nil)
			if w.Code != caso.estado {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
	for _, ruta := range []string{rutaProcesosLocales, rutaSimulacionProceso} {
		w := request(h, http.MethodGet, ruta+"?empleado_ref=otro", "", nil)
		if w.Code != http.StatusBadRequest {
			t.Fatal("consulta extra no cerrada", w.Code)
		}
	}
}
