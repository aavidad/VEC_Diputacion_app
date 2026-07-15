package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func plantillaPublicadaValida() PlantillaDocumento {
	fecha := time.Date(2026, time.July, 14, 10, 0, 0, 0, time.UTC)
	return PlantillaDocumento{
		ID:             "contrato_bolsa",
		Version:        3,
		ModuloID:       "bolsa",
		TipoDocumental: "contrato",
		Nombre:         "Contrato de personal de bolsa",
		Titulo:         "Contrato {{numero_contrato}}",
		Parrafos: []string{
			"Interviene {{nombre_persona}}.",
			"Observaciones: {{observaciones}}",
		},
		Campos: []CampoPlantillaDocumento{
			{Clave: "numero_contrato", Etiqueta: "Numero de contrato", Obligatorio: true},
			{Clave: "nombre_persona", Etiqueta: "Nombre de la persona", Obligatorio: true, Sensible: true},
			{Clave: "observaciones", Etiqueta: "Observaciones"},
		},
		Formatos:          []FormatoDocumento{FormatoDocumentoDOCX, FormatoDocumentoPDF},
		PermisoGenerar:    "bolsa.documentos.generar",
		GarantiaMinima:    AuthAssuranceSubstantial,
		Estado:            EstadoPlantillaPublicada,
		CreadaPor:         "rrhh-1",
		CreadaEn:          fecha.Add(-time.Hour),
		PublicadaPor:      "rrhh-2",
		PublicadaEn:       fecha,
		AprobacionRef:     "aprobacion-plantilla-3",
		MotivoPublicacion: "Aprobacion de la plantilla por Seleccion Externa",
	}
}

func TestPlantillaFusionaSoloCamposDeclaradosDeFormaLiteral(t *testing.T) {
	plantilla := plantillaPublicadaValida()
	contenido, err := plantilla.Fusionar(map[string]string{
		"numero_contrato": "2026/$1",
		"nombre_persona":  "Maria <Nunez> & familia",
		"observaciones":   "Incluye {{texto}} literal; no se vuelve a interpretar.",
	})
	if err != nil {
		t.Fatalf("Fusionar() error = %v", err)
	}
	if contenido.Titulo != "Contrato 2026/$1" {
		t.Fatalf("titulo fusionado = %q", contenido.Titulo)
	}
	if contenido.Parrafos[0] != "Interviene Maria <Nunez> & familia." {
		t.Fatalf("primer parrafo = %q", contenido.Parrafos[0])
	}
	if contenido.Parrafos[1] != "Observaciones: Incluye {{texto}} literal; no se vuelve a interpretar." {
		t.Fatalf("segundo parrafo = %q", contenido.Parrafos[1])
	}
}

func TestPlantillaRechazaCamposFaltantesYAdicionales(t *testing.T) {
	plantilla := plantillaPublicadaValida()
	_, err := plantilla.Fusionar(map[string]string{"numero_contrato": "C-1"})
	if !errors.Is(err, ErrCampoPlantillaFaltante) {
		t.Fatalf("campo faltante: error = %v", err)
	}

	_, err = plantilla.Fusionar(map[string]string{
		"numero_contrato":  "C-1",
		"nombre_persona":   "Persona",
		"observaciones":    "",
		"dni_no_declarado": "00000000T",
	})
	if !errors.Is(err, ErrCampoPlantillaDesconocido) {
		t.Fatalf("campo adicional: error = %v", err)
	}
}

func TestPlantillaExigeVersionPublicadaYMarcadoresCoherentes(t *testing.T) {
	plantilla := plantillaPublicadaValida()
	plantilla.Estado = EstadoPlantillaBorrador
	plantilla.PublicadaPor = ""
	plantilla.PublicadaEn = time.Time{}
	plantilla.AprobacionRef = ""
	plantilla.MotivoPublicacion = ""
	if _, err := plantilla.Fusionar(map[string]string{}); !errors.Is(err, ErrPlantillaNoPublicada) {
		t.Fatalf("borrador: error = %v", err)
	}

	plantilla = plantillaPublicadaValida()
	plantilla.Parrafos[0] = "Interviene {{campo_sin_declarar}}."
	if err := plantilla.Validar(); !errors.Is(err, ErrCampoPlantillaDesconocido) {
		t.Fatalf("marcador desconocido: error = %v", err)
	}

	plantilla = plantillaPublicadaValida()
	plantilla.Parrafos[0] = "Interviene {{nombre_persona."
	if err := plantilla.Validar(); !errors.Is(err, ErrPlantillaDocumentoInvalida) {
		t.Fatalf("marcador mal cerrado: error = %v", err)
	}
}

