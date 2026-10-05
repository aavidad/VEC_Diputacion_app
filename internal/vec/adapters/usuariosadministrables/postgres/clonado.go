package postgres

import (
	"encoding/json"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func clonarIdentidadLectura(actor domain.ContextoActor, evidencia domain.EvidenciaSesionAdministracionPerfiles) (domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, error) {
	copiaActor, err := actor.Clonar()
	if err != nil {
		return domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	resultado, err := evidencia.ResultadoContexto.Clonar()
	if err != nil {
		return domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	copiaEvidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: evidencia.Vinculo}
	if copiaEvidencia.ValidarPara(copiaActor) != nil {
		return domain.ContextoActor{}, domain.EvidenciaSesionAdministracionPerfiles{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return copiaActor, copiaEvidencia, nil
}

func clonarInstantaneaLectura(i domain.InstantaneaAutorizacion) (domain.InstantaneaAutorizacion, error) {
	if i.Validar() != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	b, err := json.Marshal(i)
	if err != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	var copia domain.InstantaneaAutorizacion
	if json.Unmarshal(b, &copia) != nil || copia.Validar() != nil {
		return domain.InstantaneaAutorizacion{}, ports.ErrLecturaUsuariosAdministrablesNoDisponible
	}
	return copia, nil
}
