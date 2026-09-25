package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCierreRecursosInversoEIdempotente(t *testing.T) {
	var orden []string
	fallo := errors.New("fallo de vaciado")
	cerrar := nuevoCierreRecursos([]func(context.Context) error{
		func(context.Context) error { orden = append(orden, "pool"); return nil },
		func(context.Context) error { orden = append(orden, "emisor"); return fallo },
	})
	if err := cerrar(context.Background()); !errors.Is(err, fallo) {
		t.Fatalf("fallo de emisor perdido: %v", err)
	}
	if err := cerrar(context.Background()); !errors.Is(err, fallo) {
		t.Fatalf("segundo cierre: %v", err)
	}
	if !reflect.DeepEqual(orden, []string{"emisor", "pool"}) {
		t.Fatalf("orden de cierre: %v", orden)
	}
}
