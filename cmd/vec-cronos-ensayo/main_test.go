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

	"vec-diputacion-granada/internal/modules/cronos/adapters/catalogoefectos"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

func shaCLI(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func TestCLIJSONRealYSinSalidaConEntradaIncompatible(t *testing.T) {
	cat := "../../data/demo/reglas/cronos-efectos-permisos.demo.json"
	esc := "../../data/demo/cronos/escenarios-saldo.json"
	b, err := os.ReadFile(cat)
	if err != nil {
		t.Fatal(err)
	}
	be, err := os.ReadFile(esc)
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"-catalogo", cat, "-catalogo-sha256", shaCLI(b), "-escenarios", esc, "-escenarios-sha256", shaCLI(be), "-idioma"}
	var s ports.SnapshotEnsayoSaldoPermisos
	if err = json.Unmarshal(be, &s); err != nil {
		t.Fatal(err)
	}
	c, err := catalogoefectos.Cargar(b, shaCLI(b))
	if err != nil {
		t.Fatal(err)
	}
	for idioma, textos := range c.Textos {
		var salida bytes.Buffer
		if err = ejecutar(append(append([]string{}, args...), idioma), &salida); err != nil {
			t.Fatal(err)
		}
		var r ports.ResultadoEnsayoSaldoPermisos
		if err = json.Unmarshal(salida.Bytes(), &r); err != nil {
			t.Fatal(err)
		}
		if !r.Demostracion || *r.Escenarios[0].Dias[0].TrabajadosMinutos != *s.Escenarios[0].Dias[0].TrabajadosMinutos || *r.Escenarios[0].Dias[0].PermisoComputadoMinutos != *s.Escenarios[0].Dias[0].Programacion.MinutosPrevistos || !strings.Contains(salida.String(), textos["aviso"]) {
			t.Fatal("resultado sin fuente o aviso")
		}
	}
	for _, caso := range [][]string{nil, append(append([]string{}, args...), "inexistente"), {"-catalogo", "inexistente", "-escenarios", esc, "-idioma", "es"}} {
		var salida bytes.Buffer
		if ejecutar(caso, &salida) == nil || salida.Len() != 0 {
			t.Fatal("salida con entrada incompatible")
		}
	}
	dir := t.TempDir()
	enlace := filepath.Join(dir, "enlace")
	if err = os.Symlink("ausente", enlace); err != nil {
		t.Fatal(err)
	}
	if _, err = leer(enlace); err == nil {
		t.Fatal("enlace admitido")
	}
	fifo := filepath.Join(dir, "fifo")
	if err = syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = leer(fifo); err == nil {
		t.Fatal("FIFO admitido")
	}
}
