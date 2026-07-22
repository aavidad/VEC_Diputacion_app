package canonico

import (
	"strings"
	"testing"
	"time"
)

func manifiestoPrueba(t *testing.T) ManifiestoPublicoV1 {
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
	instante := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	return ManifiestoPublicoV1{
		Esquema: EsquemaManifiestoPublicoV1,
		Fuente:  FuenteManifiestoPublicoV1{Revision: "revision-001", ActualizadaEn: instante},
		Catalogos: []CatalogoManifiestoV1{{
			Referencia: "tipos_convocatoria", Version: 1,
			Entradas: []EntradaCatalogoManifiestoV1{
				{Clave: "bolsa_temporal", Etiqueta: "Bolsa temporal", Descripcion: "Proceso temporal.", Semantica: "informacion", Orden: 1},
				{Clave: "ope", Etiqueta: "Oferta de empleo", Descripcion: "Proceso estable.", Semantica: "exito", Orden: 2},
			},
		}},
		Categorias: CategoriasManifiestoPublicoV1{
			HuellaGobernadaSHA256:  strings.Repeat("a", 64),
			HuellaProyeccionSHA256: huellaCategorias,
			Revision:               "revision-001", ActualizadaEn: instante, Catalogo: categorias,
		},
		Convocatorias: []ConvocatoriaManifiestoPublicoV1{
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
	const esperada = "be0d3d2128b410591bfe1849a47f7af295b44efbeae4bbf8481acb435eece81a"
	if huella != esperada {
		t.Fatalf("actualice golden de manifiesto: %s", huella)
	}
	reordenado := manifiesto
	reordenado.Catalogos = clonarCatalogosManifiesto(manifiesto.Catalogos)
	reordenado.Catalogos[0].Entradas[0], reordenado.Catalogos[0].Entradas[1] =
		reordenado.Catalogos[0].Entradas[1], reordenado.Catalogos[0].Entradas[0]
	reordenado.Categorias.Catalogo.Categorias = clonarCategorias(manifiesto.Categorias.Catalogo.Categorias)
	reordenado.Categorias.Catalogo.Categorias[0], reordenado.Categorias.Catalogo.Categorias[1] =
		reordenado.Categorias.Catalogo.Categorias[1], reordenado.Categorias.Catalogo.Categorias[0]
	reordenado.Convocatorias = append([]ConvocatoriaManifiestoPublicoV1(nil), manifiesto.Convocatorias...)
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
	mutaciones := map[string]func(*ManifiestoPublicoV1){
		"fuente": func(m *ManifiestoPublicoV1) { m.Fuente.Revision = "revision-002" },
		"fecha categorias": func(m *ManifiestoPublicoV1) {
			m.Categorias.ActualizadaEn = m.Categorias.ActualizadaEn.Add(time.Microsecond)
		},
		"revision categorias":         func(m *ManifiestoPublicoV1) { m.Categorias.Revision = "revision-002" },
		"faceta catalogo":             func(m *ManifiestoPublicoV1) { m.Catalogos[0].Entradas[0].Etiqueta = "Bolsa modificada" },
		"resumen con detalle intacto": func(m *ManifiestoPublicoV1) { m.Convocatorias[0].HuellaResumenSHA256 = strings.Repeat("f", 64) },
		"detalle":                     func(m *ManifiestoPublicoV1) { m.Convocatorias[0].HuellaCompletaSHA256 = strings.Repeat("f", 64) },
		"sustitucion fuera de pagina con igual cardinalidad": func(m *ManifiestoPublicoV1) { m.Convocatorias[1].IdentificadorPublico = "tecnicos-2026" },
		"eliminacion": func(m *ManifiestoPublicoV1) { m.Convocatorias = m.Convocatorias[:1] },
		"adicion": func(m *ManifiestoPublicoV1) {
			m.Convocatorias = append(m.Convocatorias, ConvocatoriaManifiestoPublicoV1{
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
			if err != nil || huellaMutada == huella {
				t.Fatalf("mutación no vinculada: %s, %v", huellaMutada, err)
			}
		})
	}
}

func TestHuellaManifiestoFixturePostgreSQL(t *testing.T) {
	categorias := catalogoCategoriasPrueba(t)
	huellaCategorias, err := categorias.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, 7, 22, 10, 0, 0, 0, time.UTC)
	entrada := func(clave, etiqueta, descripcion, semantica string, orden int) EntradaCatalogoManifiestoV1 {
		return EntradaCatalogoManifiestoV1{
			Clave: clave, Etiqueta: etiqueta, Descripcion: descripcion,
			Semantica: semantica, Orden: orden,
		}
	}
	manifiesto := ManifiestoPublicoV1{
		Esquema: EsquemaManifiestoPublicoV1,
		Fuente:  FuenteManifiestoPublicoV1{Revision: "revision-001", ActualizadaEn: instante},
		Catalogos: []CatalogoManifiestoV1{
			{Referencia: "tipos_convocatoria", Version: 1, Entradas: []EntradaCatalogoManifiestoV1{entrada("bolsa_temporal", "Bolsa temporal", "Proceso temporal.", "informacion", 1)}},
			{Referencia: "estados_convocatoria", Version: 1, Entradas: []EntradaCatalogoManifiestoV1{
				entrada("inscripcion", "Inscripción", "Plazo de inscripción.", "exito", 1),
				entrada("cerrada", "Cerrada", "Proceso cerrado.", "neutro", 2),
			}},
			{Referencia: "tipos_plazo", Version: 1, Entradas: []EntradaCatalogoManifiestoV1{entrada("inscripcion", "Inscripción", "Presentación de solicitudes.", "informacion", 1)}},
			{Referencia: "tipos_documento", Version: 1, Entradas: []EntradaCatalogoManifiestoV1{entrada("bases", "Bases", "Bases reguladoras.", "documento", 1)}},
			{Referencia: "categorias_ayuda", Version: 1, Entradas: []EntradaCatalogoManifiestoV1{entrada("general", "General", "Información general.", "informacion", 1)}},
		},
		Categorias: CategoriasManifiestoPublicoV1{
			HuellaGobernadaSHA256:  strings.Repeat("a", 64),
			HuellaProyeccionSHA256: huellaCategorias,
			Revision:               "revision-001", ActualizadaEn: instante, Catalogo: categorias,
		},
		Convocatorias: []ConvocatoriaManifiestoPublicoV1{{
			IdentificadorPublico: "auxiliares-2026",
			HuellaCompletaSHA256: "8bd7e733796952ed2ab1b9b9da30303e9e08660631afd634e8786641453b4724",
			HuellaResumenSHA256:  "be322bbeae04721907f6cd10d5e2abc1c660c7ac832d6d8887c1a76a0b4d3e15",
		}},
	}
	huella, err := manifiesto.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "2a85abd0a1e78d828fe27baf619349caf8e4e8a3e0bf20815279dd98a966889a"
	if huella != esperada {
		t.Fatalf("actualice golden manifiesto PostgreSQL: %s", huella)
	}
}
