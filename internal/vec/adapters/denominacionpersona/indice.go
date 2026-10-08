package denominacionpersona

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
	"vec-diputacion-granada/internal/vec/ports"
)

func normaValida(n ports.NormaDenominacionPersona) bool {
	if !referenciaValida(n.Ref) || (n.Case != "exact" && n.Case != "fold") || n.FormaUnicode != "NFC" || !utf8.ValidString(n.Separadores) || len(n.Separadores) == 0 || len(n.Separadores) > 128 || n.MaxBytes < 1 || n.MaxBytes > 4096 || n.MaxTokens < 1 || n.MaxTokens > 32 {
		return false
	}
	for _, r := range n.Separadores {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func huellaNorma(n ports.NormaDenominacionPersona) string {
	b, _ := json.Marshal(n)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func campos(valores ...[]byte) []byte {
	var b []byte
	for _, v := range valores {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(v)))
		b = append(b, n[:]...)
		b = append(b, v...)
	}
	return b
}

func (p *Protector) indice(ambito string, claro []byte, k Clave) (ports.IndiceDenominacionPersona, error) {
	if !referenciaValida(ambito) || len(claro) == 0 || len(claro) > p.norma.MaxBytes || !utf8.Valid(claro) {
		return ports.IndiceDenominacionPersona{}, ErrNoDisponible
	}
	s := norm.NFC.String(string(claro))
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ports.IndiceDenominacionPersona{}, ErrNoDisponible
		}
	}
	if p.norma.Case == "fold" {
		s = norm.NFC.String(cases.Fold().String(s))
	}
	palabras := strings.FieldsFunc(s, func(r rune) bool { return strings.ContainsRune(p.norma.Separadores, r) })
	if len(palabras) == 0 || len(palabras) > p.norma.MaxTokens {
		return ports.IndiceDenominacionPersona{}, ErrNoDisponible
	}
	indice := ports.IndiceDenominacionPersona{AmbitoRef: ambito, NormaRef: p.norma.Ref, NormaSHA256: p.normaSHA, ClaveRef: k.Ref}
	vistos := map[string]bool{}
	for _, palabra := range palabras {
		h := hmac.New(sha256.New, k.Material[:])
		_, _ = h.Write(campos([]byte("vec.persona.denominacion.token.v1"), []byte(ambito), []byte(p.norma.Ref), []byte(p.normaSHA), []byte(k.Ref), []byte(palabra)))
		t := h.Sum(nil)
		if !vistos[string(t)] {
			indice.Tokens = append(indice.Tokens, t)
			vistos[string(t)] = true
		}
	}
	sort.Slice(indice.Tokens, func(i, j int) bool { return string(indice.Tokens[i]) < string(indice.Tokens[j]) })
	return indice, nil
}

func (p *Protector) PrepararBusquedaDenominacionPersona(ctx context.Context, ambito, claveIndiceRef string, busqueda []byte) (ports.IndiceDenominacionPersona, error) {
	c, err := p.claves(ctx)
	if err != nil {
		return ports.IndiceDenominacionPersona{}, err
	}
	defer borrarClaves(&c)
	if c.Busqueda.Ref != claveIndiceRef {
		return ports.IndiceDenominacionPersona{}, ErrNoDisponible
	}
	i, err := p.indice(ambito, busqueda, c.Busqueda)
	if err != nil || p.revalidarClaves(ctx, c.Cifrado, c.Busqueda) != nil {
		return ports.IndiceDenominacionPersona{}, ErrNoDisponible
	}
	return i, nil
}
