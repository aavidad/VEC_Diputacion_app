package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"testing"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestKMSContactoUsuarioLigaSujetoYVersionYBorraClaro(t *testing.T) {
	k, _, _ := nuevosProveedoresKMSPrueba(t)
	claro := []byte("persona@prueba.local")
	s, err := k.CifrarContactoUsuario(context.Background(), "per_0123456789abcdefghijkl", 1, claro)
	if err != nil {
		t.Fatal(err)
	}
	var got, prestado []byte
	if err = k.ConContactoUsuarioDescifrado(context.Background(), "per_0123456789abcdefghijkl", s, func(v []byte) error { prestado = v; got = append([]byte(nil), v...); return nil }); err != nil || !bytes.Equal(got, claro) {
		t.Fatalf("lectura=%q %v", got, err)
	}
	if len(prestado) == 0 || !bytes.Equal(prestado, make([]byte, len(prestado))) {
		t.Fatal("el buffer prestado conserva el claro después del callback")
	}
	if err = k.ConContactoUsuarioDescifrado(context.Background(), "per_1123456789abcdefghijkl", s, func([]byte) error { return nil }); err == nil {
		t.Fatal("aceptó sujeto distinto")
	}
	s.Version = 2
	if err = k.ConContactoUsuarioDescifrado(context.Background(), "per_0123456789abcdefghijkl", s, func([]byte) error { return nil }); err == nil {
		t.Fatal("aceptó versión distinta")
	}
	for nombre, cambiar := range map[string]func(*vecports.SobreContactoUsuario){
		"clave":   func(x *vecports.SobreContactoUsuario) { x.ClaveRef = "otra" },
		"nonce":   func(x *vecports.SobreContactoUsuario) { x.Nonce[0] ^= 1 },
		"cifrado": func(x *vecports.SobreContactoUsuario) { x.Cifrado[0] ^= 1 },
	} {
		t.Run(nombre, func(t *testing.T) {
			x := s
			x.Version = 1
			x.Nonce = append([]byte(nil), s.Nonce...)
			x.Cifrado = append([]byte(nil), s.Cifrado...)
			cambiar(&x)
			if k.ConContactoUsuarioDescifrado(context.Background(), "per_0123456789abcdefghijkl", x, func([]byte) error { return nil }) == nil {
				t.Fatal("aceptó sobre alterado")
			}
		})
	}
}

type lectorCancelacionContactoKMS struct{ cancelar context.CancelFunc }

func (l *lectorCancelacionContactoKMS) Read(p []byte) (int, error) {
	l.cancelar()
	for i := range p {
		p[i] = byte(i + 1)
	}
	return len(p), nil
}

func TestKMSContactoUsuarioNiegaProveedorInvalidoYCancelacion(t *testing.T) {
	const sujeto = "per_0123456789abcdefghijkl"
	for _, caso := range []string{"clave_cero", "aleatorio_typed_nil", "sujeto_invalido", "cancelacion_rng"} {
		t.Run(caso, func(t *testing.T) {
			k, _, _ := nuevosProveedoresKMSPrueba(t)
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			ref := sujeto
			switch caso {
			case "clave_cero":
				clear(k.claveEnvoltura[:])
			case "aleatorio_typed_nil":
				k.aleatorio = (*lectorCancelacionContactoKMS)(nil)
			case "sujeto_invalido":
				ref = "per_texto\nlibre"
			case "cancelacion_rng":
				k.aleatorio = &lectorCancelacionContactoKMS{cancelar: cancelar}
			}
			sobre, err := k.CifrarContactoUsuario(ctx, ref, 1, []byte("persona@prueba.local"))
			if !errors.Is(err, errKMSContactoUsuarioNoDisponible) || len(sobre.Cifrado) != 0 || len(sobre.Nonce) != 0 {
				t.Fatal("el proveedor inválido o cancelado produjo un sobre")
			}
		})
	}
}

func TestKMSContactoUsuarioNiegaOtraClaveYBorraTrasError(t *testing.T) {
	const sujeto = "per_0123456789abcdefghijkl"
	k, _, _ := nuevosProveedoresKMSPrueba(t)
	sobre, err := k.CifrarContactoUsuario(context.Background(), sujeto, 1, []byte("persona@prueba.local"))
	if err != nil {
		t.Fatal(err)
	}
	for _, cero := range []bool{false, true} {
		otro := *k
		if cero {
			clear(otro.claveEnvoltura[:])
		} else {
			otro.claveEnvoltura[0] ^= 1
		}
		llamadas := 0
		err := otro.ConContactoUsuarioDescifrado(context.Background(), sujeto, sobre, func([]byte) error { llamadas++; return nil })
		if err == nil || llamadas != 0 {
			t.Fatal("expuso claro con clave distinta o cero")
		}
	}
	var prestado []byte
	err = k.ConContactoUsuarioDescifrado(context.Background(), sujeto, sobre, func(v []byte) error {
		prestado = v
		return errors.New("detalle privado del consumidor")
	})
	if !errors.Is(err, errKMSContactoUsuarioNoDisponible) || len(prestado) == 0 || !bytes.Equal(prestado, make([]byte, len(prestado))) {
		t.Fatal("no redactó el error o conservó el claro tras fallar el consumidor")
	}
}
