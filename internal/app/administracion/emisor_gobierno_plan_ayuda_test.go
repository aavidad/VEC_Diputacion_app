package administracion

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	"vec-diputacion-granada/internal/vec/domain"
)

func ambitoDePrueba(s domain.InstantaneaAutorizacion) (plannominal.AmbitoGobiernoPlanFirma, error) {
	return plannominal.AmbitoGobiernoPlanFirmaDeAsignacion(s.AsignacionPerfil)
}

func recursoDePrueba(material []byte, ambito plannominal.AmbitoGobiernoPlanFirma) (string, domain.RecursoAutorizable, error) {
	return plannominal.RecursoGobiernoPlanFirma(material, ambito)
}
