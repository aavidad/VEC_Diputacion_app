package application

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestConsultaCorreoPreparaAutorizaRegistraYRevalidaAntesDeExponer(t *testing.T) {
	acceso := &accesoConsultaCorreo{}
	preparador := &preparadorConsultaCorreo{}
	autorizador := &autorizadorConsultaCorreo{}
	registro := &registroConsultaCorreo{}
	servicio := servicioConsultaCorreo(t, acceso, preparador, autorizador, registro)
	resultado, err := servicio.Consultar(context.Background(), principalConsultaCorreo())
	if err != nil || resultado.Vista.Version != 7 || acceso.llamadas != 2 || preparador.llamadas != 1 || autorizador.llamadas != 1 || registro.llamadas != 1 {
		t.Fatalf("flujo de consulta: %#v %v; %d/%d/%d/%d", resultado, err, acceso.llamadas, preparador.llamadas, autorizador.llamadas, registro.llamadas)
	}
	if !reflect.DeepEqual(autorizador.preparacion, registro.orden.Preparacion) || !reflect.DeepEqual(preparador.auditoria, autorizador.preparacion.Auditoria) || resultado.ReciboAuditoria.ID == "" {
		t.Fatal("auditoria, payload y orden no quedaron ligados")
	}
	var payload struct {
		Esquema string       `json:"esquema"`
		Audit   v.AuditEntry `json:"auditoria"`
	}
	if json.Unmarshal(autorizador.preparacion.PayloadNegocio, &payload) != nil || payload.Esquema != "vec.administracion.configuracion-correo.consulta.v1" || !reflect.DeepEqual(payload.Audit, preparador.auditoria) || bytes.Contains(autorizador.preparacion.PayloadNegocio, []byte("secreto")) {
		t.Fatal("payload de consulta no es canónico o expone un secreto")
	}
}

func TestConsultaCorreoFallaCerradoSinAvanzarTrasCadaFrontera(t *testing.T) {
	casos := map[string]func(*accesoConsultaCorreo, *preparadorConsultaCorreo, *autorizadorConsultaCorreo, *registroConsultaCorreo){
		"acceso": func(a *accesoConsultaCorreo, _ *preparadorConsultaCorreo, _ *autorizadorConsultaCorreo, _ *registroConsultaCorreo) {
			a.err = errors.New("denegada")
		},
		"preparacion": func(_ *accesoConsultaCorreo, p *preparadorConsultaCorreo, _ *autorizadorConsultaCorreo, _ *registroConsultaCorreo) {
			p.err = errors.New("audit no disponible")
		},
		"v3": func(_ *accesoConsultaCorreo, _ *preparadorConsultaCorreo, a *autorizadorConsultaCorreo, _ *registroConsultaCorreo) {
			a.err = errors.New("v3 no disponible")
		},
		"registro": func(_ *accesoConsultaCorreo, _ *preparadorConsultaCorreo, _ *autorizadorConsultaCorreo, r *registroConsultaCorreo) {
			r.err = errors.New("commit no confirmado")
		},
	}
	for nombre, preparar := range casos {
		t.Run(nombre, func(t *testing.T) {
			acceso, auditoria, autorizador, registro := &accesoConsultaCorreo{}, &preparadorConsultaCorreo{}, &autorizadorConsultaCorreo{}, &registroConsultaCorreo{}
			preparar(acceso, auditoria, autorizador, registro)
			_, err := servicioConsultaCorreo(t, acceso, auditoria, autorizador, registro).Consultar(context.Background(), principalConsultaCorreo())
			if nombre == "acceso" {
				if !errors.Is(err, ErrAccesoConfiguracionCorreoDenegado) || auditoria.llamadas != 0 || autorizador.llamadas != 0 || registro.llamadas != 0 {
					t.Fatalf("acceso avanzó: %v", err)
				}
				return
			}
			if !errors.Is(err, ErrConsultaConfiguracionCorreoNoDisponible) {
				t.Fatal(err)
			}
			if nombre == "preparacion" && (autorizador.llamadas != 0 || registro.llamadas != 0) || nombre == "v3" && registro.llamadas != 0 {
				t.Fatalf("%s avanzó después del error", nombre)
			}
		})
	}
}

func TestConsultaCorreoNoExponeTrasRevocacionOPayloadNoMinimizado(t *testing.T) {
	t.Run("revocacion", func(t *testing.T) {
		acceso := &accesoConsultaCorreo{fallarDesde: 2}
		_, err := servicioConsultaCorreo(t, acceso, &preparadorConsultaCorreo{}, &autorizadorConsultaCorreo{}, &registroConsultaCorreo{}).Consultar(context.Background(), principalConsultaCorreo())
		if !errors.Is(err, ErrAccesoConfiguracionCorreoDenegado) {
			t.Fatal(err)
		}
	})
	t.Run("cancelacion durante preparacion", func(t *testing.T) {
		ctx, cancelar := context.WithCancel(context.Background())
		defer cancelar()
		preparador := &preparadorConsultaCorreo{despues: cancelar}
		autorizador, registro := &autorizadorConsultaCorreo{}, &registroConsultaCorreo{}
		_, err := servicioConsultaCorreo(t, &accesoConsultaCorreo{}, preparador, autorizador, registro).Consultar(ctx, principalConsultaCorreo())
		if !errors.Is(err, ErrConsultaConfiguracionCorreoNoDisponible) || autorizador.llamadas != 0 || registro.llamadas != 0 {
			t.Fatalf("cancelacion expuso o avanzó consulta: %v", err)
		}
	})
	t.Run("auditoria con version previa", func(t *testing.T) {
		preparador := &preparadorConsultaCorreo{mutar: func(a *v.AuditEntry) { a.ObjectVersion = 1 }}
		autorizador, registro := &autorizadorConsultaCorreo{}, &registroConsultaCorreo{}
		_, err := servicioConsultaCorreo(t, &accesoConsultaCorreo{}, preparador, autorizador, registro).Consultar(context.Background(), principalConsultaCorreo())
		if !errors.Is(err, ErrConsultaConfiguracionCorreoNoDisponible) || autorizador.llamadas != 0 || registro.llamadas != 0 {
			t.Fatalf("auditoria previa fue autorizada: %v", err)
		}
	})
}

