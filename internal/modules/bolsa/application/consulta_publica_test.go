package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	catalogosvec "vec-diputacion-granada/internal/modules/bolsa/adapters/catalogosvec"
	ficherobolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/fichero"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ficherovec "vec-diputacion-granada/internal/vec/adapters/fichero"
)

type relojPublicoFijo struct{ instante time.Time }

func (r relojPublicoFijo) Ahora() time.Time { return r.instante }

func servicioPublicoPrueba(t *testing.T, instante time.Time) *aplicacionbolsa.ServicioConsultaPublica {
	t.Helper()
	adaptador, err := ficherobolsa.NuevaConsultaConvocatorias("../../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	categorias := categoriasPublicasPrueba(t)
	servicio, err := aplicacionbolsa.NuevoServicioConsultaPublica(adaptador, categorias, relojPublicoFijo{instante: instante})
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func categoriasPublicasPrueba(t *testing.T) puertosbolsa.ConsultaCategoriasPublicas {
	t.Helper()
	paquete, err := ficherovec.NuevaConsultaCatalogos("../../../../data/catalogos/categorias-profesionales/v1.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	categorias, err := catalogosvec.NuevaConsultaCategorias(paquete, "categorias-profesionales", 1)
	if err != nil {
		t.Fatal(err)
	}
	return categorias
}

func TestListadoPublicoMinimizaDatosYResuelveCatalogos(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC))
	resultado, err := servicio.Listar(context.Background(), aplicacionbolsa.SolicitudListadoPublico{Tamano: 12})
	if err != nil || resultado.Paginacion.Total != 36 || len(resultado.Facetas.Tipos) != 2 {
		t.Fatalf("resultado = %#v, error = %v", resultado, err)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatal(err)
	}
	texto := string(contenido)
	for _, prohibido := range []string{`"proceso_ref"`, "Titulación indicada en las bases", `"documentos":`, `"requisitos":`, `"dni":`, `"correo":`} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("la proyección de listado contiene %q", prohibido)
		}
	}
}

type categoriasConHuellaDistinta struct {
	base puertosbolsa.ConsultaCategoriasPublicas
}

func (c categoriasConHuellaDistinta) ObtenerPublicadas(ctx context.Context, instante time.Time) (puertosbolsa.CatalogoCategoriasPublicas, error) {
	resultado, err := c.base.ObtenerPublicadas(ctx, instante)
	if err == nil {
		resultado.HuellaSHA256 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	}
	return resultado, err
}

func TestValidacionAnticipadaFijaCatalogoExactoAntesDePublicarRutas(t *testing.T) {
	instante := time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)
	if err := servicioPublicoPrueba(t, instante).ValidarConfiguracion(context.Background()); err != nil {
		t.Fatalf("configuracion valida rechazada: %v", err)
	}
	adaptador, err := ficherobolsa.NuevaConsultaConvocatorias("../../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := aplicacionbolsa.NuevoServicioConsultaPublica(
		adaptador,
		categoriasConHuellaDistinta{base: categoriasPublicasPrueba(t)},
		relojPublicoFijo{instante: instante},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := servicio.ValidarConfiguracion(context.Background()); !errors.Is(err, aplicacionbolsa.ErrDatosPublicosNoConfiables) {
		t.Fatalf("huella distinta no rechazada: %v", err)
	}
}

func TestListadoUsaFacetasProfesionalesConConteoSinConvertirConvocatoriasEnAutoridad(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC))
	resultado, err := servicio.Listar(context.Background(), aplicacionbolsa.SolicitudListadoPublico{Categoria: "operario"})
	if err != nil || resultado.Paginacion.Total != 2 || len(resultado.Facetas.Categorias) != 35 {
		t.Fatalf("resultado=%#v error=%v", resultado, err)
	}
	conteos := make(map[string]int, len(resultado.Facetas.Categorias))
	for _, faceta := range resultado.Facetas.Categorias {
		conteos[faceta.Clave] = faceta.NumeroResultados
	}
	if conteos["operario"] != 2 || conteos["bombero"] != 3 || conteos["tecnico-de-gestion"] != 1 {
		t.Fatalf("facetas sin conteo independiente del filtro: %#v", conteos)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(contenido), `"numero_resultados"`) != 35 {
		t.Fatalf("numero_resultados se expuso fuera de facetas categoria: %s", contenido)
	}
}

