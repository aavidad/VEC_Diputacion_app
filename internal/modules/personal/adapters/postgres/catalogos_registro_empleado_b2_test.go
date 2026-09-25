package postgres

import "testing"

func TestCatalogosRegistroEmpleadoB2RechazaRespuestaSQLIncompleta(t *testing.T) {
	consultaValida := `{"organismo_ref":"org_prueba","entradas":[],"cursor_siguiente":null,"evidencia":{"decision_ref":"dec_1","auditoria_ref":"aud_1","consumo_huella_sha256":"a","efecto_ref":"org_prueba:regimen","consultada_en":"2026-09-25T10:00:00Z"}}`
	if !formaConsultaCatalogoEmpleadoB2([]byte(consultaValida)) {
		t.Fatal("forma de consulta SQL nominal rechazada")
	}
	consultas := []string{
		`{"organismo_ref":"org_prueba","entradas":null,"cursor_siguiente":null,"evidencia":{"decision_ref":"dec_1","auditoria_ref":"aud_1","consumo_huella_sha256":"a","efecto_ref":"org_prueba:regimen","consultada_en":"2026-09-25T10:00:00Z"}}`,
		`{"organismo_ref":"org_prueba","entradas":[],"cursor_siguiente":null,"evidencia":{"decision_ref":"dec_1","auditoria_ref":"aud_1","consumo_huella_sha256":"a"}}`,
		`{"organismo_ref":"org_prueba","entradas":[],"cursor_siguiente":null,"evidencia":{},"otro":"filtracion"}`,
		`{"organismo_ref":"org_prueba","entradas":[],"entradas":[],"cursor_siguiente":null,"evidencia":{}}`,
	}
	for _, bruto := range consultas {
		if formaConsultaCatalogoEmpleadoB2([]byte(bruto)) {
			t.Fatal("consulta SQL incompleta aceptada")
		}
	}
	cambios := []string{
		`{"entrada":{},"recibo":{},"acceso_actual":{}}`,
		`{"entrada":{"organismo_ref":"org_1","tipo":"regimen","ref":"reg_1","version":1,"revision":1,"estado":"publicada","denominacion":"Régimen","huella_sha256":"a","vigente_desde":"2026-09-25","vigente_hasta":null},"recibo":{"decision_ref":"dec_1","auditoria_ref":"aud_1","consumo_huella_sha256":"a","registrado_en":"2026-09-25T10:00:00Z"},"acceso_actual":{"decision_ref":"dec_1","auditoria_ref":"aud_1","consumo_huella_sha256":"a","registrado_en":"2026-09-25T10:00:00Z"}}`,
		`{"entrada":{},"recibo":{},"acceso_actual":{},"otra":"filtracion"}`,
	}
	for _, bruto := range cambios {
		if formaCambioCatalogoEmpleadoB2([]byte(bruto)) {
			t.Fatal("cambio SQL incompleto aceptado")
		}
	}
}
