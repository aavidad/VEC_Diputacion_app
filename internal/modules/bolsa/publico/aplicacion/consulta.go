// Package aplicacion implementa casos de uso exclusivamente anónimos.
package aplicacion

import (
	"context"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/publico/dominio"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/publico/puertos"
)

var (
	ErrServicioConsultaPublicaInvalido = errors.New("bolsa: servicio de consulta publica invalido")
	ErrFiltroPublicoInvalido           = errors.New("bolsa: filtro publico invalido")
	ErrDatosPublicosNoConfiables       = errors.New("bolsa: datos publicos no confiables")
)

const (
	TamanoPaginaPublicaPredeterminado = 12
	TamanoPaginaPublicaMaximo         = 24
	PaginaPublicaMaxima               = 500
	LongitudTextoPublicoMaxima        = 100
)

var (
	patronFiltroCatalogo       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$`)
	patronIdentificadorPublico = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,79}$`)
	patronHuellaSHA256         = regexp.MustCompile(`^[a-f0-9]{64}$`)
)

type RelojConsultaPublica interface {
	Ahora() time.Time
}

type RelojSistemaConsultaPublica struct{}

func (RelojSistemaConsultaPublica) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

type ServicioConsultaPublica struct {
	fuente      puertosbolsa.ConsultaConvocatoriasPublicas
	categorias  puertosbolsa.ConsultaCategoriasPublicas
	consistente puertosbolsa.ConsultaPublicaConsistente
	reloj       RelojConsultaPublica
}

func NuevoServicioConsultaPublica(
	fuente puertosbolsa.ConsultaConvocatoriasPublicas,
	categorias puertosbolsa.ConsultaCategoriasPublicas,
	reloj RelojConsultaPublica,
) (*ServicioConsultaPublica, error) {
	if fuente == nil || categorias == nil || reloj == nil {
		return nil, ErrServicioConsultaPublicaInvalido
	}
	return &ServicioConsultaPublica{fuente: fuente, categorias: categorias, reloj: reloj}, nil
}

func NuevoServicioConsultaPublicaConsistente(
	consulta puertosbolsa.ConsultaPublicaConsistente,
	reloj RelojConsultaPublica,
) (*ServicioConsultaPublica, error) {
	if consulta == nil || reloj == nil {
		return nil, ErrServicioConsultaPublicaInvalido
	}
	return &ServicioConsultaPublica{
		fuente: consulta, categorias: consulta, consistente: consulta, reloj: reloj,
	}, nil
}

// ValidarConfiguracion coteja de forma anticipada la instantanea profesional
// con todas las convocatorias publicadas. El bootstrap debe ejecutarlo antes
// de montar las rutas: una referencia desconocida o una version/huella
// diferente impide arrancar en vez de convertirse despues en un error 500.
func (s *ServicioConsultaPublica) ValidarConfiguracion(ctx context.Context) error {
	if ctx == nil || s == nil || s.fuente == nil || s.categorias == nil || s.reloj == nil {
		return ErrServicioConsultaPublicaInvalido
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instante, err := instanteCanonico(s.reloj.Ahora())
	if err != nil {
		return err
	}
	if s.consistente != nil {
		return s.consistente.ValidarConfiguracionPublica(ctx, instante)
	}
	catalogoCategorias, err := s.categorias.ObtenerPublicadas(ctx, instante)
	if err != nil {
		return err
	}

	totalEsperado := -1
	desplazamiento := 0
	identificadores := make(map[string]struct{})
	for {
		pagina, err := s.fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{
			Instante: instante, Limite: TamanoPaginaPublicaMaximo, Desplazamiento: desplazamiento,
		})
		if err != nil {
			return err
		}
		indice, err := nuevoIndiceCatalogos(pagina.Catalogos, catalogoCategorias)
		if err != nil || validarFuente(pagina.Fuente) != nil ||
			!fuentesCoinciden(catalogoCategorias.Fuente, pagina.Fuente) ||
			validarConteosCategorias(pagina.ConteosCategorias, indice) != nil || pagina.Total < 0 ||
			pagina.Total > PaginaPublicaMaxima*TamanoPaginaPublicaMaximo ||
			len(pagina.Convocatorias) > TamanoPaginaPublicaMaximo {
			return ErrDatosPublicosNoConfiables
		}
		if totalEsperado < 0 {
			totalEsperado = pagina.Total
		} else if pagina.Total != totalEsperado {
			return ErrDatosPublicosNoConfiables
		}
		for _, convocatoria := range pagina.Convocatorias {
			resumen, err := proyectarResumen(convocatoria, indice, instante)
			if err != nil {
				return ErrDatosPublicosNoConfiables
			}
			if _, existe := identificadores[resumen.IdentificadorPublico]; existe {
				return ErrDatosPublicosNoConfiables
			}
			identificadores[resumen.IdentificadorPublico] = struct{}{}
		}
		desplazamiento += len(pagina.Convocatorias)
		if desplazamiento >= totalEsperado {
			return nil
		}
		if len(pagina.Convocatorias) == 0 {
			return ErrDatosPublicosNoConfiables
		}
	}
}

