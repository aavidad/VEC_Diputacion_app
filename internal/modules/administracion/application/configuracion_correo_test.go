package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	admin "vec-diputacion-granada/internal/modules/administracion"
	d "vec-diputacion-granada/internal/modules/administracion/domain"
	p "vec-diputacion-granada/internal/modules/administracion/ports"
	v "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func TestServicioCorreoPreparaAutorizaGuardaEnOrden(t *testing.T) {
	traza := []string{}
	pre := &preCorreo{traza: &traza}
	aud := &audCorreo{traza: &traza}
	aut := &autCorreo{traza: &traza}
	reg := &regCorreo{traza: &traza, vista: vistaCorreo(5)}
	s := servicioCorreo(t, aud, pre, aut, reg)
	if got, err := s.Actualizar(context.Background(), principalCorreo(), actualizacionCorreo(nil, 4)); err != nil || got.Version != 5 || len(traza) != 4 || traza[0] != "auditar" || traza[1] != "preparar" || traza[2] != "autorizar" || traza[3] != "guardar" {
		t.Fatalf("flujo: %#v %v %v", got, err, traza)
	}
	if aud.version != 5 || pre.entrada.Version != 0 || pre.entrada.VersionEsperada != 4 || !reflect.DeepEqual(pre.audit, aud.audit) || !reflect.DeepEqual(aut.audit, aud.audit) || !reflect.DeepEqual(reg.orden.Preparacion, pre.preparacion) {
		t.Fatal("CAS, auditoria o preparacion alterados")
	}
}
func TestServicioCorreoNoAvanzaTrasFalloYConservaConflicto(t *testing.T) {
	for n, mut := range map[string]func(*audCorreo, *preCorreo, *autCorreo, *regCorreo){"auditar": func(x *audCorreo, _ *preCorreo, _ *autCorreo, _ *regCorreo) { x.err = errors.New("x") }, "preparar": func(_ *audCorreo, x *preCorreo, _ *autCorreo, _ *regCorreo) { x.err = errors.New("x") }, "autorizar": func(_ *audCorreo, _ *preCorreo, x *autCorreo, _ *regCorreo) { x.err = errors.New("x") }, "conflicto": func(_ *audCorreo, _ *preCorreo, _ *autCorreo, x *regCorreo) {
		x.err = p.ErrConfiguracionCorreoConflicto
	}} {
		t.Run(n, func(t *testing.T) {
			aud, pre, aut, reg := &audCorreo{}, &preCorreo{}, &autCorreo{}, &regCorreo{vista: vistaCorreo(5)}
			mut(aud, pre, aut, reg)
			_, err := servicioCorreo(t, aud, pre, aut, reg).Actualizar(context.Background(), principalCorreo(), actualizacionCorreo(nil, 4))
			if n == "conflicto" {
				if !errors.Is(err, ErrConfiguracionCorreoConflicto) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrConfiguracionCorreoNoDisponible) {
				t.Fatal(err)
			}
			if n == "auditar" && (pre.n != 0 || aut.n != 0 || reg.n != 0) || n == "preparar" && (aut.n != 0 || reg.n != 0) || n == "autorizar" && reg.n != 0 {
				t.Fatal("flujo avanzo")
			}
		})
	}
}
func TestServicioCorreoDeniegaAntesDePrepararSinAutenticacionAdmitida(t *testing.T) {
	pre, aut, reg := &preCorreo{}, &autCorreo{}, &regCorreo{vista: vistaCorreo(5)}
	principal := principalCorreo()
	principal.AuthMethod = v.AuthMethodSSO
	if _, err := servicioCorreo(t, &audCorreo{}, pre, aut, reg).Actualizar(context.Background(), principal, actualizacionCorreo(nil, 4)); !errors.Is(err, ErrAccesoConfiguracionCorreoDenegado) || pre.n != 0 || aut.n != 0 || reg.n != 0 {
		t.Fatalf("acceso no denegado antes del flujo: %v", err)
	}
}

func TestServicioCorreoRechazaAuditoriaNoLigadaAntesDePreparar(t *testing.T) {
	aud, pre, aut, reg := &audCorreo{}, &preCorreo{}, &autCorreo{}, &regCorreo{vista: vistaCorreo(5)}
	aud.mutar = func(a *v.AuditEntry) { a.ObjectVersion++ }
	_, err := servicioCorreo(t, aud, pre, aut, reg).Actualizar(context.Background(), principalCorreo(), actualizacionCorreo(nil, 4))
	if !errors.Is(err, ErrConfiguracionCorreoNoDisponible) || aud.n != 1 || pre.n != 0 || aut.n != 0 || reg.n != 0 {
		t.Fatalf("auditoria no ligada avanzo el flujo: %v, %d/%d/%d/%d", err, aud.n, pre.n, aut.n, reg.n)
	}
}

