// Package verificacioncose aplica el perfil criptografico comun de COSE_Sign1.
// Solo comprueba forma canonica y firma contra una clave aportada: no gobierna
// confianza, revocacion, audiencia, vigencia ni consumo de una autorizacion.
package verificacioncose

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"

	gocose "github.com/veraison/go-cose"
)

var (
	ErrSobreSign1Invalido            = errors.New("vec: sobre COSE Sign1 estricto invalido")
	ErrConfiguracionClaveInvalida    = errors.New("vec: configuracion de clave COSE Sign1 invalida")
	ErrVerificacionFirmaSign1Fallida = errors.New("vec: verificacion de firma COSE Sign1 fallida")
	ErrSerializacionCOSEProhibida    = errors.New("vec: serializacion de verificacion COSE Sign1 prohibida")
)

const (
	TamanoMaximoAbsolutoSobreSign1 = 1024 * 1024
	tamanoMinimoSobreSign1         = 16
	tamanoMaximoClaveID            = 128
	tamanoMaximoPayload            = 512 * 1024
	tamanoMaximoAADExterno         = 1024
	marcaSobreSign1Estricto        = "vec.sobre-cose-sign1-estricto.v1"
	marcaVerificadorClaveSign1     = "vec.verificador-clave-cose-sign1.v1"
)

// Algoritmo es la lista positiva comun. Un protocolo consumidor puede
// restringirla aun mas, pero nunca ampliarla desde datos del sobre.
type Algoritmo string

const (
	AlgoritmoEdDSA Algoritmo = "EdDSA"
	AlgoritmoES256 Algoritmo = "ES256"
)

func (a Algoritmo) valida() bool {
	return a == AlgoritmoEdDSA || a == AlgoritmoES256
}

// SobreSign1Estricto es una inspeccion nominal e inmutable. No contiene una
// raiz de confianza y superar su construccion no acredita procedencia.
type SobreSign1Estricto struct {
	marca     string
	contenido []byte
	mensaje   gocose.Sign1Message
	algoritmo Algoritmo
	claveID   []byte
}

// InspeccionarSobreSign1 exige CBOR determinista, exactamente alg y kid como
// cabeceras protegidas, ninguna cabecera no protegida y firma canonica. El
// limite pertenece al protocolo consumidor y queda sujeto a un techo absoluto.
func InspeccionarSobreSign1(
	contenido []byte,
	limite int,
) (SobreSign1Estricto, error) {
	if limite < tamanoMinimoSobreSign1 ||
		limite > TamanoMaximoAbsolutoSobreSign1 ||
		len(contenido) < tamanoMinimoSobreSign1 || len(contenido) > limite {
		return SobreSign1Estricto{}, ErrSobreSign1Invalido
	}

	copia := append([]byte(nil), contenido...)
	var mensaje gocose.Sign1Message
	if err := mensaje.UnmarshalCBOR(copia); err != nil {
		return SobreSign1Estricto{}, ErrSobreSign1Invalido
	}
	mensajeDeterminista := gocose.Sign1Message{
		Headers: gocose.Headers{
			Protected:   mensaje.Headers.Protected,
			Unprotected: mensaje.Headers.Unprotected,
		},
		Payload:   append([]byte(nil), mensaje.Payload...),
		Signature: append([]byte(nil), mensaje.Signature...),
	}
	canonico, err := mensajeDeterminista.MarshalCBOR()
	if err != nil || !bytes.Equal(canonico, copia) ||
		len(mensaje.Headers.Protected) != 2 ||
		len(mensaje.Headers.Unprotected) != 0 {
		return SobreSign1Estricto{}, ErrSobreSign1Invalido
	}

	algoritmoBiblioteca, err := mensaje.Headers.Protected.Algorithm()
	if err != nil {
		return SobreSign1Estricto{}, ErrSobreSign1Invalido
	}
	algoritmo, valida := algoritmoDesdeBiblioteca(algoritmoBiblioteca)
	if !valida || !firmaCanonica(algoritmo, mensaje.Signature) {
		return SobreSign1Estricto{}, ErrSobreSign1Invalido
	}
	valorClaveID, presente := mensaje.Headers.Protected[gocose.HeaderLabelKeyID]
	claveID, tipoCorrecto := valorClaveID.([]byte)
	if !presente || !tipoCorrecto || !claveIDValida(claveID) {
		return SobreSign1Estricto{}, ErrSobreSign1Invalido
	}

	return SobreSign1Estricto{
		marca: marcaSobreSign1Estricto, contenido: copia, mensaje: mensaje,
		algoritmo: algoritmo, claveID: append([]byte(nil), claveID...),
	}, nil
}

