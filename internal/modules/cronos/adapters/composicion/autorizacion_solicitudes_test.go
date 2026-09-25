package composicion

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/application"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func TestProveedoresSolicitudesEmitenContratoExactoYSoloParaLaPersona(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	id := identidadPrueba(t, empleado)
	emisor := &emisorPrueba{err: errors.New("no disponible")}
	a, _ := NuevoAutorizadorCronos(emisor, motivosPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: id}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	actor := id.Contexto.Contexto
	ctx := context.Background()

	movs, err := r.ResolverConsultaMovimientosPropios(req)
	if err != nil {
		t.Fatal(err)
	}
	mm := domain.MaterialConsultaMovimientosPropios{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Desde: "2026-01-01", Hasta: "2026-12-31", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = movs.ProveedorMaterial().ProveerMaterialConsultaMovimientosPropios(ctx, mm)
	permisos, err := r.ResolverPermisosPropios(req)
	if err != nil {
		t.Fatal(err)
	}
	mp := domain.MaterialConsultaPermisosPropios{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Anio: 2026, ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = permisos.ProveedorMaterial().ProveerMaterialConsultaPermisosPropios(ctx, mp)
	ms := domain.MaterialSolicitudPermisoPropio{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, ClaveOperacion: "perm-clave-0001", PermisoRef: "permiso:cronos:asuntos-propios", Desde: "2026-10-05", Hasta: "2026-10-06", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = permisos.ProveedorMaterial().ProveerMaterialSolicitudPermisoPropio(ctx, ms)
	correccion, err := r.ResolverSolicitudCorreccionPropia(req)
	if err != nil {
		t.Fatal(err)
	}
	mc := domain.MaterialAutorizacionCorreccion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, SolicitudRef: "correccion:cronos:corr-clave-0001",
		ClaveOperacion: "corr-clave-0001", Paso: domain.PasoSolicitudCorreccion, ComandoSHA256: strings.Repeat("a", 64), InstanteUTC: time.Now().UTC().Truncate(time.Microsecond)}
	_, _ = correccion.ProveedorMaterial().ProveerMaterialCorreccion(ctx, mc)
	if len(emisor.solicitudes) != 4 {
		t.Fatal("no emite una decisión por acción", len(emisor.solicitudes))
	}
	esperado := []struct{ accion, finalidad, referencia, tipo string }{
		{application.AccionConsultarMovimientosPropios, application.FinalidadConsultarMovimientosPropios, "movimientos:cronos:" + empleado, "movimientos_propio"},
		{application.AccionConsultarPermisosPropios, application.FinalidadConsultarPermisosPropios, "permisos:cronos:" + empleado, "permisos_propio"},
		{application.AccionSolicitarPermisoPropio, application.FinalidadSolicitarPermisoPropio, "permiso:cronos:solicitud:perm-clave-0001", "solicitud_permiso"},
		{application.AccionSolicitarCorreccion, application.FinalidadSolicitarCorreccion, "correccion:cronos:corr-clave-0001", "correccion_marcaje"},
	}
	for i, e := range esperado {
		d := emisor.solicitudes[i]
		if d.Accion != e.accion || d.Finalidad != e.finalidad || d.Recurso.Referencia != e.referencia || d.Recurso.Tipo != e.tipo || d.Recurso.Ambitos["empleado_ref"] != empleado {
			t.Fatalf("contrato %d distinto: %+v", i, d)
		}
	}
	if emisor.solicitudes[3].Recurso.Atributos["material_sha256"] != strings.Repeat("a", 64) {
		t.Fatal("la corrección no se liga a la huella del material")
	}
	// Ni otra persona ni un paso que no sea la solicitud propia.
	ajeno := mm
	ajeno.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	if _, err := movs.ProveedorMaterial().ProveerMaterialConsultaMovimientosPropios(ctx, ajeno); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("emite movimientos de otro empleado", err)
	}
	decision := mc
	decision.Paso = domain.PasoDecisionResponsable
	if _, err := correccion.ProveedorMaterial().ProveerMaterialCorreccion(ctx, decision); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("emite una decisión de jefatura desde la persona empleada", err)
	}
	otroPerfil := ms
	otroPerfil.PerfilRef = "prf_" + strings.Repeat("q", 24)
	if _, err := permisos.ProveedorMaterial().ProveerMaterialSolicitudPermisoPropio(ctx, otroPerfil); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 4 {
		t.Fatal("emite para otro perfil", err)
	}
}

func TestResolutorSolicitudesExigeEmpleado(t *testing.T) {
	a, _ := NuevoAutorizadorCronos(&emisorPrueba{}, motivosPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: identidadPrueba(t)}, a, canalPrueba(t))
	req := httptest.NewRequest("GET", "/", nil)
	if _, err := r.ResolverConsultaMovimientosPropios(req); !errors.Is(err, ports.ErrEmpleadoNoAcreditado) {
		t.Fatal(err)
	}
	if _, err := r.ResolverPermisosPropios(req); !errors.Is(err, ports.ErrEmpleadoNoAcreditado) {
		t.Fatal(err)
	}
	if _, err := r.ResolverSolicitudCorreccionPropia(req); !errors.Is(err, ports.ErrEmpleadoNoAcreditado) {
		t.Fatal(err)
	}
}
