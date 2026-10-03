package domain

import (
	"errors"
	"regexp"
	"time"
)

var ErrCargoCompetencialInvalido = errors.New("personal: cargo competencial invalido")

var (
	patronCargoCompetencial = regexp.MustCompile(`^car_[A-Za-z0-9_-]{22,128}$`)
	patronEnlaceCargo       = regexp.MustCompile(`^enc_[A-Za-z0-9_-]{22,128}$`)
	patronHuellaCargo       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// ReferenciaCargoCompetencial identifica una version publicada de Personal.
// El identificador de un perfil de acceso no equivale a esta referencia.
type ReferenciaCargoCompetencial struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaCargoCompetencial) Validar() error {
	if !patronCargoCompetencial.MatchString(r.Referencia) || r.Version == 0 ||
		!patronHuellaCargo.MatchString(r.HuellaSHA256) {
		return ErrCargoCompetencialInvalido
	}
	return nil
}

// ClaseEjercicio mantiene separados titularidad, delegacion de competencia,
// delegacion de firma y suplencia. Cada una requiere un acto propio.
type ClaseEjercicio string

const (
	EjercicioTitular               ClaseEjercicio = "titular"
	EjercicioDelegacionCompetencia ClaseEjercicio = "delegacion_competencia"
	EjercicioDelegacionFirma       ClaseEjercicio = "delegacion_firma"
	EjercicioSuplencia             ClaseEjercicio = "suplencia"
)

type EnlaceCargoCompetencial struct {
	Referencia   string         `json:"referencia"`
	Version      uint64         `json:"version"`
	HuellaSHA256 string         `json:"huella_sha256"`
	CargoRef     string         `json:"cargo_ref"`
	PersonaRef   string         `json:"persona_ref"`
	Clase        ClaseEjercicio `json:"clase"`
	Desde        time.Time      `json:"vigente_desde"`
	Hasta        time.Time      `json:"vigente_hasta"`
	ActoRef      string         `json:"acto_ref"`
}

func (e EnlaceCargoCompetencial) Validar() error {
	if !patronEnlaceCargo.MatchString(e.Referencia) || e.Version == 0 ||
		!patronHuellaCargo.MatchString(e.HuellaSHA256) || !patronCargoCompetencial.MatchString(e.CargoRef) ||
		!patronPersonaEmpleado.MatchString(e.PersonaRef) || !patronReferenciaFuente.MatchString(e.ActoRef) ||
		e.Desde.IsZero() || e.Hasta.IsZero() || !e.Hasta.After(e.Desde) {
		return ErrCargoCompetencialInvalido
	}
	switch e.Clase {
	case EjercicioTitular, EjercicioDelegacionCompetencia, EjercicioDelegacionFirma, EjercicioSuplencia:
		return nil
	default:
		return ErrCargoCompetencialInvalido
	}
}
