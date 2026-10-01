package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	d "vec-diputacion-granada/internal/modules/provision/domain"
)

func ejemploCLI(t *testing.T) []byte {
	t.Helper()
	p, err := simulacion.EjemploAdjudicacion()
	if err != nil {
		t.Fatal(err)
	}
	datos, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return datos
}
func TestCLIAdjudicacionConsumeJSONReal(t *testing.T) {
	datos := ejemploCLI(t)
	var salida, diagnostico bytes.Buffer
	if codigo := ejecutar(bytes.NewReader(datos), &salida, &diagnostico); codigo != 0 {
		t.Fatalf("codigo=%d error=%s", codigo, diagnostico.String())
	}
	var r d.ResultadoAdjudicacion
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Estado != "propuesta_simulada" || r.Alcance != "simulacion_sintetica_sin_efectos" || len(r.Asignaciones) != 2 || len(r.HuellaResultado) != 64 {
		t.Fatal(r)
	}
	if r.Asignaciones[0].PersonaRef != "persona:a" || r.Asignaciones[0].VacanteRef != "vacante:a" || r.Asignaciones[1].PersonaRef != "persona:b" || r.Asignaciones[1].VacanteRef != "vacante:b" {
		t.Fatal(r.Asignaciones)
	}
	var segunda bytes.Buffer
	if ejecutar(bytes.NewReader(datos), &segunda, io.Discard) != 0 || !bytes.Equal(salida.Bytes(), segunda.Bytes()) {
		t.Fatal("CLI no reproduce mismos bytes")
	}
}
func TestCLIAdjudicacionEntradaHostilSinFiltrarContenido(t *testing.T) {
	base := string(ejemploCLI(t))
	casos := map[string]string{
		"campo_desconocido":  strings.Replace(base, `"entrada": {`, `"entrada": {"dni": "dato-no-autorizado",`, 1),
		"clave_repetida":     strings.Replace(base, `"cerrado": true`, `"cerrado": true, "cerrado": false`, 1),
		"clave_mayuscula":    strings.Replace(base, `"sintetico": true`, `"Sintetico": true`, 1),
		"bool_null":          strings.Replace(base, `"sintetico": true`, `"sintetico": null`, 1),
		"bool_ausente":       strings.Replace(base, `"sintetico": true,`, "", 1),
		"puntos_float":       strings.Replace(base, `"total": "20000000"`, `"total": 20.0`, 1),
		"datos_concatenados": base + base,
		"documento_null":     "null",
		"exceso":             strings.Repeat(" ", 2<<20) + base,
	}
	for nombre, datos := range casos {
		t.Run(nombre, func(t *testing.T) {
			var salida, diagnostico bytes.Buffer
			if ejecutar(strings.NewReader(datos), &salida, &diagnostico) != 1 || salida.Len() != 0 {
				t.Fatal("entrada hostil aceptada")
			}
			var e d.Error
			if json.Unmarshal(diagnostico.Bytes(), &e) != nil || e.Codigo == "" || strings.Contains(diagnostico.String(), "dato-no-autorizado") {
				t.Fatal("diagnóstico no minimizado")
			}
		})
	}
}
func TestCLIAdjudicacionPendienteYMetodoNoSoportado(t *testing.T) {
	for _, caso := range []struct{ nombre, buscar, nuevo, estado string }{
		{"pendiente", `"cerrado": true`, `"cerrado": false`, "pendiente"},
		{"metodo", d.MetodoAdjudicacionEnsayo, "metodo:otro", "no_soportado"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			datos := strings.Replace(string(ejemploCLI(t)), caso.buscar, caso.nuevo, 1)
			var salida bytes.Buffer
			if ejecutar(strings.NewReader(datos), &salida, io.Discard) != 0 {
				t.Fatal("salida nominal tratada como fallo transporte")
			}
			var r d.ResultadoAdjudicacion
			if json.Unmarshal(salida.Bytes(), &r) != nil || r.Estado != caso.estado || len(r.Asignaciones) != 0 {
				t.Fatal(r)
			}
		})
	}
}

type falloEscrituraCLI struct{}

func (falloEscrituraCLI) Write([]byte) (int, error) { return 0, errors.New("escritura_fallida") }
func TestCLIAdjudicacionFalloEscritura(t *testing.T) {
	if ejecutar(bytes.NewReader(ejemploCLI(t)), falloEscrituraCLI{}, io.Discard) != 1 {
		t.Fatal("fallo de salida ignorado")
	}
}
