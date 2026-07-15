package confianzadocumental

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"sort"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConfiguracionConfianzaDocumentalInvalida = errors.New("vec: configuracion de confianza documental invalida")
	ErrSolicitudVerificacionCOSESign1Invalida   = errors.New("vec: solicitud de verificacion COSE Sign1 invalida")
	ErrVerificacionCOSESign1Fallida             = errors.New("vec: verificacion COSE Sign1 documental fallida")
	ErrPruebaCOSESign1VerificadaInvalida        = errors.New("vec: prueba COSE Sign1 documental verificada invalida")
	ErrSerializacionAutoridadCOSESign1Prohibida = errors.New("vec: serializacion de autoridad COSE Sign1 documental prohibida")
)

const (
	maximoBytesPayloadDocumentalV4           = 64 * 1024
	maximoBytesClaveIDDocumentalV4           = 128
	maximoBytesRevisionConfianzaDocumentalV4 = 128
	margenMaximoSobreCOSEDocumentalV4        = 4 * 1024
	maximaVigenciaConfiguracionConfianzaV4   = 24 * time.Hour
	marcaPruebaCOSESign1VerificadaV4         = "vec.prueba-cose-sign1-documental-verificada.v4"
	prefijoAADDocumentalV4                   = "vec.confianza-documental.cose-sign1.audiencia.v4\x00"
)

// AlgoritmoCOSEDocumental es la lista positiva inicial del nucleo. Una clave,
// una cabecera o un adaptador no pueden ampliarla durante la ejecucion.
type AlgoritmoCOSEDocumental string

const (
	AlgoritmoCOSEDocumentalEdDSA AlgoritmoCOSEDocumental = "EdDSA"
	AlgoritmoCOSEDocumentalES256 AlgoritmoCOSEDocumental = "ES256"
)

func (a AlgoritmoCOSEDocumental) valido() bool {
	return a == AlgoritmoCOSEDocumentalEdDSA || a == AlgoritmoCOSEDocumentalES256
}

type EstadoConfianzaClaveDocumental string

const (
	EstadoConfianzaClaveDocumentalActiva   EstadoConfianzaClaveDocumental = "activa"
	EstadoConfianzaClaveDocumentalRevocada EstadoConfianzaClaveDocumental = "revocada"
)

func (e EstadoConfianzaClaveDocumental) valido() bool {
	return e == EstadoConfianzaClaveDocumentalActiva ||
		e == EstadoConfianzaClaveDocumentalRevocada
}

// RaizPublicaFijada liga kid, algoritmo, clave, una unica audiencia, ventana y
// estado de revocacion. Es configuracion local, nunca una afirmacion de un
// adaptador.
type RaizPublicaFijada struct {
	claveID                          []byte
	algoritmo                        AlgoritmoCOSEDocumental
	clavePublica                     crypto.PublicKey
	huellaClaveSHA256                string
	audiencia                        AudienciaCOSEDocumental
	suiteAtestacionPDP               string
	audienciaDespliegueAtestacionPDP string
	estado                           EstadoConfianzaClaveDocumental
	revocadaEn                       time.Time
	validaDesde, validaHasta         time.Time
}

func nuevaRaizPublicaFijada(
	claveID []byte,
	algoritmo AlgoritmoCOSEDocumental,
	clavePublica crypto.PublicKey,
	audiencia AudienciaCOSEDocumental,
	estado EstadoConfianzaClaveDocumental,
	validaDesde, validaHasta, revocadaEn time.Time,
) (RaizPublicaFijada, error) {
	return nuevaRaizPublicaFijadaConPerfilPDP(
		claveID, algoritmo, clavePublica, audiencia, estado,
		validaDesde, validaHasta, revocadaEn, "", "",
	)
}

// nuevaRaizPublicaFijadaAtestacionPDP exige el perfil completo que forma
// parte de VEC-AD-1. La audiencia COSE separa el protocolo; esta audiencia de
// despliegue separa ademas entorno, base y esquema dentro del payload firmado.
func nuevaRaizPublicaFijadaAtestacionPDP(
	claveID []byte,
	algoritmo AlgoritmoCOSEDocumental,
	clavePublica crypto.PublicKey,
	suite string,
	audienciaDespliegue string,
	estado EstadoConfianzaClaveDocumental,
	validaDesde, validaHasta, revocadaEn time.Time,
) (RaizPublicaFijada, error) {
	return nuevaRaizPublicaFijadaConPerfilPDP(
		claveID, algoritmo, clavePublica, AudienciaCOSEAtestacionAutorizacionPDP,
		estado, validaDesde, validaHasta, revocadaEn, suite, audienciaDespliegue,
	)
}

