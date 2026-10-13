package postgres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFuenteAlegacionesBolsaCanonYTresAcciones(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "..", "..", "deploy", "principal", "fuentes", "bolsa_alegaciones_propias_v1.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var fuente fuentePaqueteCatalogoAccionesV2
	if err := json.Unmarshal(b, &fuente); err != nil || fuente.ModuloID != "bolsa" || len(fuente.Entradas) != 3 {
		t.Fatal("fuente incompleta", err)
	}
	canon, h, err := canonEntradasFuenteV2(fuente.Entradas)
	if err != nil || string(canon) != fuente.EntradasCanon || h != fuente.HuellaSHA256 {
		t.Fatal("fuente divergente", err)
	}
	acciones := []string{"bolsa.alegacion.propias.consultar", "bolsa.inscripcion.alegacion.presentar", "bolsa.inscripcion.alegacion.rrhh.consultar"}
	for i, e := range fuente.Entradas {
		if e.FuenteHuellaSHA256 != h || e.Concesion.Accion != acciones[i] || e.Concesion.ModuloID != "bolsa" {
			t.Fatalf("entrada %d fuera de contrato", i)
		}
	}
}
