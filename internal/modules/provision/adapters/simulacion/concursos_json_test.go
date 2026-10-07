package simulacion_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
)

func TestConcursosJSONEstricto(t *testing.T) {
	ejemplos, err := simulacion.Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	original, err := json.Marshal(ejemplos[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		nombre string
		mutar  func(string) string
	}{
		{"alias_puesto", func(s string) string {
			return strings.Replace(s, `"nivel_puesto":22`, `"nivel_puesto":22,"NIVEL_PUESTO":1`, 1)
		}},
		{"alias_regla", func(s string) string { return strings.Replace(s, `"id":"grado"`, `"ID":"grado"`, 1) }},
		{"repetida", func(s string) string {
			return strings.Replace(s, `"nivel_puesto":22`, `"nivel_puesto":22,"nivel_puesto":1`, 1)
		}},
		{"unknown", func(s string) string {
			return strings.Replace(s, `"nivel_puesto":22`, `"nivel_puesto":22,"nombre":"inventado"`, 1)
		}},
		{"bool_acreditado_ausente", func(s string) string { return strings.Replace(s, `,"acreditado":true`, "", 1) }},
		{"bool_relacionado_ausente", func(s string) string { return strings.Replace(s, `,"relacionado":true`, "", 1) }},
		{"bool_requisito_ausente", func(s string) string { return strings.Replace(s, `,"usado_requisito":false`, "", 1) }},
		{"bool_requisito_null", func(s string) string {
			return strings.Replace(s, `"usado_requisito":false`, `"usado_requisito":null`, 1)
		}},
		{"coeficiente_ausente", func(s string) string { return strings.Replace(s, `"coeficiente":"0",`, "", 1) }},
		{"concat", func(s string) string { return s + s }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			mal := caso.mutar(string(original))
			if mal == string(original) {
				t.Fatal("la mutación no alcanzó el campo")
			}
			var destino simulacion.Ejemplo
			if err := simulacion.Decodificar(strings.NewReader(mal), &destino); err == nil {
				t.Fatal("JSON ambiguo/incompleto aceptado")
			}
		})
	}
	var correcto simulacion.Ejemplo
	if err := simulacion.Decodificar(bytes.NewReader(original), &correcto); err != nil {
		t.Fatal(err)
	}
	resultado, err := application.Simular(correcto.Configuracion, correcto.Entrada)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Total == nil || resultado.Total.Micropuntos() != 28_386_027 {
		t.Fatalf("total sintético inesperado: %+v", resultado.Total)
	}
}
func TestConcursosEjemplosSinMutacionCompartida(t *testing.T) {
	a, err := simulacion.Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	a[0].Configuracion.Reglas[0].ID = "alterado"
	a[0].Entrada.Cursos[0].Tipo = "alterado"
	b, err := simulacion.Ejemplos()
	if err != nil {
		t.Fatal(err)
	}
	if b[0].Configuracion.Reglas[0].ID == "alterado" || b[0].Entrada.Cursos[0].Tipo == "alterado" {
		t.Fatal("estado global mutable")
	}
}
func TestConcursosJSONLimites(t *testing.T) {
	var destino simulacion.Ejemplo
	if err := simulacion.Decodificar(strings.NewReader(strings.Repeat(" ", int(simulacion.MaximoBytes)+1)), &destino); err == nil {
		t.Fatal("exceso tamaño aceptado")
	}
	if err := simulacion.Decodificar(strings.NewReader(strings.Repeat("[", 26)+"0"+strings.Repeat("]", 26)), &destino); err == nil {
		t.Fatal("exceso profundidad aceptado")
	}
}