func nuevaRaizPublicaFijadaConPerfilPDP(
	claveID []byte,
	algoritmo AlgoritmoCOSEDocumental,
	clavePublica crypto.PublicKey,
	audiencia AudienciaCOSEDocumental,
	estado EstadoConfianzaClaveDocumental,
	validaDesde, validaHasta, revocadaEn time.Time,
	suiteAtestacionPDP, audienciaDespliegueAtestacionPDP string,
) (RaizPublicaFijada, error) {
	claveClonada, huellaClave, err := clonarClavePublicaDocumental(algoritmo, clavePublica)
	if err != nil {
		return RaizPublicaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	raiz := RaizPublicaFijada{
		claveID: append([]byte(nil), claveID...), algoritmo: algoritmo,
		clavePublica: claveClonada, huellaClaveSHA256: huellaClave,
		audiencia: audiencia, estado: estado, revocadaEn: revocadaEn,
		validaDesde: validaDesde, validaHasta: validaHasta,
		suiteAtestacionPDP:               suiteAtestacionPDP,
		audienciaDespliegueAtestacionPDP: audienciaDespliegueAtestacionPDP,
	}
	if raiz.validar() != nil {
		return RaizPublicaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	return raiz, nil
}

func (r RaizPublicaFijada) validar() error {
	_, huellaClave, err := clonarClavePublicaDocumental(r.algoritmo, r.clavePublica)
	if !claveIDDocumentalValida(r.claveID) || !r.algoritmo.valido() || err != nil ||
		!huellaSHA256DocumentalValida(r.huellaClaveSHA256) || huellaClave != r.huellaClaveSHA256 ||
		!r.audiencia.valida() || !r.estado.valido() || !instanteCanonicoDocumental(r.validaDesde) ||
		!instanteCanonicoDocumental(r.validaHasta) || !r.validaHasta.After(r.validaDesde) {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	if r.audiencia == AudienciaCOSEAtestacionAutorizacionPDP {
		suiteEsperada, aprobada := suiteAtestacionAutorizacionPDP(r.algoritmo)
		cabecera := domain.CabeceraAtestacionAutorizacionV1{
			FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
			Suite:          r.suiteAtestacionPDP,
			ClaveID:        string(r.claveID),
			Audiencia:      r.audienciaDespliegueAtestacionPDP,
		}
		if !aprobada || r.suiteAtestacionPDP != suiteEsperada ||
			!audienciaDespliegueAtestacionPDPValida(r.audienciaDespliegueAtestacionPDP) ||
			cabecera.Validar() != nil {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
	} else if r.suiteAtestacionPDP != "" || r.audienciaDespliegueAtestacionPDP != "" {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	switch r.estado {
	case EstadoConfianzaClaveDocumentalActiva:
		if !r.revocadaEn.IsZero() {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
	case EstadoConfianzaClaveDocumentalRevocada:
		if !instanteCanonicoDocumental(r.revocadaEn) {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
	default:
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	return nil
}

func (RaizPublicaFijada) String() string {
	return "[RAIZ-PUBLICA-FIJADA-DOCUMENTAL-REDACTADA]"
}

func (r RaizPublicaFijada) GoString() string { return r.String() }
func (r RaizPublicaFijada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r RaizPublicaFijada) LogValue() slog.Value { return slog.StringValue(r.String()) }
func (RaizPublicaFijada) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*RaizPublicaFijada) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (RaizPublicaFijada) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*RaizPublicaFijada) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (RaizPublicaFijada) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*RaizPublicaFijada) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}

// ConfiguracionConfianzaFijada es una instantanea de revocacion con revision y
// caducidad obligatorias. Al caducar, el servicio falla cerrado hasta recibir
// una configuracion nueva durante un arranque controlado.
type ConfiguracionConfianzaFijada struct {
	revision     string
	publicadaEn  time.Time
	expiraEn     time.Time
	raices       []RaizPublicaFijada
	huellaSHA256 string
}

func nuevaConfiguracionConfianzaFijada(
	revision string,
	publicadaEn, expiraEn time.Time,
	raices ...RaizPublicaFijada,
) (ConfiguracionConfianzaFijada, error) {
	configuracion := ConfiguracionConfianzaFijada{
		revision: revision, publicadaEn: publicadaEn, expiraEn: expiraEn,
		raices: make([]RaizPublicaFijada, 0, len(raices)),
	}
	for _, raiz := range raices {
		clon, err := clonarRaizPublicaFijada(raiz)
		if err != nil {
			return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
		}
		configuracion.raices = append(configuracion.raices, clon)
	}
	configuracion.huellaSHA256 = configuracion.calcularHuella()
	if configuracion.validar() != nil {
		return ConfiguracionConfianzaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	return configuracion, nil
}

func (c ConfiguracionConfianzaFijada) validar() error {
	if !referenciaConfiguracionDocumentalValida(c.revision) ||
		!instanteCanonicoDocumental(c.publicadaEn) || !instanteCanonicoDocumental(c.expiraEn) ||
		!c.expiraEn.After(c.publicadaEn) ||
		c.expiraEn.Sub(c.publicadaEn) > maximaVigenciaConfiguracionConfianzaV4 ||
		len(c.raices) == 0 || !huellaSHA256DocumentalValida(c.huellaSHA256) {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	vistas := make(map[string]struct{}, len(c.raices))
	huellasClave := make(map[string]struct{}, len(c.raices))
	for _, raiz := range c.raices {
		if raiz.validar() != nil {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
		indice := string(raiz.claveID)
		if _, duplicada := vistas[indice]; duplicada {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
		vistas[indice] = struct{}{}
		if _, duplicada := huellasClave[raiz.huellaClaveSHA256]; duplicada {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
		huellasClave[raiz.huellaClaveSHA256] = struct{}{}
		if raiz.estado == EstadoConfianzaClaveDocumentalRevocada &&
			raiz.revocadaEn.After(c.publicadaEn) {
			return ErrConfiguracionConfianzaDocumentalInvalida
		}
	}
	if c.huellaSHA256 != c.calcularHuella() {
		return ErrConfiguracionConfianzaDocumentalInvalida
	}
	return nil
}

func (c ConfiguracionConfianzaFijada) calcularHuella() string {
	registros := make([]string, 0, len(c.raices))
	for _, raiz := range c.raices {
		revocadaEn := ""
		if !raiz.revocadaEn.IsZero() {
			revocadaEn = raiz.revocadaEn.Format(time.RFC3339Nano)
		}
		registros = append(registros, strings.Join([]string{
			hex.EncodeToString(raiz.claveID), string(raiz.algoritmo), raiz.huellaClaveSHA256,
			string(raiz.audiencia), raiz.suiteAtestacionPDP,
			raiz.audienciaDespliegueAtestacionPDP, string(raiz.estado), raiz.validaDesde.Format(time.RFC3339Nano),
			raiz.validaHasta.Format(time.RFC3339Nano), revocadaEn,
		}, "\x00"))
	}
	sort.Strings(registros)
	hash := sha256.New()
	escribirCampoHuellaDocumental(hash, "vec.configuracion-confianza-documental.v4")
	escribirCampoHuellaDocumental(hash, c.revision)
	escribirCampoHuellaDocumental(hash, c.publicadaEn.Format(time.RFC3339Nano))
	escribirCampoHuellaDocumental(hash, c.expiraEn.Format(time.RFC3339Nano))
	for _, registro := range registros {
		escribirCampoHuellaDocumental(hash, registro)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func (ConfiguracionConfianzaFijada) String() string {
	return "[CONFIGURACION-CONFIANZA-DOCUMENTAL-FIJADA-REDACTADA]"
}

func (c ConfiguracionConfianzaFijada) GoString() string { return c.String() }
func (c ConfiguracionConfianzaFijada) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ConfiguracionConfianzaFijada) LogValue() slog.Value { return slog.StringValue(c.String()) }
func (ConfiguracionConfianzaFijada) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*ConfiguracionConfianzaFijada) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (ConfiguracionConfianzaFijada) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*ConfiguracionConfianzaFijada) UnmarshalText([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}
func (ConfiguracionConfianzaFijada) MarshalBinary() ([]byte, error) {
	return nil, ErrSerializacionAutoridadCOSESign1Prohibida
}
func (*ConfiguracionConfianzaFijada) UnmarshalBinary([]byte) error {
	return ErrSerializacionAutoridadCOSESign1Prohibida
}

func clonarRaizPublicaFijada(raiz RaizPublicaFijada) (RaizPublicaFijada, error) {
	clave, huella, err := clonarClavePublicaDocumental(raiz.algoritmo, raiz.clavePublica)
	if raiz.validar() != nil || err != nil || huella != raiz.huellaClaveSHA256 {
		return RaizPublicaFijada{}, ErrConfiguracionConfianzaDocumentalInvalida
	}
	return RaizPublicaFijada{
		claveID: append([]byte(nil), raiz.claveID...), algoritmo: raiz.algoritmo,
		clavePublica: clave, huellaClaveSHA256: huella,
		audiencia: raiz.audiencia, estado: raiz.estado, revocadaEn: raiz.revocadaEn,
		validaDesde: raiz.validaDesde, validaHasta: raiz.validaHasta,
		suiteAtestacionPDP:               raiz.suiteAtestacionPDP,
		audienciaDespliegueAtestacionPDP: raiz.audienciaDespliegueAtestacionPDP,
	}, nil
}

func clonarClavePublicaDocumental(
	algoritmo AlgoritmoCOSEDocumental,
	clave crypto.PublicKey,
) (crypto.PublicKey, string, error) {
	var clon crypto.PublicKey
	switch algoritmo {
	case AlgoritmoCOSEDocumentalEdDSA:
		publica, ok := clave.(ed25519.PublicKey)
		if !ok || len(publica) != ed25519.PublicKeySize {
			return nil, "", ErrConfiguracionConfianzaDocumentalInvalida
		}
		clon = append(ed25519.PublicKey(nil), publica...)
	case AlgoritmoCOSEDocumentalES256:
		publica, ok := clave.(*ecdsa.PublicKey)
		if !ok || publica == nil || publica.Curve != elliptic.P256() || publica.X == nil ||
			publica.Y == nil || !publica.Curve.IsOnCurve(publica.X, publica.Y) {
			return nil, "", ErrConfiguracionConfianzaDocumentalInvalida
		}
		clon = &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).Set(publica.X),
			Y:     new(big.Int).Set(publica.Y),
		}
	default:
		return nil, "", ErrConfiguracionConfianzaDocumentalInvalida
	}
	der, err := x509.MarshalPKIXPublicKey(clon)
	if err != nil {
		return nil, "", ErrConfiguracionConfianzaDocumentalInvalida
	}
	return clon, huellaBytesDocumentales(der), nil
}

func claveIDDocumentalValida(claveID []byte) bool {
	if len(claveID) == 0 || len(claveID) > maximoBytesClaveIDDocumentalV4 {
		return false
	}
	for _, valor := range claveID {
		if valor != 0 {
			return true
		}
	}
	return false
}

func referenciaConfiguracionDocumentalValida(referencia string) bool {
	if referencia == "" || len(referencia) > maximoBytesRevisionConfianzaDocumentalV4 ||
		strings.TrimSpace(referencia) != referencia {
		return false
	}
	for _, valor := range []byte(referencia) {
		if valor < 0x21 || valor > 0x7e {
			return false
		}
	}
	return true
}

func instanteCanonicoDocumental(instante time.Time) bool {
	return !instante.IsZero() && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Location() == time.UTC &&
		instante.Equal(instante.Truncate(time.Microsecond))
}

func huellaBytesDocumentales(contenido []byte) string {
	huella := sha256.Sum256(contenido)
	return hex.EncodeToString(huella[:])
}

func huellaSHA256DocumentalValida(huella string) bool {
	if len(huella) != sha256.Size*2 || strings.ToLower(huella) != huella {
		return false
	}
	_, err := hex.DecodeString(huella)
	return err == nil
}

func escribirCampoHuellaDocumental(destino io.Writer, campo string) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(campo)))
	_, _ = destino.Write(longitud[:])
	_, _ = io.WriteString(destino, campo)
}