func TestDirectorioPublicoExpone68CategoriasMinimizadasYConteos(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC))
	resultado, err := servicio.ListarCategorias(context.Background())
	if err != nil || resultado.Esquema != "vec.bolsa.publico.categorias.v1" ||
		resultado.Catalogo.Total != 68 || len(resultado.Categorias) != 68 || !resultado.Fuente.Demostracion {
		t.Fatalf("resultado=%#v error=%v", resultado, err)
	}
	if primera := resultado.Categorias[0]; primera.Clave != "administrativo" || primera.Orden != 1 ||
		primera.Area != "administracion_general" || primera.AreaEtiqueta != "Administración general" {
		t.Fatalf("primera categoria no conserva orden/area gobernados: %#v", primera)
	}
	conteos := map[string]aplicacionbolsa.CategoriaDirectorioPublico{}
	for _, categoria := range resultado.Categorias {
		conteos[categoria.Clave] = categoria
	}
	if conteos["operario"].NumeroConvocatorias != 2 ||
		conteos["operario"].NumeroPlazosAbiertos != 0 ||
		conteos["tecnico-de-gestion"].NumeroConvocatorias != 1 ||
		conteos["tecnico-de-gestion"].NumeroPlazosAbiertos != 0 {
		t.Fatalf("conteos=%#v", conteos)
	}
	contenido, err := json.Marshal(resultado)
	if err != nil {
		t.Fatal(err)
	}
	for _, prohibido := range []string{"source_path", "creado_por", "publicado_por", "aprobacion_ref", "origen_sha256"} {
		if strings.Contains(string(contenido), prohibido) {
			t.Fatalf("la proyeccion publica contiene %q", prohibido)
		}
	}
}

func TestConsultaPublicaAcotaFiltrosYPaginacion(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC))
	pruebas := []struct {
		nombre    string
		solicitud aplicacionbolsa.SolicitudListadoPublico
	}{
		{nombre: "tamaño excesivo", solicitud: aplicacionbolsa.SolicitudListadoPublico{Tamano: 25}},
		{nombre: "página excesiva", solicitud: aplicacionbolsa.SolicitudListadoPublico{Pagina: 501}},
		{nombre: "texto excesivo", solicitud: aplicacionbolsa.SolicitudListadoPublico{Texto: strings.Repeat("a", 101)}},
		{nombre: "tipo no canónico", solicitud: aplicacionbolsa.SolicitudListadoPublico{Tipo: " tipo"}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			_, err := servicio.Listar(context.Background(), prueba.solicitud)
			if !errors.Is(err, aplicacionbolsa.ErrFiltroPublicoInvalido) {
				t.Fatalf("error = %v para %#v", err, prueba.solicitud)
			}
		})
	}
}

func TestDetalleDemoSeparaPublicacionRealYPlazoSinteticoAbierto(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Date(2026, 7, 19, 8, 0, 0, 0, time.UTC))
	detalle, err := servicio.Obtener(context.Background(), "bolsa-operario-diputacion-2026")
	if err != nil {
		t.Fatal(err)
	}
	if len(detalle.Plazos) != 1 || detalle.Convocatoria.Estado.Clave != "inscripcion" ||
		!detalle.Convocatoria.PublicadaEn.Equal(time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)) ||
		len(detalle.Requisitos) != 6 || len(detalle.Documentos) != 2 ||
		!strings.Contains(detalle.Descripcion, "escenario sintético rotulado como DEMO") {
		t.Fatalf("detalle DEMO inesperado: %#v", detalle)
	}
}

func TestObtenerRespetaContextoCanceladoAntesDeFuente(t *testing.T) {
	servicio := servicioPublicoPrueba(t, time.Now().UTC())
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := servicio.Obtener(ctx, "bolsa-operario-diputacion-2026"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

type fuenteDocumentoNoPublicable struct {
	base puertosbolsa.ConsultaConvocatoriasPublicas
}

func (f fuenteDocumentoNoPublicable) BuscarPublicadas(ctx context.Context, filtro puertosbolsa.FiltroConvocatoriasPublicas) (puertosbolsa.PaginaConvocatorias, error) {
	return f.base.BuscarPublicadas(ctx, filtro)
}

func (f fuenteDocumentoNoPublicable) ObtenerPublicada(ctx context.Context, id string) (puertosbolsa.DetalleConvocatoria, error) {
	detalle, err := f.base.ObtenerPublicada(ctx, id)
	if err != nil {
		return detalle, err
	}
	for i := range detalle.Catalogos {
		if detalle.Catalogos[i].Referencia != puertosbolsa.CatalogoTiposDocumento {
			continue
		}
		for j := range detalle.Catalogos[i].Entradas {
			detalle.Catalogos[i].Entradas[j].Publicable = false
		}
	}
	return detalle, nil
}

func TestDetalleFallaCerradoSiDocumentoPierdeCatalogoPublicable(t *testing.T) {
	base, err := ficherobolsa.NuevaConsultaConvocatorias("../../../../data/demo/convocatorias_publicas.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	servicio, err := aplicacionbolsa.NuevoServicioConsultaPublica(
		fuenteDocumentoNoPublicable{base: base},
		categoriasPublicasPrueba(t),
		relojPublicoFijo{instante: time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC)},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Obtener(context.Background(), "bolsa-operario-diputacion-2026")
	if !errors.Is(err, aplicacionbolsa.ErrDatosPublicosNoConfiables) {
		t.Fatalf("error = %v", err)
	}
}
