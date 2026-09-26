package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	seleccionports "vec-diputacion-granada/internal/modules/seleccion/ports"
)

// Selección: los datos personales de la solicitud se cifran con una subclave
// del KMS de desarrollo distinta de las de contacto, ligada por los datos
// asociados a la persona, la convocatoria y la versión. Las huellas del
// documento y del material de idempotencia usan otra subclave (HMAC), de modo
// que la base nunca guarda ni compara datos en claro.
const (
	dominioKMSDatosSolicitudSeleccionDesarrollo = "vec.kms.desarrollo.datos-solicitud-seleccion.v1"
	dominioKMSHuellaSeleccionDesarrollo         = "vec.kms.desarrollo.huella-seleccion.v1"
	claveRefSobreSeleccionDesarrollo            = "clave:kms:desarrollo:seleccion:v1"
	maximoClaroSeleccionDesarrollo              = 16 * 1024
)

var (
	errKMSSeleccionNoDisponible = errors.New("bootstrap: KMS de Selección no disponible")
	personaKMSSeleccionValida   = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
	dominioHuellaSeleccion      = regexp.MustCompile(`^seleccion\.[a-z_]{3,64}\.v[0-9]{1,3}$`)
)

var _ seleccionports.ProtectorDatosSolicitud = (*emisorKMSDesarrollo)(nil)

func aadSeleccionDesarrollo(a seleccionports.AsociacionDatos) ([]byte, error) {
	if !personaKMSSeleccionValida.MatchString(a.PersonaRef) || a.ConvocatoriaRef == "" || len(a.ConvocatoriaRef) > 192 || a.Version < 1 {
		return nil, errKMSSeleccionNoDisponible
	}
	return json.Marshal(struct {
		Esquema      string `json:"esquema"`
		Persona      string `json:"persona_ref"`
		Convocatoria string `json:"convocatoria_ref"`
		Version      int    `json:"version"`
	}{"vec.seleccion.datos_solicitud.secreto.v1", a.PersonaRef, a.ConvocatoriaRef, a.Version})
}

func (e *emisorKMSDesarrollo) aeadSeleccion() (cipher.AEAD, error) {
	clave := derivarClaveDesarrollo(e.claveEnvoltura, dominioKMSDatosSolicitudSeleccionDesarrollo)
	defer borrarBytes(clave[:])
	b, err := aes.NewCipher(clave[:])
	if err != nil {
		return nil, errors.Join(errKMSSeleccionNoDisponible, err)
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return nil, errors.Join(errKMSSeleccionNoDisponible, err)
	}
	return g, nil
}

// CifrarDatosSolicitud cifra el canónico de los datos personales.
func (e *emisorKMSDesarrollo) CifrarDatosSolicitud(ctx context.Context, a seleccionports.AsociacionDatos, claro []byte) (seleccionports.SobreDatos, error) {
	if e == nil || ctx == nil || ctx.Err() != nil || dependenciaContactoKMSNula(e.aleatorio) || claveContactoKMSCero(e.claveEnvoltura) ||
		len(claro) == 0 || len(claro) > maximoClaroSeleccionDesarrollo {
		return seleccionports.SobreDatos{}, errKMSSeleccionNoDisponible
	}
	aad, err := aadSeleccionDesarrollo(a)
	if err != nil {
		return seleccionports.SobreDatos{}, err
	}
	defer borrarBytes(aad)
	g, err := e.aeadSeleccion()
	if err != nil {
		return seleccionports.SobreDatos{}, err
	}
	nonce := make([]byte, g.NonceSize())
	if _, err := io.ReadFull(e.aleatorio, nonce); err != nil {
		return seleccionports.SobreDatos{}, errors.Join(errKMSSeleccionNoDisponible, err)
	}
	return seleccionports.SobreDatos{ClaveRef: claveRefSobreSeleccionDesarrollo, Nonce: nonce, Cifrado: g.Seal(nil, nonce, claro, aad)}, nil
}

// ConDatosSolicitudDescifrados entrega el claro solo dentro del callback y lo
// borra al volver.
func (e *emisorKMSDesarrollo) ConDatosSolicitudDescifrados(ctx context.Context, a seleccionports.AsociacionDatos, s seleccionports.SobreDatos, usar func([]byte) error) error {
	if e == nil || ctx == nil || ctx.Err() != nil || usar == nil || claveContactoKMSCero(e.claveEnvoltura) ||
		s.ClaveRef != claveRefSobreSeleccionDesarrollo || len(s.Nonce) != 12 || len(s.Cifrado) < 17 || len(s.Cifrado) > maximoClaroSeleccionDesarrollo+16 {
		return errKMSSeleccionNoDisponible
	}
	aad, err := aadSeleccionDesarrollo(a)
	if err != nil {
		return err
	}
	defer borrarBytes(aad)
	g, err := e.aeadSeleccion()
	if err != nil {
		return err
	}
	claro, err := g.Open(nil, s.Nonce, s.Cifrado, aad)
	if err != nil {
		return errors.Join(errKMSSeleccionNoDisponible, err)
	}
	defer borrarBytes(claro)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return usar(claro)
}

// HuellaConClave es un HMAC-SHA256 con subclave por dominio: no se puede
// recalcular sin el KMS (un documento de identidad no se adivina por fuerza
// bruta desde la base).
func (e *emisorKMSDesarrollo) HuellaConClave(ctx context.Context, dominio string, datos []byte) (string, error) {
	if e == nil || ctx == nil || ctx.Err() != nil || claveContactoKMSCero(e.claveEnvoltura) || !dominioHuellaSeleccion.MatchString(dominio) || len(datos) > 64*1024 {
		return "", errKMSSeleccionNoDisponible
	}
	clave := derivarClaveDesarrollo(e.claveEnvoltura, dominioKMSHuellaSeleccionDesarrollo+"\x00"+dominio)
	defer borrarBytes(clave[:])
	mac := hmac.New(sha256.New, clave[:])
	if _, err := mac.Write(datos); err != nil {
		return "", errors.Join(errKMSSeleccionNoDisponible, err)
	}
	return hex.EncodeToString(mac.Sum(nil)), nil
}
