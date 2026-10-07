package ensayologicopg

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type lectorCancela struct {
	cancelar context.CancelFunc
	lecturas int
}

func (l *lectorCancela) Read(p []byte) (int, error) {
	l.lecturas++
	l.cancelar()
	p[0] = 'a'
	return 1, nil
}

func TestCancelacionDetieneLaLecturaEntreBloques(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	origen := &lectorCancela{cancelar: cancelar}
	_, err := io.Copy(io.Discard, lectorContexto{ctx, origen})
	if !errors.Is(err, context.Canceled) || origen.lecturas != 1 {
		t.Fatalf("error=%v lecturas=%d", err, origen.lecturas)
	}
}

func TestContextoCanceladoNoAbreEntradas(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	ruta := filepath.Join(t.TempDir(), "no-creado.dump")
	entrada := archivoFixture(t, "entrada.dump", []byte("PGDMPensayo"))
	globals := archivoFixture(t, "globals.sql", []byte("-- ensayo sintetico\n"))
	if copiarArchivo(ctx, entrada, ruta, 1024) == nil {
		t.Fatal("la copia ignoró la cancelación")
	}
	if _, err := os.Stat(ruta); !os.IsNotExist(err) {
		t.Fatal("la copia cancelada creó un archivo")
	}
	if validarGlobals(ctx, globals.Ruta, 1024) == nil {
		t.Fatal("la lectura de globals ignoró la cancelación")
	}
}
