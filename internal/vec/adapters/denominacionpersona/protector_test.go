package denominacionpersona

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const personaUno = "per_aaaaaaaaaaaaaaaaaaaaaaaa"
const personaDos = "per_bbbbbbbbbbbbbbbbbbbbbbbb"
const ambitoPrueba = "ambito:prueba"

type fuenteClavesPrueba struct {
	claves   Claves
	llamadas int
	fallaEn  int
	error    bool
}

func (f *fuenteClavesPrueba) CargarClavesDenominacionPersona(context.Context) (Claves, error) {
	f.llamadas++
	if f.error || f.llamadas == f.fallaEn {
		return Claves{}, errors.New("secreto_no_exponer")
	}
	return f.claves, nil
}

func escenario(t *testing.T) (*Protector, *fuenteClavesPrueba) {
	t.Helper()
	f := &fuenteClavesPrueba{claves: Claves{Cifrado: Clave{Ref: "kms:denom:cifrado:1"}, Busqueda: Clave{Ref: "kms:denom:busqueda:1"}}}
	for i := range f.claves.Cifrado.Material {
		f.claves.Cifrado.Material[i] = 0x31
		f.claves.Busqueda.Material[i] = 0x62
	}
	p, err := NuevoProtector(f, func() time.Time { return time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC) }, ports.NormaDenominacionPersona{Ref: "norma:nombre:1", Case: "fold", FormaUnicode: "NFC", Separadores: " -'", MaxBytes: 256, MaxTokens: 12})
	if err != nil {
		t.Fatal("constructor")
	}
	return p, f
}

func preparar(t *testing.T, p *Protector, persona string, esperada uint64) ports.PreparacionDenominacionPersona {
	t.Helper()
	x, err := p.PrepararDenominacionPersona(context.Background(), persona, esperada, "procedencia:declarada", ambitoPrueba, "kms:denom:busqueda:1", []byte("Álvaro Ejemplo"))
	if err != nil {
		t.Fatal("preparacion")
	}
	return x
}

func TestHomonimosCambiosYBusquedaCompleta(t *testing.T) {
	p, _ := escenario(t)
	uno, dos, cambio := preparar(t, p, personaUno, 0), preparar(t, p, personaDos, 0), preparar(t, p, personaUno, 1)
	if uno.PersonaRef == dos.PersonaRef || cambio.Sobre.Version != 2 || uno.Sobre.Version != 1 || uno.SobreSHA256 == dos.SobreSHA256 || bytes.Equal(uno.Sobre.Cifrado, dos.Sobre.Cifrado) {
		t.Fatal("identidad_o_version")
	}
	if !bytes.Equal(uno.Sobre.Indice.Tokens[0], dos.Sobre.Indice.Tokens[0]) {
		t.Fatal("homonimos_busqueda")
	}
	for _, s := range []ports.SobreDenominacionPersona{uno.Sobre, dos.Sobre, cambio.Sobre} {
		if err := p.ConDenominacionDescifrada(context.Background(), s, func(d domain.DenominacionPersona) error {
			return d.ConNombreMostrar(func(b []byte) error {
				if !bytes.Equal(b, []byte("Álvaro Ejemplo")) {
					t.Fatal("nombre")
				}
				return nil
			})
		}); err != nil {
			t.Fatal("lectura")
		}
	}
	buscar := func(ambito, texto string) ports.IndiceDenominacionPersona {
		i, e := p.PrepararBusquedaDenominacionPersona(context.Background(), ambito, "kms:denom:busqueda:1", []byte(texto))
		if e != nil {
			t.Fatal("busqueda")
		}
		return i
	}
	completa, prefijo, otro := buscar(ambitoPrueba, "A\u0301LVARO"), buscar(ambitoPrueba, "Álvar"), buscar("ambito:otro", "Álvaro")
	contiene := func(i ports.IndiceDenominacionPersona) bool {
		for _, tok := range uno.Sobre.Indice.Tokens {
			if bytes.Equal(tok, i.Tokens[0]) {
				return true
			}
		}
		return false
	}
	if !contiene(completa) || contiene(prefijo) || contiene(otro) {
		t.Fatal("tokens_ambito_palabra")
	}
}

