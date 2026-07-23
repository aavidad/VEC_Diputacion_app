package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/aplicacion"
	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
)

type servicioHTTPPrueba struct {
	listar   func(context.Context, aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error)
	llamadas atomic.Int32
}

type escritorBloqueadoPrueba struct {
	cabeceras http.Header
	entro     chan struct{}
	liberar   <-chan struct{}
	longitud  int
	capacidad int
}

func (e *escritorBloqueadoPrueba) Header() http.Header { return e.cabeceras }
func (*escritorBloqueadoPrueba) WriteHeader(int)       {}
func (e *escritorBloqueadoPrueba) Write(contenido []byte) (int, error) {
	e.longitud = len(contenido)
	e.capacidad = cap(contenido)
	e.entro <- struct{}{}
	<-e.liberar
	return len(contenido), nil
}

func (s *servicioHTTPPrueba) Listar(ctx context.Context, solicitud aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
	s.llamadas.Add(1)
	if s.listar != nil {
		return s.listar(ctx, solicitud)
	}
	return aplicacionbolsa.ListadoConvocatoriasPublicas{
		Esquema: "vec.bolsa.publico.convocatorias.v2",
		Convocatorias: []aplicacionbolsa.ResumenConvocatoriaPublica{{
			IdentificadorPublico: "auxiliares-2026", Titulo: "Auxiliares",
		}},
	}, nil
}

func (s *servicioHTTPPrueba) Obtener(context.Context, string) (aplicacionbolsa.DetalleConvocatoriaPublica, error) {
	return aplicacionbolsa.DetalleConvocatoriaPublica{
		Esquema: "vec.bolsa.publico.convocatoria.v2", Descripcion: "Descripción pública.",
		Documentos: []aplicacionbolsa.DocumentoPublico{},
	}, nil
}

func (s *servicioHTTPPrueba) ListarCategorias(context.Context) (aplicacionbolsa.DirectorioCategoriasPublicas, error) {
	return aplicacionbolsa.DirectorioCategoriasPublicas{
		Esquema:    "vec.bolsa.publico.categorias.v1",
		Categorias: []aplicacionbolsa.CategoriaDirectorioPublico{},
	}, nil
}

func TestHTTPPublicoRutasMetodosHEADYSanitizacion(t *testing.T) {
	servicio := &servicioHTTPPrueba{}
	handler := nuevoHandler(servicio, 2, 4, time.Second)
	pruebas := []struct {
		metodo string
		ruta   string
		estado int
	}{
		{http.MethodGet, RutaConvocatorias, http.StatusOK},
		{http.MethodGet, RutaConvocatorias + "/auxiliares-2026", http.StatusOK},
		{http.MethodGet, RutaCategorias, http.StatusOK},
		{http.MethodPost, RutaConvocatorias, http.StatusMethodNotAllowed},
		{http.MethodGet, RutaConvocatorias + "?tipo=a&tipo=b", http.StatusBadRequest},
		{http.MethodGet, RutaConvocatorias + "?interno=true", http.StatusBadRequest},
		{http.MethodGet, RutaConvocatorias + "/persona/expediente", http.StatusNotFound},
		{http.MethodGet, "/api/interno/personas", http.StatusNotFound},
	}
	for _, prueba := range pruebas {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(prueba.metodo, prueba.ruta, nil)
		req.Header.Set("X-VEC-Subject", "identidad-que-no-debe-usarse")
		handler.ServeHTTP(rec, req)
		if rec.Code != prueba.estado {
			t.Fatalf("%s %s = %d, cuerpo=%s", prueba.metodo, prueba.ruta, rec.Code, rec.Body.String())
		}
		for _, prohibido := range []string{"identidad-que-no-debe-usarse", "referencia_agregado", `"id":`, "dni"} {
			if strings.Contains(strings.ToLower(rec.Body.String()), strings.ToLower(prohibido)) {
				t.Fatalf("%s expuso %q: %s", prueba.ruta, prohibido, rec.Body.String())
			}
		}
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, RutaConvocatorias, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD = %d cuerpo=%q", rec.Code, rec.Body.String())
	}
	for nombre, contiene := range map[string]string{
		"Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
		"Content-Security-Policy": "default-src 'none'", "Referrer-Policy": "no-referrer",
	} {
		if !strings.Contains(rec.Header().Get(nombre), contiene) {
			t.Fatalf("%s = %q", nombre, rec.Header().Get(nombre))
		}
	}

	req := httptest.NewRequest(http.MethodGet, RutaConvocatorias+"/auxiliares-2026", nil)
	req.URL.RawPath = RutaConvocatorias + "/auxiliares%2d2026"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("RawPath no canónico = %d", rec.Code)
	}
}

