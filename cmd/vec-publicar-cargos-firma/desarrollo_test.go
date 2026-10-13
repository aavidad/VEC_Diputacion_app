package main

import (
	"testing"
	"time"

	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestPlanDesarrolloNoCambiaGarantiaDelContratoAnterior(t *testing.T) {
	en := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	c := cargoEntrada{RolID: "ct_cargo_tecnico_documentos_dev", Version: 1, Nombre: "Documentos RRHH",
		OperacionesV2: []string{"confirmar_original", "custodiar_firmado", "descargar_original", "escribir_original", "reservar_original"},
		Competencias:  []competencia{}, ReglaAsignacion: "lote_ordinario", OrganizacionRef: "org:rrhh", VigenteDesde: en,
		VigenteHasta: en.Add(24 * time.Hour), DuracionPropuestaSegundos: 3600}
	p, err := prepararPlan(ficheroCargos{Esquema: esquemaCargosDesarrollo, Cargos: []cargoEntrada{c}}, "rpa_cf_00000000000000000000000000000087", en, time.Hour)
	if err != nil || comprobarPlan(textoPlan(p), p) != nil {
		t.Fatalf("plan v2: %v", err)
	}
	v, err := versionRolModo(c, p.OperacionRef, en, true)
	if err != nil || len(v.Concesiones) != 5 {
		t.Fatalf("rol: %v", err)
	}
	for _, co := range v.Concesiones {
		if co.GarantiaMinima != vd.AuthAssuranceSubstantial {
			t.Fatal("garantía v2")
		}
	}
	if _, err := prepararPlan(ficheroCargos{Esquema: esquemaCargos, Cargos: []cargoEntrada{c}}, p.OperacionRef, en, time.Hour); err == nil {
		t.Fatal("v1 aceptó operaciones v2")
	}
	old, _ := concesionV2("firmar_vec")
	if old.GarantiaMinima != vd.AuthAssuranceHigh {
		t.Fatal("cambió garantía histórica")
	}
	c.RolID = "ct_cargo_secretaria_documentos_dev"
	if validarCargoModo(c, true) == nil {
		t.Fatal("Secretaría puede preparar originales")
	}
	c.RolID = "administracion_perfiles"
	if validarCargoModo(c, true) == nil {
		t.Fatal("rol administrativo admitido")
	}
}
