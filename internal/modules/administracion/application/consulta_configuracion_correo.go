package application

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const AccionConsultarConfiguracionCorreo = "administracion.configuracion_correo.consultar"

var ErrConsultaConfiguracionCorreoNoDisponible = errors.New("administracion: consulta de configuracion de correo no disponible")

type ServicioConsultaConfiguracionCorreo struct {
	acceso      adminports.VerificadorAccesoConfiguracionCorreo
	preparador  adminports.PreparadorAuditoriaConsultaConfiguracionCorreo
	autorizador adminports.AutorizadorConsultaConfiguracionCorreo
	registro    adminports.RegistroConsultaConfiguracionCorreo
}

func NuevoServicioConsultaConfiguracionCorreo(acceso adminports.VerificadorAccesoConfiguracionCorreo, preparador adminports.PreparadorAuditoriaConsultaConfiguracionCorreo, autorizador adminports.AutorizadorConsultaConfiguracionCorreo, registro adminports.RegistroConsultaConfiguracionCorreo) (*ServicioConsultaConfiguracionCorreo, error) {
	if dependenciaNula(acceso) || dependenciaNula(preparador) || dependenciaNula(autorizador) || dependenciaNula(registro) {
		return nil, ErrAccesoConfiguracionCorreoDenegado
	}
	return &ServicioConsultaConfiguracionCorreo{acceso: acceso, preparador: preparador, autorizador: autorizador, registro: registro}, nil
}

func (s *ServicioConsultaConfiguracionCorreo) Consultar(ctx context.Context, principal vecdomain.Principal) (adminports.ResultadoConsultaConfiguracionCorreo, error) {
	if err := s.autorizar(ctx, principal); err != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, err
	}
	auditoria, err := s.preparador.PrepararAuditoriaConsultaConfiguracionCorreo(ctx, principal)
	if err != nil || ctx.Err() != nil || !auditoriaConsultaConfiguracionCorreoValida(auditoria, principal) {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrConsultaConfiguracionCorreoNoDisponible
	}
	payload, err := PayloadConsultaConfiguracionCorreo(auditoria)
	if err != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrConsultaConfiguracionCorreoNoDisponible
	}
	defer borrarPayloadConsultaConfiguracionCorreo(payload)
	preparacion := adminports.PreparacionConsultaConfiguracionCorreo{Auditoria: auditoria, PayloadNegocio: payload}
	material, err := s.autorizador.AutorizarConsultaConfiguracionCorreo(ctx, preparacion)
	if err != nil || ctx.Err() != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrConsultaConfiguracionCorreoNoDisponible
	}
	resultado, err := s.registro.ConsultarConfiguracionCorreoAuditada(ctx, adminports.OrdenConsultaConfiguracionCorreoAutorizada{Preparacion: preparacion, Material: material})
	if err != nil || ctx.Err() != nil {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrConsultaConfiguracionCorreoNoDisponible
	}
	// La transacción ya confirmó lectura, consumo y T13. La autorización se
	// revalida antes de que cualquier campo de la vista llegue al llamador.
	if err := s.autorizar(ctx, principal); err != nil || resultado.Vista.Validar() != nil || !reciboConsultaConfiguracionCorreoValido(resultado.ReciboAuditoria, auditoria, resultado.Vista.Version) {
		return adminports.ResultadoConsultaConfiguracionCorreo{}, ErrAccesoConfiguracionCorreoDenegado
	}
	return resultado, nil
}

func (s *ServicioConsultaConfiguracionCorreo) autorizar(ctx context.Context, principal vecdomain.Principal) error {
	if s == nil || dependenciaNula(s.acceso) || dependenciaNula(s.preparador) || dependenciaNula(s.autorizador) || dependenciaNula(s.registro) || ctx == nil || ctx.Err() != nil || principal.Validate() != nil || !principal.HasPermission(adminmodule.PermissionIntegrationsManage) || (principal.AuthMethod != vecdomain.AuthMethodCertificate && principal.AuthMethod != vecdomain.AuthMethodDNIe) || !principal.AuthAssurance.Cumple(vecdomain.AuthAssuranceHigh) || s.acceso.VerificarAccesoConfiguracionCorreo(ctx, principal) != nil {
		return ErrAccesoConfiguracionCorreoDenegado
	}
	return nil
}

func PayloadConsultaConfiguracionCorreo(auditoria vecdomain.AuditEntry) ([]byte, error) {
	if !auditoriaConsultaConfiguracionCorreoValidaSinPrincipal(auditoria) {
		return nil, errors.New("auditoria de consulta invalida")
	}
	return json.Marshal(struct {
		Esquema   string               `json:"esquema"`
		Auditoria vecdomain.AuditEntry `json:"auditoria"`
	}{"vec.administracion.configuracion-correo.consulta.v1", auditoria})
}

