package postgres

import (
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

type reciboJSON struct {
	OperacionRef        string                            `json:"operacion_ref"`
	ActoRef             string                            `json:"acto_ref"`
	ReciboRef           string                            `json:"recibo_ref"`
	PropuestaRef        string                            `json:"propuesta_ref"`
	AuditoriaRef        string                            `json:"auditoria_ref"`
	ObjetivoPersonaRef  string                            `json:"objetivo_persona_ref"`
	PerfilRef           string                            `json:"perfil_ref"`
	VinculoRef          string                            `json:"vinculo_ref"`
	EstadoPosterior     domain.EstadoVinculoContextoActor `json:"estado_posterior"`
	VersionPosterior    uint64                            `json:"version_posterior"`
	HuellaAntesSHA256   string                            `json:"huella_antes_sha256"`
	HuellaDespuesSHA256 string                            `json:"huella_despues_sha256"`
	ConfirmadoEn        time.Time                         `json:"confirmado_en"`
	UnidadRef           string                            `json:"unidad_ref"`
	ReferenciaActo      string                            `json:"referencia_acto"`
}

func (x reciboJSON) dominio() domain.ReciboAdministracionPerfiles {
	return domain.ReciboAdministracionPerfiles{OperacionRef: x.OperacionRef, ActoRef: x.ActoRef, ReciboRef: x.ReciboRef,
		PropuestaRef: x.PropuestaRef, AuditoriaRef: x.AuditoriaRef, ObjetivoPersonaRef: x.ObjetivoPersonaRef,
		PerfilRef: x.PerfilRef, VinculoRef: x.VinculoRef, EstadoPosterior: x.EstadoPosterior, VersionPosterior: x.VersionPosterior,
		HuellaAntesSHA256: x.HuellaAntesSHA256, HuellaDespuesSHA256: x.HuellaDespuesSHA256, ConfirmadoEn: x.ConfirmadoEn,
		UnidadRef: x.UnidadRef, ReferenciaActo: x.ReferenciaActo}
}

type propuestaJSON struct {
	OperacionRef         string    `json:"operacion_ref"`
	PropuestaRef         string    `json:"propuesta_ref"`
	HuellaSHA256         string    `json:"huella_sha256"`
	ProponentePersonaRef string    `json:"proponente_persona_ref"`
	ObjetivoPersonaRef   string    `json:"objetivo_persona_ref"`
	CaducaEn             time.Time `json:"caduca_en"`
}

type cierreResultadoJSON struct {
	OperacionRef       string                                         `json:"operacion_ref"`
	PropuestaRef       string                                         `json:"propuesta_ref"`
	Decision           domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	HuellaCierreSHA256 string                                         `json:"huella_cierre_sha256"`
	ConfirmadoEn       time.Time                                      `json:"confirmado_en"`
	Recibo             *reciboJSON                                    `json:"recibo"`
}
