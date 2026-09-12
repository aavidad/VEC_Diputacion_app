package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	admin "vec-diputacion-granada/internal/modules/administracion"
	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

const accionConsultaConfiguracionCorreoAdministracionV3 = adminapp.AccionConsultarConfiguracionCorreo
const audienciaConsultaConfiguracionCorreoAdministracionV3 = "vec_administracion.consultar_configuracion_correo.v1"

var errAutorizacionConsultaConfiguracionCorreoV3 = errors.New("bootstrap: autorizacion de consulta SMTP no disponible")

type autorizadorConsultaConfiguracionCorreoV3 struct {
	sesion      sesionDurableAdministracionCorreoV3
	auditoria   *preparacionAuditoriaAdministracionDesarrollo
	emisor      emisorMaterialConfiguracionCorreoV3
	motivo      core.ReferenciaEntradaCatalogo
	referencias vp.GeneradorReferenciasAutorizacionV2
	reloj       vp.Reloj
}

var _ adminports.AutorizadorConsultaConfiguracionCorreo = (*autorizadorConsultaConfiguracionCorreoV3)(nil)

func nuevoAutorizadorConsultaConfiguracionCorreoV3(sesion sesionDurableAdministracionCorreoV3, auditoria *preparacionAuditoriaAdministracionDesarrollo, emisor emisorMaterialConfiguracionCorreoV3, motivo core.ReferenciaEntradaCatalogo, referencias vp.GeneradorReferenciasAutorizacionV2, reloj vp.Reloj) (*autorizadorConsultaConfiguracionCorreoV3, error) {
	if dependenciaAdministracionNula(sesion) || auditoria == nil || dependenciaAdministracionNula(emisor) || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) || dependenciaAdministracionNula(referencias) || dependenciaAdministracionNula(reloj) {
		return nil, errAutorizacionConsultaConfiguracionCorreoV3
	}
	return &autorizadorConsultaConfiguracionCorreoV3{sesion, auditoria, emisor, motivo, referencias, reloj}, nil
}

func (a *autorizadorConsultaConfiguracionCorreoV3) AutorizarConsultaConfiguracionCorreo(ctx context.Context, preparacion adminports.PreparacionConsultaConfiguracionCorreo) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	fallo := func() (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errAutorizacionConsultaConfiguracionCorreoV3
	}
	if a == nil || ctx == nil || ctx.Err() != nil || dependenciaAdministracionNula(a.sesion) || a.auditoria == nil || dependenciaAdministracionNula(a.emisor) || dependenciaAdministracionNula(a.referencias) || dependenciaAdministracionNula(a.reloj) || len(preparacion.PayloadNegocio) == 0 || len(preparacion.PayloadNegocio) > 32768 {
		return fallo()
	}
	capacidad, ok := capacidadAdministracionDesdeContexto(ctx)
	if !ok || capacidad.metodo != "GET" || capacidad.ruta != "/api/vec/administracion/configuracion-correo" {
		return fallo()
	}
	esperado, err := adminapp.PayloadConsultaConfiguracionCorreo(preparacion.Auditoria)
	if err != nil {
		return fallo()
	}
	defer borrarBytes(esperado)
	if !bytes.Equal(esperado, preparacion.PayloadNegocio) {
		return fallo()
	}
	sesion, err := a.sesion.ResolverSesionAdministracionCorreoV3(ctx)
	if err != nil || a.auditoria.ValidarAuditoriaConsultaParaSesion(ctx, preparacion.Auditoria, sesion) != nil {
		return fallo()
	}
	h := sha256.Sum256(esperado)
	recurso := core.RecursoAutorizable{Referencia: referenciaConfiguracionCorreoAdministracionV3, ModuloID: admin.ModuleID, Tipo: tipoRecursoConfiguracionCorreoAdministracion, Ambitos: map[string]string{"organizacion_ref": organizacionConfiguracionCorreoAdministracionV3}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, a.referencias)
	if err != nil {
		return fallo()
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: sesion.Vinculo, ReferenciaMotivo: a.motivo, Accion: accionConsultaConfiguracionCorreoAdministracionV3, Recurso: recurso, Finalidad: finalidadConfiguracionCorreoAdministracionV3, Correlacion: correlacion})
	if err != nil || a.auditoria.ValidarAuditoriaConsultaParaSesion(ctx, preparacion.Auditoria, sesion) != nil {
		return fallo()
	}
	decision, confirmacion, exportador, err := a.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, sesion.Resultado)
	if err != nil || ctx.Err() != nil || dependenciaAdministracionNula(exportador) || decision.ValidarPara(solicitud) != nil {
		return fallo()
	}
	orden, err := vp.NuevaOrdenRegistroConcesionCandidataAutorizacionLigadaV3(solicitud, decision, a.motivo, sesion.Resultado)
	ahora := a.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || confirmacion.ValidarPara(orden) != nil || !confirmacion.DentroDeVentanaEn(ahora) {
		return fallo()
	}
	canonica, err := core.RepresentacionCanonicaDecisionAutorizacionV3(decision)
	if err != nil {
		return fallo()
	}
	defer borrarBytes(canonica)
	if !camposDecisionConsultaConfiguracionCorreoExactos(canonica) {
		return fallo()
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || material.ValidarEstructura() != nil || a.auditoria.ValidarAuditoriaConsultaParaSesion(ctx, preparacion.Auditoria, sesion) != nil {
		return fallo()
	}
	ahora = a.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if ctx.Err() != nil || !confirmacion.DentroDeVentanaEn(ahora) || !materialConsultaConfiguracionCorreoExacto(material, recurso, canonica, sesion, ahora) {
		return fallo()
	}
	return material, nil
}

func camposDecisionConsultaConfiguracionCorreoExactos(canonica []byte) bool {
	var d struct {
		Campos       []string `json:"campos_permitidos"`
		Obligaciones []string `json:"obligaciones"`
	}
	if json.Unmarshal(canonica, &d) != nil || len(d.Obligaciones) != 0 {
		return false
	}
	return reflect.DeepEqual(d.Campos, []string{"configurada", "host", "modo_autenticacion", "modo_tls", "puerto", "referencia_ca", "remitente_fijo", "secreto_configurado", "server_name", "tiempo_maximo_ms", "usuario", "version"})
}

func materialConsultaConfiguracionCorreoExacto(material vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, recurso core.RecursoAutorizable, decision []byte, sesion contextoSesionAdministracionCorreoV3, ahora time.Time) bool {
	r := material.ResumenCapacidad()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	dec := material.DecisionCanonica()
	defer borrarBytes(dec)
	contexto := material.ContextoActorCanonico()
	defer borrarBytes(contexto)
	return err == nil && bytes.Equal(dec, decision) && bytes.Equal(contexto, sesion.Resultado.RepresentacionCanonica) && r.ContextoRef() == sesion.Resultado.RegistroContextoRef && r.ContextoHuellaSHA256() == sesion.Resultado.HuellaSHA256 && r.Operacion() == accionConsultaConfiguracionCorreoAdministracionV3 && r.EfectoRef() == referenciaConfiguracionCorreoAdministracionV3 && r.EfectoHuellaSHA256() == huella && r.AudienciaConsumo() == audienciaConsultaConfiguracionCorreoAdministracionV3 && !ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn())
}
