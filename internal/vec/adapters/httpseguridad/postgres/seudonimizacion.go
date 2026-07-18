// Package postgres implementa el registro durable de sesiones HTTP sobre la
// fuente V1 de autorizacion. No abre conexiones, no conserva DSN ni incorpora
// claves HMAC: la seudonimizacion procede de un conector HSM/KMS obligatorio.
package postgres

import (
	"context"
	"net/url"
	"strings"
	"unicode/utf8"
)

const EsquemaHMACSHA256V1 = "vec.identidad.hmac-sha256.v1"

// IdentificadoresAlta limita los datos que recibe el conector HMAC. Cada campo
// debe usar una etiqueta de proposito distinta dentro del dominio configurado.
type IdentificadoresAlta struct {
	EspacioIdentidad  string
	AsercionID        string
	SesionID          string
	SujetoID          string
	CuentaID          string
	CuentaOrdinariaID string
}

// SeudonimosAlta contiene exclusivamente digests y coordenadas de clave. El
// material secreto y los identificadores fuente nunca llegan a PostgreSQL.
type SeudonimosAlta struct {
	Esquema               string
	EspacioIdentidad      string
	DominioRef            string
	ClaveID               string
	ClaveVersion          uint64
	AsercionIDHMAC        [32]byte
	SesionIDHMAC          [32]byte
	SujetoIDHMAC          [32]byte
	CuentaIDHMAC          [32]byte
	CuentaOrdinariaIDHMAC [32]byte
}

// SeudonimizadorAlta debe usar HMAC-SHA-256 dentro de HSM/KMS, separar los
// cinco propositos y ligar el dominio a un emisor/namespace de identidad. No
// existe implementacion software por defecto: sin conector real el arranque
// de esta frontera debe fallar.
type SeudonimizadorAlta interface {
	SeudonimizarAlta(context.Context, IdentificadoresAlta) (SeudonimosAlta, error)
}

func (s SeudonimosAlta) valida(
	espacioEsperado, dominioEsperado string,
	requiereOrdinaria bool,
) bool {
	if s.Esquema != EsquemaHMACSHA256V1 || s.EspacioIdentidad != espacioEsperado ||
		s.DominioRef != dominioEsperado ||
		!referenciaTecnicaValida(s.DominioRef, "idh_") ||
		!textoTecnicoValido(s.ClaveID, 128) || s.ClaveVersion == 0 ||
		s.ClaveVersion > uint64(1<<63-1) ||
		huellaNula(s.AsercionIDHMAC) || huellaNula(s.SesionIDHMAC) ||
		huellaNula(s.SujetoIDHMAC) || huellaNula(s.CuentaIDHMAC) {
		return false
	}
	if requiereOrdinaria {
		if huellaNula(s.CuentaOrdinariaIDHMAC) {
			return false
		}
	} else if !huellaNula(s.CuentaOrdinariaIDHMAC) {
		return false
	}
	huellas := [][32]byte{
		s.AsercionIDHMAC, s.SesionIDHMAC, s.SujetoIDHMAC, s.CuentaIDHMAC,
	}
	if requiereOrdinaria {
		huellas = append(huellas, s.CuentaOrdinariaIDHMAC)
	}
	for primera := range huellas {
		for segunda := primera + 1; segunda < len(huellas); segunda++ {
			if huellas[primera] == huellas[segunda] {
				return false
			}
		}
	}
	return true
}

func huellaNula(huella [32]byte) bool { return huella == [32]byte{} }

func textoTecnicoValido(valor string, maximo int) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo ||
		!utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if caracter <= 0x20 || caracter >= 0x7f {
			return false
		}
	}
	return true
}

func referenciaTecnicaValida(valor, prefijo string) bool {
	if !strings.HasPrefix(valor, prefijo) || len(valor) < len(prefijo)+22 ||
		len(valor) > len(prefijo)+128 {
		return false
	}
	for _, caracter := range valor[len(prefijo):] {
		if (caracter < 'a' || caracter > 'z') &&
			(caracter < 'A' || caracter > 'Z') &&
			(caracter < '0' || caracter > '9') && caracter != '_' && caracter != '-' {
			return false
		}
	}
	return true
}

func espacioIdentidadValido(valor string) bool {
	if !textoTecnicoValido(valor, 512) {
		return false
	}
	emisor, err := url.Parse(valor)
	return err == nil && strings.EqualFold(emisor.Scheme, "https") &&
		emisor.Host != "" && emisor.User == nil && emisor.RawQuery == "" &&
		emisor.Fragment == ""
}
