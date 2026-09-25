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

type proveedorSolicitudesPrueba struct{}

func (proveedorSolicitudesPrueba) ProveerMaterialConsultaMovimientosPropios(context.Context, domain.MaterialConsultaMovimientosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorSolicitudesPrueba) ProveerMaterialConsultaPermisosPropios(context.Context, domain.MaterialConsultaPermisosPropios) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}
func (proveedorSolicitudesPrueba) ProveerMaterialSolicitudPermisoPropio(context.Context, domain.MaterialSolicitudPermisoPropio) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("sin V3 en la prueba")
}

type repoMovimientosPrueba struct {
	empleado, desde, hasta string
	r                      ports.ConsultaMovimientos
}

func (r *repoMovimientosPrueba) ConsultarMovimientos(_ context.Context, _ ports.OrdenConsultaMovimientos, empleado, desde, hasta, _ string) (ports.ConsultaMovimientos, error) {
	r.empleado, r.desde, r.hasta = empleado, desde, hasta
	res := r.r
	res.Periodo = ports.PeriodoConsultaSaldo{Desde: desde, Hasta: hasta}
	return res, nil
}

func TestMovimientosDelAnioDerivaEmpleadoYRechazaHechosFueraDelPeriodo(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, err := ports.NuevaOrdenConsultaMovimientos(actor, proveedorSolicitudesPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Now().UTC()
	anio := ahora.In(zona).Format("2006")
	repo := &repoMovimientosPrueba{r: ports.ConsultaMovimientos{Calendario: ports.CalendarioMovimientos{Disponible: true, Dias: []ports.DiaCalendario{{Fecha: anio + "-01-01", Tipo: "festivo", Nombre: "Año nuevo"}}}}}
	s, _ := NuevoServicioConsultaMovimientos(repo, relojMarcajePrueba{ahora}, zona)
	r, err := s.ConsultarMovimientos(context.Background(), orden, ports.PeriodoSaldoAnio, "", "")
	if err != nil || repo.empleado != "emp_0123456789abcdefghijkl" || repo.desde != anio+"-01-01" || repo.hasta != anio+"-12-31" || r.Periodo.Tipo != ports.PeriodoSaldoAnio {
		t.Fatalf("periodo o empleado distintos: %+v %v", repo, err)
	}
	repo.r.Calendario.Dias[0].Fecha = "1999-01-01"
	if _, err := s.ConsultarMovimientos(context.Background(), orden, ports.PeriodoSaldoAnio, "", ""); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un festivo fuera del periodo", err)
	}
	repo.r.Calendario = ports.CalendarioMovimientos{Disponible: false, Dias: []ports.DiaCalendario{{Fecha: anio + "-01-01", Tipo: "festivo", Nombre: "x"}}}
	if _, err := s.ConsultarMovimientos(context.Background(), orden, ports.PeriodoSaldoAnio, "", ""); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("inventa días de un calendario no disponible", err)
	}
	if _, err := s.ConsultarMovimientos(context.Background(), orden, ports.PeriodoSaldoRango, "2026-01-01", "2027-06-01"); !errors.Is(err, ports.ErrConsultaSaldoInvalida) {
		t.Fatal("acepta un periodo de más de un año", err)
	}
}

type repoPermisosPrueba struct {
	fuente   ports.FuentePermisosPropios
	material domain.MaterialSolicitudPermisoPropio
	recibo   ports.ReciboPermisoPropio
}

func (r *repoPermisosPrueba) ConsultarPermisosPropios(_ context.Context, _ ports.OrdenPermisosPropios, empleado string, anio int, _ string) (ports.FuentePermisosPropios, error) {
	f := r.fuente
	f.EmpleadoRef, f.Anio = empleado, anio
	return f, nil
}

func (r *repoPermisosPrueba) SolicitarPermisoPropio(_ context.Context, _ ports.OrdenPermisosPropios, m domain.MaterialSolicitudPermisoPropio) (ports.ReciboPermisoPropio, error) {
	r.material = m
	return r.recibo, nil
}

func catalogoPrueba(ref string, maxAnual int64) domain.CatalogoPermisoVersion {
	return domain.CatalogoPermisoVersion{PermisoRef: "permiso:cronos:" + ref, VersionRef: "catalogo:cronos:" + ref + ":v1", FuenteRef: "fuente:sintetica:duda-41", Nombre: ref,
		VigenteDesde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Unidad: domain.LeaveUnitDay, Computo: domain.ComputoLaborables,
		Circuito: domain.CircuitoAdministracion, Minimo: 1, MaximoAnual: &maxAnual}
}

