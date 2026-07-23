package ports

import (
	"bytes"
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type AcuseEventoLlamamientoBolsa struct {
	AutoridadRef         string    `json:"autoridad_ref"`
	EventoRef            string    `json:"evento_ref"`
	OrganizacionRef      string    `json:"organizacion_ref"`
	ExpedienteRef        string    `json:"expediente_ref"`
	CorrelacionRef       string    `json:"correlacion_ref"`
	PeticionRef          string    `json:"peticion_ref"`
	HuellaPeticionSHA256 string    `json:"huella_peticion_sha256"`
	ReciboRef            string    `json:"recibo_ref"`
	HuellaReciboSHA256   string    `json:"huella_recibo_sha256"`
	HuellaEventoSHA256   string    `json:"huella_evento_sha256"`
	SecuenciaAnterior    uint64    `json:"secuencia_anterior"`
	Secuencia            uint64    `json:"secuencia"`
	VersionAnterior      uint64    `json:"version_anterior"`
	VersionResultante    uint64    `json:"version_resultante"`
	ActuacionRef         string    `json:"actuacion_ref"`
	AuditoriaRef         string    `json:"auditoria_ref"`
	InboxRef             string    `json:"inbox_ref"`
	RegistradoEn         time.Time `json:"registrado_en"`
}

func (a AcuseEventoLlamamientoBolsa) ValidarPara(evento EventoLlamamientoBolsa) error {
	if evento.validarEstructuraDurable() != nil ||
		a.AutoridadRef != evento.Procedencia.AutoridadRef ||
		a.EventoRef != evento.EventoRef ||
		a.OrganizacionRef != evento.OrganizacionRef ||
		a.ExpedienteRef != evento.ExpedienteRef ||
		a.CorrelacionRef != evento.CorrelacionRef ||
		a.PeticionRef != evento.PeticionRef ||
		!huellasBolsaIguales(a.HuellaPeticionSHA256, evento.HuellaPeticionSHA256) ||
		a.ReciboRef != evento.ReciboRef ||
		!huellasBolsaIguales(a.HuellaReciboSHA256, evento.HuellaReciboSHA256) ||
		a.HuellaEventoSHA256 != evento.HuellaCargaSHA256 ||
		a.SecuenciaAnterior != evento.SecuenciaAnterior ||
		a.Secuencia != evento.Secuencia ||
		a.VersionAnterior != evento.VersionExpedienteEsperada ||
		evento.VersionExpedienteEsperada >= MaximoEnteroSeguroIntegracionBolsa ||
		a.VersionResultante != evento.VersionExpedienteEsperada+1 ||
		!domain.ReferenciaOpacaValida(a.ActuacionRef) ||
		!domain.ReferenciaOpacaValida(a.AuditoriaRef) ||
		!domain.ReferenciaOpacaValida(a.InboxRef) ||
		!instanteBolsaCanonico(a.RegistradoEn) ||
		a.RegistradoEn.Before(evento.PublicadoEn) {
		return ErrAcuseEventoBolsaNoConfiable
	}
	return nil
}

func ValidarReplayAcuseEventoBolsa(
	primero AcuseEventoLlamamientoBolsa,
	repetido AcuseEventoLlamamientoBolsa,
	evento EventoLlamamientoBolsa,
) error {
	if primero.ValidarPara(evento) != nil || repetido.ValidarPara(evento) != nil ||
		primero != repetido {
		return ErrAcuseEventoBolsaNoConfiable
	}
	return nil
}

func ValidarIdentidadEventoBolsa(
	primero EventoLlamamientoBolsa,
	repetido EventoLlamamientoBolsa,
) error {
	mismaIdentidad := primero.Procedencia.AutoridadRef == repetido.Procedencia.AutoridadRef &&
		primero.EventoRef == repetido.EventoRef
	if !mismaIdentidad ||
		!domain.ReferenciaOpacaValida(primero.Procedencia.AutoridadRef) ||
		!domain.ReferenciaOpacaValida(primero.EventoRef) {
		return ErrEventoBolsaInvalido
	}
	if primero.HuellaCargaSHA256 != repetido.HuellaCargaSHA256 ||
		!bytes.Equal(materialEventoBolsa(primero), materialEventoBolsa(repetido)) {
		return ErrColisionEventoBolsa
	}
	if primero.validarEstructuraDurable() != nil ||
		repetido.validarEstructuraDurable() != nil {
		return ErrEventoBolsaInvalido
	}
	return nil
}

type BandejaEventosLlamamientoBolsa interface {
	RegistrarEventoLlamamiento(
		context.Context,
		ComandoRegistrarEventoBolsa,
	) (AcuseEventoLlamamientoBolsa, error)
}
