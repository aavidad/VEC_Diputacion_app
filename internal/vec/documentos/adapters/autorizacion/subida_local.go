package autorizacion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	audienciaActoSubidaLocal     = "vec_documentos.subida_local.acto_modulo.v1"
	audienciaDescargaSubidaLocal = "vec_documentos.subida_local.descarga_propia.v1"
)

type AutoridadSubidaLocal struct {
	emisor   EmisorMaterialSubidaLocalV3
	vinculos VinculoSubidaLocalCertificadoV3
	motivo   vecdomain.ReferenciaEntradaCatalogo
	reloj    vecports.Reloj
}

type EmisorMaterialSubidaLocalV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3,
		vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3,
		vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type VinculoSubidaLocalCertificadoV3 interface {
	VinculoCargaLocalCertificado(context.Context, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.VinculoAutenticacionActorV2, error)
}

type correlacionSubidaLocalConservada struct{ ref string }

func (g correlacionSubidaLocalConservada) NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error) {
	return g.ref, nil
}

func NuevaAutoridadSubidaLocal(emisor EmisorMaterialSubidaLocalV3,
	vinculos VinculoSubidaLocalCertificadoV3, motivo vecdomain.ReferenciaEntradaCatalogo,
	reloj vecports.Reloj) (*AutoridadSubidaLocal, error) {
	if nulo(emisor) || nulo(vinculos) || nulo(reloj) ||
		motivo.CatalogoID != "motivos_autorizacion_meritos_cartera" ||
		!vecdomain.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, docports.ErrSubidaLocalNoDisponible
	}
	return &AutoridadSubidaLocal{emisor: emisor, vinculos: vinculos, motivo: motivo, reloj: reloj}, nil
}

func parametrosAccionLocal(accion string) (modulo, tipo, finalidad, audiencia string, ok bool) {
	switch accion {
	case docports.AccionMeritosComprobanteSubirLocal:
		return "meritos", "hecho", "custodiar_evidencia_meritos", audienciaActoSubidaLocal, true
	case docports.AccionMeritosComprobanteDescargarLocal:
		return "meritos", "hecho", "custodiar_evidencia_meritos", audienciaDescargaSubidaLocal, true
	default:
		// Dietas y Cronos se habilitan cuando sus módulos publiquen fuente,
		// motivo y fachada propietaria con la misma semántica.
		return "", "", "", "", false
	}
}

func (a *AutoridadSubidaLocal) emitir(ctx context.Context, actor vecdomain.ResultadoContextoActorRegistradoV2,
	accion, id, finalidad, correlacion string, recurso vecdomain.RecursoAutorizable,
	audiencia string) (vecdomain.SolicitudAutorizacionLigadaV3,
	vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
	vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var zs vecdomain.SolicitudAutorizacionLigadaV3
	var zd vecdomain.DecisionAutorizacionLigadaV3
	var zc vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	var zm vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	if a == nil || ctx == nil || ctx.Err() != nil || actor.Validar() != nil || recurso.Validar() != nil ||
		recurso.Referencia != id || finalidad == "" || correlacion == "" {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	vinculo, err := a.vinculos.VinculoCargaLocalCertificado(ctx, actor)
	if err != nil || vinculo.ValidarPara(actor) != nil {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	corr, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx,
		correlacionSubidaLocalConservada{correlacion})
	if err != nil {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: vinculo, ReferenciaMotivo: a.motivo, Accion: accion,
		Recurso: recurso, Finalidad: finalidad, Correlacion: corr})
	if err != nil {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	decision, confirmacion, exportador, err := a.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, actor)
	if err != nil || nulo(exportador) {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, actor,
		a.motivo, material, audiencia) {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	ahora := a.reloj.Ahora().UTC()
	resumen := material.ResumenCapacidad()
	if ahora.IsZero() || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return zs, zd, zc, zm, docports.ErrSubidaLocalDenegada
	}
	return solicitud, decision, confirmacion, material, nil
}

