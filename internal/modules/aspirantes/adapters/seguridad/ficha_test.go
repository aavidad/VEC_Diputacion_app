package seguridad

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
)

type fuentePrueba struct {
	c   ClavesAspirantes
	err error
}

func (f *fuentePrueba) CargarClavesAspirantes(context.Context) (ClavesAspirantes, error) {
	return f.c, f.err
}

func clave(ref string, b byte) Clave {
	var m [32]byte
	for i := range m {
		m[i] = b
	}
	return Clave{Ref: ref, Material: m}
}

func clavesPrueba() ClavesAspirantes {
	return ClavesAspirantes{
		CifradoActivo: clave("clave:aspirantes:cifrado:v1", 1), DocumentoActivo: clave("clave:aspirantes:documento:v1", 4), Indice: clave("clave:aspirantes:indice:v1", 2),
		SemanticaActiva: clave("clave:aspirantes:huella:v1", 3),
	}
}

const (
	aspA = "asp_AAAAAAAAAAAAAAAAAAAAAA"
	aspB = "asp_BBBBBBBBBBBBBBBBBBBBBA"
	docA = "aspdoc_AAAAAAAAAAAAAAAAAAAAAA"
)

func TestCifradoLigadoAFichaCampoYVersion(t *testing.T) {
	f := &fuentePrueba{c: clavesPrueba()}
	a, _ := NuevoAdaptador(f)
	ctx := context.Background()
	s, err := a.CifrarValor(ctx, aspA, domain.CampoTelefono, 3, []byte("958123456"))
	if err != nil || len(s.Nonce) != 12 || len(s.Cifrado) != 9+16 || bytes.Contains(s.Cifrado, []byte("958123456")) {
		t.Fatalf("sobre %+v %v", s, err)
	}
	claro, err := a.DescifrarValor(ctx, aspA, domain.CampoTelefono, 3, s)
	if err != nil || string(claro) != "958123456" {
		t.Fatalf("descifrado %q %v", claro, err)
	}
	for nombre, intento := range map[string]func() ([]byte, error){
		"otra ficha":   func() ([]byte, error) { return a.DescifrarValor(ctx, aspB, domain.CampoTelefono, 3, s) },
		"otro campo":   func() ([]byte, error) { return a.DescifrarValor(ctx, aspA, domain.CampoMovil, 3, s) },
		"otra versión": func() ([]byte, error) { return a.DescifrarValor(ctx, aspA, domain.CampoTelefono, 4, s) },
		"como documento": func() ([]byte, error) {
			return a.DescifrarDocumento(ctx, aspA, docA, s)
		},
	} {
		if _, err := intento(); !errors.Is(err, ErrCriptoNoDisponible) {
			t.Fatalf("%s: %v", nombre, err)
		}
	}
	otro, _ := a.CifrarValor(ctx, aspA, domain.CampoTelefono, 3, []byte("958123456"))
	if bytes.Equal(otro.Nonce, s.Nonce) || bytes.Equal(otro.Cifrado, s.Cifrado) {
		t.Fatal("nonce repetido")
	}
}

func TestRotacionRetieneYRevoca(t *testing.T) {
	f := &fuentePrueba{c: clavesPrueba()}
	a, _ := NuevoAdaptador(f)
	ctx := context.Background()
	viejo, _ := a.CifrarDocumento(ctx, aspA, docA, []byte("12345678Z"))
	rotadas := clavesPrueba()
	rotadas.DocumentoRetenidas = []Clave{rotadas.DocumentoActivo}
	rotadas.DocumentoActivo = clave("clave:aspirantes:documento:v2", 9)
	f.c = rotadas
	if claro, err := a.DescifrarDocumento(ctx, aspA, docA, viejo); err != nil || string(claro) != "12345678Z" {
		t.Fatalf("retenida %v", err)
	}
	nuevo, _ := a.CifrarDocumento(ctx, aspA, docA, []byte("12345678Z"))
	if nuevo.ClaveRef != "clave:aspirantes:documento:v2" {
		t.Fatal("cifra con la activa")
	}
	rotadas.DocumentoRetenidas[0].Revocada = true
	f.c = rotadas
	if _, err := a.DescifrarDocumento(ctx, aspA, docA, viejo); err == nil {
		t.Fatal("una revocada no descifra")
	}
}

