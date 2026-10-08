package postgres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestFuenteB1ConservaDescriptorCanónicoAdmitible(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "..", "..", "deploy", "principal", "fuentes", "bolsa_carga_convoca_b1_v1.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var fuente fuentePaqueteCatalogoAccionesV2
	if err := json.Unmarshal(b, &fuente); err != nil || fuente.ModuloID != "bolsa" || len(fuente.Entradas) != 1 {
		t.Fatal("fuente B1 incompleta")
	}
	canon, huella, err := canonEntradasFuenteV2(fuente.Entradas)
	if err != nil || string(canon) != fuente.EntradasCanon || huella != fuente.HuellaSHA256 ||
		fuente.Entradas[0].FuenteHuellaSHA256 != huella ||
		fuente.Entradas[0].ClaseControl != string(domain.ClaseControlPerfilOrdinario) {
		t.Fatal("descriptor B1 no coincide con la fuente versionada")
	}
	concesion := fuente.Entradas[0].Concesion
	if concesion.Accion != "bolsa.carga_convoca.confirmar" || concesion.ModuloID != "bolsa" ||
		concesion.TipoRecurso != "carga_convoca" || concesion.GarantiaMinima != domain.AuthAssuranceHigh ||
		len(concesion.Finalidades) != 1 || concesion.Finalidades[0] != "carga_bolsa_convoca" ||
		len(concesion.CamposPermitidos) != 0 || len(concesion.Obligaciones) != 0 {
		t.Fatal("fuente B1 amplía la capacidad admitida")
	}
}
