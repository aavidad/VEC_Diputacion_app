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

var ErrAutorizacionLectorRelacionRPTNoDisponible = errors.New("personal: autorización de la relación para RPT no disponible")

// IdentidadRegistradaLectorRelacionRPT procede exclusivamente de la frontera de
// sesión de la petición: ContextoActor registrado y el
// vínculo de autenticación revalidado.
type IdentidadRegistradaLectorRelacionRPT struct {
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

type ResolutorIdentidadLectorRelacionRPT interface {
	ResolverIdentidadLectorRelacionRPT(context.Context) (IdentidadRegistradaLectorRelacionRPT, error)
}

// EmisorMaterialLectorRelacionRPTV3 es la autoridad común V3 ya compuesta.
// Este contrato no admite permisos ni políticas enviados por el consumidor.
type EmisorMaterialLectorRelacionRPTV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

// ProveedorAutorizacionLectorRelacionRPT pide al emisor V3 común (PDP, firmante y
// verificador ya compuestos) la concesión de la relación para RPT.
type ProveedorAutorizacionLectorRelacionRPT struct {
	identidad ResolutorIdentidadLectorRelacionRPT
	emisor    EmisorMaterialLectorRelacionRPTV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionLectorRelacionRPT(identidad ResolutorIdentidadLectorRelacionRPT, emisor EmisorMaterialLectorRelacionRPTV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionLectorRelacionRPT, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || !vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, ErrAutorizacionLectorRelacionRPTNoDisponible
	}
	return &ProveedorAutorizacionLectorRelacionRPT{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionLectorRelacionRPT) AutorizarRelacionParaRPT(ctx context.Context, material personaldomain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	identidad, err := identidadOriginalLectorRelacionRPT(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	// El material de la lectura nominal pertenece a la persona, perfil y contexto de esta petición.
	canonActor, err := material.Actor().RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(canonActor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personalports.AccionRelacionParaRPTV1,
		Recurso:                   material.Recurso(),
		Finalidad:                 personaldomain.FinalidadLectorRelacionRPT,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil {
		return vacio, clasificarErrorAutorizacionLectorRelacionRPT(ctx, err)
	}
	if decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personalports.AudienciaRelacionParaRPTV1) {
		return vacio, personaldomain.ErrLectorRelacionRPTNoDisponible
	}
	return autorizacion, nil
}

// Solo una denegación explícita y registrada por el PDP es «acceso denegado»;
// cualquier otra causa, incluida una cancelación, es no disponible.
func clasificarErrorAutorizacionLectorRelacionRPT(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() == nil &&
		errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) &&
		!errors.Is(err, vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return personaldomain.ErrLectorRelacionRPTDenegado
	}
	return personaldomain.ErrLectorRelacionRPTNoDisponible
}

var _ personalports.ProveedorAutorizacionLectorRelacionRPT = (*ProveedorAutorizacionLectorRelacionRPT)(nil)
