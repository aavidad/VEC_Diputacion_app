package domain

import "time"

// EnlaceOcupanteCompetencialV1 conserva el enlace nominal de Personal, no
// deduce el ocupante de la etiqueta del cargo ni de una cuenta de acceso.
// En una delegacion, este enlace pertenece a la persona delegante; la persona
// firmante conserva su asignacion RBAC propia y el acto de delegacion vigente.
type EnlaceOcupanteCompetencialV1 struct {
	Referencia, HuellaSHA256, PersonaRef, CargoRef string
	Version                                        uint64
	VigenteDesde, VigenteHasta                     time.Time
}

func (e EnlaceOcupanteCompetencialV1) ValidarParaEn(personaRef, cargoRef string, instante time.Time) error {
	if !textoAutorizacionSinComodinSeguro(e.Referencia, 512, false) || e.Version == 0 ||
		!huellaAsignacionCompetencialV1Valida(e.HuellaSHA256) ||
		!referenciaOpacaContextoActorValida(personaRef, "per_") || e.PersonaRef != personaRef ||
		!textoAutorizacionSinComodinSeguro(cargoRef, 512, false) || e.CargoRef != cargoRef ||
		!ventanaAsignacionCompetencialV1Vigente(e.VigenteDesde, e.VigenteHasta, instante) {
		return ErrEvidenciaAsignacionCompetencialV1Invalida
	}
	return nil
}

// DelegacionCompetencialV1 identifica el acto versionado de delegacion o
// suplencia cuando la competencia lo necesita. Su ausencia nunca se completa
// con una delegacion sintetica; la fuente decide si el circuito la requiere.
type DelegacionCompetencialV1 struct {
	Referencia, HuellaSHA256, DelegantePersonaRef, DelegadoPersonaRef, CargoRef string
	Version                                                                     uint64
	VigenteDesde, VigenteHasta                                                  time.Time
}

func (d DelegacionCompetencialV1) ValidarParaEn(personaRef, cargoRef string, instante time.Time) error {
	if !textoAutorizacionSinComodinSeguro(d.Referencia, 512, false) || d.Version == 0 ||
		!huellaAsignacionCompetencialV1Valida(d.HuellaSHA256) ||
		!referenciaOpacaContextoActorValida(d.DelegantePersonaRef, "per_") ||
		!referenciaOpacaContextoActorValida(personaRef, "per_") || d.DelegadoPersonaRef != personaRef ||
		d.DelegantePersonaRef == personaRef ||
		!textoAutorizacionSinComodinSeguro(cargoRef, 512, false) || d.CargoRef != cargoRef ||
		!ventanaAsignacionCompetencialV1Vigente(d.VigenteDesde, d.VigenteHasta, instante) {
		return ErrEvidenciaAsignacionCompetencialV1Invalida
	}
	return nil
}

func ventanaAsignacionCompetencialV1Vigente(desde, hasta, instante time.Time) bool {
	return instanteAutorizacionCanonico(desde) && instanteAutorizacionCanonico(hasta) &&
		instanteAutorizacionCanonico(instante) && hasta.After(desde) && !instante.Before(desde) && instante.Before(hasta)
}
