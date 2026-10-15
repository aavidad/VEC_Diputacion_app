package bootstrap

import (
	"context"
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	seguridad "vec-diputacion-granada/internal/vec/adapters/seguridad"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const audienciaJustificanteBolsaSubir = "vec_documentos.bolsa.justificante_renuncia.subida.v1"
const audienciaJustificanteBolsaDescargar = "vec_documentos.bolsa.justificante_renuncia.descarga_propia.v1"
const audienciaJustificanteBolsaPreparar = "vec_documentos.bolsa.justificante_renuncia.preparar_subida.v1"
const audienciaJustificanteBolsaRRHH = "vec_documentos.bolsa.justificante_renuncia.descarga_rrhh.v1"
const audienciaJustificanteBolsaPrepararRRHH = "vec_documentos.bolsa.justificante_renuncia.preparar_descarga_rrhh.v1"

type proveedorIdentidadJustificanteBolsa struct{}

func (proveedorIdentidadJustificanteBolsa) PrepararJustificanteRenuncia(r *http.Request) (bolsahttp.IdentidadJustificanteRenuncia, error) {
	var vacia bolsahttp.IdentidadJustificanteRenuncia
	if r == nil {
		return vacia, bolsahttp.ErrAutenticacionAusente
	}
	c, ok := r.Context().Value(claveOrdenMiBolsaPortalExterno{}).(contextoMiBolsaPortalExterno)
	if !ok || c.autoridad == nil {
		return vacia, bolsahttp.ErrAutenticacionAusente
	}
	orden, err := c.autoridad.PrepararMiBolsa(r)
	if err != nil {
		return vacia, err
	}
	corr, err := orden.Correlacion.ValorCanonico()
	if err != nil {
		return vacia, err
	}
	return bolsahttp.IdentidadJustificanteRenuncia{Actor: orden.ResultadoContexto, Vinculo: orden.Vinculo, CorrelacionRef: corr}, nil
}

// El vínculo fijado se crea dentro de una petición ya identificada. Ningún
// estado mutable de la factoría puede cambiar el perfil de otra petición.
type vinculoJustificanteBolsaFijado struct {
	identidad bolsahttp.IdentidadJustificanteRenuncia
}

func (v vinculoJustificanteBolsaFijado) VinculoCargaLocalCertificado(ctx context.Context, actor vec.ResultadoContextoActorRegistradoV2) (vec.VinculoAutenticacionActorV2, error) {
	if ctx == nil || ctx.Err() != nil || actor.Validar() != nil ||
		v.identidad.Actor.RegistroContextoRef != actor.RegistroContextoRef ||
		v.identidad.Actor.HuellaSHA256 != actor.HuellaSHA256 || v.identidad.Vinculo.ValidarPara(actor) != nil {
		return vec.VinculoAutenticacionActorV2{}, docports.ErrSubidaLocalDenegada
	}
	return v.identidad.Vinculo, nil
}

type emisoresJustificanteBolsa struct {
	subir, descargar *emisorMaterialRenovableCTDesarrollo
}

func (e emisoresJustificanteBolsa) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s vec.SolicitudAutorizacionLigadaV3,
	a vec.ResultadoContextoActorRegistradoV2) (vec.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	d, err := s.Datos()
	if err != nil {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, err
	}
	emisor := e.subir
	if d.Accion == docports.AccionBolsaJustificanteRenunciaDescargarLocal {
		emisor = e.descargar
	} else if d.Accion != docports.AccionBolsaJustificanteRenunciaSubirLocal {
		emisor = nil
	}
	if emisor == nil {
		return vec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, docports.ErrSubidaLocalDenegada
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, a)
}

type autoridadPreparacionJustificanteBolsa struct {
	emisor *emisorMaterialRenovableCTDesarrollo
	motivo vec.ReferenciaEntradaCatalogo
}

