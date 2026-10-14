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

func TestRecuperacionCorreccionEmiteAccionPropiaYSoloParaEmpleadoAcreditado(t *testing.T) {
	empleado := "emp_" + strings.Repeat("e", 24)
	id := identidadPrueba(t, empleado)
	emisor := &emisorPrueba{err: errors.New("fallo de prueba")}
	a, _ := NuevoAutorizadorCronos(emisor, motivosPrueba())
	r, _ := NuevoResolutorPeticionCronos(identidadFija{id: id}, a, canalPrueba(t))
	orden, err := r.ResolverSolicitudCorreccionPropia(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	proveedor, ok := orden.ProveedorMaterial().(ports.ProveedorMaterialRecuperacionCorreccionPropia)
	if !ok {
		t.Fatal("falta autoridad nominal de recuperación")
	}
	actor := id.Contexto.Contexto
	m := domain.MaterialRecuperacionCorreccionPropia{
		ActorRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, EmpleadoRef: empleado,
		SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001",
	}
	_, _ = proveedor.ProveerMaterialRecuperacionCorreccionPropia(context.Background(), m)
	if len(emisor.solicitudes) != 1 || emisor.solicitudes[0].Accion != application.AccionRecuperarReciboCorreccion ||
		emisor.solicitudes[0].Finalidad != application.FinalidadRecuperarReciboCorreccionPropia ||
		emisor.solicitudes[0].Recurso.Tipo != "recibo_correccion_propia" ||
		emisor.solicitudes[0].Recurso.Referencia != m.SolicitudRef {
		t.Fatal("recuperación usó otra autoridad", emisor.solicitudes)
	}
	ajena := m
	ajena.EmpleadoRef = "emp_" + strings.Repeat("x", 24)
	if _, err := proveedor.ProveerMaterialRecuperacionCorreccionPropia(context.Background(), ajena); !errors.Is(err, ports.ErrDependenciaNoDisponible) || len(emisor.solicitudes) != 1 {
		t.Fatal("emitió sobre empleado ajeno", err)
	}
}
