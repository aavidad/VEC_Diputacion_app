package composicion

import (
	"context"
	"errors"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// FuenteCompetenciaPerfilFijo consulta la misma asignación central que el PDP.
// No publica perfiles ni concede acciones. V3 vuelve a evaluar la acción exacta
// y el consumidor revalida su autorización antes de leer o cambiar la comisión.
type FuenteCompetenciaPerfilFijo struct {
	fuente vecports.FuenteAutorizacion
	reloj  vecports.Reloj
}

func NuevaFuenteCompetenciaPerfilFijo(f vecports.FuenteAutorizacion, r vecports.Reloj) (*FuenteCompetenciaPerfilFijo, error) {
	if nulo(f) || nulo(r) {
		return nil, ErrIdentidadCircuitoNoDisponible
	}
	return &FuenteCompetenciaPerfilFijo{fuente: f, reloj: r}, nil
}

func (f *FuenteCompetenciaPerfilFijo) EstadoCompetencias(ctx context.Context, actor vecdomain.ResultadoContextoActorRegistradoV2) (dietasports.EstadoCompetenciasCircuito, error) {
	etapa, _, err := f.competencia(ctx, actor)
	if err != nil {
		return dietasports.EstadoCompetenciasCircuito{}, err
	}
	etapas := []domain.EtapaCircuito{}
	if etapa != "" {
		etapas = append(etapas, etapa)
	}
	return dietasports.EstadoCompetenciasCircuito{Fuente: dietasports.FuenteCompetenciaAcreditada, Etapas: etapas}, nil
}

func (f *FuenteCompetenciaPerfilFijo) UnidadCompetente(ctx context.Context, actor vecdomain.ResultadoContextoActorRegistradoV2, solicitada domain.EtapaCircuito) (string, error) {
	etapa, unidad, err := f.competencia(ctx, actor)
	if err != nil {
		return "", err
	}
	if etapa == "" || solicitada != etapa {
		return "", dietasports.ErrAccesoCircuitoDenegado
	}
	return unidad, nil
}

func (f *FuenteCompetenciaPerfilFijo) competencia(ctx context.Context, actor vecdomain.ResultadoContextoActorRegistradoV2) (domain.EtapaCircuito, string, error) {
	denegado := dietasports.ErrAccesoCircuitoDenegado
	if f == nil || ctx == nil || nulo(f.fuente) || nulo(f.reloj) || actor.Validar() != nil {
		return "", "", denegado
	}
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	a := actor.Contexto
	i, err := f.fuente.ObtenerInstantaneaAutorizacion(ctx, a.PersonaRef, a.PerfilActivoRef)
	if ctx.Err() != nil {
		return "", "", ctx.Err()
	}
	if err != nil {
		return "", "", errors.Join(denegado, err)
	}
	ahora := f.reloj.Ahora()
	p := i.AsignacionPerfil
	if i.Validar() != nil || ahora.IsZero() || !a.Instantanea.VigenteEn(ahora) ||
		p.PrincipalID != a.PersonaRef || p.PerfilActivoRef != a.PerfilActivoRef ||
		!p.VigenteEn(ahora) || p.EmitidaEn.After(ahora) || i.VersionRol.PublicadaEn.After(ahora) ||
		i.VersionRol.Estado != vecdomain.EstadoVersionRolPublicada ||
		i.ControlVigenciaVersionRol.Estado != vecdomain.EstadoControlVigenciaVersionRolHabilitada || i.ControlVigenciaVersionRol.ActualizadoEn.After(ahora) {
		return "", "", denegado
	}
	var etapa domain.EtapaCircuito
	switch i.VersionRol.RolID {
	case "dietas_revision_administrativa":
		etapa = domain.EtapaRevision
	case "dietas_autorizacion":
		etapa = domain.EtapaAutorizacion
	case "dietas_liquidacion_rrhh":
		etapa = domain.EtapaLiquidacion
	case "dietas_fiscalizacion":
		etapa = domain.EtapaFiscalizacion
	default:
		// Un perfil de solicitante no recibe etapas de otros perfiles.
		return "", "", nil
	}
	if len(p.Ambitos) != 2 {
		return "", "", denegado
	}
	var persona, unidad string
	for _, ambito := range p.Ambitos {
		if len(ambito.Valores) != 1 {
			return "", "", denegado
		}
		switch ambito.Clave {
		case "persona_ref":
			persona = ambito.Valores[0]
		case "unidad_ref":
			unidad = ambito.Valores[0]
		default:
			return "", "", denegado
		}
	}
	if persona != a.PersonaRef || dietasapp.ValidarConsultaBandejaCircuito(dietasports.ConsultaBandejaCircuito{Etapa: etapa, UnidadRef: unidad, Limite: 1}) != nil {
		return "", "", denegado
	}
	return etapa, unidad, nil
}

var _ dietasports.FuenteCompetenciaCircuito = (*FuenteCompetenciaPerfilFijo)(nil)
