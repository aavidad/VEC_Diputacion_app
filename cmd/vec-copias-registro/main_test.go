package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fsregistro "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

func cliConfig(t *testing.T) string {
	t.Helper()
	control, restaurado := t.TempDir(), t.TempDir()
	if os.Chmod(control, 0700) != nil {
		t.Fatal("chmod")
	}
	c := fsregistro.Config{Directorio: control, RaicesRestauradas: []string{restaurado}, LimiteListado: 5}
	b, _ := json.Marshal(c)
	p := filepath.Join(t.TempDir(), "config.json")
	if os.WriteFile(p, b, 0600) != nil {
		t.Fatal("write")
	}
	return p
}
func cli(t *testing.T, config, accion, idioma string, v any) (int, []byte, []byte) {
	t.Helper()
	b, _ := json.Marshal(v)
	var out, err bytes.Buffer
	code := ejecutar([]string{"-config", config, "-accion", accion, "-idioma", idioma, "-textos", "../../web/static/textos/" + idioma + "/copias_registro.json"}, bytes.NewReader(b), &out, &err)
	return code, out.Bytes(), err.Bytes()
}
func TestCLIReservaAplicarReabrirReplayYListar(t *testing.T) {
	c := cliConfig(t)
	d := port.Declaracion{Actor: "actor:sintetico", Correlacion: "correlacion:cli"}
	s := operacionescopias.Solicitud{Operacion: "op:cli", Clave: "clave:cli", SHA256: strings.Repeat("a", 64), Conjunto: "conjunto:cli", Destino: "destino:cli", Politica: "politica:cli"}
	code, b, e := cli(t, c, "reservar", "es", entrada{Sintetica: true, Declaracion: d, Solicitud: &s})
	if code != 0 {
		t.Fatal(string(e))
	}
	var original struct {
		Resultado            port.Resultado `json:"resultado"`
		RegistroDurable      bool           `json:"registro_durable"`
		HabilitaRestauracion bool           `json:"habilita_restauracion"`
		Aviso                string         `json:"aviso"`
	}
	if json.Unmarshal(b, &original) != nil || !original.RegistroDurable || original.HabilitaRestauracion || original.Aviso == "" {
		t.Fatal(string(b))
	}
	cmd := operacionescopias.Comando{Clave: "c:cli", VersionEsperada: 0, SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"}
	code, b, e = cli(t, c, "aplicar", "es", entrada{Sintetica: true, Declaracion: d, Operacion: s.Operacion, Comando: &cmd})
	if code != 0 {
		t.Fatal(string(e))
	}
	var aplicado struct {
		Resultado port.Resultado `json:"resultado"`
	}
	if json.Unmarshal(b, &aplicado) != nil {
		t.Fatal(string(b))
	}
	code, b, e = cli(t, c, "aplicar", "en", entrada{Sintetica: true, Declaracion: d, Operacion: s.Operacion, Comando: &cmd})
	if code != 0 {
		t.Fatal(string(e))
	}
	var replay struct {
		Resultado port.Resultado `json:"resultado"`
		Aviso     string         `json:"aviso"`
	}
	if json.Unmarshal(b, &replay) != nil || !replay.Resultado.Replay || replay.Resultado.Recibo != aplicado.Resultado.Recibo || !strings.Contains(replay.Aviso, "synthetic") {
		t.Fatal(string(b))
	}
	code, b, e = cli(t, c, "consultar", "es", entrada{Sintetica: true, Declaracion: d, Operacion: s.Operacion})
	if code != 0 {
		t.Fatal(string(e))
	}
	var q struct {
		Resultado port.Resultado `json:"resultado"`
	}
	if json.Unmarshal(b, &q) != nil || q.Resultado.Recibo != aplicado.Resultado.Recibo || len(q.Resultado.Historia) != 1 {
		t.Fatal(string(b))
	}
	code, b, e = cli(t, c, "listar", "en", entrada{Sintetica: true, Declaracion: d, Consulta: &port.Consulta{Limite: 1}})
	if code != 0 {
		t.Fatal(string(e))
	}
	if json.Unmarshal(b, &q) != nil || len(q.Resultado.Operaciones) != 1 || q.Resultado.Operaciones[0].Recibo != aplicado.Resultado.Recibo {
		t.Fatal(string(b))
	}
}
func TestCLIRechazoNoSinteticoYConflictoAuditado(t *testing.T) {
	c := cliConfig(t)
	d := port.Declaracion{Actor: "actor:sintetico", Correlacion: "correlacion:cli"}
	s := operacionescopias.Solicitud{Operacion: "op:cli", Clave: "clave:cli", SHA256: strings.Repeat("a", 64), Conjunto: "conjunto:cli", Destino: "destino:cli", Politica: "politica:cli"}
	code, _, e := cli(t, c, "reservar", "es", entrada{Declaracion: d, Solicitud: &s})
	if code != 2 || !bytes.Contains(e, []byte("entrada_invalida")) {
		t.Fatal(code, string(e))
	}
	code, _, e = cli(t, c, "reservar", "es", entrada{Sintetica: true, Declaracion: d, Solicitud: &s})
	if code != 0 {
		t.Fatal(string(e))
	}
	s.Destino = "destino:otro"
	code, _, e = cli(t, c, "reservar", "es", entrada{Sintetica: true, Declaracion: d, Solicitud: &s})
	var r struct {
		Error     string         `json:"error"`
		Auditoria port.Auditoria `json:"auditoria"`
	}
	if code != 2 || json.Unmarshal(e, &r) != nil || r.Error != "operacion_conflicto_idempotencia" || r.Auditoria.Referencia == "" {
		t.Fatal(string(e))
	}
}
func TestStrictJSONAndCatalogues(t *testing.T) {
	for _, s := range []string{`{"sintetica":true,"sintetica":false}`, `{"Sintetica":true}`, `{"sintetica":true,"desconocido":1}`, `{"sintetica":true} {}`, `null`} {
		var e entrada
		if err := decodificar(strings.NewReader(s), &e); err == nil && e.Sintetica {
			t.Fatal("accepted", s)
		}
	}
	if decodificar(strings.NewReader(strings.Repeat(" ", maxBytes+1)), new(entrada)) == nil {
		t.Fatal("oversize")
	}
	var base map[string]string
	for _, lang := range []string{"es", "en"} {
		var m map[string]string
		if leerArchivo("../../web/static/textos/"+lang+"/copias_registro.json", &m) != nil {
			t.Fatal(lang)
		}
		for _, key := range claves {
			if m[key] == "" {
				t.Fatal(lang, key)
			}
		}
		if base == nil {
			base = m
		} else {
			if len(base) != len(m) {
				t.Fatal("catalog mismatch")
			}
			for key := range base {
				if m[key] == "" {
					t.Fatal(key)
				}
			}
		}
	}
}

func TestCLIAbandonoNoSeAutorizaPorDeclaracionNiDatos(t *testing.T) {
	cfg := cliConfig(t)
	d := port.Declaracion{Actor: "actor:administrador_declarado", Correlacion: "correlacion:cli"}
	s := operacionescopias.Solicitud{Operacion: "op:cli", Clave: "clave:cli", SHA256: strings.Repeat("a", 64), Conjunto: "conjunto:cli", Destino: "destino:cli", Politica: "politica:cli"}
	if code, _, err := cli(t, cfg, "reservar", "es", entrada{Sintetica: true, Declaracion: d, Solicitud: &s}); code != 0 {
		t.Fatal(string(err))
	}
	c := operacionescopias.Comando{Clave: "captura:cli", SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"}
	if code, _, err := cli(t, cfg, "aplicar", "es", entrada{Sintetica: true, Declaracion: d, Operacion: s.Operacion, Comando: &c}); code != 0 {
		t.Fatal(string(err))
	}
	c.Clave, c.VersionEsperada, c.Accion = "abandono:cli", 1, "abandonar_captura"
	c.Abandono = &operacionescopias.ObservacionAbandono{Operacion: s.Operacion, Destino: s.Destino, FalloReferencia: "captura_fallida", FalloSHA256: s.SHA256, Lease: "lease:cli", EstadoEfecto: "inactivo", EstadoLease: "cancelada", EstadoPlataforma: "sin_efectos_pendientes"}
	if code, _, err := cli(t, cfg, "aplicar", "es", entrada{Sintetica: true, Declaracion: d, Operacion: s.Operacion, Comando: &c}); code != 2 || !bytes.Contains(err, []byte("registro_entrada_invalida")) {
		t.Fatal("free abort", code, string(err))
	}
	code, b, err := cli(t, cfg, "consultar", "es", entrada{Sintetica: true, Declaracion: d, Operacion: s.Operacion})
	var r struct {
		Resultado port.Resultado `json:"resultado"`
	}
	if code != 0 || json.Unmarshal(b, &r) != nil || r.Resultado.Recibo.Estado != operacionescopias.Capturando || r.Resultado.Recibo.Version != 1 {
		t.Fatal("state changed", code, string(err))
	}
}
