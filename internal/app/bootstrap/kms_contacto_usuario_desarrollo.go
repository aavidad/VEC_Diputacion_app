package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"reflect"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const dominioKMSContactoUsuarioDesarrollo = "vec.kms.desarrollo.contacto-usuario.v1"
const claveKMSContactoUsuarioDesarrolloRef = "clave:kms:desarrollo:contacto-usuario:v1"

var errKMSContactoUsuarioNoDisponible = errors.New("bootstrap: KMS de contacto de usuario no disponible")

// CifrarContactoUsuario usa una subclave del KMS distinta de SMTP y liga el
// sobre a la persona VEC y a su versión. Nunca serializa el valor claro.
func (e *emisorKMSDesarrollo) CifrarContactoUsuario(ctx context.Context, sujeto string, version uint64, claro []byte) (vecports.SobreContactoUsuario, error) {
	if e == nil || ctx == nil || ctx.Err() != nil || dependenciaContactoKMSNula(e.aleatorio) || claveContactoKMSCero(e.claveEnvoltura) || !sujetoContactoKMSValido(sujeto) || version == 0 || len(claro) == 0 || len(claro) > 320 {
		return vecports.SobreContactoUsuario{}, errKMSContactoUsuarioNoDisponible
	}
	aad, err := aadContactoUsuarioDesarrollo(sujeto, version)
	if err != nil {
		return vecports.SobreContactoUsuario{}, errKMSContactoUsuarioNoDisponible
	}
	defer borrarBytes(aad)
	clave := claveContactoUsuarioDesarrollo(e.claveEnvoltura)
	defer borrarBytes(clave[:])
	b, err := aes.NewCipher(clave[:])
	if err != nil {
		return vecports.SobreContactoUsuario{}, errKMSContactoUsuarioNoDisponible
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return vecports.SobreContactoUsuario{}, errKMSContactoUsuarioNoDisponible
	}
	nonce := make([]byte, g.NonceSize())
	if _, err = io.ReadFull(e.aleatorio, nonce); err != nil {
		borrarBytes(nonce)
		return vecports.SobreContactoUsuario{}, errKMSContactoUsuarioNoDisponible
	}
	cifrado := g.Seal(nil, nonce, claro, aad)
	if ctx.Err() != nil {
		borrarBytes(nonce)
		borrarBytes(cifrado)
		return vecports.SobreContactoUsuario{}, errKMSContactoUsuarioNoDisponible
	}
	return vecports.SobreContactoUsuario{Version: version, ClaveRef: claveKMSContactoUsuarioDesarrolloRef, Nonce: nonce, Cifrado: cifrado}, nil
}

func (e *emisorKMSDesarrollo) ConContactoUsuarioDescifrado(ctx context.Context, sujeto string, sobre vecports.SobreContactoUsuario, usar func([]byte) error) error {
	if e == nil || ctx == nil || ctx.Err() != nil || usar == nil || dependenciaContactoKMSNula(e.aleatorio) || claveContactoKMSCero(e.claveEnvoltura) || !sujetoContactoKMSValido(sujeto) || sobre.Version == 0 || sobre.ClaveRef != claveKMSContactoUsuarioDesarrolloRef || len(sobre.Nonce) != 12 || len(sobre.Cifrado) < 16 || len(sobre.Cifrado) > 336 {
		return errKMSContactoUsuarioNoDisponible
	}
	aad, err := aadContactoUsuarioDesarrollo(sujeto, sobre.Version)
	if err != nil {
		return errKMSContactoUsuarioNoDisponible
	}
	defer borrarBytes(aad)
	clave := claveContactoUsuarioDesarrollo(e.claveEnvoltura)
	defer borrarBytes(clave[:])
	b, err := aes.NewCipher(clave[:])
	if err != nil {
		return errKMSContactoUsuarioNoDisponible
	}
	g, err := cipher.NewGCM(b)
	if err != nil {
		return errKMSContactoUsuarioNoDisponible
	}
	claro, err := g.Open(nil, sobre.Nonce, sobre.Cifrado, aad)
	if err != nil || ctx.Err() != nil {
		borrarBytes(claro)
		return errKMSContactoUsuarioNoDisponible
	}
	defer borrarBytes(claro)
	if err = usar(claro); err != nil || ctx.Err() != nil {
		return errKMSContactoUsuarioNoDisponible
	}
	return nil
}

func claveContactoUsuarioDesarrollo(base [sha256.Size]byte) [sha256.Size]byte {
	return derivarClaveDesarrollo(base, dominioKMSContactoUsuarioDesarrollo)
}
func aadContactoUsuarioDesarrollo(sujeto string, version uint64) ([]byte, error) {
	if !sujetoContactoKMSValido(sujeto) || version == 0 {
		return nil, errKMSContactoUsuarioNoDisponible
	}
	return json.Marshal(struct {
		Esquema string `json:"esquema"`
		Sujeto  string `json:"sujeto_ref"`
		Version uint64 `json:"version"`
	}{"vec.contacto_usuario.secreto.v1", sujeto, version})
}
func sujetoContactoKMSValido(s string) bool {
	return vecdomain.ReferenciaSujetoContactoUsuarioValida(s)
}
func claveContactoKMSCero(v [sha256.Size]byte) bool { var cero [sha256.Size]byte; return v == cero }
func dependenciaContactoKMSNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	return (r.Kind() == reflect.Ptr || r.Kind() == reflect.Interface || r.Kind() == reflect.Func || r.Kind() == reflect.Map || r.Kind() == reflect.Slice) && r.IsNil()
}

var _ vecports.ProtectorContactoUsuario = (*emisorKMSDesarrollo)(nil)
