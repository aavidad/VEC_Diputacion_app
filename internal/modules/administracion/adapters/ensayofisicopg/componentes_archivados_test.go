package ensayofisicopg

import (
	"archive/tar"
	"context"
	"strings"
	"testing"
)

func TestIDsCS05SonReferenciasYNoRutas(t *testing.T) {
	a := tarMuestra(t, []*tar.Header{{Name: "contenido", Typeflag: tar.TypeReg, Mode: 0600}})
	for _, id := range []string{"fisica:almacen:documentos", "fisica:Release.VEC-18", "fisica:" + strings.Repeat("a", 121)} {
		t.Run(id, func(t *testing.T) {
			salida, montajes, err := PrepararArchivados(context.Background(), t.TempDir(), []Componente{{ID: id, Tipo: "documentos", Tar: a}}, 4096)
			if err != nil || len(salida) != 1 || salida[0].ID != id || salida[0].RutaInterna != "/componentes/0000/contenido" || len(montajes) == 0 {
				t.Fatal("CS05 referencia no aceptada", err, salida)
			}
		})
	}
	for _, id := range []string{"fisica:../escape", "fisica:/absoluto", "fisica:" + strings.Repeat("a", 122)} {
		t.Run(id, func(t *testing.T) {
			if _, _, err := PrepararArchivados(context.Background(), t.TempDir(), []Componente{{ID: id, Tipo: "documentos", Tar: a}}, 4096); err == nil {
				t.Fatal("ID no admitido aceptado")
			}
		})
	}
	if _, _, err := PrepararArchivados(context.Background(), t.TempDir(), []Componente{{ID: "fisica:almacen:docs", Tipo: "documentos", Tar: a}, {ID: "fisica:almacen:docs", Tipo: "documentos", Tar: a}}, 4096); err == nil {
		t.Fatal("referencia duplicada aceptada")
	}
}
