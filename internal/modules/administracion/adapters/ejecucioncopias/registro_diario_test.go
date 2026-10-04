package ejecucioncopias

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiarioExteriorReabreCASYBloqueaTruncado(t *testing.T) {
	dir, raiz := t.TempDir(), t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	d, err := abrirDiarioExterior(dir, []string{raiz})
	if err != nil {
		t.Fatal(err)
	}
	e := estadoExterior{Ref: "op:prueba", Actor: "actor:prueba", Correlacion: "op:prueba", ConjuntoRef: "conjunto:prueba", DestinoRef: "destino:prueba", PoliticaRef: "politica:prueba", Estado: "solicitada", PreimagenSHA256: "preimagen:prueba"}
	if err = d.reservarRestauracion(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if err = d.actualizar(context.Background(), e.Ref, "anotar", func(v estadoExterior) (estadoExterior, error) { v.Estado = "exclusion_solicitada"; return v, nil }); err != nil {
		t.Fatal(err)
	}
	actual, err := d.leer(context.Background(), e.Ref)
	if err != nil || actual.Version != 1 || actual.Estado != "exclusion_solicitada" {
		t.Fatalf("%+v %v", actual, err)
	}
	if err = os.Truncate(filepath.Join(dir, "restauraciones.confirmadas"), 0); err != nil {
		t.Fatal(err)
	}
	if _, err = d.leer(context.Background(), e.Ref); err == nil {
		t.Fatal("aceptó testigo truncado")
	}
}
