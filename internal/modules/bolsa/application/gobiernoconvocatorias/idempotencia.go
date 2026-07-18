package gobiernoconvocatorias

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
)

const longitudClaveClienteBase64URL = 43

var (
	ErrClaveClienteIdempotenciaInvalida = errors.New("gobierno convocatorias: clave cliente de idempotencia invalida")
	ErrHMACIdempotenciaInvalido         = errors.New("gobierno convocatorias: HMAC nominal de idempotencia invalido")
	ErrSerializacionDiarioProhibida     = errors.New("gobierno convocatorias: serializacion de valor interno del diario prohibida")
)

// bloqueoSerializacionDiario evita que valores internos sensibles se
// conviertan accidentalmente en DTO, registro o mensaje. Los tipos que lo
// embeben heredan el cierre de codificacion y decodificacion.
type bloqueoSerializacionDiario struct{}

func (bloqueoSerializacionDiario) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) UnmarshalJSON([]byte) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) UnmarshalText([]byte) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) UnmarshalBinary([]byte) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) MarshalCBOR() ([]byte, error) {
	return nil, ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) UnmarshalCBOR([]byte) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) MarshalYAML() (any, error) {
	return nil, ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) UnmarshalYAML(func(any) error) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) GobEncode() ([]byte, error) {
	return nil, ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) GobDecode([]byte) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) MarshalXML(*xml.Encoder, xml.StartElement) error {
	return ErrSerializacionDiarioProhibida
}

func (*bloqueoSerializacionDiario) UnmarshalXML(*xml.Decoder, xml.StartElement) error {
	return ErrSerializacionDiarioProhibida
}

func (bloqueoSerializacionDiario) String() string {
	return "[VALOR-DIARIO-GOBIERNO-CONVOCATORIAS-PROTEGIDO]"
}

func (b bloqueoSerializacionDiario) GoString() string { return b.String() }

func (b bloqueoSerializacionDiario) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, b.String())
}

func (b bloqueoSerializacionDiario) LogValue() slog.Value {
	return slog.StringValue(b.String())
}

// ClaveClienteIdempotenciaConvocatoria conserva exactamente 32 bytes. El
// constructor solo acredita la forma canonica base64url sin relleno de 43
// caracteres. Nunca pretende inferir que la muestra tenga entropia CSPRNG.
type ClaveClienteIdempotenciaConvocatoria struct {
	bloqueoSerializacionDiario
	valor        [32]byte
	inicializada bool
}

func NuevaClaveClienteIdempotenciaConvocatoria(
	formaBase64URL string,
) (ClaveClienteIdempotenciaConvocatoria, error) {
	if len(formaBase64URL) != longitudClaveClienteBase64URL {
		return ClaveClienteIdempotenciaConvocatoria{}, ErrClaveClienteIdempotenciaInvalida
	}
	bytes, err := base64.RawURLEncoding.Strict().DecodeString(formaBase64URL)
	if err != nil || len(bytes) != 32 || base64.RawURLEncoding.EncodeToString(bytes) != formaBase64URL {
		return ClaveClienteIdempotenciaConvocatoria{}, ErrClaveClienteIdempotenciaInvalida
	}
	var valor [32]byte
	copy(valor[:], bytes)
	return ClaveClienteIdempotenciaConvocatoria{valor: valor, inicializada: true}, nil
}

func (c ClaveClienteIdempotenciaConvocatoria) Valida() bool {
	return c.inicializada
}

type hmacNominalIdempotencia struct {
	versionEsquema uint16
	clave          ReferenciaClaveHMAC
	valor          [32]byte
	definido       bool
}

type dominioClaveHMAC uint8

const (
	dominioClaveHMACLocalizador dominioClaveHMAC = iota + 1
	dominioClaveHMACHuellaSolicitud
)

// ReferenciaClaveHMAC liga una generacion exacta de clave a uno de los dos
// dominios. La referencia no contiene material criptografico.
type ReferenciaClaveHMAC struct {
	bloqueoSerializacionDiario
	dominio         dominioClaveHMAC
	referencia      [128]byte
	longitud        uint8
	generacionClave uint32
	definida        bool
}

