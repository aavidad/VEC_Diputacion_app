package administracion

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/plannominal"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
)

// NuevaConfianzaGobiernoPlanFirmaV3 compone la cadena V3 con una sola
// capacidad: la del gobierno del plan nominal de firma (conjunto 2 de AD202).
func NuevaConfianzaGobiernoPlanFirmaV3(cfg ConfiguracionConfianzaPerfilesV3, deps DependenciasConfianzaPerfilesV3) (ConfianzaPerfilesV3, error) {
	return nuevaConfianzaConAudienciasV3(cfg, deps, []string{AudienciaGobiernoPlanFirmaV3})
}

// autoridadGobiernoPlanFirma es lo que el servicio necesita de la autoridad
// PostgreSQL de Contratación temporal.
type autoridadGobiernoPlanFirma interface {
	GobernarPlanFirma(context.Context, plannominal.SolicitudGobiernoPlanFirma) (plannominal.ReciboGobiernoPlanFirma, error)
}

// ServicioGobiernoPlanFirma traduce entre la ruta de vec-admin y la autoridad
// de Contratación temporal. No decide nada: tipos y errores.
type ServicioGobiernoPlanFirma struct{ autoridad autoridadGobiernoPlanFirma }

var _ api.ServicioGobiernoPlanFirmaADMIN = (*ServicioGobiernoPlanFirma)(nil)

func NuevoServicioGobiernoPlanFirma(a autoridadGobiernoPlanFirma) (*ServicioGobiernoPlanFirma, error) {
	if dependenciaComposicionNula(a) {
		return nil, ErrConfiguracion
	}
	return &ServicioGobiernoPlanFirma{autoridad: a}, nil
}

func (s *ServicioGobiernoPlanFirma) GobernarPlanFirma(ctx context.Context, x api.SolicitudGobiernoPlanFirmaADMIN) (api.ReciboGobiernoPlanFirmaADMIN, error) {
	if s == nil || dependenciaComposicionNula(s.autoridad) {
		return api.ReciboGobiernoPlanFirmaADMIN{}, plannominal.ErrGobiernoPlanFirmaNoDisponible
	}
	r, err := s.autoridad.GobernarPlanFirma(ctx, plannominal.SolicitudGobiernoPlanFirma{Actor: x.Actor, Evidencia: x.Evidencia,
		Instantanea: x.Instantanea, Material: x.Material, CorrelacionRef: x.CorrelacionRef})
	if err != nil {
		if errors.Is(err, plannominal.ErrGobiernoPlanFirmaConflicto) {
			return api.ReciboGobiernoPlanFirmaADMIN{}, errors.Join(api.ErrConflictoEstado, err)
		}
		return api.ReciboGobiernoPlanFirmaADMIN{}, err
	}
	publicacion := ""
	if r.PublicacionSHA256 != nil {
		publicacion = *r.PublicacionSHA256
	}
	return api.ReciboGobiernoPlanFirmaADMIN{Accion: r.Accion, CatalogoRef: r.CatalogoRef, ReciboRef: r.ReciboRef, Estado: r.Estado,
		Revision: r.Revision, PublicacionSHA256: publicacion, ConfirmadoEn: r.ConfirmadoEn, AuditoriaRef: r.AuditoriaRef,
		ConsumoAuditoriaRef: r.ConsumoAuditoriaRef}, nil
}
