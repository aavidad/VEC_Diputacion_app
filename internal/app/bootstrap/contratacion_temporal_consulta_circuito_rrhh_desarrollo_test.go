package bootstrap

import (
	"context"
	"maps"
	"testing"

	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestConsultaCircuitoRRHHV3LigaSolicitudYRecursoExactos(t *testing.T) {
	s := ports.SolicitudConsultaCircuitoRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:    "expediente:ct:circuito:001",
		VersionObservada: 3,
	}
	recurso, err := postgresct.RecursoConsultaCircuitoRRHH(s)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.WithValue(context.Background(), claveConsultaCircuitoRRHHDesarrollo{}, s)
	datos := dominiovec.DatosSolicitudAutorizacionLigadaV3{
		Accion: postgresct.AccionConsultaCircuitoRRHH, Recurso: recurso,
		Finalidad:        "gestionar_contratacion_temporal",
		ReferenciaMotivo: motivoConsultaCircuitoRRHHDesarrollo(),
	}
	if !solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		t.Fatal("la consulta exacta se ha denegado")
	}
	datos.Recurso.Atributos = maps.Clone(recurso.Atributos)
	datos.Recurso.Atributos["material_sha256"] = huellaAltaContratacionTemporalDesarrollo("otra-consulta")
	if solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		t.Fatal("se aceptó otra versión o expediente con el mismo permiso")
	}
	datos.Recurso = recurso
	datos.Accion = ports.AccionConsultarDetalleRRHH
	if solicitudAutorizacionConsultaCircuitoRRHHValida(ctx, datos) {
		t.Fatal("se aceptó un permiso de consulta del detalle antiguo")
	}
}
