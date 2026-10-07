package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoagregadosincidencias"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func huellaCLI(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func argumentosTest(t *testing.T, lang string) ([]string, []byte) {
	t.Helper()
	s := "../../data/demo/cronos/agregados-incidencias-periodo.json"
	tx := "../../web/static/textos/" + lang + "/cronos-agregados-incidencias-ensayo.json"
	b, e := os.ReadFile(s)
	if e != nil {
		t.Fatal(e)
	}
	bt, e := os.ReadFile(tx)
	if e != nil {
		t.Fatal(e)
	}
	return []string{"--snapshot", s, "--snapshot-sha256", huellaCLI(b), "--textos", tx, "--textos-sha256", huellaCLI(bt), "--idioma", lang}, b
}

func TestCLIAgregadosESENYStdin(t *testing.T) {
	for _, lang := range []string{"es", "en"} {
		t.Run(lang, func(t *testing.T) {
			args, b := argumentosTest(t, lang)
			var out, diag bytes.Buffer
			if correr(args, nil, &out, &diag) != 0 || diag.Len() != 0 {
				t.Fatal(diag.String())
			}
			var r struct {
				ports.ResultadoEnsayoAgregadosIncidencias
				Idioma string            `json:"idioma"`
				Textos map[string]string `json:"textos"`
			}
			if json.Unmarshal(out.Bytes(), &r) != nil || r.Idioma != lang || r.Agregado.IncidenciasObservadas != 3 || r.Agregado.TotalIncidencias != nil || r.Textos["titulo"] == "" {
				t.Fatal(out.String())
			}
			if strings.Contains(out.String(), "demo:persona:") || strings.Contains(out.String(), "demo:incidencia:") {
				t.Fatal("individual references escaped")
			}
			esperado := out.String()
			out.Reset()
			args[1] = "-"
			if correr(args, bytes.NewReader(b), &out, &diag) != 0 || out.String() != esperado {
				t.Fatal("stdin differs")
			}
		})
	}
}

func TestCLIAgregadosNegativosNoCero(t *testing.T) {
	for _, mutar := range []func([]string) []string{
		func(a []string) []string { return nil },
		func(a []string) []string { a[3] = strings.Repeat("0", 64); return a },
		func(a []string) []string { a[7] = strings.Repeat("0", 64); return a },
		func(a []string) []string { a[9] = "other"; return a },
		func(a []string) []string { a[1] = "/nonexistent-demo-fixture"; return a },
		func(a []string) []string { return append(a, "--unknown") },
		func(a []string) []string { return append(a, "extra") },
	} {
		args, _ := argumentosTest(t, "es")
		var out, diag bytes.Buffer
		if correr(mutar(args), nil, &out, &diag) != 2 || out.Len() != 0 || strings.TrimSpace(diag.String()) != `{"error":"entrada_invalida","estado":"desconocido","total_incidencias":null}` {
			t.Fatalf("unexpected diagnostic: %q %q", out.String(), diag.String())
		}
	}
}

type falloIO struct{}

func (falloIO) Read([]byte) (int, error)  { return 0, io.ErrUnexpectedEOF }
func (falloIO) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestCLIAgregadosIOYCancelacion(t *testing.T) {
	args, b := argumentosTest(t, "es")
	if correr(args, nil, falloIO{}, io.Discard) != 2 {
		t.Fatal("write failure hidden")
	}
	args[1] = "-"
	for _, r := range []io.Reader{nil, falloIO{}, bytes.NewReader(bytes.Repeat([]byte(" "), catalogoagregadosincidencias.LimiteJSON+1))} {
		var out bytes.Buffer
		if correr(args, r, &out, falloIO{}) != 2 || out.Len() != 0 {
			t.Fatal("read failure hidden")
		}
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	var out bytes.Buffer
	if ejecutar(ctx, args, bytes.NewReader(b), &out) == nil || out.Len() != 0 {
		t.Fatal("cancelled CLI produced result")
	}
}

func TestCLIAgregadosArchivosRegulares(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular.json")
	if err := os.WriteFile(regular, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(regular, link); err != nil {
		t.Fatal(err)
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{link, fifo, dir} {
		if _, err := leerArchivo(p); err == nil {
			t.Fatal("accepted non-regular input")
		}
	}
	if _, err := leerArchivo(regular); err != nil {
		t.Fatal(err)
	}
}
