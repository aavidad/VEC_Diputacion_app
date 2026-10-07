package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	domain "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
)

func fixture() entrada {
	return entrada{Sintetica: true, Instante: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), Pasos: []string{"verificar", "aceptar", "replay"}, Orden: domain.Datos{Orden: "orden:uno", Operacion: "operacion:uno", Accion: "restaurar_conjunto", SolicitudSHA256: strings.Repeat("a", 64), Conjunto: "conjunto:uno", ManifiestoSHA256: strings.Repeat("b", 64), PreimagenSHA256: strings.Repeat("c", 64), Destino: "destino:uno", ProponentePersona: "persona:uno", AprobadorPersona: "persona:dos", Politica: "politica:uno", PoliticaSHA256: strings.Repeat("d", 64), DecisionV3: "decision:uno", DecisionSHA256: strings.Repeat("e", 64), ConsumoV3: strings.Repeat("f", 64), Auditoria: "auditoria:uno", Outbox: "outbox:uno", Epoca: "epoca:uno", Fence: 1, VersionCAS: 1, EmitidaEn: time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC), CaducaEn: time.Date(2026, 10, 1, 11, 0, 0, 0, time.UTC)}}
}
func TestCLIFirmaAceptacionReplayYLimites(t *testing.T) {
	e := fixture()
	b, _ := json.Marshal(e)
	var out, diag bytes.Buffer
	args := argumentos(t)
	if code := ejecutar(args, bytes.NewReader(b), &out, &diag); code != 0 {
		t.Fatal(code, diag.String())
	}
	var r struct {
		Durable bool   `json:"aceptacion_durable"`
		V3      bool   `json:"consumo_v3_acreditado"`
		Efecto  bool   `json:"efecto_plataforma"`
		Pasos   []paso `json:"pasos"`
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Durable || r.V3 || r.Efecto || len(r.Pasos) != 3 || !r.Pasos[2].Replay || r.Pasos[1].Recibo != r.Pasos[2].Recibo {
		t.Fatal(out.String())
	}
	for _, mutate := range []func(*entrada){func(e *entrada) { e.Sintetica = false }, func(e *entrada) { e.Instante = e.Orden.CaducaEn }, func(e *entrada) { e.Pasos = []string{"replay"} }, func(e *entrada) { e.Pasos = []string{"restaurar"} }} {
		e := fixture()
		mutate(&e)
		b, _ := json.Marshal(e)
		out.Reset()
		diag.Reset()
		if code := ejecutar(args, bytes.NewReader(b), &out, &diag); code != 2 || out.Len() != 0 {
			t.Fatal(code, out.String())
		}
	}
}
func TestJSONClavesDuplicadasDesconocidasTamanosYCatalogos(t *testing.T) {
	for _, s := range []string{`{"sintetica":true,"sintetica":false}`, `{"Sintetica":true}`, `{"sintetica":true,"token":"secreto"}`, `{} {}`, strings.Repeat("x", maxBytes+1)} {
		var e entrada
		if decodificar(strings.NewReader(s), &e) == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
	for _, lang := range []string{"es", "en"} {
		b, err := os.ReadFile("../../web/static/textos/" + lang + "/copias_orden.json")
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]string
		if decodificar(bytes.NewReader(b), &m) != nil {
			t.Fatal(lang)
		}
		if _, err := catalogo("../../web/static/textos/" + lang + "/copias_orden.json"); err != nil {
			t.Fatal(lang, err)
		}
	}
}

func argumentos(t *testing.T) []string {
	t.Helper()
	dir := t.TempDir()
	keyfile := filepath.Join(dir, "maestra-sintetica.bin")
	dummy := make([]byte, 32)
	dummy[0] = 1
	if os.WriteFile(keyfile, dummy, 0600) != nil {
		t.Fatal("key fixture")
	}
	cfg := configuracion{ClaveFichero: keyfile, ClaveRef: "clave:sintetica:orden", ClaveVersion: "version:1"}
	b, _ := json.Marshal(cfg)
	cfgfile := filepath.Join(dir, "config.json")
	if os.WriteFile(cfgfile, b, 0600) != nil {
		t.Fatal("config fixture")
	}
	return []string{"--textos", "../../web/static/textos/es/copias_orden.json", "--config", cfgfile}
}
