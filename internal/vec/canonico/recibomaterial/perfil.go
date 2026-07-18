package recibomaterial

import (
	"crypto/sha256"

	almacencanonico "vec-diputacion-granada/internal/vec/canonico/almacen"
)

// Perfil describe exclusivamente capacidades materiales, sin topologia fisica.
type Perfil struct {
	Esquema                string
	VersionEsquema         uint16
	Referencia             string
	Version                uint32
	ConectorLogicoID       string
	EscrituraEnFlujo       bool
	ReferenciasOpacas      bool
	IntegridadSHA256       bool
	Versionado             bool
	Retencion              bool
	BloqueoLegal           bool
	CifradoEnTransito      bool
	CifradoEnReposo        bool
	CifradoPorObjeto       bool
	PreservaObjetoOriginal bool
	TamanoMaximoObjeto     int64
}

func PerfilValido(p Perfil) bool {
	return p.Esquema == EsquemaPerfil && p.VersionEsquema == EsquemaVersion &&
		AliasLogicoValido(p.Referencia, 512) && p.Version > 0 &&
		AliasLogicoValido(p.ConectorLogicoID, 128) && p.EscrituraEnFlujo &&
		p.ReferenciasOpacas && p.IntegridadSHA256 && p.Versionado &&
		p.CifradoEnTransito && p.CifradoEnReposo && p.TamanoMaximoObjeto > 0
}

func CanonicoPerfil(p Perfil) ([]byte, error) {
	if !PerfilValido(p) {
		return nil, ErrReciboNoValido
	}
	var canonico []byte
	canonico = AnexarTLV(canonico, 0, []byte(p.Esquema))
	canonico = AnexarTLV(canonico, 1, Uint16(p.VersionEsquema))
	canonico = AnexarTLV(canonico, 2, []byte(p.Referencia))
	canonico = AnexarTLV(canonico, 3, Uint32(p.Version))
	canonico = AnexarTLV(canonico, 4, []byte(p.ConectorLogicoID))
	canonico = AnexarTLV(canonico, 5, Bool(p.EscrituraEnFlujo))
	canonico = AnexarTLV(canonico, 6, Bool(p.ReferenciasOpacas))
	canonico = AnexarTLV(canonico, 7, Bool(p.IntegridadSHA256))
	canonico = AnexarTLV(canonico, 8, Bool(p.Versionado))
	canonico = AnexarTLV(canonico, 9, Bool(p.Retencion))
	canonico = AnexarTLV(canonico, 10, Bool(p.BloqueoLegal))
	canonico = AnexarTLV(canonico, 11, Bool(p.CifradoEnTransito))
	canonico = AnexarTLV(canonico, 12, Bool(p.CifradoEnReposo))
	canonico = AnexarTLV(canonico, 13, Bool(p.CifradoPorObjeto))
	canonico = AnexarTLV(canonico, 14, Bool(p.PreservaObjetoOriginal))
	canonico = AnexarTLV(canonico, 15, Int64(p.TamanoMaximoObjeto))
	return canonico, nil
}

func HuellaPerfil(p Perfil) ([sha256.Size]byte, error) {
	canonico, err := CanonicoPerfil(p)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(canonico), nil
}

func PerfilCotejaCapacidades(p Perfil, c almacencanonico.Capacidades) bool {
	return PerfilValido(p) && p.ConectorLogicoID == c.ConectorID &&
		p.EscrituraEnFlujo == c.EscrituraEnFlujo && p.ReferenciasOpacas == c.ReferenciasOpacas &&
		p.IntegridadSHA256 == c.IntegridadSHA256 && p.Versionado == c.Versionado &&
		p.Retencion == c.Retencion && p.BloqueoLegal == c.BloqueoLegal &&
		p.CifradoEnTransito == c.CifradoEnTransito && p.CifradoEnReposo == c.CifradoEnReposo &&
		p.CifradoPorObjeto == c.CifradoPorObjeto && p.PreservaObjetoOriginal == c.PreservaObjetoOriginal &&
		p.TamanoMaximoObjeto == c.TamanoMaximoObjeto
}
