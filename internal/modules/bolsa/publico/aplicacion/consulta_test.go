package aplicacion

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	canonicopublico "vec-diputacion-granada/internal/modules/bolsa/publico/canonico"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

type relojFijoPrueba struct{ instante time.Time }

func (r relojFijoPrueba) Ahora() time.Time { return r.instante }

type fuentePublicaPrueba struct {
	pagina          puertosbolsa.PaginaConvocatorias
	detalle         puertosbolsa.DetalleConvocatoria
	categorias      puertosbolsa.CatalogoCategoriasPublicas
	errorValidacion error
}

type categoriasSoloActualPrueba struct {
	catalogo puertosbolsa.CatalogoCategoriasPublicas
}

func (c categoriasSoloActualPrueba) ObtenerPublicadas(
	context.Context, time.Time,
) (puertosbolsa.CatalogoCategoriasPublicas, error) {
	return c.catalogo, nil
}

func nuevaFuentePublicaPrueba(t *testing.T) (*fuentePublicaPrueba, time.Time) {
	t.Helper()
	publicada := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	ahora := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	detalle := dominiobolsa.Convocatoria{
		Version: "v1", Estado: dominiobolsa.EstadoConvocatoriaInscripcion,
		HuellaSHA256: strings.Repeat("0", 64),
		DatosPublicos: &dominiobolsa.DatosPublicosConvocatoria{
			IdentificadorPublico: "auxiliares-2026", Tipo: "bolsa_temporal",
			CatalogoCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
				CatalogoID: "categorias-profesionales", CatalogoVersion: 1,
				CatalogoHuellaSHA256:           strings.Repeat("a", 64),
				CatalogoHuellaProyeccionSHA256: strings.Repeat("b", 64),
			},
			Categorias: []string{"auxiliar-administrativo"},
			Titulo:     "Bolsa temporal de auxiliares", Resumen: "Resumen público.",
			Descripcion: "Descripción pública completa.", PublicadaEn: publicada, ActualizadaEn: publicada,
			Plazos: []dominiobolsa.PlazoConvocatoria{{
				Referencia: "plazo:inscripcion", Tipo: "inscripcion", Titulo: "Inscripción",
				Descripcion: "Presentación de solicitudes.", AbreEn: publicada,
				CierraEn: time.Date(2026, 7, 31, 23, 59, 59, 0, time.UTC),
			}},
			Requisitos: []dominiobolsa.RequisitoConvocatoria{{
				Referencia: "requisito:edad", Orden: 1, Titulo: "Edad",
				Descripcion: "Cumplir la edad exigida.", Obligatorio: true,
			}},
			Documentos: []dominiobolsa.DocumentoConvocatoria{{
				Referencia: "documento:bases", Tipo: "bases", Orden: 1, Titulo: "Bases",
				Descripcion: "Bases oficiales.", Formato: "pdf",
				URL: "/bolsa/documentos/bases.pdf", PublicadoEn: publicada,
			}},
			Ayuda: []dominiobolsa.AyudaConvocatoria{{
				Referencia: "ayuda:general", Categoria: "general", Orden: 1,
				Pregunta: "¿Cómo me inscribo?", Respuesta: "Desde el área personal.",
			}},
		},
	}
	huella, err := canonicopublico.HuellaConvocatoriaV2(detalle)
	if err != nil {
		t.Fatal(err)
	}
	detalle.HuellaSHA256 = huella
	resumen := resumirDetalle(detalle)
	metadatos := puertosbolsa.MetadatosFuenteConvocatorias{
		Revision: "revision-001", ActualizadaEn: publicada,
	}
	metadatosCategorias := puertosbolsa.MetadatosFuenteCategorias{
		Revision: "revision-001", ActualizadaEn: publicada,
	}
	catalogos := []puertosbolsa.CatalogoPublico{
		catalogoPrueba(puertosbolsa.CatalogoTiposConvocatoria, "bolsa_temporal", "Bolsa temporal", "informacion"),
		{Referencia: puertosbolsa.CatalogoEstadosConvocatoria, Version: 1, Entradas: []puertosbolsa.EntradaCatalogoPublico{
			entradaCatalogoPrueba("inscripcion", "Inscripción", "exito"),
			entradaCatalogoPrueba("cerrada", "Cerrada", "neutro"),
		}},
		catalogoPrueba(puertosbolsa.CatalogoTiposPlazo, "inscripcion", "Inscripción", "informacion"),
		catalogoPrueba(puertosbolsa.CatalogoTiposDocumento, "bases", "Bases", "documento"),
		catalogoPrueba(puertosbolsa.CatalogoCategoriasAyuda, "general", "General", "informacion"),
	}
	categorias := puertosbolsa.CatalogoCategoriasPublicas{
		ID: "categorias-profesionales", Version: 1,
		HuellaGobernadaSHA256:  strings.Repeat("a", 64),
		HuellaProyeccionSHA256: strings.Repeat("b", 64),
		Fuente:                 metadatosCategorias,
		Categorias: []puertosbolsa.CategoriaPublica{{
			Clave: "auxiliar-administrativo", Version: 1, Etiqueta: "Auxiliar administrativo",
			Descripcion: "Categoría profesional.", Semantica: "informacion", Orden: 1,
			Area: "administracion", AreaEtiqueta: "Administración", Suscribible: true,
		}},
	}
	pagina := puertosbolsa.PaginaConvocatorias{
		Convocatorias: []dominiobolsa.ResumenConvocatoria{resumen}, Total: 1,
		Catalogos: catalogos, Fuente: metadatos,
		ConteosCategorias: map[string]puertosbolsa.ConteoCategoriaConvocatorias{
			"auxiliar-administrativo": {NumeroConvocatorias: 1, NumeroPlazosAbiertos: 1},
		},
	}
	return &fuentePublicaPrueba{
		pagina: pagina, categorias: categorias,
		detalle: puertosbolsa.DetalleConvocatoria{Convocatoria: detalle, Catalogos: catalogos, Fuente: metadatos},
	}, ahora
}

