package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func alcanceEmpleadoPrueba(t *testing.T) domain.AlcanceProyeccionesContextoActor {
	t.Helper()
	alcance, err := domain.NuevoAlcanceProyeccionesContextoActor(domain.ProyeccionContextoActorEmpleado)
	if err != nil {
		t.Fatal(err)
	}
	return alcance
}

// contextoConEmpleadoDePersonalPrueba sustituye el puntero heredado vin_ de
// empleado por la entrada pep_ que aporta la proyeccion de Personal.
func contextoConEmpleadoDePersonalPrueba(t *testing.T, instante time.Time, solicitud domain.SolicitudContextoActor) domain.ContextoActor {
	t.Helper()
	instantanea := instantaneaServicioContextoActorPrueba(instante, solicitud)
	instantanea.Vinculos[1].VinculoRef = referenciaServicioContextoActorPrueba("pep_", "e")
	return contextoActorPruebaMutador{solicitud: solicitud, instantanea: instantanea, instante: instante}.crear(t)
}

func servicioAlcancePrueba(
	t *testing.T,
	alcance domain.AlcanceProyeccionesContextoActor,
	resolutor *resolutorRegistroContextoActorV2Prueba,
	generador *generadorOperacionContextoActorV2Prueba,
	instantes ...time.Time,
) *ServicioContextoActor {
	t.Helper()
	servicio, err := NuevoServicioContextoActorProductivoV2ConAlcance(
		resolutor, generador, &relojSecuenciaContextoActorPrueba{instantes: instantes}, alcance,
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio
}

func TestServicioContextoActorPideElAlcanceDeLaComposicion(t *testing.T) {
	solicitadoEn := instanteServicioContextoActorPrueba()
	resuelto := solicitadoEn.Add(time.Millisecond)
	solicitud := solicitudServicioContextoActorPrueba()
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacionRegistroContextoActorV2Prueba(
		t, contextoConEmpleadoDePersonalPrueba(t, resuelto, solicitud), generador.ref,
	)}
	servicio := servicioAlcancePrueba(t, alcanceEmpleadoPrueba(t), resolutor, generador, solicitadoEn, resuelto)
	resultado, err := servicio.ResolverRegistrado(context.Background(), solicitud)
	if err != nil || !resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
		t.Fatalf("contexto con empleado de Personal rechazado: %v", err)
	}
	if _, recibida := resolutor.observacion(); !recibida.Proyecciones.IncluyeEmpleado() {
		t.Fatal("el servicio no pidio el alcance de su composicion")
	}

	// La composicion heredada (CT y resto) pide el alcance vacio.
	heredado, err := NuevoServicioContextoActorProductivoV2(resolutor, generador,
		&relojSecuenciaContextoActorPrueba{instantes: []time.Time{solicitadoEn, resuelto}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := heredado.ResolverRegistrado(context.Background(), solicitud); err == nil {
		t.Fatal("un recibo con empleado de Personal se acepto sin pedirlo")
	}
	if _, recibida := resolutor.observacion(); !recibida.Proyecciones.Vacio() {
		t.Fatal("la composicion heredada amplio el alcance")
	}
}

func TestServicioContextoActorRechazaReciboSinElAlcancePedido(t *testing.T) {
	solicitadoEn := instanteServicioContextoActorPrueba()
	resuelto := solicitadoEn.Add(time.Millisecond)
	solicitud := solicitudServicioContextoActorPrueba()
	generador := nuevoGeneradorOperacionContextoActorV2Prueba()
	// El puntero heredado vin_ de empleado no satisface {empleado}.
	resolutor := &resolutorRegistroContextoActorV2Prueba{resultado: confirmacionRegistroContextoActorV2Prueba(
		t, contextoActorServicioPrueba(t, resuelto, solicitud), generador.ref,
	)}
	servicio := servicioAlcancePrueba(t, alcanceEmpleadoPrueba(t), resolutor, generador, solicitadoEn, resuelto)
	if _, err := servicio.ResolverRegistrado(context.Background(), solicitud); !errors.Is(err, domain.ErrContextoActorNoResuelto) {
		t.Fatalf("recibo sin proyeccion de Personal aceptado: %v", err)
	}
}

func TestServicioContextoActorConservaMotivoDeProyeccionPedida(t *testing.T) {
	for _, motivo := range []error{
		ports.ErrProyeccionEmpleadoContextoActorAusente,
		ports.ErrProyeccionEmpleadoContextoActorAmbigua,
	} {
		solicitadoEn := instanteServicioContextoActorPrueba()
		resolutor := &resolutorRegistroContextoActorV2Prueba{
			error: errors.Join(ports.ErrResolutorRegistroContextoActorNoDisponible, motivo),
		}
		servicio := servicioAlcancePrueba(t, alcanceEmpleadoPrueba(t), resolutor,
			nuevoGeneradorOperacionContextoActorV2Prueba(), solicitadoEn)
		_, err := servicio.ResolverRegistrado(context.Background(), solicitudServicioContextoActorPrueba())
		if !errors.Is(err, motivo) || !errors.Is(err, domain.ErrContextoActorNoResuelto) {
			t.Fatalf("motivo perdido: %v", err)
		}
		// Sin alcance pedido, el mismo error vuelve a ser opaco.
		heredado := servicioAlcancePrueba(t, domain.AlcanceProyeccionesContextoActor{}, resolutor,
			nuevoGeneradorOperacionContextoActorV2Prueba(), solicitadoEn)
		_, err = heredado.ResolverRegistrado(context.Background(), solicitudServicioContextoActorPrueba())
		if errors.Is(err, motivo) || !errors.Is(err, ports.ErrResolutorRegistroContextoActorNoDisponible) {
			t.Fatalf("motivo filtrado sin alcance pedido: %v", err)
		}
	}
}
