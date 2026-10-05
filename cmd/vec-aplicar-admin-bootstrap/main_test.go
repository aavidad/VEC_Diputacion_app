package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func planFixture(t *testing.T) documento {
	t.Helper()
	b, e := os.ReadFile("testdata/plan.sintetico.json")
	if e != nil {
		t.Fatal(e)
	}
	var p documento
	if decodificarEstricto(b, &p) != nil {
		t.Fatal("fixture")
	}
	return p
}
func respuestaFixture(t *testing.T, p documento, estado string, replay bool) []byte {
	t.Helper()
	hex := func(prefix, c string) string { return prefix + strings.Repeat(c, 32) }
	e := envoltura{Estado: estado, Replay: replay, AuditoriaIntento: auditoriaIntento{hex("aud_v3_bi_", "a"), 3, strings.Repeat("e", 64), hex("correlacion_", "a"), "2026-10-04T00:00:00.123456Z"}}
	if estado == "permitido" {
		r := &reciboBootstrap{ActoRef: hex("acto_admin:", "b"), ReciboRef: hex("recibo_bootstrap:", "c"), HuellaPlanSHA256: p.HuellaPlanSHA256, PrimeraPersonaRef: p.Plan.Personas[0].PersonaRef, SegundaPersonaRef: p.Plan.Personas[1].PersonaRef, AuditoriaRef: hex("aud_v3_p_", "d"), ConfirmadoEn: "2026-10-04T00:00:00.123456Z"}
		i := 0
		for _, persona := range p.Plan.Personas {
			r.Perfiles[i] = perfilBootstrap{persona.PersonaRef, persona.CuentaRef, persona.PerfilRef, persona.VinculoRef, hex("asignacion:bootstrap_", string(rune('a'+i))) + ":v1", p.Plan.Rol.VersionRef}
			i++
			for _, sistema := range persona.Sistemas {
				r.Perfiles[i] = perfilBootstrap{persona.PersonaRef, persona.CuentaRef, sistema.PerfilRef, sistema.VinculoRef, hex("asignacion:bootstrap_", string(rune('a'+i))) + ":v1", sistema.Rol.VersionRef}
				i++
			}
		}
		e.Recibo = r
	} else {
		codigo := "bootstrap_rechazado"
		if estado == "error" {
			codigo = "bootstrap_no_disponible"
		}
		e.Codigo = &codigo
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type txFixture struct {
	respuesta  []byte
	pasos      []string
	fallo      string
	aprobacion string
}

func (t *txFixture) PrepararUTC(context.Context) error { t.pasos = append(t.pasos, "utc"); return nil }
func (t *txFixture) Provisionar(_ context.Context, _ string, a string) ([]byte, error) {
	t.pasos = append(t.pasos, "consulta")
	t.aprobacion = a
	return append([]byte{}, t.respuesta...), nil
}
func (t *txFixture) Confirmar(context.Context) error {
	t.pasos = append(t.pasos, "commit")
	if t.fallo == "commit" {
		return errors.New("error_privado")
	}
	return nil
}
func (t *txFixture) Cerrar(context.Context) { t.pasos = append(t.pasos, "cerrar") }
func argsFixture(t *testing.T) ([]string, string, documento) {
	t.Helper()
	p := planFixture(t)
	dir := filepath.Join(t.TempDir(), "privado")
	if os.Mkdir(dir, 0700) != nil {
		t.Fatal("directorio")
	}
	put := func(name string, v any) string {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		path := filepath.Join(dir, name)
		if os.WriteFile(path, b, 0600) != nil {
			t.Fatal("archivo")
		}
		return path
	}
	plan := put("plan.json", p)
	cfg := put("conexion.json", conexionPrivada{DSN: "host=/tmp dbname=sintetica user=operador sslmode=disable", PermitirSocketDesarrollo: true})
	a := put("aprobacion.json", aprobacionPrivada{strings.Repeat("f", 64)})
	ack := filepath.Join(dir, "acuse.json")
	textos, e := filepath.Abs("../../web/static/textos/es/admin-bootstrap-aplicar.json")
	if e != nil {
		t.Fatal(e)
	}
	return []string{"--plan", plan, "--conexion", cfg, "--aprobacion", a, "--acuse", ack, "--textos", textos, "--timeout", "5s"}, ack, p
}
func TestBootstrapCommitTresEstadosAcuseOriginal(t *testing.T) {
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			args, path, p := argsFixture(t)
			tx := &txFixture{respuesta: respuestaFixture(t, p, estado, estado == "permitido")}
			llamadas := 0
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
				llamadas++
				b, e := os.ReadFile(path)
				if e != nil || len(b) != 0 {
					t.Fatal("acuse no reservado")
				}
				return tx, nil
			}
			var salida, errores bytes.Buffer
			code := ejecutar(args, &salida, &errores, abrir)
			esperado := 0
			if estado != "permitido" {
				esperado = 1
			}
			if code != esperado || errores.Len() != 0 || !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "commit", "cerrar"}) {
				t.Fatal("protocolo", code, tx.pasos)
			}
			b, e := os.ReadFile(path)
			if e != nil || !bytes.Equal(b, append(append([]byte{}, tx.respuesta...), '\n')) {
				t.Fatal("acuse alterado")
			}
			if tx.aprobacion != strings.Repeat("f", 64) {
				t.Fatal("sustituye aprobaciónexterna")
			}
			if strings.Contains(salida.String(), "persona_ref") || strings.Contains(salida.String(), "perfil_ref") {
				t.Fatal("expone contenido")
			}
			if ejecutar(args, &salida, &errores, abrir) == 0 || llamadas != 1 {
				t.Fatal("reenvía destino existente")
			}
		})
	}
}
func TestBootstrapCommitInciertoNoAcuse(t *testing.T) {
	args, path, p := argsFixture(t)
	tx := &txFixture{respuesta: respuestaFixture(t, p, "permitido", false), fallo: "commit"}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, abrir) != 2 || salida.Len() != 0 {
		t.Fatal("éxito incierto")
	}
	var d diagnostico
	if json.Unmarshal(errores.Bytes(), &d) != nil || d.Estado != "indeterminado" || d.Confirmado {
		t.Fatal("no indica incertidumbre")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("acuse no confirmado")
	}
}
func TestBootstrapCatalogoAusenteSinDB(t *testing.T) {
	args, path, _ := argsFixture(t)
	args[9] = filepath.Join(filepath.Dir(path), "catalogo-ausente.json")
	calls := 0
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { calls++; return nil, nil }
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, abrir) != 2 || calls != 0 || errores.String() != "{\"codigo\":\"catalogo_no_disponible\"}\n" {
		t.Fatal("catálogo silencioso/abreDB")
	}
}
func TestBootstrapEnvolturaAjenaNoCommit(t *testing.T) {
	args, _, p := argsFixture(t)
	b := respuestaFixture(t, p, "permitido", false)
	b = bytes.ReplaceAll(b, []byte(p.Plan.Personas[0].PersonaRef), []byte("per_"+strings.Repeat("x", 32)))
	tx := &txFixture{respuesta: b}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, abrir) != 1 || !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "cerrar"}) {
		t.Fatal("acepta persona ajena")
	}
}
