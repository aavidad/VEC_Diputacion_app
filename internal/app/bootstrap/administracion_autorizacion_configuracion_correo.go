package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"reflect"
	"time"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	adminapp "vec-diputacion-granada/internal/modules/administracion/application"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	accionConfiguracionCorreoAdministracionV3     = "administracion.configuracion_correo.actualizar"
	finalidadConfiguracionCorreoAdministracionV3  = "administrar_integraciones"
	tipoRecursoConfiguracionCorreoAdministracion  = "configuracion_correo_administracion"
	referenciaConfiguracionCorreoAdministracionV3 = "configuracion:smtp:diputacion"
	audienciaConfiguracionCorreoAdministracionV3  = "vec_administracion.actualizar_configuracion_correo.v1"
)

var errAutorizacionConfiguracionCorreoAdministracionV3 = errors.New("bootstrap: autorizacion V3 de configuracion de correo no disponible")

// sesionDurableAdministracionCorreoV3 no acepta un Principal enviado por la
// petición. Su implementador debe crear y revalidar una sesión ADMIN separada
// y devolver el vínculo y recibo registrados que esa sesión acaba de producir.
// La composición mTLS es la única que puede implementarlo.
type sesionDurableAdministracionCorreoV3 interface {
	ResolverSesionAdministracionCorreoV3(context.Context) (contextoSesionAdministracionCorreoV3, error)
}

type contextoSesionAdministracionCorreoV3 struct {
	Principal vecdomain.Principal
	Vinculo   vecdomain.VinculoAutenticacionActorV2
	Resultado vecdomain.ResultadoContextoActorRegistradoV2
}

// proveedorSesionDurableAdministracionCorreoV3 registra una sesión distinta
// para ADMIN en cada petición válida. La capacidad mTLS sólo entra aquí; ni el
// formulario ni Principal pueden fabricar este recorrido.
type proveedorSesionDurableAdministracionCorreoV3 struct {
	registro    httpseguridad.RegistroSesiones
	revalidador vecports.RevalidadorAutenticacionActorV1
	resolutor   vecdomain.ResolutorContextoActorRegistradoV2
	permisos    fuentePermisoAdministracionCorreoV3
	reloj       interface{ Ahora() time.Time }
}

// fuentePermisoAdministracionCorreoV3 se consulta después de revalidar la
// sesión y resolver el actor. No se alimenta de Roles/Permissions del
// ContextoActor ni llama de vuelta a PrincipalConfiguracionCorreo.
type fuentePermisoAdministracionCorreoV3 interface {
	ResolverPrincipalAdministracionCorreoV3(context.Context, vecdomain.VinculoAutenticacionActorV2, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.Principal, error)
}

