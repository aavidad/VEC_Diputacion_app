package confianzaatestacion

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"hash"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConfiguracionConfianzaAtestacionV2Invalida = errors.New(
		"vec: configuracion de confianza de atestacion V2 invalida",
	)
	ErrVerificacionConfianzaAtestacionV2Fallida = errors.New(
		"vec: verificacion de confianza de atestacion V2 fallida",
	)
	ErrPruebaConfianzaAtestacionV2Invalida = errors.New(
		"vec: prueba de confianza de atestacion V2 invalida",
	)
	ErrSerializacionConfianzaAtestacionV2Prohibida = errors.New(
		"vec: serializacion generica de confianza de atestacion V2 prohibida",
	)
)

const (
	// SuiteAtestacionAutorizacionV2COSEEdDSA separa el perfil VEC-AD-2 del
	// perfil COSE de VEC-AD-1. Que la primitiva comun admita ES256 no lo
	// incorpora a esta lista positiva.
	SuiteAtestacionAutorizacionV2COSEEdDSA = "VEC-AD-2-COSE-EDDSA-1"

	AlgoritmoCOSEAtestacionAutorizacionV2EdDSA = "EdDSA"

	maximoRaicesConfianzaAtestacionV2        = 64
	maximoBytesRevisionConfianzaAtestacionV2 = 128
	maximaVigenciaConfiguracionAtestacionV2  = 24 * time.Hour
	dominioHuellaConfiguracionAtestacionV2   = "vec.configuracion-confianza-atestacion-autorizacion.v2"
	marcaRaizPublicaConfianzaAtestacionV2    = "vec.raiz-publica-confianza-atestacion-autorizacion.v2"
	marcaConfiguracionConfianzaAtestacionV2  = "vec.configuracion-confianza-atestacion-autorizacion.v2"
)

type EstadoClaveAtestacionAutorizacionV2 string

const (
	EstadoClaveAtestacionAutorizacionV2Activa   EstadoClaveAtestacionAutorizacionV2 = "activa"
	EstadoClaveAtestacionAutorizacionV2Revocada EstadoClaveAtestacionAutorizacionV2 = "revocada"
)

func (e EstadoClaveAtestacionAutorizacionV2) valido() bool {
	return e == EstadoClaveAtestacionAutorizacionV2Activa ||
		e == EstadoClaveAtestacionAutorizacionV2Revocada
}

// RaizPublicaAtestacionAutorizacionV2 fija una clave Ed25519 a una unica
// audiencia de despliegue y ventana. No se reconstruye desde una peticion.
type RaizPublicaAtestacionAutorizacionV2 struct {
	bloqueoSerializacion
	marca                 string
	claveID               string
	clavePublica          ed25519.PublicKey
	huellaClaveSPKISHA256 string
	audienciaDespliegue   string
	estado                EstadoClaveAtestacionAutorizacionV2
	validaDesde           time.Time
	validaHasta           time.Time
	revocadaEn            time.Time
}

