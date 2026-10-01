package bootstrap

import (
	"errors"
	"strings"
	"testing"
)

func TestPlanIdentidadesInternasH6RechazaFuenteNoAcreditada(t *testing.T) {
	for _, fuente := range []string{"", `{}`, `{"esquema":"vec.fuente.sintetica.interna.propuesta.v1","datos_sinteticos":true}`} {
		if _, err := PlanificarIdentidadesInternasH6(strings.NewReader(fuente)); !errors.Is(err, ErrFuenteIdentidadesInternasH6) {
			t.Fatalf("fuente sin huella acreditada: %v", err)
		}
	}
}

func TestProvisionIdentidadesInternasH6FallaCerrada(t *testing.T) {
	plan := PlanIdentidadesInternasH6{PlanSHA256: strings.Repeat("a", 64)}
	if _, err := ReconciliarIdentidadesInternasH6(plan, strings.Repeat("b", 64)); !errors.Is(err, ErrPlanIdentidadesInternasH6) {
		t.Fatalf("reconcile aceptó huella distinta: %v", err)
	}
	if err := AplicarIdentidadesInternasH6(plan); !errors.Is(err, ErrProvisionIdentidadesInternasH6) {
		t.Fatalf("apply sin autoridad CAS: %v", err)
	}
}