func nuevoProveedorSesionDurableAdministracionCorreoV3(registro httpseguridad.RegistroSesiones, revalidador vecports.RevalidadorAutenticacionActorV1, resolutor vecdomain.ResolutorContextoActorRegistradoV2, permisos fuentePermisoAdministracionCorreoV3, reloj interface{ Ahora() time.Time }) (*proveedorSesionDurableAdministracionCorreoV3, error) {
	if dependenciaAdministracionAutorizacionNula(registro) || dependenciaAdministracionAutorizacionNula(revalidador) || dependenciaAdministracionAutorizacionNula(resolutor) || dependenciaAdministracionAutorizacionNula(permisos) || dependenciaAdministracionAutorizacionNula(reloj) {
		return nil, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return &proveedorSesionDurableAdministracionCorreoV3{registro: registro, revalidador: revalidador, resolutor: resolutor, permisos: permisos, reloj: reloj}, nil
}

func (p *proveedorSesionDurableAdministracionCorreoV3) ResolverSesionAdministracionCorreoV3(ctx context.Context) (contextoSesionAdministracionCorreoV3, error) {
	var cero contextoSesionAdministracionCorreoV3
	if p == nil || ctx == nil || ctx.Err() != nil || dependenciaAdministracionAutorizacionNula(p.registro) || dependenciaAdministracionAutorizacionNula(p.revalidador) || dependenciaAdministracionAutorizacionNula(p.resolutor) || dependenciaAdministracionAutorizacionNula(p.permisos) || dependenciaAdministracionAutorizacionNula(p.reloj) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	capacidad, ok := capacidadAdministracionDesdeContexto(ctx)
	if !ok || capacidad.identidad == nil || ((capacidad.metodo != "GET" && capacidad.metodo != "PUT") && !rutaLecturaAdministracionDesarrollo(capacidad.ruta, capacidad.metodo)) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	ahora := p.reloj.Ahora()
	if !capacidadAdministracionCorreoV3Valida(capacidad, ahora) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	alta, err := altaSesionAdministracionCorreoV3(capacidad, ahora)
	if err != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	confirmacion, err := p.registro.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil || ctx.Err() != nil || confirmacion.ValidarPara(alta) != nil || confirmacion.CuentaRef != capacidad.identidad.cuentaRef || confirmacion.CuentaOrdinariaRef != capacidad.identidad.cuentaOrdinariaRef || !ahora.Before(confirmacion.SesionValidaHasta) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	revalidador := revalidadorSesionAdministracionCorreoV3{delegado: p.revalidador, alta: alta, confirmacion: confirmacion, reloj: p.reloj}
	vinculo, resultado, err := vecdomain.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador, vecdomain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef}, p.resolutor, vecdomain.SolicitudContextoActor{Cuenta: vecdomain.CuentaAutenticadaContextoActor{CuentaRef: capacidad.identidad.cuentaRef, Metodo: vecdomain.AuthMethodCertificate, Garantia: vecdomain.AuthAssuranceHigh}, PerfilActivoRef: capacidad.identidad.perfilRef}, p.reloj)
	if err != nil || vinculo.ValidarPara(resultado) != nil || resultado.Contexto.PersonaRef != capacidad.identidad.personaRef || resultado.Contexto.Instantanea.CuentaRef != capacidad.identidad.cuentaRef || resultado.Contexto.PerfilActivoRef != capacidad.identidad.perfilRef {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	principal, err := p.permisos.ResolverPrincipalAdministracionCorreoV3(ctx, vinculo, resultado)
	if err != nil || !principalAdministracionCorreoV3Valido(principal, capacidad, ahora) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return contextoSesionAdministracionCorreoV3{Principal: clonarPrincipalDesarrollo(principal), Vinculo: vinculo, Resultado: resultado}, nil
}

