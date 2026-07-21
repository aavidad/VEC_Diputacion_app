package gobiernoconvocatorias

import (
	"context"
	"errors"
	"reflect"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionListarBorradoresGobernados    = "bolsa.convocatoria.borrador.listar"
	AccionConsultarBorradorGobernado    = "bolsa.convocatoria.borrador.consultar"
	TipoColeccionBorradoresGobernados   = "coleccion_versiones_convocatoria_gobernada"
	FinalidadLecturaBorradoresGobernada = "consulta_interna_convocatorias"
)

var (
	ErrLecturaBorradoresGobernadaInvalida = errors.New("gobierno convocatorias: lectura gobernada de borradores invalida")
	ErrPreautorizacionLecturaBorrador     = errors.New("gobierno convocatorias: preautorizacion de lectura de borrador invalida")
)

// CapacidadLecturaBorrador es una autorización ya emitida y atestada. El
// motivo de catálogo se resuelve dentro del preautorizador: ni este valor ni
// su constructor aceptan texto o referencias procedentes de HTTP.
type CapacidadLecturaBorrador struct {
	Solicitud              dominiovec.SolicitudAutorizacionLigadaV2
	Evidencia              puertosvec.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	Motivo                 dominiovec.ReferenciaEntradaCatalogo
	Recurso                dominiovec.RecursoAutorizable
	OrganizacionRef        string
	UnidadGestionRef       string
	AtestacionRef          string
	VersionAtestacion      uint32
	EstadoAtestacion       string
	HuellaAtestacionSHA256 string
}

// PreautorizadorLecturaBorrador es la única dependencia que puede obtener el
// motivo publicado, pedir evidencia V2 y registrar la atestación PDP. Una
// denegación o fallo impide invocar el repositorio.
type PreautorizadorLecturaBorrador interface {
	PreautorizarLecturaBorrador(
		context.Context,
		ContextoOperacionBorrador,
		string,
		puertosbolsa.SelectorVersionConvocatoriaExacta,
	) (CapacidadLecturaBorrador, error)
}

type SolicitudListadoBorradoresGobernada struct {
	Contexto  ContextoOperacionBorrador
	Selector  SelectorListaBorradores
	Capacidad CapacidadLecturaBorrador
}

type SolicitudDetalleBorradorGobernada struct {
	Contexto  ContextoOperacionBorrador
	Selector  puertosbolsa.SelectorVersionConvocatoriaExacta
	Capacidad CapacidadLecturaBorrador
}

type RepositorioLecturaBorradoresGobernada interface {
	ListarBorradoresGobernados(context.Context, SolicitudListadoBorradoresGobernada) (ListaBorradores, error)
	ObtenerBorradorGobernado(context.Context, SolicitudDetalleBorradorGobernada) (DetalleBorrador, error)
}

// ServicioLecturaBorradoresGobernada compone preautorización y almacenamiento
// en ese orden. No implementa opciones: las opciones tienen catálogo y
// autorización propios y se conectarán con su caso de uso, sin constantes.
type ServicioLecturaBorradoresGobernada struct {
	preautorizador PreautorizadorLecturaBorrador
	repositorio    RepositorioLecturaBorradoresGobernada
}

func NuevoServicioLecturaBorradoresGobernada(
	preautorizador PreautorizadorLecturaBorrador,
	repositorio RepositorioLecturaBorradoresGobernada,
) (*ServicioLecturaBorradoresGobernada, error) {
	if dependenciaLecturaGobernadaNula(preautorizador) || dependenciaLecturaGobernadaNula(repositorio) {
		return nil, ErrLecturaBorradoresGobernadaInvalida
	}
	return &ServicioLecturaBorradoresGobernada{preautorizador: preautorizador, repositorio: repositorio}, nil
}

func (s *ServicioLecturaBorradoresGobernada) ObtenerOpciones(
	context.Context, ContextoOperacionBorrador,
) (OpcionesBorradores, error) {
	return OpcionesBorradores{}, ErrConsultaBorradoresNoDisponible
}

