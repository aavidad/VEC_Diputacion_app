package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func documentoConsumosAD173CLI(t *testing.T) ([]byte, []string) {
	t.Helper()
	b, err := os.ReadFile("testdata/consumos_ad173_mixtos.json")
	if err != nil {
		t.Fatal(err)
	}
	return b, []string{"--checkpoint", "testdata/consumos_ad173_checkpoint.json", "--max-bytes", "16384", "--max-registros", "3"}
}

func TestCLICadenaConsumosAD173MixtosYAvisoHistorico(t *testing.T) {
	b, args := documentoConsumosAD173CLI(t)
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 {
		t.Fatalf("cadena mixta rechazada: %d %s", codigo, &salida)
	}
	var informe auditoria.InformeVerificacion
	if json.Unmarshal(salida.Bytes(), &informe) != nil || !informe.ConsumosHistoricosSinFechaLigada ||
		!informe.FechaConsumoLigadaCotejada || informe.ContenidoConsumoRecalculado ||
		informe.AutenticidadCheckpoint != "no_comprobada" || informe.AutenticidadFuentesHistoricas != "no_comprobada" {
		t.Fatalf("alcance mixto incorrecto: %s", &salida)
	}
	// El JSON conserva la entrada mixto-v2 para consumos sin eventos AD171.
	b = bytes.Replace(b, []byte(auditoria.EsquemaVerificacionPreperfil), []byte(auditoria.EsquemaVerificacionMixta), 1)
	salida.Reset()
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 {
		t.Fatalf("cadena mixto-v2 rechazada: %d %s", codigo, &salida)
	}
}

func TestCLIConsumoAD173JSONEstrictoYAlteraciones(t *testing.T) {
	const privado = "dato_privado_sintetico"
	for _, tc := range []struct {
		nombre string
		codigo int
		mutar  func(map[string]any, map[string]any)
	}{
		{"fecha", 1, func(d, c map[string]any) { c["registrada_en"] = "2026-10-03T12:34:56.123457Z" }},
		{"dos_fechas", 1, func(d, c map[string]any) {
			c["registrada_en"], c["consumida_en"] = "2026-10-03T12:34:56.123457Z", "2026-10-03T12:34:56.123457Z"
		}},
		{"precision", 1, func(d, c map[string]any) { c["registrada_en"] = "2026-10-03T12:34:56.1234561Z" }},
		{"offset", 1, func(d, c map[string]any) { c["registrada_en"] = "2026-10-03T14:34:56.123456+02:00" }},
		{"actor", 1, func(d, c map[string]any) { c["actor_ref"] = privado }},
		{"perfil", 1, func(d, c map[string]any) { c["perfil_activo_ref"] = privado }},
		{"finalidad", 1, func(d, c map[string]any) { c["finalidad_ref"] = privado }},
		{"version", 1, func(d, c map[string]any) { c["version_consumo"] = 2 }},
		{"tipo_interno", 1, func(d, c map[string]any) { c["tipo_registro"] = auditoria.TipoConsumoOrigenV2 }},
		{"tipo_futuro", 2, func(d, c map[string]any) {
			d["registros"].([]any)[2].(map[string]any)["tipo_registro"] = "consumo_confirmado_v4"
		}},
		{"desconocido", 2, func(d, c map[string]any) { c["campo_ajeno"] = privado }},
		{"capitalizacion", 2, func(d, c map[string]any) { c["ACTOR_REF"] = c["actor_ref"]; delete(c, "actor_ref") }},
		{"null", 2, func(d, c map[string]any) { c["actor_ref"] = nil }},
		{"ausente", 2, func(d, c map[string]any) { delete(c, "consumida_en") }},
		{"union", 2, func(d, c map[string]any) { d["registros"].([]any)[2].(map[string]any)["intento"] = map[string]any{} }},
		{"fecha_historica", 2, func(d, c map[string]any) {
			d["registros"].([]any)[1].(map[string]any)["consumo"].(map[string]any)["registrada_en"] = c["registrada_en"]
		}},
	} {
		t.Run(tc.nombre, func(t *testing.T) {
			b, args := documentoConsumosAD173CLI(t)
			var d map[string]any
			if err := json.Unmarshal(b, &d); err != nil {
				t.Fatal(err)
			}
			c := d["registros"].([]any)[2].(map[string]any)["consumo"].(map[string]any)
			tc.mutar(d, c)
			b, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != tc.codigo || strings.Contains(salida.String(), privado) {
				t.Fatalf("rechazo incorrecto o dato reflejado: %d %s", codigo, &salida)
			}
		})
	}
	// El parseo por versión no admite duplicados dentro del consumo nuevo.
	b, args := documentoConsumosAD173CLI(t)
	b = bytes.Replace(b, []byte(`"actor_ref":`), []byte(`"actor_ref":"otro","actor_ref":`), 1)
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 2 {
		t.Fatalf("clave duplicada admitida: %d %s", codigo, &salida)
	}
}
