package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/auditoria"
	"vec-diputacion-granada/internal/vec/domain"
)

func checkpointMixto(t *testing.T) ([]byte, []string, string) {
	return checkpointConCadena(t, "testdata/cadena_mixta_ad173.json")
}

func checkpointConCadena(t *testing.T, fixture string) ([]byte, []string, string) {
	t.Helper()
	b, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var documento struct {
		Manifiesto domain.CoberturaCheckpoint `json:"manifiesto"`
	}
	if err := json.Unmarshal(b, &documento); err != nil {
		t.Fatal(err)
	}
	d := t.TempDir()
	master, tsa := make([]byte, 32), make([]byte, 32)
	master[0], tsa[0] = 1, 2
	defer clear(master)
	defer clear(tsa)
	cobertura, err := json.Marshal(documento.Manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	for nombre, material := range map[string][]byte{"kms": master, "tsa": tsa, "cobertura.json": cobertura, "cadena.json": b} {
		if err := os.WriteFile(filepath.Join(d, nombre), material, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var out, log bytes.Buffer
	if run([]string{"--operacion", "emitir", "--config", "testdata/config.sintetica.json", "--entrada", filepath.Join(d, "cobertura.json"), "--kms-master", filepath.Join(d, "kms"), "--tsa-secret", filepath.Join(d, "tsa"), "--salida", filepath.Join(d, "recibo.json"), "--spki", filepath.Join(d, "publica.der")}, &out, &log) != 0 {
		t.Fatalf("emisión sintética: %s", &out)
	}
	var emision struct {
		Pin string `json:"pin_spki_sha256"`
	}
	if json.Unmarshal(out.Bytes(), &emision) != nil || emision.Pin == "" {
		t.Fatalf("pin no disponible: %s", &out)
	}
	return b, []string{"--operacion", "verificar", "--config", "testdata/config.sintetica.json", "--entrada", filepath.Join(d, "recibo.json"), "--spki", filepath.Join(d, "publica.der"), "--pin-spki-sha256", emision.Pin, "--cadena", filepath.Join(d, "cadena.json")}, filepath.Join(d, "cadena.json")
}

func resultadoMixto(t *testing.T, args []string, codigo int) application.ResultadoCheckpointDesarrollo {
	t.Helper()
	var out, log bytes.Buffer
	if obtenido := run(args, &out, &log); obtenido != codigo {
		t.Fatalf("código obtenido=%d esperado=%d: %s", obtenido, codigo, &out)
	}
	var resultado application.ResultadoCheckpointDesarrollo
	if err := json.Unmarshal(out.Bytes(), &resultado); err != nil {
		t.Fatal(err)
	}
	for _, clave := range []string{"consumos_historicos_sin_fecha_ligada", "fecha_consumo_ligada_cotejada"} {
		if !strings.Contains(out.String(), `"`+clave+`":`) {
			t.Fatalf("indicador omitido: %s", clave)
		}
	}
	return resultado
}

func TestCheckpointMixtoHistoricoV2MantieneAviso(t *testing.T) {
	_, args, _ := checkpointConCadena(t, "testdata/cadena_mixta_v2.json")
	r := resultadoMixto(t, args, 0)
	if r.IntegridadCadena != "verificada" || !r.ConsumosHistoricosSinFechaLigada || r.FechaConsumoLigadaCotejada {
		t.Fatalf("alcance histórico incorrecto: %+v", r)
	}
}

func TestCheckpointMixtoFirmaPinYFechasHistoricas(t *testing.T) {
	_, args, _ := checkpointMixto(t)
	r := resultadoMixto(t, args, 0)
	if r.Firma != "verificada_con_pin_externo" || r.IntegridadCadena != "verificada" ||
		!r.ConsumosHistoricosSinFechaLigada || !r.FechaConsumoLigadaCotejada ||
		r.OrigenExtraccion != "no_acreditado" || r.TSA != "no_verificada_offline" || r.TiempoIndependiente || r.FirmaLegal {
		t.Fatalf("alcance mixto incorrecto: %+v", r)
	}
	// Sin cadena ambos indicadores siguen explícitos, sin cotejo temporal.
	r = resultadoMixto(t, args[:len(args)-2], 0)
	if r.IntegridadCadena != "no_evaluada" || r.ConsumosHistoricosSinFechaLigada || r.FechaConsumoLigadaCotejada {
		t.Fatalf("sin cadena: %+v", r)
	}
	var out, log bytes.Buffer
	args[len(args)-3] = strings.Repeat("0", 64)
	if run(args, &out, &log) == 0 {
		t.Fatal("pin ajeno admitido con cadena correcta")
	}
}

func TestCheckpointMixtoFamiliasIntegradas(t *testing.T) {
	for _, ruta := range []string{
		"../vec-auditoria-verificar/testdata/fuentes_iniciales_ad174.json",
		"../vec-auditoria-verificar/testdata/unidad_inicial_ad176.json",
		"../vec-auditoria-verificar/testdata/bootstrap_intentos_ad179.json",
	} {
		t.Run(filepath.Base(ruta), func(t *testing.T) {
			_, args, _ := checkpointConCadena(t, ruta)
			r := resultadoMixto(t, args, 0)
			if r.Firma != "verificada_con_pin_externo" || r.IntegridadCadena != "verificada" ||
				r.OrigenExtraccion != "no_acreditado" || r.TSA != "no_verificada_offline" || r.TiempoIndependiente || r.FirmaLegal {
				t.Fatalf("familia integrada: %+v", r)
			}
		})
	}
}

func TestCheckpointMixtoRechazaAlterarQuinceCampos(t *testing.T) {
	b, args, ruta := checkpointMixto(t)
	for _, campo := range []string{"tipo_registro", "version_consumo", "secuencia", "anterior_sha256", "decision_ref", "efecto_ref", "huella_efecto_sha256", "consumo_huella_sha256", "proceso", "canal", "registrada_en", "consumida_en", "actor_ref", "perfil_activo_ref", "finalidad_ref"} {
		t.Run(campo, func(t *testing.T) {
			var d map[string]any
			if json.Unmarshal(b, &d) != nil {
				t.Fatal("fixture inválida")
			}
			c := d["registros"].([]any)[4].(map[string]any)["consumo"].(map[string]any)
			switch campo {
			case "version_consumo", "secuencia":
				c[campo] = float64(4)
			case "tipo_registro":
				c[campo] = auditoria.TipoConsumoOrigenV2
			case "anterior_sha256", "huella_efecto_sha256", "consumo_huella_sha256":
				c[campo] = strings.Repeat("e", 64)
			case "registrada_en", "consumida_en":
				c[campo] = "2026-10-03T12:34:56.123457Z"
			case "canal":
				c[campo] = "externa_personal"
			default:
				c[campo] = "alteracion_privada_sintetica"
			}
			alterada, err := json.Marshal(d)
			if err != nil || os.WriteFile(ruta, alterada, 0600) != nil {
				t.Fatal("no se escribió alteración")
			}
			r := resultadoMixto(t, args, 1)
			if r.Firma != "verificada_con_pin_externo" || r.IntegridadCadena != "rechazada" || r.ConsumosHistoricosSinFechaLigada || r.FechaConsumoLigadaCotejada {
				t.Fatalf("rechazo o aviso histórico incorrecto: %+v", r)
			}
		})
	}
	// Ambas fechas válidas e iguales, con un microsegundo distinto: la forma
	// sigue siendo válida y el eslabón de quince campos debe detectarlo.
	alterada := bytes.ReplaceAll(b, []byte("2026-10-03T12:34:56.123456Z"), []byte("2026-10-03T12:34:56.123457Z"))
	if os.WriteFile(ruta, alterada, 0600) != nil {
		t.Fatal("no se escribió alteración temporal")
	}
	r := resultadoMixto(t, args, 1)
	if r.IntegridadCadena != "rechazada" || r.ConsumosHistoricosSinFechaLigada || r.FechaConsumoLigadaCotejada {
		t.Fatalf("fechas cambiadas admitidas: %+v", r)
	}
}

func TestCheckpointMixtoParseoCerrado(t *testing.T) {
	b, args, ruta := checkpointMixto(t)
	for _, caso := range []struct {
		nombre string
		mutar  func(map[string]any, map[string]any)
	}{
		{"null", func(d, c map[string]any) { c["actor_ref"] = nil }},
		{"ausente", func(d, c map[string]any) { delete(c, "consumida_en") }},
		{"desconocida", func(d, c map[string]any) { c["campo_ajeno"] = "privado" }},
		{"alias", func(d, c map[string]any) { c["ACTOR_REF"] = c["actor_ref"]; delete(c, "actor_ref") }},
		{"union", func(d, c map[string]any) { d["registros"].([]any)[4].(map[string]any)["intento"] = map[string]any{} }},
		{"tipo_futuro", func(d, c map[string]any) {
			d["registros"].([]any)[4].(map[string]any)["tipo_registro"] = "consumo_confirmado_v4"
		}},
		{"fecha_historica", func(d, c map[string]any) {
			d["registros"].([]any)[0].(map[string]any)["consumo"].(map[string]any)["registrada_en"] = c["registrada_en"]
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var d map[string]any
			if json.Unmarshal(b, &d) != nil {
				t.Fatal("fixture inválida")
			}
			c := d["registros"].([]any)[4].(map[string]any)["consumo"].(map[string]any)
			caso.mutar(d, c)
			alterada, _ := json.Marshal(d)
			if os.WriteFile(ruta, alterada, 0600) != nil {
				t.Fatal("no se escribió alteración")
			}
			var out, log bytes.Buffer
			if run(args, &out, &log) != 1 || strings.Contains(out.String(), "privado") {
				t.Fatalf("entrada ambigua admitida o reflejada: %s", &out)
			}
		})
	}
	for _, alterada := range [][]byte{
		bytes.Replace(b, []byte(`"actor_ref":`), []byte(`"actor_ref":"duplicada","actor_ref":`), 1),
		append(append([]byte(nil), b...), []byte(` {}`)...),
		[]byte(strings.Repeat("[", 18) + "0" + strings.Repeat("]", 18)),
	} {
		if verificarEntrada := decodificar(alterada, new(auditoria.DocumentoVerificacionMixta)); verificarEntrada == nil {
			t.Fatal("duplicada, concatenación o profundidad excesiva admitida")
		}
	}
}
