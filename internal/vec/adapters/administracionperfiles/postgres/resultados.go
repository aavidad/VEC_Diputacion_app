package postgres

import (
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

type reciboJSON struct {
	CentroRef           string                            `json:"centro_ref"`
	ActorPersonaRef     string                            `json:"actor_persona_ref"`
	PerfilActivoRef     string                            `json:"perfil_activo_ref"`
	AsignacionPerfilRef string                            `json:"asignacion_perfil_ref"`
	CorrelacionRef      string                            `json:"correlacion_ref"`
	RolVersionRef       string                            `json:"rol_version_ref"`
	VigenteDesde        time.Time                         `json:"vigente_desde"`
	VigenteHasta        time.Time                         `json:"vigente_hasta"`
	Motivo              domain.ReferenciaEntradaCatalogo  `json:"motivo"`
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

// utcDesplazamientoCero devuelve en time.UTC un instante con desplazamiento
// cero. jsonb serializa timestamptz con TimeZone=UTC como «…+00:00» y
// encoding/json lo deja en una zona fija, no en time.UTC; el dominio exige UTC
// canónico. Un desplazamiento distinto de cero no se toca: sigue siendo una
// proyección ajena y se rechaza más adelante.
func utcDesplazamientoCero(t time.Time) time.Time {
	if _, desplazamiento := t.Zone(); desplazamiento == 0 && !t.IsZero() {
		return t.UTC()
	}
	return t
}

// normalizarUTC aplica utcDesplazamientoCero a los instantes del recibo.
func (x *reciboJSON) normalizarUTC() {
	x.VigenteDesde = utcDesplazamientoCero(x.VigenteDesde)
	x.VigenteHasta = utcDesplazamientoCero(x.VigenteHasta)
	x.ConfirmadoEn = utcDesplazamientoCero(x.ConfirmadoEn)
}

// La vigencia de una revocación procede de la asignación histórica conservada;
// la preimagen de revocación no la contiene y nunca puede rellenarla.
func (x reciboJSON) vigenciaHistoricaCompleta() bool {
	return instantePersistible(x.VigenteDesde) && instantePersistible(x.VigenteHasta) &&
		x.VigenteDesde.Location() == time.UTC && x.VigenteHasta.Location() == time.UTC &&
		x.VigenteHasta.After(x.VigenteDesde)
}

func (x reciboJSON) dominio() domain.ReciboAdministracionPerfiles {
	return domain.ReciboAdministracionPerfiles{CentroRef: x.CentroRef, ActorPersonaRef: x.ActorPersonaRef,
		PerfilActivoRef: x.PerfilActivoRef, AsignacionPerfilRef: x.AsignacionPerfilRef, CorrelacionRef: x.CorrelacionRef,
		RolVersionRef: x.RolVersionRef, VigenteDesde: x.VigenteDesde, VigenteHasta: x.VigenteHasta, Motivo: x.Motivo,
		OperacionRef: x.OperacionRef, ActoRef: x.ActoRef, ReciboRef: x.ReciboRef,
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
	PropuestaHuellaSHA256 string                                         `json:"propuesta_huella_sha256"`
	OperacionRef          string                                         `json:"operacion_ref"`
	PropuestaRef          string                                         `json:"propuesta_ref"`
	Decision              domain.DecisionPropuestaAdministracionPerfiles `json:"decision"`
	HuellaCierreSHA256    string                                         `json:"huella_cierre_sha256"`
	ConfirmadoEn          time.Time                                      `json:"confirmado_en"`
	Recibo                *reciboJSON                                    `json:"recibo"`
}