type SolicitudListadoPublico struct {
	Texto            string
	Tipo             string
	Categoria        string
	Estado           string
	SoloPlazoAbierto bool
	Pagina           int
	Tamano           int
}

type ValorCatalogoPublico struct {
	Clave       string `json:"clave"`
	Version     int    `json:"version"`
	Etiqueta    string `json:"etiqueta"`
	Descripcion string `json:"descripcion,omitempty"`
	Semantica   string `json:"semantica"`
}

type FacetaCategoriaPublica struct {
	Clave            string `json:"clave"`
	Version          int    `json:"version"`
	Etiqueta         string `json:"etiqueta"`
	Descripcion      string `json:"descripcion,omitempty"`
	Semantica        string `json:"semantica"`
	NumeroResultados int    `json:"numero_resultados"`
}

type FacetasConvocatorias struct {
	Tipos      []ValorCatalogoPublico   `json:"tipos"`
	Categorias []FacetaCategoriaPublica `json:"categorias"`
	Estados    []ValorCatalogoPublico   `json:"estados"`
}

type PaginacionPublica struct {
	Pagina  int `json:"pagina"`
	Tamano  int `json:"tamano"`
	Total   int `json:"total"`
	Paginas int `json:"paginas"`
}

type FuentePublica struct {
	Revision      string    `json:"revision"`
	ActualizadaEn time.Time `json:"actualizada_en"`
	Demostracion  bool      `json:"demostracion"`
	Aviso         string    `json:"aviso,omitempty"`
}

type PlazoPublico struct {
	Referencia  string               `json:"referencia"`
	Tipo        ValorCatalogoPublico `json:"tipo"`
	Titulo      string               `json:"titulo"`
	Descripcion string               `json:"descripcion,omitempty"`
	AbreEn      time.Time            `json:"abre_en"`
	CierraEn    time.Time            `json:"cierra_en"`
	Situacion   string               `json:"situacion"`
	Etiqueta    string               `json:"etiqueta_situacion"`
	Semantica   string               `json:"semantica_situacion"`
}

type ResumenConvocatoriaPublica struct {
	IdentificadorPublico string               `json:"identificador_publico"`
	Version              string               `json:"version"`
	HuellaSHA256         string               `json:"huella_sha256"`
	Titulo               string               `json:"titulo"`
	Resumen              string               `json:"resumen"`
	Tipo                 ValorCatalogoPublico `json:"tipo"`
	Estado               ValorCatalogoPublico `json:"estado"`
	// Categorias contiene referencias nominales. Etiqueta, descripción y
	// semántica se publican una sola vez en Facetas.Categorias y se resuelven
	// por la pareja exacta clave+versión.
	Categorias       []ReferenciaCategoriaPublica `json:"categorias"`
	PlazoDestacado   *PlazoPublico                `json:"plazo_destacado,omitempty"`
	NumeroRequisitos int                          `json:"numero_requisitos"`
	NumeroDocumentos int                          `json:"numero_documentos"`
	NumeroAyudas     int                          `json:"numero_ayudas"`
	PublicadaEn      time.Time                    `json:"publicada_en"`
	ActualizadaEn    time.Time                    `json:"actualizada_en"`
}

