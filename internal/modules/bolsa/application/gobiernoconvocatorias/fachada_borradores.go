package gobiernoconvocatorias

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrFachadaBorradoresInvalida      = errors.New("gobierno convocatorias: fachada de borradores invalida")
	ErrSolicitudBorradorInvalida      = errors.New("gobierno convocatorias: solicitud editable de borrador invalida")
	ErrPreparacionBorradorInsegura    = errors.New("gobierno convocatorias: preparacion editable de borrador no confiable")
	ErrConsultaBorradoresNoDisponible = errors.New("gobierno convocatorias: consulta de borradores no disponible")
)

// ContextoOperacionBorrador contiene exclusivamente capacidades procedentes de
// la frontera autenticada del servidor. Ninguno de sus campos se admite en el
// cuerpo, la query ni una cabecera declarativa del cliente.
type ContextoOperacionBorrador struct {
	Actor          dominiovec.ContextoActor
	Vinculo        dominiovec.VinculoAutenticacionActorV1
	CorrelacionRef string
}

func (c ContextoOperacionBorrador) clonarValidado() (ContextoOperacionBorrador, error) {
	actor, err := c.Actor.Clonar()
	if err != nil || c.Vinculo.ValidarPara(actor) != nil || !correlacionFachadaValida(c.CorrelacionRef) {
		return ContextoOperacionBorrador{}, errors.Join(dominiovec.ErrAutorizacionDenegada, ErrSolicitudBorradorInvalida, err)
	}
	c.Actor = actor
	return c, nil
}

// ContenidoEditableBorrador representa solo los campos que el operador puede
// proponer. Catálogos, documentos oficiales, ámbito, flujo e identificadores
// conservados se recomponen en PreparadorContenidoBorrador desde fuentes
// gobernadas; nunca se aceptan como datos confiables del navegador.
type ContenidoEditableBorrador struct {
	Tipo        string
	Categorias  []string
	Titulo      string
	Resumen     string
	Descripcion string
	Plazos      []dominiobolsa.PlazoConvocatoria
	Requisitos  []dominiobolsa.RequisitoConvocatoria
	Ayuda       []dominiobolsa.AyudaConvocatoria
}

type SelectorMotivoBorrador struct {
	Referencia   string
	Version      int
	HuellaSHA256 string
}

type SolicitudAltaBorrador struct {
	ClaveIdempotencia    string
	Plantilla            SelectorPlantillaBorrador
	CodigoVersionPublica string
	IdentificadorPublico string
	ExpedienteRef        string
	Contenido            ContenidoEditableBorrador
	Motivo               SelectorMotivoBorrador
}

type SolicitudActualizacionBorrador struct {
	ClaveIdempotencia string
	Esperada          puertosbolsa.ReferenciaEstadoVersionConvocatoria
	Contenido         ContenidoEditableBorrador
	Motivo            SelectorMotivoBorrador
}

type PreparacionContenidoBorrador struct {
	Contenido        dominiobolsa.ContenidoPublicableConvocatoria
	MotivoSolicitado SelectorMotivoBorrador
	MotivoCatalogo   dominiovec.ReferenciaEntradaCatalogo
}

// PreparadorContenidoBorrador resuelve los datos no editables contra la
// plantilla exacta o la versión exacta. Su adaptador productivo debe efectuar
// cualquier lectura sensible con el contexto autenticado recibido.
type PreparadorContenidoBorrador interface {
	PrepararAlta(
		context.Context,
		ContextoOperacionBorrador,
		SolicitudAltaBorrador,
	) (PreparacionContenidoBorrador, error)
	PrepararActualizacion(
		context.Context,
		ContextoOperacionBorrador,
		SolicitudActualizacionBorrador,
	) (PreparacionContenidoBorrador, error)
}

type MutadorBorradores interface {
	Crear(context.Context, OrdenCrearBorrador) (ProyeccionReciboBorrador, error)
	Actualizar(context.Context, OrdenActualizarBorrador) (ProyeccionReciboBorrador, error)
}

type SelectorListaBorradores struct {
	Limite    int
	Cursor    string
	Texto     string
	Categoria string
}