func (a *AutoridadSubidaLocal) AutorizarSubidaLocal(ctx context.Context, pre []byte,
	actor vecdomain.ResultadoContextoActorRegistradoV2, id, recursoRef, finalidad, correlacion string,
) (docports.AutorizacionActoSubidaLocalV3, error) {
	var cero docports.AutorizacionActoSubidaLocalV3
	if len(pre) == 0 || len(pre) > 65536 || actor.Validar() != nil ||
		!vecdomain.ReferenciaCorrelacionAutorizacionV2Valida(correlacion) {
		return cero, docports.ErrSubidaLocalDenegada
	}
	var dato struct {
		Accion         string `json:"accion"`
		CargaID        string `json:"carga_id"`
		DocumentoID    string `json:"documento_id"`
		RecursoRef     string `json:"recurso_ref"`
		PersonaRef     string `json:"persona_ref"`
		CorrelacionRef string `json:"correlacion_ref"`
	}
	if json.Unmarshal(pre, &dato) != nil || dato.Accion == "" || dato.PersonaRef != actor.Contexto.PersonaRef || dato.CorrelacionRef != correlacion {
		return cero, docports.ErrSubidaLocalDenegada
	}
	modulo, tipo, esperada, audiencia, ok := parametrosAccionLocal(dato.Accion)
	if !ok || esperada != finalidad || (dato.CargaID != id && dato.DocumentoID != id) ||
		(dato.RecursoRef != "" && dato.RecursoRef != recursoRef) {
		return cero, docports.ErrSubidaLocalDenegada
	}
	suma := sha256.Sum256(pre)
	recurso := vecdomain.RecursoAutorizable{Referencia: id, ModuloID: modulo, Tipo: tipo,
		Ambitos:   map[string]string{"persona_ref": actor.Contexto.PersonaRef},
		Atributos: map[string]string{"preimagen_sha256": hex.EncodeToString(suma[:])}}
	_, _, _, material, err := a.emitir(ctx, actor, dato.Accion, id, finalidad, correlacion, recurso, audiencia)
	if err != nil {
		return cero, err
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || material.ResumenCapacidad().EfectoHuellaSHA256() != huella {
		return cero, docports.ErrSubidaLocalDenegada
	}
	return docports.AutorizacionActoSubidaLocalV3{Accion: dato.Accion, CargaID: id, RecursoRef: recursoRef,
		Finalidad: finalidad, CorrelacionRef: correlacion, Material: material}, nil
}

func (a *AutoridadSubidaLocal) AutorizarAlmacenSubidaLocal(ctx context.Context,
	actor vecdomain.ResultadoContextoActorRegistradoV2, acto docports.ActoSubidaLocal,
	accion, correlacion string, fase vecports.FaseAlmacenSubidaLocal,
	v vecports.VinculosOperacionAlmacen) (vecports.ContextoOperacionAlmacen, error) {
	var cero vecports.ContextoOperacionAlmacen
	modulo, tipo, finalidad, audiencia, ok := parametrosAccionLocal(accion)
	if !ok || acto.Recurso.ModuloID != modulo || acto.Recurso.Tipo != tipo || acto.Finalidad != finalidad ||
		acto.PersonaRef != actor.Contexto.PersonaRef || v.OperacionRef == "" ||
		!vecdomain.ReferenciaCorrelacionAutorizacionV2Valida(correlacion) {
		return cero, docports.ErrSubidaLocalDenegada
	}
	atributos := map[string]string{
		"recurso_dueno_ref": acto.Recurso.Referencia, "fase": string(fase),
		vecports.AtributoAlmacenOperacionRef:        v.OperacionRef,
		vecports.AtributoAlmacenCargaRef:            v.CargaRef,
		vecports.AtributoAlmacenClasificacion:       v.Clasificacion,
		vecports.AtributoAlmacenSujetoSeudonimoHMAC: v.SujetoSeudonimoHMAC,
		vecports.AtributoAlmacenHuellaSolicitudHMAC: v.HuellaSolicitudHMAC,
		vecports.AtributoAlmacenEfectoRef:           v.EfectoRef,
	}
	if v.ObjetoVinculado != (vecports.ReferenciaObjetoAlmacen{}) {
		atributos[vecports.AtributoAlmacenObjetoRef] = v.ObjetoVinculado.Referencia
		atributos[vecports.AtributoAlmacenObjetoVersion] = v.ObjetoVinculado.Version
	}
	recurso := vecdomain.RecursoAutorizable{Referencia: v.OperacionRef, ModuloID: modulo, Tipo: tipo,
		Ambitos: map[string]string{"persona_ref": acto.PersonaRef}, Atributos: atributos}
	s, d, c, _, err := a.emitir(ctx, actor, accion, v.OperacionRef, finalidad, correlacion, recurso, audiencia)
	if err != nil {
		return cero, err
	}
	instante := a.reloj.Ahora().UTC().Truncate(time.Microsecond)
	return vecports.NuevoContextoSubidaLocalAlmacenV3(s, d, c, v, fase, instante)
}

var _ docports.AutoridadActoSubidaLocalV3 = (*AutoridadSubidaLocal)(nil)
