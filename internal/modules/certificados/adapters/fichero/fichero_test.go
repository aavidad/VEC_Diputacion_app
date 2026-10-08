package fichero

import (
	"context"
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

func TestFuenteDeEnsayoNoAdmiteLaFormaV1EscritaAMano(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "v1.json")
	contenido := `{"esquema":"vec.certificados.fuente-servicios.personal-v1.ensayo","sintetica":true,"procedencia_ref":"ensayo:x","nombre":"Elena Martín Robles",` +
		`"corte":{"vigente_en":"2026-10-01","conocido_en":"2026-10-01T08:00:00Z"},"cobertura":"completa",` +
		`"servicios":[{"inicio":"2024-01-01","fin":"2024-02-01","clase":"c","clase_version":1,"estado":"reconocido","certeza":"acreditado","servicio_ref":"s:1","acto_ref":"a:1"}]}`
	if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := (FuenteServicios{Ruta: ruta}).Obtener(context.Background()); err == nil {
		t.Fatal("la forma V1 entró sin pasar por el traductor")
	}
}
