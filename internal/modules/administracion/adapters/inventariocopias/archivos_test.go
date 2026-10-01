package inventariocopias

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestHuellaYTamano(t *testing.T) {
	dir := t.TempDir()
	contenido := []byte("binario sintetico")
	if err := os.WriteFile(filepath.Join(dir, "binario"), contenido, 0600); err != nil {
		t.Fatal(err)
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer raiz.Close()
	huella, tamano, err := leerHuella(raiz, "binario", int64(len(contenido)))
	esperada := sha256.Sum256(contenido)
	if err != nil || huella != hex.EncodeToString(esperada[:]) || tamano != int64(len(contenido)) {
		t.Fatalf("huella=%q tamano=%d error=%v", huella, tamano, err)
	}
	huella, tamano, err = leerHuella(raiz, "binario", 1)
	if err != nil || huella != "" || tamano != int64(len(contenido)) {
		t.Fatalf("diferencia de tamano: huella=%q tamano=%d error=%v", huella, tamano, err)
	}
}

func TestNoLeeFueraDeRaizNiEspeciales(t *testing.T) {
	dir := t.TempDir()
	privado := filepath.Join(t.TempDir(), "privado")
	if err := os.WriteFile(privado, []byte("sintetico"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(privado, filepath.Join(dir, "enlace")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "directorio"), 0700); err != nil {
		t.Fatal(err)
	}
	raiz, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer raiz.Close()
	for _, ruta := range []string{"../privado", privado, "enlace", "directorio", ".", "a/../privado", "a\\privado", "ausente"} {
		t.Run(ruta, func(t *testing.T) {
			if _, _, err := leerHuella(raiz, ruta, 9); err != errArchivo {
				t.Fatalf("error=%v", err)
			}
		})
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(privado), filepath.Join(dir, "sub", "fuera")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := leerHuella(raiz, "sub/fuera/privado", 9); err != errArchivo {
		t.Fatalf("escape por directorio simbolico: %v", err)
	}
}
