package autorizacion

import (
	"bytes"
	"context"
	"errors"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

// ProveedorPreparacionBases reutiliza la identidad registrada y el emisor
// central de S1. No concede permisos ni adapta capacidades funcionales V1.
type ProveedorPreparacionBases struct {
	identidad ResolutorIdentidad
	emisor    EmisorV3
	motivo    vec.ReferenciaEntradaCatalogo
}

func NuevoProveedorPreparacionBases(i ResolutorIdentidad, e EmisorV3, m vec.ReferenciaEntradaCatalogo) (*ProveedorPreparacionBases, error) {
	if nulo(i) || nulo(e) || !vec.ReferenciaMotivoAutorizacionV2Valida(m) {
		return nil, ports.ErrPreparacionBasesNoDisponible
	}
	return &ProveedorPreparacionBases{i, e, m}, nil
}

func (p *ProveedorPreparacionBases) AutorizarGuardadoPreparacionBases(ctx context.Context, s ports.SolicitudGuardarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	preparacion, err := bolsa.PrepararGuardadoPreparacionBasesV3(s)
	if err != nil {
		return core.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return p.emitirPreparacion(ctx, s.Actor, s.Correlacion, preparacion, ports.AccionGuardarPreparacionBases, bolsa.AudienciaGuardarPreparacionBasesV3)
}

func (p *ProveedorPreparacionBases) AutorizarConsultaPreparacionBases(ctx context.Context, s ports.SolicitudConsultarPreparacionBasesV3) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	preparacion, err := bolsa.PrepararConsultaPreparacionBasesV3(s)
	if err != nil {
		return core.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	return p.emitirPreparacion(ctx, s.Actor, s.Correlacion, preparacion, ports.AccionConsultarPreparacionBases, bolsa.AudienciaConsultarPreparacionBasesV3)
}

func (p *ProveedorPreparacionBases) emitirPreparacion(ctx context.Context, actor vec.ContextoActor, c vec.ReferenciaCorrelacionAutorizacionV2, preparacion bolsa.PreparacionOperacionBasesV3, accion, audiencia string) (core.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero core.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p == nil || nulo(p.identidad) || nulo(p.emisor) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	i, err := p.identidad.ResolverIdentidadConvocatoria(ctx)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, clasificarErrorPreparacion(ctx, err)
	}
	if i.Resultado.Validar() != nil || i.Vinculo.ValidarPara(i.Resultado) != nil {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	b, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(b, i.Resultado.RepresentacionCanonica) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	solicitud, err := vec.NuevaSolicitudAutorizacionLigadaV3(vec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: i.Vinculo, ReferenciaMotivo: p.motivo, Accion: accion, Recurso: preparacion.Recurso, Finalidad: ports.FinalidadPreparacionBases, Correlacion: c})
	if err != nil {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	d, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, i.Resultado)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, clasificarErrorPreparacion(ctx, err)
	}
	if d.ValidarPara(solicitud) != nil || nulo(exportador) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	a, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !core.MaterialAtestadoLigadoV3(solicitud, d, confirmacion, i.Resultado, p.motivo, a, audiencia) {
		return cero, ports.ErrPreparacionBasesNoDisponible
	}
	if err := bolsa.ValidarExportacionPreparacionBasesV3(a, preparacion, actor, accion, audiencia); err != nil {
		return cero, err
	}
	return a, nil
}

func clasificarErrorPreparacion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, causa := range []error{ports.ErrPreparacionBasesNoDisponible, core.ErrFuenteAutorizacionNoDisponible, core.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, core.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, causa) {
			return ports.ErrPreparacionBasesNoDisponible
		}
	}
	if errors.Is(err, ports.ErrPreparacionBasesDenegada) || errors.Is(err, core.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return ports.ErrPreparacionBasesDenegada
	}
	return ports.ErrPreparacionBasesNoDisponible
}

var _ ports.ProveedorAutorizacionPreparacionBasesV3 = (*ProveedorPreparacionBases)(nil)
