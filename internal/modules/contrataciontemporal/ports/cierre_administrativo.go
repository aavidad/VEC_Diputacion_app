package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
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
	Operacion       OperacionCierreAdministrativo
	OrganizacionRef string
	ExpedienteRef   string
	SeguimientoRef  string
	VersionEsperada uint64
	TransicionClave domain.ClaveCatalogo
	MotivoClave     domain.ClaveCatalogo
}

func (s SolicitudTransaccionCierreAdministrativo) Validar() error {
	if !s.Operacion.Valida() ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(s.SeguimientoRef) ||
		s.VersionEsperada == ^uint64(0) ||
		!s.TransicionClave.Valida() || !s.MotivoClave.Valida() {
		return ErrSolicitudCierreAdministrativoInvalida
	}
	return nil
}

// InventarioTareasCierreAdministrativo identifica la lectura completa y
// bloqueada del libro autoritativo de tareas. No contiene nombres ni detalles
// funcionales que puedan convertirse en un oráculo.
type InventarioTareasCierreAdministrativo struct {
	Referencia string
	Version    uint64
	Total      uint32
	Pendientes uint32
	Completo   bool
}

func (i InventarioTareasCierreAdministrativo) Validar() error {
	if !i.Completo || !domain.ReferenciaOpacaValida(i.Referencia) ||
		i.Version == 0 || i.Pendientes > i.Total {
		return ErrPreparacionCierreAdministrativoInvalida
	}
	return nil
}

// PreparacionTransaccionCierreAdministrativo solo puede proceder de la
// implementación confiable del puerto, una vez bloqueados el seguimiento, el
// inventario de tareas y la autorización exacta. AutorizacionRef se consume
// en la misma transacción si el callback devuelve un estado nuevo.
type PreparacionTransaccionCierreAdministrativo struct {
	Solicitud             SolicitudTransaccionCierreAdministrativo
	Definicion            domain.DefinicionSeguimiento
	Seguimiento           domain.Seguimiento
	Inventario            InventarioTareasCierreAdministrativo
	AutorizacionRef       string
	AutorizacionConcedida bool
	ActorRef              string
	UnidadRef             string
	ActuacionRef          string
	ReciboRef             string
	CorrelacionRef        string
	Documentos            []domain.DocumentoSeguimiento
	EfectivoEn            time.Time
	RegistradaEn          time.Time
}

func (p PreparacionTransaccionCierreAdministrativo) ValidarPara(
	solicitud SolicitudTransaccionCierreAdministrativo,
) error {
	estado := p.Seguimiento.Estado()
	if solicitud.Validar() != nil || p.Solicitud != solicitud ||
		!p.AutorizacionConcedida ||
		!domain.ReferenciaOpacaValida(p.AutorizacionRef) ||
		!domain.ReferenciaOpacaValida(p.ActorRef) ||
		!domain.ReferenciaOpacaValida(p.UnidadRef) ||
		!domain.ReferenciaOpacaValida(p.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(p.ReciboRef) ||
		!domain.ReferenciaOpacaValida(p.CorrelacionRef) ||
		!domain.InstanteUTCCanonico(p.EfectivoEn) ||
		!domain.InstanteUTCCanonico(p.RegistradaEn) ||
		p.EfectivoEn.Before(p.RegistradaEn) ||
		p.Inventario.Validar() != nil || p.Definicion.Validar() != nil ||
		p.Seguimiento.Validar(p.Definicion) != nil ||
		estado.OrganizacionRef != solicitud.OrganizacionRef ||
		estado.ExpedienteRef != solicitud.ExpedienteRef ||
		estado.Referencia != solicitud.SeguimientoRef {
		return ErrPreparacionCierreAdministrativoInvalida
	}
	return nil
}

type datosResultadoCierreAdministrativo struct {
	solicitud         SolicitudTransaccionCierreAdministrativo
	versionResultante uint64
	actuacionRef      string
	reciboRef         string
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
}

func NuevoResultadoCierreAdministrativo(
	datos DatosResultadoCierreAdministrativo,
) (ResultadoCierreAdministrativo, error) {
	if datos.Solicitud.Validar() != nil ||
		datos.VersionResultante != datos.Solicitud.VersionEsperada+1 ||
		!domain.ReferenciaOpacaValida(datos.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(datos.ReciboRef) {
		return ResultadoCierreAdministrativo{},
			ErrResultadoCierreAdministrativoInvalido
	}
	return ResultadoCierreAdministrativo{datos: &datosResultadoCierreAdministrativo{
		solicitud: datos.Solicitud, versionResultante: datos.VersionResultante,
		actuacionRef: datos.ActuacionRef, reciboRef: datos.ReciboRef,
	}}, nil
}

func (r ResultadoCierreAdministrativo) ValidarPara(
	solicitud SolicitudTransaccionCierreAdministrativo,
	actuacionRef string,
	reciboRef string,
) error {
	if r.datos == nil || solicitud.Validar() != nil ||
		r.datos.solicitud != solicitud ||
		r.datos.versionResultante != solicitud.VersionEsperada+1 ||
		r.datos.actuacionRef != actuacionRef || r.datos.reciboRef != reciboRef {
		return ErrResultadoCierreAdministrativoInvalido
	}
	return nil
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

// TransaccionCierreAdministrativo ejecuta el callback una vez dentro de una
// transacción durable. La implementación debe bloquear y reacreditar el
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
