package bootstrap

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/fichero"
	"vec-diputacion-granada/internal/vec/reglas"
)

const rutaCircuitoFirmaRRHHV2Prueba = "../../../data/demo/reglas/ct_circuito_firma.rrhh.v2.json"

func resolverCircuitoFirmaR5Prueba(t *testing.T, ruta string) *reglas.Resolutor {
	t.Helper()
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resolutor, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta,
		CatalogoID: reglas.CatalogoCircuitoFirmaCT, ModuloID: reglas.ModuloContratacionTemporal,
		Reloj: relojFijoAltaContratacionTemporalDesarrollo{ahora: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resolutor
}

func TestFuenteCircuitoFirmaR5ConservaPoliticaAlternativasYProcedencia(t *testing.T) {
	resolutor := resolverCircuitoFirmaR5Prueba(t, rutaCircuitoFirmaRRHHV2Prueba)
	origen, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := (fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolutor}).CircuitoFirma(t.Context())
	if err != nil || obtenido.CatalogoRef != "vec.contratacion_temporal.circuito_firma:2" ||
		obtenido.HuellaCatalogo != origen.HuellaCatalogo || obtenido.PermiteMismaPersonaEnPasos ||
		len(obtenido.Documentos) != 2 || len(obtenido.Documentos[1].Pasos) != 2 {
		t.Fatalf("procedencia o política R5 perdida: %+v, %v", obtenido, err)
	}
	paso := obtenido.Documentos[1].Pasos[0]
	if paso.PerfilRef != "perfil:ct:jefatura_servicio_rrhh" || len(paso.PerfilesAlternativos) != 1 ||
		paso.PerfilesAlternativos[0] != "perfil:ct:direccion_rrhh" ||
		paso.Referencia != origen.Documentos[1].Pasos[0].Referencia || obtenido.Documentos[1].Validar() != nil {
		t.Fatalf("alternativa o procedencia del paso perdida: %+v", paso)
	}
}

func TestFuenteCircuitoFirmaR5SoloPermiteMismaPersonaSiCatalogoLoDeclara(t *testing.T) {
	contenido, err := os.ReadFile(rutaCircuitoFirmaRRHHV2Prueba)
	if err != nil {
		t.Fatal(err)
	}
	declaracion := []byte(`"misma_persona_en_dos_pasos": "false"`)
	if bytes.Count(contenido, declaracion) != 3 {
		t.Fatal("el catálogo debe declarar la política en cada paso")
	}
	ruta := filepath.Join(t.TempDir(), "circuito.json")
	if err := os.WriteFile(ruta, bytes.ReplaceAll(contenido, declaracion,
		[]byte(`"misma_persona_en_dos_pasos": "true"`)), 0600); err != nil {
		t.Fatal(err)
	}
	resolutor := resolverCircuitoFirmaR5Prueba(t, ruta)
	origen, err := resolutor.CircuitoFirma(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	obtenido, err := (fuenteCircuitoFirmaReglasDesarrollo{resolutor: resolutor}).CircuitoFirma(t.Context())
	if err != nil || !origen.PermiteMismaPersonaEnPasos || !obtenido.PermiteMismaPersonaEnPasos ||
		obtenido.CatalogoRef != "vec.contratacion_temporal.circuito_firma:2" || obtenido.HuellaCatalogo != origen.HuellaCatalogo {
		t.Fatalf("política publicada no conservada: %+v, %v", obtenido, err)
	}
}
