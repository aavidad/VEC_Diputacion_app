package bootstrap

import (
	"context"
	"errors"
	"strconv"

	plantillaspg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres/plantillascatalogo"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	finalidadCatalogoPlantillasCT   = "gestionar_catalogo_plantillas_contratacion_temporal"
	finalidadDocumentalPlantillasCT = "consultar_borradores_expediente"
	audienciaCatalogoPlantillasCT   = "vec_contratacion_temporal.catalogo_plantillas.v1"
	audienciaDocumentalPlantillasCT = "vec_contratacion_temporal.catalogo_plantillas_documental.v1"
	tipoCatalogoPlantillasCT        = "catalogo_plantillas_contratacion_temporal"
	tipoDocumentalPlantillasCT      = "catalogo_plantillas_documental_ct"
)

// proveedorCatalogoPlantillasCT no tiene autoridad propia. La sesión de CT,
// el PDP común y el material V3 ya configurado llegan de composición.
type proveedorCatalogoPlantillasCT struct {
	soporte            *soporteAltaContratacionTemporalDesarrollo
	pdp                *autorizadorComunDesarrollo
	materialCatalogo   *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialDocumental *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivoCatalogo     vecdomain.ReferenciaEntradaCatalogo
	motivoDocumental   vecdomain.ReferenciaEntradaCatalogo
	reloj              relojContratacionTemporalDesarrollo
}

func nuevoProveedorCatalogoPlantillasCT(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	pdp *autorizadorComunDesarrollo,
	materialCatalogo, materialDocumental *proveedorMaterialAltaContratacionTemporalDesarrollo,
	motivoCatalogo, motivoDocumental vecdomain.ReferenciaEntradaCatalogo,
	reloj relojContratacionTemporalDesarrollo,
) (*proveedorCatalogoPlantillasCT, error) {
	if soporte == nil || pdp == nil || pdp.servicio == nil ||
		materialCatalogo == nil || materialDocumental == nil ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivoCatalogo) ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivoDocumental) {
		return nil, plantillasapp.ErrNoDisponible
	}
	return &proveedorCatalogoPlantillasCT{soporte: soporte, pdp: pdp,
		materialCatalogo: materialCatalogo, materialDocumental: materialDocumental,
		motivoCatalogo: motivoCatalogo, motivoDocumental: motivoDocumental, reloj: reloj}, nil
}

func (p *proveedorCatalogoPlantillasCT) ResolverContextoActor(ctx context.Context) (vecdomain.ContextoActor, error) {
	if p == nil || p.soporte == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return vecdomain.ContextoActor{}, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || operativo.Resultado.Validar() != nil ||
		operativo.Vinculo.ValidarPara(operativo.Resultado) != nil ||
		!operativo.Vinculo.VigenteEn(p.reloj.Ahora(), operativo.Resultado) {
		return vecdomain.ContextoActor{}, vecdomain.ErrAutorizacionDenegada
	}
	return operativo.Resultado.Contexto.Clonar()
}

func (p *proveedorCatalogoPlantillasCT) AutorizarCatalogoPlantillas(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p == nil || !recursoCatalogoPlantillasCTValido(accion, recurso, false) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	return p.autorizar(ctx, actor, accion, recurso, finalidadCatalogoPlantillasCT,
		p.motivoCatalogo, p.materialCatalogo, audienciaCatalogoPlantillasCT)
}

func (p *proveedorCatalogoPlantillasCT) AutorizarCatalogoDocumental(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if p == nil || !recursoDocumentalPlantillasCTValido(accion, recurso) {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, vecdomain.ErrAutorizacionDenegada
	}
	return p.autorizar(ctx, actor, accion, recurso, finalidadDocumentalPlantillasCT,
		p.motivoDocumental, p.materialDocumental, audienciaDocumentalPlantillasCT)
}

// El indicador de interfaz evalúa la política vigente sin registrar una
// concesión candidata. Editar/Publicar volverán a autorizar el material exacto.
func (p *proveedorCatalogoPlantillasCT) ComprobarCapacidadCatalogoPlantillas(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
) (bool, error) {
	if p == nil || !recursoCatalogoPlantillasCTValido(accion, recurso, true) {
		return false, vecdomain.ErrAutorizacionDenegada
	}
	solicitud, operativo, err := p.solicitud(ctx, actor, accion, recurso,
		finalidadCatalogoPlantillasCT, p.motivoCatalogo)
	if err != nil {
		return false, err
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return false, vecdomain.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	seleccionado, err := p.pdp.contextoSolicitud(ctx, solicitud)
	if err != nil {
		return false, vecdomain.ErrAutorizacionDenegada
	}
	_, _, err = p.pdp.servicio.PrepararSolicitudLigadaV3(seleccionado, solicitud, operativo.Resultado)
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return false, nil
	}
	return err == nil, err
}

