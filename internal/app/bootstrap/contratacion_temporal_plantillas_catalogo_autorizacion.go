package bootstrap

import (
	"context"
	"errors"
	"strconv"

	plantillashttp "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpapi/plantillascatalogo"
	plantillaspg "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres/plantillascatalogo"
	plantillasapp "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	finalidadCatalogoPlantillasCT   = "gestionar_catalogo_plantillas_contratacion_temporal"
	audienciaCatalogoPlantillasCT   = "vec_contratacion_temporal.catalogo_plantillas.v1"
	tipoCatalogoPlantillasCT        = "catalogo_plantillas_contratacion_temporal"
	audienciaDocumentalPlantillasCT = "vec_contratacion_temporal.catalogo_plantillas_documental.v1"
	finalidadDocumentalPlantillasCT = "consultar_borradores_expediente"
	tipoDocumentalPlantillasCT      = "catalogo_plantillas_documental_ct"
)

// proveedorCatalogoPlantillasCT reutiliza identidad mTLS y el PDP central.
// El perfil de catálogo es distinto del de tramitación y se coteja en cada
// llamada; ninguna instantánea local puede reemplazar una revocación vigente.
type proveedorCatalogoPlantillasCT struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
	pdp     interface {
		vecports.AutorizadorSolicitudLigadaV3
		vecports.PreparadorRegistroCompuestoSolicitudLigadaV3
	}
	materialCatalogo *proveedorMaterialAltaContratacionTemporalDesarrollo
	motivo           vecdomain.ReferenciaEntradaCatalogo
	reloj            relojContratacionTemporalDesarrollo
}

func nuevoProveedorCatalogoPlantillasCT(
	soporte *soporteAltaContratacionTemporalDesarrollo,
	pdp *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo,
	motivo vecdomain.ReferenciaEntradaCatalogo,
	reloj relojContratacionTemporalDesarrollo,
) (*proveedorCatalogoPlantillasCT, error) {
	if soporte == nil || pdp == nil || material == nil ||
		motivo != motivoCatalogoPlantillasCTDesarrollo() ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, plantillasapp.ErrNoDisponible
	}
	return &proveedorCatalogoPlantillasCT{soporte: soporte, pdp: pdp,
		materialCatalogo: material, motivo: motivo, reloj: reloj}, nil
}

func rutaAccionPlantillasCatalogoCT(accion string) string {
	switch accion {
	case "contratacion_temporal.plantillas_documentos.consultar":
		return plantillashttp.RutaCatalogo
	case "contratacion_temporal.plantillas_documentos.editar":
		return plantillashttp.RutaEntradas
	case "contratacion_temporal.plantillas_documentos.publicar":
		return plantillashttp.RutaPublicar
	default:
		return ""
	}
}