type accesoConsultaCorreo struct {
	llamadas    int
	fallarDesde int
	err         error
}

func (a *accesoConsultaCorreo) VerificarAccesoConfiguracionCorreo(context.Context, v.Principal) error {
	a.llamadas++
	if a.err != nil || a.fallarDesde != 0 && a.llamadas >= a.fallarDesde {
		return errors.New("acceso no vigente")
	}
	return nil
}

type preparadorConsultaCorreo struct {
	llamadas  int
	auditoria v.AuditEntry
	err       error
	mutar     func(*v.AuditEntry)
	despues   func()
}

func (p *preparadorConsultaCorreo) PrepararAuditoriaConsultaConfiguracionCorreo(_ context.Context, principal v.Principal) (v.AuditEntry, error) {
	p.llamadas++
	if p.err != nil {
		return v.AuditEntry{}, p.err
	}
	p.auditoria = auditoriaConsultaCorreoPrueba(principal)
	if p.mutar != nil {
		p.mutar(&p.auditoria)
	}
	if p.despues != nil {
		p.despues()
	}
	return p.auditoria, nil
}

type autorizadorConsultaCorreo struct {
	llamadas    int
	preparacion p.PreparacionConsultaConfiguracionCorreo
	err         error
}

func (a *autorizadorConsultaCorreo) AutorizarConsultaConfiguracionCorreo(_ context.Context, preparacion p.PreparacionConsultaConfiguracionCorreo) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	a.preparacion = preparacion
	a.preparacion.PayloadNegocio = append([]byte(nil), preparacion.PayloadNegocio...)
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, a.err
}

type registroConsultaCorreo struct {
	llamadas int
	orden    p.OrdenConsultaConfiguracionCorreoAutorizada
	err      error
}

func (r *registroConsultaCorreo) ConsultarConfiguracionCorreoAuditada(_ context.Context, orden p.OrdenConsultaConfiguracionCorreoAutorizada) (p.ResultadoConsultaConfiguracionCorreo, error) {
	r.llamadas++
	r.orden = orden
	r.orden.Preparacion.PayloadNegocio = append([]byte(nil), orden.Preparacion.PayloadNegocio...)
	if r.err != nil {
		return p.ResultadoConsultaConfiguracionCorreo{}, r.err
	}
	recibo := orden.Preparacion.Auditoria
	recibo.ID, recibo.Seq, recibo.Signature, recibo.AuthorizationRef, recibo.ObjectVersion = "acc_abc", 1, "firma_t13", "decision:consulta", 7
	return p.ResultadoConsultaConfiguracionCorreo{Vista: vistaConsultaCorreo(), ReciboAuditoria: recibo}, nil
}

func servicioConsultaCorreo(t *testing.T, acceso p.VerificadorAccesoConfiguracionCorreo, preparador p.PreparadorAuditoriaConsultaConfiguracionCorreo, autorizador p.AutorizadorConsultaConfiguracionCorreo, registro p.RegistroConsultaConfiguracionCorreo) *ServicioConsultaConfiguracionCorreo {
	t.Helper()
	servicio, err := NuevoServicioConsultaConfiguracionCorreo(acceso, preparador, autorizador, registro)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func principalConsultaCorreo() v.Principal {
	return v.Principal{ID: "persona:nominal", Roles: []string{"rol:administracion"}, Permissions: []string{admin.PermissionIntegrationsManage}, AuthMethod: v.AuthMethodCertificate, AuthAssurance: v.AuthAssuranceHigh}
}

func auditoriaConsultaCorreoPrueba(principal v.Principal) v.AuditEntry {
	return v.AuditEntry{ActorID: "hmac-sha256:bolsa_registro_accesos_t13:0000000000000000000000000000000000000000000000000000000000000001", ActorProfile: "perfil:administracion", ActorRoles: append([]string(nil), principal.Roles...), AuthMethod: v.AuthMethodCertificate, AuthAssurance: v.AuthAssuranceHigh, Purpose: finalidadActualizarConfiguracionCorreo, Action: AccionConsultarConfiguracionCorreo, ModuleID: admin.ModuleID, SubjectRef: "configuracion:smtp:diputacion", Result: "permitido", CorrelationRef: "correlacion_00000000000000000000000000000001", OccurredAt: time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}
}

func vistaConsultaCorreo() d.VistaConfiguracionCorreo {
	return d.VistaConfiguracionCorreo{Configurada: true, Host: "smtp.intranet.local", Puerto: 465, NombreServidor: "smtp.intranet.local", ReferenciaCA: "ca:v1", RemitenteFijo: "rrhh@diputacion.example", Usuario: "rrhh", ModoTLS: d.ModoTLSCorreoImplicito, ModoAutenticacion: d.ModoAutenticacionCorreoXOAUTH2, TiempoMaximoMillis: 1, SecretoConfigurado: true, Version: 7}
}
