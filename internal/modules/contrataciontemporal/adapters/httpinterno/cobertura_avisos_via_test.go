package httpinterno

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
)

func TestProyeccionAvisosViaCobertura(t *testing.T) {
	if proyectarAvisosViaCobertura(nil) != nil {
		t.Fatal("sin avisos no se añade el campo")
	}
	salida := proyectarAvisosViaCobertura(&application.ResultadoAvisosViaCobertura{
		Estado: application.EstadoAvisosEvaluados, EvaluadaEn: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC),
		Avisos: []application.AvisoViaCobertura{
			{Clave: application.AvisoBolsaAgotadaProvisionalmente, Integrantes: 3, Reglas: []application.ProcedenciaReglaAviso{{Clave: "b26.agotamiento", Origen: "reglamento", Articulo: "arts. 3.3 y 7.c"}}},
			{Clave: application.AvisoPropuestaOfertaSAE, DuracionMaximaMeses: 9, FinMaximo: "2027-07-01", FinPrevisto: "2027-03-31"},
		},
	})
	material, err := json.Marshal(salida)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(material)
	for _, esperado := range []string{`"disponibles":0`, `"umbral":0`, `"excede_duracion":false`, `"articulo":"arts. 3.3 y 7.c"`, `"reglas":[]`, `"esquema":"vec.contratacion-temporal.avisos-via-cobertura.v1"`} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("falta %s en %s", esperado, texto)
		}
	}
}
