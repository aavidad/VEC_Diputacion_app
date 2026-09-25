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

var ErrAutorizacionRectificacionDietasNoDisponible = errors.New("personal: autorización de rectificación de dietas no disponible")

// ProveedorAutorizacionRectificacionDietas liga cada intención al contexto V2
// registrado y a una audiencia V3 nominal. El sujeto indicado por el gestor
// permanece en el recurso del material; nunca sustituye al actor autenticado.
type ProveedorAutorizacionRectificacionDietas struct {
	identidad ResolutorIdentidadRelacionDietas
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionRectificacionDietas(identidad ResolutorIdentidadRelacionDietas, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionRectificacionDietas, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || motivo.Validar() != nil {
		return nil, ErrAutorizacionRectificacionDietasNoDisponible
	}
	return &ProveedorAutorizacionRectificacionDietas{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionRectificacionDietas) AutorizarRectificacionDietas(ctx context.Context, material personaldomain.MaterialRectificacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	identidad, err := p.identidad.ResolverIdentidadRelacionDietas(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	actor, err := material.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	accion, finalidad, audiencia := contratoAutorizacionRectificacionDietas(material.Solicitud().Operacion)
	if accion == "" {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    accion,
		Recurso:                   material.Recurso(),
		Finalidad:                 finalidad,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil || decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, audiencia) {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	return autorizacion, nil
}

func contratoAutorizacionRectificacionDietas(operacion personaldomain.OperacionRectificacionDietas) (accion, finalidad, audiencia string) {
	switch operacion {
	case personaldomain.SolicitarRectificacionDietas:
		return personalports.AccionSolicitarRectificacionDietas, "solicitar_rectificacion_dietas", personalports.AudienciaSolicitarRectificacionDietas
	case personaldomain.ConsultarRectificacionDietas:
		return personalports.AccionConsultarRectificacionDietas, "consultar_rectificacion_dietas_propia", personalports.AudienciaConsultarRectificacionDietas
	case personaldomain.ConfirmarRectificacionDietas, personaldomain.RechazarRectificacionDietas:
		return personalports.AccionResolverRectificacionDietas, "resolver_rectificacion_dietas", personalports.AudienciaResolverRectificacionDietas
	default:
		return "", "", ""
	}
}

var _ personalports.ProveedorAutorizacionRectificacionDietas = (*ProveedorAutorizacionRectificacionDietas)(nil)

// ProveedorAutorizacionRectificacionesCompetentesDietas autoriza la bandeja
// interna del gestor a partir del actor V2; el cliente no aporta sujetos.
type ProveedorAutorizacionRectificacionesCompetentesDietas struct {
	identidad ResolutorIdentidadRelacionDietas
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionRectificacionesCompetentesDietas(identidad ResolutorIdentidadRelacionDietas, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionRectificacionesCompetentesDietas, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || motivo.Validar() != nil {
		return nil, ErrAutorizacionRectificacionDietasNoDisponible
	}
	return &ProveedorAutorizacionRectificacionesCompetentesDietas{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionRectificacionesCompetentesDietas) AutorizarRectificacionesCompetentesDietas(ctx context.Context, material personaldomain.MaterialRectificacionesCompetentesDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	identidad, err := p.identidad.ResolverIdentidadRelacionDietas(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	actor, err := material.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personalports.AccionConsultarRectificacionesCompetentesDietas,
		Recurso:                   material.Recurso(),
		Finalidad:                 personalports.FinalidadConsultarRectificacionesCompetentesDietas,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil || decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personalports.AudienciaConsultarRectificacionesCompetentesDietas) {
		return vacio, personalports.ErrRectificacionDietasDenegada
	}
	return autorizacion, nil
}

var _ personalports.ProveedorAutorizacionRectificacionesCompetentesDietas = (*ProveedorAutorizacionRectificacionesCompetentesDietas)(nil)
