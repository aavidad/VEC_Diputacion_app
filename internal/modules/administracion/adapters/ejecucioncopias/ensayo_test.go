package ejecucioncopias

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	fisica "vec-diputacion-granada/internal/modules/administracion/adapters/ensayofisicopg"
)

func TestArchivadosLogicosPreparaCS06SinPGDATAPreservaComplementos(t *testing.T) {
	var contenido bytes.Buffer
	w := tar.NewWriter(&contenido)
	if err := w.WriteHeader(&tar.Header{Name: "contenido", Typeflag: tar.TypeReg, Mode: 0600, Size: 3}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("vec")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "archivado.tar")
	if err := os.WriteFile(ruta, contenido.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	archivo := fisica.Archivo{Ruta: ruta, SHA256: cs03huella(contenido.Bytes())}
	todos := []fisica.Componente{
		{ID: "fisica:pgdata", Tipo: "base_fisica", Tar: archivo},
		{ID: "fisica:almacen:documentos", Tipo: "documentos", Tar: archivo},
		{ID: "fisica:binario", Tipo: "binario", Tar: archivo},
		{ID: "fisica:configuracion", Tipo: "configuracion", Tar: archivo},
	}
	ctx := context.Background()
	if _, _, err := fisica.PrepararArchivados(ctx, t.TempDir(), todos, 1<<20); err == nil {
		t.Fatal("CS06 admitió PGDATA en el cluster lógico")
	}
	complementos := archivadosLogicos(todos)
	preparados, _, err := fisica.PrepararArchivados(ctx, t.TempDir(), complementos, 1<<20)
	if err != nil || len(preparados) != len(todos)-1 {
		t.Fatalf("complementos reales CS06: %v %+v", err, preparados)
	}
	for n, c := range preparados {
		if c.ID != todos[n+1].ID || c.Tipo != todos[n+1].Tipo || c.ContenidoSHA256 != cs03huella([]byte("vec")) {
			t.Fatalf("complemento omitido o modificado: %+v", c)
		}
	}
	if len(todos) != 4 || todos[0].ID != "fisica:pgdata" {
		t.Fatal("el filtro modificó los componentes del ensayo físico")
	}
}
