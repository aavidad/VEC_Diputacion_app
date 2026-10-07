package bootstrap

import (
	"context"
	"reflect"
	"testing"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestProvisionLecturasRRHHBolsaConservaConcesionesPrevias(t *testing.T) {
	p, _, datos := politicaProvisionBolsaPrueba(t, 0)
	ahora := p.reloj.Ahora()
	for _, version := range []int{5, 6, 9, 10, 13, 14, 16} {
		base, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef,
			ahora, version)
		if err != nil {
			t.Fatalf("base %d: %v", version, err)
		}
		nominal, err := nuevaInstantaneaAutorizacionBorradorLlamamientoBolsaDesarrolloVersion(
			datos.PrincipalID, datos.PerfilActivoRef, p.soporte.unidadRef, p.soporte.ambitoRef,
			ahora, version+saltoProvisionLecturasNominalesBolsa)
		if err != nil {
			t.Fatalf("nominal %d: %v", version, err)
		}
		previas := base.VersionRol.Concesiones
		if len(nominal.VersionRol.Concesiones) != len(previas)+3 ||
			!reflect.DeepEqual(nominal.VersionRol.Concesiones[:len(previas)], previas) {
			t.Fatalf("v%d añadió o alteró una concesión ajena", version+saltoProvisionLecturasNominalesBolsa)
		}
		acciones := []string{puertosbolsa.AccionRRHHBolsasConsultar,
			puertosbolsa.AccionRRHHEstadisticasConsultar, puertosbolsa.AccionRRHHCandidatosConsultar}
		for i, accion := range acciones {
			if nominal.VersionRol.Concesiones[len(previas)+i].Accion != accion {
				t.Fatalf("v%d sin acción nominal %s", version+saltoProvisionLecturasNominalesBolsa, accion)
			}
		}
	}
}

func TestProvisionLecturasRRHHBolsaExigeCASYHuellasExactas(t *testing.T) {
	p, autoridad, _ := politicaProvisionBolsaPrueba(t, 6)
	pre, post, err := HuellasProvisionLecturasNominalesRRHHBolsa(context.Background(), p)
	if err != nil {
		t.Fatal(err)
	}
	if err := ProvisionarLecturasNominalesRRHHBolsa(context.Background(), p,
		"aprobacion:prueba-b84", pre, "0000000000000000000000000000000000000000000000000000000000000000"); err == nil || autoridad.publicadas != 0 {
		t.Fatal("aceptó objetivo distinto o escribió sin aprobación exacta")
	}
	if err := ProvisionarLecturasNominalesRRHHBolsa(context.Background(), p,
		"aprobacion:prueba-b84", pre, post); err != nil || autoridad.publicadas != 1 || autoridad.preimagen != 6 ||
		autoridad.publicada.VersionRol.Version != 22 {
		t.Fatalf("provisión CAS: publicadas=%d preimagen=%d versión=%d error=%v", autoridad.publicadas,
			autoridad.preimagen, autoridad.publicada.VersionRol.Version, err)
	}
}
