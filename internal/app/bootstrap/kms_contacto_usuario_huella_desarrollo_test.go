package bootstrap

import (
	"context"
	"testing"
)

func TestHuellaPeticionContactoUsuarioEsEstableYLigada(t *testing.T) {
	k, _, _ := nuevosProveedoresKMSPrueba(t)
	sujeto := "per_0123456789abcdefghijkl"
	correo := []byte("persona@prueba.local")
	a, e := k.HuellaPeticionContactoUsuario(context.Background(), sujeto, 0, correo)
	if e != nil || len(a) != 64 {
		t.Fatalf("huella %q %v", a, e)
	}
	b, _ := k.HuellaPeticionContactoUsuario(context.Background(), sujeto, 0, correo)
	if a != b {
		t.Fatal("no estable")
	}
	for _, x := range []struct {
		s string
		v uint64
		c []byte
	}{{"per_1123456789abcdefghijkl", 0, correo}, {sujeto, 1, correo}, {sujeto, 0, []byte("otra@prueba.local")}} {
		h, _ := k.HuellaPeticionContactoUsuario(context.Background(), x.s, x.v, x.c)
		if h == a {
			t.Fatal("colisión de intención")
		}
	}
}
