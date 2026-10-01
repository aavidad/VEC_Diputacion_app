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

func TestReconciliarIdentidadesInternasH6DetectaPlanAlterado(t *testing.T) {
	plan := PlanIdentidadesInternasH6{
		Esquema: "vec.h6.provision.plan.v1", FuenteSHA256: HuellaFuenteIdentidadesInternasH6,
		PlanSHA256: HuellaPlanIdentidadesInternasH6, Estado: "bloqueado",
		Actores: []ActorPlanIdentidadesInternasH6{
			{Funcion: "rrhh", Estado: "denegado", Bloqueos: []string{"aprobacion_nominal_ausente", "asignacion_h6_ausente", "autoridad_cas_ausente", "contexto_h6_ausente", "cuenta_h6_ausente", "cuenta_hmac_sin_provisionar", "posesion_certificado_no_acreditada"}},
			{Funcion: "intervencion", Estado: "denegado", Bloqueos: []string{"aprobacion_nominal_ausente", "asignacion_h6_ausente", "autoridad_cas_ausente", "contexto_h6_ausente", "cuenta_h6_ausente", "cuenta_hmac_sin_provisionar", "posesion_certificado_no_acreditada"}},
			{Funcion: "solicitante_centro", Estado: "denegado", Bloqueos: []string{"aprobacion_nominal_ausente", "asignacion_h6_preimagen_sin_revalidar", "autoridad_cas_ausente", "certificado_pendiente", "contexto_nominal_no_acreditado", "cuenta_nominal_no_acreditada", "material_hmac_no_acreditado", "sujeto_pendiente"}},
			{Funcion: "ratificador_centro", Estado: "denegado", Bloqueos: []string{"aprobacion_nominal_ausente", "asignacion_h6_preimagen_sin_revalidar", "autoridad_cas_ausente", "certificado_pendiente", "contexto_nominal_no_acreditado", "cuenta_nominal_no_acreditada", "material_hmac_no_acreditado", "sujeto_pendiente"}},
		},
	}
	if _, err := ReconciliarIdentidadesInternasH6(plan, HuellaPlanIdentidadesInternasH6); err != nil {
		t.Fatalf("plan fijado: %v", err)
	}
	plan.Actores[0].Estado = "confirmado"
	if _, err := ReconciliarIdentidadesInternasH6(plan, HuellaPlanIdentidadesInternasH6); !errors.Is(err, ErrPlanIdentidadesInternasH6) {
		t.Fatalf("se aceptó plan alterado con el mismo campo de huella: %v", err)
	}
	plan.Actores[0].Estado = "denegado"
	plan.FuenteSHA256 = strings.Repeat("a", 64)
	if _, err := ReconciliarIdentidadesInternasH6(plan, HuellaPlanIdentidadesInternasH6); !errors.Is(err, ErrPlanIdentidadesInternasH6) {
		t.Fatalf("se aceptó fuente ajena con el mismo campo de huella: %v", err)
	}
}
