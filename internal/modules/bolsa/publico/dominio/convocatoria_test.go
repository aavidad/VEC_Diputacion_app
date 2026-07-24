package dominio

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func convocatoriaPublicaValidaPrueba() Convocatoria {
	publicada := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	return Convocatoria{
		Version: "v1", Estado: EstadoConvocatoriaInscripcion,
		HuellaSHA256: strings.Repeat("a", 64),
		DatosPublicos: &DatosPublicosConvocatoria{
			IdentificadorPublico: "auxiliares-2026", Tipo: "bolsa_temporal",
			CatalogoCategorias: ReferenciaCatalogoCategorias{
				CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
				CatalogoHuellaSHA256:           strings.Repeat("b", 64),
				CatalogoHuellaProyeccionSHA256: strings.Repeat("c", 64),
			},
			Categorias: []string{"auxiliar-administrativo"},
			Titulo:     "Bolsa temporal de auxiliares", Resumen: "Resumen público.",
			Descripcion: "Descripción pública.", PublicadaEn: publicada, ActualizadaEn: publicada,
			Plazos: []PlazoConvocatoria{{
				Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción",
				Descripcion: "Presentación de solicitudes.", AbreEn: publicada,
				CierraEn: publicada.Add(24 * time.Hour),
			}},
			Documentos: []DocumentoConvocatoria{{
				Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
				Descripcion: "Bases oficiales.", Formato: "pdf",
				URL: "/bolsa/documentos/bases.pdf", PublicadoEn: publicada,
			}},
		},
	}
}

func TestConvocatoriaPublicaNominalNoContieneIdentidadInterna(t *testing.T) {
	tipo := reflect.TypeOf(Convocatoria{})
	for _, prohibido := range []string{"ID", "ReferenciaAgregado", "Expediente", "Actor"} {
		if _, existe := tipo.FieldByName(prohibido); existe {
			t.Fatalf("el modelo público contiene %s", prohibido)
		}
	}
	convocatoria := convocatoriaPublicaValidaPrueba()
	if err := convocatoria.ValidarPublicacion(); err != nil {
		t.Fatalf("fixture válido rechazado: %v", err)
	}
}

func TestConvocatoriaPublicaRechazaMaterialNoCanonico(t *testing.T) {
	pruebas := map[string]func(*Convocatoria){
		"huella":              func(c *Convocatoria) { c.HuellaSHA256 = strings.Repeat("A", 64) },
		"url externa":         func(c *Convocatoria) { c.DatosPublicos.Documentos[0].URL = "https://example.invalid/bases.pdf" },
		"sin plazo operativo": func(c *Convocatoria) { c.DatosPublicos.Plazos = nil },
		"texto con control":   func(c *Convocatoria) { c.DatosPublicos.Titulo = "Título\u0000" },
		"categoría repetida": func(c *Convocatoria) {
			c.DatosPublicos.Categorias = []string{"auxiliar-administrativo", "auxiliar-administrativo"}
		},
	}
	for nombre, alterar := range pruebas {
		t.Run(nombre, func(t *testing.T) {
			convocatoria := convocatoriaPublicaValidaPrueba()
			alterar(&convocatoria)
			if err := convocatoria.ValidarPublicacion(); err == nil {
				t.Fatal("material no canónico aceptado")
			}
		})
	}
}

func TestResumenPublicoAcotaContadoresYMaterial(t *testing.T) {
	detalle := convocatoriaPublicaValidaPrueba()
	resumen := ResumenConvocatoria{
		Version: detalle.Version, Estado: detalle.Estado, HuellaSHA256: detalle.HuellaSHA256,
		DatosPublicos: &DatosPublicosResumenConvocatoria{
			IdentificadorPublico: detalle.DatosPublicos.IdentificadorPublico,
			Tipo:                 detalle.DatosPublicos.Tipo, CatalogoCategorias: detalle.DatosPublicos.CatalogoCategorias,
			Categorias: append([]string(nil), detalle.DatosPublicos.Categorias...),
			Titulo:     detalle.DatosPublicos.Titulo, Resumen: detalle.DatosPublicos.Resumen,
			PublicadaEn: detalle.DatosPublicos.PublicadaEn, ActualizadaEn: detalle.DatosPublicos.ActualizadaEn,
			Plazos: append([]PlazoConvocatoria(nil), detalle.DatosPublicos.Plazos...),
		},
		NumeroDocumentos: 1,
	}
	if err := resumen.ValidarPublicacion(); err != nil {
		t.Fatalf("resumen válido rechazado: %v", err)
	}
	resumen.NumeroAyudas = maximoAyudasConvocatoria + 1
	if err := resumen.ValidarPublicacion(); err == nil {
		t.Fatal("resumen sin cota aceptado")
	}
}
