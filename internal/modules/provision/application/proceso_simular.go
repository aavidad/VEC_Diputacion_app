package application

import (
	"slices"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

// PrepararProceso valida el borrador suministrado. No lee fuentes oficiales,
// publica bases ni crea un efecto administrativo.
func PrepararProceso(p domain.ProcesoProvision) (domain.ProcesoProvision, error) {
	if err := domain.ValidarProceso(p); err != nil {
		return domain.ProcesoProvision{}, err
	}
	return domain.CopiarProceso(p), nil
}

// SimularProceso valora cada preferencia contra el mismo conjunto versionado
// de reglas usando el motor existente. No tiene repositorio, firma ni registro.
func SimularProceso(p ports.PeticionProceso) (domain.ResultadoProceso, error) {
	if err := domain.ValidarSolicitud(p.Proceso, p.Solicitud); err != nil {
		return domain.ResultadoProceso{}, err
	}
	p.Proceso = domain.CopiarProceso(p.Proceso)
	p.Solicitud = domain.CopiarSolicitud(p.Solicitud)
	salida := domain.ResultadoProceso{SchemaVersion: domain.VersionProceso, Alcance: "simulacion", Estado: "borrador", Proceso: p.Proceso, Solicitud: p.Solicitud, Valoraciones: []domain.ValoracionPuesto{}}
	entradas := map[string]domain.EntradaValoracionPuesto{}
	for _, v := range p.Solicitud.Valoraciones {
		entradas[v.PuestoRef] = v
	}
	for _, pref := range p.Solicitud.Preferencias {
		v := entradas[pref.PuestoRef]
		r, err := Simular(p.Proceso.Configuracion, v.Entrada)
		if err != nil {
			return domain.ResultadoProceso{}, err
		}
		salida.Valoraciones = append(salida.Valoraciones, domain.ValoracionPuesto{PuestoRef: pref.PuestoRef, Orden: pref.Orden, RequisitosEstado: domain.EstadoRequisitos(v.Requisitos), Requisitos: slices.Clone(v.Requisitos), Resultado: r})
	}
	salida.HuellaSimulacion = domain.HuellaSimulacionProceso(salida)
	return salida, nil
}