func (s SobreSign1Estricto) validar() error {
	if s.marca != marcaSobreSign1Estricto || !s.algoritmo.valida() ||
		!claveIDValida(s.claveID) || len(s.contenido) < tamanoMinimoSobreSign1 ||
		len(s.contenido) > TamanoMaximoAbsolutoSobreSign1 ||
		len(s.mensaje.Headers.Protected) != 2 ||
		len(s.mensaje.Headers.Unprotected) != 0 ||
		!firmaCanonica(s.algoritmo, s.mensaje.Signature) {
		return ErrSobreSign1Invalido
	}
	return nil
}

func (s SobreSign1Estricto) Algoritmo() (Algoritmo, error) {
	if s.validar() != nil {
		return "", ErrSobreSign1Invalido
	}
	return s.algoritmo, nil
}

func (s SobreSign1Estricto) ClaveID() ([]byte, error) {
	if s.validar() != nil {
		return nil, ErrSobreSign1Invalido
	}
	return append([]byte(nil), s.claveID...), nil
}

// VerificadorClave liga una clave publica clonada a algoritmo y kid. Su
// constructor no convierte esa clave en confiable; esa decision corresponde al
// catalogo privado del protocolo consumidor.
type VerificadorClave struct {
	marca       string
	algoritmo   Algoritmo
	claveID     []byte
	verificador gocose.Verifier
}

func NuevoVerificadorClave(
	claveID []byte,
	algoritmo Algoritmo,
	clavePublica crypto.PublicKey,
) (*VerificadorClave, error) {
	if !claveIDValida(claveID) || !algoritmo.valida() {
		return nil, ErrConfiguracionClaveInvalida
	}
	claveClonada, err := clonarClavePublica(algoritmo, clavePublica)
	if err != nil {
		return nil, ErrConfiguracionClaveInvalida
	}
	algoritmoBiblioteca, valida := algoritmoParaBiblioteca(algoritmo)
	if !valida {
		return nil, ErrConfiguracionClaveInvalida
	}
	verificador, err := gocose.NewVerifier(algoritmoBiblioteca, claveClonada)
	if err != nil {
		return nil, ErrConfiguracionClaveInvalida
	}
	return &VerificadorClave{
		marca: marcaVerificadorClaveSign1, algoritmo: algoritmo,
		claveID: append([]byte(nil), claveID...), verificador: verificador,
	}, nil
}

func (v *VerificadorClave) validar() error {
	if v == nil || v.marca != marcaVerificadorClaveSign1 ||
		!v.algoritmo.valida() || !claveIDValida(v.claveID) ||
		v.verificador == nil {
		return ErrConfiguracionClaveInvalida
	}
	return nil
}

// Verificar comprueba la firma, el payload exacto y el AAD externo exacto. No
// consulta tiempo, revocacion o audiencia y no devuelve una capacidad.
func (v *VerificadorClave) Verificar(
	sobre SobreSign1Estricto,
	payloadEsperado []byte,
	aadExterno []byte,
) error {
	if v.validar() != nil || sobre.validar() != nil ||
		len(payloadEsperado) == 0 || len(payloadEsperado) > tamanoMaximoPayload ||
		len(aadExterno) == 0 || len(aadExterno) > tamanoMaximoAADExterno ||
		v.algoritmo != sobre.algoritmo ||
		!bytes.Equal(v.claveID, sobre.claveID) ||
		!bytes.Equal(payloadEsperado, sobre.mensaje.Payload) ||
		sobre.mensaje.Verify(aadExterno, v.verificador) != nil {
		return ErrVerificacionFirmaSign1Fallida
	}
	return nil
}

func firmaCanonica(algoritmo Algoritmo, firma []byte) bool {
	switch algoritmo {
	case AlgoritmoEdDSA:
		return len(firma) == ed25519.SignatureSize
	case AlgoritmoES256:
		const bytesComponenteP256 = 32
		if len(firma) != 2*bytesComponenteP256 {
			return false
		}
		orden := elliptic.P256().Params().N
		mitadOrden := new(big.Int).Rsh(new(big.Int).Set(orden), 1)
		r := new(big.Int).SetBytes(firma[:bytesComponenteP256])
		s := new(big.Int).SetBytes(firma[bytesComponenteP256:])
		return r.Sign() > 0 && r.Cmp(orden) < 0 &&
			s.Sign() > 0 && s.Cmp(mitadOrden) <= 0
	default:
		return false
	}
}

func algoritmoDesdeBiblioteca(algoritmo gocose.Algorithm) (Algoritmo, bool) {
	switch algoritmo {
	case gocose.AlgorithmEdDSA:
		return AlgoritmoEdDSA, true
	case gocose.AlgorithmES256:
		return AlgoritmoES256, true
	default:
		return "", false
	}
}

func algoritmoParaBiblioteca(algoritmo Algoritmo) (gocose.Algorithm, bool) {
	switch algoritmo {
	case AlgoritmoEdDSA:
		return gocose.AlgorithmEdDSA, true
	case AlgoritmoES256:
		return gocose.AlgorithmES256, true
	default:
		return gocose.AlgorithmReserved, false
	}
}