func NuevaReferenciaClaveHMACLocalizador(
	referencia string,
	generacionClave uint32,
) (ReferenciaClaveHMAC, error) {
	return nuevaReferenciaClaveHMAC(dominioClaveHMACLocalizador, referencia, generacionClave)
}

func NuevaReferenciaClaveHMACHuellaSolicitud(
	referencia string,
	generacionClave uint32,
) (ReferenciaClaveHMAC, error) {
	return nuevaReferenciaClaveHMAC(dominioClaveHMACHuellaSolicitud, referencia, generacionClave)
}

func nuevaReferenciaClaveHMAC(
	dominio dominioClaveHMAC,
	referencia string,
	generacionClave uint32,
) (ReferenciaClaveHMAC, error) {
	resultado := ReferenciaClaveHMAC{dominio: dominio, generacionClave: generacionClave}
	if len(referencia) > len(resultado.referencia) {
		return ReferenciaClaveHMAC{}, ErrHMACIdempotenciaInvalido
	}
	copy(resultado.referencia[:], []byte(referencia))
	resultado.longitud = uint8(len(referencia))
	resultado.definida = true
	if !resultado.valida() {
		return ReferenciaClaveHMAC{}, ErrHMACIdempotenciaInvalido
	}
	return resultado, nil
}

func (r ReferenciaClaveHMAC) valida() bool {
	prefijo := ""
	switch r.dominio {
	case dominioClaveHMACLocalizador:
		prefijo = "clave:hmac:convocatorias:localizador:"
	case dominioClaveHMACHuellaSolicitud:
		prefijo = "clave:hmac:convocatorias:huella:"
	default:
		return false
	}
	longitud := int(r.longitud)
	validez := banderaConstante(r.definida)
	validez &= 1 ^ subtle.ConstantTimeEq(int32(r.generacionClave), 0)
	validez &= subtle.ConstantTimeLessOrEq(len(prefijo)+1, longitud)
	validez &= subtle.ConstantTimeLessOrEq(longitud, len(r.referencia))
	validez &= subtle.ConstantTimeCompare(r.referencia[:len(prefijo)], []byte(prefijo))
	for indice, caracter := range r.referencia {
		activo := subtle.ConstantTimeLessOrEq(indice+1, longitud)
		letraMinuscula := subtle.ConstantTimeLessOrEq(int('a'), int(caracter)) &
			subtle.ConstantTimeLessOrEq(int(caracter), int('z'))
		digito := subtle.ConstantTimeLessOrEq(int('0'), int(caracter)) &
			subtle.ConstantTimeLessOrEq(int(caracter), int('9'))
		separador := subtle.ConstantTimeByteEq(caracter, ':') |
			subtle.ConstantTimeByteEq(caracter, '_') |
			subtle.ConstantTimeByteEq(caracter, '-') |
			subtle.ConstantTimeByteEq(caracter, '.')
		caracterValido := letraMinuscula | digito | separador
		rellenoValido := subtle.ConstantTimeByteEq(caracter, 0)
		validez &= activo&caracterValido | (1-activo)&rellenoValido
		esUltimo := subtle.ConstantTimeEq(int32(indice+1), int32(longitud))
		validez &= (1 - esUltimo) | letraMinuscula | digito
	}
	if validez != 1 {
		return false
	}
	return true
}

func nuevoHMACNominalIdempotencia(
	versionEsquema uint16,
	clave ReferenciaClaveHMAC,
	dominio dominioClaveHMAC,
	valorHexadecimal string,
) (hmacNominalIdempotencia, error) {
	if versionEsquema == 0 || !clave.valida() || clave.dominio != dominio || len(valorHexadecimal) != 64 {
		return hmacNominalIdempotencia{}, ErrHMACIdempotenciaInvalido
	}
	bytes, err := hex.DecodeString(valorHexadecimal)
	if err != nil || len(bytes) != 32 || hex.EncodeToString(bytes) != valorHexadecimal {
		return hmacNominalIdempotencia{}, ErrHMACIdempotenciaInvalido
	}
	var valor [32]byte
	copy(valor[:], bytes)
	resultado := hmacNominalIdempotencia{
		versionEsquema: versionEsquema, clave: clave, valor: valor, definido: true,
	}
	if !resultado.valido() {
		return hmacNominalIdempotencia{}, ErrHMACIdempotenciaInvalido
	}
	return resultado, nil
}

