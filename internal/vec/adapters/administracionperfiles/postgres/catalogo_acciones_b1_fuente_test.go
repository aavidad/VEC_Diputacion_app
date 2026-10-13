package postgres

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

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
		fuente.Entradas[0].ClaseControl != string(domain.ClaseControlPerfilOrdinario) ||
		!fuente.Entradas[0].VigenteHasta.After(time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)) {
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

func TestFuenteContactoC3ConservaCanonYNoConcedeClaro(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "..", "..", "deploy", "principal", "fuentes", "bolsa_origen_contacto_rrhh_v1.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var fuente fuentePaqueteCatalogoAccionesV2
	if err := json.Unmarshal(b, &fuente); err != nil {
		t.Fatal(err)
	}
	canon, h, err := canonEntradasFuenteV2(fuente.Entradas)
	if err != nil || len(fuente.Entradas) != 1 || string(canon) != fuente.EntradasCanon || h != fuente.HuellaSHA256 || fuente.Entradas[0].FuenteHuellaSHA256 != h {
		t.Fatal("fuente C3 divergente", err)
	}
	c := fuente.Entradas[0].Concesion
	if c.Accion != domain.AccionLecturaOrigenContactoBolsa || c.TipoRecurso != "participacion_bolsa" || len(c.CamposPermitidos) != 6 {
		t.Fatal("proyección C3 fuera de contrato")
	}
	for j, campo := range domain.CamposLecturaOrigenContactoBolsa() {
		if c.CamposPermitidos[j] != campo {
			t.Fatal("concedió una hoja ajena a la máscara")
		}
	}
}

func TestFuenteMeritosCarteraConservaCanonYAmbitoPropio(t *testing.T) {
	ruta := filepath.Join("..", "..", "..", "..", "..", "deploy", "principal", "fuentes", "meritos_cartera_propia_v1.json")
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var fuente fuentePaqueteCatalogoAccionesV2
	if err := json.Unmarshal(b, &fuente); err != nil || fuente.ModuloID != "meritos" || len(fuente.Entradas) != 3 {
		t.Fatal("fuente de cartera incompleta", err)
	}
	canon, huella, err := canonEntradasFuenteV2(fuente.Entradas)
	if err != nil || string(canon) != fuente.EntradasCanon || huella != fuente.HuellaSHA256 {
		t.Fatal("fuente de cartera divergente", err)
	}
	acciones := []string{"meritos.cartera.consultar_propia", "meritos.cartera.retirar_propio", "meritos.hecho.declarar"}
	for i, entrada := range fuente.Entradas {
		if entrada.FuenteHuellaSHA256 != huella || entrada.Concesion.Accion != acciones[i] ||
			entrada.Concesion.ModuloID != "meritos" || entrada.Concesion.GarantiaMinima != domain.AuthAssuranceHigh ||
			len(entrada.DimensionesAmbito) != 1 || entrada.DimensionesAmbito[0] != "persona_ref" ||
			len(entrada.Concesion.Obligaciones) != 1 || entrada.Concesion.Obligaciones[0] != "auditar" {
			t.Fatalf("concesión %d de cartera fuera de contrato", i)
		}
	}
}
