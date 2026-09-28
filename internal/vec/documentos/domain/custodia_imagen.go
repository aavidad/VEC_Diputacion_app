package domain

import (
	"errors"
	"math"
	"strings"
)

var ErrCustodiaImagenInvalida = errors.New("documentos: custodia de imagen invalida")

type EstadoCustodiaImagen string

const (
	EstadoImagenReservada  EstadoCustodiaImagen = "reservada"
	EstadoImagenCuarentena EstadoCustodiaImagen = "cuarentena"
	EstadoImagenAdmitida   EstadoCustodiaImagen = "admitida"
	EstadoImagenConfirmada EstadoCustodiaImagen = "confirmada"
)

// La reserva identifica una única intención de Usuarios. La clave se reclama
// de forma única por persona, y la huella distingue un replay de una colisión.
type IdentidadCustodiaImagen struct {
	PersonaRef         string
	PerfilRef          string
	Audiencia          string
	Finalidad          string
	VersionEsperada    uint64
	CatalogoVersionRef string
	Paleta             string
	ClaveOperacion     string
	HuellaPeticion     string
	OriginalSHA256     string
	ContenidoSHA256    string
	DocumentoRef       string
}

func (i IdentidadCustodiaImagen) Validar() error {
	if !referenciaImagen(i.PersonaRef, "per_", 96) || !referenciaImagen(i.PerfilRef, "prf_", 96) ||
		(i.Audiencia != "portal_personal_autenticado" && i.Audiencia != "portal_interno_autenticado") ||
		i.Finalidad != "finalidad:usuarios:imagen-propia:v1" || i.VersionEsperada >= math.MaxInt64 ||
		!codigoImagen(i.CatalogoVersionRef, 96) || !codigoImagen(i.Paleta, 96) ||
		!referenciaImagen(i.ClaveOperacion, "", 128) || len(i.ClaveOperacion) < 16 ||
		!sha256Imagen(i.HuellaPeticion) || !sha256Imagen(i.OriginalSHA256) || !sha256Imagen(i.ContenidoSHA256) ||
		(i.DocumentoRef != "" && !referenciaImagen(i.DocumentoRef, "", 128)) {
		return ErrCustodiaImagenInvalida
	}
	return nil
}

func (i IdentidadCustodiaImagen) MismaPeticion(otra IdentidadCustodiaImagen) bool {
	return i.PersonaRef == otra.PersonaRef && i.PerfilRef == otra.PerfilRef && i.Audiencia == otra.Audiencia &&
		i.Finalidad == otra.Finalidad && i.VersionEsperada == otra.VersionEsperada &&
		i.CatalogoVersionRef == otra.CatalogoVersionRef && i.Paleta == otra.Paleta &&
		i.ClaveOperacion == otra.ClaveOperacion &&
		i.HuellaPeticion == otra.HuellaPeticion && i.OriginalSHA256 == otra.OriginalSHA256 &&
		i.ContenidoSHA256 == otra.ContenidoSHA256
}

func (e EstadoCustodiaImagen) Valido() bool {
	switch e {
	case EstadoImagenReservada, EstadoImagenCuarentena, EstadoImagenAdmitida, EstadoImagenConfirmada:
		return true
	}
	return false
}

func (e EstadoCustodiaImagen) PuedePasarA(siguiente EstadoCustodiaImagen) bool {
	switch e {
	case EstadoImagenReservada:
		return siguiente == EstadoImagenCuarentena
	case EstadoImagenCuarentena:
		return siguiente == EstadoImagenAdmitida
	case EstadoImagenAdmitida:
		return siguiente == EstadoImagenConfirmada
	}
	return false
}

func sha256Imagen(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune("0123456789abcdef", c) {
			return false
		}
	}
	return true
}
func referenciaImagen(s, prefijo string, max int) bool {
	if len(s) < 16 || len(s) > max || !strings.HasPrefix(s, prefijo) {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' || c == ':') {
			return false
		}
	}
	return true
}
func codigoImagen(s string, max int) bool {
	if len(s) == 0 || len(s) > max {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
