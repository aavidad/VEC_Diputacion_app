package bootstrap

import (
	"context"
	"strings"
	"testing"
)

func TestHuellasContactoConservanGeneracionHistoricaYSeparanCorreo(t *testing.T) {
	sujeto := "per_" + strings.Repeat("a", 22)
	anterior := huellasContactoDesarrollo{nuevoDerivadorIdempotenciaPrueba(t, 3, 2)}
	posterior := huellasContactoDesarrollo{nuevoDerivadorIdempotenciaPrueba(t, 2, 1)}
	a, err := anterior.DerivarHuellasContactoUsuario(context.Background(), sujeto, 0, "persona@example.test")
	if err != nil {
		t.Fatal(err)
	}
	b, err := posterior.DerivarHuellasContactoUsuario(context.Background(), sujeto, 0, "persona@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || len(b) != 2 || a[1] != b[0] || a[0] == b[0] {
		t.Fatal("rotación no conservó la generación solapada")
	}
	otro, err := posterior.DerivarHuellasContactoUsuario(context.Background(), sujeto, 0, "otra@example.test")
	if err != nil || otro[0].ValorHMACSHA256 == b[0].ValorHMACSHA256 {
		t.Fatal("correo cambiado reutilizó huella")
	}
	version, err := posterior.DerivarHuellasContactoUsuario(context.Background(), sujeto, 1, "persona@example.test")
	if err != nil || version[0].ValorHMACSHA256 == b[0].ValorHMACSHA256 {
		t.Fatal("versión distinta reutilizó huella")
	}
	persona, err := posterior.DerivarHuellasContactoUsuario(context.Background(), "per_"+strings.Repeat("b", 22), 0, "persona@example.test")
	if err != nil || persona[0].ValorHMACSHA256 == b[0].ValorHMACSHA256 {
		t.Fatal("persona ajena reutilizó huella")
	}
	if strings.Contains(a[0].ClaveRef, "persona@example.test") || strings.Contains(a[0].ValorHMACSHA256, "persona@example.test") {
		t.Fatal("material claro expuesto")
	}
}
