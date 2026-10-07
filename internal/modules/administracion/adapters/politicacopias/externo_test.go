package politicacopias

import (
	"os"
	"path/filepath"
	"testing"
)

func TestControlOutsideDeclaredRollbackRoots(t *testing.T) {
	rollback := t.TempDir()
	inside := filepath.Join(rollback, "control")
	if _, e := AbrirExterno(ConfigExterna{Directorio: inside, RaicesRestauradas: []string{rollback}}); e == nil {
		t.Fatal("control inside rollback accepted")
	}
	if _, e := os.Stat(inside); !os.IsNotExist(e) {
		t.Fatal("rejected configuration created directory")
	}
	outside := filepath.Join(t.TempDir(), "control")
	store, e := AbrirExterno(ConfigExterna{Directorio: outside, RaicesRestauradas: []string{rollback}})
	if e != nil {
		t.Fatal(e)
	}
	_ = store.Cerrar()
	alias := filepath.Join(t.TempDir(), "alias")
	if e = os.Symlink(rollback, alias); e != nil {
		t.Fatal(e)
	}
	if _, e = AbrirExterno(ConfigExterna{Directorio: filepath.Join(alias, "control"), RaicesRestauradas: []string{rollback}}); e == nil {
		t.Fatal("aliased rollback accepted")
	}
}
