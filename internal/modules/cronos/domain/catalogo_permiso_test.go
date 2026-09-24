package domain

import (
	"testing"
	"time"
)

func catalogoPrueba() CatalogoPermisoVersion {
	max := int64(20)
	return CatalogoPermisoVersion{PermisoRef: "permiso:prueba", VersionRef: "catalogo:v1", FuenteRef: "fuente:validada", Nombre: "Permiso de prueba", VigenteDesde: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Unidad: LeaveUnitDay, Computo: ComputoLaborables, Circuito: CircuitoResponsableAdministracion, Minimo: 1, MaximoAnual: &max}
}

func TestProyeccionAnualSinDobleDescuento(t *testing.T) {
	c := catalogoPrueba()
	r, err := ProyectarPermisoAnual(c, 2026, []HechoResumenPermiso{
		{SolicitudRef: "sol:1", CatalogoVersionRef: "catalogo:v1", Unidad: LeaveUnitDay, Cantidad: 3, Estado: EstadoPermisoSolicitado},
		{SolicitudRef: "sol:2", CatalogoVersionRef: "catalogo:v1", Unidad: LeaveUnitDay, Cantidad: 4, Estado: EstadoPermisoPendienteAdministracion},
		{SolicitudRef: "sol:3", CatalogoVersionRef: "catalogo:v0", Unidad: LeaveUnitDay, Cantidad: 5, Estado: EstadoPermisoConcedido, PendienteJustificar: true},
		{SolicitudRef: "sol:4", CatalogoVersionRef: "catalogo:v0", Unidad: LeaveUnitDay, Cantidad: 2, Estado: EstadoPermisoDenegado},
	})
	if err != nil || r.Solicitado != 7 || r.Concedido != 5 || r.PendienteJustificar != 5 || r.Resta == nil || *r.Resta != 8 {
		t.Fatalf("proyeccion inesperada: %+v, %v", r, err)
	}
	c.MaximoAnual = nil
	r, err = ProyectarPermisoAnual(c, 2026, nil)
	if err != nil || r.Resta != nil {
		t.Fatalf("cuota no acreditada debe quedar ausente: %+v, %v", r, err)
	}
}

func TestProyeccionRechazaHechosAmbiguos(t *testing.T) {
	c := catalogoPrueba()
	for _, hechos := range [][]HechoResumenPermiso{
		{{SolicitudRef: "sol:1", CatalogoVersionRef: "catalogo:v1", Unidad: LeaveUnitDay, Cantidad: 1, Estado: EstadoPermisoSolicitado}, {SolicitudRef: "sol:1", CatalogoVersionRef: "catalogo:v1", Unidad: LeaveUnitDay, Cantidad: 1, Estado: EstadoPermisoConcedido}},
		{{SolicitudRef: "sol:2", CatalogoVersionRef: "catalogo:v1", Unidad: LeaveUnitDay, Cantidad: 1, Estado: EstadoPermisoDenegado, PendienteJustificar: true}},
		{{SolicitudRef: "sol:3", CatalogoVersionRef: "catalogo:v1", Unidad: LeaveUnitDay, Cantidad: 21, Estado: EstadoPermisoConcedido}},
		{{SolicitudRef: "sol:4", CatalogoVersionRef: "catalogo:v0", Unidad: LeaveUnitHour, Cantidad: 2, Estado: EstadoPermisoConcedido}},
	} {
		if _, err := ProyectarPermisoAnual(c, 2026, hechos); err == nil {
			t.Fatalf("se aceptaron hechos invalidos: %+v", hechos)
		}
	}
}

func TestCircuitoPermisoExigePasosSeparados(t *testing.T) {
	if _, err := SiguienteEstadoPermiso(CircuitoResponsableAdministracion, EstadoPermisoSolicitado, PasoAdministracion, DecisionAprobar); err == nil {
		t.Fatal("administracion salto al responsable")
	}
	estado, err := SiguienteEstadoPermiso(CircuitoResponsableAdministracion, EstadoPermisoSolicitado, PasoResponsable, DecisionAprobar)
	if err != nil || estado != EstadoPermisoPendienteAdministracion {
		t.Fatalf("paso jefatura: %s %v", estado, err)
	}
	estado, err = SiguienteEstadoPermiso(CircuitoResponsableAdministracion, estado, PasoAdministracion, DecisionAprobar)
	if err != nil || estado != EstadoPermisoConcedido {
		t.Fatalf("paso administracion: %s %v", estado, err)
	}
	if _, err := SiguienteEstadoPermiso(CircuitoResponsableAdministracion, estado, PasoAdministracion, DecisionAprobar); err == nil {
		t.Fatal("concesion terminal modificada")
	}
	estado, err = SiguienteEstadoPermiso(CircuitoAdministracion, EstadoPermisoSolicitado, PasoAdministracion, DecisionDenegar)
	if err != nil || estado != EstadoPermisoDenegado {
		t.Fatalf("circuito A: %s %v", estado, err)
	}
}