func TestHTTPPublicoCancelaOperacionYReservaLimpieza(t *testing.T) {
	if duracionMaximaOperacionPublica+reservaLimpiezaPublica != duracionMaximaRetencionCupo ||
		duracionMaximaRetencionCupo+presupuestoEscrituraPublica != duracionMaximaPeticionPublica ||
		reservaLimpiezaPublica < postgresqlcompartido.DuracionMaximaReversion ||
		maximoRespuestasConcurrentes < maximoOperacionesConcurrentes ||
		maximoRespuestasConcurrentes*maximoBytesRespuestaPublica != presupuestoRespuestasPublicas {
		t.Fatalf(
			"presupuestos incompatibles: operación=%s limpieza=%s escritura=%s petición=%s",
			duracionMaximaOperacionPublica, reservaLimpiezaPublica,
			presupuestoEscrituraPublica, duracionMaximaPeticionPublica,
		)
	}
	const limpieza = 10 * time.Millisecond
	servicio := &servicioHTTPPrueba{
		listar: func(ctx context.Context, _ aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
			<-ctx.Done()
			time.Sleep(limpieza)
			return aplicacionbolsa.ListadoConvocatoriasPublicas{}, ctx.Err()
		},
	}
	handler := nuevoHandler(servicio, 1, 2, 10*time.Millisecond)
	rec := httptest.NewRecorder()
	inicio := time.Now()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	if rec.Code != http.StatusGatewayTimeout || time.Since(inicio) > time.Second {
		t.Fatalf("operación y limpieza = %d en %s: %s", rec.Code, time.Since(inicio), rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"codigo":"tiempo_operacion_agotado"`) ||
		strings.Contains(strings.ToLower(rec.Body.String()), "postgres") {
		t.Fatalf("cuerpo 504 no saneado: %s", rec.Body.String())
	}
	if servicio.llamadas.Load() != 1 {
		t.Fatalf("llamadas = %d", servicio.llamadas.Load())
	}
}

func TestHTTPPublicoRechazaServicioConPunteroNuloTipado(t *testing.T) {
	var servicio *servicioHTTPPrueba
	handler := nuevoHandler(servicio, 1, 2, time.Second)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("servicio nulo tipado = %d", rec.Code)
	}
}

func TestHTTPPublicoNoConvierteCancelacionDelClienteEnErrorInterno(t *testing.T) {
	servicio := &servicioHTTPPrueba{
		listar: func(ctx context.Context, _ aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
			return aplicacionbolsa.ListadoConvocatoriasPublicas{}, ctx.Err()
		},
	}
	handler := nuevoHandler(servicio, 1, 2, time.Second)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil).WithContext(ctx))
	if rec.Code != http.StatusRequestTimeout {
		t.Fatalf("cancelación del cliente = %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHTTPPublicoRechazaAntesDeInvocarAlAgotarCupos(t *testing.T) {
	entrada := make(chan struct{}, 1)
	liberar := make(chan struct{})
	servicio := &servicioHTTPPrueba{
		listar: func(ctx context.Context, _ aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
			entrada <- struct{}{}
			select {
			case <-liberar:
				return aplicacionbolsa.ListadoConvocatoriasPublicas{Esquema: "vec.bolsa.publico.convocatorias.v2"}, nil
			case <-ctx.Done():
				return aplicacionbolsa.ListadoConvocatoriasPublicas{}, ctx.Err()
			}
		},
	}
	handler := nuevoHandler(servicio, 1, 2, time.Second)
	primera := httptest.NewRecorder()
	terminada := make(chan struct{})
	go func() {
		defer close(terminada)
		handler.ServeHTTP(primera, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	}()
	<-entrada

	segunda := httptest.NewRecorder()
	handler.ServeHTTP(segunda, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	if segunda.Code != http.StatusTooManyRequests || segunda.Header().Get("Retry-After") != "1" {
		t.Fatalf("segundo acceso = %d, Retry-After=%q", segunda.Code, segunda.Header().Get("Retry-After"))
	}
	if servicio.llamadas.Load() != 1 {
		t.Fatalf("la petición rechazada invocó el servicio: %d", servicio.llamadas.Load())
	}
	tercera := httptest.NewRecorder()
	handler.ServeHTTP(tercera, httptest.NewRequest(http.MethodHead, RutaConvocatorias, nil))
	if tercera.Code != http.StatusTooManyRequests || tercera.Body.Len() != 0 {
		t.Fatalf("HEAD sin cupo = %d, cuerpo=%q", tercera.Code, tercera.Body.String())
	}
	if servicio.llamadas.Load() != 1 {
		t.Fatalf("HEAD rechazada invocó el servicio: %d", servicio.llamadas.Load())
	}
	close(liberar)
	<-terminada
	if primera.Code != http.StatusOK {
		t.Fatalf("primera petición = %d", primera.Code)
	}
}

func TestHTTPPublicoSeparaCuposServicioYRespuesta(t *testing.T) {
	servicio := &servicioHTTPPrueba{
		listar: func(context.Context, aplicacionbolsa.SolicitudListadoPublico) (aplicacionbolsa.ListadoConvocatoriasPublicas, error) {
			return aplicacionbolsa.ListadoConvocatoriasPublicas{}, nil
		},
	}
	handler := nuevoHandler(servicio, 1, 2, time.Second)
	liberar := make(chan struct{})
	finalizadas := make(chan struct{}, 2)
	for indice := 0; indice < 2; indice++ {
		escritor := &escritorBloqueadoPrueba{
			cabeceras: make(http.Header), entro: make(chan struct{}, 1), liberar: liberar,
		}
		go func() {
			defer func() { finalizadas <- struct{}{} }()
			handler.ServeHTTP(escritor, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
		}()
		<-escritor.entro
	}
	if servicio.llamadas.Load() != 2 {
		t.Fatalf("el escritor lento retuvo el cupo de servicio: llamadas=%d", servicio.llamadas.Load())
	}

	rechazada := httptest.NewRecorder()
	handler.ServeHTTP(rechazada, httptest.NewRequest(http.MethodGet, RutaConvocatorias, nil))
	if rechazada.Code != http.StatusTooManyRequests || rechazada.Header().Get("Retry-After") != "1" {
		t.Fatalf("tope de respuestas = %d, Retry-After=%q", rechazada.Code, rechazada.Header().Get("Retry-After"))
	}
	if servicio.llamadas.Load() != 2 {
		t.Fatalf("el rechazo por respuestas invocó el servicio: %d", servicio.llamadas.Load())
	}
	close(liberar)
	<-finalizadas
	<-finalizadas
}

func TestTechoRespuestaPublicaIncluyeFacetasYMaximosContractuales(t *testing.T) {
	valor := valorCatalogoMaximoPrueba()
	categoriasResumen := make([]aplicacionbolsa.ReferenciaCategoriaPublica, 128)
	for indice := range categoriasResumen {
		categoriasResumen[indice] = aplicacionbolsa.ReferenciaCategoriaPublica{Clave: valor.Clave, Version: valor.Version}
	}
	plazo := plazoMaximoPrueba(valor)
	resumen := aplicacionbolsa.ResumenConvocatoriaPublica{
		IdentificadorPublico: strings.Repeat("a", 80), Version: strings.Repeat("v", 160),
		HuellaSHA256: strings.Repeat("a", 64), Titulo: strings.Repeat("<", 180),
		Resumen: strings.Repeat("<", 500), Tipo: valor, Estado: valor,
		Categorias: categoriasResumen, PlazoDestacado: &plazo,
		NumeroRequisitos: 256, NumeroDocumentos: 256, NumeroAyudas: 128,
		PublicadaEn:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ActualizadaEn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	tipos, estados := make([]aplicacionbolsa.ValorCatalogoPublico, 256), make([]aplicacionbolsa.ValorCatalogoPublico, 256)
	for indice := range tipos {
		tipos[indice], estados[indice] = valor, valor
	}
	caracteristicas := make([]aplicacionbolsa.FacetaCategoriaPublica, 1_024)
	for indice := range caracteristicas {
		caracteristicas[indice] = aplicacionbolsa.FacetaCategoriaPublica{
			Clave: valor.Clave, Version: 1, Etiqueta: valor.Etiqueta,
			Descripcion: valor.Descripcion, Semantica: valor.Semantica,
			NumeroResultados: 12_000,
		}
	}
	resumenes := make([]aplicacionbolsa.ResumenConvocatoriaPublica, 24)
	for indice := range resumenes {
		resumenes[indice] = resumen
	}
	listado := aplicacionbolsa.ListadoConvocatoriasPublicas{
		Esquema: "vec.bolsa.publico.convocatorias.v2",
		Facetas: aplicacionbolsa.FacetasConvocatorias{
			Tipos: tipos, Estados: estados, Categorias: caracteristicas,
		},
		Paginacion:    aplicacionbolsa.PaginacionPublica{Pagina: 500, Tamano: 24, Total: 12_000, Paginas: 500},
		Convocatorias: resumenes,
	}
	detalle := detalleMaximoPrueba(resumen, plazo, valor)
	directorio := directorioCategoriasMaximoPrueba(valor)
	for nombre, respuesta := range map[string]any{
		"listado": listado, "detalle": detalle, "categorias": directorio,
	} {
		contenido, err := json.Marshal(respuesta)
		if err != nil {
			t.Fatalf("codificar %s: %v", nombre, err)
		}
		t.Logf("%s máximo: %.2f MiB", nombre, float64(len(contenido))/(1<<20))
		if len(contenido) > maximoBytesRespuestaPublica {
			t.Fatalf("%s ocupa %d bytes; techo %d", nombre, len(contenido), maximoBytesRespuestaPublica)
		}
	}
	medirPresupuestoCodificacionConcurrente(t, func() aplicacionbolsa.DetalleConvocatoriaPublica {
		valorIndependiente := valorCatalogoMaximoPrueba()
		categoriasIndependientes := make([]aplicacionbolsa.ReferenciaCategoriaPublica, 128)
		for indice := range categoriasIndependientes {
			categoriasIndependientes[indice] = aplicacionbolsa.ReferenciaCategoriaPublica{
				Clave: valorIndependiente.Clave, Version: valorIndependiente.Version,
			}
		}
		plazoIndependiente := plazoMaximoPrueba(valorIndependiente)
		resumenIndependiente := resumen
		resumenIndependiente.Categorias = categoriasIndependientes
		resumenIndependiente.Tipo = valorIndependiente
		resumenIndependiente.Estado = valorIndependiente
		resumenIndependiente.PlazoDestacado = &plazoIndependiente
		return detalleMaximoPrueba(resumenIndependiente, plazoIndependiente, valorIndependiente)
	})
}

func medirPresupuestoCodificacionConcurrente(
	t *testing.T,
	construir func() aplicacionbolsa.DetalleConvocatoriaPublica,
) {
	t.Helper()
	liberar := make(chan struct{})
	errores := make(chan error, maximoRespuestasConcurrentes)
	escritores := make([]*escritorBloqueadoPrueba, maximoRespuestasConcurrentes)
	respuestas := make([]aplicacionbolsa.DetalleConvocatoriaPublica, maximoRespuestasConcurrentes)
	var bytesObjetos uint64
	for indice := range escritores {
		respuestas[indice] = construir()
		bytesObjetos += bytesDinamicosConservadores(reflect.ValueOf(respuestas[indice]))
		escritores[indice] = &escritorBloqueadoPrueba{
			cabeceras: make(http.Header), entro: make(chan struct{}, 1), liberar: liberar,
		}
		go func(escritor *escritorBloqueadoPrueba, respuesta aplicacionbolsa.DetalleConvocatoriaPublica) {
			errores <- json.NewEncoder(escritor).Encode(respuesta)
		}(escritores[indice], respuestas[indice])
	}
	var bytesBuffers uint64
	for _, escritor := range escritores {
		<-escritor.entro
		if escritor.longitud > maximoBytesRespuestaPublica {
			t.Fatalf("respuesta codificada = %d; techo = %d", escritor.longitud, maximoBytesRespuestaPublica)
		}
		bytesBuffers += uint64(escritor.capacidad)
	}
	bytesTotales := bytesBuffers + bytesObjetos
	t.Logf(
		"%d detalles máximos simultáneos: buffers=%.2f MiB, objetos conservadores=%.2f MiB, total=%.2f MiB",
		maximoRespuestasConcurrentes, float64(bytesBuffers)/(1<<20),
		float64(bytesObjetos)/(1<<20), float64(bytesTotales)/(1<<20),
	)
	if bytesTotales > presupuestoRespuestasPublicas {
		t.Fatalf("buffers + objetos = %d; presupuesto = %d", bytesTotales, presupuestoRespuestasPublicas)
	}
	close(liberar)
	for range escritores {
		if err := <-errores; err != nil {
			t.Fatalf("codificación bloqueada: %v", err)
		}
	}
}

// bytesDinamicosConservadores suma la capacidad completa de slices y la
// longitud de cada string, aunque varias vistas compartan el mismo respaldo.
// Por ello es deliberadamente un techo y no una estimación optimista del heap.
func bytesDinamicosConservadores(valor reflect.Value) uint64 {
	if !valor.IsValid() {
		return 0
	}
	switch valor.Kind() {
	case reflect.Interface:
		if valor.IsNil() {
			return 0
		}
		return bytesDinamicosConservadores(valor.Elem())
	case reflect.Pointer:
		if valor.IsNil() || valor.Type().Elem().PkgPath() == "time" {
			return 0
		}
		return uint64(valor.Type().Elem().Size()) + bytesDinamicosConservadores(valor.Elem())
	case reflect.String:
		return uint64(valor.Len())
	case reflect.Slice:
		bytes := uint64(valor.Cap()) * uint64(valor.Type().Elem().Size())
		for indice := 0; indice < valor.Len(); indice++ {
			bytes += bytesDinamicosConservadores(valor.Index(indice))
		}
		return bytes
	case reflect.Array:
		var bytes uint64
		for indice := 0; indice < valor.Len(); indice++ {
			bytes += bytesDinamicosConservadores(valor.Index(indice))
		}
		return bytes
	case reflect.Struct:
		if valor.Type().PkgPath() == "time" {
			return 0
		}
		var bytes uint64
		for indice := 0; indice < valor.NumField(); indice++ {
			bytes += bytesDinamicosConservadores(valor.Field(indice))
		}
		return bytes
	default:
		return 0
	}
}

func valorCatalogoMaximoPrueba() aplicacionbolsa.ValorCatalogoPublico {
	return aplicacionbolsa.ValorCatalogoPublico{
		Clave: strings.Repeat("a", 80), Version: 1,
		Etiqueta: strings.Repeat("<", 120), Descripcion: strings.Repeat("<", 600),
		Semantica: strings.Repeat("s", 80),
	}
}

func plazoMaximoPrueba(valor aplicacionbolsa.ValorCatalogoPublico) aplicacionbolsa.PlazoPublico {
	return aplicacionbolsa.PlazoPublico{
		Referencia: strings.Repeat("r", 160), Tipo: valor,
		Titulo: strings.Repeat("<", 180), Descripcion: strings.Repeat("<", 1_000),
		AbreEn:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CierraEn:  time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		Situacion: "abierto", Etiqueta: "Plazo abierto", Semantica: "exito",
	}
}

func detalleMaximoPrueba(
	resumen aplicacionbolsa.ResumenConvocatoriaPublica,
	plazo aplicacionbolsa.PlazoPublico,
	valor aplicacionbolsa.ValorCatalogoPublico,
) aplicacionbolsa.DetalleConvocatoriaPublica {
	requisitos := make([]aplicacionbolsa.RequisitoPublico, 256)
	documentos := make([]aplicacionbolsa.DocumentoPublico, 256)
	ayudas := make([]aplicacionbolsa.AyudaPublica, 128)
	plazos := make([]aplicacionbolsa.PlazoPublico, 64)
	diccionarioCategorias := make([]aplicacionbolsa.CategoriaDiccionarioPublico, len(resumen.Categorias))
	for indice := range diccionarioCategorias {
		diccionarioCategorias[indice] = aplicacionbolsa.CategoriaDiccionarioPublico{
			CatalogoCategorias: resumen.CatalogoCategorias,
			Clave:              valor.Clave, Version: valor.Version, Etiqueta: valor.Etiqueta,
			Descripcion: valor.Descripcion, Semantica: valor.Semantica,
		}
	}
	for indice := range requisitos {
		requisitos[indice] = aplicacionbolsa.RequisitoPublico{
			Referencia: strings.Repeat("r", 160), Orden: indice + 1,
			Titulo: strings.Repeat("<", 180), Descripcion: strings.Repeat("<", 3_000), Obligatorio: true,
		}
		documentos[indice] = aplicacionbolsa.DocumentoPublico{
			Referencia: strings.Repeat("r", 160), Tipo: valor, Orden: indice + 1,
			Titulo: strings.Repeat("<", 180), Descripcion: strings.Repeat("<", 1_000),
			Formato: strings.Repeat("f", 80), URL: "/bolsa/documentos/" + strings.Repeat("a", 219),
			PublicadoEn: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		}
	}
	for indice := range ayudas {
		ayudas[indice] = aplicacionbolsa.AyudaPublica{
			Referencia: strings.Repeat("r", 160), Categoria: valor, Orden: indice + 1,
			Pregunta: strings.Repeat("<", 300), Respuesta: strings.Repeat("<", 5_000),
		}
	}
	for indice := range plazos {
		plazos[indice] = plazo
	}
	return aplicacionbolsa.DetalleConvocatoriaPublica{
		Esquema: "vec.bolsa.publico.convocatoria.v2", Convocatoria: resumen,
		DiccionarioCategorias: diccionarioCategorias,
		Descripcion:           strings.Repeat("<", 12_000), Plazos: plazos,
		Requisitos: requisitos, Documentos: documentos, Ayuda: ayudas,
	}
}

func directorioCategoriasMaximoPrueba(valor aplicacionbolsa.ValorCatalogoPublico) aplicacionbolsa.DirectorioCategoriasPublicas {
	categorias := make([]aplicacionbolsa.CategoriaDirectorioPublico, 1_024)
	for indice := range categorias {
		categorias[indice] = aplicacionbolsa.CategoriaDirectorioPublico{
			Clave: valor.Clave, Version: 1, Etiqueta: valor.Etiqueta,
			Descripcion: valor.Descripcion, Semantica: valor.Semantica, Orden: indice + 1,
			Area: strings.Repeat("a", 80), AreaEtiqueta: strings.Repeat("<", 120),
			Suscribible: true, NumeroConvocatorias: 12_000, NumeroPlazosAbiertos: 12_000,
		}
	}
	return aplicacionbolsa.DirectorioCategoriasPublicas{
		Esquema: "vec.bolsa.publico.categorias.v1",
		Catalogo: aplicacionbolsa.ReferenciaCatalogoCategoriasPublico{
			CatalogoID: strings.Repeat("a", 128), Version: 1,
			HuellaSHA256:           strings.Repeat("a", 64),
			HuellaProyeccionSHA256: strings.Repeat("b", 64), Total: 1_024,
		},
		Categorias: categorias,
	}
}
