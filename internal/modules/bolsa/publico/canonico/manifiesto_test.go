package canonico

import (
	"strings"
	"testing"
	"time"
)

func manifiestoPrueba(t *testing.T) ManifiestoPublicoV2 {
	t.Helper()
	categorias := catalogoCategoriasPrueba(t)
	categorias.Categorias = append(categorias.Categorias, CategoriaCatalogoV1{
		Clave: "tecnico-administracion", Etiqueta: "Técnico de administración",
		Descripcion: "Categoría profesional de técnico de administración.",
		Semantica:   "informacion", Orden: 2, Area: "administracion",
		AreaEtiqueta: "Administración", Suscribible: true,
		VigenteDesde: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	huellaCategorias, err := categorias.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	categoriasHistoricas := categorias
	categoriasHistoricas.Version = 2
	categoriasHistoricas.Categorias = clonarCategorias(categorias.Categorias)
	categoriasHistoricas.Categorias[0].Etiqueta = "Auxiliar administrativo histórico"
	huellaCategoriasHistoricas, err := categoriasHistoricas.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	return ManifiestoPublicoV2{
		Esquema: EsquemaManifiestoPublicoV2,
		Fuente:  FuenteManifiestoPublicoV2{Revision: "revision-001", ActualizadaEn: instante},
		Catalogos: []CatalogoManifiestoV2{{
			Referencia: "tipos_convocatoria", Version: 1,
			Entradas: []EntradaCatalogoManifiestoV2{
				{Clave: "bolsa_temporal", Etiqueta: "Bolsa temporal", Descripcion: "Proceso temporal.", Semantica: "informacion", Orden: 1},
				{Clave: "ope", Etiqueta: "Oferta de empleo", Descripcion: "Proceso estable.", Semantica: "exito", Orden: 2},
			},
		}},
		Categorias: CategoriasManifiestoPublicoV2{
			Actual: ReferenciaCatalogoCategoriasManifiestoV2{
				CatalogoID: categorias.CatalogoID, CatalogoVersion: categorias.Version,
				CatalogoHuellaSHA256:           strings.Repeat("a", 64),
				CatalogoHuellaProyeccionSHA256: huellaCategorias,
			},
			Snapshots: []SnapshotCategoriasManifiestoV2{
				{HuellaGobernadaSHA256: strings.Repeat("a", 64), HuellaProyeccionSHA256: huellaCategorias, Catalogo: categorias},
				{HuellaGobernadaSHA256: strings.Repeat("f", 64), HuellaProyeccionSHA256: huellaCategoriasHistoricas, Catalogo: categoriasHistoricas},
			},
		},
		Convocatorias: []ConvocatoriaManifiestoPublicoV2{
			{IdentificadorPublico: "auxiliares-2026", HuellaCompletaSHA256: strings.Repeat("b", 64), HuellaResumenSHA256: strings.Repeat("c", 64)},
			{IdentificadorPublico: "operarios-2026", HuellaCompletaSHA256: strings.Repeat("d", 64), HuellaResumenSHA256: strings.Repeat("e", 64)},
		},
	}
}

func TestHuellaManifiestoPublicoGoldenYOrdenEstable(t *testing.T) {
	manifiesto := manifiestoPrueba(t)
	huella, err := manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "e480e5c70e02f8a95e6766316a277d649a4d0c57fdb160f9eeef48daddc03987"
	if huella != esperada {
		t.Fatalf("actualice golden de manifiesto: %s", huella)
	}
	reordenado := manifiesto
	reordenado.Catalogos = clonarCatalogosManifiesto(manifiesto.Catalogos)
	reordenado.Catalogos[0].Entradas[0], reordenado.Catalogos[0].Entradas[1] =
		reordenado.Catalogos[0].Entradas[1], reordenado.Catalogos[0].Entradas[0]
	reordenado.Categorias.Snapshots = clonarSnapshotsCategoriasManifiesto(manifiesto.Categorias.Snapshots)
	reordenado.Categorias.Snapshots[0], reordenado.Categorias.Snapshots[1] =
		reordenado.Categorias.Snapshots[1], reordenado.Categorias.Snapshots[0]
	reordenado.Categorias.Snapshots[1].Catalogo.Categorias[0], reordenado.Categorias.Snapshots[1].Catalogo.Categorias[1] =
		reordenado.Categorias.Snapshots[1].Catalogo.Categorias[1], reordenado.Categorias.Snapshots[1].Catalogo.Categorias[0]
	reordenado.Convocatorias = append([]ConvocatoriaManifiestoPublicoV2(nil), manifiesto.Convocatorias...)
	reordenado.Convocatorias[0], reordenado.Convocatorias[1] =
		reordenado.Convocatorias[1], reordenado.Convocatorias[0]
	huellaReordenada, err := reordenado.HuellaSHA256()
	if err != nil || huellaReordenada != huella {
		t.Fatalf("el orden de entrada alteró el canónico: %s, %v", huellaReordenada, err)
	}
}

func TestManifiestoVinculaFuenteCatalogosCategoriasYConjuntoCompleto(t *testing.T) {
	original := manifiestoPrueba(t)
	huella, err := original.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	mutaciones := map[string]func(*ManifiestoPublicoV2){
		"fuente": func(m *ManifiestoPublicoV2) { m.Fuente.Revision = "revision-002" },
		"snapshot categorias": func(m *ManifiestoPublicoV2) {
			m.Categorias.Snapshots[0].Catalogo.Categorias[0].Etiqueta = "Etiqueta modificada"
		},
		"referencia actual": func(m *ManifiestoPublicoV2) {
			m.Categorias.Actual.CatalogoHuellaSHA256 = strings.Repeat("f", 64)
		},
		"faceta catalogo":             func(m *ManifiestoPublicoV2) { m.Catalogos[0].Entradas[0].Etiqueta = "Bolsa modificada" },
		"resumen con detalle intacto": func(m *ManifiestoPublicoV2) { m.Convocatorias[0].HuellaResumenSHA256 = strings.Repeat("f", 64) },
		"detalle":                     func(m *ManifiestoPublicoV2) { m.Convocatorias[0].HuellaCompletaSHA256 = strings.Repeat("f", 64) },
		"sustitucion fuera de pagina con igual cardinalidad": func(m *ManifiestoPublicoV2) { m.Convocatorias[1].IdentificadorPublico = "tecnicos-2026" },
		"eliminacion": func(m *ManifiestoPublicoV2) { m.Convocatorias = m.Convocatorias[:1] },
		"adicion": func(m *ManifiestoPublicoV2) {
			m.Convocatorias = append(m.Convocatorias, ConvocatoriaManifiestoPublicoV2{
				IdentificadorPublico: "tecnicos-2026", HuellaCompletaSHA256: strings.Repeat("1", 64),
				HuellaResumenSHA256: strings.Repeat("2", 64),
			})
		},
	}
	for nombre, mutar := range mutaciones {
		t.Run(nombre, func(t *testing.T) {
			mutado := manifiestoPrueba(t)
			mutar(&mutado)
			huellaMutada, err := mutado.HuellaSHA256()
			if err == nil && huellaMutada == huella {
				t.Fatalf("mutación no vinculada: %s, %v", huellaMutada, err)
			}
		})
	}
}

func TestHuellaManifiestoFixtureSimpleV2(t *testing.T) {
	categorias := catalogoCategoriasPrueba(t)
	huellaCategorias, err := categorias.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	entrada := func(clave, etiqueta, descripcion, semantica string, orden int) EntradaCatalogoManifiestoV2 {
		return EntradaCatalogoManifiestoV2{
			Clave: clave, Etiqueta: etiqueta, Descripcion: descripcion,
			Semantica: semantica, Orden: orden,
		}
	}
	manifiesto := ManifiestoPublicoV2{
		Esquema: EsquemaManifiestoPublicoV2,
		Fuente:  FuenteManifiestoPublicoV2{Revision: "revision-001", ActualizadaEn: instante},
		Catalogos: []CatalogoManifiestoV2{
			{Referencia: "tipos_convocatoria", Version: 1, Entradas: []EntradaCatalogoManifiestoV2{entrada("bolsa_temporal", "Bolsa temporal", "Proceso temporal.", "informacion", 1)}},
			{Referencia: "estados_convocatoria", Version: 1, Entradas: []EntradaCatalogoManifiestoV2{
				entrada("inscripcion", "Inscripción", "Plazo de inscripción.", "exito", 1),
				entrada("cerrada", "Cerrada", "Proceso cerrado.", "neutro", 2),
			}},
			{Referencia: "tipos_plazo", Version: 1, Entradas: []EntradaCatalogoManifiestoV2{entrada("inscripcion", "Inscripción", "Presentación de solicitudes.", "informacion", 1)}},
			{Referencia: "tipos_documento", Version: 1, Entradas: []EntradaCatalogoManifiestoV2{entrada("bases", "Bases", "Bases reguladoras.", "documento", 1)}},
			{Referencia: "categorias_ayuda", Version: 1, Entradas: []EntradaCatalogoManifiestoV2{entrada("general", "General", "Información general.", "informacion", 1)}},
		},
		Categorias: CategoriasManifiestoPublicoV2{
			Actual: ReferenciaCatalogoCategoriasManifiestoV2{
				CatalogoID: categorias.CatalogoID, CatalogoVersion: categorias.Version,
				CatalogoHuellaSHA256:           strings.Repeat("a", 64),
				CatalogoHuellaProyeccionSHA256: huellaCategorias,
			},
			Snapshots: []SnapshotCategoriasManifiestoV2{{
				HuellaGobernadaSHA256:  strings.Repeat("a", 64),
				HuellaProyeccionSHA256: huellaCategorias, Catalogo: categorias,
			}},
		},
		Convocatorias: []ConvocatoriaManifiestoPublicoV2{{
			IdentificadorPublico: "auxiliares-2026",
			HuellaCompletaSHA256: "8bd7e733796952ed2ab1b9b9da30303e9e08660631afd634e8786641453b4724",
			HuellaResumenSHA256:  "be322bbeae04721907f6cd10d5e2abc1c660c7ac832d6d8887c1a76a0b4d3e15",
		}},
	}
	huella, err := manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "68e647bbe475a05d114fe1ab6b23b3809d9fbb4c1fed14e70571b99836b539e1"
	if huella != esperada {
		t.Fatalf("actualice golden manifiesto PostgreSQL: %s", huella)
	}
}