func (p *proveedorCatalogoPlantillasCT) contextoVigente(ctx context.Context, accion string, indicador bool) (contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if p == nil || p.soporte == nil || p.pdp == nil || contextoInterfazNulo(ctx) || ctx.Err() != nil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	ruta := rutaAccionPlantillasCatalogoCT(accion)
	capacidadEsperada := accion
	capacidad, valida := p.soporte.capacidadValida(ctx)
	if indicador {
		// Servicio.Consultar evalúa los controles dentro del GET ya
		// autenticado. Es una evaluación sin concesión ni efecto durable.
		ruta = plantillashttp.RutaCatalogo
		capacidadEsperada = "contratacion_temporal.plantillas_documentos.consultar"
	} else if accion == "contratacion_temporal.plantillas_documentos.consultar" {
		// Editar y publicar consultan primero la cabeza actual dentro del
		// mismo POST. Esa lectura obtiene su propia decisión V3 nominal.
		switch capacidad.ruta {
		case plantillashttp.RutaEntradas:
			ruta = plantillashttp.RutaEntradas
			capacidadEsperada = "contratacion_temporal.plantillas_documentos.editar"
		case plantillashttp.RutaPublicar:
			ruta = plantillashttp.RutaPublicar
			capacidadEsperada = "contratacion_temporal.plantillas_documentos.publicar"
		}
	}
	frontera, fronteraValida := fronteraSeguridadComunDesdeContexto(ctx)
	perfil := p.soporte.contexto.Resultado.Contexto.PerfilActivoRef
	if ruta == "" || !valida || capacidad.ruta != ruta || !fronteraValida ||
		frontera.ruta != ruta || frontera.descriptor.ClaveCapacidad != capacidadEsperada ||
		!frontera.descriptor.admitePerfil(perfil) ||
		capacidad.principal.ID != p.soporte.principalID ||
		capacidad.principal.Attributes["certificate_sha256"] != p.soporte.certificadoSHA256 {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil || operativo.Resultado.Validar() != nil ||
		operativo.Vinculo.ValidarPara(operativo.Resultado) != nil ||
		!operativo.Vinculo.VigenteEn(p.reloj.Ahora(), operativo.Resultado) ||
		operativo.Resultado.Contexto.PerfilActivoRef != perfil {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	return contextoSeguridadComunDesarrollo{Vinculo: operativo.Vinculo, Resultado: operativo.Resultado}, nil
}

func (p *proveedorCatalogoPlantillasCT) ResolverContextoActor(ctx context.Context) (vecdomain.ContextoActor, error) {
	if p == nil || ctx == nil {
		return vecdomain.ContextoActor{}, vecdomain.ErrAutorizacionDenegada
	}
	capacidad, valido := p.soporte.capacidadValida(ctx)
	if !valido {
		return vecdomain.ContextoActor{}, vecdomain.ErrAutorizacionDenegada
	}
	accion := ""
	switch capacidad.ruta {
	case plantillashttp.RutaCatalogo:
		accion = "contratacion_temporal.plantillas_documentos.consultar"
	case plantillashttp.RutaEntradas:
		accion = "contratacion_temporal.plantillas_documentos.editar"
	case plantillashttp.RutaPublicar:
		accion = "contratacion_temporal.plantillas_documentos.publicar"
	}
	operativo, err := p.contextoVigente(ctx, accion, false)
	if err != nil {
		return vecdomain.ContextoActor{}, err
	}
	return operativo.Resultado.Contexto.Clonar()
}

func (p *proveedorCatalogoPlantillasCT) AutorizarCatalogoPlantillas(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacio := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	if p == nil || p.materialCatalogo == nil || !recursoCatalogoPlantillasCTValido(accion, recurso, false) {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	solicitud, operativo, err := p.solicitud(ctx, actor, accion, recurso, false)
	if err != nil {
		return vacio, err
	}
	decision, confirmacion, err := p.pdp.ExigirSolicitudLigadaV3(ctx, solicitud, operativo.Resultado)
	if err != nil {
		return vacio, err
	}
	exportacion, err := p.materialCatalogo.proveerMaterialConfirmacion(
		ctx, solicitud, decision, confirmacion, p.motivo, operativo.Resultado)
	if err != nil || exportacion.ValidarEstructura() != nil {
		return vacio, plantillasapp.ErrNoDisponible
	}
	resumen := exportacion.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || resumen.Operacion() != accion || resumen.AudienciaConsumo() != audienciaCatalogoPlantillasCT ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		exportacion.PersonaVersion() != actor.Instantanea.PersonaVersion ||
		exportacion.PerfilVersion() != actor.Instantanea.PerfilVersion {
		return vacio, vecdomain.ErrAutorizacionDenegada
	}
	return exportacion, nil
}

// El indicador sólo prepara la solicitud en el PDP central; la escritura
// emitirá una decisión nueva y CT131 la consumirá dentro de su transacción.
func (p *proveedorCatalogoPlantillasCT) ComprobarCapacidadCatalogoPlantillas(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable,
) (bool, error) {
	if p == nil || !recursoCatalogoPlantillasCTValido(accion, recurso, true) {
		return false, vecdomain.ErrAutorizacionDenegada
	}
	solicitud, operativo, err := p.solicitud(ctx, actor, accion, recurso, true)
	if err != nil {
		return false, err
	}
	// Esta preparación no registra candidata ni denegación. Una consulta
	// visual no equivale a intentar una escritura administrativa.
	decision, _, err := p.pdp.PrepararRegistroCompuestoSolicitudLigadaV3(
		ctx, solicitud, operativo.Resultado, seguridadvec.GeneradorReferenciasCriptograficas{})
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	concedida, _, err := decision.Resultado()
	return err == nil && concedida, err
}

func (p *proveedorCatalogoPlantillasCT) solicitud(
	ctx context.Context, actor vecdomain.ContextoActor, accion string, recurso vecdomain.RecursoAutorizable, indicador bool,
) (vecdomain.SolicitudAutorizacionLigadaV3, contextoSeguridadComunDesarrollo, error) {
	vacio := contextoSeguridadComunDesarrollo{}
	if p == nil || actor.Validar() != nil || recurso.Validar() != nil {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	operativo, err := p.contextoVigente(ctx, accion, indicador)
	if err != nil {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, err
	}
	huellaActor, err := actor.HuellaSHA256VinculadaV2()
	huellaActual, errActual := operativo.Resultado.Contexto.HuellaSHA256VinculadaV2()
	if err != nil || errActual != nil || huellaActor != huellaActual ||
		actor.Principal.ID != operativo.Resultado.Contexto.Principal.ID ||
		actor.PerfilActivoRef != operativo.Resultado.Contexto.PerfilActivoRef {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, plantillasapp.ErrNoDisponible
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: operativo.Vinculo, ReferenciaMotivo: p.motivo,
		Accion: accion, Recurso: recurso, Finalidad: finalidadCatalogoPlantillasCT, Correlacion: correlacion,
	})
	if err != nil {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	datos, err := solicitud.Datos()
	if err != nil || !solicitudAutorizacionPlantillasCTDesarrolloValida(rutaAccionPlantillasCatalogoCT(accion), datos) {
		return vecdomain.SolicitudAutorizacionLigadaV3{}, vacio, vecdomain.ErrAutorizacionDenegada
	}
	return solicitud, operativo, nil
}

func recursoCatalogoPlantillasCTValido(accion string, r vecdomain.RecursoAutorizable, indicador bool) bool {
	if r.Validar() != nil || r.Referencia != plantillasapp.CatalogoID || r.ModuloID != plantillasapp.ModuloID ||
		r.Tipo != tipoCatalogoPlantillasCT || len(r.Ambitos) != 1 ||
		r.Ambitos["organizacion_ref"] != organizacionAltaContratacionTemporalDesarrollo {
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
var _ plantillaspg.ResolverActor = (*proveedorCatalogoPlantillasCT)(nil)
