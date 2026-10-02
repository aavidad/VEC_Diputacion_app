package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func huellaCLI(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func argsCLI(t *testing.T, idioma string) []string {
	t.Helper()
	snapshot := "../../data/demo/cronos/presencia-equipo.json"
	textos := "../../web/static/textos/" + idioma + "/cronos-presencia-ensayo.json"
	s, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	c, err := os.ReadFile(textos)
	if err != nil {
		t.Fatal(err)
	}
	return []string{"--snapshot", snapshot, "--snapshot-sha256", huellaCLI(s), "--textos", textos, "--textos-sha256", huellaCLI(c), "--idioma", idioma}
}
func TestCLIRealCatalogosYAgregado(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		var salida, diagnostico bytes.Buffer
		if code := correr(argsCLI(t, idioma), &salida, &diagnostico); code != 0 || diagnostico.Len() != 0 {
			t.Fatalf("code=%d %s", code, diagnostico.String())
		}
		var r struct {
			ports.ResultadoEnsayoPresencia
			Idioma string            `json:"idioma"`
			Textos map[string]string `json:"textos"`
		}
		if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if !r.Demostracion || r.Agregado.Total != 6 || r.Agregado.EntradasRegistradas != 1 || r.Agregado.PausasRegistradas != 1 || r.Agregado.SalidasRegistradas != 1 || r.Agregado.Indeterminado != 3 || len(r.Personas) != 6 {
			t.Fatalf("%+v", r)
		}
		if r.Idioma != idioma {
			t.Fatal("wrong output locale")
		}
		motivos := map[domain.CausaPresencia]bool{}
		for _, p := range r.Personas {
			if p.Estado == domain.PresenciaIndeterminada {
				if p.Motivo == "" || r.Textos[string(p.Motivo)] == "" {
					t.Fatalf("undetermined state lacks translated reason: %+v", p)
				}
				motivos[p.Motivo] = true
			} else if p.Motivo != "" {
				t.Fatalf("determined state has reason: %+v", p)
			}
		}
		if len(motivos) != 3 || !motivos[domain.CoberturaIncompleta] || !motivos[domain.SinMarcajes] || !motivos[domain.SecuenciaAmbigua] {
			t.Fatalf("missing reasons: %+v", motivos)
		}
		var raw struct {
			Personas []map[string]json.RawMessage `json:"personas"`
		}
		if err := json.Unmarshal(salida.Bytes(), &raw); err != nil {
			t.Fatal(err)
		}
		for i, p := range raw.Personas {
			_, presente := p["motivo"]
			if presente != (r.Personas[i].Estado == domain.PresenciaIndeterminada) {
				t.Fatalf("reason omission changed: %+v", p)
			}
		}
	}
}
func TestCLIErrorEstableSinEntradaNiStdout(t *testing.T) {
	args := argsCLI(t, "es")
	shaMalo := append([]string{}, args...)
	shaMalo[3] = strings.Repeat("0", 64)
	for _, caso := range [][]string{nil, shaMalo, append(append([]string{}, args...), "--idioma", "inexistente"), append(append([]string{}, args...), "argumento-secreto"), {"--snapshot", "dato-secreto"}} {
		var salida, diagnostico bytes.Buffer
		if code := correr(caso, &salida, &diagnostico); code != 2 || salida.Len() != 0 || diagnostico.String() != "{\"error\":\"entrada_invalida\"}\n" {
			t.Fatalf("code=%d out=%s err=%s", code, salida.String(), diagnostico.String())
		}
	}
	// A correctly hashed snapshot must still reject extra sensitive fields.
	b, err := os.ReadFile(args[1])
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), `"version_esquema": 1`, `"version_esquema": 1, "motivo": "dato-secreto"`, 1))
	p := filepath.Join(t.TempDir(), "snapshot.json")
	if err = os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	args[1] = p
	args[3] = huellaCLI(b)
	var salida, diagnostico bytes.Buffer
	if code := correr(args, &salida, &diagnostico); code != 2 || salida.Len() != 0 || strings.Contains(diagnostico.String(), "secreto") {
		t.Fatal("sensitive source accepted or leaked")
	}
}
func TestCLIArchivosRegularesAcotados(t *testing.T) {
	dir := t.TempDir()
	normal := filepath.Join(dir, "normal")
	if err := os.WriteFile(normal, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(dir, "enlace")
	if err := os.Symlink(normal, enlace); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	grande := filepath.Join(dir, "grande")
	if err := os.WriteFile(grande, bytes.Repeat([]byte("x"), 1<<20+1), 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{enlace, fifo, dir, grande} {
		if _, err := leer(p); err == nil {
			t.Fatal("nonregular/unbounded file accepted")
		}
	}
}
