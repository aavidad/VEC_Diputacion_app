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

var ErrAutorizacionVinculoCRN11NoDisponible = errors.New("personal: autorización del vínculo propio CRN11 no disponible")

// IdentidadRegistradaVinculoCRN11 procede exclusivamente de la frontera de
// sesión de la petición: ContextoActor registrado con alcance {empleado} y el
// vínculo de autenticación revalidado.
type IdentidadRegistradaVinculoCRN11 struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type ResolutorIdentidadVinculoCRN11 interface {
	ResolverIdentidadVinculoCRN11(context.Context) (IdentidadRegistradaVinculoCRN11, error)
}

// EmisorMaterialVinculoCRN11V3 es la autoridad común V3 ya compuesta.
// Este contrato no admite permisos ni políticas enviados por el consumidor.
type EmisorMaterialVinculoCRN11V3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// ProveedorAutorizacionVinculoCRN11 pide al emisor V3 común (PDP, firmante y
// verificador ya compuestos) la concesión del vínculo propio CRN11.
type ProveedorAutorizacionVinculoCRN11 struct {
	identidad ResolutorIdentidadVinculoCRN11
	emisor    EmisorMaterialVinculoCRN11V3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionVinculoCRN11(identidad ResolutorIdentidadVinculoCRN11, emisor EmisorMaterialVinculoCRN11V3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionVinculoCRN11, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrAutorizacionVinculoCRN11NoDisponible
	}
	return &ProveedorAutorizacionVinculoCRN11{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionVinculoCRN11) AutorizarVinculoPropioCRN11(ctx context.Context, material personaldomain.MaterialVinculoPropioCRN11) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	identidad, err := p.identidad.ResolverIdentidadVinculoCRN11(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	// El material de la lectura nominal pertenece a la persona, perfil y contexto de esta petición.
	canonActor, err := material.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonActor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personaldomain.AccionVinculoPropioCRN11,
		Recurso:                   material.Recurso(),
		Finalidad:                 personaldomain.FinalidadVinculoPropioCRN11,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionVinculoCRN11(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personaldomain.AudienciaVinculoPropioCRN11) {
		return vacio, personaldomain.ErrVinculoCRN11NoDisponible
	}
	return autorizacion, nil
}

// Solo una denegación explícita y registrada por el PDP es «acceso denegado»;
// cualquier otra causa, incluida una cancelación, es no disponible.
func clasificarErrorAutorizacionVinculoCRN11(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return personaldomain.ErrVinculoCRN11Denegado
	}
	return personaldomain.ErrVinculoCRN11NoDisponible
}

var _ personalports.ProveedorAutorizacionVinculoCRN11 = (*ProveedorAutorizacionVinculoCRN11)(nil)
