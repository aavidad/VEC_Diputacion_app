// Package autorizacion adapta la identidad registrada y el emisor central V3.
package autorizacion

import (
	"bytes"
	"context"
	"errors"
	"reflect"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type IdentidadRegistrada struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type ResolutorIdentidad interface {
	ResolverIdentidadConvocatoria(context.Context) (IdentidadRegistrada, error)
}
type EmisorV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type Proveedor struct {
	identidad ResolutorIdentidad
	emisor    EmisorV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedor(i ResolutorIdentidad, e EmisorV3, m vecdomain.ReferenciaEntradaCatalogo) (*Proveedor, error) {
	if nulo(i) || nulo(e) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(m) {
		return nil, ports.ErrConvocatoriaNoDisponible
	}
	return &Proveedor{i, e, m}, nil
}

func (p *Proveedor) AutorizarConsultaConvocatoria(ctx context.Context, s ports.SolicitudConsultaConvocatoria) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || p == nil || nulo(p.identidad) || nulo(p.emisor) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	preparacion, err := application.PrepararConsultaConvocatoria(s)
	if err != nil {
		return cero, err
	}
	i, err := p.identidad.ResolverIdentidadConvocatoria(ctx)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil || i.Resultado.Validar() != nil || i.Vinculo.ValidarPara(i.Resultado) != nil {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	actor, err := s.Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actor, i.Resultado.RepresentacionCanonica) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: i.Vinculo, ReferenciaMotivo: p.motivo, Accion: bolsaports.AccionConsultarVersionConvocatoria,
		Recurso: preparacion.Recurso, Finalidad: bolsaports.FinalidadConsultaInternaConvocatorias, Correlacion: s.Correlacion})
	if err != nil {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	d, c, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, i.Resultado)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return cero, cancelacion
	}
	if err != nil {
		return cero, clasificarError(ctx, err)
	}
	if d.ValidarPara(solicitud) != nil || nulo(exportador) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	a, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, d, c, i.Resultado, p.motivo, a, application.AudienciaConsultaConvocatoriaV3) {
		return cero, ports.ErrConvocatoriaNoDisponible
	}
	return a, nil
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

var _ ports.ProveedorAutorizacionConsultaConvocatoria = (*Proveedor)(nil)

func clasificarError(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	for _, tecnica := range []error{vecports.ErrFuenteAutorizacionNoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, tecnica) {
			return ports.ErrConvocatoriaNoDisponible
		}
	}
	if errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return ports.ErrConsultaConvocatoriaDenegada
	}
	return ports.ErrConvocatoriaNoDisponible
}
