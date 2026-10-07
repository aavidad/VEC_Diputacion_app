// Package denominacionpersona prepara sobres cifrados y consume puertos
// autorizados. No configura KMS, concede permisos ni monta rutas.
package denominacionpersona

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

var ErrNoDisponible = errors.New("vec.denominacion_persona.no_disponible")

type Clave struct {
	Ref          string
	Material     [32]byte
	RetenerHasta time.Time
	Revocada     bool
}

func (Clave) String() string               { return "denominacionpersona.Clave{redactado}" }
func (Clave) GoString() string             { return "denominacionpersona.Clave{redactado}" }
func (k Clave) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, k.String()) }
func (k Clave) LogValue() slog.Value       { return slog.StringValue(k.String()) }
func (Clave) MarshalJSON() ([]byte, error) { return []byte(`{"redactado":true}`), nil }

// La composición obtiene estos ámbitos dedicados desde el KMS/configuración
// existentes. Nunca reutiliza claves de contacto, correo o autenticación.
// Rotar Busqueda exige reindexación durable y cierre de altas/búsquedas hasta
// que el registro anuncie explícitamente la nueva ClaveRef.
type Claves struct {
	Cifrado, Busqueda Clave
	CifradoRetenidas  []Clave
}

func (Claves) String() string               { return "denominacionpersona.Claves{redactado}" }
func (c Claves) GoString() string           { return c.String() }
func (c Claves) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, c.String()) }
func (c Claves) LogValue() slog.Value       { return slog.StringValue(c.String()) }
func (Claves) MarshalJSON() ([]byte, error) { return []byte(`{"redactado":true}`), nil }

type FuenteClaves interface {
	CargarClavesDenominacionPersona(context.Context) (Claves, error)
}

type Protector struct {
	fuente   FuenteClaves
	ahora    func() time.Time
	norma    ports.NormaDenominacionPersona
	normaSHA string
}

func (*Protector) String() string               { return "denominacionpersona.Protector{redactado}" }
func (p *Protector) GoString() string           { return p.String() }
func (p *Protector) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, p.String()) }
func (p *Protector) LogValue() slog.Value       { return slog.StringValue(p.String()) }
func (*Protector) MarshalJSON() ([]byte, error) { return []byte(`{"redactado":true}`), nil }

func NuevoProtector(fuente FuenteClaves, ahora func() time.Time, norma ports.NormaDenominacionPersona) (*Protector, error) {
	if nulo(fuente) || ahora == nil || !normaValida(norma) {
		return nil, ErrNoDisponible
	}
	return &Protector{fuente: fuente, ahora: ahora, norma: norma, normaSHA: huellaNorma(norma)}, nil
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Ptr || r.Kind() == reflect.Interface || r.Kind() == reflect.Func || r.Kind() == reflect.Map || r.Kind() == reflect.Slice || r.Kind() == reflect.Chan) && r.IsNil()
}

func referenciaValida(s string) bool {
	if len(s) < 8 || len(s) > 128 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == ':' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func usable(k Clave, ahora time.Time) bool {
	return referenciaValida(k.Ref) && k.Material != ([32]byte{}) && !k.Revocada && (k.RetenerHasta.IsZero() || ahora.Before(k.RetenerHasta))
}

func (p *Protector) claves(ctx context.Context) (Claves, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || nulo(p.fuente) {
		return Claves{}, ErrNoDisponible
	}
	c, err := p.fuente.CargarClavesDenominacionPersona(ctx)
	if len(c.CifradoRetenidas) > 8 {
		clear(c.Cifrado.Material[:])
		clear(c.Busqueda.Material[:])
		return Claves{}, ErrNoDisponible
	}
	c.CifradoRetenidas = append([]Clave(nil), c.CifradoRetenidas...)
	if err != nil || ctx.Err() != nil || len(c.CifradoRetenidas) > 8 || !usable(c.Cifrado, p.ahora()) || !usable(c.Busqueda, p.ahora()) {
		borrarClaves(&c)
		return Claves{}, ErrNoDisponible
	}
	refs := map[string]bool{}
	mats := map[[32]byte]bool{}
	for i, k := range append([]Clave{c.Cifrado, c.Busqueda}, c.CifradoRetenidas...) {
		if !referenciaValida(k.Ref) || k.Material == ([32]byte{}) || refs[k.Ref] || mats[k.Material] || i >= 2 && k.RetenerHasta.IsZero() {
			borrarClaves(&c)
			return Claves{}, ErrNoDisponible
		}
		refs[k.Ref], mats[k.Material] = true, true
	}
	return c, nil
}

func borrarClaves(c *Claves) {
	clear(c.Cifrado.Material[:])
	clear(c.Busqueda.Material[:])
	for i := range c.CifradoRetenidas {
		clear(c.CifradoRetenidas[i].Material[:])
	}
}

func (p *Protector) revalidarClaves(ctx context.Context, cifrado, busqueda Clave) error {
	c, err := p.claves(ctx)
	if err != nil {
		return ErrNoDisponible
	}
	defer borrarClaves(&c)
	k, ok := buscarClave(c, cifrado.Ref, p.ahora())
	if !ok || c.Busqueda.Ref != busqueda.Ref || subtle.ConstantTimeCompare(k.Material[:], cifrado.Material[:]) != 1 || subtle.ConstantTimeCompare(c.Busqueda.Material[:], busqueda.Material[:]) != 1 {
		return ErrNoDisponible
	}
	return nil
}

func buscarClave(c Claves, ref string, ahora time.Time) (Clave, bool) {
	for _, k := range append([]Clave{c.Cifrado}, c.CifradoRetenidas...) {
		if k.Ref == ref && usable(k, ahora) {
			return k, true
		}
	}
	return Clave{}, false
}
