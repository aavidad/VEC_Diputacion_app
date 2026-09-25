package composicion

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

func motivosNotificacionesPrueba() MotivosCronos {
	m := motivosPrueba()
	r := vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("8", 32)}
	m.Notificacion, m.Notificaciones, m.BandejaNotificaciones, m.AtencionNotificacion = r, r, r, r
	return m
}

func TestMotivosNotificacionesVanJuntosYSonIndependientesDeLaResolucion(t *testing.T) {
	a, err := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	if err != nil || a.NotificacionesConfiguradas() {
		t.Fatal("sin motivos las notificaciones deben quedar apagadas", err)
	}
	a, err = NuevoAutorizadorCronos(&emisorPrueba{}, motivosNotificacionesPrueba())
	if err != nil || !a.NotificacionesConfiguradas() || a.ResolucionConfigurada() {
		t.Fatal("con los cuatro motivos se componen solas", err)
	}
	parcial := motivosNotificacionesPrueba()
	parcial.AtencionNotificacion = vecdomain.ReferenciaEntradaCatalogo{}
	if _, err := NuevoAutorizadorCronos(&emisorPrueba{}, parcial); err == nil {
		t.Fatal("configuración parcial aceptada")
	}
	sinMotivos, _ := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: identidadPrueba(t, "emp_"+strings.Repeat("e", 24))}, sinMotivos, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	if _, err := r.ResolverNotificacionesPropias(req); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := r.ResolverBandejaNotificaciones(req); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
}

func TestProveedoresNotificacionesEmitenContratoExacto(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	id := identidadPrueba(t, empleado)
	emisor := &emisorPrueba{err: errors.New("no disponible")}
	a, _ := NuevoAutorizadorCronos(emisor, motivosNotificacionesPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: id}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	actor := id.Contexto.Contexto
	ctx := context.Background()
	propias, err := r.ResolverNotificacionesPropias(req)
	if err != nil {
		t.Fatal(err)
	}
	mr := domain.MaterialRegistroNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "not-clave-0001",
		TipoVersionRef: "notificacion:cronos:tipo:otra-comunicacion:sintetico-1", FechaReferida: "2026-09-24", Texto: "Texto", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = propias.ProveedorMaterial().ProveerMaterialRegistroNotificacion(ctx, mr)
	mc := domain.MaterialConsultaNotificaciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = propias.ProveedorMaterial().ProveerMaterialConsultaNotificacionesPropias(ctx, mc)
	bandeja, err := r.ResolverBandejaNotificaciones(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = bandeja.ProveedorMaterial().ProveerMaterialBandejaNotificaciones(ctx, mc)
	ma := domain.MaterialAtencionNotificacion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "ate-clave-0001",
		NotificacionRef: "notificacion:cronos:0f0e0d0c-0b0a-4000-8000-000000000001", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = bandeja.ProveedorMaterial().ProveerMaterialAtencionNotificacion(ctx, ma)
	if len(emisor.solicitudes) != 4 {
		t.Fatal("no emite una decisión por acción", len(emisor.solicitudes))
	}
	esperado := []struct{ accion, finalidad, referencia, tipo string }{
		{application.AccionRegistrarNotificacion, application.FinalidadRegistrarNotificacion, "notificacion:cronos:not-clave-0001", "notificacion_propia"},
		{application.AccionConsultarNotificacionesPropias, application.FinalidadConsultarNotificacionesPropias, "notificaciones:cronos:" + empleado, "notificaciones_propio"},
		{application.AccionConsultarBandejaNotif, application.FinalidadConsultarBandejaNotif, "bandeja:cronos:notificaciones", "bandeja_notificaciones"},
		{application.AccionAtenderNotificacion, application.FinalidadAtenderNotificacion, "notificacion:cronos:atencion:ate-clave-0001", "atencion_notificacion"},
	}
	for i, e := range esperado {
		d := emisor.solicitudes[i]
		if d.Accion != e.accion || d.Finalidad != e.finalidad || d.Recurso.Referencia != e.referencia || d.Recurso.Tipo != e.tipo {
			t.Fatalf("contrato %d distinto: %+v", i, d)
		}
	}
	for _, d := range emisor.solicitudes[:2] {
		if len(d.Recurso.Ambitos) != 1 || d.Recurso.Ambitos["empleado_ref"] != empleado {
			t.Fatalf("ámbitos de la persona distintos: %+v", d.Recurso.Ambitos)
		}
	}
	for _, d := range emisor.solicitudes[2:] {
		if len(d.Recurso.Ambitos) != 2 || d.Recurso.Ambitos["persona_ref"] != actor.PersonaRef || d.Recurso.Ambitos["paso_resolucion"] != "administracion" {
			t.Fatalf("ámbitos de RRHH distintos: %+v", d.Recurso.Ambitos)
		}
	}
	otro := mr
	otro.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	if _, err := propias.ProveedorMaterial().ProveerMaterialRegistroNotificacion(ctx, otro); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 4 {
		t.Fatal("emite para otro empleado", err)
	}
}