type LimitesEdicionBorrador struct {
	MaximoCategorias           int
	MaximoPlazos               int
	MaximoRequisitos           int
	MaximoDocumentos           int
	MaximoAyudas               int
	MaximoTitulo               int
	MaximoResumen              int
	MaximoDescripcion          int
	MaximoTituloPlazo          int
	MaximoDescripcionPlazo     int
	MaximoTituloRequisito      int
	MaximoDescripcionRequisito int
	MaximoPreguntaAyuda        int
	MaximoRespuestaAyuda       int
}

type OpcionCatalogoBorrador struct {
	Referencia   string
	Version      int
	HuellaSHA256 string
	Clave        string
	Etiqueta     string
}

type OpcionPlantillaBorrador struct {
	Referencia   string
	Version      int
	HuellaSHA256 string
	Nombre       string
	Descripcion  string
}

type OpcionMotivoBorrador struct {
	Referencia   string
	Version      int
	HuellaSHA256 string
	Etiqueta     string
	Descripcion  string
}

type CapacidadesGlobalesBorrador struct {
	Consultar bool
	Crear     bool
}

type CapacidadesFilaBorrador struct {
	Consultar  bool
	Actualizar bool
}

type OpcionesBorradores struct {
	Categorias  []OpcionCatalogoBorrador
	Tipos       []OpcionCatalogoBorrador
	Plantillas  []OpcionPlantillaBorrador
	Motivos     []OpcionMotivoBorrador
	Limites     LimitesEdicionBorrador
	Capacidades CapacidadesGlobalesBorrador
}

type FilaBorrador struct {
	Estado               puertosbolsa.ReferenciaEstadoVersionConvocatoria
	CodigoVersionPublica string
	IdentificadorPublico string
	Titulo               string
	Tipo                 string
	Categorias           []string
	ExpedienteRef        string
	CreadaEn             time.Time
	ActualizadaEn        time.Time
	NumeroPlazos         int
	NumeroRequisitos     int
	NumeroDocumentos     int
	NumeroAyudas         int
	Capacidades          CapacidadesFilaBorrador
}

type ListaBorradores struct {
	Selector        SelectorListaBorradores
	Total           int
	SiguienteCursor string
	Capacidades     CapacidadesGlobalesBorrador
	Elementos       []FilaBorrador
}

type ReferenciaConfiguracionLecturaBorrador struct {
	Referencia   string
	Version      int
	HuellaSHA256 string
}

type DocumentoLecturaBorrador struct {
	Rol                   string
	PublicacionRef        string
	DocumentoRef          string
	VersionDocumento      int
	RepresentacionRef     string
	HuellaContenidoSHA256 string
	FirmaValidadaRef      string
	ReciboCustodiaRef     string
}

type ConfiguracionLecturaBorrador struct {
	Catalogos        ReferenciaConfiguracionLecturaBorrador
	Calendario       ReferenciaConfiguracionLecturaBorrador
	ReglasBaremacion ReferenciaConfiguracionLecturaBorrador
	FlujoProceso     ReferenciaConfiguracionLecturaBorrador
	FlujoSolicitud   ReferenciaConfiguracionLecturaBorrador
	Plantilla        ReferenciaConfiguracionLecturaBorrador
	Documentos       []DocumentoLecturaBorrador
}

type AmbitoLecturaBorrador struct {
	OrganizacionRef  string
	UnidadGestionRef string
}

type DetalleBorrador struct {
	Estado               puertosbolsa.ReferenciaEstadoVersionConvocatoria
	CodigoVersionPublica string
	IdentificadorPublico string
	Ambito               AmbitoLecturaBorrador
	ExpedienteRef        string
	Contenido            ContenidoEditableBorrador
	Configuracion        ConfiguracionLecturaBorrador
	Capacidades          CapacidadesFilaBorrador
}

