package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorCorreccionVacio struct{}

func (proveedorCorreccionVacio) ProveerMaterialCorreccion(context.Context, domain.MaterialAutorizacionCorreccion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrDependenciaNoDisponible
}

type repositorioCorreccionPrueba struct {
	solicitudes     int
	actuaciones     int
	recuperaciones  int
	ultimaSolicitud domain.SolicitudCorreccion
	ultimaActuacion domain.ActuacionCorreccion
	err             error
	recuperado      *ports.ReciboCorreccion
}

func (r *repositorioCorreccionPrueba) SolicitarOlvido(_ context.Context, solicitud domain.SolicitudCorreccion, _ ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	r.solicitudes++
	r.ultimaSolicitud = solicitud
	if r.err != nil {
		return ports.ReciboCorreccion{}, r.err
	}
	return ports.ReciboCorreccion{SolicitudRef: "correccion:cronos:" + solicitud.ClaveOperacion, ActuacionRef: "correccion:actuacion:1", ReciboRef: "recibo:cronos:1", Estado: domain.CorreccionPendienteResponsable, Version: 1, InstanteUTC: solicitud.SolicitadaEnUTC}, nil
}

func (r *repositorioCorreccionPrueba) RegistrarActuacion(_ context.Context, actuacion domain.ActuacionCorreccion, _ ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	r.actuaciones++
	r.ultimaActuacion = actuacion
	if r.err != nil {
		return ports.ReciboCorreccion{}, r.err
	}
	return ports.ReciboCorreccion{SolicitudRef: actuacion.SolicitudRef, ActuacionRef: "correccion:actuacion:2", ReciboRef: "recibo:cronos:2", Estado: estadoResultanteCorreccion(actuacion.Paso, actuacion.Resultado), Version: actuacion.VersionEsperada + 1, InstanteUTC: actuacion.RegistradaEnUTC}, nil
}

func (r *repositorioCorreccionPrueba) RecuperarRecibo(_ context.Context, clave ports.ClaveRecuperacionCorreccion, _ ports.OrdenConsumoCorreccion) (ports.ReciboCorreccion, error) {
	r.recuperaciones++
	if r.err != nil {
		return ports.ReciboCorreccion{}, r.err
	}
	if r.recuperado != nil {
		return *r.recuperado, nil
	}
	return ports.ReciboCorreccion{SolicitudRef: clave.SolicitudRef, ActuacionRef: "correccion:actuacion:1", ReciboRef: "recibo:cronos:1", Estado: domain.CorreccionPendienteResponsable, Version: 1, InstanteUTC: time.Now().UTC().Truncate(time.Microsecond), Replay: true}, nil
}

func TestCorreccionRecuperaSoloReciboDelPasoPedido(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	repo := &repositorioCorreccionPrueba{}
	servicio, _ := NuevoServicioCorrecciones(repo, relojMarcajePrueba{instante})
	clave := ports.ClaveRecuperacionCorreccion{SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001", Paso: domain.PasoSolicitudCorreccion}
	recibo, err := servicio.RecuperarRecibo(context.Background(), ordenCorreccionPrueba(t), clave)
	if err != nil || !recibo.Replay || repo.recuperaciones != 1 {
		t.Fatal("recibo original no recuperado", recibo, err)
	}
	forjado := recibo
	forjado.Estado = domain.CorreccionAplicada
	repo.recuperado = &forjado
	_, err = servicio.RecuperarRecibo(context.Background(), ordenCorreccionPrueba(t), clave)
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("recibo de otro paso aceptado", err)
	}
}

func ordenCorreccionPrueba(t *testing.T) ports.OrdenConsumoCorreccion {
	t.Helper()
	actor, err := contexto(t).OrdenConsumo.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	orden, err := ports.NuevaOrdenConsumoCorreccion(actor, proveedorCorreccionVacio{})
	if err != nil {
		t.Fatal(err)
	}
	return orden
}

