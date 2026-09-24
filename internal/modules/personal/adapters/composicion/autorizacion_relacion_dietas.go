// Package composicion une la consulta de relaciones de Personal con la
// identidad registrada y la autoridad común V3, sin prestar tablas a Dietas.
package composicion

import (
	"bytes"
	"context"
	"errors"
	"reflect"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrAutorizacionRelacionDietasNoDisponible = errors.New("personal: autorización de relación para dietas no disponible")

type IdentidadRegistradaRelacionDietas struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type ResolutorIdentidadRelacionDietas interface {
	ResolverIdentidadRelacionDietas(context.Context) (IdentidadRegistradaRelacionDietas, error)
}

type EmisorMaterialRelacionDietasV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type ProveedorAutorizacionRelacionDietas struct {
	identidad ResolutorIdentidadRelacionDietas
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionRelacionDietas(identidad ResolutorIdentidadRelacionDietas, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionRelacionDietas, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || motivo.Validar() != nil {
		return nil, ErrAutorizacionRelacionDietasNoDisponible
	}
	return &ProveedorAutorizacionRelacionDietas{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionRelacionDietas) AutorizarConsultaRelacionPropia(ctx context.Context, material personaldomain.MaterialConsultaRelacionPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	identidad, err := p.identidad.ResolverIdentidadRelacionDietas(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	canonActor, err := material.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonActor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    "personal.relacion.propia.consultar_dietas",
		Recurso:                   material.Recurso(),
		Finalidad:                 "preparar_borrador_dietas",
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionRelacionDietas(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, "vec_personal.relacion_propia.consultar_dietas.v1") {
		return vacio, personalports.ErrRelacionEmpleadoNoDisponible
	}
	return autorizacion, nil
}

func clasificarErrorAutorizacionRelacionDietas(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) {
		return personalports.ErrRelacionEmpleadoDenegada
	}
	return personalports.ErrRelacionEmpleadoNoDisponible
}

func dependenciaNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	return (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface || v.Kind() == reflect.Func || v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && v.IsNil()
}

var _ personalports.ProveedorAutorizacionConsultaRelacionPropia = (*ProveedorAutorizacionRelacionDietas)(nil)
