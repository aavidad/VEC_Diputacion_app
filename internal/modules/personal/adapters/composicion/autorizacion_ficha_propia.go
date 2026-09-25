package composicion

import (
	"bytes"
	"context"
	"errors"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrAutorizacionFichaPropiaNoDisponible = errors.New("personal: autorización de la ficha propia no disponible")

// IdentidadRegistradaFichaPropia procede exclusivamente de la frontera de
// sesión de la petición: ContextoActor registrado con alcance {empleado} y el
// vínculo de autenticación revalidado.
type IdentidadRegistradaFichaPropia struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type ResolutorIdentidadFichaPropia interface {
	ResolverIdentidadFichaPropia(context.Context) (IdentidadRegistradaFichaPropia, error)
}

// ProveedorAutorizacionFichaPropia pide al emisor V3 común (PDP, firmante y
// verificador ya compuestos) la concesión de la ficha propia.
type ProveedorAutorizacionFichaPropia struct {
	identidad ResolutorIdentidadFichaPropia
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionFichaPropia(identidad ResolutorIdentidadFichaPropia, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionFichaPropia, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrAutorizacionFichaPropiaNoDisponible
	}
	return &ProveedorAutorizacionFichaPropia{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionFichaPropia) AutorizarFichaPropia(ctx context.Context, material personaldomain.MaterialFichaPropia) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	identidad, err := p.identidad.ResolverIdentidadFichaPropia(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	// El material pertenece a la persona, perfil y contexto de esta petición.
	canonActor, err := material.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonActor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personaldomain.AccionFichaPropia,
		Recurso:                   material.Recurso(),
		Finalidad:                 personaldomain.FinalidadFichaPropia,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionFichaPropia(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personaldomain.AudienciaFichaPropia) {
		return vacio, personaldomain.ErrFichaPropiaNoDisponible
	}
	return autorizacion, nil
}

// Solo una denegación explícita y registrada por el PDP es «acceso denegado»;
// cualquier otra causa, incluida una cancelación, es no disponible.
func clasificarErrorAutorizacionFichaPropia(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return personaldomain.ErrFichaPropiaDenegada
	}
	return personaldomain.ErrFichaPropiaNoDisponible
}

var _ personalports.ProveedorAutorizacionFichaPropia = (*ProveedorAutorizacionFichaPropia)(nil)