func TestCorreccionSolicitaHuecoPropioSinModificarOriginal(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	repo := &repositorioCorreccionPrueba{}
	servicio, _ := NuevoServicioCorrecciones(repo, relojMarcajePrueba{instante})
	recibo, err := servicio.SolicitarOlvido(context.Background(), ordenCorreccionPrueba(t), ports.SolicitudOlvidoMarcaje{
		ClaveOperacion: "olvido_0001", HuecoDeclarado: true, Movimiento: domain.PunchEntry,
		FechaCivil: "2026-09-24", HoraPretendida: "08:15",
	})
	if err != nil || recibo.Estado != domain.CorreccionPendienteResponsable || repo.solicitudes != 1 ||
		repo.ultimaSolicitud.EmpleadoRef != "emp_0123456789abcdefghijkl" || repo.ultimaSolicitud.MotivoCodigo != domain.MotivoOlvidoMarcaje ||
		!repo.ultimaSolicitud.SolicitadaEnUTC.Equal(instante) {
		t.Fatal("solicitud propia no preparada", recibo, err)
	}
	_, err = servicio.SolicitarOlvido(context.Background(), ordenCorreccionPrueba(t), ports.SolicitudOlvidoMarcaje{
		ClaveOperacion: "olvido_0002", HuecoDeclarado: true, MarcajeOriginalRef: "marcaje:cronos:otro_0001",
		Movimiento: domain.PunchEntry, FechaCivil: "2026-09-24", HoraPretendida: "08:15",
	})
	if !errors.Is(err, domain.ErrCorreccionInvalida) || repo.solicitudes != 1 {
		t.Fatal("solicitud contradictoria alcanzo repositorio", err)
	}
}

func TestCorreccionValidaPasoYVersionAntesDelRepositorio(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	repo := &repositorioCorreccionPrueba{}
	servicio, _ := NuevoServicioCorrecciones(repo, relojMarcajePrueba{instante})
	orden := ordenCorreccionPrueba(t)
	ref := "correccion:cronos:olvido_0001"
	_, err := servicio.ResolverRRHH(context.Background(), orden, ports.ResolucionRRHHCorreccion{
		SolicitudRef: ref, ClaveOperacion: "resol_0001", VersionEsperada: 1, Resultado: domain.ResultadoFavorable,
	})
	if !errors.Is(err, domain.ErrCorreccionInvalida) || repo.actuaciones != 0 {
		t.Fatal("RRHH acepto version de responsable", err)
	}
	recibo, err := servicio.DecidirResponsable(context.Background(), orden, ports.DecisionResponsableCorreccion{
		SolicitudRef: ref, ClaveOperacion: "decision_0001", VersionEsperada: 1, Resultado: domain.ResultadoFavorable,
	})
	if err != nil || recibo.Estado != domain.CorreccionPendienteRRHH || repo.actuaciones != 1 || repo.ultimaActuacion.Paso != domain.PasoDecisionResponsable {
		t.Fatal("decision no preparada", recibo, err)
	}
	_, err = servicio.AplicarResolucion(context.Background(), orden, ports.AplicacionCorreccion{SolicitudRef: ref, ClaveOperacion: "aplicar_0001", VersionEsperada: 2})
	if !errors.Is(err, domain.ErrCorreccionInvalida) || repo.actuaciones != 1 {
		t.Fatal("aplicacion prematura alcanzo repositorio", err)
	}
}

func TestCorreccionSinPuenteDurableFallaCerrada(t *testing.T) {
	instante := time.Now().UTC().Truncate(time.Microsecond)
	repo := &repositorioCorreccionPrueba{err: ports.ErrDependenciaNoDisponible}
	servicio, _ := NuevoServicioCorrecciones(repo, relojMarcajePrueba{instante})
	_, err := servicio.SolicitarOlvido(context.Background(), ordenCorreccionPrueba(t), ports.SolicitudOlvidoMarcaje{
		ClaveOperacion: "olvido_0001", HuecoDeclarado: true, Movimiento: domain.PunchEntry,
		FechaCivil: "2026-09-24", HoraPretendida: "08:15",
	})
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("fallo del puente durable no propagado", err)
	}
	_, err = servicio.RecuperarRecibo(context.Background(), ordenCorreccionPrueba(t), ports.ClaveRecuperacionCorreccion{SolicitudRef: "correccion:cronos:olvido_0001", ClaveOperacion: "olvido_0001", Paso: domain.PasoSolicitudCorreccion})
	if !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("recuperacion afirmada sin fuente durable", err)
	}
}
