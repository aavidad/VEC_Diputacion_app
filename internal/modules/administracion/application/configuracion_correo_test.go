package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	admin "vec-diputacion-granada/internal/modules/administracion"
	d "vec-diputacion-granada/internal/modules/administracion/domain"
	p "vec-diputacion-granada/internal/modules/administracion/ports"
	v "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

func TestServicioCorreoPreparaAutorizaGuardaEnOrden(t *testing.T) {
	traza := []string{}
	pre := &preCorreo{traza: &traza}
	aut := &autCorreo{traza: &traza}
	reg := &regCorreo{traza: &traza, vista: vistaCorreo(5)}
	s := servicioCorreo(t, pre, aut, reg)
	if got, err := s.Actualizar(context.Background(), principalCorreo(), actualizacionCorreo(nil, 4)); err != nil || got.Version != 5 || len(traza) != 3 || traza[0] != "preparar" || traza[1] != "autorizar" || traza[2] != "guardar" {
		t.Fatalf("flujo: %#v %v %v", got, err, traza)
	}
	if pre.entrada.Version != 0 || pre.entrada.VersionEsperada != 4 || pre.audit.Action != AccionActualizarConfiguracionCorreo || !reflect.DeepEqual(reg.orden.Preparacion, pre.preparacion) {
		t.Fatal("CAS, auditoria o preparacion alterados")
	}
}
func TestServicioCorreoNoAvanzaTrasFalloYConservaConflicto(t *testing.T) {
	for n, mut := range map[string]func(*preCorreo, *autCorreo, *regCorreo){"preparar": func(x *preCorreo, _ *autCorreo, _ *regCorreo) { x.err = errors.New("x") }, "autorizar": func(_ *preCorreo, x *autCorreo, _ *regCorreo) { x.err = errors.New("x") }, "conflicto": func(_ *preCorreo, _ *autCorreo, x *regCorreo) { x.err = p.ErrConfiguracionCorreoConflicto }} {
		t.Run(n, func(t *testing.T) {
			pre, aut, reg := &preCorreo{}, &autCorreo{}, &regCorreo{vista: vistaCorreo(5)}
			mut(pre, aut, reg)
			_, err := servicioCorreo(t, pre, aut, reg).Actualizar(context.Background(), principalCorreo(), actualizacionCorreo(nil, 4))
			if n == "conflicto" {
				if !errors.Is(err, ErrConfiguracionCorreoConflicto) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, ErrConfiguracionCorreoNoDisponible) {
				t.Fatal(err)
			}
			if n == "preparar" && (aut.n != 0 || reg.n != 0) || n == "autorizar" && reg.n != 0 {
				t.Fatal("flujo avanzo")
			}
		})
	}
}
func TestServicioCorreoDeniegaAntesDePrepararSinAutenticacionAdmitida(t *testing.T) {
	pre, aut, reg := &preCorreo{}, &autCorreo{}, &regCorreo{vista: vistaCorreo(5)}
	principal := principalCorreo()
	principal.AuthMethod = v.AuthMethodSSO
	if _, err := servicioCorreo(t, pre, aut, reg).Actualizar(context.Background(), principal, actualizacionCorreo(nil, 4)); !errors.Is(err, ErrAccesoConfiguracionCorreoDenegado) || pre.n != 0 || aut.n != 0 || reg.n != 0 {
		t.Fatalf("acceso no denegado antes del flujo: %v", err)
	}
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
}

func (x *autCorreo) AutorizarConfiguracionCorreo(context.Context, p.PreparacionConfiguracionCorreo, v.AuditEntry) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	x.n++
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
func servicioCorreo(t *testing.T, pre p.PreparadorConfiguracionCorreo, aut p.AutorizadorConfiguracionCorreo, reg p.RegistroConfiguracionCorreo) *ServicioConfiguracionCorreo {
	t.Helper()
	s, e := NuevoServicioConfiguracionCorreo(&accesoCorreo{}, pre, aut, reg)
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func principalCorreo() v.Principal {
	return v.Principal{ID: "admin", Permissions: []string{admin.PermissionIntegrationsManage}, AuthMethod: v.AuthMethodCertificate, AuthAssurance: v.AuthAssuranceHigh}
}
func actualizacionCorreo(s *d.SecretoCorreo, cas uint64) d.ActualizacionConfiguracionCorreo {
	return d.ActualizacionConfiguracionCorreo{VistaConfiguracionCorreo: d.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh", ModoTLS: d.ModoTLSCorreoImplicito, ModoAutenticacion: d.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 1, SecretoConfigurado: true}, VersionEsperada: cas, SecretoNuevo: s}
}
func vistaCorreo(ver uint64) d.VistaConfiguracionCorreo {
	x := actualizacionCorreo(nil, 4).VistaConfiguracionCorreo
	x.Version = ver
	return x
}
