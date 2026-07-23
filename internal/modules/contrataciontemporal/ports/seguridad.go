// Package ports define las fronteras hexagonales del módulo de contratación
// temporal.
package ports

import (
	"context"
	"errors"
	"regexp"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	AccionCrearSolicitud    = "contratacion_temporal.solicitud.crear"
	FinalidadCrearSolicitud = "tramitar_necesidad_personal_temporal"
	TipoRecursoExpediente   = "expediente_contratacion_temporal"
)

var (
	ErrIdentidadOperacionInvalida = errors.New("contratacion temporal: identidad de operacion invalida")
	ErrAutorizacionEfectoInvalida = errors.New("contratacion temporal: autorizacion de efecto invalida")
	ErrAutorizacionDenegada       = errors.New("contratacion temporal: autorizacion denegada")
)

var patronHuellaSHA256 = regexp.MustCompile(`^[a-f0-9]{64}$`)

type SuperficieOperacion string
type GarantiaOperacion string

const (
	SuperficieGestionInterna SuperficieOperacion = "gestion_interna"
	SuperficieAdministracion SuperficieOperacion = "administracion_privilegiada"
	GarantiaAlta             GarantiaOperacion   = "alta"
)

func (s SuperficieOperacion) valida() bool {
	return s == SuperficieGestionInterna || s == SuperficieAdministracion
}

// IdentidadOperacion es el resultado opaco de un resolutor confiable. No debe
// construirse con nombres, roles o cabeceras enviados por el navegador.
type IdentidadOperacion struct {
	datos *datosIdentidadOperacion
}

type datosIdentidadOperacion struct {
	actorRef, cuentaRef, perfilRef, contextoRegistroRef string
	superficie                                          SuperficieOperacion
	garantia                                            GarantiaOperacion
	resueltaEn, validaHasta                             time.Time
}

type DatosIdentidadOperacion struct {
	ActorRef            string
	CuentaRef           string
	PerfilRef           string
	ContextoRegistroRef string
	Superficie          SuperficieOperacion
	Garantia            GarantiaOperacion
	ResueltaEn          time.Time
	ValidaHasta         time.Time
}

func NuevaIdentidadOperacion(datos DatosIdentidadOperacion) (IdentidadOperacion, error) {
	if !domain.ReferenciaOpacaValida(datos.ActorRef) ||
		!domain.ReferenciaOpacaValida(datos.CuentaRef) ||
		!domain.ReferenciaOpacaValida(datos.PerfilRef) ||
		!domain.ReferenciaOpacaValida(datos.ContextoRegistroRef) ||
		!datos.Superficie.valida() || datos.Garantia != GarantiaAlta ||
		!domain.InstanteUTCCanonico(datos.ResueltaEn) ||
		!domain.InstanteUTCCanonico(datos.ValidaHasta) ||
		!datos.ValidaHasta.After(datos.ResueltaEn) {
		return IdentidadOperacion{}, ErrIdentidadOperacionInvalida
	}
	return IdentidadOperacion{datos: &datosIdentidadOperacion{
		actorRef: datos.ActorRef, cuentaRef: datos.CuentaRef,
		perfilRef: datos.PerfilRef, contextoRegistroRef: datos.ContextoRegistroRef,
		superficie: datos.Superficie, garantia: datos.Garantia,
		resueltaEn:  datos.ResueltaEn,
		validaHasta: datos.ValidaHasta,
	}}, nil
}

func (i IdentidadOperacion) Datos() (DatosIdentidadOperacion, error) {
	if i.datos == nil {
		return DatosIdentidadOperacion{}, ErrIdentidadOperacionInvalida
	}
	datos := DatosIdentidadOperacion{
		ActorRef: i.datos.actorRef, CuentaRef: i.datos.cuentaRef,
		PerfilRef: i.datos.perfilRef, ContextoRegistroRef: i.datos.contextoRegistroRef,
		Superficie: i.datos.superficie, Garantia: i.datos.garantia,
		ResueltaEn:  i.datos.resueltaEn,
		ValidaHasta: i.datos.validaHasta,
	}
	if _, err := NuevaIdentidadOperacion(datos); err != nil {
		return DatosIdentidadOperacion{}, err
	}
	return datos, nil
}

func (i IdentidadOperacion) VigenteEn(instante time.Time) bool {
	datos, err := i.Datos()
	return err == nil && domain.InstanteUTCCanonico(instante) &&
		!instante.Before(datos.ResueltaEn) && instante.Before(datos.ValidaHasta)
}

type SolicitudResolverIdentidad struct {
	SesionRef      string
	PerfilRef      string
	CorrelacionRef string
}