type ReferenciaCategoriaPublica struct {
	Clave   string `json:"clave"`
	Version int    `json:"version"`
}

type ListadoConvocatoriasPublicas struct {
	Esquema       string                       `json:"esquema"`
	Fuente        FuentePublica                `json:"fuente"`
	Facetas       FacetasConvocatorias         `json:"facetas"`
	Paginacion    PaginacionPublica            `json:"paginacion"`
	Convocatorias []ResumenConvocatoriaPublica `json:"convocatorias"`
}

type ReferenciaCatalogoCategoriasPublico struct {
	Referencia   string `json:"referencia"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
	Total        int    `json:"total"`
}

type CategoriaDirectorioPublico struct {
	Clave                string `json:"clave"`
	Version              int    `json:"version"`
	Etiqueta             string `json:"etiqueta"`
	Descripcion          string `json:"descripcion,omitempty"`
	Semantica            string `json:"semantica"`
	Orden                int    `json:"orden"`
	Area                 string `json:"area"`
	AreaEtiqueta         string `json:"area_etiqueta"`
	Suscribible          bool   `json:"suscribible"`
	NumeroConvocatorias  int    `json:"numero_convocatorias"`
	NumeroPlazosAbiertos int    `json:"numero_plazos_abiertos"`
}

type DirectorioCategoriasPublicas struct {
	Esquema    string                              `json:"esquema"`
	Fuente     FuentePublica                       `json:"fuente"`
	Catalogo   ReferenciaCatalogoCategoriasPublico `json:"catalogo"`
	Categorias []CategoriaDirectorioPublico        `json:"categorias"`
}

type RequisitoPublico struct {
	Referencia  string `json:"referencia"`
	Orden       int    `json:"orden"`
	Titulo      string `json:"titulo"`
	Descripcion string `json:"descripcion"`
	Obligatorio bool   `json:"obligatorio"`
}

type DocumentoPublico struct {
	Referencia  string               `json:"referencia"`
	Tipo        ValorCatalogoPublico `json:"tipo"`
	Orden       int                  `json:"orden"`
	Titulo      string               `json:"titulo"`
	Descripcion string               `json:"descripcion"`
	Formato     string               `json:"formato"`
	URL         string               `json:"url"`
	PublicadoEn time.Time            `json:"publicado_en"`
}

type AyudaPublica struct {
	Referencia string               `json:"referencia"`
	Categoria  ValorCatalogoPublico `json:"categoria"`
	Orden      int                  `json:"orden"`
	Pregunta   string               `json:"pregunta"`
	Respuesta  string               `json:"respuesta"`
}

type DetalleConvocatoriaPublica struct {
	Esquema               string                     `json:"esquema"`
	Fuente                FuentePublica              `json:"fuente"`
	Convocatoria          ResumenConvocatoriaPublica `json:"convocatoria"`
	DiccionarioCategorias []ValorCatalogoPublico     `json:"diccionario_categorias"`
	Descripcion           string                     `json:"descripcion"`
	Plazos                []PlazoPublico             `json:"plazos"`
	Requisitos            []RequisitoPublico         `json:"requisitos"`
	Documentos            []DocumentoPublico         `json:"documentos"`
	Ayuda                 []AyudaPublica             `json:"ayuda"`
}

func (s *ServicioConsultaPublica) Listar(ctx context.Context, solicitud SolicitudListadoPublico) (ListadoConvocatoriasPublicas, error) {
	if ctx == nil || s == nil || s.fuente == nil || s.categorias == nil || s.reloj == nil {
		return ListadoConvocatoriasPublicas{}, ErrServicioConsultaPublicaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ListadoConvocatoriasPublicas{}, err
	}
	filtro, pagina, tamano, err := s.prepararFiltro(solicitud)
	if err != nil {
		return ListadoConvocatoriasPublicas{}, err
	}
	var catalogoCategorias puertosbolsa.CatalogoCategoriasPublicas
	var resultado puertosbolsa.PaginaConvocatorias
	if s.consistente != nil {
		lectura, err := s.consistente.BuscarPublicadasConCategorias(ctx, filtro)
		if err != nil {
			return ListadoConvocatoriasPublicas{}, err
		}
		catalogoCategorias, resultado = lectura.Categorias, lectura.Pagina
	} else {
		catalogoCategorias, err = s.categorias.ObtenerPublicadas(ctx, filtro.Instante)
		if err != nil {
			return ListadoConvocatoriasPublicas{}, err
		}
		resultado, err = s.fuente.BuscarPublicadas(ctx, filtro)
		if err != nil {
			return ListadoConvocatoriasPublicas{}, err
		}
	}
	if filtro.Categoria != "" && !contieneCategoriaPublica(catalogoCategorias, filtro.Categoria) {
		return ListadoConvocatoriasPublicas{}, ErrFiltroPublicoInvalido
	}
	indice, err := nuevoIndiceCatalogos(resultado.Catalogos, catalogoCategorias)
	if err != nil || validarFuente(resultado.Fuente) != nil ||
		!fuentesCoinciden(catalogoCategorias.Fuente, resultado.Fuente) ||
		resultado.Total < 0 || len(resultado.Convocatorias) > tamano ||
		validarConteosCategorias(resultado.ConteosCategorias, indice) != nil {
		return ListadoConvocatoriasPublicas{}, ErrDatosPublicosNoConfiables
	}
	resumenes := make([]ResumenConvocatoriaPublica, 0, len(resultado.Convocatorias))
	for _, convocatoria := range resultado.Convocatorias {
		resumen, err := proyectarResumen(convocatoria, indice, filtro.Instante)
		if err != nil {
			return ListadoConvocatoriasPublicas{}, err
		}
		resumenes = append(resumenes, resumen)
	}
	facetas := indice.facetas(resultado.ConteosCategorias)
	if err := validarCoberturaCategoriasResumenes(resumenes, facetas.Categorias); err != nil {
		return ListadoConvocatoriasPublicas{}, err
	}
	paginas := 0
	if resultado.Total > 0 {
		paginas = (resultado.Total + tamano - 1) / tamano
	}
	return ListadoConvocatoriasPublicas{
		Esquema:       "vec.bolsa.publico.convocatorias.v2",
		Fuente:        proyectarFuente(resultado.Fuente),
		Facetas:       facetas,
		Paginacion:    PaginacionPublica{Pagina: pagina, Tamano: tamano, Total: resultado.Total, Paginas: paginas},
		Convocatorias: resumenes,
	}, nil
}

func (s *ServicioConsultaPublica) Obtener(ctx context.Context, identificador string) (DetalleConvocatoriaPublica, error) {
	if ctx == nil || s == nil || s.fuente == nil || s.categorias == nil || s.reloj == nil {
		return DetalleConvocatoriaPublica{}, ErrServicioConsultaPublicaInvalido
	}
	if err := ctx.Err(); err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	identificador = strings.TrimSpace(identificador)
	if !patronIdentificadorPublico.MatchString(identificador) {
		return DetalleConvocatoriaPublica{}, ErrFiltroPublicoInvalido
	}
	ahora, err := instanteCanonico(s.reloj.Ahora())
	if err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	var catalogoCategorias puertosbolsa.CatalogoCategoriasPublicas
	var resultado puertosbolsa.DetalleConvocatoria
	if s.consistente != nil {
		lectura, err := s.consistente.ObtenerPublicadaConCategorias(ctx, identificador, ahora)
		if err != nil {
			return DetalleConvocatoriaPublica{}, err
		}
		catalogoCategorias, resultado = lectura.Categorias, lectura.Detalle
	} else {
		catalogoCategorias, err = s.categorias.ObtenerPublicadas(ctx, ahora)
		if err != nil {
			return DetalleConvocatoriaPublica{}, err
		}
		resultado, err = s.fuente.ObtenerPublicada(ctx, identificador)
		if err != nil {
			return DetalleConvocatoriaPublica{}, err
		}
	}
	indice, err := nuevoIndiceCatalogos(resultado.Catalogos, catalogoCategorias)
	if err != nil || validarFuente(resultado.Fuente) != nil ||
		!fuentesCoinciden(catalogoCategorias.Fuente, resultado.Fuente) {
		return DetalleConvocatoriaPublica{}, ErrDatosPublicosNoConfiables
	}
	huellaCalculada, err := huellaConvocatoriaPublica(resultado.Convocatoria)
	if err != nil || !huellasPublicasIguales(huellaCalculada, resultado.Convocatoria.HuellaSHA256) {
		return DetalleConvocatoriaPublica{}, ErrDatosPublicosNoConfiables
	}
	resumen, err := proyectarResumen(resumirDetalle(resultado.Convocatoria), indice, ahora)
	if err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	diccionarioCategorias, err := indice.diccionarioCategorias(resumen.Categorias)
	if err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	datos := resultado.Convocatoria.DatosPublicos
	plazos, err := proyectarPlazos(datos.Plazos, indice, ahora)
	if err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	documentos, err := proyectarDocumentos(datos.Documentos, indice)
	if err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	ayuda, err := proyectarAyuda(datos.Ayuda, indice)
	if err != nil {
		return DetalleConvocatoriaPublica{}, err
	}
	detalle := DetalleConvocatoriaPublica{
		Esquema:               "vec.bolsa.publico.convocatoria.v2",
		Fuente:                proyectarFuente(resultado.Fuente),
		Convocatoria:          resumen,
		DiccionarioCategorias: diccionarioCategorias,
		Descripcion:           datos.Descripcion,
		Plazos:                plazos,
		Requisitos:            proyectarRequisitos(datos.Requisitos),
		Documentos:            documentos,
		Ayuda:                 ayuda,
	}
	if detalle.Plazos == nil {
		detalle.Plazos = []PlazoPublico{}
	}
	if detalle.Requisitos == nil {
		detalle.Requisitos = []RequisitoPublico{}
	}
	if detalle.Documentos == nil {
		detalle.Documentos = []DocumentoPublico{}
	}
	if detalle.Ayuda == nil {
		detalle.Ayuda = []AyudaPublica{}
	}
	return detalle, nil
}

func (s *ServicioConsultaPublica) ListarCategorias(ctx context.Context) (DirectorioCategoriasPublicas, error) {
	if ctx == nil || s == nil || s.fuente == nil || s.categorias == nil || s.reloj == nil {
		return DirectorioCategoriasPublicas{}, ErrServicioConsultaPublicaInvalido
	}
	if err := ctx.Err(); err != nil {
		return DirectorioCategoriasPublicas{}, err
	}
	ahora, err := instanteCanonico(s.reloj.Ahora())
	if err != nil {
		return DirectorioCategoriasPublicas{}, err
	}
	var catalogo puertosbolsa.CatalogoCategoriasPublicas
	var resultado puertosbolsa.PaginaConvocatorias
	if s.consistente != nil {
		lectura, err := s.consistente.ConsultarCategoriasConConteos(ctx, ahora)
		if err != nil {
			return DirectorioCategoriasPublicas{}, err
		}
		catalogo, resultado = lectura.Categorias, lectura.Pagina
	} else {
		catalogo, err = s.categorias.ObtenerPublicadas(ctx, ahora)
		if err != nil {
			return DirectorioCategoriasPublicas{}, err
		}
		resultado, err = s.fuente.BuscarPublicadas(ctx, puertosbolsa.FiltroConvocatoriasPublicas{
			Instante: ahora, Limite: 1,
		})
		if err != nil {
			return DirectorioCategoriasPublicas{}, err
		}
	}
	indice, err := nuevoIndiceCatalogos(resultado.Catalogos, catalogo)
	if err != nil || validarFuente(resultado.Fuente) != nil || validarFuenteCategorias(catalogo.Fuente) != nil ||
		!fuentesCoinciden(catalogo.Fuente, resultado.Fuente) ||
		validarConteosCategorias(resultado.ConteosCategorias, indice) != nil {
		return DirectorioCategoriasPublicas{}, ErrDatosPublicosNoConfiables
	}
	categorias := make([]CategoriaDirectorioPublico, 0, len(catalogo.Categorias))
	for _, categoria := range catalogo.Categorias {
		conteo := resultado.ConteosCategorias[categoria.Clave]
		categorias = append(categorias, CategoriaDirectorioPublico{
			Clave:                categoria.Clave,
			Version:              categoria.Version,
			Etiqueta:             categoria.Etiqueta,
			Descripcion:          categoria.Descripcion,
			Semantica:            categoria.Semantica,
			Orden:                categoria.Orden,
			Area:                 categoria.Area,
			AreaEtiqueta:         categoria.AreaEtiqueta,
			Suscribible:          categoria.Suscribible,
			NumeroConvocatorias:  conteo.NumeroConvocatorias,
			NumeroPlazosAbiertos: conteo.NumeroPlazosAbiertos,
		})
	}
	return DirectorioCategoriasPublicas{
		Esquema: "vec.bolsa.publico.categorias.v1",
		Fuente:  proyectarFuenteCategorias(catalogo.Fuente),
		Catalogo: ReferenciaCatalogoCategoriasPublico{
			Referencia:   catalogo.ID,
			Version:      catalogo.Version,
			HuellaSHA256: catalogo.HuellaSHA256,
			Total:        len(categorias),
		},
		Categorias: categorias,
	}, nil
}

func (s *ServicioConsultaPublica) prepararFiltro(solicitud SolicitudListadoPublico) (puertosbolsa.FiltroConvocatoriasPublicas, int, int, error) {
	texto := strings.TrimSpace(solicitud.Texto)
	if len([]rune(texto)) > LongitudTextoPublicoMaxima || strings.ContainsAny(texto, "\x00\r\n") {
		return puertosbolsa.FiltroConvocatoriasPublicas{}, 0, 0, ErrFiltroPublicoInvalido
	}
	for _, valor := range []string{solicitud.Tipo, solicitud.Categoria, solicitud.Estado} {
		if valor != "" && (valor != strings.TrimSpace(valor) || !patronFiltroCatalogo.MatchString(valor)) {
			return puertosbolsa.FiltroConvocatoriasPublicas{}, 0, 0, ErrFiltroPublicoInvalido
		}
	}
	pagina := solicitud.Pagina
	if pagina == 0 {
		pagina = 1
	}
	tamano := solicitud.Tamano
	if tamano == 0 {
		tamano = TamanoPaginaPublicaPredeterminado
	}
	if pagina < 1 || pagina > PaginaPublicaMaxima || tamano < 1 || tamano > TamanoPaginaPublicaMaximo {
		return puertosbolsa.FiltroConvocatoriasPublicas{}, 0, 0, ErrFiltroPublicoInvalido
	}
	ahora, err := instanteCanonico(s.reloj.Ahora())
	if err != nil {
		return puertosbolsa.FiltroConvocatoriasPublicas{}, 0, 0, err
	}
	return puertosbolsa.FiltroConvocatoriasPublicas{
		Texto: texto, Tipo: solicitud.Tipo, Categoria: solicitud.Categoria, Estado: solicitud.Estado,
		SoloPlazoAbierto: solicitud.SoloPlazoAbierto, Instante: ahora, Limite: tamano, Desplazamiento: (pagina - 1) * tamano,
	}, pagina, tamano, nil
}

type indiceCatalogos struct {
	porCatalogo          map[string]map[string]puertosbolsa.EntradaCatalogoPublico
	ordenados            map[string][]puertosbolsa.EntradaCatalogoPublico
	referenciaCategorias dominiobolsa.ReferenciaCatalogoCategorias
}

func nuevoIndiceCatalogos(
	catalogos []puertosbolsa.CatalogoPublico,
	categorias puertosbolsa.CatalogoCategoriasPublicas,
) (indiceCatalogos, error) {
	if !patronFiltroCatalogo.MatchString(categorias.ID) || categorias.Version < 1 ||
		!patronHuellaSHA256.MatchString(categorias.HuellaSHA256) || len(categorias.Categorias) == 0 ||
		validarFuenteCategorias(categorias.Fuente) != nil {
		return indiceCatalogos{}, ErrDatosPublicosNoConfiables
	}
	catalogosCompletos := make([]puertosbolsa.CatalogoPublico, 0, len(catalogos)+1)
	catalogosCompletos = append(catalogosCompletos, catalogos...)
	catalogoProfesional := puertosbolsa.CatalogoPublico{
		Referencia: puertosbolsa.CatalogoCategoriasConvocatoria,
		Version:    categorias.Version,
		Entradas:   make([]puertosbolsa.EntradaCatalogoPublico, 0, len(categorias.Categorias)),
	}
	for _, categoria := range categorias.Categorias {
		if categoria.Version != categorias.Version || !patronFiltroCatalogo.MatchString(categoria.Area) ||
			!textoPublicoCanonico(categoria.AreaEtiqueta, 120, false) {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
		catalogoProfesional.Entradas = append(catalogoProfesional.Entradas, puertosbolsa.EntradaCatalogoPublico{
			Clave:       categoria.Clave,
			Version:     categoria.Version,
			Etiqueta:    categoria.Etiqueta,
			Descripcion: categoria.Descripcion,
			Semantica:   categoria.Semantica,
			Orden:       categoria.Orden,
			Publicable:  true,
		})
	}
	catalogosCompletos = append(catalogosCompletos, catalogoProfesional)
	indice := indiceCatalogos{
		porCatalogo: map[string]map[string]puertosbolsa.EntradaCatalogoPublico{},
		ordenados:   map[string][]puertosbolsa.EntradaCatalogoPublico{},
		referenciaCategorias: dominiobolsa.ReferenciaCatalogoCategorias{
			CatalogoID: categorias.ID, CatalogoVersion: categorias.Version, CatalogoHuellaSHA256: categorias.HuellaSHA256,
		},
	}
	for _, catalogo := range catalogosCompletos {
		if !patronFiltroCatalogo.MatchString(catalogo.Referencia) || catalogo.Version < 1 || len(catalogo.Entradas) == 0 || indice.porCatalogo[catalogo.Referencia] != nil {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
		indice.porCatalogo[catalogo.Referencia] = map[string]puertosbolsa.EntradaCatalogoPublico{}
		for _, entrada := range catalogo.Entradas {
			if !patronFiltroCatalogo.MatchString(entrada.Clave) || entrada.Version != catalogo.Version ||
				!textoPublicoCanonico(entrada.Etiqueta, 120, false) ||
				(entrada.Descripcion != "" && !textoPublicoCanonico(entrada.Descripcion, 600, true)) ||
				!patronFiltroCatalogo.MatchString(entrada.Semantica) || entrada.Orden < 1 || indice.porCatalogo[catalogo.Referencia][entrada.Clave].Clave != "" {
				return indiceCatalogos{}, ErrDatosPublicosNoConfiables
			}
			indice.porCatalogo[catalogo.Referencia][entrada.Clave] = entrada
			indice.ordenados[catalogo.Referencia] = append(indice.ordenados[catalogo.Referencia], entrada)
		}
		sort.Slice(indice.ordenados[catalogo.Referencia], func(i, j int) bool {
			a, b := indice.ordenados[catalogo.Referencia][i], indice.ordenados[catalogo.Referencia][j]
			if a.Orden == b.Orden {
				return a.Clave < b.Clave
			}
			return a.Orden < b.Orden
		})
	}
	for _, requerido := range []string{puertosbolsa.CatalogoTiposConvocatoria, puertosbolsa.CatalogoEstadosConvocatoria, puertosbolsa.CatalogoCategoriasConvocatoria, puertosbolsa.CatalogoTiposPlazo, puertosbolsa.CatalogoTiposDocumento, puertosbolsa.CatalogoCategoriasAyuda} {
		if indice.porCatalogo[requerido] == nil {
			return indiceCatalogos{}, ErrDatosPublicosNoConfiables
		}
	}
	return indice, nil
}

func (i indiceCatalogos) resolver(catalogo, clave string) (ValorCatalogoPublico, error) {
	entrada, existe := i.porCatalogo[catalogo][clave]
	if !existe || !entrada.Publicable {
		return ValorCatalogoPublico{}, ErrDatosPublicosNoConfiables
	}
	return ValorCatalogoPublico{Clave: entrada.Clave, Version: entrada.Version, Etiqueta: entrada.Etiqueta, Descripcion: entrada.Descripcion, Semantica: entrada.Semantica}, nil
}

func (i indiceCatalogos) facetas(conteos map[string]puertosbolsa.ConteoCategoriaConvocatorias) FacetasConvocatorias {
	categorias := make([]FacetaCategoriaPublica, 0, len(conteos))
	for _, entrada := range i.ordenados[puertosbolsa.CatalogoCategoriasConvocatoria] {
		conteo := conteos[entrada.Clave]
		if !entrada.Publicable || conteo.NumeroConvocatorias < 1 {
			continue
		}
		categorias = append(categorias, FacetaCategoriaPublica{
			Clave:            entrada.Clave,
			Version:          entrada.Version,
			Etiqueta:         entrada.Etiqueta,
			Descripcion:      entrada.Descripcion,
			Semantica:        entrada.Semantica,
			NumeroResultados: conteo.NumeroConvocatorias,
		})
	}
	return FacetasConvocatorias{
		Tipos:      i.valoresPublicables(puertosbolsa.CatalogoTiposConvocatoria),
		Categorias: categorias,
		Estados:    i.valoresPublicables(puertosbolsa.CatalogoEstadosConvocatoria),
	}
}

func (i indiceCatalogos) valoresPublicables(catalogo string) []ValorCatalogoPublico {
	valores := make([]ValorCatalogoPublico, 0, len(i.ordenados[catalogo]))
	for _, entrada := range i.ordenados[catalogo] {
		if entrada.Publicable {
			valores = append(valores, ValorCatalogoPublico{Clave: entrada.Clave, Version: entrada.Version, Etiqueta: entrada.Etiqueta, Descripcion: entrada.Descripcion, Semantica: entrada.Semantica})
		}
	}
	return valores
}

func (i indiceCatalogos) diccionarioCategorias(
	referencias []ReferenciaCategoriaPublica,
) ([]ValorCatalogoPublico, error) {
	resultado := make([]ValorCatalogoPublico, 0, len(referencias))
	vistas := make(map[ReferenciaCategoriaPublica]struct{}, len(referencias))
	for _, referencia := range referencias {
		if referencia.Clave == "" || referencia.Version < 1 {
			return nil, ErrDatosPublicosNoConfiables
		}
		if _, duplicada := vistas[referencia]; duplicada {
			return nil, ErrDatosPublicosNoConfiables
		}
		valor, err := i.resolver(puertosbolsa.CatalogoCategoriasConvocatoria, referencia.Clave)
		if err != nil || valor.Version != referencia.Version {
			return nil, ErrDatosPublicosNoConfiables
		}
		vistas[referencia] = struct{}{}
		resultado = append(resultado, valor)
	}
	if len(resultado) != len(referencias) {
		return nil, ErrDatosPublicosNoConfiables
	}
	return resultado, nil
}

func validarCoberturaCategoriasResumenes(
	resumenes []ResumenConvocatoriaPublica,
	facetas []FacetaCategoriaPublica,
) error {
	diccionario := make(map[ReferenciaCategoriaPublica]struct{}, len(facetas))
	for _, faceta := range facetas {
		referencia := ReferenciaCategoriaPublica{Clave: faceta.Clave, Version: faceta.Version}
		if referencia.Clave == "" || referencia.Version < 1 {
			return ErrDatosPublicosNoConfiables
		}
		if _, duplicada := diccionario[referencia]; duplicada {
			return ErrDatosPublicosNoConfiables
		}
		diccionario[referencia] = struct{}{}
	}
	for _, resumen := range resumenes {
		vistas := make(map[ReferenciaCategoriaPublica]struct{}, len(resumen.Categorias))
		for _, referencia := range resumen.Categorias {
			if _, duplicada := vistas[referencia]; duplicada {
				return ErrDatosPublicosNoConfiables
			}
			if _, existe := diccionario[referencia]; !existe {
				return ErrDatosPublicosNoConfiables
			}
			vistas[referencia] = struct{}{}
		}
	}
	return nil
}