// LectorBorradoresInternos es un puerto de lectura autorizado. Listado,
// capacidades y opciones dependen del actor y perfil efectivos; el adaptador
// no puede convertir una ausencia de permiso en un ámbito global.
type LectorBorradoresInternos interface {
	ObtenerOpciones(context.Context, ContextoOperacionBorrador) (OpcionesBorradores, error)
	Listar(context.Context, ContextoOperacionBorrador, SelectorListaBorradores) (ListaBorradores, error)
	ObtenerDetalle(
		context.Context,
		ContextoOperacionBorrador,
		puertosbolsa.SelectorVersionConvocatoriaExacta,
	) (DetalleBorrador, error)
}

// FachadaBorradoresInternos compone la superficie interna. Las mutaciones
// conservan la recuperación del ServicioBorradores: repetir exactamente la
// misma solicitud y ClaveIdempotencia reanuda o recupera el recibo; no existe
// una consulta lateral por clave que debilite la preimagen semántica.
type FachadaBorradoresInternos struct {
	mutador    MutadorBorradores
	preparador PreparadorContenidoBorrador
	lector     LectorBorradoresInternos
}

func NuevaFachadaBorradoresInternos(
	mutador MutadorBorradores,
	preparador PreparadorContenidoBorrador,
	lector LectorBorradoresInternos,
) (*FachadaBorradoresInternos, error) {
	if dependenciaNulaBorrador(mutador) || dependenciaNulaBorrador(preparador) || dependenciaNulaBorrador(lector) {
		return nil, ErrFachadaBorradoresInvalida
	}
	return &FachadaBorradoresInternos{mutador: mutador, preparador: preparador, lector: lector}, nil
}

func (f *FachadaBorradoresInternos) Crear(
	ctx context.Context,
	contexto ContextoOperacionBorrador,
	solicitud SolicitudAltaBorrador,
) (ProyeccionReciboBorrador, error) {
	contexto, err := f.validar(ctx, contexto)
	if err != nil || !solicitudAltaValida(solicitud) {
		return ProyeccionReciboBorrador{}, errors.Join(ErrSolicitudBorradorInvalida, err)
	}
	clave, err := NuevaClaveClienteIdempotenciaConvocatoria(solicitud.ClaveIdempotencia)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrSolicitudBorradorInvalida, err)
	}
	preparacion, err := f.preparador.PrepararAlta(ctx, contexto, clonarSolicitudAlta(solicitud))
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	if !preparacionValidaParaAlta(preparacion, solicitud) {
		return ProyeccionReciboBorrador{}, ErrPreparacionBorradorInsegura
	}
	return f.mutador.Crear(ctx, OrdenCrearBorrador{
		ClaveCliente: clave, Actor: contexto.Actor, VinculoAutenticacionActor: contexto.Vinculo,
		Plantilla: solicitud.Plantilla, CodigoVersionPublica: solicitud.CodigoVersionPublica,
		Contenido: preparacion.Contenido, ExpedienteRef: solicitud.ExpedienteRef,
		MotivoCatalogo: preparacion.MotivoCatalogo, CorrelacionRef: contexto.CorrelacionRef,
	})
}

func (f *FachadaBorradoresInternos) Actualizar(
	ctx context.Context,
	contexto ContextoOperacionBorrador,
	solicitud SolicitudActualizacionBorrador,
) (ProyeccionReciboBorrador, error) {
	contexto, err := f.validar(ctx, contexto)
	if err != nil || !solicitudActualizacionValida(solicitud) {
		return ProyeccionReciboBorrador{}, errors.Join(ErrSolicitudBorradorInvalida, err)
	}
	clave, err := NuevaClaveClienteIdempotenciaConvocatoria(solicitud.ClaveIdempotencia)
	if err != nil {
		return ProyeccionReciboBorrador{}, errors.Join(ErrSolicitudBorradorInvalida, err)
	}
	preparacion, err := f.preparador.PrepararActualizacion(ctx, contexto, clonarSolicitudActualizacion(solicitud))
	if err != nil {
		return ProyeccionReciboBorrador{}, err
	}
	if !preparacionValidaParaActualizacion(preparacion, solicitud) {
		return ProyeccionReciboBorrador{}, ErrPreparacionBorradorInsegura
	}
	return f.mutador.Actualizar(ctx, OrdenActualizarBorrador{
		ClaveCliente: clave, Actor: contexto.Actor, VinculoAutenticacionActor: contexto.Vinculo,
		Esperada: solicitud.Esperada, Contenido: preparacion.Contenido,
		MotivoCatalogo: preparacion.MotivoCatalogo, CorrelacionRef: contexto.CorrelacionRef,
	})
}

