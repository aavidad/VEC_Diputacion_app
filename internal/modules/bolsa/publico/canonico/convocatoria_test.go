package canonico

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestHuellaConvocatoriaPublicaNoTieneIdentidadInterna(t *testing.T) {
	instante := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	convocatoria := ConvocatoriaV2{
		Esquema: EsquemaConvocatoriaV2, IdentificadorPublico: "auxiliares-2026",
		Version: "v1", Estado: "cerrada", Tipo: "bolsa_temporal",
		CatalogoCategorias: ReferenciaCatalogoCategoriasV2{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			CatalogoHuellaProyeccionSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Categorias: []string{"auxiliar-administrativo"}, Titulo: "Auxiliares",
		Resumen: "Resumen público.", Descripcion: "Descripción pública.",
		PublicadaEn: instante, ActualizadaEn: instante,
	}
	huella, err := convocatoria.HuellaSHA256()
	if err != nil || len(huella) != 64 {
		t.Fatalf("huella pública = %q, %v", huella, err)
	}
	otraProyeccion := convocatoria
	otraProyeccion.CatalogoCategorias.CatalogoHuellaProyeccionSHA256 =
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	huellaOtraProyeccion, err := otraProyeccion.HuellaSHA256()
	if err != nil || huellaOtraProyeccion == huella {
		t.Fatalf("la referencia no inmoviliza la proyección: %q, %v", huellaOtraProyeccion, err)
	}
	contenido, err := json.Marshal(convocatoria)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(contenido, []byte("referencia_agregado")) || bytes.Contains(contenido, []byte(`"id"`)) {
		t.Fatalf("canon público contiene identidad interna: %s", contenido)
	}
}

func TestHuellaConvocatoriaPublicaFixturePostgreSQL(t *testing.T) {
	publicada := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	convocatoria := ConvocatoriaV2{
		Esquema: EsquemaConvocatoriaV2, IdentificadorPublico: "auxiliares-2026",
		Version: "v1", Estado: "inscripcion", Tipo: "bolsa_temporal",
		CatalogoCategorias: ReferenciaCatalogoCategoriasV2{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256:           "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			CatalogoHuellaProyeccionSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Categorias:  []string{"auxiliar-administrativo"},
		Titulo:      "Bolsa temporal de auxiliares administrativos",
		Resumen:     "Convocatoria pública para auxiliares administrativos.",
		Descripcion: "Proceso selectivo sujeto a bases firmadas y publicadas.",
		PublicadaEn: publicada, ActualizadaEn: publicada,
		Plazos: []PlazoConvocatoriaV1{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción",
			Descripcion: "Plazo de presentación de solicitudes.",
			AbreEn:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			CierraEn:    time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC),
		}},
		Requisitos: []RequisitoConvocatoriaV1{{
			Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
			Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
		}},
		Documentos: []DocumentoConvocatoriaV1{{
			Referencia: "documento:bases", Tipo: "bases", Orden: 1,
			Titulo: "Bases reguladoras", Descripcion: "Bases oficiales de la convocatoria.",
			Formato: "pdf", URL: "/bolsa/documentos/bases-auxiliares-2026.pdf", PublicadoEn: publicada,
		}},
		Ayuda: []AyudaConvocatoriaV1{{
			Referencia: "ayuda:inscripcion", Categoria: "general", Orden: 1,
			Pregunta: "¿Cómo presento la solicitud?", Respuesta: "Acceda al área personal durante el plazo.",
		}},
	}
	huella, err := convocatoria.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "15f9d2d4bf6fb37c6bf915ff60ccfe189df78b6119330e834ffdcdbfba14e5e1"
	if huella != esperada {
		t.Fatalf("actualice el golden del fixture PostgreSQL: %s", huella)
	}
}