func (s *ServicioLecturaBorradoresGobernada) Listar(
	ctx context.Context, contexto ContextoOperacionBorrador, selector SelectorListaBorradores,
) (ListaBorradores, error) {
	if ctx == nil || s == nil || dependenciaLecturaGobernadaNula(s.preautorizador) || dependenciaLecturaGobernadaNula(s.repositorio) || ctx.Err() != nil || selector.Validar() != nil || !dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(contexto.CorrelacionRef) {
		return ListaBorradores{}, errorLecturaGobernada(ctx)
	}
	contexto, err := contexto.clonarValidado()
	if err != nil {
		return ListaBorradores{}, errors.Join(errorAutorizacionGobernada(), ErrPreautorizacionLecturaBorrador, err)
	}
	capacidad, err := s.preautorizador.PreautorizarLecturaBorrador(ctx, contexto, AccionListarBorradoresGobernados, puertosbolsa.SelectorVersionConvocatoriaExacta{})
	if err != nil {
		return ListaBorradores{}, errorPreautorizacionGobernada(ctx, err)
	}
	solicitud := SolicitudListadoBorradoresGobernada{Contexto: contexto, Selector: selector, Capacidad: capacidad}
	if solicitud.validar() != nil {
		return ListaBorradores{}, ErrPreautorizacionLecturaBorrador
	}
	return s.repositorio.ListarBorradoresGobernados(ctx, solicitud)
}

func (s *ServicioLecturaBorradoresGobernada) ObtenerDetalle(
	ctx context.Context, contexto ContextoOperacionBorrador, selector puertosbolsa.SelectorVersionConvocatoriaExacta,
) (DetalleBorrador, error) {
	if ctx == nil || s == nil || dependenciaLecturaGobernadaNula(s.preautorizador) || dependenciaLecturaGobernadaNula(s.repositorio) || ctx.Err() != nil || selector.Validar() != nil || !dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(contexto.CorrelacionRef) {
		return DetalleBorrador{}, errorLecturaGobernada(ctx)
	}
	contexto, err := contexto.clonarValidado()
	if err != nil {
		return DetalleBorrador{}, errors.Join(errorAutorizacionGobernada(), ErrPreautorizacionLecturaBorrador, err)
	}
	capacidad, err := s.preautorizador.PreautorizarLecturaBorrador(ctx, contexto, AccionConsultarBorradorGobernado, selector)
	if err != nil {
		return DetalleBorrador{}, errorPreautorizacionGobernada(ctx, err)
	}
	solicitud := SolicitudDetalleBorradorGobernada{Contexto: contexto, Selector: selector, Capacidad: capacidad}
	if solicitud.validar() != nil {
		return DetalleBorrador{}, ErrPreautorizacionLecturaBorrador
	}
	return s.repositorio.ObtenerBorradorGobernado(ctx, solicitud)
}

func (s SolicitudListadoBorradoresGobernada) validar() error {
	if s.Selector.Validar() != nil {
		return ErrLecturaBorradoresGobernadaInvalida
	}
	return validarCapacidadLecturaGobernada(s.Contexto, s.Capacidad, AccionListarBorradoresGobernados, "")
}

func (s SolicitudDetalleBorradorGobernada) validar() error {
	if s.Selector.Validar() != nil {
		return ErrLecturaBorradoresGobernadaInvalida
	}
	return validarCapacidadLecturaGobernada(s.Contexto, s.Capacidad, AccionConsultarBorradorGobernado, s.Selector.Referencia())
}

