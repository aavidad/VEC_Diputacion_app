package ports

import (
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestBootstrapAdministracionPerfilesExigeDosPersonasAcreditadas(t *testing.T) {
	persona := func(letra string) domain.PreimagenAdministracionPerfiles {
		return domain.PreimagenAdministracionPerfiles{
			CuentaRef: "cta_" + strings.Repeat(letra, 22), CuentaVersion: 1,
			PersonaRef: "per_" + strings.Repeat(letra, 22), PersonaVersion: 1,
			PerfilRef:      "prf_" + strings.Repeat(letra, 22),
			VinculoRef:     "vca_" + strings.Repeat(letra, 22),
			HuellaSHA256:   strings.Repeat("a", 64),
			ProcedenciaRef: "procedencia:maestra:1", ProcedenciaVersion: 1,
			ProcedenciaHuellaSHA256: strings.Repeat("b", 64),
			VigenteHasta:            time.Date(2027, 9, 30, 0, 0, 0, 0, time.UTC),
		}
	}
	plan := PreimagenBootstrapAdministracionPerfiles{
		Primera: persona("a"), Segunda: persona("b"), HuellaPlanSHA256: strings.Repeat("c", 64),
	}
	if err := plan.Validar(); err != nil {
		t.Fatalf("dos personas: %v", err)
	}
	recibo := ReciboBootstrapAdministracionPerfiles{
		ActoRef:           "acto_admin:" + strings.Repeat("d", 32),
		ReciboRef:         "recibo_admin:" + strings.Repeat("e", 32),
		HuellaPlanSHA256:  plan.HuellaPlanSHA256,
		PrimeraPersonaRef: plan.Primera.PersonaRef,
		SegundaPersonaRef: plan.Segunda.PersonaRef,
		ConfirmadoEn:      time.Date(2026, 9, 30, 19, 0, 0, 0, time.UTC),
	}
	if err := recibo.ValidarPara(plan); err != nil {
		t.Fatalf("recibo unico: %v", err)
	}
	recibo.SegundaPersonaRef = recibo.PrimeraPersonaRef
	if err := recibo.ValidarPara(plan); err == nil {
		t.Fatal("recibo para la misma persona aceptado")
	}
	plan.Segunda.PersonaRef = plan.Primera.PersonaRef
	if err := plan.Validar(); err == nil {
		t.Fatal("dos cuentas de la misma persona aceptadas")
	}
	plan.Segunda = persona("b")
	plan.Segunda.PersonaVersion = 0
	if err := plan.Validar(); err == nil {
		t.Fatal("persona sin version acreditada aceptada")
	}
	plan.Segunda = persona("b")
	plan.Primera.RevisionContinuidad = 1
	if err := plan.Validar(); err == nil {
		t.Fatal("bootstrap sobre continuidad ya iniciada aceptado")
	}
}