func TestClavesSeparadasPorFuncion(t *testing.T) {
	ctx := context.Background()
	malas := []func(*ClavesAspirantes){
		func(c *ClavesAspirantes) { c.Indice.Material = c.CifradoActivo.Material },
		func(c *ClavesAspirantes) { c.DocumentoActivo.Material = c.CifradoActivo.Material },
		func(c *ClavesAspirantes) { c.DocumentoActivo = Clave{} },
		func(c *ClavesAspirantes) { c.SemanticaActiva.Ref = c.Indice.Ref },
		func(c *ClavesAspirantes) { c.CifradoActivo = Clave{} },
		func(c *ClavesAspirantes) { c.Indice.Revocada = true },
		func(c *ClavesAspirantes) { c.CifradoRetenidas = []Clave{c.CifradoActivo} },
		func(c *ClavesAspirantes) { c.SemanticaActiva.Ref = "ref con espacios" },
	}
	for i, mal := range malas {
		c := clavesPrueba()
		mal(&c)
		a, _ := NuevoAdaptador(&fuentePrueba{c: c})
		if _, err := a.CifrarValor(ctx, aspA, domain.CampoTelefono, 1, []byte("958123456")); err == nil {
			t.Fatalf("caso %d aceptado", i)
		}
	}
	a, _ := NuevoAdaptador(&fuentePrueba{err: errors.New("gestor caído")})
	if _, err := a.IndiceDocumento(ctx, domain.DocumentoIdentidad{}); err == nil {
		t.Fatal("sin claves")
	}
	if _, err := NuevoAdaptador(nil); err == nil {
		t.Fatal("sin fuente")
	}
}

func TestIndiceCiegoEstableYSeparado(t *testing.T) {
	a, _ := NuevoAdaptador(&fuentePrueba{c: clavesPrueba()})
	ctx := context.Background()
	dni, _ := domain.NuevoDocumentoIdentidad(domain.DocumentoDNI, "ES", "12345678Z")
	dni2, _ := domain.NuevoDocumentoIdentidad(domain.DocumentoDNI, "ES", "12.345.678-z")
	pas, _ := domain.NuevoDocumentoIdentidad(domain.DocumentoPasaporte, "ES", "12345678Z")
	i1, err := a.IndiceDocumento(ctx, dni)
	if err != nil || len(i1.Valor) != 64 || i1.ClaveRef != "clave:aspirantes:indice:v1" {
		t.Fatalf("índice %+v %v", i1, err)
	}
	i2, _ := a.IndiceDocumento(ctx, dni2)
	i3, _ := a.IndiceDocumento(ctx, pas)
	if i1 != i2 || i1.Valor == i3.Valor || strings.Contains(i1.Valor, "12345678") {
		t.Fatal("el índice depende solo del documento normalizado y de su tipo")
	}
}

func TestHuellaConRetenidas(t *testing.T) {
	c := clavesPrueba()
	c.SemanticaRetenidas = []Clave{clave("clave:aspirantes:huella:v0", 7), {Ref: "clave:aspirantes:huella:rev", Material: [32]byte{8}, Revocada: true}}
	a, _ := NuevoAdaptador(&fuentePrueba{c: c})
	h, err := a.SellarHuella(context.Background(), []byte("preimagen"))
	if err != nil || len(h.Retenidas) != 1 || h.Activa.Valor == h.Retenidas[0].Valor || len(h.Activa.Valor) != 64 {
		t.Fatalf("huellas %+v %v", h, err)
	}
	var _ ports.ProtectorFicha = a
	var _ ports.SelladorHuella = a
}

func TestDocumentoYContactoConClavesDistintas(t *testing.T) {
	a, _ := NuevoAdaptador(&fuentePrueba{c: clavesPrueba()})
	ctx := context.Background()
	d, _ := a.CifrarDocumento(ctx, aspA, docA, []byte("12345678Z"))
	v, _ := a.CifrarValor(ctx, aspA, domain.CampoTelefono, 1, []byte("958123456"))
	if d.ClaveRef == v.ClaveRef || d.ClaveRef != "clave:aspirantes:documento:v1" {
		t.Fatalf("claves %s %s", d.ClaveRef, v.ClaveRef)
	}
}
