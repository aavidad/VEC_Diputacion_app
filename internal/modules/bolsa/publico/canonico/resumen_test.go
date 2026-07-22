package canonico

import (
	"strings"
	"testing"
	"time"
)

func TestHuellaResumenVinculaListadoSinConfiarEnHuellaCompleta(t *testing.T) {
	instante := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	resumen := ResumenConvocatoriaV2{
		Esquema: EsquemaResumenConvocatoriaV2, IdentificadorPublico: "auxiliares-2026",
		Version: "v1", Estado: "inscripcion", Tipo: "bolsa_temporal",
		CatalogoCategorias: ReferenciaCatalogoCategoriasV2{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256:           strings.Repeat("a", 64),
			CatalogoHuellaProyeccionSHA256: strings.Repeat("c", 64),
		},
		Categorias: []string{"auxiliar-administrativo"}, Titulo: "Auxiliares",
		Resumen: "Convocatoria pública.", PublicadaEn: instante, ActualizadaEn: instante,
		Plazos: []PlazoConvocatoriaV1{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción",
			Descripcion: "Presentación de solicitudes.", AbreEn: instante,
			CierraEn: instante.Add(24 * time.Hour),
		}},
		NumeroRequisitos: 1, NumeroDocumentos: 1, NumeroAyudas: 1,
		HuellaCompletaSHA256: strings.Repeat("b", 64),
	}
	huella, err := resumen.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "bdf1c9a1238d3b5ca517ed9003154a9617a5a4a28e45023893f592bcb85efc88"
	if huella != esperada {
		t.Fatalf("actualice golden de resumen: %s", huella)
	}
	mutado := resumen
	mutado.Titulo = "Auxiliares modificados"
	huellaMutada, err := mutado.HuellaSHA256()
	if err != nil || huellaMutada == huella || mutado.HuellaCompletaSHA256 != resumen.HuellaCompletaSHA256 {
		t.Fatalf("el resumen alterado no cambió su huella propia: %s, %v", huellaMutada, err)
	}
}

func TestHuellaResumenFixturePostgreSQL(t *testing.T) {
	publicada := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	resumen := ResumenConvocatoriaV2{
		Esquema: EsquemaResumenConvocatoriaV2, IdentificadorPublico: "auxiliares-2026",
		Version: "v1", Estado: "inscripcion", Tipo: "bolsa_temporal",
		CatalogoCategorias: ReferenciaCatalogoCategoriasV2{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256:           strings.Repeat("a", 64),
			CatalogoHuellaProyeccionSHA256: strings.Repeat("c", 64),
		},
		Categorias:  []string{"auxiliar-administrativo"},
		Titulo:      "Bolsa temporal de auxiliares administrativos",
		Resumen:     "Convocatoria pública para auxiliares administrativos.",
		PublicadaEn: publicada, ActualizadaEn: publicada,
		Plazos: []PlazoConvocatoriaV1{{
			Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción",
			Descripcion: "Plazo de presentación de solicitudes.",
			AbreEn:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			CierraEn:    time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC),
		}},
		NumeroRequisitos: 1, NumeroDocumentos: 1, NumeroAyudas: 1,
		HuellaCompletaSHA256: "15f9d2d4bf6fb37c6bf915ff60ccfe189df78b6119330e834ffdcdbfba14e5e1",
	}
	huella, err := resumen.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "b71b88adc718cb3853153cf807c88c59ccf9dd5c0fe6ed7e78146f7d521941ef"
	if huella != esperada {
		t.Fatalf("actualice golden resumen PostgreSQL: %s", huella)
	}
}
