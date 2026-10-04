package ejecucioncopias

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
)

func artefactoPrueba(id, tipo string, b []byte) copias.Artefacto {
	h := sha256.Sum256(b)
	return copias.Artefacto{ID: id, Tipo: tipo, SHA256: hex.EncodeToString(h[:]), TamanoBytes: int64(len(b))}
}
func TestMaterialSóloEntregaCapturaYDetectaCambio(t *testing.T) {
	raiz := t.TempDir()
	fisica := filepath.Join(raiz, "fisica")
	logica := filepath.Join(raiz, "logica")
	for _, r := range []string{fisica, logica} {
		if err := os.Mkdir(r, 0700); err != nil {
			t.Fatal(err)
		}
	}
	a := artefactoPrueba("fisica:pgdata", "base_fisica", []byte("tar sintetico"))
	dump := artefactoPrueba("base:sintetica", "postgresql_logico", []byte("dump sintetico"))
	inv := copias.Inventario{Ref: "inventario:sintetico"}
	idx := indiceFisico{FormatoVersion: 1, Estado: "pendiente_cifrado_y_verificacion", InventarioSHA256: copias.HuellaInventario(inv)}
	idx.Entradas = append(idx.Entradas, struct {
		Archivo   string           `json:"archivo"`
		Artefacto copias.Artefacto `json:"artefacto"`
	}{"componente-0000.tar", a})
	b, _ := json.Marshal(idx)
	for path, data := range map[string][]byte{filepath.Join(fisica, "indice.json"): b, filepath.Join(fisica, "componente-0000.tar"): []byte("tar sintetico"), filepath.Join(logica, "base-0.dump"): []byte("dump sintetico")} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := &MaterialArchivos{RaizLogica: logica, DirectorioFisico: func() string { return fisica }, Bases: []string{"base:sintetica"}, LimiteBytes: 4096}
	salida, err := m.Registrar(context.Background(), "conjunto:1", []copias.Artefacto{a, dump}, inv)
	if err != nil || len(salida) != 2 || salida[1].Tipo != "base_logica" {
		t.Fatalf("registro %v %v", salida, err)
	}
	if _, err := m.Leer(context.Background(), "conjunto:1", salida[1]); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logica, "base-0.dump"), []byte("dump cambiado "), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Leer(context.Background(), "conjunto:1", salida[1]); err == nil {
		t.Fatal("material alterado aceptado")
	}
	if _, err := m.Leer(context.Background(), "conjunto:ajeno", salida[0]); err == nil {
		t.Fatal("referencia ajena aceptada")
	}
}
func TestArchivoCapturadoRechazaEnlaceYLimite(t *testing.T) {
	r := t.TempDir()
	if err := os.WriteFile(filepath.Join(r, "material"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("material", filepath.Join(r, "enlace")); err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{"enlace", "../material", "material"} {
		limite := int64(2)
		if a != "material" {
			limite = 20
		}
		if _, err := leerArchivoCapturado(r, a, limite); err == nil {
			t.Fatalf("aceptado %s", a)
		}
	}
}
