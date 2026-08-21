package ports

import (
	"context"
	"errors"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionAutorizacionCerrarAdministrativamente    = "contratacion_temporal.seguimiento.cerrar"
	AccionAutorizacionReabrirExcepcionalmente      = "contratacion_temporal.seguimiento.reabrir"
	FinalidadAutorizacionCerrarAdministrativamente = "cerrar_expediente_contratacion_temporal"
	FinalidadAutorizacionReabrirExcepcionalmente   = "reabrir_expediente_contratacion_temporal"
	TipoRecursoCierreAdministrativo                = "seguimiento_contratacion_temporal"

	ambitoCierreOrganizacion = "organizacion_ref"
	ambitoCierreExpediente   = "expediente_ref"
	ambitoCierreSeguimiento  = "seguimiento_ref"
	atributoCierreOperacion  = "operacion"
	atributoCierreVersion    = "version_esperada"
	atributoCierreTransicion = "transicion_clave"
	atributoCierreMotivo     = "motivo_clave"
)

var (
	ErrSolicitudCierreAdministrativoInvalida = errors.New(
		"contratacion temporal: solicitud de cierre administrativo invalida",
	)
	ErrPreparacionCierreAdministrativoInvalida = errors.New(
		"contratacion temporal: preparacion de cierre administrativo invalida",
	)
	ErrResultadoCierreAdministrativoInvalido = errors.New(
		"contratacion temporal: resultado de cierre administrativo invalido",
	)
	ErrCierreAdministrativoDenegado = errors.New(
		"contratacion temporal: cierre administrativo denegado",
	)
	ErrTransaccionCierreAdministrativoNoDisponible = errors.New(
		"contratacion temporal: transaccion de cierre administrativo no disponible",
	)
	ErrClaveIdempotenciaCierreAdministrativoUsada = errors.New(
		"contratacion temporal: clave de idempotencia de cierre usada con otros datos",
	)
)

type OperacionCierreAdministrativo string

const (
	OperacionCerrarAdministrativamente OperacionCierreAdministrativo = "cerrar_administrativamente"
	OperacionReabrirExcepcionalmente   OperacionCierreAdministrativo = "reabrir_excepcionalmente"
)

func (o OperacionCierreAdministrativo) Valida() bool {
	return o == OperacionCerrarAdministrativamente ||
		o == OperacionReabrirExcepcionalmente
}

// SolicitudTransaccionCierreAdministrativo transporta solo intención. Actor,
// perfil, unidad, autorización, instantes y referencias de actuación se
// resuelven desde la frontera confiable dentro de la transacción.
type SolicitudTransaccionCierreAdministrativo struct {
	Operacion         OperacionCierreAdministrativo
	OrganizacionRef   string
	ExpedienteRef     string
	SeguimientoRef    string
	VersionEsperada   uint64
	ClaveIdempotencia string
	TransicionClave   domain.ClaveCatalogo
	MotivoClave       domain.ClaveCatalogo
}

func (s SolicitudTransaccionCierreAdministrativo) Validar() error {
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.SeguimientoRef) ||
		s.VersionEsperada == ^uint64(0) ||
		!ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		!s.TransicionClave.Valida() || !s.MotivoClave.Valida() {
		return ErrSolicitudCierreAdministrativoInvalida
	}
	return nil
}

// InventarioTareasCierreAdministrativo identifica la lectura completa y
// bloqueada del libro autoritativo de tareas. No contiene nombres ni detalles
// funcionales que puedan convertirse en un oráculo.
type InventarioTareasCierreAdministrativo struct {
	Referencia      string
	OrganizacionRef string
	ExpedienteRef   string
	SeguimientoRef  string
	Version         uint64
	Total           uint32
	Pendientes      uint32
	Completo        bool
}

