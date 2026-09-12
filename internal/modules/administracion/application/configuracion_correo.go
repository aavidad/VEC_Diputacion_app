package application

import (
	"context"
	"errors"
	"reflect"
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

type ServicioConfiguracionCorreo struct {
	acceso      adminports.VerificadorAccesoConfiguracionCorreo
	preparador  adminports.PreparadorConfiguracionCorreo
	autorizador adminports.AutorizadorConfiguracionCorreo
	registro    adminports.RegistroConfiguracionCorreo
	ahora       func() time.Time
}

func NuevoServicioConfiguracionCorreo(acceso adminports.VerificadorAccesoConfiguracionCorreo, preparador adminports.PreparadorConfiguracionCorreo, autorizador adminports.AutorizadorConfiguracionCorreo, registro adminports.RegistroConfiguracionCorreo) (*ServicioConfiguracionCorreo, error) {
	if dependenciaNula(acceso) || dependenciaNula(preparador) || dependenciaNula(autorizador) || dependenciaNula(registro) {
		return nil, ErrAccesoConfiguracionCorreoDenegado
	}
	return &ServicioConfiguracionCorreo{acceso: acceso, preparador: preparador, autorizador: autorizador, registro: registro, ahora: time.Now}, nil
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
	ahora := s.ahora().UTC()
	auditoria := vecdomain.AuditEntry{ActorID: principal.ID, ActorRoles: append([]string(nil), principal.Roles...), Action: AccionActualizarConfiguracionCorreo, ModuleID: adminmodule.ModuleID, SubjectRef: "configuracion:smtp:diputacion", Result: "accepted", CorrelationRef: "configuracion_correo", OccurredAt: ahora}
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
	if s == nil || dependenciaNula(s.acceso) || dependenciaNula(s.preparador) || dependenciaNula(s.autorizador) || dependenciaNula(s.registro) || ctx == nil || ctx.Err() != nil || principal.Validate() != nil || !principal.HasPermission(adminmodule.PermissionIntegrationsManage) ||
		(principal.AuthMethod != vecdomain.AuthMethodCertificate && principal.AuthMethod != vecdomain.AuthMethodDNIe) || !principal.AuthAssurance.Cumple(vecdomain.AuthAssuranceHigh) || s.acceso.VerificarAccesoConfiguracionCorreo(ctx, principal) != nil {
		return ErrAccesoConfiguracionCorreoDenegado
	}
	return nil
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
