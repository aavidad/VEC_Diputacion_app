package bootstrap

import (
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestModalidadRetiradaSinConfirmacionSeDeniegaEnFlujoYMotivo(t *testing.T) {
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	soporte.opcionesCatalogo = &opcionesAnalisisCTDesarrollo{}
	ctx := contextoRutaCoberturaDesarrolloPrueba(soporte, principal, httpinterno.RutaAltaSolicitudes)
	flujo := ports.SolicitudResolverFlujo{
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		CentroRef:       centroAltaContratacionTemporalDesarrollo,
		CategoriaRef:    categoriaAltaContratacionTemporalDesarrollo,
		MotivoClave:     "modalidad.retirada.sintetica",
		Instante:        soporte.reloj.Ahora(),
	}
	if _, err := soporte.ResolverFlujoAlta(ctx, flujo); !errors.Is(err, ports.ErrFlujoNoDisponible) {
		t.Fatalf("flujo admitió modalidad retirada sin CT167: %v", err)
	}
	motivo := ports.SolicitudResolverMotivoAutorizacionAltaV3{
		OrganizacionRef: flujo.OrganizacionRef,
		Flujo:           soporte.flujo.Flujo,
		MotivoClave:     flujo.MotivoClave,
		Instante:        flujo.Instante,
	}
	if _, err := soporte.ResolverMotivoAutorizacionAltaV3(ctx, motivo); !errors.Is(err, ports.ErrMotivoAutorizacionNoDisponible) {
		t.Fatalf("motivo admitió modalidad retirada sin CT167: %v", err)
	}
}
