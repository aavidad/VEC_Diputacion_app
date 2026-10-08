package bootstrap

import (
	"net/http"

	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
)

// sesionAspiranteInscripcionExterna adapta la resolución nominal común de
// Área personal al contrato de Bolsa. No crea ni conserva otra sesión.
type sesionAspiranteInscripcionExterna struct {
	fuente *SesionExternaInscripcion
}

var _ SesionInscripcionBolsa = (*sesionAspiranteInscripcionExterna)(nil)

func NuevaSesionAspiranteInscripcionExterna(fuente *SesionExternaInscripcion) (SesionInscripcionBolsa, error) {
	if fuente == nil {
		return nil, inscripcion.ErrNoDisponible
	}
	return &sesionAspiranteInscripcionExterna{fuente: fuente}, nil
}

func (s *sesionAspiranteInscripcionExterna) ResolverInscripcion(r *http.Request) (contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, error) {
	var vacio contextoSeguridadComunDesarrollo
	var sin AcreditacionSesionInscripcionBolsa
	if s == nil || s.fuente == nil {
		return vacio, sin, inscripcion.ErrNoDisponible
	}
	ctx, acreditacion, err := s.fuente.ResolverSesionExterna(r)
	if err != nil {
		return vacio, sin, err
	}
	return ctx, AcreditacionSesionInscripcionBolsa{
		CertificadoHuellaSHA256: acreditacion.CertificadoHuellaSHA256,
		Canal:                   acreditacion.Canal,
		PersonaRef:              acreditacion.PersonaRef,
		PerfilRef:               acreditacion.PerfilRef,
		CuentaRef:               acreditacion.CuentaRef,
		SesionRef:               acreditacion.SesionRef,
		AutenticacionRef:        acreditacion.AutenticacionRef,
		VerificadaEn:            acreditacion.VerificadaEn,
		ValidaHasta:             acreditacion.ValidaHasta,
	}, nil
}
