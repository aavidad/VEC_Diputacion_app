package bootstrap

import (
	"bytes"
	"context"
	"testing"

	seleccionports "vec-diputacion-granada/internal/modules/seleccion/ports"
)

func TestKMSSeleccionCifraLigadoAPersonaConvocatoriaYVersion(t *testing.T) {
	k, _, _ := nuevosProveedoresKMSPrueba(t)
	ctx := context.Background()
	a := seleccionports.AsociacionDatos{PersonaRef: "per_0123456789abcdefghijkl", ConvocatoriaRef: "bolsa-operario-diputacion-2026", Version: 2}
	claro := []byte(`{"documento_identidad":"12345678Z"}`)
	sobre, err := k.CifrarDatosSolicitud(ctx, a, claro)
	if err != nil || bytes.Contains(sobre.Cifrado, []byte("12345678Z")) || len(sobre.Nonce) != 12 {
		t.Fatalf("sobre inválido: %v", err)
	}
	var leido []byte
	if err := k.ConDatosSolicitudDescifrados(ctx, a, sobre, func(b []byte) error { leido = append([]byte(nil), b...); return nil }); err != nil || !bytes.Equal(leido, claro) {
		t.Fatalf("no descifra su propio sobre: %v", err)
	}
	for _, otra := range []seleccionports.AsociacionDatos{
		{PersonaRef: "per_0123456789abcdefghijkX", ConvocatoriaRef: a.ConvocatoriaRef, Version: 2},
		{PersonaRef: a.PersonaRef, ConvocatoriaRef: "otra-convocatoria", Version: 2},
		{PersonaRef: a.PersonaRef, ConvocatoriaRef: a.ConvocatoriaRef, Version: 3},
	} {
		if err := k.ConDatosSolicitudDescifrados(ctx, otra, sobre, func([]byte) error { return nil }); err == nil {
			t.Fatalf("un sobre copiado a %+v no debe descifrarse", otra)
		}
	}
	h1, err1 := k.HuellaConClave(ctx, "seleccion.documento_identidad.v1", []byte("12345678Z"))
	h2, err2 := k.HuellaConClave(ctx, "seleccion.documento_identidad.v1", []byte("12345678Z"))
	h3, err3 := k.HuellaConClave(ctx, "seleccion.material_borrador.v1", []byte("12345678Z"))
	if err1 != nil || err2 != nil || err3 != nil || h1 != h2 || h1 == h3 || len(h1) != 64 {
		t.Fatal("la huella con clave debe ser estable por dominio y distinta entre dominios")
	}
	if _, err := k.HuellaConClave(ctx, "otro.dominio", nil); err == nil {
		t.Fatal("dominio de huella no admitido")
	}
}