func (f *FachadaBorradoresInternos) ObtenerOpciones(
	ctx context.Context, contexto ContextoOperacionBorrador,
) (OpcionesBorradores, error) {
	contexto, err := f.validar(ctx, contexto)
	if err != nil {
		return OpcionesBorradores{}, err
	}
	return f.lector.ObtenerOpciones(ctx, contexto)
}

func (f *FachadaBorradoresInternos) Listar(
	ctx context.Context, contexto ContextoOperacionBorrador, selector SelectorListaBorradores,
) (ListaBorradores, error) {
	contexto, err := f.validar(ctx, contexto)
	if err != nil || selector.Limite < 1 || selector.Limite > 50 {
		return ListaBorradores{}, errors.Join(ErrSolicitudBorradorInvalida, err)
	}
	return f.lector.Listar(ctx, contexto, selector)
}

func (f *FachadaBorradoresInternos) ObtenerDetalle(
	ctx context.Context,
	contexto ContextoOperacionBorrador,
	selector puertosbolsa.SelectorVersionConvocatoriaExacta,
) (DetalleBorrador, error) {
	contexto, err := f.validar(ctx, contexto)
	if err != nil || selector.Validar() != nil {
		return DetalleBorrador{}, errors.Join(ErrSolicitudBorradorInvalida, err)
	}
	return f.lector.ObtenerDetalle(ctx, contexto, selector)
}

func (f *FachadaBorradoresInternos) validar(
	ctx context.Context, contexto ContextoOperacionBorrador,
) (ContextoOperacionBorrador, error) {
	if ctx == nil || f == nil || dependenciaNulaBorrador(f.mutador) ||
		dependenciaNulaBorrador(f.preparador) || dependenciaNulaBorrador(f.lector) {
		return ContextoOperacionBorrador{}, ErrFachadaBorradoresInvalida
	}
	if err := ctx.Err(); err != nil {
		return ContextoOperacionBorrador{}, err
	}
	return contexto.clonarValidado()
}

func solicitudAltaValida(s SolicitudAltaBorrador) bool {
	return s.Plantilla.ID != "" && s.Plantilla.Version > 0 && huellaHexValida(s.Plantilla.HuellaContenidoSHA256) &&
		referenciaFachadaValida(s.CodigoVersionPublica, 80) && referenciaFachadaValida(s.IdentificadorPublico, 80) &&
		referenciaFachadaValida(s.ExpedienteRef, 512) && selectorMotivoValido(s.Motivo) && contenidoEditableBasicoValido(s.Contenido)
}

func solicitudActualizacionValida(s SolicitudActualizacionBorrador) bool {
	return s.Esperada.Validar() == nil && selectorMotivoValido(s.Motivo) && contenidoEditableBasicoValido(s.Contenido)
}

func selectorMotivoValido(s SelectorMotivoBorrador) bool {
	return referenciaFachadaValida(s.Referencia, 512) && s.Version > 0 && huellaHexValida(s.HuellaSHA256)
}

func preparacionValidaParaAlta(p PreparacionContenidoBorrador, s SolicitudAltaBorrador) bool {
	return p.MotivoSolicitado == s.Motivo && p.MotivoCatalogo.Validar() == nil &&
		p.MotivoCatalogo.CatalogoVersion == s.Motivo.Version &&
		p.MotivoCatalogo.CatalogoHuellaSHA256 == s.Motivo.HuellaSHA256 &&
		p.Contenido.Validar() == nil && p.Contenido.IdentificadorPublico == s.IdentificadorPublico &&
		contenidoEditableCoincide(p.Contenido, s.Contenido)
}

