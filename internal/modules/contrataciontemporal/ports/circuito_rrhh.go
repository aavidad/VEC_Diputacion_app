package ports

import (
	"context"
	"errors"
	"math"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConsultaCircuitoRRHHInvalida     = errors.New("contratacion temporal: consulta de circuito invalida")
	ErrConsultaCircuitoRRHHDenegada     = errors.New("contratacion temporal: consulta de circuito denegada")
	ErrConsultaCircuitoRRHHNoDisponible = errors.New("contratacion temporal: consulta de circuito no disponible")
	ErrResultadoCircuitoRRHHNoConfiable = errors.New("contratacion temporal: resultado de circuito no confiable")
)

// Los datos de identidad y organización proceden del canal autenticado. El
// cuerpo HTTP solo puede indicar expediente y versión observada.
type SolicitudConsultaCircuitoRRHH struct {
	AutenticacionRef string
	SesionRef        string
	PerfilRef        string
	OrganizacionRef  string
	ExpedienteRef    string
	VersionObservada uint64
}

func (s SolicitudConsultaCircuitoRRHH) Validar() error {
	if (SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: s.AutenticacionRef,
		SesionRef:        s.SesionRef,
		PerfilRef:        s.PerfilRef,
	}).Validar() != nil || !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionObservada == 0 || s.VersionObservada > math.MaxInt64 {
		return ErrConsultaCircuitoRRHHInvalida
	}
	return nil
}

// Preparar traduce el vínculo autenticado a una autorización V3 nominal. El
// lector SQL consume esa capacidad dentro de la misma transacción auditada.
type PreparadorConsultaCircuitoRRHH interface {
	PrepararConsultaCircuitoRRHH(context.Context, SolicitudConsultaCircuitoRRHH) (MaterialConsultaCircuitoRRHH, error)
}

type MaterialConsultaCircuitoRRHH struct {
	Solicitud    SolicitudConsultaCircuitoRRHH
	ActorRef     string
	PerfilRef    string
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (m MaterialConsultaCircuitoRRHH) ValidarPara(s SolicitudConsultaCircuitoRRHH) error {
	if s.Validar() != nil || m.Solicitud != s ||
		!domain.ReferenciaOpacaValida(m.ActorRef) ||
		m.PerfilRef != s.PerfilRef || m.Autorizacion.ValidarEstructura() != nil {
		return ErrConsultaCircuitoRRHHDenegada
	}
	return nil
}

type LectorCircuitoRRHH interface {
	ConsultarCircuitoRRHH(context.Context, MaterialConsultaCircuitoRRHH) (ResultadoConsultaCircuitoRRHH, error)
}

type ResultadoConsultaCircuitoRRHH struct {
	// ExpedienteRef se comprueba internamente y nunca se serializa en la vista.
	ExpedienteRef          string
	Flujo                  domain.ReferenciaFlujo
	Circuito               domain.CircuitoAdministrativo
	VersionExpediente      uint64
	TransicionesPermitidas []domain.TransicionCircuitoRRHH
}

func (r ResultadoConsultaCircuitoRRHH) ValidarPara(s SolicitudConsultaCircuitoRRHH) error {
	if s.Validar() != nil || r.ExpedienteRef != s.ExpedienteRef ||
		r.VersionExpediente != s.VersionObservada || r.Flujo.Validar() != nil ||
		r.Circuito.Definicion != r.Flujo || !r.Circuito.EstadoActual.Valida() ||
		r.Circuito.Hitos == nil || len(r.Circuito.Hitos) > 128 ||
		len(r.TransicionesPermitidas) != 0 {
		return ErrResultadoCircuitoRRHHNoConfiable
	}
	var versionAnterior uint64
	var destinoAnterior domain.ClaveFase
	for i, hito := range r.Circuito.Hitos {
		if hito.Secuencia != uint64(i+1) ||
			hito.VersionExpedienteEntrada < versionAnterior ||
			hito.VersionExpedienteEntrada >= r.VersionExpediente ||
			!hito.Clave.Valida() || !hito.Tipo.Valido() ||
			!hito.Origen.Valida() || !hito.Destino.Valida() ||
			(i > 0 && hito.Origen != destinoAnterior) ||
			!domain.ReferenciaOpacaValida(hito.ReciboRef) ||
			!domain.InstanteUTCCanonico(hito.RegistradoEn) {
			return ErrResultadoCircuitoRRHHNoConfiable
		}
		versionAnterior, destinoAnterior = hito.VersionExpedienteEntrada, hito.Destino
	}
	if len(r.Circuito.Hitos) != 0 && destinoAnterior != r.Circuito.EstadoActual {
		return ErrResultadoCircuitoRRHHNoConfiable
	}
	return nil
}
