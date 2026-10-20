package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// AntecedentePortalPropuesta es la proyección de CT211 de una resolución de
// respuesta firme en Mi Bolsa. No contiene correo, comunicación ni justificante.
type AntecedentePortalPropuesta struct {
	FuenteRespuesta          string                   `json:"fuente_respuesta"`
	OrganizacionRef          string                   `json:"organizacion_ref"`
	ExpedienteRef            string                   `json:"expediente_ref"`
	LlamamientoRef           string                   `json:"llamamiento_ref"`
	ResolucionRef            string                   `json:"resolucion_ref"`
	ReciboResolucionRef      string                   `json:"recibo_resolucion_ref"`
	ResueltaEn               time.Time                `json:"resuelta_en"`
	VersionResultante        uint64                   `json:"version_resultante"`
	RespuestaPortalRef       string                   `json:"respuesta_portal_ref"`
	ReciboRespuestaPortalRef string                   `json:"recibo_respuesta_portal_ref"`
	ParticipacionRef         string                   `json:"participacion_ref"`
	RespondidaEn             time.Time                `json:"respondida_en"`
	EstadoPlazo              string                   `json:"estado_plazo"`
	RevisionRespuestaRRHH    bool                     `json:"revision_respuesta_rrhh"`
	RevisionPlazoRRHH        bool                     `json:"revision_plazo_rrhh"`
	CriterioValidacionRef    string                   `json:"criterio_validacion_ref"`
	Politica                 PoliticaPortalPropuesta  `json:"politica"`
	Seleccion                SeleccionPortalPropuesta `json:"seleccion"`
}

type PoliticaPortalPropuesta struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (p PoliticaPortalPropuesta) snapshot() SnapshotGobernadoFormalizacion {
	return SnapshotGobernadoFormalizacion{Referencia: p.Referencia, Version: p.Version, HuellaSHA256: p.HuellaSHA256}
}

// SeleccionPortalPropuesta copia solo referencias del recibo real de CT.
// B98 y B113 se consultan otra vez antes de preparar el registro.
type SeleccionPortalPropuesta struct {
	ClaveCT           string    `json:"clave_ct"`
	UnidadRef         string    `json:"unidad_ref"`
	CategoriaRef      string    `json:"categoria_ref"`
	NecesidadRef      string    `json:"necesidad_ref"`
	BolsaRef          string    `json:"bolsa_ref"`
	LlamamientoRef    string    `json:"llamamiento_ref"`
	ReciboEmisionRef  string    `json:"recibo_emision_ref"`
	ConfirmadaEn      time.Time `json:"confirmada_en"`
	VersionExpediente uint64    `json:"version_expediente"`
}

func (a AntecedentePortalPropuesta) Validar() error {
	if a.FuenteRespuesta != "portal" || a.VersionResultante != 1 ||
		a.EstadoPlazo != "revision_manual_portal" || !a.RevisionRespuestaRRHH || !a.RevisionPlazoRRHH ||
		!a.Politica.snapshot().valido() || a.CriterioValidacionRef != a.Politica.Referencia ||
		!ClaveIdempotenciaValida(a.Seleccion.ClaveCT) || !enteroSeguroBolsa(a.Seleccion.VersionExpediente) ||
		a.Seleccion.VersionExpediente < 6 {
		return ErrResultadoPropuestaFormalizacionNoConfiable
	}
	for _, ref := range []string{a.OrganizacionRef, a.ExpedienteRef, a.LlamamientoRef,
		a.ResolucionRef, a.ReciboResolucionRef, a.RespuestaPortalRef, a.ReciboRespuestaPortalRef,
		a.ParticipacionRef, a.Seleccion.UnidadRef, a.Seleccion.CategoriaRef,
		a.Seleccion.NecesidadRef, a.Seleccion.BolsaRef, a.Seleccion.LlamamientoRef,
		a.Seleccion.ReciboEmisionRef} {
		if !domain.ReferenciaOpacaValida(ref) {
			return ErrResultadoPropuestaFormalizacionNoConfiable
		}
	}
	for _, instante := range []time.Time{a.Seleccion.ConfirmadaEn, a.RespondidaEn, a.ResueltaEn} {
		if !domain.InstanteUTCCanonico(instante) || instante.Nanosecond()%1000 != 0 {
			return ErrResultadoPropuestaFormalizacionNoConfiable
		}
	}
	if a.Seleccion.ConfirmadaEn.After(a.RespondidaEn) || a.RespondidaEn.After(a.ResueltaEn) ||
		a.Seleccion.LlamamientoRef != a.LlamamientoRef {
		return ErrResultadoPropuestaFormalizacionNoConfiable
	}
	return nil
}

func (a AntecedentePortalPropuesta) ValidarPara(s SolicitudPropuestaFormalizacion) error {
	if (MaterialPropuestaFormalizacion{Etapa: "consulta", Solicitud: s}).Validar() != nil ||
		a.Validar() != nil || a.OrganizacionRef != s.OrganizacionRef ||
		a.ExpedienteRef != s.ExpedienteRef || a.LlamamientoRef != s.LlamamientoRef ||
		a.ResolucionRef != s.ResolucionLlamamientoAceptadaRef ||
		a.ReciboResolucionRef != s.ReciboResolucionAceptadaRef ||
		a.Seleccion.VersionExpediente != s.VersionEsperada {
		return ErrResultadoPropuestaFormalizacionNoConfiable
	}
	return nil
}