func clonarClavePublica(
	algoritmo Algoritmo,
	clave crypto.PublicKey,
) (crypto.PublicKey, error) {
	switch algoritmo {
	case AlgoritmoEdDSA:
		publica, correcta := clave.(ed25519.PublicKey)
		if !correcta || len(publica) != ed25519.PublicKeySize {
			return nil, ErrConfiguracionClaveInvalida
		}
		return append(ed25519.PublicKey(nil), publica...), nil
	case AlgoritmoES256:
		publica, correcta := clave.(*ecdsa.PublicKey)
		if !correcta || publica == nil || publica.Curve != elliptic.P256() ||
			publica.X == nil || publica.Y == nil ||
			!publica.Curve.IsOnCurve(publica.X, publica.Y) {
			return nil, ErrConfiguracionClaveInvalida
		}
		return &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).Set(publica.X),
			Y:     new(big.Int).Set(publica.Y),
		}, nil
	default:
		return nil, ErrConfiguracionClaveInvalida
	}
}

func claveIDValida(claveID []byte) bool {
	if len(claveID) == 0 || len(claveID) > tamanoMaximoClaveID {
		return false
	}
	for _, valor := range claveID {
		if valor != 0 {
			return true
		}
	}
	return false
}

func (SobreSign1Estricto) String() string {
	return "[SOBRE-COSE-SIGN1-ESTRICTO-NOMINAL-REDACTADO]"
}

func (s SobreSign1Estricto) GoString() string { return s.String() }
func (s SobreSign1Estricto) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SobreSign1Estricto) LogValue() slog.Value { return slog.StringValue(s.String()) }

func (*VerificadorClave) String() string     { return "[VERIFICADOR-CLAVE-COSE-SIGN1-REDACTADO]" }
func (v *VerificadorClave) GoString() string { return v.String() }
func (v *VerificadorClave) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, v.String())
}
func (v *VerificadorClave) LogValue() slog.Value { return slog.StringValue(v.String()) }

func (SobreSign1Estricto) MarshalJSON() ([]byte, error) { return nil, ErrSerializacionCOSEProhibida }
func (*SobreSign1Estricto) UnmarshalJSON([]byte) error  { return ErrSerializacionCOSEProhibida }
func (SobreSign1Estricto) MarshalText() ([]byte, error) { return nil, ErrSerializacionCOSEProhibida }
func (*SobreSign1Estricto) UnmarshalText([]byte) error  { return ErrSerializacionCOSEProhibida }
func (SobreSign1Estricto) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionCOSEProhibida
}
func (*SobreSign1Estricto) UnmarshalBinary([]byte) error { return ErrSerializacionCOSEProhibida }
func (SobreSign1Estricto) GobEncode() ([]byte, error)    { return nil, ErrSerializacionCOSEProhibida }
func (*SobreSign1Estricto) GobDecode([]byte) error       { return ErrSerializacionCOSEProhibida }
func (SobreSign1Estricto) MarshalCBOR() ([]byte, error)  { return nil, ErrSerializacionCOSEProhibida }
func (*SobreSign1Estricto) UnmarshalCBOR([]byte) error   { return ErrSerializacionCOSEProhibida }
func (SobreSign1Estricto) MarshalYAML() (any, error)     { return nil, ErrSerializacionCOSEProhibida }
func (*SobreSign1Estricto) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCOSEProhibida
}
func (SobreSign1Estricto) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionCOSEProhibida
}
func (*SobreSign1Estricto) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionCOSEProhibida
}

func (*VerificadorClave) MarshalJSON() ([]byte, error) { return nil, ErrSerializacionCOSEProhibida }
func (*VerificadorClave) UnmarshalJSON([]byte) error   { return ErrSerializacionCOSEProhibida }
func (*VerificadorClave) MarshalText() ([]byte, error) { return nil, ErrSerializacionCOSEProhibida }
func (*VerificadorClave) UnmarshalText([]byte) error   { return ErrSerializacionCOSEProhibida }
func (*VerificadorClave) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionCOSEProhibida
}
func (*VerificadorClave) UnmarshalBinary([]byte) error { return ErrSerializacionCOSEProhibida }
func (*VerificadorClave) GobEncode() ([]byte, error)   { return nil, ErrSerializacionCOSEProhibida }
func (*VerificadorClave) GobDecode([]byte) error       { return ErrSerializacionCOSEProhibida }
func (*VerificadorClave) MarshalCBOR() ([]byte, error) { return nil, ErrSerializacionCOSEProhibida }
func (*VerificadorClave) UnmarshalCBOR([]byte) error   { return ErrSerializacionCOSEProhibida }
func (*VerificadorClave) MarshalYAML() (any, error)    { return nil, ErrSerializacionCOSEProhibida }
func (*VerificadorClave) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionCOSEProhibida
}
func (*VerificadorClave) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionCOSEProhibida
}
func (*VerificadorClave) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionCOSEProhibida
}
