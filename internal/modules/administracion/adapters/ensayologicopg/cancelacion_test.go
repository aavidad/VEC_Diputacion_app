package ensayologicopg

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
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
	if copiarArchivo(ctx, Archivo{Ruta: "entrada-no-existe", SHA256: strings.Repeat("a", 64)}, ruta, 1024) == nil {
		t.Fatal("la copia ignoró la cancelación")
	}
	if globalsAdmitidos(ctx, "entrada-no-existe", 1024) {
		t.Fatal("la lectura de globals ignoró la cancelación")
	}
}