func (h hmacNominalIdempotencia) valido() bool {
	return h.definido && h.versionEsquema > 0 && h.clave.valida()
}

func (h hmacNominalIdempotencia) coincide(otro hmacNominalIdempotencia) bool {
	material := h.materialComparacionConstante()
	materialOtro := otro.materialComparacionConstante()
	valido := banderaConstante(h.definido) & banderaConstante(otro.definido)
	valido &= banderaConstante(h.clave.valida()) & banderaConstante(otro.clave.valida())
	valido &= 1 ^ subtle.ConstantTimeEq(int32(h.versionEsquema), 0)
	valido &= 1 ^ subtle.ConstantTimeEq(int32(otro.versionEsquema), 0)
	valido &= 1 ^ subtle.ConstantTimeEq(int32(h.clave.generacionClave), 0)
	valido &= 1 ^ subtle.ConstantTimeEq(int32(otro.clave.generacionClave), 0)
	return valido&subtle.ConstantTimeCompare(material[:], materialOtro[:]) == 1
}

func (h hmacNominalIdempotencia) materialComparacionConstante() [168]byte {
	var material [168]byte
	binary.BigEndian.PutUint16(material[0:2], h.versionEsquema)
	binary.BigEndian.PutUint32(material[2:6], h.clave.generacionClave)
	material[6] = byte(h.clave.dominio)
	material[7] = h.clave.longitud
	copy(material[8:136], h.clave.referencia[:])
	copy(material[136:], h.valor[:])
	return material
}

func banderaConstante(valor bool) int {
	if valor {
		return 1
	}
	return 0
}

// LocalizadorOperacion es L: HMAC versionado para localizar una operacion. No
// contiene la clave cliente, el principal, la organizacion ni sus valores en
// claro. Su tipo nominal impide confundirlo con F.
type LocalizadorOperacion struct {
	bloqueoSerializacionDiario
	hmac hmacNominalIdempotencia
}

// NuevoLocalizadorOperacion solo valida y separa la representacion nominal.
// No acredita que el HMAC haya sido derivado por una clave confiable y no debe
// exponerse como constructor de un DTO. El futuro caso de uso lo invocara solo
// con el resultado de su derivador privado.
func NuevoLocalizadorOperacion(
	versionEsquema uint16,
	clave ReferenciaClaveHMAC,
	valorHMACSHA256 string,
) (LocalizadorOperacion, error) {
	hmac, err := nuevoHMACNominalIdempotencia(
		versionEsquema, clave, dominioClaveHMACLocalizador, valorHMACSHA256,
	)
	if err != nil {
		return LocalizadorOperacion{}, err
	}
	return LocalizadorOperacion{hmac: hmac}, nil
}

func (l LocalizadorOperacion) Valido() bool { return l.hmac.valido() }

func (l LocalizadorOperacion) CoincideExactamente(otro LocalizadorOperacion) bool {
	return l.hmac.coincide(otro.hmac)
}

// HuellaSolicitud es F: HMAC versionado de la orden semantica canonica. No
// conserva el cuerpo, el motivo ni datos personales en claro.
type HuellaSolicitud struct {
	bloqueoSerializacionDiario
	hmac hmacNominalIdempotencia
}

// NuevaHuellaSolicitud valida forma y dominio, no procedencia criptografica.
// La frontera HTTP nunca debe aceptar este valor ya construido por el cliente.
func NuevaHuellaSolicitud(
	versionEsquema uint16,
	clave ReferenciaClaveHMAC,
	valorHMACSHA256 string,
) (HuellaSolicitud, error) {
	hmac, err := nuevoHMACNominalIdempotencia(
		versionEsquema, clave, dominioClaveHMACHuellaSolicitud, valorHMACSHA256,
	)
	if err != nil {
		return HuellaSolicitud{}, err
	}
	return HuellaSolicitud{hmac: hmac}, nil
}

func (h HuellaSolicitud) Valida() bool { return h.hmac.valido() }

func (h HuellaSolicitud) CoincideExactamente(otra HuellaSolicitud) bool {
	return h.hmac.coincide(otra.hmac)
}
