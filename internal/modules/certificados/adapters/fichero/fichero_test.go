package fichero

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/certificados/domain"
)

func TestJSONCerradoYAcotado(t *testing.T) {
	for _, b := range []string{
		`{"esquema":"a","esquema":"b"}`,
		`{"nombre":"Elena Martín Robles","Nombre":"Lucía Navarro Moreno"}`,
		`{"secreto":"unwanted"}`,
		`{} {}`,
		`{"servicios":[{"inicio":"a","inicio":"b"}]}`,
		strings.Repeat(" ", LimiteJSON+1),
		`{"nombre":"` + string([]byte{0xff}) + `"}`,
	} {
		ruta := filepath.Join(t.TempDir(), "input.json")
		if e := os.WriteFile(ruta, []byte(b), 0600); e != nil {
			t.Fatal(e)
		}
		var f domain.FuenteServicios
		if LeerJSON(ruta, &f) == nil {
			t.Fatal("unsafe JSON accepted")
		}
	}
}
func TestSymlinkNoSeLee(t *testing.T) {
	d := t.TempDir()
	ruta := filepath.Join(d, "link")
	if e := os.Symlink("testdata/servicios.ensayo.json", ruta); e != nil {
		t.Fatal(e)
	}
	var f domain.FuenteServicios
	if LeerJSON(ruta, &f) == nil {
		t.Fatal("symlink accepted")
	}
}
