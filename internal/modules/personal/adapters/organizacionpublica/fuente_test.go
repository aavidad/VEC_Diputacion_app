package organizacionpublica

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFuenteEstructuraPublicaInmovilizaPaqueteYRechazaSustitucionAlConstruir(t *testing.T) {
	origen := filepath.Join("..", "..", "..", "..", "..", "data", "catalogos", "estructura-organizativa", "v1.rpt-publica.json")
	contenido, err := os.ReadFile(origen)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "estructura.json")
	if err := os.WriteFile(ruta, contenido, 0o600); err != nil {
		t.Fatal(err)
	}
	fuente, err := NuevaFuente(ruta)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := fuente.ObtenerEstructuraOrganizativaPublica(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Fuente.HuellaSHA256 != HuellaPaqueteEstructura2026 || len(resultado.Unidades) != 66 {
		t.Fatalf("proyeccion invalida: %#v", resultado)
	}
	if err := os.WriteFile(ruta, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	repetida, err := fuente.ObtenerEstructuraOrganizativaPublica(context.Background())
	if err != nil || repetida.Fuente.HuellaSHA256 != resultado.Fuente.HuellaSHA256 || len(repetida.Unidades) != len(resultado.Unidades) {
		t.Fatalf("la sustitucion altero la instantanea: %#v %v", repetida, err)
	}
	if _, err := NuevaFuente(ruta); err == nil {
		t.Fatal("acepto paquete sustituido al construir")
	}
}
