package confianzaatestacion

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrConfiguracionConfianzaAtestacionV3Invalida = errors.New(
		"vec: configuracion de confianza de atestacion V3 invalida",
	)
	ErrVerificacionConfianzaAtestacionV3Fallida = errors.New(
		"vec: verificacion de confianza de atestacion V3 fallida",
	)
	ErrPruebaConfianzaAtestacionV3Invalida = errors.New(
		"vec: prueba de confianza de atestacion V3 invalida",
	)
	ErrSerializacionConfianzaAtestacionV3Prohibida = errors.New(
		"vec: serializacion generica de confianza de atestacion V3 prohibida",
	)
)

const (
	// SuiteAtestacionAutorizacionV3COSEEdDSA es la unica suite admitida por
	// este perfil. La disponibilidad de otras primitivas en verificacioncose
	// no las incorpora a la lista positiva VEC-AD-3.
	SuiteAtestacionAutorizacionV3COSEEdDSA = "VEC-AD-3-COSE-EDDSA-1"

	AlgoritmoCOSEAtestacionAutorizacionV3EdDSA = "EdDSA"

	maximoRaicesConfianzaAtestacionV3       = 64
	maximaVigenciaConfiguracionAtestacionV3 = 24 * time.Hour
	dominioHuellaConfiguracionAtestacionV3  = "vec.configuracion-confianza-atestacion-autorizacion.v3"
	marcaRaizPublicaConfianzaAtestacionV3   = "vec.raiz-publica-confianza-atestacion-autorizacion.v3"
	marcaConfiguracionConfianzaAtestacionV3 = "vec.configuracion-confianza-atestacion-autorizacion.v3"
)

// Los estados son comunes al gobierno V2; versionar el perfil criptografico
// no introduce un segundo vocabulario de rotacion o revocacion.
type EstadoClaveAtestacionAutorizacionV3 = EstadoClaveAtestacionAutorizacionV2

const (
	EstadoClaveAtestacionAutorizacionV3Activa   = EstadoClaveAtestacionAutorizacionV2Activa
	EstadoClaveAtestacionAutorizacionV3Revocada = EstadoClaveAtestacionAutorizacionV2Revocada
)

// RaizPublicaAtestacionAutorizacionV3 fija una clave Ed25519 a la audiencia
// exacta del despliegue. No acepta una cabecera ni una clave procedentes de la
// peticion verificada.
type RaizPublicaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionV3
	marca                 string
	claveID               string
	version               uint64
	clavePublica          ed25519.PublicKey
	huellaClaveSPKISHA256 string
	audienciaDespliegue   string
	estado                EstadoClaveAtestacionAutorizacionV3
	validaDesde           time.Time
	validaHasta           time.Time
	revocadaEn            time.Time
}

func NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(
	claveID string,
	version uint64,
	clavePublica ed25519.PublicKey,
	audienciaDespliegue string,
	estado EstadoClaveAtestacionAutorizacionV3,
	validaDesde time.Time,
	validaHasta time.Time,
	revocadaEn time.Time,
) (RaizPublicaAtestacionAutorizacionV3, error) {
	claveClonada, huella, err := clonarClavePublicaAtestacionV2(clavePublica)
	if err != nil {
		return RaizPublicaAtestacionAutorizacionV3{},
			ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	raiz := RaizPublicaAtestacionAutorizacionV3{
		marca:   marcaRaizPublicaConfianzaAtestacionV3,
		claveID: claveID, version: version, clavePublica: claveClonada,
		huellaClaveSPKISHA256: huella,
		audienciaDespliegue:   audienciaDespliegue,
		estado:                estado,
		validaDesde:           validaDesde,
		validaHasta:           validaHasta,
		revocadaEn:            revocadaEn,
	}
	if raiz.validar() != nil {
		return RaizPublicaAtestacionAutorizacionV3{},
			ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return raiz, nil
}

func (r RaizPublicaAtestacionAutorizacionV3) validar() error {
	_, huella, err := clonarClavePublicaAtestacionV2(r.clavePublica)
	cabecera := domain.CabeceraAtestacionAutorizacionV3{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV3,
		Suite:          SuiteAtestacionAutorizacionV3COSEEdDSA,
		ClaveID:        r.claveID,
		Audiencia:      r.audienciaDespliegue,
	}
	if r.marca != marcaRaizPublicaConfianzaAtestacionV3 || err != nil ||
		r.version == 0 ||
		huella != r.huellaClaveSPKISHA256 ||
		!huellaSHA256ConfianzaValida(huella) ||
		cabecera.Validar() != nil ||
		!r.estado.valido() ||
		!instanteCanonicoConfianza(r.validaDesde) ||
		!instanteCanonicoConfianza(r.validaHasta) ||
		!r.validaHasta.After(r.validaDesde) {
		return ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	switch r.estado {
	case EstadoClaveAtestacionAutorizacionV3Activa:
		if !r.revocadaEn.IsZero() {
			return ErrConfiguracionConfianzaAtestacionV3Invalida
		}
	case EstadoClaveAtestacionAutorizacionV3Revocada:
		if !instanteCanonicoConfianza(r.revocadaEn) ||
			r.revocadaEn.Before(r.validaDesde) {
			return ErrConfiguracionConfianzaAtestacionV3Invalida
		}
	default:
		return ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return nil
}

// ConfiguracionConfianzaAtestacionAutorizacionV3 es una instantanea cerrada
// y versionada. Su huella usa dominio V3; nunca reutiliza la huella V2.
type ConfiguracionConfianzaAtestacionAutorizacionV3 struct {
	bloqueoSerializacionV3
	marca        string
	revision     string
	secuencia    uint64
	publicadaEn  time.Time
	expiraEn     time.Time
	raices       []RaizPublicaAtestacionAutorizacionV3
	huellaSHA256 string
}

func NuevaConfiguracionConfianzaAtestacionAutorizacionV3(
	revision string,
	secuencia uint64,
	publicadaEn time.Time,
	expiraEn time.Time,
	raices ...RaizPublicaAtestacionAutorizacionV3,
) (ConfiguracionConfianzaAtestacionAutorizacionV3, error) {
	configuracion := ConfiguracionConfianzaAtestacionAutorizacionV3{
		marca:    marcaConfiguracionConfianzaAtestacionV3,
		revision: revision, secuencia: secuencia,
		publicadaEn: publicadaEn, expiraEn: expiraEn,
		raices: make([]RaizPublicaAtestacionAutorizacionV3, 0, len(raices)),
	}
	for _, raiz := range raices {
		clon, err := clonarRaizPublicaAtestacionV3(raiz)
		if err != nil {
			return ConfiguracionConfianzaAtestacionAutorizacionV3{},
				ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		configuracion.raices = append(configuracion.raices, clon)
	}
	configuracion.huellaSHA256 = configuracion.calcularHuella()
	if configuracion.validar() != nil {
		return ConfiguracionConfianzaAtestacionAutorizacionV3{},
			ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return configuracion, nil
}

func (c ConfiguracionConfianzaAtestacionAutorizacionV3) ValidarHuellaSHA256Esperada(
	esperada string,
) error {
	if c.validar() != nil || !huellaSHA256ConfianzaValida(esperada) ||
		subtle.ConstantTimeCompare([]byte(c.huellaSHA256), []byte(esperada)) != 1 {
		return ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return nil
}

// HuellaSHA256ParaGobierno entrega únicamente el testigo público que debe
// coincidir con el gobierno PostgreSQL. No exporta raíces ni material secreto.
func (c ConfiguracionConfianzaAtestacionAutorizacionV3) HuellaSHA256ParaGobierno() (
	string,
	error,
) {
	if c.validar() != nil {
		return "", ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return c.huellaSHA256, nil
}

func (c ConfiguracionConfianzaAtestacionAutorizacionV3) validar() error {
	if c.marca != marcaConfiguracionConfianzaAtestacionV3 ||
		!referenciaConfiguracionConfianzaValida(c.revision) ||
		c.secuencia == 0 ||
		!instanteCanonicoConfianza(c.publicadaEn) ||
		!instanteCanonicoConfianza(c.expiraEn) ||
		!c.expiraEn.After(c.publicadaEn) ||
		c.expiraEn.Sub(c.publicadaEn) > maximaVigenciaConfiguracionAtestacionV3 ||
		len(c.raices) == 0 || len(c.raices) > maximoRaicesConfianzaAtestacionV3 ||
		!huellaSHA256ConfianzaValida(c.huellaSHA256) {
		return ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	claves := make(map[string]struct{}, len(c.raices))
	huellas := make(map[string]struct{}, len(c.raices))
	for _, raiz := range c.raices {
		if raiz.validar() != nil {
			return ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		if _, existe := claves[raiz.claveID]; existe {
			return ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		claves[raiz.claveID] = struct{}{}
		if _, existe := huellas[raiz.huellaClaveSPKISHA256]; existe {
			return ErrConfiguracionConfianzaAtestacionV3Invalida
		}
		huellas[raiz.huellaClaveSPKISHA256] = struct{}{}
		if raiz.estado == EstadoClaveAtestacionAutorizacionV3Revocada &&
			raiz.revocadaEn.After(c.publicadaEn) {
			return ErrConfiguracionConfianzaAtestacionV3Invalida
		}
	}
	if c.huellaSHA256 != c.calcularHuella() {
		return ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return nil
}

func (c ConfiguracionConfianzaAtestacionAutorizacionV3) calcularHuella() string {
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
				strconv.FormatUint(raiz.version, 10),
				AlgoritmoCOSEAtestacionAutorizacionV3EdDSA,
				raiz.huellaClaveSPKISHA256,
				SuiteAtestacionAutorizacionV3COSEEdDSA,
				raiz.audienciaDespliegue,
				string(raiz.estado),
				raiz.validaDesde.Format(time.RFC3339Nano),
				raiz.validaHasta.Format(time.RFC3339Nano),
				revocadaEn,
			},
		})
	}
	sort.Slice(registros, func(i, j int) bool {
		return bytes.Compare(
			[]byte(registros[i].claveID),
			[]byte(registros[j].claveID),
		) < 0
	})
	calculador := sha256.New()
	escribirCampoHuellaConfianza(calculador, dominioHuellaConfiguracionAtestacionV3)
	escribirCampoHuellaConfianza(calculador, c.revision)
	escribirCampoHuellaConfianza(
		calculador,
		strconv.FormatUint(c.secuencia, 10),
	)
	escribirCampoHuellaConfianza(calculador, c.publicadaEn.Format(time.RFC3339Nano))
	escribirCampoHuellaConfianza(calculador, c.expiraEn.Format(time.RFC3339Nano))
	for _, registro := range registros {
		for _, campo := range registro.campos {
			escribirCampoHuellaConfianza(calculador, campo)
		}
	}
	return hex.EncodeToString(calculador.Sum(nil))
}

func clonarRaizPublicaAtestacionV3(
	raiz RaizPublicaAtestacionAutorizacionV3,
) (RaizPublicaAtestacionAutorizacionV3, error) {
	clave, huella, err := clonarClavePublicaAtestacionV2(raiz.clavePublica)
	if raiz.validar() != nil || err != nil ||
		huella != raiz.huellaClaveSPKISHA256 {
		return RaizPublicaAtestacionAutorizacionV3{},
			ErrConfiguracionConfianzaAtestacionV3Invalida
	}
	return RaizPublicaAtestacionAutorizacionV3{
		marca: raiz.marca, claveID: raiz.claveID, version: raiz.version,
		clavePublica:          clave,
		huellaClaveSPKISHA256: raiz.huellaClaveSPKISHA256,
		audienciaDespliegue:   raiz.audienciaDespliegue, estado: raiz.estado,
		validaDesde: raiz.validaDesde, validaHasta: raiz.validaHasta,
		revocadaEn: raiz.revocadaEn,
	}, nil
}
