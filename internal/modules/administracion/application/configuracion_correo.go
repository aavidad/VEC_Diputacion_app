package application

import (
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"sort"
	"strings"
	"time"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	admindomain "vec-diputacion-granada/internal/modules/administracion/domain"
	adminports "vec-diputacion-granada/internal/modules/administracion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

var ErrAccesoConfiguracionCorreoDenegado = errors.New("administracion: acceso a configuracion de correo denegado")

var (
	ErrConfiguracionCorreoNoDisponible = errors.New("administracion: configuracion de correo no disponible")
	ErrConfiguracionCorreoConflicto    = errors.New("administracion: conflicto de configuracion de correo")
)

const AccionActualizarConfiguracionCorreo = "administracion.configuracion_correo.actualizar"

const finalidadActualizarConfiguracionCorreo = "administrar_integraciones"

const maximaVersionAuditoriaConfiguracionCorreo = uint64(9007199254740991)

type ServicioConfiguracionCorreo struct {
	acceso      adminports.VerificadorAccesoConfiguracionCorreo
	auditoria   adminports.PreparadorAuditoriaConfiguracionCorreo
	preparador  adminports.PreparadorConfiguracionCorreo
	autorizador adminports.AutorizadorConfiguracionCorreo
	registro    adminports.RegistroConfiguracionCorreo
	ahora       func() time.Time
}

func NuevoServicioConfiguracionCorreo(acceso adminports.VerificadorAccesoConfiguracionCorreo, auditoria adminports.PreparadorAuditoriaConfiguracionCorreo, preparador adminports.PreparadorConfiguracionCorreo, autorizador adminports.AutorizadorConfiguracionCorreo, registro adminports.RegistroConfiguracionCorreo) (*ServicioConfiguracionCorreo, error) {
	if dependenciaNula(acceso) || dependenciaNula(auditoria) || dependenciaNula(preparador) || dependenciaNula(autorizador) || dependenciaNula(registro) {
		return nil, ErrAccesoConfiguracionCorreoDenegado
	}
	return &ServicioConfiguracionCorreo{acceso: acceso, auditoria: auditoria, preparador: preparador, autorizador: autorizador, registro: registro, ahora: time.Now}, nil
}

func (s *ServicioConfiguracionCorreo) Consultar(ctx context.Context, principal vecdomain.Principal) (admindomain.VistaConfiguracionCorreo, error) {
	if err := s.autorizar(ctx, principal); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, err
	}
	vista, err := s.registro.LeerConfiguracionCorreo(ctx)
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, errorRegistroCorreo(err)
	}
	// La lectura puede bloquear mientras la sesión o el permiso se revocan. No
	// se expone una configuración obtenida con una autorización ya caducada.
	if err := s.autorizar(ctx, principal); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, err
	}
	if vista.Validar() != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrAccesoConfiguracionCorreoDenegado
	}
	return vista, nil
}

func (s *ServicioConfiguracionCorreo) Actualizar(ctx context.Context, principal vecdomain.Principal, entrada admindomain.ActualizacionConfiguracionCorreo) (admindomain.VistaConfiguracionCorreo, error) {
	if err := s.autorizar(ctx, principal); err != nil || entrada.Validar() != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrAccesoConfiguracionCorreoDenegado
	}
	if entrada.VersionEsperada >= maximaVersionAuditoriaConfiguracionCorreo {
		return admindomain.VistaConfiguracionCorreo{}, ErrAccesoConfiguracionCorreoDenegado
	}
	versionNueva := entrada.VersionEsperada + 1
	auditoria, err := s.auditoria.PrepararAuditoriaConfiguracionCorreo(ctx, principal, versionNueva)
	if err != nil || ctx.Err() != nil || !auditoriaConfiguracionCorreoValida(auditoria, principal, versionNueva) {
		return admindomain.VistaConfiguracionCorreo{}, errorRegistroCorreo(err)
	}
	preparacion, err := s.preparador.PrepararConfiguracionCorreo(ctx, entrada, auditoria)
	if err != nil || ctx.Err() != nil {
		return admindomain.VistaConfiguracionCorreo{}, errorRegistroCorreo(err)
	}
	// Preparar cifra y compone la preimagen, por lo que puede requerir I/O. La
	// decisión V3 sólo se pide si la misma capacidad sigue vigente.
	if err := s.autorizar(ctx, principal); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, err
	}
	material, err := s.autorizador.AutorizarConfiguracionCorreo(ctx, preparacion, auditoria)
	if err != nil || ctx.Err() != nil {
		return admindomain.VistaConfiguracionCorreo{}, errorRegistroCorreo(err)
	}
	// La emisión V3 también puede requerir I/O. Antes de consumir su material
	// y modificar el estado durable se revalida la misma frontera. Una vez que
	// Guardar confirma el commit, no se hace una comprobación posterior.
	if err := s.autorizar(ctx, principal); err != nil {
		return admindomain.VistaConfiguracionCorreo{}, err
	}
	vista, err := s.registro.GuardarConfiguracionCorreo(ctx, adminports.OrdenConfiguracionCorreoAutorizada{Preparacion: preparacion, Material: material})
	if err != nil {
		return admindomain.VistaConfiguracionCorreo{}, errorRegistroCorreo(err)
	}
	if vista.Validar() != nil {
		return admindomain.VistaConfiguracionCorreo{}, ErrAccesoConfiguracionCorreoDenegado
	}
	return vista, nil
}

