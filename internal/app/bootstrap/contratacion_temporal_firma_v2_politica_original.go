package bootstrap

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La frontera del original existe desde CT133 y conserva la política legacy.
// Este selector usa la misma instancia del PDP común: una sesión nominal de
// firma elige fuentes PostgreSQL de su perfil, y toda otra petición conserva
// la política original. Una sesión nominal fallida nunca cae al perfil legacy.
type politicaOriginalFirmaR5CT struct {
	legacy politicaAutorizacionSolicitudLigadaV3Desarrollo
	m      *montajeFirmaVecR5CT
}

type claveMontajeFirmaVecR5CT struct{}

func contextoMontajeFirmaVecR5CT(ctx context.Context, m *montajeFirmaVecR5CT) context.Context {
	if m == nil {
		return ctx
	}
	return context.WithValue(ctx, claveMontajeFirmaVecR5CT{}, m)
}

func descriptoresAutorizacionContratacionTemporalConFirmaR5Desarrollo(ctx context.Context,
	politica politicaAutorizacionSolicitudLigadaV3Desarrollo, reincorporacion ...bool) []descriptorAutorizacionComunDesarrollo {
	descriptores := descriptoresAutorizacionContratacionTemporalDesarrollo(politica, reincorporacion...)
	m, ok := ctx.Value(claveMontajeFirmaVecR5CT{}).(*montajeFirmaVecR5CT)
	if !ok || m == nil {
		return descriptores
	}
	contextual := m.politicaOriginalContextual(politica)
	for i := range descriptores {
		for _, frontera := range descriptores[i].Fronteras {
			if frontera == "ct-original-firmable-preparar" || frontera == "ct-cuadro-consultar" ||
				frontera == "ct-expediente-consultar" || frontera == "ct-circuito-rrhh-consultar" {
				descriptores[i].Politica = contextual
			}
		}
	}
	return descriptores
}

func (m *montajeFirmaVecR5CT) politicaOriginalContextual(legacy politicaAutorizacionSolicitudLigadaV3Desarrollo) politicaAutorizacionSolicitudLigadaV3Desarrollo {
	p := &politicaOriginalFirmaR5CT{legacy: legacy, m: m}
	return politicaAutorizacionSolicitudLigadaV3Desarrollo{fuente: p, registroConcesiones: p,
		registroDenegaciones: p, validadorMotivos: p}
}

func (p *politicaOriginalFirmaR5CT) nominal(ctx context.Context) (bool, bool) {
	if p == nil || p.m == nil || p.m.nominal == nil || ctx == nil ||
		!(rutaNominalFirmaCT(ctx) == httpinterno.RutaOriginalFirmableCT ||
			rutaFirmaVecR5CT(rutaNominalFirmaCT(ctx)) || rutaLecturaCTFirmaV2(rutaNominalFirmaCT(ctx))) {
		return false, false
	}
	if _, ok := ctx.Value(clavePeticionNominalFirmaCT{}).(*peticionNominalFirmaCT); !ok {
		return false, false
	}
	_, err := p.m.nominal.contexto(ctx)
	return true, err == nil
}

func (p *politicaOriginalFirmaR5CT) ObtenerInstantaneaAutorizacion(ctx context.Context, persona, perfil string) (core.InstantaneaAutorizacion, error) {
	nominal, valida := p.nominal(ctx)
	if nominal {
		if !valida {
			return core.InstantaneaAutorizacion{}, vp.ErrFuenteAutorizacionNoDisponible
		}
		return p.m.fuenteAutorizacion.ObtenerInstantaneaAutorizacion(ctx, persona, perfil)
	}
	return p.legacy.fuente.ObtenerInstantaneaAutorizacion(ctx, persona, perfil)
}

func (p *politicaOriginalFirmaR5CT) RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx context.Context,
	o vp.OrdenRegistroConcesionCandidataAutorizacionLigadaV3) (time.Time, error) {
	nominal, valida := p.nominal(ctx)
	if nominal {
		if !valida {
			return time.Time{}, vp.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible
		}
		return p.m.registro.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, o)
	}
	return p.legacy.registroConcesiones.RegistrarConcesionCandidataAutorizacionLigadaV3SiInstantaneaVigente(ctx, o)
}

func (p *politicaOriginalFirmaR5CT) RegistrarDenegacionAutorizacionLigadaV3(ctx context.Context,
	o vp.OrdenRegistroDenegacionAutorizacionLigadaV3) error {
	nominal, valida := p.nominal(ctx)
	if nominal {
		if !valida {
			return vp.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible
		}
		return p.m.registro.RegistrarDenegacionAutorizacionLigadaV3(ctx, o)
	}
	return p.legacy.registroDenegaciones.RegistrarDenegacionAutorizacionLigadaV3(ctx, o)
}

func (p *politicaOriginalFirmaR5CT) ValidarReferenciaMotivoAutorizacionV2(ctx context.Context,
	r core.ReferenciaEntradaCatalogo, instante time.Time) error {
	nominal, valida := p.nominal(ctx)
	if nominal {
		if !valida {
			return core.ErrSolicitudAutorizacionInvalida
		}
		switch r.CatalogoID {
		case motivoConsultaCircuitoRRHHDesarrollo().CatalogoID:
			if p.m.nominal.lecturas == nil || p.m.nominal.lecturas.motivoCircuito == nil {
				return core.ErrSolicitudAutorizacionInvalida
			}
			return p.m.nominal.lecturas.motivoCircuito.ValidarReferenciaMotivoAutorizacionV2(ctx, r, instante)
		case p.m.alta.soporte.motivoCuadroRRHH.CatalogoID:
			vigente, err := p.m.consultas.motivos.ResolverMotivoCuadroRRHH(ctx, instante)
			if err == nil && vigente == r {
				return nil
			}
			return core.ErrSolicitudAutorizacionInvalida
		case motivoOriginalFirmableCTDesarrollo().CatalogoID:
			return p.m.motivosOriginal.ValidarReferenciaMotivoAutorizacionV2(ctx, r, instante)
		case p.m.motivosDetalleCatalogoID():
			return p.m.motivosDetalle.ValidarReferenciaMotivoAutorizacionV2(ctx, r, instante)
		default:
			return core.ErrSolicitudAutorizacionInvalida
		}
	}
	return p.legacy.validadorMotivos.ValidarReferenciaMotivoAutorizacionV2(ctx, r, instante)
}

func (m *montajeFirmaVecR5CT) motivosDetalleCatalogoID() string {
	if m == nil || m.alta == nil || m.alta.soporte == nil {
		return ""
	}
	return m.alta.soporte.motivoDetalleRRHH.CatalogoID
}
