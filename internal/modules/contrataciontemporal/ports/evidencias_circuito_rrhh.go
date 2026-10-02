package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

var ErrEvidenciasCircuitoRRHHNoDisponibles = errors.New(
	"contratacion temporal: evidencias del circuito RRHH no disponibles",
)

// SolicitudEvidenciasAnalisisCircuitoRRHH solo identifica el expediente y el
// registrador CT. La fuente resuelve el acto de Dirección y las firmas del
// técnico y Delegación por autoridad corporativa, nunca desde el comando web.
type SolicitudEvidenciasAnalisisCircuitoRRHH struct {
	OrganizacionRef string
	ExpedienteRef   string
	VersionEntrada  uint64
	Flujo           domain.ReferenciaFlujo
	ActorRef        string
	PerfilRef       string
}

func (s SolicitudEvidenciasAnalisisCircuitoRRHH) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		s.VersionEntrada == 0 || s.Flujo.Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) {
		return ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	return nil
}

type EvidenciasAnalisisCircuitoRRHH struct {
	Definicion               domain.DefinicionCircuitoRRHH
	PerfilRegistradorClave   domain.ClaveCatalogo
	DocumentoPeticionRef     string
	HuellaPeticionSHA256     string
	FirmasPeticion           []domain.FirmaCircuitoRRHH
	DocumentoAutorizacionRef string
	HuellaAutorizacionSHA256 string
	ActoAutorizacionRef      string
	AutorizanteRef           string
	CargoAutorizanteClave    domain.ClaveCatalogo
}

func (e EvidenciasAnalisisCircuitoRRHH) ValidarPara(s SolicitudEvidenciasAnalisisCircuitoRRHH) error {
	if s.Validar() != nil || e.Definicion.Validar() != nil ||
		e.Definicion.Flujo != s.Flujo ||
		!e.PerfilRegistradorClave.Valida() ||
		!domain.ReferenciaOpacaValida(e.DocumentoPeticionRef) ||
		!domain.HuellaSHA256FirmaValida(e.HuellaPeticionSHA256) ||
		len(e.FirmasPeticion) == 0 ||
		!domain.ReferenciaOpacaValida(e.DocumentoAutorizacionRef) ||
		!domain.HuellaSHA256FirmaValida(e.HuellaAutorizacionSHA256) ||
		!domain.ReferenciaOpacaValida(e.ActoAutorizacionRef) ||
		!domain.ReferenciaOpacaValida(e.AutorizanteRef) ||
		!e.CargoAutorizanteClave.Valida() {
		return ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	for _, firma := range e.FirmasPeticion {
		if !firma.CargoClave.Valida() ||
			!domain.ReferenciaOpacaValida(firma.FirmaRef) ||
			!domain.ReferenciaOpacaValida(firma.FirmanteRef) ||
			firma.HuellaDocumentoSHA256 != e.HuellaPeticionSHA256 {
			return ErrEvidenciasCircuitoRRHHNoDisponibles
		}
	}
	return nil
}

// FuenteEvidenciasCircuitoRRHH solo se compone con una autoridad que acredita
// firma, cargo y acto. Sin fuente el circuito nuevo se deniega antes del efecto.
type FuenteEvidenciasCircuitoRRHH interface {
	AcreditarAnalisisCircuitoRRHH(
		context.Context,
		SolicitudEvidenciasAnalisisCircuitoRRHH,
	) (EvidenciasAnalisisCircuitoRRHH, error)
}
