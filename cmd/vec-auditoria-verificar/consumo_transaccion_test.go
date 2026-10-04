package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/auditoria"
)

func entradaAD193CLI(t *testing.T) ([]byte, []string) {
	t.Helper()
	b, err := os.ReadFile("testdata/consumo_transaccion_ad193.json")
	if err != nil {
		t.Fatal(err)
	}
	return b, []string{"--checkpoint", "testdata/consumo_transaccion_ad193_checkpoint.json", "--max-bytes", "16384", "--max-registros", "5"}
}

func TestCLIAD193ConservaSellosUint64YFormatos(t *testing.T) {
	for _, esquema := range []string{auditoria.EsquemaVerificacionMixta, auditoria.EsquemaVerificacionPreperfil,
		auditoria.EsquemaVerificacionFuentesIniciales, auditoria.EsquemaVerificacionUnidadInicial,
		auditoria.EsquemaVerificacionBootstrapCentral, auditoria.EsquemaVerificacionMantenimientoFijo,
		auditoria.EsquemaVerificacionFronteraAdminTecnicaV1} {
		b, args := entradaAD193CLI(t)
		b = bytes.Replace(b, []byte(auditoria.EsquemaVerificacionPreperfil), []byte(esquema), 1)
		var salida bytes.Buffer
		if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 0 {
			t.Fatalf("esquema=%s código=%d salida=%s", esquema, codigo, &salida)
		}
		if !strings.Contains(salida.String(), `"fecha_consumo_ligada_cotejada":true`) || !strings.Contains(salida.String(), `"consumos_historicos_sin_fecha_ligada":true`) || strings.Contains(salida.String(), `"actor_perfil_contexto_cotejados":true`) {
			t.Fatalf("alcance ampliado: %s", &salida)
		}
	}
}

func TestCLIAD193CierraTiposCamposYMutaciones(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		codigo int
		mutar  func(map[string]any, map[string]any)
	}{
		{"numero_json", 2, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = json.Number("9007199254740993") }},
		{"numero_consumo", 2, func(_ map[string]any, c map[string]any) {
			c["transaccion_consumo_origen"] = json.Number("18446744073709551615")
		}},
		{"null", 2, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = nil }},
		{"omitida", 2, func(_ map[string]any, c map[string]any) { delete(c, "transaccion_origen") }},
		{"omitida_consumo", 2, func(_ map[string]any, c map[string]any) { delete(c, "transaccion_consumo_origen") }},
		{"alias", 2, func(_ map[string]any, c map[string]any) {
			c["Transaccion_origen"] = c["transaccion_origen"]
			delete(c, "transaccion_origen")
		}},
		{"extra", 2, func(_ map[string]any, c map[string]any) { c["dato_ajeno"] = "dato_privado_sintetico" }},
		{"cero", 1, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = "0" }},
		{"overflow", 1, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = "18446744073709551616" }},
		{"signo", 1, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = "+9007199254740993" }},
		{"exponente", 1, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = "9e15" }},
		{"ceros", 1, func(_ map[string]any, c map[string]any) { c["transaccion_origen"] = "09007199254740993" }},
		{"distinto", 1, func(_ map[string]any, c map[string]any) { c["transaccion_consumo_origen"] = "9007199254740994" }},
		{"version", 1, func(_ map[string]any, c map[string]any) { c["version_consumo"] = 3 }},
		{"fecha", 1, func(_ map[string]any, c map[string]any) { c["registrada_en"] = "2026-10-03T12:34:56.123457Z" }},
		{"cruce", 2, func(d map[string]any, _ map[string]any) {
			d["registros"].([]any)[3].(map[string]any)["intento"] = map[string]any{}
		}},
		{"sello_legacy", 2, func(d map[string]any, _ map[string]any) {
			d["registros"].([]any)[2].(map[string]any)["consumo"].(map[string]any)["transaccion_origen"] = "9007199254740993"
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			b, args := entradaAD193CLI(t)
			var d map[string]any
			if json.Unmarshal(b, &d) != nil {
				t.Fatal("fixture inválido")
			}
			c := d["registros"].([]any)[3].(map[string]any)["consumo"].(map[string]any)
			caso.mutar(d, c)
			b, err := json.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			var salida bytes.Buffer
			if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != caso.codigo || strings.Contains(salida.String(), "dato_privado_sintetico") {
				t.Fatalf("rechazo=%d salida=%s", codigo, &salida)
			}
		})
	}
	b, args := entradaAD193CLI(t)
	b = bytes.Replace(b, []byte(`"transaccion_origen":`), []byte(`"transaccion_origen":"1","transaccion_origen":`), 1)
	var salida bytes.Buffer
	if codigo := ejecutar(args, bytes.NewReader(b), &salida); codigo != 2 {
		t.Fatal("sello duplicado admitido")
	}
}