func (p *proveedorCatalogoPlantillasCT) autorizar(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
	finalidad string, motivo vecdomain.ReferenciaEntradaCatalogo,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, audiencia string,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if material == nil {
		return vacio, plantillasapp.ErrNoDisponible
	}
	solicitud, operativo, err := p.solicitud(ctx, actor, accion, recurso, finalidad, motivo)
	if err != nil {
		return vacio, err
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	ctx = context.WithValue(ctx, claveSolicitudAutorizacionContratacionTemporalDesarrollo{}, datos)
	decision, confirmacion, err := p.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		return vacio, err
	}
	exportacion, err := material.proveerMaterialConfirmacion(ctx, solicitud, decision, confirmacion, motivo, operativo.Resultado)
	if err != nil || exportacion.ValidarEstructura() != nil {
		return vacio, plantillasapp.ErrNoDisponible
	}
	resumen := exportacion.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audiencia ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		exportacion.PersonaVersion() != actor.Instantanea.PersonaVersion ||
		exportacion.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	return exportacion, nil
}

func (p *proveedorCatalogoPlantillasCT) solicitud(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
	finalidad string, motivo vecdomain.ReferenciaEntradaCatalogo,
) (vecdomain.SolicitudAutorizacionLigadaV3, contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if p == nil || p.soporte == nil || p.pdp == nil || contextoInterfazNulo(ctx) ||
		ctx.Err() != nil || actor.Validar() != nil || recurso.Validar() != nil ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	actual, err := p.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || actual.Resultado.Validar() != nil ||
		actual.Vinculo.ValidarPara(actual.Resultado) != nil ||
		!actual.Vinculo.VigenteEn(p.reloj.Ahora(), actual.Resultado) ||
		actor.Principal.ID != actual.Resultado.Contexto.Principal.ID ||
		actor.PerfilActivoRef != actual.Resultado.Contexto.PerfilActivoRef ||
		actor.Instantanea.PersonaVersion != actual.Resultado.Contexto.Instantanea.PersonaVersion ||
		actor.Instantanea.PerfilVersion != actual.Resultado.Contexto.Instantanea.PerfilVersion {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	// La fuente vuelve a resolver el contexto; el actor entregado por el
	// handler no puede sustituir una sesión revalidada ni cambiar de perfil.
	huellaActor, err := actor.HuellaSHA256VinculadaV2()
	huellaActual, errActual := actual.Resultado.Contexto.HuellaSHA256VinculadaV2()
	if err != nil || errActual != nil || huellaActor != huellaActual {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, plantillasapp.ErrNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: actual.Vinculo, ReferenciaMotivo: motivo,
		Accion: accion, Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	return solicitud, contextoSeguridadComunDesarrollo{Vinculo: actual.Vinculo, Resultado: actual.Resultado}, nil
}

func recursoCatalogoPlantillasCTValido(accion string, r vecdomain.RecursoAutorizable, indicador bool) bool {
	if r.Validar() != nil || r.Referencia != plantillasapp.CatalogoID || r.ModuloID != plantillasapp.ModuloID ||
		r.Tipo != tipoCatalogoPlantillasCT || len(r.Ambitos) != 0 {
		return false
	}
	switch accion {
	case "contratacion_temporal.plantillas_documentos.consultar":
		return !indicador && len(r.Atributos) == 1 && huellaMaterialPlantillasCTValida(r.Atributos["material_sha256"])
	case "contratacion_temporal.plantillas_documentos.editar", "contratacion_temporal.plantillas_documentos.publicar":
		if !indicador {
			return len(r.Atributos) == 1 && huellaMaterialPlantillasCTValida(r.Atributos["material_sha256"])
		}
		operacion := "editar"
		if accion == "contratacion_temporal.plantillas_documentos.publicar" {
			operacion = "publicar"
		}
		version, errVersion := strconv.Atoi(r.Atributos["version"])
		revision, errRevision := strconv.Atoi(r.Atributos["revision"])
		return len(r.Atributos) == 4 && r.Atributos["operacion"] == operacion &&
			(r.Atributos["estado"] == "borrador" || r.Atributos["estado"] == "publicado") &&
			errVersion == nil && errRevision == nil && version >= 1 && revision >= 0 &&
			strconv.Itoa(version) == r.Atributos["version"] && strconv.Itoa(revision) == r.Atributos["revision"]
	}
	return false
}

func recursoDocumentalPlantillasCTValido(accion string, r vecdomain.RecursoAutorizable) bool {
	return (accion == "contratacion_temporal.plantillas_documentos.documental_listar" ||
		accion == "contratacion_temporal.plantillas_documentos.documental_descargar") &&
		r.Validar() == nil && r.ModuloID == plantillasapp.ModuloID && r.Tipo == tipoDocumentalPlantillasCT &&
		len(r.Ambitos) == 0 && len(r.Atributos) == 1 &&
		huellaMaterialPlantillasCTValida(r.Atributos["material_sha256"]) &&
		len(r.Referencia) > len("expediente:") && r.Referencia[:len("expediente:")] == "expediente:"
}

func huellaMaterialPlantillasCTValida(v string) bool {
	if len(v) != 64 {
		return false
	}
	for _, c := range v {
		if c < '0' || c > '9' && c < 'a' || c > 'f' {
			return false
		}
	}
	return true
}

var _ plantillaspg.ProveedorAutorizacion = (*proveedorCatalogoPlantillasCT)(nil)
var _ plantillaspg.ProveedorAutorizacionDocumental = (*proveedorCatalogoPlantillasCT)(nil)
var _ plantillaspg.ResolverActor = (*proveedorCatalogoPlantillasCT)(nil)
