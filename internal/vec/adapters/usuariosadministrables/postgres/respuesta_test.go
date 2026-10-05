package postgres

import (
	"encoding/json"
	"fmt"
	"testing"

	"vec-diputacion-granada/internal/vec/ports"
)

func consumoPrueba() map[string]any {
	return map[string]any{"decision_ref": "dec_prueba", "efecto_ref": "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"huella_efecto_sha256": fmt.Sprintf("%064x", 1), "consumo_huella_sha256": fmt.Sprintf("%064x", 2),
		"auditoria_ref": fmt.Sprintf("aud_v3_%032x", 3), "consumida_en": "2026-10-04T05:00:00Z", "consumo_nuevo": true}
}

func TestFichaNulaConsumeYMetadatosNoTruncanPerfiles(t *testing.T) {
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	persona := "per_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	consumo := consumoPrueba()
	bruto, _ := json.Marshal(map[string]any{"datos": nil, "consumo": consumo})
	x, err := fichaRespuesta(bruto, a, persona, "dec_prueba", persona, fmt.Sprintf("%064x", 1))
	if err != nil || x != nil {
		t.Fatalf("ficha nula: %v %v", x, err)
	}
	perfiles := make([]map[string]any, 51)
	for i := range perfiles {
		perfiles[i] = map[string]any{"perfil_ref": fmt.Sprintf("prf_%024d", i), "rol_version_ref": "rol:administracion_perfiles:v5", "version": i + 1, "estado": "caducado", "vigente_desde": "2025-01-01T00:00:00Z", "vigente_hasta": "2026-01-01T00:00:00Z"}
	}
	datos := map[string]any{"persona_ref": persona, "unidad_ref": a.UnidadRef, "denominacion_version": nil, "perfiles": perfiles}
	bruto, _ = json.Marshal(map[string]any{"datos": datos, "consumo": consumo})
	x, err = fichaRespuesta(bruto, a, persona, "dec_prueba", persona, fmt.Sprintf("%064x", 1))
	if err != nil || x == nil || x.DenominacionVersion != nil || len(x.Perfiles) != 51 {
		t.Fatalf("perfiles completos: %v %v", x, err)
	}
	datos["nombre"] = "No debe salir"
	bruto, _ = json.Marshal(map[string]any{"datos": datos, "consumo": consumo})
	if _, err = fichaRespuesta(bruto, a, persona, "dec_prueba", persona, fmt.Sprintf("%064x", 1)); err == nil {
		t.Fatal("nombre ajeno admitido")
	}
}

func TestListaVaciaExigeAcuseNuevo(t *testing.T) {
	a := ambito{"org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "unidad_admin_sintetica"}
	conjunto, _ := conjuntoUsuarios(a)
	c := consumoPrueba()
	c["efecto_ref"] = conjunto
	bruto, _ := json.Marshal(map[string]any{"datos": map[string]any{"personas": []any{}, "siguiente_cursor": ""}, "consumo": c})
	x, err := listaRespuesta(bruto, a, ports.FiltrosUsuariosAdministrables{}, "dec_prueba", conjunto, fmt.Sprintf("%064x", 1))
	if err != nil || x.Personas == nil || len(x.Personas) != 0 {
		t.Fatalf("lista vacía: %v %v", x, err)
	}
	c["consumo_nuevo"] = false
	bruto, _ = json.Marshal(map[string]any{"datos": map[string]any{"personas": []any{}, "siguiente_cursor": ""}, "consumo": c})
	if _, err = listaRespuesta(bruto, a, ports.FiltrosUsuariosAdministrables{}, "dec_prueba", conjunto, fmt.Sprintf("%064x", 1)); err == nil {
		t.Fatal("replay admitido para lectura")
	}
}
