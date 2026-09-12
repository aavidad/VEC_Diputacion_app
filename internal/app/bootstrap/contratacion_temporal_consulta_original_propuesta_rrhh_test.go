package bootstrap

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestMultiplexorOriginalPropuestaRRHHSeparaSesionHistorica(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	lector := &soporteAltaContratacionTemporalDesarrollo{
		sello:             soporte.sello,
		principalID:       "lector:rrhh:original",
		certificadoSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	}
	cuadro := &consultorCuadroLectorPrueba{}
	detalle := &consultorDetalleLectorPrueba{}
	original := &consultorDetalleLectorPrueba{}
	mux := nuevoMultiplexoresLectoresRRHHDesarrollo(soporte.sello)
	if !mux.registrar(lector.principalID, lector, cuadro, detalle, original) {
		t.Fatal("no registró el lector histórico")
	}
	actor := clonarPrincipalDesarrollo(principal)
	actor.ID = lector.principalID
	actor.Attributes["certificate_sha256"] = lector.certificadoSHA256
	ctx := context.WithValue(context.Background(), claveCapacidadConsultasContratacionTemporalDesarrollo{}, capacidadConsultaContratacionTemporalDesarrollo{
		sello: soporte.sello, ruta: httpinterno.RutaConsultaDetalleRRHH, principal: actor,
	})
	v7, err := ports.NuevaSolicitudDetalleRRHH("expediente:rrhh:original", 7)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mux.ConsultarOriginalPropuesta(ctx, v7); err != nil || original.llamadas != 1 || detalle.llamadas != 0 {
		t.Fatalf("lectura histórica no separada: %v, original=%d normal=%d", err, original.llamadas, detalle.llamadas)
	}
	v8, err := ports.NuevaSolicitudDetalleRRHH("expediente:rrhh:original", 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mux.ConsultarOriginalPropuesta(ctx, v8); !errors.Is(err, ports.ErrAutorizacionDenegada) || original.llamadas != 1 {
		t.Fatalf("fachada histórica admitió otra versión: %v", err)
	}
	if _, err := mux.ConsultarDetalle(ctx, v7); err != nil || detalle.llamadas != 1 || original.llamadas != 1 {
		t.Fatalf("lectura ordinaria cambió de sesión: %v", err)
	}
}
