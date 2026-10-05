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

var ErrAutorizacionHistoriaRelacionesPropiaNoDisponible = errors.New("personal.autorizacion_historia_relaciones_propia.no_disponible")

// ProveedorAutorizacionHistoriaRelacionesPropia pide al emisor V3 común (PDP, firmante y
// verificador ya compuestos) la concesión de historia propia de relaciones.
type ProveedorAutorizacionHistoriaRelacionesPropia struct {
	identidad ResolutorIdentidadFichaPropia
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionHistoriaRelacionesPropia(identidad ResolutorIdentidadFichaPropia, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionHistoriaRelacionesPropia, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrAutorizacionHistoriaRelacionesPropiaNoDisponible
	}
	return &ProveedorAutorizacionHistoriaRelacionesPropia{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionHistoriaRelacionesPropia) AutorizarHistoriaRelacionesPropia(ctx context.Context, material personaldomain.MaterialHistoriaRelacionesPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	identidad, err := identidadOriginalFichaPropia(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	// El material pertenece a la persona, perfil y contexto de esta petición.
	canonActor, err := material.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonActor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personaldomain.AccionHistoriaRelacionesPropia,
		Recurso:                   material.Recurso(),
		Finalidad:                 personaldomain.FinalidadHistoriaRelacionesPropia,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionHistoriaRelacionesPropia(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personaldomain.AudienciaHistoriaRelacionesPropia) {
		return vacio, personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
	}
	return autorizacion, nil
}

// Solo una denegación explícita y registrada por el PDP es «acceso denegado»;
// cualquier otra causa, incluida una cancelación, es no disponible.
func clasificarErrorAutorizacionHistoriaRelacionesPropia(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return personaldomain.ErrHistoriaRelacionesPropiaDenegada
	}
	return personaldomain.ErrHistoriaRelacionesPropiaNoDisponible
}

var _ personalports.ProveedorAutorizacionHistoriaRelacionesPropia = (*ProveedorAutorizacionHistoriaRelacionesPropia)(nil)