func TestGarantiaTieneOrdenExplicito(t *testing.T) {
	if !AuthAssuranceHigh.Cumple(AuthAssuranceLow) ||
		!AuthAssuranceHigh.Cumple(AuthAssuranceSubstantial) ||
		AuthAssuranceLow.Cumple(AuthAssuranceHigh) ||
		AuthAssurance("inventada").Cumple(AuthAssuranceLow) {
		t.Fatal("la comparacion de niveles de garantia no aplica el orden bajo < sustancial < alto")
	}
}

func TestPublicarPlantillaExigeSegregacionYAprobacion(t *testing.T) {
	borrador := plantillaPublicadaValida()
	borrador.Estado = EstadoPlantillaBorrador
	borrador.PublicadaPor = ""
	borrador.PublicadaEn = time.Time{}
	borrador.AprobacionRef = ""
	borrador.MotivoPublicacion = ""
	fecha := borrador.CreadaEn.Add(2 * time.Hour)
	if _, err := borrador.Publicar(borrador.CreadaPor, "aprobacion-1", "publicar", fecha); !errors.Is(err, ErrPlantillaDocumentoInvalida) {
		t.Fatalf("autopublicacion: error = %v", err)
	}
	publicada, err := borrador.Publicar("rrhh-publicador", "aprobacion-1", "Aprobada por el flujo", fecha)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	if publicada.Estado != EstadoPlantillaPublicada || publicada.AprobacionRef != "aprobacion-1" {
		t.Fatalf("plantilla publicada inesperada: %+v", publicada)
	}
	huellaBorrador, err := borrador.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella borrador: %v", err)
	}
	huellaPublicada, err := publicada.HuellaSHA256()
	if err != nil {
		t.Fatalf("huella publicada: %v", err)
	}
	if huellaBorrador == huellaPublicada {
		t.Fatal("la publicacion no cambio la huella versionada")
	}
}

func TestPlantillaLimitaExpansionTotalAntesDeRenderizar(t *testing.T) {
	plantilla := plantillaPublicadaValida()
	plantilla.Titulo = "Certificado"
	plantilla.Parrafos = []string{strings.Repeat("{{nombre_persona}}", 80)}
	plantilla.Campos = []CampoPlantillaDocumento{
		{Clave: "nombre_persona", Etiqueta: "Nombre", Obligatorio: true, Sensible: true},
	}
	_, err := plantilla.Fusionar(map[string]string{
		"nombre_persona": strings.Repeat("x", maximoCaracteresDato),
	})
	if !errors.Is(err, ErrContenidoFusionadoExcesivo) {
		t.Fatalf("expansion excesiva: error = %v", err)
	}
}

func TestPlantillaRechazaClavesNoCanonicas(t *testing.T) {
	plantilla := plantillaPublicadaValida()
	plantilla.Campos[0].Clave = " numero_contrato "
	if err := plantilla.Validar(); !errors.Is(err, ErrCampoPlantillaInvalido) {
		t.Fatalf("clave con espacios: error = %v", err)
	}

	plantilla = plantillaPublicadaValida()
	plantilla.ID = " contrato_bolsa "
	if err := plantilla.Validar(); !errors.Is(err, ErrPlantillaDocumentoInvalida) {
		t.Fatalf("id con espacios: error = %v", err)
	}

	plantilla = plantillaPublicadaValida()
	plantilla.PermisoGenerar = "bolsa.documentos.*"
	if err := plantilla.Validar(); !errors.Is(err, ErrPlantillaDocumentoInvalida) {
		t.Fatalf("permiso positivo con comodin: error = %v", err)
	}
}