func (p autoridadPreparacionJustificanteBolsa) PrepararJustificanteRenuncia(ctx context.Context,
	a vec.ResultadoContextoActorRegistradoV2, v vec.VinculoAutenticacionActorV2, bolsa string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	candidato, err := bolsaapp.CandidatoDelActo(a, v)
	if err != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, err
	}
	recurso := vec.RecursoAutorizable{Referencia: bolsa, ModuloID: "bolsa", Tipo: bolsaports.TipoRecursoJustificanteRenuncia,
		Ambitos:   map[string]string{"candidato_ref": candidato},
		Atributos: map[string]string{"bolsa_ref": bolsa, "candidato_ref": candidato}}
	return emitirPreparacionJustificanteBolsa(ctx, p.emisor, p.motivo, a, v, recurso,
		bolsaports.AccionSubirJustificanteRenunciaPropio, audienciaJustificanteBolsaPreparar)
}

type autoridadPreparacionJustificanteBolsaRRHH struct {
	emisor *emisorMaterialRenovableCTDesarrollo
	motivo vec.ReferenciaEntradaCatalogo
}

func (p autoridadPreparacionJustificanteBolsaRRHH) PrepararConsultaJustificanteRenunciaRRHH(ctx context.Context,
	a vec.ResultadoContextoActorRegistradoV2, v vec.VinculoAutenticacionActorV2, bolsa, participacion string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	recurso := vec.RecursoAutorizable{Referencia: participacion, ModuloID: "bolsa", Tipo: bolsaports.TipoRecursoJustificanteRenuncia,
		Ambitos: map[string]string{"bolsa_ref": bolsa, "participacion_ref": participacion}, Atributos: map[string]string{}}
	return emitirPreparacionJustificanteBolsa(ctx, p.emisor, p.motivo, a, v, recurso,
		bolsaports.AccionDescargarJustificanteRenunciaRRHH, audienciaJustificanteBolsaPrepararRRHH)
}

func emitirPreparacionJustificanteBolsa(ctx context.Context, emisor *emisorMaterialRenovableCTDesarrollo,
	motivo vec.ReferenciaEntradaCatalogo, actor vec.ResultadoContextoActorRegistradoV2, vinculo vec.VinculoAutenticacionActorV2,
	recurso vec.RecursoAutorizable, accion, audiencia string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var vacio vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if ctx == nil || ctx.Err() != nil || emisor == nil || vinculo.ValidarPara(actor) != nil || recurso.Validar() != nil {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	corr, err := vec.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacio, err
	}
	s, err := vec.NuevaSolicitudAutorizacionLigadaV3(vec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: motivo, Accion: accion, Recurso: recurso,
		Finalidad: bolsaports.FinalidadJustificanteRenunciaBolsa, Correlacion: corr})
	if err != nil {
		return vacio, err
	}
	d, c, exportador, err := emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, actor)
	if err != nil {
		return vacio, err
	}
	if exportador == nil {
		return vacio, docports.ErrSubidaLocalNoDisponible
	}
	m, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(s, d, c, actor, motivo, m, audiencia) {
		return vacio, docports.ErrSubidaLocalDenegada
	}
	return m, nil
}

type resolutorDocumentalJustificanteBolsa struct {
	bolsa    bolsaports.ResolutorJustificanteRenuncia
	bolsaRef string
}

func (r resolutorDocumentalJustificanteBolsa) ResolverActoSubidaLocal(ctx context.Context, q docports.SolicitudSubidaLocal) (docports.ActoSubidaLocal, error) {
	a, err := r.bolsa.ResolverJustificanteRenuncia(ctx, bolsaports.SolicitudJustificanteRenuncia{
		ActorRegistradoV2: q.ActorRegistradoV2, VinculoV2: q.VinculoV2, BolsaRef: r.bolsaRef})
	if err != nil {
		return docports.ActoSubidaLocal{}, err
	}
	return docports.ActoSubidaLocal{Recurso: vec.RecursoAutorizable{Referencia: a.RecursoRef, ModuloID: "bolsa",
		Tipo: bolsaports.TipoRecursoJustificanteRenuncia, Ambitos: map[string]string{"candidato_ref": a.CandidatoRef}},
		PersonaRef: a.PersonaRef, Finalidad: bolsaports.FinalidadJustificanteRenunciaBolsa,
		Clasificacion: "confidencial", VersionDueno: a.VersionDueno}, nil
}