func preparacionValidaParaActualizacion(
	p PreparacionContenidoBorrador, s SolicitudActualizacionBorrador,
) bool {
	return p.MotivoSolicitado == s.Motivo && p.MotivoCatalogo.Validar() == nil &&
		p.MotivoCatalogo.CatalogoVersion == s.Motivo.Version &&
		p.MotivoCatalogo.CatalogoHuellaSHA256 == s.Motivo.HuellaSHA256 &&
		p.Contenido.Validar() == nil && contenidoEditableCoincide(p.Contenido, s.Contenido)
}

func contenidoEditableBasicoValido(c ContenidoEditableBorrador) bool {
	return c.Categorias != nil && c.Plazos != nil && c.Requisitos != nil && c.Ayuda != nil &&
		c.Tipo != "" && c.Titulo != "" && c.Resumen != "" && c.Descripcion != "" &&
		len(c.Categorias) > 0 && len(c.Categorias) <= 1024 && len(c.Plazos) > 0 && len(c.Plazos) <= 64 &&
		len(c.Requisitos) <= 256 && len(c.Ayuda) <= 128 && utf8.ValidString(c.Tipo+c.Titulo+c.Resumen+c.Descripcion)
}

func contenidoEditableCoincide(
	completo dominiobolsa.ContenidoPublicableConvocatoria, editable ContenidoEditableBorrador,
) bool {
	obtenido := ContenidoEditableBorrador{
		Tipo: completo.Tipo, Categorias: completo.Categorias, Titulo: completo.Titulo,
		Resumen: completo.Resumen, Descripcion: completo.Descripcion, Plazos: completo.Plazos,
		Requisitos: completo.Requisitos, Ayuda: completo.Ayuda,
	}
	return reflect.DeepEqual(clonarContenidoEditable(obtenido), clonarContenidoEditable(editable))
}

func clonarSolicitudAlta(s SolicitudAltaBorrador) SolicitudAltaBorrador {
	s.Contenido = clonarContenidoEditable(s.Contenido)
	return s
}

func clonarSolicitudActualizacion(s SolicitudActualizacionBorrador) SolicitudActualizacionBorrador {
	s.Contenido = clonarContenidoEditable(s.Contenido)
	return s
}

func clonarContenidoEditable(c ContenidoEditableBorrador) ContenidoEditableBorrador {
	c.Categorias = append([]string(nil), c.Categorias...)
	c.Plazos = append([]dominiobolsa.PlazoConvocatoria(nil), c.Plazos...)
	c.Requisitos = append([]dominiobolsa.RequisitoConvocatoria(nil), c.Requisitos...)
	c.Ayuda = append([]dominiobolsa.AyudaConvocatoria(nil), c.Ayuda...)
	sort.Strings(c.Categorias)
	sort.Slice(c.Plazos, func(i, j int) bool { return c.Plazos[i].Referencia < c.Plazos[j].Referencia })
	sort.Slice(c.Requisitos, func(i, j int) bool {
		if c.Requisitos[i].Orden == c.Requisitos[j].Orden {
			return c.Requisitos[i].Referencia < c.Requisitos[j].Referencia
		}
		return c.Requisitos[i].Orden < c.Requisitos[j].Orden
	})
	sort.Slice(c.Ayuda, func(i, j int) bool {
		if c.Ayuda[i].Orden == c.Ayuda[j].Orden {
			return c.Ayuda[i].Referencia < c.Ayuda[j].Referencia
		}
		return c.Ayuda[i].Orden < c.Ayuda[j].Orden
	})
	return c
}

func referenciaFachadaValida(valor string, maximo int) bool {
	if valor == "" || len(valor) > maximo || valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) || strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) ||
			unicode.Is(unicode.Bidi_Control, caracter) || caracter == unicode.ReplacementChar {
			return false
		}
	}
	return true
}

func correlacionFachadaValida(valor string) bool {
	if valor == "" || len(valor) > 180 || valor != strings.TrimSpace(valor) {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < 0x21 || valor[indice] > 0x7e || valor[indice] == '*' {
			return false
		}
	}
	return true
}

var _ MutadorBorradores = (*ServicioBorradores)(nil)