// PrincipalAdministracionDesarrollo satisface la frontera HTTP sólo después
// de construir la sesión durable y consultar el permiso gobernado.
func (p *proveedorSesionDurableAdministracionCorreoV3) PrincipalAdministracionDesarrollo(ctx context.Context) (vecdomain.Principal, error) {
	sesion, err := p.ResolverSesionAdministracionCorreoV3(ctx)
	if err != nil {
		return vecdomain.Principal{}, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return clonarPrincipalDesarrollo(sesion.Principal), nil
}

func capacidadAdministracionCorreoV3Valida(capacidad *capacidadAdministracionDesarrollo, ahora time.Time) bool {
	return capacidad != nil && capacidad.identidad != nil && capacidad.identidad.identidad.principal.Validate() == nil && referenciasIdentidadAdministracionDesarrolloValidas(capacidad.identidad) && (capacidad.identidad.identidad.principal.AuthMethod == vecdomain.AuthMethodCertificate || capacidad.identidad.identidad.principal.AuthMethod == vecdomain.AuthMethodDNIe) && capacidad.identidad.identidad.principal.AuthAssurance.Cumple(vecdomain.AuthAssuranceHigh) && !ahora.Before(capacidad.certificadoVerificadoEn) && ahora.Before(capacidad.certificadoValidoHasta)
}

func principalAdministracionCorreoV3Valido(principal vecdomain.Principal, capacidad *capacidadAdministracionDesarrollo, ahora time.Time) bool {
	return capacidad != nil && capacidad.identidad != nil && principal.Validate() == nil && principal.ID == capacidad.identidad.personaRef && principal.AuthMethod == vecdomain.AuthMethodCertificate && principal.AuthAssurance.Cumple(vecdomain.AuthAssuranceHigh) && principal.HasPermission(adminmodule.PermissionIntegrationsManage) && !ahora.Before(capacidad.certificadoVerificadoEn) && ahora.Before(capacidad.certificadoValidoHasta)
}

func altaSesionAdministracionCorreoV3(capacidad *capacidadAdministracionDesarrollo, ahora time.Time) (httpseguridad.AltaSesionAtomica, error) {
	var alta httpseguridad.AltaSesionAtomica
	if capacidad == nil || capacidad.identidad == nil || !ahora.Equal(ahora.UTC().Truncate(time.Microsecond)) || ahora.Before(capacidad.certificadoVerificadoEn) || !ahora.Before(capacidad.certificadoValidoHasta) {
		return alta, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	var asercion, sesion [32]byte
	if _, err := rand.Read(asercion[:]); err != nil {
		return alta, err
	}
	if _, err := rand.Read(sesion[:]); err != nil || asercion == sesion {
		return alta, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	politica := sha256.Sum256([]byte("vec.admin.mtls.dnie.privilegiada.v1"))
	hasta := ahora.Add(2 * time.Minute)
	if capacidad.certificadoValidoHasta.Before(hasta) {
		hasta = capacidad.certificadoValidoHasta
	}
	alta = httpseguridad.AltaSesionAtomica{AsercionID: hex.EncodeToString(asercion[:]), SesionID: hex.EncodeToString(sesion[:]), SujetoID: capacidad.identidad.identidad.principal.ID, CuentaID: capacidad.identidad.cuentaRef, CuentaPrivilegiada: true, Superficie: httpseguridad.SuperficieAdministracionPrivilegiada, EspacioIdentidad: espacioIdentidadSesionDesarrollo, MetodoObservado: vecdomain.AuthMethodCertificate, GarantiaObservada: vecdomain.AuthAssuranceHigh, AutenticacionVerificadaEn: capacidad.certificadoVerificadoEn, SesionEmitidaEn: ahora, AsercionExpiraEn: hasta, PoliticaGarantiaRef: referenciaAltaContratacionTemporalDesarrollo("pga_", "admin-mtls-dnie-v1"), PoliticaGarantiaHuellaSHA256: hex.EncodeToString(politica[:])}
	if alta.Validar() != nil {
		return httpseguridad.AltaSesionAtomica{}, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return alta, nil
}

type revalidadorSesionAdministracionCorreoV3 struct {
	delegado     vecports.RevalidadorAutenticacionActorV1
	alta         httpseguridad.AltaSesionAtomica
	confirmacion httpseguridad.ConfirmacionAltaSesion
	reloj        interface{ Ahora() time.Time }
}

func (r revalidadorSesionAdministracionCorreoV3) RevalidarAutenticacionActorV1(ctx context.Context, s vecdomain.SolicitudRevalidacionAutenticacionActorV1) (vecdomain.AutenticacionRevalidadaV1, error) {
	var cero vecdomain.AutenticacionRevalidadaV1
	if s.AutenticacionRef != r.confirmacion.AutenticacionRef || s.SesionRef != r.confirmacion.SesionRef {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	v, err := r.delegado.RevalidarAutenticacionActorV1(ctx, s)
	if err != nil || v.Validar() != nil || v.AutenticacionRef != r.confirmacion.AutenticacionRef || v.SesionRef != r.confirmacion.SesionRef || v.CuentaRef != r.confirmacion.CuentaRef || v.CuentaOrdinariaRef != r.confirmacion.CuentaOrdinariaRef || !v.CuentaPrivilegiada || v.Superficie != vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1 || v.MetodoObservado != vecdomain.AuthMethodCertificate || !v.GarantiaObservada.Cumple(vecdomain.AuthAssuranceHigh) || !r.reloj.Ahora().Before(v.SesionValidaHasta) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return v, nil
}

type emisorMaterialConfiguracionCorreoV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, vecdomain.SolicitudAutorizacionLigadaV3, vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

var _ emisorMaterialConfiguracionCorreoV3 = (*confianza.EmisorMaterialAutorizacionAtestadaV3)(nil)

// proveedorAutorizacionConfiguracionCorreoV3 ata en una única VEC-AD-3 la
// configuración completa, su CAS, la operación de secreto y la huella del AAD
// del sobre ya cifrado. Nunca recibe ni serializa el secreto claro.
type validadorAuditoriaAdministracionV3 interface {
	ValidarAuditoriaParaSesion(context.Context, vecdomain.AuditEntry, contextoSesionAdministracionCorreoV3, uint64) error
}

type proveedorAutorizacionConfiguracionCorreoV3 struct {
	auditoria   validadorAuditoriaAdministracionV3
	sesion      sesionDurableAdministracionCorreoV3
	emisor      emisorMaterialConfiguracionCorreoV3
	motivo      vecdomain.ReferenciaEntradaCatalogo
	referencias vecports.GeneradorReferenciasAutorizacionV2
	reloj       interface{ Ahora() time.Time }
}

var _ adminports.AutorizadorConfiguracionCorreo = (*proveedorAutorizacionConfiguracionCorreoV3)(nil)

func nuevoProveedorAutorizacionConfiguracionCorreoV3(sesion sesionDurableAdministracionCorreoV3, auditoria validadorAuditoriaAdministracionV3, emisor emisorMaterialConfiguracionCorreoV3, motivo vecdomain.ReferenciaEntradaCatalogo, referencias vecports.GeneradorReferenciasAutorizacionV2, reloj interface{ Ahora() time.Time }) (*proveedorAutorizacionConfiguracionCorreoV3, error) {
	if dependenciaAdministracionAutorizacionNula(auditoria) || dependenciaAdministracionAutorizacionNula(sesion) || dependenciaAdministracionAutorizacionNula(emisor) || motivo.Validar() != nil || dependenciaAdministracionAutorizacionNula(referencias) || dependenciaAdministracionAutorizacionNula(reloj) {
		return nil, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return &proveedorAutorizacionConfiguracionCorreoV3{auditoria: auditoria, sesion: sesion, emisor: emisor, motivo: motivo, referencias: referencias, reloj: reloj}, nil
}

func (p *proveedorAutorizacionConfiguracionCorreoV3) AutorizarConfiguracionCorreo(ctx context.Context, preparacion adminports.PreparacionConfiguracionCorreo, auditoria vecdomain.AuditEntry) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	var cero vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	capacidad, acreditada := capacidadAdministracionDesdeContexto(ctx)
	if !acreditada || capacidad.metodo != "PUT" {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	if p == nil || ctx == nil || ctx.Err() != nil || dependenciaAdministracionAutorizacionNula(p.auditoria) || dependenciaAdministracionAutorizacionNula(p.sesion) || dependenciaAdministracionAutorizacionNula(p.emisor) || dependenciaAdministracionAutorizacionNula(p.referencias) || dependenciaAdministracionAutorizacionNula(p.reloj) || preparacion.Entrada.Validar() != nil || !auditoriaConfiguracionCorreoV3Valida(auditoria) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	payload, err := payloadPreparadoConfiguracionCorreoV3(preparacion, auditoria)
	if err != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	defer borrarBytes(payload)
	h := sha256.Sum256(payload)
	recurso := vecdomain.RecursoAutorizable{Referencia: referenciaConfiguracionCorreoAdministracionV3, ModuloID: adminmodule.ModuleID, Tipo: tipoRecursoConfiguracionCorreoAdministracion, Ambitos: map[string]string{"organizacion_ref": organizacionConfiguracionCorreoAdministracionV3}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if recurso.Validar() != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	sesion, err := p.sesion.ResolverSesionAdministracionCorreoV3(ctx)
	if err != nil || !contextoSesionAdministracionCorreoV3Valido(sesion, p.reloj.Ahora()) || p.auditoria.ValidarAuditoriaParaSesion(ctx, auditoria, sesion, preparacion.Entrada.VersionEsperada+1) != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, p.referencias)
	if err != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	solicitud, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: sesion.Vinculo, ReferenciaMotivo: p.motivo, Accion: accionConfiguracionCorreoAdministracionV3, Recurso: recurso, Finalidad: finalidadConfiguracionCorreoAdministracionV3, Correlacion: correlacion})
	if err != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	_, _, exportador, err := p.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, sesion.Resultado)
	if err != nil || dependenciaAdministracionAutorizacionNula(exportador) || ctx.Err() != nil {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || material.ValidarEstructura() != nil || !materialConfiguracionCorreoV3Exacto(material, recurso, p.reloj.Ahora()) {
		return cero, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return material, nil
}

func contextoSesionAdministracionCorreoV3Valido(c contextoSesionAdministracionCorreoV3, ahora time.Time) bool {
	if c.Principal.Validate() != nil || c.Resultado.Validar() != nil || c.Vinculo.ValidarPara(c.Resultado) != nil || !c.Principal.HasPermission(adminmodule.PermissionIntegrationsManage) || (c.Principal.AuthMethod != vecdomain.AuthMethodCertificate && c.Principal.AuthMethod != vecdomain.AuthMethodDNIe) || !c.Principal.AuthAssurance.Cumple(vecdomain.AuthAssuranceHigh) || !c.Resultado.Contexto.Instantanea.VigenteEn(ahora) {
		return false
	}
	v, err := c.Vinculo.Datos()
	return err == nil && v.PrincipalID == c.Principal.ID && v.CuentaPrivilegiada && v.Superficie == vecdomain.SuperficieAutenticacionAdministracionPrivilegiadaV1 && (v.MetodoObservado == vecdomain.AuthMethodCertificate || v.MetodoObservado == vecdomain.AuthMethodDNIe) && v.GarantiaObservada.Cumple(vecdomain.AuthAssuranceHigh)
}

func auditoriaConfiguracionCorreoV3Valida(a vecdomain.AuditEntry) bool {
	return a.Action == accionConfiguracionCorreoAdministracionV3 && a.ModuleID == adminmodule.ModuleID && a.SubjectRef == referenciaConfiguracionCorreoAdministracionV3 && a.Result == "accepted"
}

func materialConfiguracionCorreoV3Exacto(m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, recurso vecdomain.RecursoAutorizable, ahora time.Time) bool {
	r := m.ResumenCapacidad()
	efecto, err := recurso.HuellaContextoAutorizacionSHA256()
	return err == nil && r.Operacion() == accionConfiguracionCorreoAdministracionV3 && r.EfectoRef() == referenciaConfiguracionCorreoAdministracionV3 && r.EfectoHuellaSHA256() == efecto && r.AudienciaConsumo() == audienciaConfiguracionCorreoAdministracionV3 && !ahora.Before(r.EmitidaEn()) && ahora.Before(r.ExpiraEn())
}

// La representación se limita al estado SMTP y al sobre cifrado. La entrada
// write-only no tiene campo secreto en este valor; la huella AAD se coteja con
// el sobre que el adaptador cifró antes de pedir autorización.
func payloadPreparadoConfiguracionCorreoV3(p adminports.PreparacionConfiguracionCorreo, auditoria vecdomain.AuditEntry) ([]byte, error) {
	if p.Entrada.SecretoNuevo != nil || p.Entrada.Validar() != nil || len(p.PayloadNegocio) == 0 || len(p.PayloadNegocio) > 65536 || !auditoriaConfiguracionCorreoV3Valida(auditoria) {
		return nil, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	esperado, err := adminapp.PayloadNegocioConfiguracionCorreo(p, auditoria)
	if err != nil || !bytes.Equal(esperado, p.PayloadNegocio) {
		return nil, errAutorizacionConfiguracionCorreoAdministracionV3
	}
	return append([]byte(nil), p.PayloadNegocio...), nil
}

func dependenciaAdministracionAutorizacionNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	}
	return false
}
