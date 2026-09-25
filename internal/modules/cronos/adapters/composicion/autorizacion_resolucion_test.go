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

func motivosResolucionPrueba() MotivosCronos {
	m := motivosPrueba()
	r := vecdomain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_cronos", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("7", 32)}
	m.Bandeja, m.Resolucion, m.Avisos, m.ArchivoAviso = r, r, r, r
	return m
}

func TestMotivosResolucionVanJuntos(t *testing.T) {
	a, err := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	if err != nil || a.ResolucionConfigurada() {
		t.Fatal("sin motivos de resolución la capacidad debe quedar apagada", err)
	}
	a, err = NuevoAutorizadorCronos(&emisorPrueba{}, motivosResolucionPrueba())
	if err != nil || !a.ResolucionConfigurada() {
		t.Fatal("con los cuatro motivos la capacidad se compone", err)
	}
	parcial := motivosResolucionPrueba()
	parcial.ArchivoAviso = vecdomain.ReferenciaEntradaCatalogo{}
	if _, err := NuevoAutorizadorCronos(&emisorPrueba{}, parcial); err == nil {
		t.Fatal("configuración parcial aceptada")
	}
	invalido := motivosResolucionPrueba()
	invalido.Bandeja.EntradaClave = "x"
	if _, err := NuevoAutorizadorCronos(&emisorPrueba{}, invalido); err == nil {
		t.Fatal("motivo inválido aceptado")
	}
}

func TestResolucionSinMotivosNoEmiteOrden(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	a, _ := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: identidadPrueba(t, empleado)}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	if _, err := r.ResolverResolucionPermisos(req); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
	if _, err := r.ResolverAvisosPropios(req); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal(err)
	}
}

func TestProveedoresResolucionEmitenContratoExacto(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	id := identidadPrueba(t, empleado)
	emisor := &emisorPrueba{err: errors.New("no disponible")}
	a, _ := NuevoAutorizadorCronos(emisor, motivosResolucionPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: id}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	actor := id.Contexto.Contexto
	ctx := context.Background()

	res, err := r.ResolverResolucionPermisos(req)
	if err != nil {
		t.Fatal(err)
	}
	mb := domain.MaterialBandejaPermisos{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Paso: domain.PasoResponsable, ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = res.ProveedorMaterial().ProveerMaterialBandejaPermisos(ctx, mb)
	mr := domain.MaterialResolucionPermiso{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "res-clave-0001",
		SolicitudRef: "permiso:cronos:solicitud:perm-clave-0001", Paso: domain.PasoAdministracion, Decision: domain.DecisionAprobar, VersionEsperada: 2, ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = res.ProveedorMaterial().ProveerMaterialResolucionPermiso(ctx, mr)
	avisos, err := r.ResolverAvisosPropios(req)
	if err != nil {
		t.Fatal(err)
	}
	ma := domain.MaterialConsultaAvisosPropios{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = avisos.ProveedorMaterial().ProveerMaterialConsultaAvisosPropios(ctx, ma)
	mx := domain.MaterialArchivoAvisoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "arch-clave-0001",
		AvisoRef: "aviso:cronos:00000000-0000-4000-8000-000000000001", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = avisos.ProveedorMaterial().ProveerMaterialArchivoAvisoPropio(ctx, mx)
	if len(emisor.solicitudes) != 4 {
		t.Fatal("no emite una decisión por acción", len(emisor.solicitudes))
	}
	esperado := []struct{ accion, finalidad, referencia, tipo string }{
		{application.AccionConsultarBandeja, application.FinalidadConsultarBandeja, "bandeja:cronos:permisos:responsable", "bandeja_permisos"},
		{application.AccionResolverPermiso, application.FinalidadResolverPermiso, "permiso:cronos:resolucion:res-clave-0001", "resolucion_permiso"},
		{application.AccionConsultarAvisosPropios, application.FinalidadConsultarAvisosPropios, "avisos:cronos:" + empleado, "avisos_propio"},
		{application.AccionArchivarAvisoPropio, application.FinalidadArchivarAvisoPropio, "aviso:cronos:archivo:arch-clave-0001", "archivo_aviso"},
	}
	for i, e := range esperado {
		d := emisor.solicitudes[i]
		if d.Accion != e.accion || d.Finalidad != e.finalidad || d.Recurso.Referencia != e.referencia || d.Recurso.Tipo != e.tipo {
			t.Fatalf("contrato %d distinto: %+v", i, d)
		}
	}
	// Quien resuelve se autoriza sobre su persona y el paso, nunca sobre el
	// empleado de la solicitud; la persona, sobre su propio empleado.
	for i, paso := range []string{"responsable", "administracion"} {
		amb := emisor.solicitudes[i].Recurso.Ambitos
		if len(amb) != 2 || amb["persona_ref"] != actor.PersonaRef || amb["paso_resolucion"] != paso {
			t.Fatalf("ámbitos de quien resuelve distintos: %+v", amb)
		}
	}
	for _, d := range emisor.solicitudes[2:] {
		if len(d.Recurso.Ambitos) != 1 || d.Recurso.Ambitos["empleado_ref"] != empleado {
			t.Fatalf("ámbitos de la persona distintos: %+v", d.Recurso.Ambitos)
		}
	}
	ajeno := mr
	ajeno.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	if _, err := res.ProveedorMaterial().ProveerMaterialResolucionPermiso(ctx, ajeno); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("emite con un empleado propio distinto del de la sesión", err)
	}
	otroActor := mx
	otroActor.ActorRef = "per_" + strings.Repeat("q", 24)
	if _, err := avisos.ProveedorMaterial().ProveerMaterialArchivoAvisoPropio(ctx, otroActor); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 4 {
		t.Fatal("emite para otra persona", err)
	}
}

func TestResolutorResolucionExigeEmpleado(t *testing.T) {
	a, _ := NuevoAutorizadorCronos(&emisorPrueba{}, motivosResolucionPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: identidadPrueba(t)}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	if _, err := r.ResolverResolucionPermisos(req); !errors.Is(err, ports.ErrEmpleadoNoAcreditado) {
		t.Fatal(err)
	}
	if _, err := r.ResolverAvisosPropios(req); !errors.Is(err, ports.ErrEmpleadoNoAcreditado) {
		t.Fatal(err)
	}
}
