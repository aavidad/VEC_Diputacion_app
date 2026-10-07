package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var ErrDenominacionPersonaInvalida = errors.New("vec.denominacion_persona.invalida")

// DenominacionPersona contiene un nombre declarado para presentación. No
// acredita identidad civil, empleo, autenticación ni permiso alguno.
type DenominacionPersona struct {
	persona, norma string
	version        uint64
	nombre         []byte
}

func NuevaDenominacionPersona(persona string, version uint64, nombre []byte, normaRef string, maxBytes int) (DenominacionPersona, error) {
	if !ReferenciaPersonaDenominacionValida(persona) || version == 0 || version > 1<<53-1 || normaRef == "" || len(normaRef) > 128 || !NombreDenominacionPersonaValido(nombre, maxBytes) {
		return DenominacionPersona{}, ErrDenominacionPersonaInvalida
	}
	for _, r := range normaRef {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
			return DenominacionPersona{}, ErrDenominacionPersonaInvalida
		}
	}
	return DenominacionPersona{persona: persona, norma: normaRef, version: version, nombre: append([]byte(nil), nombre...)}, nil
}

func ReferenciaPersonaDenominacionValida(ref string) bool {
	return referenciaOpacaContextoActorValida(ref, "per_")
}

// El límite funcional procede del catálogo. 4096 es un límite técnico de
// transporte; NFC y rechazo de controles preservan una representación única.
func NombreDenominacionPersonaValido(nombre []byte, maxBytes int) bool {
	if maxBytes <= 0 || maxBytes > 4096 || len(nombre) == 0 || len(nombre) > maxBytes || !utf8.Valid(nombre) || !norm.NFC.IsNormal(nombre) {
		return false
	}
	first, _ := utf8.DecodeRune(nombre)
	last, _ := utf8.DecodeLastRune(nombre)
	if unicode.IsSpace(first) || unicode.IsSpace(last) {
		return false
	}
	for _, r := range string(nombre) {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	return true
}

func (d DenominacionPersona) PersonaRef() string { return d.persona }
func (d DenominacionPersona) Version() uint64    { return d.version }
func (d DenominacionPersona) NormaRef() string   { return d.norma }

// ConNombreMostrar presta una copia durante el callback y la borra después.
// El consumidor autorizado no debe conservarla ni registrarla.
func (d DenominacionPersona) ConNombreMostrar(usar func([]byte) error) error {
	if usar == nil || !ReferenciaPersonaDenominacionValida(d.persona) || d.version == 0 || !NombreDenominacionPersonaValido(d.nombre, 4096) {
		return ErrDenominacionPersonaInvalida
	}
	b := append([]byte(nil), d.nombre...)
	defer clear(b)
	return usar(b)
}

func (DenominacionPersona) String() string               { return "vec.DenominacionPersona{redactado}" }
func (DenominacionPersona) GoString() string             { return "vec.DenominacionPersona{redactado}" }
func (d DenominacionPersona) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, d.String()) }
func (d DenominacionPersona) LogValue() slog.Value       { return slog.StringValue(d.String()) }
func (DenominacionPersona) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}