func entradaCatalogoPrueba(clave, etiqueta, semantica string) puertosbolsa.EntradaCatalogoPublico {
	return puertosbolsa.EntradaCatalogoPublico{
		Clave: clave, Version: 1, Etiqueta: etiqueta, Descripcion: "Descripción pública.",
		Semantica: semantica, Orden: 1, Publicable: true,
	}
}

func catalogoPrueba(referencia, clave, etiqueta, semantica string) puertosbolsa.CatalogoPublico {
	return puertosbolsa.CatalogoPublico{
		Referencia: referencia, Version: 1,
		Entradas: []puertosbolsa.EntradaCatalogoPublico{entradaCatalogoPrueba(clave, etiqueta, semantica)},
	}
}

func (f *fuentePublicaPrueba) BuscarPublicadas(context.Context, puertosbolsa.FiltroConvocatoriasPublicas) (puertosbolsa.PaginaConvocatorias, error) {
	return f.pagina, nil
}
func (f *fuentePublicaPrueba) ObtenerPublicada(_ context.Context, id string) (puertosbolsa.DetalleConvocatoria, error) {
	if id != "auxiliares-2026" {
		return puertosbolsa.DetalleConvocatoria{}, puertosbolsa.ErrConvocatoriaNoEncontrada
	}
	return f.detalle, nil
}
func (f *fuentePublicaPrueba) ObtenerPublicadas(context.Context, time.Time) (puertosbolsa.CatalogoCategoriasPublicas, error) {
	return f.categorias, nil
}
func (f *fuentePublicaPrueba) ObtenerSnapshotsPublicados(context.Context) ([]puertosbolsa.CatalogoCategoriasPublicas, error) {
	return []puertosbolsa.CatalogoCategoriasPublicas{f.categorias}, nil
}
func (f *fuentePublicaPrueba) BuscarPublicadasConCategorias(context.Context, puertosbolsa.FiltroConvocatoriasPublicas) (puertosbolsa.LecturaListadoPublicoConsistente, error) {
	return puertosbolsa.LecturaListadoPublicoConsistente{
		Pagina: f.pagina, Categorias: f.categorias,
		SnapshotsCategorias: []puertosbolsa.CatalogoCategoriasPublicas{f.categorias},
	}, nil
}
func (f *fuentePublicaPrueba) ObtenerPublicadaConCategorias(ctx context.Context, id string, _ time.Time) (puertosbolsa.LecturaDetallePublicoConsistente, error) {
	detalle, err := f.ObtenerPublicada(ctx, id)
	return puertosbolsa.LecturaDetallePublicoConsistente{
		Detalle: detalle, Categorias: f.categorias,
		SnapshotsCategorias: []puertosbolsa.CatalogoCategoriasPublicas{f.categorias},
	}, err
}
func (f *fuentePublicaPrueba) ConsultarCategoriasConConteos(context.Context, time.Time) (puertosbolsa.LecturaCategoriasPublicasConsistente, error) {
	return puertosbolsa.LecturaCategoriasPublicasConsistente{Pagina: f.pagina, Categorias: f.categorias}, nil
}
func (f *fuentePublicaPrueba) ValidarConfiguracionPublica(context.Context, time.Time) error {
	return f.errorValidacion
}

