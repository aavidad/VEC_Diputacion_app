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

var ErrAutorizacionAsignacionDietasNoDisponible = errors.New("personal: autorización de asignación de dietas no disponible")

// ProveedorAutorizacionAsignacionDietas usa el contexto registrado de la sesión
// interna. El efecto y el actor se cotejan antes de pedir una concesión V3.
type ProveedorAutorizacionAsignacionDietas struct {
	identidad ResolutorIdentidadRelacionDietas
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionAsignacionDietas(identidad ResolutorIdentidadRelacionDietas, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionAsignacionDietas, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || motivo.Validar() != nil {
		return nil, ErrAutorizacionAsignacionDietasNoDisponible
	}
	return &ProveedorAutorizacionAsignacionDietas{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionAsignacionDietas) AutorizarAsignacionDietas(ctx context.Context, material personaldomain.MaterialAsignacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	identidad, err := p.identidad.ResolverIdentidadRelacionDietas(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	actor, err := material.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	accion, finalidad, audiencia := contratoAutorizacionAsignacionDietas(material.Solicitud().Operacion)
	if accion == "" {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personalports.ErrAsignacionDietasDenegada
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
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil || decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, audiencia) {
		return vacio, personalports.ErrAsignacionDietasDenegada
	}
	return autorizacion, nil
}

func contratoAutorizacionAsignacionDietas(operacion personaldomain.OperacionAsignacionDietas) (accion, finalidad, audiencia string) {
	switch operacion {
	case personaldomain.ConsultarAsignacionDietas:
		return personalports.AccionConsultarAsignacionDietas, "preparar_borrador_dietas", personalports.AudienciaConsultarAsignacionDietas
	case personaldomain.RegistrarInicialAsignacionDietas:
		return personalports.AccionRegistrarInicialAsignacionDietas, "registrar_asignacion_dietas_inicial", personalports.AudienciaRegistrarInicialAsignacionDietas
	case personaldomain.CorregirAsignacionDietas:
		return personalports.AccionCorregirAsignacionDietas, "corregir_asignacion_dietas", personalports.AudienciaCorregirAsignacionDietas
	case personaldomain.CorregirGrupoAsignacionDietas:
		return personalports.AccionCorregirGrupoDietas, "corregir_grupo_dieta", personalports.AudienciaCorregirGrupoDietas
	default:
		return "", "", ""
	}
}

var _ personalports.ProveedorAutorizacionAsignacionDietas = (*ProveedorAutorizacionAsignacionDietas)(nil)

// ProveedorAutorizacionCompetenciasAsignacionDietas acredita la consulta
// interna actor→asignaciones sin aceptar unidad ni persona del HTTP.
type ProveedorAutorizacionCompetenciasAsignacionDietas struct {
	identidad ResolutorIdentidadRelacionDietas
	emisor    EmisorMaterialRelacionDietasV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoProveedorAutorizacionCompetenciasAsignacionDietas(identidad ResolutorIdentidadRelacionDietas, emisor EmisorMaterialRelacionDietasV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ProveedorAutorizacionCompetenciasAsignacionDietas, error) {
	if dependenciaNula(identidad) || dependenciaNula(emisor) || motivo.Validar() != nil {
		return nil, ErrAutorizacionAsignacionDietasNoDisponible
	}
	return &ProveedorAutorizacionCompetenciasAsignacionDietas{identidad: identidad, emisor: emisor, motivo: motivo}, nil
}

func (p *ProveedorAutorizacionCompetenciasAsignacionDietas) AutorizarCompetenciasAsignacionDietas(ctx context.Context, material personaldomain.MaterialCompetenciasAsignacionDietas) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || dependenciaNula(p.identidad) || dependenciaNula(p.emisor) || ctx == nil || ctx.Err() != nil || len(material.Canonico()) == 0 {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	identidad, err := p.identidad.ResolverIdentidadRelacionDietas(ctx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	actor, err := material.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(actor, identidad.Resultado.RepresentacionCanonica) {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: identidad.Vinculo,
		ReferenciaMotivo:          p.motivo,
		Accion:                    personalports.AccionConsultarCompetenciasAsignacionDietas,
		Recurso:                   material.Recurso(),
		Finalidad:                 personalports.FinalidadConsultarCompetenciasAsignacionDietas,
		Correlacion:               correlacion,
	})
	if err != nil {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	decision, confirmacion, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, identidad.Resultado)
	if err != nil || decision.ValidarPara(solicitud) != nil || dependenciaNula(exportador) {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	autorizacion, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, identidad.Resultado, p.motivo, autorizacion, personalports.AudienciaConsultarCompetenciasAsignacionDietas) {
		return vacio, personalports.ErrCompetenciasAsignacionDietasDenegadas
	}
	return autorizacion, nil
}

var _ personalports.ProveedorAutorizacionCompetenciasAsignacionDietas = (*ProveedorAutorizacionCompetenciasAsignacionDietas)(nil)