func NuevaRaizPublicaAtestacionAutorizacionV2EdDSA(
	claveID string,
	clavePublica ed25519.PublicKey,
	audienciaDespliegue string,
	estado EstadoClaveAtestacionAutorizacionV2,
	validaDesde time.Time,
	validaHasta time.Time,
	revocadaEn time.Time,
) (RaizPublicaAtestacionAutorizacionV2, error) {
	claveClonada, huella, err := clonarClavePublicaAtestacionV2(clavePublica)
	if err != nil {
		return RaizPublicaAtestacionAutorizacionV2{}, ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	raiz := RaizPublicaAtestacionAutorizacionV2{
		marca:                 marcaRaizPublicaConfianzaAtestacionV2,
		claveID:               claveID,
		clavePublica:          claveClonada,
		huellaClaveSPKISHA256: huella,
		audienciaDespliegue:   audienciaDespliegue,
		estado:                estado,
		validaDesde:           validaDesde,
		validaHasta:           validaHasta,
		revocadaEn:            revocadaEn,
	}
	if raiz.validar() != nil {
		return RaizPublicaAtestacionAutorizacionV2{}, ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return raiz, nil
}

func (r RaizPublicaAtestacionAutorizacionV2) validar() error {
	_, huella, err := clonarClavePublicaAtestacionV2(r.clavePublica)
	cabecera := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          SuiteAtestacionAutorizacionV2COSEEdDSA,
		ClaveID:        r.claveID,
		Audiencia:      r.audienciaDespliegue,
	}
	if r.marca != marcaRaizPublicaConfianzaAtestacionV2 || err != nil ||
		huella != r.huellaClaveSPKISHA256 || !huellaSHA256ConfianzaValida(huella) ||
		cabecera.Validar() != nil ||
		!audienciaDespliegueAtestacionV2Valida(r.audienciaDespliegue) ||
		!r.estado.valido() || !instanteCanonicoConfianza(r.validaDesde) ||
		!instanteCanonicoConfianza(r.validaHasta) ||
		!r.validaHasta.After(r.validaDesde) {
		return ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	switch r.estado {
	case EstadoClaveAtestacionAutorizacionV2Activa:
		if !r.revocadaEn.IsZero() {
			return ErrConfiguracionConfianzaAtestacionV2Invalida
		}
	case EstadoClaveAtestacionAutorizacionV2Revocada:
		if !instanteCanonicoConfianza(r.revocadaEn) || r.revocadaEn.Before(r.validaDesde) {
			return ErrConfiguracionConfianzaAtestacionV2Invalida
		}
	default:
		return ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return nil
}

// ConfiguracionConfianzaAtestacionAutorizacionV2 es una instantanea acotada y
// con caducidad obligatoria. Su huella permite que el efecto durable compruebe
// despues que consume exactamente la revision verificada fuera de SQL.
type ConfiguracionConfianzaAtestacionAutorizacionV2 struct {
	bloqueoSerializacion
	marca        string
	revision     string
	publicadaEn  time.Time
	expiraEn     time.Time
	raices       []RaizPublicaAtestacionAutorizacionV2
	huellaSHA256 string
}

func NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
	revision string,
	publicadaEn time.Time,
	expiraEn time.Time,
	raices ...RaizPublicaAtestacionAutorizacionV2,
) (ConfiguracionConfianzaAtestacionAutorizacionV2, error) {
	configuracion := ConfiguracionConfianzaAtestacionAutorizacionV2{
		marca:       marcaConfiguracionConfianzaAtestacionV2,
		revision:    revision,
		publicadaEn: publicadaEn,
		expiraEn:    expiraEn,
		raices:      make([]RaizPublicaAtestacionAutorizacionV2, 0, len(raices)),
	}
	for _, raiz := range raices {
		clon, err := clonarRaizPublicaAtestacionV2(raiz)
		if err != nil {
			return ConfiguracionConfianzaAtestacionAutorizacionV2{}, ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		configuracion.raices = append(configuracion.raices, clon)
	}
	configuracion.huellaSHA256 = configuracion.calcularHuella()
	if configuracion.validar() != nil {
		return ConfiguracionConfianzaAtestacionAutorizacionV2{}, ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return configuracion, nil
}

// ValidarHuellaSHA256Esperada comprueba una huella durable sin exponer la
// representacion interna de la configuracion. Permite a los adaptadores de
// persistencia acreditar que reconstruyeron exactamente la revision leida.
func (c ConfiguracionConfianzaAtestacionAutorizacionV2) ValidarHuellaSHA256Esperada(
	esperada string,
) error {
	if c.validar() != nil || !huellaSHA256ConfianzaValida(esperada) ||
		subtle.ConstantTimeCompare([]byte(c.huellaSHA256), []byte(esperada)) != 1 {
		return ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return nil
}

func (c ConfiguracionConfianzaAtestacionAutorizacionV2) validar() error {
	if c.marca != marcaConfiguracionConfianzaAtestacionV2 ||
		!referenciaConfiguracionConfianzaValida(c.revision) ||
		!instanteCanonicoConfianza(c.publicadaEn) ||
		!instanteCanonicoConfianza(c.expiraEn) ||
		!c.expiraEn.After(c.publicadaEn) ||
		c.expiraEn.Sub(c.publicadaEn) > maximaVigenciaConfiguracionAtestacionV2 ||
		len(c.raices) == 0 || len(c.raices) > maximoRaicesConfianzaAtestacionV2 ||
		!huellaSHA256ConfianzaValida(c.huellaSHA256) {
		return ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	claves := make(map[string]struct{}, len(c.raices))
	huellas := make(map[string]struct{}, len(c.raices))
	for _, raiz := range c.raices {
		if raiz.validar() != nil {
			return ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		if _, existe := claves[raiz.claveID]; existe {
			return ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		claves[raiz.claveID] = struct{}{}
		if _, existe := huellas[raiz.huellaClaveSPKISHA256]; existe {
			return ErrConfiguracionConfianzaAtestacionV2Invalida
		}
		huellas[raiz.huellaClaveSPKISHA256] = struct{}{}
		if raiz.estado == EstadoClaveAtestacionAutorizacionV2Revocada &&
			raiz.revocadaEn.After(c.publicadaEn) {
			return ErrConfiguracionConfianzaAtestacionV2Invalida
		}
	}
	if c.huellaSHA256 != c.calcularHuella() {
		return ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return nil
}

func (c ConfiguracionConfianzaAtestacionAutorizacionV2) calcularHuella() string {
	type registroRaiz struct {
		claveID string
		campos  []string
	}
	registros := make([]registroRaiz, 0, len(c.raices))
	for _, raiz := range c.raices {
		revocadaEn := ""
		if !raiz.revocadaEn.IsZero() {
			revocadaEn = raiz.revocadaEn.Format(time.RFC3339Nano)
		}
		registros = append(registros, registroRaiz{
			claveID: raiz.claveID,
			campos: []string{
				raiz.claveID,
				AlgoritmoCOSEAtestacionAutorizacionV2EdDSA,
				raiz.huellaClaveSPKISHA256,
				SuiteAtestacionAutorizacionV2COSEEdDSA,
				raiz.audienciaDespliegue,
				string(raiz.estado),
				raiz.validaDesde.Format(time.RFC3339Nano),
				raiz.validaHasta.Format(time.RFC3339Nano),
				revocadaEn,
			},
		})
	}
	sort.Slice(registros, func(i, j int) bool {
		return bytes.Compare([]byte(registros[i].claveID), []byte(registros[j].claveID)) < 0
	})
	calculador := sha256.New()
	escribirCampoHuellaConfianza(calculador, dominioHuellaConfiguracionAtestacionV2)
	escribirCampoHuellaConfianza(calculador, c.revision)
	escribirCampoHuellaConfianza(calculador, c.publicadaEn.Format(time.RFC3339Nano))
	escribirCampoHuellaConfianza(calculador, c.expiraEn.Format(time.RFC3339Nano))
	for _, registro := range registros {
		for _, campo := range registro.campos {
			escribirCampoHuellaConfianza(calculador, campo)
		}
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func clonarRaizPublicaAtestacionV2(
	raiz RaizPublicaAtestacionAutorizacionV2,
) (RaizPublicaAtestacionAutorizacionV2, error) {
	clave, huella, err := clonarClavePublicaAtestacionV2(raiz.clavePublica)
	if raiz.validar() != nil || err != nil || huella != raiz.huellaClaveSPKISHA256 {
		return RaizPublicaAtestacionAutorizacionV2{}, ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	return RaizPublicaAtestacionAutorizacionV2{
		marca: raiz.marca, claveID: raiz.claveID, clavePublica: clave,
		huellaClaveSPKISHA256: raiz.huellaClaveSPKISHA256,
		audienciaDespliegue:   raiz.audienciaDespliegue,
		estado:                raiz.estado, validaDesde: raiz.validaDesde,
		validaHasta: raiz.validaHasta, revocadaEn: raiz.revocadaEn,
	}, nil
}

func clonarClavePublicaAtestacionV2(
	clave ed25519.PublicKey,
) (ed25519.PublicKey, string, error) {
	if len(clave) != ed25519.PublicKeySize || bytes.Equal(clave, make([]byte, ed25519.PublicKeySize)) {
		return nil, "", ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	clon := append(ed25519.PublicKey(nil), clave...)
	spki, err := x509.MarshalPKIXPublicKey(clon)
	if err != nil {
		return nil, "", ErrConfiguracionConfianzaAtestacionV2Invalida
	}
	suma := sha256.Sum256(spki)
	return clon, hex.EncodeToString(suma[:]), nil
}

func audienciaDespliegueAtestacionV2Valida(audiencia string) bool {
	partes := strings.Split(audiencia, "/")
	if len(partes) != 4 || partes[0] != "vec-diputacion" {
		return false
	}
	for _, parte := range partes[1:] {
		if parte == "" || parte == "." || parte == ".." {
			return false
		}
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV2{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV2,
		Suite:          SuiteAtestacionAutorizacionV2COSEEdDSA,
		ClaveID:        "clave-validacion-audiencia-v2",
		Audiencia:      audiencia,
	}
	return cabecera.Validar() == nil
}

func instanteCanonicoConfianza(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func referenciaConfiguracionConfianzaValida(valor string) bool {
	return textoReferenciaConfianzaValido(valor, maximoBytesRevisionConfianzaAtestacionV2)
}

func textoReferenciaConfianzaValido(valor string, limite int) bool {
	if limite <= 0 || valor == "" || len(valor) > limite ||
		valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.IsSpace(caracter) {
			return false
		}
	}
	return true
}

func huellaSHA256ConfianzaValida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

func escribirCampoHuellaConfianza(destino hash.Hash, valor string) {
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len([]byte(valor))))
	_, _ = destino.Write(longitud[:])
	_, _ = destino.Write([]byte(valor))
}
