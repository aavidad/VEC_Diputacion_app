package main

import (
	"slices"
	"testing"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
)

func TestPlanCargoFirmaDesarrolloPublicaRecuperacionExacta(t *testing.T) {
	en := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	c := cargoEntrada{RolID: "ct_cargo_jefatura_firma_dev", Version: 2,
		VersionAnteriorSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Nombre:                "Jefatura de RRHH", OperacionesV2: []string{"consultar_r5", "firmar_vec", "recuperar_r5"},
		Competencias: []competencia{{Accion: ct.AccionRegistrarFirmaVec,
			TipoRecurso: "documento_contratacion_temporal", Finalidad: "gestionar_contratacion_temporal"}},
		ReglaAsignacion: "lote_ordinario", OrganizacionRef: "org:rrhh", VigenteDesde: en,
		VigenteHasta: en.Add(48 * time.Hour), DuracionPropuestaSegundos: 3600}
	plan, err := prepararPlan(ficheroCargos{Esquema: esquemaCargosDesarrollo, Cargos: []cargoEntrada{c}},
		"rpa_cf_00000000000000000000000000000104", en, time.Hour)
	if err != nil || comprobarPlan(textoPlan(plan), plan) != nil {
		t.Fatalf("plan de recuperación: %v", err)
	}
	rol, err := versionRolModo(c, plan.OperacionRef, en, true)
	if err != nil || len(rol.Concesiones) != 4 {
		t.Fatalf("rol de firma: %v", err)
	}
	recuperacion := rol.Concesiones[2]
	if recuperacion.Accion != ct.AccionRecuperarFirmasR5V2 ||
		recuperacion.GarantiaMinima != vd.AuthAssuranceSubstantial ||
		!slices.Equal(recuperacion.CamposPermitidos, ct.CamposRecuperacionFirmasV2()) ||
		len(recuperacion.Obligaciones) != 0 ||
		rol.Concesiones[0].Accion != ct.AccionConsultarFirmasR5V2 ||
		rol.Concesiones[1].Accion != ct.AccionRegistrarFirmaVec {
		t.Fatalf("permisos del cargo alterados: %+v", rol.Concesiones)
	}
	c.RolID = "ct_cargo_jefatura_consulta_dev"
	c.Competencias = nil
	if validarCargoModo(c, true) == nil {
		t.Fatal("cargo de consulta admitió recuperación")
	}
	c.RolID = "ct_cargo_jefatura_firma_dev"
	if validarCargoModo(c, false) == nil {
		t.Fatal("plan alto admitió recuperación de desarrollo")
	}
}

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

func TestPlanConsultaV2PublicaSoloCampoSelectoresYConservaOtrasAcciones(t *testing.T) {
	en := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	c := cargoEntrada{RolID: "ct_cargo_jefatura_consulta_dev", Version: 2, Nombre: "Consulta de Jefatura RRHH",
		VersionAnteriorSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		OperacionesV2:         []string{"consultar_cuadro", "consultar_detalle_selectores", "descargar_borrador"},
		ReglaAsignacion:       "lote_ordinario", OrganizacionRef: "org:rrhh", VigenteDesde: en,
		VigenteHasta: en.Add(48 * time.Hour), DuracionPropuestaSegundos: 3600}
	plan, err := prepararPlan(ficheroCargos{Esquema: esquemaCargosDesarrollo, Cargos: []cargoEntrada{c}},
		"rpa_cf_00000000000000000000000000000103", en, time.Hour)
	if err != nil || comprobarPlan(textoPlan(plan), plan) != nil {
		t.Fatalf("plan de consulta v2: %v", err)
	}
	rol, err := versionRolModo(c, plan.OperacionRef, en, true)
	if err != nil || len(rol.Concesiones) != 3 {
		t.Fatalf("rol de consulta: %v", err)
	}
	if len(rol.Concesiones[0].CamposPermitidos) != 0 ||
		len(rol.Concesiones[1].CamposPermitidos) != 1 ||
		rol.Concesiones[1].CamposPermitidos[0] != "selectores_recuperacion_firmas" ||
		len(rol.Concesiones[2].CamposPermitidos) != 0 {
		t.Fatalf("se ampliaron cuadro o descarga: %+v", rol.Concesiones)
	}
	c.OperacionesV2 = []string{"consultar_cuadro", "consultar_detalle", "consultar_detalle_selectores", "descargar_borrador"}
	if validarCargoModo(c, true) == nil {
		t.Fatal("dos concesiones de la misma acción de detalle admitidas")
	}
}
