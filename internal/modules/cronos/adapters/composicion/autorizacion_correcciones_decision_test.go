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
)

func TestProveedorCorreccionC5bEmiteContratoNominal(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	id := identidadPrueba(t, empleado)
	emisor := &emisorPrueba{err: errors.New("emisor de prueba")}
	a, err := NuevoAutorizadorCronos(emisor, motivosPrueba())
	if err != nil {
		t.Fatal(err)
	}
	r, err := NuevoResolutorPeticionCronos(identidadFija{id: id}, a, canalPrueba(t))
	if err != nil {
		t.Fatal(err)
	}
	orden, err := r.ResolverDecisionCorrecciones(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	proveedor, ok := orden.ProveedorMaterial().(ports.ProveedorMaterialDecisionCorreccion)
	if !ok {
		t.Fatal("proveedor C5b ausente")
	}
	actor := id.Contexto.Contexto
	b := domain.MaterialBandejaCorrecciones{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado, Paso: "responsable", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = proveedor.ProveerMaterialBandejaCorrecciones(context.Background(), b)
	m := domain.MaterialActuacionCorreccion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		SolicitudRef: "correccion:cronos:solicitud-0001", ClaveOperacion: "actuacion-0001", Paso: domain.PasoDecisionResponsable,
		Resultado: domain.ResultadoFavorable, Motivo: "Olvido acreditado", VersionEsperada: 1, ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = proveedor.ProveerMaterialActuacionCorreccion(context.Background(), m)
	h := domain.MaterialHistoriaCorreccion{ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		SolicitudRef: m.SolicitudRef, Vista: "propia", ZonaHoraria: domain.ZonaSaldoPeninsula}
	_, _ = proveedor.ProveerMaterialHistoriaCorreccion(context.Background(), h)
	h.Vista, h.Paso = "equipo", "responsable"
	_, _ = proveedor.ProveerMaterialHistoriaCorreccion(context.Background(), h)
	if len(emisor.solicitudes) != 4 || emisor.solicitudes[0].Accion != application.AccionBandejaCorrecciones ||
		emisor.solicitudes[1].Accion != application.AccionDecidirCorreccion ||
		emisor.solicitudes[1].Finalidad != application.FinalidadDecisionResponsableCorreccion ||
		emisor.solicitudes[2].Accion != application.AccionHistoriaCorreccionPropia ||
		emisor.solicitudes[3].Accion != application.AccionHistoriaCorreccionEquipo {
		t.Fatalf("emisión C5b divergente: %+v", emisor.solicitudes)
	}
	m.EmpleadoRef = "emp_" + strings.Repeat("z", 24)
	_, err = proveedor.ProveerMaterialActuacionCorreccion(context.Background(), m)
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 4 {
		t.Fatalf("emite para empleado elegido por cliente: %v", err)
	}
}