func (s SolicitudResolverIdentidad) Validar() error {
	if !domain.ReferenciaOpacaValida(s.SesionRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.ReferenciaOpacaValida(s.CorrelacionRef) {
		return ErrIdentidadOperacionInvalida
	}
	return nil
}

type ResolutorIdentidadOperacion interface {
	ResolverIdentidadOperacion(context.Context, SolicitudResolverIdentidad) (IdentidadOperacion, error)
}

type RecursoAltaExpediente struct {
	ExpedienteRef   string
	OrganizacionRef string
	CentroRef       string
	CategoriaRef    string
	FlujoRef        string
	FlujoVersion    uint64
}

func (r RecursoAltaExpediente) Validar() error {
	if !domain.ReferenciaOpacaValida(r.ExpedienteRef) ||
		!domain.ReferenciaOpacaValida(r.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(r.CentroRef) ||
		!domain.ReferenciaOpacaValida(r.CategoriaRef) ||
		!domain.ReferenciaOpacaValida(r.FlujoRef) || r.FlujoVersion == 0 {
		return ErrAutorizacionEfectoInvalida
	}
	return nil
}

type SolicitudAutorizarAlta struct {
	Identidad      IdentidadOperacion
	Recurso        RecursoAltaExpediente
	MotivoRef      string
	CorrelacionRef string
	SolicitadaEn   time.Time
}

func (s SolicitudAutorizarAlta) Validar() error {
	if _, err := s.Identidad.Datos(); err != nil || s.Recurso.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.MotivoRef) ||
		!domain.ReferenciaOpacaValida(s.CorrelacionRef) ||
		!domain.InstanteUTCCanonico(s.SolicitadaEn) ||
		!s.Identidad.VigenteEn(s.SolicitadaEn) {
		return ErrAutorizacionEfectoInvalida
	}
	return nil
}

// AutorizacionEfecto solo solicita a la transacción que coteje y consuma una
// decisión ya confirmada. No concede por sí misma ni evita suplantar el puerto.
type AutorizacionEfecto struct {
	datos *datosAutorizacionEfecto
}

type datosAutorizacionEfecto struct {
	decisionRef, huellaSHA256, accion, recursoRef, actorRef, perfilRef string
	emitidaEn, validaHasta                                             time.Time
}

type DatosAutorizacionEfecto struct {
	DecisionRef  string
	HuellaSHA256 string
	Accion       string
	RecursoRef   string
	ActorRef     string
	PerfilRef    string
	EmitidaEn    time.Time
	ValidaHasta  time.Time
}

func NuevaAutorizacionEfecto(datos DatosAutorizacionEfecto) (AutorizacionEfecto, error) {
	if !domain.ReferenciaOpacaValida(datos.DecisionRef) ||
		!patronHuellaSHA256.MatchString(datos.HuellaSHA256) ||
		datos.Accion != AccionCrearSolicitud ||
		!domain.ReferenciaOpacaValida(datos.RecursoRef) ||
		!domain.ReferenciaOpacaValida(datos.ActorRef) ||
		!domain.ReferenciaOpacaValida(datos.PerfilRef) ||
		!domain.InstanteUTCCanonico(datos.EmitidaEn) ||
		!domain.InstanteUTCCanonico(datos.ValidaHasta) ||
		!datos.ValidaHasta.After(datos.EmitidaEn) {
		return AutorizacionEfecto{}, ErrAutorizacionEfectoInvalida
	}
	return AutorizacionEfecto{datos: &datosAutorizacionEfecto{
		decisionRef: datos.DecisionRef, huellaSHA256: datos.HuellaSHA256,
		accion: datos.Accion, recursoRef: datos.RecursoRef,
		actorRef: datos.ActorRef, perfilRef: datos.PerfilRef,
		emitidaEn: datos.EmitidaEn, validaHasta: datos.ValidaHasta,
	}}, nil
}

func (a AutorizacionEfecto) Datos() (DatosAutorizacionEfecto, error) {
	if a.datos == nil {
		return DatosAutorizacionEfecto{}, ErrAutorizacionEfectoInvalida
	}
	datos := DatosAutorizacionEfecto{
		DecisionRef: a.datos.decisionRef, HuellaSHA256: a.datos.huellaSHA256,
		Accion: a.datos.accion, RecursoRef: a.datos.recursoRef,
		ActorRef: a.datos.actorRef, PerfilRef: a.datos.perfilRef,
		EmitidaEn: a.datos.emitidaEn, ValidaHasta: a.datos.validaHasta,
	}
	if _, err := NuevaAutorizacionEfecto(datos); err != nil {
		return DatosAutorizacionEfecto{}, err
	}
	return datos, nil
}

type AutorizadorAltaExpediente interface {
	AutorizarAltaExpediente(context.Context, SolicitudAutorizarAlta) (AutorizacionEfecto, error)
}