func (s *ServicioConfiguracionCorreo) autorizar(ctx context.Context, principal vecdomain.Principal) error {
	if s == nil || dependenciaNula(s.acceso) || dependenciaNula(s.auditoria) || dependenciaNula(s.preparador) || dependenciaNula(s.autorizador) || dependenciaNula(s.registro) || ctx == nil || ctx.Err() != nil || principal.Validate() != nil || !principal.HasPermission(adminmodule.PermissionIntegrationsManage) ||
		(principal.AuthMethod != vecdomain.AuthMethodCertificate && principal.AuthMethod != vecdomain.AuthMethodDNIe) || !principal.AuthAssurance.Cumple(vecdomain.AuthAssuranceHigh) || s.acceso.VerificarAccesoConfiguracionCorreo(ctx, principal) != nil {
		return ErrAccesoConfiguracionCorreoDenegado
	}
	return nil
}

func auditoriaConfiguracionCorreoValida(auditoria vecdomain.AuditEntry, principal vecdomain.Principal, versionNueva uint64) bool {
	roles := append([]string(nil), principal.Roles...)
	sort.Strings(roles)
	if versionNueva == 0 || versionNueva > maximaVersionAuditoriaConfiguracionCorreo || auditoria.ID != "" || auditoria.Seq != 0 || auditoria.Signature != "" || auditoria.RepresentedSubjectID != "" || auditoria.AuthorizationRef != "" || auditoria.ExpedienteRef != "" || auditoria.DocumentRef != "" || auditoria.RuleRef != "" || auditoria.Reason != "" || auditoria.BeforeHash != "" || auditoria.AfterHash != "" || len(auditoria.Metadata) != 0 || auditoria.IntegrityAlgorithm != "" || auditoria.PrevSignature != "" || !actorAuditoriaConfiguracionCorreoValido(auditoria.ActorID) || auditoria.ActorProfile == "" || len(auditoria.ActorProfile) > 160 || auditoria.Purpose != finalidadActualizarConfiguracionCorreo || auditoria.Action != AccionActualizarConfiguracionCorreo || auditoria.ModuleID != adminmodule.ModuleID || auditoria.SubjectRef != "configuracion:smtp:diputacion" || auditoria.Result != "accepted" || auditoria.ObjectVersion != int(versionNueva) || !correlacionAuditoriaConfiguracionCorreoValida(auditoria.CorrelationRef) || auditoria.OccurredAt.IsZero() || auditoria.OccurredAt.Location() != time.UTC || !auditoria.OccurredAt.Equal(auditoria.OccurredAt.Truncate(time.Microsecond)) || (auditoria.AuthMethod != vecdomain.AuthMethodCertificate && auditoria.AuthMethod != vecdomain.AuthMethodDNIe) || auditoria.AuthAssurance != vecdomain.AuthAssuranceHigh || principal.AuthMethod != auditoria.AuthMethod || principal.AuthAssurance != auditoria.AuthAssurance || len(auditoria.ActorRoles) == 0 || len(auditoria.ActorRoles) > 16 || !sort.StringsAreSorted(auditoria.ActorRoles) {
		return false
	}
	if len(auditoria.ActorRoles) != len(roles) || !reflect.DeepEqual(auditoria.ActorRoles, roles) {
		return false
	}
	for _, rol := range auditoria.ActorRoles {
		if rol == "" || len(rol) > 128 {
			return false
		}
	}
	return true
}

func actorAuditoriaConfiguracionCorreoValido(actor string) bool {
	partes := strings.Split(actor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] != "" && len(partes[2]) == 64 && hexadecimalNoNulo(partes[2])
}

func correlacionAuditoriaConfiguracionCorreoValida(correlacion string) bool {
	const prefijo = "correlacion_"
	return strings.HasPrefix(correlacion, prefijo) && len(correlacion) == len(prefijo)+32 && hexadecimalNoNulo(correlacion[len(prefijo):])
}

func hexadecimalNoNulo(valor string) bool {
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

func errorRegistroCorreo(err error) error {
	if errors.Is(err, adminports.ErrConfiguracionCorreoConflicto) {
		return ErrConfiguracionCorreoConflicto
	}
	return ErrConfiguracionCorreoNoDisponible
}

func dependenciaNula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
