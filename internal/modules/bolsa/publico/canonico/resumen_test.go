package canonico

import (
	"strings"
	"testing"
	"time"
)

func TestHuellaResumenVinculaListadoSinConfiarEnHuellaCompleta(t *testing.T) {
	instante := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	resumen := ResumenConvocatoriaV1{
		Esquema: EsquemaResumenConvocatoriaV1, IdentificadorPublico: "auxiliares-2026",
		Version: "v1", Estado: "inscripcion", Tipo: "bolsa_temporal",
		CatalogoCategorias: ReferenciaCatalogoCategoriasV1{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64),
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
	const esperada = "1462cb3abc361358330572b1216067284d796609575cc4e5e1ff3de949f9d915"
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
	resumen := ResumenConvocatoriaV1{
		Esquema: EsquemaResumenConvocatoriaV1, IdentificadorPublico: "auxiliares-2026",
		Version: "v1", Estado: "inscripcion", Tipo: "bolsa_temporal",
		CatalogoCategorias: ReferenciaCatalogoCategoriasV1{
			CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("a", 64),
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
		HuellaCompletaSHA256: "8bd7e733796952ed2ab1b9b9da30303e9e08660631afd634e8786641453b4724",
	}
	huella, err := resumen.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "be322bbeae04721907f6cd10d5e2abc1c660c7ac832d6d8887c1a76a0b4d3e15"
	if huella != esperada {
		t.Fatalf("actualice golden resumen PostgreSQL: %s", huella)
	}
}
