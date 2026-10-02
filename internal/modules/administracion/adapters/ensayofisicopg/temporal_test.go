package ensayofisicopg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRaizTemporalPropiaConfinada(t *testing.T) {
	base, err := os.MkdirTemp("/var/tmp", "vec-cs06-base-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if os.RemoveAll(base) != nil {
			t.Error("limpiar base propia")
		}
	})
	raiz, err := CrearRaizTemporal(base, "vec-cs06f-")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(raiz)
	if err != nil || info.Mode().Perm() != 0700 || filepath.Dir(raiz) != base || !raizTemporalAdmitida(raiz) {
		t.Fatal("raíz temporal no privada")
	}
	enlace := filepath.Join(base, "enlace")
	if os.Symlink(raiz, enlace) != nil {
		t.Fatal("enlace fixture")
	}
	if _, err := CrearRaizTemporal(enlace, "vec-cs06f-"); err == nil {
		t.Fatal("base enlazada aceptada")
	}
	if os.Chmod(base, 0755) != nil {
		t.Fatal("modo fixture")
	}
	if _, err := CrearRaizTemporal(base, "vec-cs06f-"); err == nil {
		t.Fatal("base compartida aceptada")
	}
	for _, p := range []string{"/tmp", "/dev/shm", "/var/tmp/../var/tmp"} {
		if _, err := CrearRaizTemporal(p, "vec-cs06f-"); err == nil {
			t.Fatal("base no admitida", p)
		}
	}
}
