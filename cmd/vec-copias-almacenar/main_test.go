package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLISyntheticStoreCheckDeleteAndRedactedErrors(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "destino")
	if e := os.Mkdir(root, 0700); e != nil {
		t.Fatal(e)
	}
	write := func(name string, b []byte) string {
		p := filepath.Join(dir, name)
		if e := os.WriteFile(p, b, 0600); e != nil {
			t.Fatal(e)
		}
		return p
	}
	key := write("key", bytes.Repeat([]byte{1}, 32))
	cfg, _ := json.Marshal(configuracion{Raiz: root, ClaveFichero: key, ClaveRef: "clave:fixture", ClaveVersion: "v1", MaximoClaroBytes: 65536, TiempoMaximoSegundos: 30})
	config := write("config", cfg)
	input := write("contenido", []byte("contenido sintetico"))
	manifest := write("manifiesto", []byte(`{"fixture":true}`))
	textos, e := filepath.Abs("../../web/static/textos/es/copias_destino.json")
	if e != nil {
		t.Fatal(e)
	}
	base := []string{"-config", config, "-sintetico", "-catalogo", textos, "-idioma", "fixture"}
	var out, errors bytes.Buffer
	args := append(append([]string{}, base...), "-accion", "almacenar", "-entrada", input, "-manifiesto", manifest, "-conjunto", "conjunto:fixture", "-componente", "componente:fixture", "-posicion", "1")
	if run(args, &out, &errors) != 0 {
		t.Fatalf("store: %s", errors.String())
	}
	var r resultado
	if e = json.Unmarshal(out.Bytes(), &r); e != nil || r.Referencia == nil || r.Mensaje == "" || r.Mensaje == r.Codigo {
		t.Fatalf("reference: %v", e)
	}
	ref, _ := json.Marshal(r.Referencia)
	refPath := write("ref", ref)
	for _, action := range []string{"comprobar", "borrar"} {
		out.Reset()
		args = append(append([]string{}, base...), "-accion", action, "-referencia", refPath)
		if run(args, &out, &errors) != 0 {
			t.Fatalf("%s: %s", action, errors.String())
		}
		if strings.Contains(out.String(), root) || strings.Contains(out.String(), key) {
			t.Fatal("path exposed")
		}
	}
	out.Reset()
	errors.Reset()
	args = append(append([]string{}, base...), "-accion", "comprobar", "-referencia", refPath)
	if run(args, &out, &errors) == 0 {
		t.Fatal("missing object accepted")
	}
	if strings.Contains(errors.String(), dir) || strings.Contains(errors.String(), key) {
		t.Fatal("path leaked")
	}
}
func TestCLIDeniesWithoutSyntheticFlag(t *testing.T) {
	var out, errors bytes.Buffer
	if run([]string{"-accion", "almacenar"}, &out, &errors) == 0 {
		t.Fatal("synthetic guard missing")
	}
}

func TestCatalogosDestinoTraduceCodigosNominales(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		catalogo, err := cargarCatalogo("../../web/static/textos/"+idioma+"/copias_destino.json", idioma)
		if err != nil {
			t.Fatal(err)
		}
		for _, codigo := range []string{"copias_destino.configuracion", "copias_destino.catalogo", "copias_destino.material", "copias_destino.no_disponible", "copias_destino.existe", "copias_destino.almacenar", "copias_destino.comprobar", "copias_destino.borrar"} {
			if mensaje := mensajeCodigo(catalogo, idioma, codigo); mensaje == "" || mensaje == codigo {
				t.Fatalf("sin traducción: %s %s", idioma, codigo)
			}
		}
	}
}