func TestSobreNoAdmiteSustitucionNiDebugClaro(t *testing.T) {
	p, _ := escenario(t)
	prep := preparar(t, p, personaUno, 0)
	mutaciones := []func(*ports.SobreDenominacionPersona){func(s *ports.SobreDenominacionPersona) { s.PersonaRef = personaDos }, func(s *ports.SobreDenominacionPersona) { s.Version++ }, func(s *ports.SobreDenominacionPersona) { s.Esquema = "vec.persona.denominacion.aead.v2" }, func(s *ports.SobreDenominacionPersona) { s.Indice.AmbitoRef = "ambito:otro" }, func(s *ports.SobreDenominacionPersona) { s.Indice.Tokens[0][0] ^= 1 }, func(s *ports.SobreDenominacionPersona) { s.Cifrado[0] ^= 1 }}
	for _, mutar := range mutaciones {
		s := clonarSobre(prep.Sobre)
		mutar(&s)
		llamado := false
		err := p.ConDenominacionDescifrada(context.Background(), s, func(domain.DenominacionPersona) error { llamado = true; return nil })
		if !errors.Is(err, ErrNoDisponible) || llamado {
			t.Fatal("sustitucion_aceptada")
		}
	}
	err := p.ConDenominacionDescifrada(context.Background(), prep.Sobre, func(d domain.DenominacionPersona) error {
		b, _ := json.Marshal(d)
		if bytes.Contains(b, []byte("Álvaro")) || bytes.Contains([]byte(fmt.Sprintf("%#v", d)), []byte("Álvaro")) || fmt.Sprintf("%d", d) != d.String() {
			t.Fatal("claro_debug")
		}
		var prestado []byte
		err := d.ConNombreMostrar(func(b []byte) error { prestado = b; return nil })
		if err != nil || !bytes.Equal(prestado, make([]byte, len(prestado))) {
			t.Fatal("borrado_callback")
		}
		return nil
	})
	if err != nil {
		t.Fatal("lectura_redactada")
	}
}

func TestRotacionRevocacionYFalloKMS(t *testing.T) {
	p, f := escenario(t)
	prep := preparar(t, p, personaUno, 0)
	vieja := f.claves.Cifrado
	vieja.RetenerHasta = p.ahora().Add(time.Hour)
	f.claves.CifradoRetenidas = []Clave{vieja}
	f.claves.Cifrado.Ref = "kms:denom:cifrado:2"
	for i := range f.claves.Cifrado.Material {
		f.claves.Cifrado.Material[i] = 0x43
	}
	if p.ConDenominacionDescifrada(context.Background(), prep.Sobre, func(domain.DenominacionPersona) error { return nil }) != nil {
		t.Fatal("retenida")
	}
	if f.claves.CifradoRetenidas[0].Material != vieja.Material {
		t.Fatal("fuente_modificada")
	}
	fallo := func() {
		llamado := false
		err := p.ConDenominacionDescifrada(context.Background(), prep.Sobre, func(domain.DenominacionPersona) error { llamado = true; return nil })
		if !errors.Is(err, ErrNoDisponible) || llamado {
			t.Fatal("fallo_abierto")
		}
	}
	f.claves.CifradoRetenidas[0].Revocada = true
	fallo()
	f.claves.CifradoRetenidas[0].Revocada = false
	f.fallaEn = f.llamadas + 2
	fallo()
	f.fallaEn = 0
	f.claves.Busqueda.Ref = "kms:denom:busqueda:2"
	if _, err := p.PrepararBusquedaDenominacionPersona(context.Background(), ambitoPrueba, "kms:denom:busqueda:1", []byte("Álvaro")); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("rotacion_indice")
	}
	f.claves.Busqueda.Material = f.claves.Cifrado.Material
	if _, err := p.PrepararBusquedaDenominacionPersona(context.Background(), ambitoPrueba, f.claves.Busqueda.Ref, []byte("Álvaro")); !errors.Is(err, ErrNoDisponible) {
		t.Fatal("claves_compartidas")
	}
}
