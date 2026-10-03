package main

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

func fuentesPrueba() (*fuenteEfimera, *fuenteEfimera) {
	var a, b [32]byte
	for i := range a {
		a[i] = byte(i + 1)
		b[i] = byte(i + 33)
	}
	return fuenteDesdeSemilla(a, false), fuenteDesdeSemilla(b, true)
}

func TestConversionConservaDireccionVersionYAAD(t *testing.T) {
	ctx := context.Background()
	vieja, propia := fuentesPrueba()
	defer vieja.borrar()
	defer propia.borrar()
	anterior, _ := seguridad.NuevoAdaptadorCorreos(vieja, time.Now)
	nueva, _ := seguridad.NuevoAdaptadorCorreos(propia, time.Now)
	persona := "per_" + strings.Repeat("a", 22)
	correo := "correo:" + strings.Repeat("b", 32)
	s, err := anterior.CifrarDireccionCorreo(ctx, persona, correo, 7, []byte("lucia.morales@example.invalid"))
	if err != nil {
		t.Fatal(err)
	}
	fila := direccionDB{correo, 7, s.ClaveRef, s.ClaveIgualdadRef, `\x` + hex.EncodeToString(s.Nonce), `\x` + hex.EncodeToString(s.Cifrado), `\x` + hex.EncodeToString(s.HuellaIgualdad)}
	m, err := convertir(ctx, persona, []direccionDB{fila}, vieja, propia)
	if err != nil || len(m.Direcciones) != 1 {
		t.Fatal("conversión rechazada", err)
	}
	d := m.Direcciones[0]
	if d.CorreoRef != correo || d.Version != 7 || d.Nonce == hex.EncodeToString(s.Nonce) || d.Igualdad == hex.EncodeToString(s.HuellaIgualdad) {
		t.Fatal("versión, nonce o igualdad incorrectos")
	}
	nonce, _ := hex.DecodeString(d.Nonce)
	cifrado, _ := hex.DecodeString(d.Cifrado)
	igualdad, _ := hex.DecodeString(d.Igualdad)
	sobre := ports.SobreDireccionCorreo{CorreoRef: correo, Version: 7, ClaveRef: m.ClaveSobre, ClaveIgualdadRef: m.ClaveIgualdad, Nonce: nonce, Cifrado: cifrado, HuellaIgualdad: igualdad}
	if err := nueva.ConDireccionCorreoDescifrada(ctx, persona, sobre, func(claro []byte) error {
		if string(claro) != "lucia.morales@example.invalid" {
			return errors.New("dirección distinta")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if anterior.ConDireccionCorreoDescifrada(ctx, persona, sobre, func([]byte) error { return nil }) == nil {
		t.Fatal("clave anterior aceptó sobre propio")
	}
	if nueva.ConDireccionCorreoDescifrada(ctx, "per_"+strings.Repeat("z", 22), sobre, func([]byte) error { return nil }) == nil {
		t.Fatal("otra persona aceptó sobre")
	}

	for _, caso := range []string{"nonce", "cifrado", "igualdad", "ref", "version"} {
		t.Run(caso, func(t *testing.T) {
			x := fila
			switch caso {
			case "nonce":
				x.Nonce = `\x` + strings.Repeat("0", 24)
			case "cifrado":
				x.Cifrado = `\x` + strings.Repeat("0", 80)
			case "igualdad":
				x.Igualdad = `\x` + strings.Repeat("0", 64)
			case "ref":
				x.ClaveSobre = m.ClaveSobre
			case "version":
				x.Version++
			}
			if _, err := convertir(ctx, persona, []direccionDB{x}, vieja, propia); err == nil {
				t.Fatal("preimagen alterada aceptada")
			}
		})
	}
}

func TestFuenteBorraMaterialYSinRetenidas(t *testing.T) {
	vieja, propia := fuentesPrueba()
	for _, f := range []*fuenteEfimera{vieja, propia} {
		if len(f.claves.CifradoRetenidas)+len(f.claves.CodigoRetenidas)+len(f.claves.SemanticaRetenidas) != 0 {
			t.Fatal("claves retenidas")
		}
		f.borrar()
		if f.claves.CifradoActivo.Material != ([32]byte{}) || f.claves.Igualdad.Material != ([32]byte{}) || f.claves.SemanticaActiva.Material != ([32]byte{}) || f.claves.CodigoActivo.Material != ([32]byte{}) {
			t.Fatal("material no borrado")
		}
	}
}
