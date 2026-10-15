package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La cobertura se agrega después de la consulta CT autorizada y auditada.
// Nunca se toma una unidad ni una versión del cuerpo HTTP adicional.
type consultorDetalleConCoberturaFirmaNominalRRHH struct {
	lector            httpinterno.ConsultorDetalleRRHH
	plan              ports.FuentePlanCoberturaFirmaNominalRRHH
	fijado            domain.VersionPlanFirmaV2
	documentalMontado bool
}

func (c consultorDetalleConCoberturaFirmaNominalRRHH) Consultar(ctx context.Context, solicitud ports.SolicitudDetalleRRHH) (ports.DetalleExpedienteRRHH, error) {
	if ctx == nil || ctx.Err() != nil || c.lector == nil {
		return ports.DetalleExpedienteRRHH{}, ports.ErrConsultaRRHHNoDisponible
	}
	detalle, err := c.lector.Consultar(ctx, solicitud)
	if err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.DetalleExpedienteRRHH{}, err
	}
	if detalle.EstadoBorradoresPublicados == "" {
		detalle.EstadoBorradoresPublicados = ports.BorradoresRRHHSinMontaje
		if c.documentalMontado {
			detalle.EstadoBorradoresPublicados = ports.BorradoresRRHHMontado
		}
	}
	cobertura := ports.CoberturaFirmaNominalRRHH{Estado: ports.CoberturaFirmaNominalNoDisponible, Documentos: []string{}}
	if detalle.Asignacion != nil {
		cobertura.UnidadRef = detalle.Asignacion.UnidadRef
	}
	if c.plan != nil && c.fijado.Valida() &&
		solicitud.ExpedienteRef() == detalle.Resumen.ExpedienteRef &&
		(solicitud.VersionObservada() == 0 || solicitud.VersionObservada() == detalle.Resumen.Version) &&
		detalle.Resumen.OrganizacionRef != "" && cobertura.UnidadRef != "" {
		publicado, err := c.plan.Plan(ctx)
		if err != nil {
			slog.WarnContext(ctx, "ct.cobertura_firma_no_disponible", "etapa", "lectura_plan", "error.type", fmt.Sprintf("%T", err))
		} else if err = publicado.Validar(); err != nil {
			slog.WarnContext(ctx, "ct.cobertura_firma_no_disponible", "etapa", "validacion_plan", "error.type", fmt.Sprintf("%T", err))
		} else if publicado.Version != c.fijado {
			slog.WarnContext(ctx, "ct.cobertura_firma_no_disponible", "etapa", "plan_cruzado")
		} else if ctx.Err() == nil {
			vistas := make(map[string]bool)
			for _, paso := range publicado.Pasos {
				if paso.OrganizacionRef == detalle.Resumen.OrganizacionRef && paso.UnidadRef == cobertura.UnidadRef && !vistas[paso.Documento] {
					vistas[paso.Documento] = true
					cobertura.Documentos = append(cobertura.Documentos, paso.Documento)
				}
			}
			if len(cobertura.Documentos) > 0 {
				sort.Strings(cobertura.Documentos)
				cobertura.Estado = ports.CoberturaFirmaNominalDisponible
				cobertura.Plan = publicado.Version
			}
		}
	}
	detalle.CoberturaFirmaNominal = cobertura
	return detalle, nil
}