func validarCapacidadLecturaGobernada(contexto ContextoOperacionBorrador, c CapacidadLecturaBorrador, accion, referencia string) error {
	actor, errActor := contexto.Actor.Clonar()
	datos, errDatos := c.Evidencia.Datos()
	solicitud, errSolicitud := c.Solicitud.Datos()
	correlacion, errCorrelacion := solicitud.Correlacion.ValorCanonico()
	huellaSolicitud, errHuellaSolicitud := dominiovec.HuellaSHA256SolicitudAutorizacionV2(c.Solicitud)
	if errActor != nil || errDatos != nil || errSolicitud != nil || errCorrelacion != nil ||
		errHuellaSolicitud != nil || contexto.Vinculo.ValidarPara(actor) != nil ||
		c.Evidencia.ValidarEn(datos.VerificadaEn) != nil || c.Evidencia.ValidarMotivo(c.Motivo) != nil ||
		c.Recurso.Validar() != nil || c.OrganizacionRef == "" || c.EstadoAtestacion != "activa" ||
		c.VersionAtestacion == 0 || !referenciaFachadaValida(c.AtestacionRef, 512) ||
		!huellaHexValida(c.HuellaAtestacionSHA256) ||
		!dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(contexto.CorrelacionRef) {
		return ErrLecturaBorradoresGobernadaInvalida
	}
	huella, errHuella := c.Recurso.HuellaContextoAutorizacionSHA256()
	d := datos.Decision
	esperada, tipo := referencia, puertosbolsa.TipoRecursoVersionConvocatoriaGobernada
	if accion == AccionListarBorradoresGobernados {
		esperada, tipo = "borradores:"+c.OrganizacionRef, TipoColeccionBorradoresGobernados
	}
	if errHuella != nil || d.Accion != accion || d.RecursoRef != esperada || c.Recurso.Referencia != esperada ||
		d.ModuloID != puertosbolsa.ModuloGobiernoConvocatorias || d.TipoRecurso != tipo ||
		d.Finalidad != FinalidadLecturaBorradoresGobernada || d.PrincipalID != actor.PersonaRef ||
		d.PerfilActivoRef != actor.PerfilActivoRef || d.CorrelacionRef != contexto.CorrelacionRef ||
		d.ContextoRecursoHuellaSHA256 != huella ||
		d.EsquemaHuellaSolicitud != dominiovec.EsquemaHuellaSolicitudAutorizacionV2 ||
		d.SolicitudHuellaSHA256 != huellaSolicitud ||
		d.EsquemaHuellaMotivo != dominiovec.EsquemaHuellaMotivoAutorizacionV2 ||
		solicitud.Accion != accion || solicitud.Finalidad != FinalidadLecturaBorradoresGobernada ||
		correlacion != contexto.CorrelacionRef || solicitud.ReferenciaMotivo != c.Motivo ||
		!reflect.DeepEqual(solicitud.ContextoActor, actor) ||
		!reflect.DeepEqual(solicitud.VinculoAutenticacionActor, contexto.Vinculo) ||
		!reflect.DeepEqual(solicitud.Recurso, c.Recurso) ||
		!reflect.DeepEqual(c.Recurso.Atributos, map[string]string{}) ||
		c.Recurso.Ambitos["organizacion_ref"] != c.OrganizacionRef ||
		c.Recurso.Ambitos["unidad_gestion_ref"] != c.UnidadGestionRef ||
		len(c.Recurso.Ambitos) != numeroAmbitosLectura(c.UnidadGestionRef) {
		return ErrLecturaBorradoresGobernadaInvalida
	}
	return nil
}

func numeroAmbitosLectura(unidad string) int {
	if unidad == "" {
		return 1
	}
	return 2
}
func dependenciaLecturaGobernadaNula(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return x.IsNil()
	}
	return false
}
func errorLecturaGobernada(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ErrLecturaBorradoresGobernadaInvalida
}
func errorPreautorizacionGobernada(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, dominiovec.ErrAutorizacionDenegada) {
		return dominiovec.ErrAutorizacionDenegada
	}
	return ErrPreautorizacionLecturaBorrador
}
func errorAutorizacionGobernada() error { return dominiovec.ErrAutorizacionDenegada }
