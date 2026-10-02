package ensayofisicopg

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func tarMuestra(t *testing.T, hs []*tar.Header) Archivo {
	t.Helper()
	p := filepath.Join(t.TempDir(), "muestra.tar")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	w := tar.NewWriter(f)
	for _, h := range hs {
		if err = w.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if h.Size > 0 {
			if _, err = w.Write(make([]byte, h.Size)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if w.Close() != nil || f.Close() != nil {
		t.Fatal("cerrar tar")
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256(b)
	return Archivo{p, hex.EncodeToString(sha[:])}
}
func TestTarRechazaCrucesDeFrontera(t *testing.T) {
	base := func() *tar.Header { return &tar.Header{Name: "contenido/", Typeflag: tar.TypeDir, Mode: 0700} }
	casos := map[string][]*tar.Header{
		"traversal":     {base(), {Name: "contenido/../escapa", Typeflag: tar.TypeReg, Mode: 0600}},
		"absoluta":      {base(), {Name: "/contenido/absoluta", Typeflag: tar.TypeReg, Mode: 0600}},
		"symlink":       {base(), {Name: "contenido/enlace", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0600}},
		"hardlink":      {base(), {Name: "contenido/enlace", Typeflag: tar.TypeLink, Linkname: "contenido/dato", Mode: 0600}},
		"fifo":          {base(), {Name: "contenido/fifo", Typeflag: tar.TypeFifo, Mode: 0600}},
		"duplicada":     {base(), {Name: "contenido/dato", Typeflag: tar.TypeReg, Mode: 0600}, {Name: "contenido/dato", Typeflag: tar.TypeReg, Mode: 0600}},
		"padre_regular": {base(), {Name: "contenido/dato", Typeflag: tar.TypeReg, Mode: 0600}, {Name: "contenido/dato/hijo", Typeflag: tar.TypeReg, Mode: 0600}},
		"setuid":        {base(), {Name: "contenido/bin", Typeflag: tar.TypeReg, Mode: 04700}},
	}
	c := Configuracion{LimiteEntradas: 10, LimiteExtraidoBytes: 100}
	for n, hs := range casos {
		t.Run(n, func(t *testing.T) {
			a := tarMuestra(t, hs)
			if revisarTar(context.Background(), a.Ruta, c, &cuentaTar{}) == nil {
				t.Fatal("tar no admitido aceptado")
			}
		})
	}
	t.Run("limite_acumulado", func(t *testing.T) {
		a := tarMuestra(t, []*tar.Header{base(), {Name: "contenido/dato", Typeflag: tar.TypeReg, Mode: 0600, Size: 80}})
		cuenta := &cuentaTar{}
		if revisarTar(context.Background(), a.Ruta, c, cuenta) != nil {
			t.Fatal("primer tar válido")
		}
		if revisarTar(context.Background(), a.Ruta, c, cuenta) == nil {
			t.Fatal("segundo tar supera total")
		}
	})
	t.Run("huella_incorrecta", func(t *testing.T) {
		a := tarMuestra(t, []*tar.Header{base()})
		a.SHA256 = string(make([]byte, 64))
		if copiarArchivo(context.Background(), a, filepath.Join(t.TempDir(), "privada.tar"), 1<<20) == nil {
			t.Fatal("SHA distinta aceptada")
		}
	})
	t.Run("cancelacion", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		a := tarMuestra(t, []*tar.Header{base()})
		if revisarTar(ctx, a.Ruta, c, &cuentaTar{}) == nil {
			t.Fatal("contexto cancelado aceptado")
		}
	})
}
