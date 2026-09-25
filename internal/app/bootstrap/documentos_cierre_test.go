package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type emisorCierreDocumentosPrueba struct {
	cerrar func(context.Context) error
}

func (e emisorCierreDocumentosPrueba) Cerrar(ctx context.Context) error { return e.cerrar(ctx) }

func TestCerrarDocumentosVaciaEmisorAntesDePoolsYPropagaError(t *testing.T) {
	var orden []string
	fallo := errors.New("fallo de vaciado")
	ctx := context.Background()
	cerrar := nuevoCierreDocumentos(func(ctx context.Context) error {
		return cerrarDocumentos(ctx, emisorCierreDocumentosPrueba{cerrar: func(recibido context.Context) error {
			if recibido != ctx {
				t.Fatal("el emisor no recibió el contexto del cierre")
			}
			orden = append(orden, "emisor")
			return fallo
		}}, []func(){
			func() { orden = append(orden, "pool 1") },
			func() { orden = append(orden, "pool 2") },
		})
	})
	for i := 0; i < 2; i++ {
		if err := cerrar(ctx); !errors.Is(err, fallo) {
			t.Fatalf("cierre %d perdió el error del emisor: %v", i+1, err)
		}
	}
	if esperado := []string{"emisor", "pool 2", "pool 1"}; !reflect.DeepEqual(orden, esperado) {
		t.Fatalf("orden de cierre = %v, esperado %v", orden, esperado)
	}
}