func TestServicioPublicoListaYDetalleConLaMismaHuella(t *testing.T) {
	fuente, ahora := nuevaFuentePublicaPrueba(t)
	servicio, err := NuevoServicioConsultaPublicaConsistente(fuente, relojFijoPrueba{ahora})
	if err != nil {
		t.Fatal(err)
	}
	listado, err := servicio.Listar(context.Background(), SolicitudListadoPublico{})
	if err != nil || listado.Esquema != "vec.bolsa.publico.convocatorias.v2" ||
		listado.Paginacion.Total != 1 || len(listado.Convocatorias) != 1 {
		t.Fatalf("listado = %+v, %v", listado, err)
	}
	detalle, err := servicio.Obtener(context.Background(), "auxiliares-2026")
	if err != nil || detalle.Esquema != "vec.bolsa.publico.convocatoria.v2" ||
		detalle.Convocatoria.HuellaSHA256 != listado.Convocatorias[0].HuellaSHA256 ||
		len(detalle.Documentos) != 1 || detalle.Descripcion == "" ||
		len(detalle.DiccionarioCategorias) != len(detalle.Convocatoria.Categorias) {
		t.Fatalf("detalle = %+v, %v", detalle, err)
	}
	if listado.Convocatorias[0].NumeroRequisitos != 1 || listado.Convocatorias[0].NumeroDocumentos != 1 ||
		listado.Convocatorias[0].NumeroAyudas != 1 {
		t.Fatalf("contadores de resumen = %+v", listado.Convocatorias[0])
	}
	facetas := make(map[ReferenciaCategoriaPublica]struct{}, len(listado.Facetas.Categorias))
	for _, categoria := range listado.Facetas.Categorias {
		facetas[ReferenciaCategoriaPublica{Clave: categoria.Clave, Version: categoria.Version}] = struct{}{}
	}
	for _, referencia := range listado.Convocatorias[0].Categorias {
		if _, existe := facetas[referencia]; !existe {
			t.Fatalf("referencia de categoría sin faceta exacta: %+v", referencia)
		}
	}
}

func TestConstructorLegacyExigePuertoDeSnapshotsHistoricos(t *testing.T) {
	fuente, ahora := nuevaFuentePublicaPrueba(t)
	_, err := NuevoServicioConsultaPublica(
		fuente, categoriasSoloActualPrueba{catalogo: fuente.categorias}, relojFijoPrueba{ahora},
	)
	if !errors.Is(err, ErrServicioConsultaPublicaInvalido) {
		t.Fatalf("constructor current-only no fallo cerrado: %v", err)
	}
}

func TestIndiceMultiversionAdmiteActualVacioYRechazaFacetaReinterpretada(t *testing.T) {
	fuente, _ := nuevaFuentePublicaPrueba(t)
	historico := fuente.categorias
	historico.Categorias = append([]puertosbolsa.CategoriaPublica(nil), fuente.categorias.Categorias...)
	actual := fuente.categorias
	actual.Version = 2
	actual.HuellaGobernadaSHA256 = strings.Repeat("c", 64)
	actual.HuellaProyeccionSHA256 = strings.Repeat("d", 64)
	actual.Categorias = nil
	snapshotActual := fuente.categorias
	snapshotActual.Categorias = append([]puertosbolsa.CategoriaPublica(nil), fuente.categorias.Categorias...)
	snapshotActual.Version = 2
	snapshotActual.HuellaGobernadaSHA256 = actual.HuellaGobernadaSHA256
	snapshotActual.HuellaProyeccionSHA256 = actual.HuellaProyeccionSHA256
	snapshotActual.Categorias[0].Version = 2

	indice, err := nuevoIndiceCatalogos(
		fuente.pagina.Catalogos, actual,
		[]puertosbolsa.CatalogoCategoriasPublicas{historico, snapshotActual},
	)
	if err != nil || len(indice.ordenados[puertosbolsa.CatalogoCategoriasConvocatoria]) != 0 {
		t.Fatalf("actual vacío con histórico resoluble = %+v, %v", indice, err)
	}
	referenciaHistorica := dominiobolsa.ReferenciaCatalogoCategorias{
		CatalogoID: historico.ID, CatalogoVersion: historico.Version,
		CatalogoHuellaSHA256:           historico.HuellaGobernadaSHA256,
		CatalogoHuellaProyeccionSHA256: historico.HuellaProyeccionSHA256,
	}
	if categoria, err := indice.resolverCategoria(referenciaHistorica, "auxiliar-administrativo"); err != nil || categoria.Etiqueta != "Auxiliar administrativo" {
		t.Fatalf("snapshot histórico no resoluble: %+v, %v", categoria, err)
	}

	actualAlterado := snapshotActual
	actualAlterado.Categorias = append([]puertosbolsa.CategoriaPublica(nil), snapshotActual.Categorias...)
	actualAlterado.Categorias[0].Etiqueta = "Etiqueta reinterpretada"
	if _, err := nuevoIndiceCatalogos(
		fuente.pagina.Catalogos, actualAlterado,
		[]puertosbolsa.CatalogoCategoriasPublicas{historico, snapshotActual},
	); !errors.Is(err, ErrDatosPublicosNoConfiables) {
		t.Fatalf("faceta ajena al snapshot no fallo cerrada: %v", err)
	}
}

