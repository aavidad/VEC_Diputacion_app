package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	config "vec-diputacion-granada/config/auditoriacheckpoint"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestCLIEmiteVerificaYRechazaRaizAjena(t *testing.T) {
	d := t.TempDir()
	master := filepath.Join(d, "kms")
	tsa := filepath.Join(d, "tsa")
	cobertura := filepath.Join(d, "cobertura.json")
	recibo := filepath.Join(d, "recibo.json")
	publica := filepath.Join(d, "publica.der")
	m, ts := make([]byte, 32), make([]byte, 32)
	m[0] = 1
	ts[0] = 2
	defer clear(m)
	defer clear(ts)
	for path, b := range map[string][]byte{master: m, tsa: ts} {
		if e := os.WriteFile(path, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	z := strings.Repeat("0", 64)
	c := domain.CoberturaCheckpoint{CadenaID: "cadena:sintetica", AnteriorSHA256: z, CabezaSHA256: z}
	b, _ := json.Marshal(c)
	_ = os.WriteFile(cobertura, b, 0600)
	var out, log bytes.Buffer
	emitir := []string{"-operacion", "emitir", "-config", "testdata/config.sintetica.json", "-entrada", cobertura, "-salida", recibo, "-spki", publica, "-kms-master", master, "-tsa-secret", tsa}
	if run(emitir, &out, &log) != 0 {
		t.Fatalf("emisión: %s", out.String())
	}
	if !strings.Contains(log.String(), `"componente":"auditoria"`) || !strings.Contains(log.String(), `"resultado":"correcto"`) || strings.Contains(log.String(), master) {
		t.Fatalf("registro técnico: %s", log.String())
	}
	var resultado struct {
		Pin string `json:"pin_spki_sha256"`
	}
	_ = json.Unmarshal(out.Bytes(), &resultado)
	verificar := []string{"-operacion", "verificar", "-config", "testdata/config.sintetica.json", "-entrada", recibo, "-spki", publica, "-pin-spki-sha256", resultado.Pin}
	out.Reset()
	log.Reset()
	if run(verificar, &out, &log) != 0 {
		t.Fatalf("verificación: %s", out.String())
	}
	if !strings.Contains(out.String(), "verificada_con_pin_externo") || !strings.Contains(out.String(), "no_verificada_offline") {
		t.Fatal(out.String())
	}
	cadena := filepath.Join(d, "cadena.json")
	cadenaJSON := `{"esquema":"vec.auditoria.verificacion.v1","manifiesto":` + string(b) + `,"registros":[]}`
	if err := os.WriteFile(cadena, []byte(cadenaJSON), 0600); err != nil {
		t.Fatal(err)
	}
	conCadena := append(append([]string(nil), verificar...), "-cadena", cadena)
	out.Reset()
	if run(conCadena, &out, &log) != 0 || !strings.Contains(out.String(), `"integridad_cadena":"verificada"`) {
		t.Fatalf("cadena: %s", out.String())
	}
	cadenaJSON = strings.Replace(cadenaJSON, "cadena:sintetica", "cadena:otra", 1)
	_ = os.WriteFile(cadena, []byte(cadenaJSON), 0600)
	out.Reset()
	if run(conCadena, &out, &log) == 0 || !strings.Contains(out.String(), `"firma":"verificada_con_pin_externo"`) || !strings.Contains(out.String(), `"integridad_cadena":"rechazada"`) {
		t.Fatalf("cadena alterada: %s", out.String())
	}
	verificar[len(verificar)-1] = strings.Repeat("0", 64)
	out.Reset()
	if run(verificar, &out, &log) == 0 {
		t.Fatal("raíz ajena admitida")
	}
	out.Reset()
	if run(emitir, &out, &log) == 0 {
		t.Fatal("sobrescritura admitida")
	}
}
func TestCLIParseoYLecturaLimitados(t *testing.T) {
	for _, s := range []string{`{"cadena_id":"uno","cadena_id":"dos"}`, `{"desconocida":1}`, `{} {}`, `{"a":{"b":1,"b":2}}`} {
		var c domain.CoberturaCheckpoint
		if decodificar([]byte(s), &c) == nil {
			t.Fatalf("JSON admitido: %s", s)
		}
	}
	d := t.TempDir()
	p := filepath.Join(d, "secreto")
	if e := os.WriteFile(p, make([]byte, 32), 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := leerRegular(p, 32, true); e == nil {
		t.Fatal("secreto público")
	}
	_ = os.Chmod(p, 0600)
	if _, e := leerRegular(p, 31, true); e == nil {
		t.Fatal("límite superado")
	}
	link := filepath.Join(d, "enlace")
	_ = os.Symlink(p, link)
	if _, e := leerRegular(link, 32, true); e == nil {
		t.Fatal("enlace secreto")
	}
	_ = os.Mkdir(filepath.Join(d, ".git"), 0700)
	if _, e := leerRegular(p, 32, true); e == nil {
		t.Fatal("secreto en repositorio")
	}
}

func TestCLIRechazaAliasDeClavesJSON(t *testing.T) {
	casos := []struct {
		nombre, entrada string
		destino         any
	}{
		{"cobertura_duplicada", `{"cadena_id":"cadena:uno","CADENA_ID":"cadena:dos"}`, &domain.CoberturaCheckpoint{}},
		{"cobertura_solo_alias", `{"CADENA_ID":"cadena:uno"}`, &domain.CoberturaCheckpoint{}},
		{"cobertura_escape", `{"cadena_id":"cadena:uno","\u0043ADENA_ID":"cadena:dos"}`, &domain.CoberturaCheckpoint{}},
		{"config_raiz", `{"max_bytes":1024,"MAX_BYTES":2048}`, &config.AuditoriaCheckpointOffline{}},
		{"config_objeto", `{"politica":{"modo":"DESARROLLO"},"POLITICA":{"modo":"DESARROLLO"}}`, &config.AuditoriaCheckpointOffline{}},
		{"config_anidada", `{"politica":{"clave_ref":"clave:uno","CLAVE_REF":"clave:dos"}}`, &config.AuditoriaCheckpointOffline{}},
		{"recibo_raiz", `{"checkpoint":{"esquema":"uno"},"CHECKPOINT":{"esquema":"dos"}}`, &domain.ReciboCheckpointDesarrollo{}},
		{"recibo_anidado", `{"checkpoint":{"politica":{"modo":"DESARROLLO","MODO":"DESARROLLO"}}}`, &domain.ReciboCheckpointDesarrollo{}},
		{"unicode_casefold", `{"checkpoint":{"e\u017fquema":"uno"}}`, &domain.ReciboCheckpointDesarrollo{}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if decodificar([]byte(caso.entrada), caso.destino) == nil {
				t.Fatal("alias JSON admitido")
			}
		})
	}
}
