package bootstrap

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"io"

	adminpostgres "vec-diputacion-granada/internal/modules/administracion/adapters/postgres"
)

const (
	claveSecretoCorreoDesarrolloRef = "clave:kms:desarrollo:correo-smtp:v1"
	dominioSecretoCorreoDesarrollo  = "vec.kms.desarrollo.correo-smtp.v1"
	esquemaSecretoCorreo            = "vec.administracion.configuracion-correo.secreto.v1"
	referenciaSecretoCorreo         = "configuracion:smtp:diputacion"
)

// protectorSecretoCorreoDesarrollo mantiene una subclave KMS privada fuera de
// PostgreSQL. Recibe el material ya cargado: no abre ficheros ni genera claves.
type protectorSecretoCorreoDesarrollo struct {
	clave     [sha256.Size]byte
	aleatorio io.Reader
}

func nuevoProtectorSecretoCorreoDesarrollo(
	claveMaestra [sha256.Size]byte,
	aleatorio io.Reader,
) (*protectorSecretoCorreoDesarrollo, error) {
	if subtle.ConstantTimeCompare(claveMaestra[:], make([]byte, sha256.Size)) == 1 ||
		dependenciaAdministracionNula(aleatorio) {
		return nil, ErrKMSDesarrolloNoDisponible
	}
	return &protectorSecretoCorreoDesarrollo{
		clave:     derivarClaveDesarrollo(claveMaestra, dominioSecretoCorreoDesarrollo),
		aleatorio: aleatorio,
	}, nil
}

func (protectorSecretoCorreoDesarrollo) String() string {
	return "bootstrap.protectorSecretoCorreoDesarrollo{redactado}"
}

func (protectorSecretoCorreoDesarrollo) GoString() string {
	return "bootstrap.protectorSecretoCorreoDesarrollo{redactado}"
}

func (protectorSecretoCorreoDesarrollo) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Redactado bool `json:"redactado"`
	}{true})
}

func (p *protectorSecretoCorreoDesarrollo) CifrarSecretoCorreo(
	ctx context.Context,
	claro, aad []byte,
) (adminpostgres.SobreSecretoCorreo, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || p.aleatorio == nil ||
		len(claro) == 0 || len(claro) > 16384 {
		return adminpostgres.SobreSecretoCorreo{}, ErrKMSDesarrolloNoDisponible
	}
	version, ok := versionAADSecretoCorreo(aad)
	if !ok {
		return adminpostgres.SobreSecretoCorreo{}, ErrKMSDesarrolloNoDisponible
	}
	bloque, err := aes.NewCipher(p.clave[:])
	if err != nil {
		return adminpostgres.SobreSecretoCorreo{}, ErrKMSDesarrolloNoDisponible
	}
	aead, err := cipher.NewGCM(bloque)
	if err != nil {
		return adminpostgres.SobreSecretoCorreo{}, ErrKMSDesarrolloNoDisponible
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(p.aleatorio, nonce); err != nil {
		borrarBytes(nonce)
		return adminpostgres.SobreSecretoCorreo{}, ErrKMSDesarrolloNoDisponible
	}
	cifrado := aead.Seal(nil, nonce, claro, aad)
	if ctx.Err() != nil {
		borrarBytes(nonce)
		borrarBytes(cifrado)
		return adminpostgres.SobreSecretoCorreo{}, ErrKMSDesarrolloNoDisponible
	}
	return adminpostgres.SobreSecretoCorreo{
		Version: version, ClaveRef: claveSecretoCorreoDesarrolloRef,
		Nonce: nonce, Cifrado: cifrado,
	}, nil
}

// ConSecretoCorreoDescifrado entrega el texto claro sólo al callback y borra
// ese buffer al retornar, también si el callback falla.
func (p *protectorSecretoCorreoDesarrollo) ConSecretoCorreoDescifrado(
	ctx context.Context,
	sobre adminpostgres.SobreSecretoCorreo,
	aad []byte,
	usar func([]byte) error,
) error {
	if p == nil || ctx == nil || ctx.Err() != nil || usar == nil ||
		sobre.Version == 0 || !textoConstanteIgual(sobre.ClaveRef, claveSecretoCorreoDesarrolloRef) {
		return ErrKMSDesarrolloNoDisponible
	}
	version, ok := versionAADSecretoCorreo(aad)
	if !ok || version != sobre.Version {
		return ErrKMSDesarrolloNoDisponible
	}
	bloque, err := aes.NewCipher(p.clave[:])
	if err != nil {
		return ErrKMSDesarrolloNoDisponible
	}
	aead, err := cipher.NewGCM(bloque)
	if err != nil || len(sobre.Nonce) != aead.NonceSize() || len(sobre.Cifrado) < aead.Overhead() {
		return ErrKMSDesarrolloNoDisponible
	}
	claro, err := aead.Open(nil, sobre.Nonce, sobre.Cifrado, aad)
	if err != nil || len(claro) == 0 || ctx.Err() != nil {
		borrarBytes(claro)
		return ErrKMSDesarrolloNoDisponible
	}
	defer borrarBytes(claro)
	if err := usar(claro); err != nil {
		return ErrKMSDesarrolloNoDisponible
	}
	if ctx.Err() != nil {
		return ErrKMSDesarrolloNoDisponible
	}
	return nil
}

func versionAADSecretoCorreo(aad []byte) (uint64, bool) {
	var declarado struct {
		Esquema, Referencia string
		Version             uint64
	}
	if len(aad) == 0 || len(aad) > 4096 || json.Unmarshal(aad, &declarado) != nil ||
		declarado.Version == 0 ||
		!textoConstanteIgual(declarado.Esquema, esquemaSecretoCorreo) ||
		!textoConstanteIgual(declarado.Referencia, referenciaSecretoCorreo) {
		return 0, false
	}
	canonico, err := json.Marshal(declarado)
	if err != nil || len(canonico) != len(aad) || subtle.ConstantTimeCompare(canonico, aad) != 1 {
		borrarBytes(canonico)
		return 0, false
	}
	borrarBytes(canonico)
	return declarado.Version, true
}

var _ adminpostgres.ProtectorSecretoCorreo = (*protectorSecretoCorreoDesarrollo)(nil)