func TestServicioPublicoResuelveCategoriaAunqueNoSeaFacetaActual(t *testing.T) {
	fuente, ahora := nuevaFuentePublicaPrueba(t)
	delete(fuente.pagina.ConteosCategorias, "auxiliar-administrativo")
	servicio, _ := NuevoServicioConsultaPublicaConsistente(fuente, relojFijoPrueba{ahora})
	listado, err := servicio.Listar(context.Background(), SolicitudListadoPublico{})
	if err != nil || len(listado.Facetas.Categorias) != 0 || len(listado.DiccionarioCategorias) != 1 {
		t.Fatalf("resolución histórica sin faceta actual = %+v, %v", listado, err)
	}
}

func TestCoberturaCategoriasRechazaDiccionarioDuplicadoOVersionDesconocida(t *testing.T) {
	resumenes := []ResumenConvocatoriaPublica{{
		CatalogoCategorias: ReferenciaCatalogoCategoriasConvocatoriaPublica{
			Referencia: "categorias-profesionales", Version: 1,
			HuellaSHA256:           strings.Repeat("a", 64),
			HuellaProyeccionSHA256: strings.Repeat("b", 64),
		},
		Categorias: []ReferenciaCategoriaPublica{{Clave: "auxiliar-administrativo", Version: 2}},
	}}
	duplicadas := []CategoriaDiccionarioPublico{
		{CatalogoCategorias: resumenes[0].CatalogoCategorias, Clave: "auxiliar-administrativo", Version: 1},
		{CatalogoCategorias: resumenes[0].CatalogoCategorias, Clave: "auxiliar-administrativo", Version: 1},
	}
	if err := validarCoberturaCategoriasResumenes(resumenes, duplicadas); !errors.Is(err, ErrDatosPublicosNoConfiables) {
		t.Fatalf("diccionario duplicado aceptado: %v", err)
	}
	if err := validarCoberturaCategoriasResumenes(resumenes, duplicadas[:1]); !errors.Is(err, ErrDatosPublicosNoConfiables) {
		t.Fatalf("versión desconocida aceptada: %v", err)
	}
}

func TestServicioPublicoRechazaHuellaDeDetalleAlterada(t *testing.T) {
	fuente, ahora := nuevaFuentePublicaPrueba(t)
	fuente.detalle.Convocatoria.HuellaSHA256 = strings.Repeat("f", 64)
	servicio, _ := NuevoServicioConsultaPublicaConsistente(fuente, relojFijoPrueba{ahora})
	if _, err := servicio.Obtener(context.Background(), "auxiliares-2026"); !errors.Is(err, ErrDatosPublicosNoConfiables) {
		t.Fatalf("huella alterada aceptada: %v", err)
	}
}

func TestServicioPublicoFiltrosYCierreInclusivo(t *testing.T) {
	fuente, ahora := nuevaFuentePublicaPrueba(t)
	servicio, _ := NuevoServicioConsultaPublicaConsistente(fuente, relojFijoPrueba{ahora})
	for _, solicitud := range []SolicitudListadoPublico{
		{Texto: strings.Repeat("x", LongitudTextoPublicoMaxima+1)},
		{Pagina: PaginaPublicaMaxima + 1},
		{Tamano: TamanoPaginaPublicaMaximo + 1},
		{Categoria: "categoria inexistente"},
	} {
		if _, err := servicio.Listar(context.Background(), solicitud); !errors.Is(err, ErrFiltroPublicoInvalido) {
			t.Fatalf("filtro inválido aceptado: %+v, %v", solicitud, err)
		}
	}
	plazo := fuente.detalle.Convocatoria.DatosPublicos.Plazos[0]
	if situacion, _, _ := situacionPlazo(plazo, plazo.CierraEn); situacion != "abierto" {
		t.Fatalf("el instante de cierre no fue inclusivo: %s", situacion)
	}
	if situacion, _, _ := situacionPlazo(plazo, plazo.CierraEn.Add(time.Microsecond)); situacion != "cerrado" {
		t.Fatalf("el microsegundo posterior no cerró el plazo: %s", situacion)
	}
}