type audCorreo struct {
	n       int
	version uint64
	audit   v.AuditEntry
	err     error
	traza   *[]string
	mutar   func(*v.AuditEntry)
}

func (x *audCorreo) PrepararAuditoriaConfiguracionCorreo(_ context.Context, principal v.Principal, version uint64) (v.AuditEntry, error) {
	x.n++
	x.version = version
	if x.traza != nil {
		*x.traza = append(*x.traza, "auditar")
	}
	if x.err != nil {
		return v.AuditEntry{}, x.err
	}
	x.audit = v.AuditEntry{ActorID: "hmac-sha256:bolsa_registro_accesos_t13:0000000000000000000000000000000000000000000000000000000000000001", ActorProfile: "perfil:admin", ActorRoles: append([]string(nil), principal.Roles...), AuthMethod: v.AuthMethodCertificate, AuthAssurance: v.AuthAssuranceHigh, Purpose: finalidadActualizarConfiguracionCorreo, Action: AccionActualizarConfiguracionCorreo, ModuleID: admin.ModuleID, SubjectRef: "configuracion:smtp:diputacion", ObjectVersion: int(version), Result: "accepted", CorrelationRef: "correlacion_00000000000000000000000000000001", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}
	if x.mutar != nil {
		x.mutar(&x.audit)
	}
	return x.audit, nil
}

type accesoCorreo struct{}

func (*accesoCorreo) VerificarAccesoConfiguracionCorreo(context.Context, v.Principal) error {
	return nil
}

type preCorreo struct {
	n           int
	entrada     d.ActualizacionConfiguracionCorreo
	audit       v.AuditEntry
	preparacion p.PreparacionConfiguracionCorreo
	err         error
	traza       *[]string
}

func (x *preCorreo) PrepararConfiguracionCorreo(_ context.Context, e d.ActualizacionConfiguracionCorreo, a v.AuditEntry) (p.PreparacionConfiguracionCorreo, error) {
	x.n++
	x.entrada, x.audit = e, a
	if x.traza != nil {
		*x.traza = append(*x.traza, "preparar")
	}
	if x.err != nil {
		return p.PreparacionConfiguracionCorreo{}, x.err
	}
	x.preparacion = p.PreparacionConfiguracionCorreo{Entrada: e, PayloadNegocio: []byte("p")}
	return x.preparacion, nil
}

type autCorreo struct {
	n     int
	err   error
	traza *[]string
	audit v.AuditEntry
}

func (x *autCorreo) AutorizarConfiguracionCorreo(_ context.Context, _ p.PreparacionConfiguracionCorreo, auditoria v.AuditEntry) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	x.n++
	x.audit = auditoria
	if x.traza != nil {
		*x.traza = append(*x.traza, "autorizar")
	}
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, x.err
}

type regCorreo struct {
	n     int
	orden p.OrdenConfiguracionCorreoAutorizada
	vista d.VistaConfiguracionCorreo
	err   error
	traza *[]string
}

func (x *regCorreo) LeerConfiguracionCorreo(context.Context) (d.VistaConfiguracionCorreo, error) {
	return x.vista, x.err
}
func (x *regCorreo) GuardarConfiguracionCorreo(_ context.Context, o p.OrdenConfiguracionCorreoAutorizada) (d.VistaConfiguracionCorreo, error) {
	x.n++
	x.orden = o
	if x.traza != nil {
		*x.traza = append(*x.traza, "guardar")
	}
	return x.vista, x.err
}
func servicioCorreo(t *testing.T, aud p.PreparadorAuditoriaConfiguracionCorreo, pre p.PreparadorConfiguracionCorreo, aut p.AutorizadorConfiguracionCorreo, reg p.RegistroConfiguracionCorreo) *ServicioConfiguracionCorreo {
	t.Helper()
	s, e := NuevoServicioConfiguracionCorreo(&accesoCorreo{}, aud, pre, aut, reg)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func principalCorreo() v.Principal {
	return v.Principal{ID: "admin", Roles: []string{"rol:administracion"}, Permissions: []string{admin.PermissionIntegrationsManage}, AuthMethod: v.AuthMethodCertificate, AuthAssurance: v.AuthAssuranceHigh}
}
func actualizacionCorreo(s *d.SecretoCorreo, cas uint64) d.ActualizacionConfiguracionCorreo {
	return d.ActualizacionConfiguracionCorreo{VistaConfiguracionCorreo: d.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh", ModoTLS: d.ModoTLSCorreoImplicito, ModoAutenticacion: d.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 1, SecretoConfigurado: true}, VersionEsperada: cas, SecretoNuevo: s}
}
func vistaCorreo(ver uint64) d.VistaConfiguracionCorreo {
	x := actualizacionCorreo(nil, 4).VistaConfiguracionCorreo
	x.Version = ver
	return x
}
