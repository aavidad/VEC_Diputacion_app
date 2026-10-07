package composicion

import (
	"bytes"
	"context"
	"errors"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrAutorizacionHistoriaServiciosPropiaNoDisponible = errors.New("personal.autorizacion_historia_servicios_propia.no_disponible")

// ProveedorAutorizacionHistoriaServiciosPropia pide al emisor V3 común (PDP, firmante y
// verificador ya compuestos) la concesión de historia propia de servicios.
type ProveedorAutorizacionHistoriaServiciosPropia struct {
	identidad ResolutorIdentidadFichaPropia
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionHistoriaServiciosPropia(identidad ResolutorIdentidadFichaPropia, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionHistoriaServiciosPropia, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrAutorizacionHistoriaServiciosPropiaNoDisponible
	}
	return &ProveedorAutorizacionHistoriaServiciosPropia{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionHistoriaServiciosPropia) AutorizarHistoriaServiciosPropia(ctx context.Context, material personaldomain.MaterialHistoriaServiciosPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	identidad, err := identidadOriginalFichaPropia(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	// El material pertenece a la persona, perfil y contexto de esta petición.
	canonActor, err := material.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonActor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personaldomain.AccionHistoriaServiciosPropia,
		Recurso:                   material.Recurso(),
		Finalidad:                 personaldomain.FinalidadHistoriaServiciosPropia,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionHistoriaServiciosPropia(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personaldomain.AudienciaHistoriaServiciosPropia) {
		return vacio, personaldomain.ErrHistoriaServiciosPropiaNoDisponible
	}
	return autorizacion, nil
}

// Solo una denegación explícita y registrada por el PDP es «acceso denegado»;
// cualquier otra causa, incluida una cancelación, es no disponible.
func clasificarErrorAutorizacionHistoriaServiciosPropia(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return personaldomain.ErrHistoriaServiciosPropiaDenegada
	}
	return personaldomain.ErrHistoriaServiciosPropiaNoDisponible
}

var _ personalports.ProveedorAutorizacionHistoriaServiciosPropia = (*ProveedorAutorizacionHistoriaServiciosPropia)(nil)