func auditoriaConsultaConfiguracionCorreoValida(auditoria vecdomain.AuditEntry, principal vecdomain.Principal) bool {
	roles := append([]string(nil), principal.Roles...)
	sort.Strings(roles)
	return auditoriaConsultaConfiguracionCorreoValidaSinPrincipal(auditoria) && principal.AuthMethod == auditoria.AuthMethod && principal.AuthAssurance == auditoria.AuthAssurance && reflect.DeepEqual(roles, auditoria.ActorRoles)
}

func auditoriaConsultaConfiguracionCorreoValidaSinPrincipal(auditoria vecdomain.AuditEntry) bool {
	if auditoria.ID != "" || auditoria.Seq != 0 || auditoria.Signature != "" || auditoria.RepresentedSubjectID != "" || auditoria.AuthorizationRef != "" || auditoria.ObjectVersion != 0 || auditoria.ExpedienteRef != "" || auditoria.DocumentRef != "" || auditoria.RuleRef != "" || auditoria.Reason != "" || auditoria.BeforeHash != "" || auditoria.AfterHash != "" || len(auditoria.Metadata) != 0 || auditoria.IntegrityAlgorithm != "" || auditoria.PrevSignature != "" || !actorConsultaConfiguracionCorreoValido(auditoria.ActorID) || auditoria.ActorProfile == "" || len(auditoria.ActorProfile) > 160 || len(auditoria.ActorRoles) == 0 || len(auditoria.ActorRoles) > 16 || !sort.StringsAreSorted(auditoria.ActorRoles) || (auditoria.AuthMethod != vecdomain.AuthMethodCertificate && auditoria.AuthMethod != vecdomain.AuthMethodDNIe) || auditoria.AuthAssurance != vecdomain.AuthAssuranceHigh || auditoria.Purpose != finalidadActualizarConfiguracionCorreo || auditoria.Action != AccionConsultarConfiguracionCorreo || auditoria.ModuleID != adminmodule.ModuleID || auditoria.SubjectRef != "configuracion:smtp:diputacion" || auditoria.Result != "permitido" || !correlacionConsultaConfiguracionCorreoValida(auditoria.CorrelationRef) || auditoria.OccurredAt.IsZero() || auditoria.OccurredAt.Location() != time.UTC || !auditoria.OccurredAt.Equal(auditoria.OccurredAt.Truncate(time.Microsecond)) {
		return false
	}
	for _, rol := range auditoria.ActorRoles {
		if rol == "" || len(rol) > 128 {
			return false
		}
	}
	return true
}

func reciboConsultaConfiguracionCorreoValido(recibo, preparada vecdomain.AuditEntry, versionVista uint64) bool {
	return recibo.ID != "" && recibo.Seq > 0 && recibo.Signature != "" && recibo.AuthorizationRef != "" && recibo.ObjectVersion >= 0 && uint64(recibo.ObjectVersion) == versionVista && recibo.ActorID == preparada.ActorID && recibo.ActorProfile == preparada.ActorProfile && reflect.DeepEqual(recibo.ActorRoles, preparada.ActorRoles) && recibo.AuthMethod == preparada.AuthMethod && recibo.AuthAssurance == preparada.AuthAssurance && recibo.Purpose == preparada.Purpose && recibo.Action == preparada.Action && recibo.ModuleID == preparada.ModuleID && recibo.SubjectRef == preparada.SubjectRef && recibo.Result == preparada.Result && recibo.CorrelationRef == preparada.CorrelationRef && !recibo.OccurredAt.IsZero()
}

func actorConsultaConfiguracionCorreoValido(actor string) bool {
	partes := strings.Split(actor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] != "" && len(partes[2]) == 64 && hexadecimalConsultaNoNulo(partes[2])
}

func correlacionConsultaConfiguracionCorreoValida(correlacion string) bool {
	const prefijo = "correlacion_"
	return strings.HasPrefix(correlacion, prefijo) && len(correlacion) == len(prefijo)+32 && hexadecimalConsultaNoNulo(correlacion[len(prefijo):])
}

func hexadecimalConsultaNoNulo(valor string) bool {
	decodificado, err := hex.DecodeString(valor)
	if err != nil {
		return false
	}
	for _, b := range decodificado {
		if b != 0 {
			return true
		}
	}
	return false
}

func borrarPayloadConsultaConfiguracionCorreo(payload []byte) {
	for i := range payload {
		payload[i] = 0
	}
}