func (i InventarioTareasCierreAdministrativo) Validar() error {
	if !i.Completo || !domain.ReferenciaOpacaValida(i.Referencia) ||
		!domain.ReferenciaOpacaValida(i.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(i.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(i.SeguimientoRef) ||
		i.Version == 0 || i.Pendientes > i.Total {
		return ErrPreparacionCierreAdministrativoInvalida
	}
	return nil
}

func (i InventarioTareasCierreAdministrativo) ValidarPara(
	solicitud SolicitudTransaccionCierreAdministrativo,
) error {
	if i.Validar() != nil || solicitud.Validar() != nil ||
		i.OrganizacionRef != solicitud.OrganizacionRef ||
		i.ExpedienteRef != solicitud.ExpedienteRef ||
		i.SeguimientoRef != solicitud.SeguimientoRef {
		return ErrPreparacionCierreAdministrativoInvalida
	}
	return nil
}

// PreparacionTransaccionCierreAdministrativo solo puede proceder de la
// implementación confiable del puerto, una vez bloqueados el seguimiento, el
// inventario de tareas y la autorización V3 exacta. La confirmación V3 se
// coteja y consume en la misma transacción si el callback devuelve un estado
// nuevo.
type PreparacionTransaccionCierreAdministrativo struct {
	Solicitud                    SolicitudTransaccionCierreAdministrativo
	Definicion                   domain.DefinicionSeguimiento
	Seguimiento                  domain.Seguimiento
	Inventario                   InventarioTareasCierreAdministrativo
	ContextoAutorizacionV3       ContextoAutorizacionAltaV3
	SolicitudAutorizacionV3      dominiovec.SolicitudAutorizacionLigadaV3
	DecisionAutorizacionV3       dominiovec.DecisionAutorizacionLigadaV3
	ConfirmacionAutorizacionV3   puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	MotivoAutorizacionV3         dominiovec.ReferenciaEntradaCatalogo
	CorrelacionAutorizacionV3Ref string
	ActorRef                     string
	PerfilRef                    string
	UnidadRef                    string
	ActuacionRef                 string
	ReciboRef                    string
	CorrelacionRef               string
	Documentos                   []domain.DocumentoSeguimiento
	EfectivoEn                   time.Time
	RegistradaEn                 time.Time
}

func (p PreparacionTransaccionCierreAdministrativo) ValidarPara(
	solicitud SolicitudTransaccionCierreAdministrativo,
) error {
	estado := p.Seguimiento.Estado()
	if solicitud.Validar() != nil || p.Solicitud != solicitud ||
		!domain.ReferenciaOpacaValida(p.ActorRef) ||
		!domain.ReferenciaOpacaValida(p.PerfilRef) ||
		!domain.ReferenciaOpacaValida(p.UnidadRef) ||
		!domain.ReferenciaOpacaValida(p.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(p.ReciboRef) ||
		!domain.ReferenciaOpacaValida(p.CorrelacionRef) ||
		!domain.InstanteUTCCanonico(p.EfectivoEn) ||
		!domain.InstanteUTCCanonico(p.RegistradaEn) ||
		p.EfectivoEn.Before(p.RegistradaEn) ||
		p.Inventario.ValidarPara(solicitud) != nil ||
		p.Definicion.Validar() != nil ||
		p.Seguimiento.Validar(p.Definicion) != nil ||
		estado.OrganizacionRef != solicitud.OrganizacionRef ||
		estado.ExpedienteRef != solicitud.ExpedienteRef ||
		estado.Referencia != solicitud.SeguimientoRef ||
		validarAutorizacionCierreAdministrativoV3(p, solicitud) != nil {
		return ErrPreparacionCierreAdministrativoInvalida
	}
	return nil
}

func validarAutorizacionCierreAdministrativoV3(
	p PreparacionTransaccionCierreAdministrativo,
	solicitud SolicitudTransaccionCierreAdministrativo,
) error {
	datos, err := p.SolicitudAutorizacionV3.Datos()
	vinculo, errVinculo := datos.VinculoAutenticacionActor.Datos()
	concedida, _, errDecision := p.DecisionAutorizacionV3.Resultado()
	huellaDecision, errHuella := dominiovec.HuellaSHA256DecisionAutorizacionV3(
		p.DecisionAutorizacionV3,
	)
	confirmacion, errConfirmacion := p.ConfirmacionAutorizacionV3.Datos()
	correlacion, errCorrelacion := datos.Correlacion.ValorCanonico()
	accion, finalidad := parametrosAutorizacionCierreAdministrativo(
		solicitud.Operacion,
	)
	if err != nil || errVinculo != nil || errDecision != nil ||
		errHuella != nil || errConfirmacion != nil || errCorrelacion != nil ||
		!concedida || accion == "" || finalidad == "" ||
		p.DecisionAutorizacionV3.ValidarPara(p.SolicitudAutorizacionV3) != nil ||
		p.ContextoAutorizacionV3.Vinculo.ValidarPara(
			p.ContextoAutorizacionV3.Resultado,
		) != nil ||
		!datos.VinculoAutenticacionActor.CoincideExactamenteCon(
			p.ContextoAutorizacionV3.Vinculo,
		) ||
		!datos.VinculoAutenticacionActor.VigenteEn(
			p.RegistradaEn,
			p.ContextoAutorizacionV3.Resultado,
		) ||
		vinculo.PerfilActivoRef != p.PerfilRef ||
		!dominiovec.CumpleGarantiaAutenticacion(
			vinculo.GarantiaObservada,
			dominiovec.AuthAssuranceHigh,
		) ||
		datos.ReferenciaMotivo != p.MotivoAutorizacionV3 ||
		datos.Accion != accion || datos.Finalidad != finalidad ||
		correlacion != p.CorrelacionAutorizacionV3Ref ||
		confirmacion.DecisionHuellaSHA256 != huellaDecision ||
		!p.ConfirmacionAutorizacionV3.DentroDeVentanaEn(p.RegistradaEn) ||
		!recursoAutorizacionCierreAdministrativoValido(
			datos.Recurso,
			solicitud,
		) {
		return ErrPreparacionCierreAdministrativoInvalida
	}
	return nil
}

func parametrosAutorizacionCierreAdministrativo(
	operacion OperacionCierreAdministrativo,
) (string, string) {
	switch operacion {
	case OperacionCerrarAdministrativamente:
		return AccionAutorizacionCerrarAdministrativamente,
			FinalidadAutorizacionCerrarAdministrativamente
	case OperacionReabrirExcepcionalmente:
		return AccionAutorizacionReabrirExcepcionalmente,
			FinalidadAutorizacionReabrirExcepcionalmente
	default:
		return "", ""
	}
}

func recursoAutorizacionCierreAdministrativoValido(
	recurso dominiovec.RecursoAutorizable,
	solicitud SolicitudTransaccionCierreAdministrativo,
) bool {
	return recurso.Validar() == nil &&
		recurso.Referencia == solicitud.SeguimientoRef &&
		recurso.ModuloID == ModuloContratacion &&
		recurso.Tipo == TipoRecursoCierreAdministrativo &&
		len(recurso.Ambitos) == 3 && len(recurso.Atributos) == 4 &&
		recurso.Ambitos[ambitoCierreOrganizacion] == solicitud.OrganizacionRef &&
		recurso.Ambitos[ambitoCierreExpediente] == solicitud.ExpedienteRef &&
		recurso.Ambitos[ambitoCierreSeguimiento] == solicitud.SeguimientoRef &&
		recurso.Atributos[atributoCierreOperacion] == string(solicitud.Operacion) &&
		recurso.Atributos[atributoCierreVersion] == strconv.FormatUint(solicitud.VersionEsperada, 10) &&
		recurso.Atributos[atributoCierreTransicion] == string(solicitud.TransicionClave) &&
		recurso.Atributos[atributoCierreMotivo] == string(solicitud.MotivoClave)
}

type EstadoResultadoCierreAdministrativo string

const (
	EstadoResultadoCierreAdministrativoConfirmado       EstadoResultadoCierreAdministrativo = "confirmado"
	EstadoResultadoCierreAdministrativoReplayConfirmado EstadoResultadoCierreAdministrativo = "replay_confirmado"
)

func (e EstadoResultadoCierreAdministrativo) valida() bool {
	return e == EstadoResultadoCierreAdministrativoConfirmado ||
		e == EstadoResultadoCierreAdministrativoReplayConfirmado
}

type datosResultadoCierreAdministrativo struct {
	solicitud         SolicitudTransaccionCierreAdministrativo
	versionResultante uint64
	actuacionRef      string
	reciboRef         string
	estado            EstadoResultadoCierreAdministrativo
}

// ResultadoCierreAdministrativo oculta estado, identidad, inventario y
// diagnóstico. Solo publica el recibo opaco y la versión confirmada.
type ResultadoCierreAdministrativo struct {
	datos *datosResultadoCierreAdministrativo
}

type DatosResultadoCierreAdministrativo struct {
	Solicitud         SolicitudTransaccionCierreAdministrativo
	VersionResultante uint64
	ActuacionRef      string
	ReciboRef         string
	Estado            EstadoResultadoCierreAdministrativo
}

func NuevoResultadoCierreAdministrativo(
	datos DatosResultadoCierreAdministrativo,
) (ResultadoCierreAdministrativo, error) {
	if datos.Solicitud.Validar() != nil ||
		datos.VersionResultante != datos.Solicitud.VersionEsperada+1 ||
		!domain.ReferenciaOpacaValida(datos.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboRef) ||
		!datos.Estado.valida() {
		return ResultadoCierreAdministrativo{},
			ErrResultadoCierreAdministrativoInvalido
	}
	return ResultadoCierreAdministrativo{datos: &datosResultadoCierreAdministrativo{
		solicitud: datos.Solicitud, versionResultante: datos.VersionResultante,
		actuacionRef: datos.ActuacionRef, reciboRef: datos.ReciboRef,
		estado: datos.Estado,
	}}, nil
}

func (r ResultadoCierreAdministrativo) ValidarPara(
	solicitud SolicitudTransaccionCierreAdministrativo,
) error {
	if r.datos == nil || solicitud.Validar() != nil ||
		r.datos.solicitud != solicitud ||
		r.datos.versionResultante != solicitud.VersionEsperada+1 ||
		!domain.ReferenciaOpacaValida(r.datos.actuacionRef) ||
		!domain.ReferenciaOpacaValida(r.datos.reciboRef) ||
		!r.datos.estado.valida() {
		return ErrResultadoCierreAdministrativoInvalido
	}
	return nil
}

func (r ResultadoCierreAdministrativo) CoincideConEfecto(
	actuacionRef string,
	reciboRef string,
) bool {
	return r.datos != nil &&
		r.datos.actuacionRef == actuacionRef && r.datos.reciboRef == reciboRef
}

func (r ResultadoCierreAdministrativo) EsReplayConfirmado() bool {
	return r.datos != nil &&
		r.datos.estado == EstadoResultadoCierreAdministrativoReplayConfirmado
}

func (r ResultadoCierreAdministrativo) ReciboRef() string {
	if r.datos == nil {
		return ""
	}
	return r.datos.reciboRef
}

func (r ResultadoCierreAdministrativo) VersionSeguimiento() uint64 {
	if r.datos == nil {
		return 0
	}
	return r.datos.versionResultante
}

type AplicarCierreAdministrativo func(
	PreparacionTransaccionCierreAdministrativo,
) (domain.Seguimiento, error)

// TransaccionCierreAdministrativo ejecuta el callback como máximo una vez
// dentro de una transacción durable. La identidad idempotente es la clave más
// todos los campos de la solicitud exacta: una colisión semántica devuelve
// ErrClaveIdempotenciaCierreAdministrativoUsada. Un replay ya confirmado
// devuelve el resultado previo marcado como replay sin invocar el callback.
// La implementación debe bloquear y reacreditar el
// seguimiento, su definición y el inventario completo de tareas; resolver la
// autoridad solo desde la frontera confiable; y, si el callback tiene éxito,
// consumir esa autorización y añadir actuación, auditoría y outbox junto con
// el nuevo estado. Cualquier error revierte todo el efecto.
type TransaccionCierreAdministrativo interface {
	EjecutarCierreAdministrativo(
		context.Context,
		SolicitudTransaccionCierreAdministrativo,
		AplicarCierreAdministrativo,
	) (ResultadoCierreAdministrativo, error)
}