func TestPermisosPropiosProyectaRestaYNoConcede(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, _ := ports.NuevaOrdenPermisosPropios(actor, proveedorSolicitudesPrueba{})
	repo := &repoPermisosPrueba{fuente: ports.FuentePermisosPropios{
		Catalogo: []ports.EntradaCatalogoPropio{{Version: catalogoPrueba("asuntos-propios", 6), Solicitable: true, Sintetico: true}},
		Solicitudes: []ports.SolicitudPermisoPropia{
			{SolicitudRef: "permiso:cronos:solicitud:a-00000001", CatalogoVersionRef: "catalogo:cronos:asuntos-propios:v1", PermisoRef: "permiso:cronos:asuntos-propios", Desde: "2026-03-02", Hasta: "2026-03-03", Cantidad: 2, Unidad: domain.LeaveUnitDay, Estado: domain.EstadoPermisoConcedido, Version: 2, PendienteJustificar: true},
			{SolicitudRef: "permiso:cronos:solicitud:a-00000002", CatalogoVersionRef: "catalogo:cronos:asuntos-propios:v1", PermisoRef: "permiso:cronos:asuntos-propios", Desde: "2026-04-06", Hasta: "2026-04-06", Cantidad: 1, Unidad: domain.LeaveUnitDay, Estado: domain.EstadoPermisoSolicitado, Version: 1},
		}}}
	s, _ := NuevoServicioPermisosPropios(repo, relojMarcajePrueba{time.Now().UTC()}, zona)
	r, err := s.ConsultarPermisosPropios(context.Background(), orden, 2026)
	if err != nil || r.Anio != 2026 || len(r.Permisos) != 1 {
		t.Fatal(r, err)
	}
	p := r.Permisos[0]
	if p.Solicitado != 1 || p.Concedido != 2 || p.PendienteJustificar != 2 || p.Resta == nil || *p.Resta != 3 || !p.Solicitable || !p.Sintetico {
		t.Fatalf("proyección distinta: %+v", p)
	}
	repo.fuente.Solicitudes[1].Desde = "2025-12-30"
	if _, err := s.ConsultarPermisosPropios(context.Background(), orden, 2026); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta una solicitud de otro año", err)
	}
	if _, err := s.ConsultarPermisosPropios(context.Background(), orden, 1999); !errors.Is(err, ports.ErrSolicitudCronosInvalida) {
		t.Fatal("acepta un año fuera de rango", err)
	}
}

func TestSolicitarPermisoPropioConstruyeMaterialDelServidorYExigeReciboPendiente(t *testing.T) {
	zona, _ := time.LoadLocation("Europe/Madrid")
	actor, _ := contexto(t).OrdenConsumo.ContextoActor()
	orden, _ := ports.NuevaOrdenPermisosPropios(actor, proveedorSolicitudesPrueba{})
	repo := &repoPermisosPrueba{recibo: ports.ReciboPermisoPropio{SolicitudRef: "permiso:cronos:solicitud:perm-clave-0001", ReciboRef: "recibo:cronos:x", CatalogoVersionRef: "catalogo:cronos:asuntos-propios:v1",
		Version: 1, Estado: domain.EstadoPermisoSolicitado, Cantidad: 2, Unidad: domain.LeaveUnitDay, InstanteUTC: time.Now().UTC()}}
	s, _ := NuevoServicioPermisosPropios(repo, relojMarcajePrueba{time.Now().UTC()}, zona)
	p := ports.PeticionPermisoPropio{PermisoRef: "permiso:cronos:asuntos-propios", Desde: "2026-10-05", Hasta: "2026-10-06", ClaveOperacion: "perm-clave-0001"}
	if _, err := s.SolicitarPermisoPropio(context.Background(), orden, p); err != nil {
		t.Fatal(err)
	}
	if repo.material.EmpleadoRef != "emp_0123456789abcdefghijkl" || repo.material.ActorRef != actor.PersonaRef || repo.material.ZonaHoraria != "Europe/Madrid" {
		t.Fatalf("material no derivado del servidor: %+v", repo.material)
	}
	repo.recibo.Estado = domain.EstadoPermisoConcedido
	if _, err := s.SolicitarPermisoPropio(context.Background(), orden, p); !errors.Is(err, ports.ErrDependenciaNoDisponible) {
		t.Fatal("acepta un recibo que concede desde la solicitud", err)
	}
	p.HoraInicio = "09:00"
	if _, err := s.SolicitarPermisoPropio(context.Background(), orden, p); !errors.Is(err, ports.ErrSolicitudCronosInvalida) {
		t.Fatal("acepta tramo horario incompleto", err)
	}
	p.HoraFin, p.Hasta = "08:00", p.Desde
	if _, err := s.SolicitarPermisoPropio(context.Background(), orden, p); !errors.Is(err, ports.ErrSolicitudCronosInvalida) {
		t.Fatal("acepta tramo horario invertido", err)
	}
}
