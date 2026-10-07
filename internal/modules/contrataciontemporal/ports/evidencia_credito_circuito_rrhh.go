package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// SolicitudEvidenciaCreditoCircuitoRRHH liga la consulta de la huella documental
// a la RC ya validada y persistida. La fuente no recibe datos libres del canal.
type SolicitudEvidenciaCreditoCircuitoRRHH struct {
	OrganizacionRef string
	ExpedienteRef   string
	VersionEntrada  uint64
	Flujo           domain.ReferenciaFlujo
	ActorRef        string
	PerfilRef       string
	CreditoRef      string
	DocumentoRef    string
}

func (s SolicitudEvidenciaCreditoCircuitoRRHH) Validar() error {
	if !domain.ReferenciaOpacaValida(s.OrganizacionRef) ||
		!domain.ReferenciaOpacaValida(s.ExpedienteRef) || s.VersionEntrada == 0 ||
		s.Flujo.Validar() != nil || !domain.ReferenciaOpacaValida(s.CreditoRef) ||
		!domain.ReferenciaOpacaValida(s.ActorRef) ||
		!domain.ReferenciaOpacaValida(s.PerfilRef) ||
		!domain.ReferenciaOpacaValida(s.DocumentoRef) {
		return ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	return nil
}

func NuevaSolicitudEvidenciaCreditoCircuitoRRHH(
	credito domain.DatosCreditoCircuitoRRHH,
	actorRef, perfilRef string,
) (SolicitudEvidenciaCreditoCircuitoRRHH, error) {
	s := SolicitudEvidenciaCreditoCircuitoRRHH{
		OrganizacionRef: credito.OrganizacionRef,
		ExpedienteRef:   credito.ExpedienteRef,
		VersionEntrada:  credito.VersionEntrada,
		Flujo:           credito.Flujo,
		ActorRef:        actorRef,
		PerfilRef:       perfilRef,
		CreditoRef:      credito.CreditoRef,
		DocumentoRef:    credito.DocumentoRef,
	}
	return s, s.Validar()
}

// EvidenciaCreditoCircuitoRRHH aporta la definición publicada y la huella
// calculada por la autoridad documental. La huella de entrada de análisis
// representa otro material y nunca se usa como huella de este documento.
type EvidenciaCreditoCircuitoRRHH struct {
	Solicitud             SolicitudEvidenciaCreditoCircuitoRRHH
	Definicion            domain.DefinicionCircuitoRRHH
	DocumentoVersion      uint64
	HuellaDocumentoSHA256 string
}

func (e EvidenciaCreditoCircuitoRRHH) ValidarPara(s SolicitudEvidenciaCreditoCircuitoRRHH) error {
	if s.Validar() != nil || e.Solicitud != s || e.Definicion.Validar() != nil ||
		e.Definicion.Flujo != s.Flujo ||
		e.DocumentoVersion == 0 || e.DocumentoVersion > 9007199254740991 ||
		!domain.HuellaSHA256FirmaValida(e.HuellaDocumentoSHA256) {
		return ErrEvidenciasCircuitoRRHHNoDisponibles
	}
	return nil
}

func (e EvidenciaCreditoCircuitoRRHH) Clonar() EvidenciaCreditoCircuitoRRHH {
	e.Definicion.Transiciones = append([]domain.TransicionCircuitoRRHH(nil), e.Definicion.Transiciones...)
	for i := range e.Definicion.Transiciones {
		e.Definicion.Transiciones[i].FirmasRequeridas = append(
			[]domain.ClaveCatalogo(nil), e.Definicion.Transiciones[i].FirmasRequeridas...,
		)
	}
	return e
}

// FuenteEvidenciaCreditoCircuitoRRHH debe comprobar el documento en su propia
// autoridad. Su ausencia impide confirmar la decisión del circuito nuevo.
type FuenteEvidenciaCreditoCircuitoRRHH interface {
	AcreditarCreditoCircuitoRRHH(context.Context, SolicitudEvidenciaCreditoCircuitoRRHH) (EvidenciaCreditoCircuitoRRHH, error)
}
