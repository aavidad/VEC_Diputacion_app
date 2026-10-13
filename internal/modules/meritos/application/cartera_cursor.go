package application

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
)

var ErrCursorCartera = errors.New("meritos.error.cursor_cartera_invalido")

const vigenciaCursorCartera = 10 * time.Minute

type cursorCartera struct {
	PersonaRef  string    `json:"persona_ref"`
	Filtro      string    `json:"filtro"`
	Idioma      string    `json:"idioma"`
	Limite      int       `json:"limite"`
	CorteEn     time.Time `json:"corte_en"`
	UltimoHecho string    `json:"ultimo_hecho_ref"`
	CaducaEn    time.Time `json:"caduca_en"`
}

// SelladorCursorCartera impide cambiar el corte, el filtro o la persona entre
// páginas. Su clave vive únicamente en memoria: tras reiniciar se consulta de
// nuevo la primera página con el filtro conservado en la URL.
type SelladorCursorCartera struct{ clave [32]byte }

func NuevoSelladorCursorCartera() (*SelladorCursorCartera, error) {
	s := new(SelladorCursorCartera)
	if _, err := rand.Read(s.clave[:]); err != nil {
		return nil, errors.Join(ErrCursorCartera, err)
	}
	return s, nil
}

func (s *SelladorCursorCartera) Sellar(persona, filtro, idioma string, limite int,
	corte time.Time, ultimo string, ahora time.Time) (string, error) {
	if s == nil || !idiomaCarteraValido(idioma) || !cursorCarteraValido(persona, filtro, limite, corte, ultimo, ahora) {
		return "", ErrCursorCartera
	}
	c := cursorCartera{PersonaRef: persona, Filtro: filtro, Idioma: idioma, Limite: limite,
		CorteEn: corte.UTC().Truncate(time.Microsecond), UltimoHecho: ultimo,
		CaducaEn: corte.UTC().Truncate(time.Microsecond).Add(vigenciaCursorCartera)}
	b, err := json.Marshal(c)
	if err != nil {
		return "", errors.Join(ErrCursorCartera, err)
	}
	mac := hmac.New(sha256.New, s.clave[:])
	if _, err := mac.Write(b); err != nil {
		return "", errors.Join(ErrCursorCartera, err)
	}
	return base64.RawURLEncoding.EncodeToString(b) + "." +
		base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *SelladorCursorCartera) Abrir(token, persona, filtro, idioma string, limite int,
	ahora time.Time) (time.Time, string, error) {
	if s == nil || len(token) == 0 || len(token) > 2048 || !domain.ReferenciaValida(persona) ||
		!filtroCarteraValido(filtro) || !idiomaCarteraValido(idioma) ||
		limite < 1 || limite > 50 || ahora.IsZero() {
		return time.Time{}, "", ErrCursorCartera
	}
	separador := -1
	for i := range token {
		if token[i] == '.' {
			if separador != -1 {
				return time.Time{}, "", ErrCursorCartera
			}
			separador = i
		}
	}
	if separador < 1 || separador == len(token)-1 {
		return time.Time{}, "", ErrCursorCartera
	}
	b, err := base64.RawURLEncoding.DecodeString(token[:separador])
	if err != nil || len(b) == 0 || len(b) > 1024 {
		return time.Time{}, "", ErrCursorCartera
	}
	firma, err := base64.RawURLEncoding.DecodeString(token[separador+1:])
	if err != nil || len(firma) != sha256.Size {
		return time.Time{}, "", ErrCursorCartera
	}
	mac := hmac.New(sha256.New, s.clave[:])
	if _, err := mac.Write(b); err != nil || !hmac.Equal(firma, mac.Sum(nil)) {
		return time.Time{}, "", ErrCursorCartera
	}
	var c cursorCartera
	if err := json.Unmarshal(b, &c); err != nil || c.PersonaRef != persona || c.Filtro != filtro || c.Idioma != idioma ||
		c.Limite != limite || !cursorCarteraValido(persona, filtro, limite, c.CorteEn, c.UltimoHecho, ahora) ||
		!c.CaducaEn.After(ahora) || c.CaducaEn.After(c.CorteEn.Add(vigenciaCursorCartera)) {
		return time.Time{}, "", ErrCursorCartera
	}
	return c.CorteEn, c.UltimoHecho, nil
}

func cursorCarteraValido(persona, filtro string, limite int, corte time.Time,
	ultimo string, ahora time.Time) bool {
	return domain.ReferenciaValida(persona) && filtroCarteraValido(filtro) &&
		limite >= 1 && limite <= 50 && domain.ReferenciaValida(ultimo) &&
		!corte.IsZero() && !ahora.IsZero() && !corte.After(ahora) &&
		ahora.Before(corte.Add(vigenciaCursorCartera))
}

func filtroCarteraValido(filtro string) bool {
	return filtro == "todos" || filtro == "disponibles" || filtro == "retirados"
}

func idiomaCarteraValido(idioma string) bool { return idioma == "es" || idioma == "en" }
