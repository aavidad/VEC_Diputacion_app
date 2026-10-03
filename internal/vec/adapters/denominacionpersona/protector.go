package denominacionpersona

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const esquemaSobre = "vec.persona.denominacion.aead.v1"

func aead(k Clave) (cipher.AEAD, error) {
	b, err := aes.NewCipher(k.Material[:])
	if err != nil {
		return nil, ErrNoDisponible
	}
	return cipher.NewGCM(b)
}

func aad(s ports.SobreDenominacionPersona) []byte {
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], s.Version)
	i, _ := json.Marshal(s.Indice)
	return campos([]byte(s.Esquema), []byte(s.PersonaRef), v[:], []byte(s.ClaveRef), i)
}

func huellaSobre(s ports.SobreDenominacionPersona) string {
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (p *Protector) sobreValido(s ports.SobreDenominacionPersona) bool {
	if s.Esquema != esquemaSobre || !domain.ReferenciaPersonaDenominacionValida(s.PersonaRef) || s.Version == 0 || s.Version > 1<<53-1 || !referenciaValida(s.ClaveRef) || len(s.Nonce) != 12 || len(s.Cifrado) < 17 || len(s.Cifrado) > p.norma.MaxBytes+16 || !referenciaValida(s.Indice.AmbitoRef) || s.Indice.NormaRef != p.norma.Ref || s.Indice.NormaSHA256 != p.normaSHA || !referenciaValida(s.Indice.ClaveRef) || len(s.Indice.Tokens) == 0 || len(s.Indice.Tokens) > p.norma.MaxTokens {
		return false
	}
	for i, t := range s.Indice.Tokens {
		if len(t) != 32 || i > 0 && string(s.Indice.Tokens[i-1]) >= string(t) {
			return false
		}
	}
	return true
}

// claveIndiceRef es la versión anunciada por el registro gobernado. Una
// discrepancia cierra la operación; nunca cambia la versión silenciosamente.
func (p *Protector) PrepararDenominacionPersona(ctx context.Context, persona string, versionEsperada uint64, procedencia, ambito, claveIndiceRef string, nombre []byte) (ports.PreparacionDenominacionPersona, error) {
	if p == nil || !domain.ReferenciaPersonaDenominacionValida(persona) || versionEsperada >= 1<<53-1 || !referenciaValida(procedencia) || !domain.NombreDenominacionPersonaValido(nombre, p.norma.MaxBytes) {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	c, err := p.claves(ctx)
	if err != nil {
		return ports.PreparacionDenominacionPersona{}, err
	}
	defer borrarClaves(&c)
	if c.Busqueda.Ref != claveIndiceRef {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	i, err := p.indice(ambito, nombre, c.Busqueda)
	if err != nil {
		return ports.PreparacionDenominacionPersona{}, err
	}
	g, err := aead(c.Cifrado)
	if err != nil {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	s := ports.SobreDenominacionPersona{Esquema: esquemaSobre, PersonaRef: persona, Version: versionEsperada + 1, ClaveRef: c.Cifrado.Ref, Indice: i, Nonce: make([]byte, g.NonceSize())}
	if _, err = io.ReadFull(rand.Reader, s.Nonce); err != nil {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	s.Cifrado = g.Seal(nil, s.Nonce, nombre, aad(s))
	if p.revalidarClaves(ctx, c.Cifrado, c.Busqueda) != nil {
		return ports.PreparacionDenominacionPersona{}, ErrNoDisponible
	}
	return ports.PreparacionDenominacionPersona{PersonaRef: persona, VersionEsperada: versionEsperada, ProcedenciaRef: procedencia, Sobre: s, SobreSHA256: huellaSobre(s)}, nil
}

// Sólo el consumidor de una lectura autorizada puede prestar el nombre a la
// aplicación. Descifrar un sobre no autoriza a consultar Persona.
func (p *Protector) ConDenominacionDescifrada(ctx context.Context, s ports.SobreDenominacionPersona, usar func(domain.DenominacionPersona) error) error {
	if p == nil || usar == nil || !p.sobreValido(s) {
		return ErrNoDisponible
	}
	c, err := p.claves(ctx)
	if err != nil {
		return err
	}
	defer borrarClaves(&c)
	k, ok := buscarClave(c, s.ClaveRef, p.ahora())
	if !ok {
		return ErrNoDisponible
	}
	g, err := aead(k)
	if err != nil {
		return ErrNoDisponible
	}
	claro, err := g.Open(nil, s.Nonce, s.Cifrado, aad(s))
	if err != nil {
		return ErrNoDisponible
	}
	defer clear(claro)
	d, err := domain.NuevaDenominacionPersona(s.PersonaRef, s.Version, claro, p.norma.Ref, p.norma.MaxBytes)
	if err != nil || p.revalidarClaves(ctx, k, c.Busqueda) != nil {
		return ErrNoDisponible
	}
	return usar(d)
}

var _ ports.ProtectorDenominacionPersona = (*Protector)(nil)

// Comprueba la clave vigente contra el sobre auténtico después de revalidar
// Persona y autorización, inmediatamente antes de entregar el nombre.
func (p *Protector) RevalidarProteccionDenominacionPersona(ctx context.Context, s ports.SobreDenominacionPersona) error {
	return p.ConDenominacionDescifrada(ctx, s, func(domain.DenominacionPersona) error { return nil })
}
