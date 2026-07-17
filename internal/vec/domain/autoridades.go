package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var (
	ErrFuenteAutoridadInvalida        = errors.New("vec: fuente de autoridad invalida")
	ErrReferenciaAutoridadInvalida    = errors.New("vec: referencia de autoridad invalida")
	ErrEvidenciaActoAutoridadInvalida = errors.New("vec: evidencia de acto de autoridad invalida")
	ErrTransicionAutoridadInvalida    = errors.New("vec: transicion de fuente de autoridad invalida")
	ErrRevisionAutoridadEnConflicto   = errors.New("vec: revision de fuente de autoridad en conflicto")
	ErrSolicitudAutoridadObsoleta     = errors.New("vec: solicitud de transicion de autoridad obsoleta")
	ErrSolicitudAutoridadExpirada     = errors.New("vec: solicitud de transicion de autoridad expirada")
	ErrLimiteAutoridadAlcanzado       = errors.New("vec: limite de fuente de autoridad alcanzado")
)

const (
	maximoAmbitosFuenteAutoridad      = 128
	maximoValoresAmbitoAutoridad      = 512
	maximoPreceptosFuenteAutoridad    = 1_024
	maximoEdicionesFuenteAutoridad    = 128
	maximoTransicionesFuenteAutoridad = 128
	maximoFirmasActoFuenteAutoridad   = 64
	maximoCaracteresNombreAutoridad   = 1_024
	maximoCaracteresCitaAutoridad     = 2_048
	maximoBytesContenidoAutoridad     = 4 * 1_024 * 1_024
	maximoBytesSobreContenido         = maximoBytesContenidoAutoridad + 4*1_024
	maximoBytesEstadoAutoridad        = 16 * 1_024 * 1_024
)

const (
	AccionFuenteAutoridadBorradorCreado      = "vec.fuentes_autoridad.borrador.creado"
	AccionFuenteAutoridadBorradorActualizado = "vec.fuentes_autoridad.borrador.actualizado"
	AccionFuenteAutoridadPublicada           = "vec.fuentes_autoridad.publicada"
	AccionFuenteAutoridadSuspendida          = "vec.fuentes_autoridad.suspendida"
	AccionFuenteAutoridadSuspensionLevantada = "vec.fuentes_autoridad.suspension_levantada"
	AccionFuenteAutoridadDerogada            = "vec.fuentes_autoridad.derogada"
)

// PeriodoFuenteAutoridad representa un intervalo semiabierto [desde, hasta).
// Hasta cero significa que el periodo no tiene fin conocido. Vigencia y
// efectos usan instancias distintas: ninguna de las dos se deduce de la otra.
type PeriodoFuenteAutoridad struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta,omitempty"`
}

func (p PeriodoFuenteAutoridad) Validar() error {
	if !instanteFuenteAutoridadCanonico(p.Desde) ||
		(!p.Hasta.IsZero() && (!instanteFuenteAutoridadCanonico(p.Hasta) || !p.Hasta.After(p.Desde))) {
		return ErrFuenteAutoridadInvalida
	}
	return nil
}

func (p PeriodoFuenteAutoridad) Contiene(instante time.Time) bool {
	if p.Validar() != nil || instante.IsZero() {
		return false
	}
	instante = normalizarInstanteFuenteAutoridad(instante)
	return instanteFuenteAutoridadCanonico(instante) &&
		!instante.Before(p.Desde) && (p.Hasta.IsZero() || instante.Before(p.Hasta))
}

// AmbitoFuenteAutoridad usa claves gobernadas. El nucleo no interpreta
// valores como colectivos, territorios o centros ni compila sus catalogos.
type AmbitoFuenteAutoridad struct {
	DimensionClave string   `json:"dimension_clave"`
	ValoresClave   []string `json:"valores_clave"`
}

func (a AmbitoFuenteAutoridad) Validar() error {
	if !esClaveDocumentalCanonica(a.DimensionClave) || len(a.ValoresClave) == 0 ||
		len(a.ValoresClave) > maximoValoresAmbitoAutoridad {
		return ErrFuenteAutoridadInvalida
	}
	vistos := make(map[string]struct{}, len(a.ValoresClave))
	for _, valor := range a.ValoresClave {
		if !esClaveDocumentalCanonica(valor) {
			return ErrFuenteAutoridadInvalida
		}
		if _, repetido := vistos[valor]; repetido {
			return ErrFuenteAutoridadInvalida
		}
		vistos[valor] = struct{}{}
	}
	return nil
}

// PreceptoFuenteAutoridad identifica un articulo, apartado, anexo o seccion.
// Cita es solo una etiqueta verificable por personas; no se ejecuta ni se
// interpreta como expresion.
type PreceptoFuenteAutoridad struct {
	Clave string `json:"clave"`
	Cita  string `json:"cita"`
}

func (p PreceptoFuenteAutoridad) Validar() error {
	if !esClaveDocumentalCanonica(p.Clave) || !textoFuenteAutoridadSeguro(p.Cita, maximoCaracteresCitaAutoridad, true) {
		return ErrFuenteAutoridadInvalida
	}
	return nil
}

// DocumentoFuenteAutoridad fija la representacion concreta examinada. El
// contenido y las firmas siguen custodiados por las capacidades documentales;
// este agregado solo conserva referencias opacas y huellas.
type DocumentoFuenteAutoridad struct {
	DocumentoID           string `json:"documento_id"`
	DocumentoVersion      uint64 `json:"documento_version"`
	RepresentacionRef     string `json:"representacion_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	PublicacionOficialRef string `json:"publicacion_oficial_ref"`
	ActoOrigenRef         string `json:"acto_origen_ref"`
	OrganoEmisorRef       string `json:"organo_emisor_ref"`
}

func (d DocumentoFuenteAutoridad) Validar() error {
	if !referenciaFuenteAutoridadValida(d.DocumentoID) || d.DocumentoVersion < 1 ||
		!referenciaFuenteAutoridadValida(d.RepresentacionRef) || !esSHA256Autoridad(d.HuellaContenidoSHA256) ||
		!referenciaFuenteAutoridadValida(d.PublicacionOficialRef) ||
		!referenciaFuenteAutoridadValida(d.ActoOrigenRef) || !referenciaFuenteAutoridadValida(d.OrganoEmisorRef) {
		return ErrFuenteAutoridadInvalida
	}
	return nil
}

// ContenidoFuenteAutoridad agrupa la semantica inmutable al publicarse. No
// contiene texto normativo, datos personales ni parametros de reglas.
type ContenidoFuenteAutoridad struct {
	MateriaClave string                    `json:"materia_clave"`
	Nombre       string                    `json:"nombre"`
	Ambitos      []AmbitoFuenteAutoridad   `json:"ambitos"`
	Documento    DocumentoFuenteAutoridad  `json:"documento"`
	Preceptos    []PreceptoFuenteAutoridad `json:"preceptos"`
	Vigencia     PeriodoFuenteAutoridad    `json:"vigencia"`
	Efectos      PeriodoFuenteAutoridad    `json:"efectos"`
	ConocidaEn   time.Time                 `json:"conocida_en"`
}

func (c ContenidoFuenteAutoridad) Validar() error {
	_, _, err := prepararContenidoFuenteAutoridad(c)
	return err
}

func (c ContenidoFuenteAutoridad) validarEstructura() error {
	if !esClaveDocumentalCanonica(c.MateriaClave) ||
		!textoFuenteAutoridadSeguro(c.Nombre, maximoCaracteresNombreAutoridad, true) ||
		len(c.Ambitos) == 0 || len(c.Ambitos) > maximoAmbitosFuenteAutoridad ||
		len(c.Preceptos) == 0 || len(c.Preceptos) > maximoPreceptosFuenteAutoridad ||
		c.Documento.Validar() != nil || c.Vigencia.Validar() != nil || c.Efectos.Validar() != nil ||
		!instanteFuenteAutoridadCanonico(c.ConocidaEn) {
		return ErrFuenteAutoridadInvalida
	}
	dimensiones := make(map[string]struct{}, len(c.Ambitos))
	for _, ambito := range c.Ambitos {
		if ambito.Validar() != nil {
			return ErrFuenteAutoridadInvalida
		}
		if _, repetida := dimensiones[ambito.DimensionClave]; repetida {
			return ErrFuenteAutoridadInvalida
		}
		dimensiones[ambito.DimensionClave] = struct{}{}
	}
	preceptos := make(map[string]struct{}, len(c.Preceptos))
	for _, precepto := range c.Preceptos {
		if precepto.Validar() != nil {
			return ErrFuenteAutoridadInvalida
		}
		if _, repetido := preceptos[precepto.Clave]; repetido {
			return ErrFuenteAutoridadInvalida
		}
		preceptos[precepto.Clave] = struct{}{}
	}
	return nil
}

func (c ContenidoFuenteAutoridad) ClonarCanonico() (ContenidoFuenteAutoridad, error) {
	clon, _, err := prepararContenidoFuenteAutoridad(c)
	return clon, err
}

func clonarContenidoFuenteAutoridadOrdenado(c ContenidoFuenteAutoridad) ContenidoFuenteAutoridad {
	clon := c
	clon.Ambitos = make([]AmbitoFuenteAutoridad, len(c.Ambitos))
	for indice, ambito := range c.Ambitos {
		clon.Ambitos[indice] = ambito
		clon.Ambitos[indice].ValoresClave = append([]string(nil), ambito.ValoresClave...)
		sort.Strings(clon.Ambitos[indice].ValoresClave)
	}
	sort.Slice(clon.Ambitos, func(i, j int) bool {
		return clon.Ambitos[i].DimensionClave < clon.Ambitos[j].DimensionClave
	})
	clon.Preceptos = append([]PreceptoFuenteAutoridad(nil), c.Preceptos...)
	sort.Slice(clon.Preceptos, func(i, j int) bool {
		if clon.Preceptos[i].Clave != clon.Preceptos[j].Clave {
			return clon.Preceptos[i].Clave < clon.Preceptos[j].Clave
		}
		return clon.Preceptos[i].Cita < clon.Preceptos[j].Cita
	})
	return clon
}

func prepararContenidoFuenteAutoridad(c ContenidoFuenteAutoridad) (ContenidoFuenteAutoridad, []byte, error) {
	if err := c.validarEstructura(); err != nil {
		return ContenidoFuenteAutoridad{}, nil, err
	}
	clon := clonarContenidoFuenteAutoridadOrdenado(c)
	bytesCanonicos, err := serializarContenidoPersistibleAutoridadV1(clon)
	if err != nil {
		return ContenidoFuenteAutoridad{}, nil, ErrFuenteAutoridadInvalida
	}
	return clon, bytesCanonicos, nil
}

type ReferenciaFuenteAutoridad struct {
	FuenteID              string `json:"fuente_id"`
	Version               uint64 `json:"version"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
}

// ReferenciaLinajeFuenteAutoridad fija no solo el contenido de la
// predecesora, sino también el estado e historia exactos desde los que nació
// una sucesora. No se usa como cita funcional.
type ReferenciaLinajeFuenteAutoridad struct {
	Fuente               ReferenciaFuenteAutoridad `json:"fuente"`
	Revision             uint64                    `json:"revision"`
	Estado               EstadoFuenteAutoridad     `json:"estado"`
	HuellaHistoriaSHA256 string                    `json:"huella_historia_sha256"`
	HuellaEstadoSHA256   string                    `json:"huella_estado_sha256"`
}

func (r ReferenciaLinajeFuenteAutoridad) Validar() error {
	if r.Fuente.Validar() != nil || r.Revision == 0 || !r.Estado.Valido() ||
		r.Estado == EstadoFuenteAutoridadBorrador || !esSHA256Autoridad(r.HuellaHistoriaSHA256) ||
		!esSHA256Autoridad(r.HuellaEstadoSHA256) {
		return ErrReferenciaAutoridadInvalida
	}
	return nil
}

func (r ReferenciaFuenteAutoridad) Validar() error {
	if !esClaveDocumentalCanonica(r.FuenteID) || r.Version < 1 || !esSHA256Autoridad(r.HuellaContenidoSHA256) {
		return ErrReferenciaAutoridadInvalida
	}
	return nil
}

func (r ReferenciaFuenteAutoridad) Referencia() (string, error) {
	if err := r.Validar(); err != nil {
		return "", err
	}
	return "fuente:" + r.FuenteID + ":v" + strconv.FormatUint(r.Version, 10) + ":sha256:" + r.HuellaContenidoSHA256, nil
}

// CitaFuenteAutoridad selecciona preceptos exactos de una version. Una lista
// vacia no significa "toda la fuente".
type CitaFuenteAutoridad struct {
	Fuente    ReferenciaFuenteAutoridad `json:"fuente"`
	Preceptos []string                  `json:"preceptos"`
}

func (c CitaFuenteAutoridad) Validar() error {
	if c.Fuente.Validar() != nil || len(c.Preceptos) == 0 || len(c.Preceptos) > maximoPreceptosFuenteAutoridad {
		return ErrReferenciaAutoridadInvalida
	}
	vistos := make(map[string]struct{}, len(c.Preceptos))
	for _, precepto := range c.Preceptos {
		if !esClaveDocumentalCanonica(precepto) {
			return ErrReferenciaAutoridadInvalida
		}
		if _, repetido := vistos[precepto]; repetido {
			return ErrReferenciaAutoridadInvalida
		}
		vistos[precepto] = struct{}{}
	}
	return nil
}

func (c CitaFuenteAutoridad) ClonarCanonica() (CitaFuenteAutoridad, error) {
	if err := c.Validar(); err != nil {
		return CitaFuenteAutoridad{}, err
	}
	clon := c
	clon.Preceptos = append([]string(nil), c.Preceptos...)
	sort.Strings(clon.Preceptos)
	return clon, nil
}

type EstadoFuenteAutoridad string

const (
	EstadoFuenteAutoridadBorrador   EstadoFuenteAutoridad = "borrador"
	EstadoFuenteAutoridadPublicada  EstadoFuenteAutoridad = "publicada"
	EstadoFuenteAutoridadSuspendida EstadoFuenteAutoridad = "suspendida"
	EstadoFuenteAutoridadDerogada   EstadoFuenteAutoridad = "derogada"
)

func (e EstadoFuenteAutoridad) Valido() bool {
	return e == EstadoFuenteAutoridadBorrador || e == EstadoFuenteAutoridadPublicada ||
		e == EstadoFuenteAutoridadSuspendida || e == EstadoFuenteAutoridadDerogada
}

type AccionActoFuenteAutoridad string

const (
	AccionActoPublicarFuenteAutoridad           AccionActoFuenteAutoridad = "publicar"
	AccionActoSuspenderFuenteAutoridad          AccionActoFuenteAutoridad = "suspender"
	AccionActoLevantarSuspensionFuenteAutoridad AccionActoFuenteAutoridad = "levantar_suspension"
	AccionActoDerogarFuenteAutoridad            AccionActoFuenteAutoridad = "derogar"
)

func (a AccionActoFuenteAutoridad) Valida() bool {
	return a == AccionActoPublicarFuenteAutoridad || a == AccionActoSuspenderFuenteAutoridad ||
		a == AccionActoLevantarSuspensionFuenteAutoridad || a == AccionActoDerogarFuenteAutoridad
}

// CodigoMotivoFuenteAutoridad referencia un motivo gobernado por catalogo.
// El detalle humano y su documentacion justificativa viven fuera del agregado
// con clasificacion y conservacion propias.
type CodigoMotivoFuenteAutoridad string

func (m CodigoMotivoFuenteAutoridad) Valido() bool {
	return esClaveDocumentalCanonica(string(m))
}

func normalizarContenidoFuenteAutoridad(contenido ContenidoFuenteAutoridad) (ContenidoFuenteAutoridad, error) {
	contenido.ConocidaEn = normalizarInstanteFuenteAutoridad(contenido.ConocidaEn)
	contenido.Vigencia.Desde = normalizarInstanteFuenteAutoridad(contenido.Vigencia.Desde)
	if !contenido.Vigencia.Hasta.IsZero() {
		contenido.Vigencia.Hasta = normalizarInstanteFuenteAutoridad(contenido.Vigencia.Hasta)
	}
	contenido.Efectos.Desde = normalizarInstanteFuenteAutoridad(contenido.Efectos.Desde)
	if !contenido.Efectos.Hasta.IsZero() {
		contenido.Efectos.Hasta = normalizarInstanteFuenteAutoridad(contenido.Efectos.Hasta)
	}
	return contenido.ClonarCanonico()
}

func prepararHuellaContenidoFuenteAutoridad(
	id string,
	version uint64,
	versionAnterior ReferenciaLinajeFuenteAutoridad,
	contenido ContenidoFuenteAutoridad,
) (ContenidoFuenteAutoridad, string, error) {
	if !esClaveDocumentalCanonica(id) || version < 1 {
		return ContenidoFuenteAutoridad{}, "", ErrFuenteAutoridadInvalida
	}
	if version == 1 {
		if versionAnterior != (ReferenciaLinajeFuenteAutoridad{}) {
			return ContenidoFuenteAutoridad{}, "", ErrFuenteAutoridadInvalida
		}
	} else if versionAnterior.Validar() != nil || versionAnterior.Fuente.FuenteID != id ||
		versionAnterior.Fuente.Version != version-1 {
		return ContenidoFuenteAutoridad{}, "", ErrFuenteAutoridadInvalida
	}
	contenido, _, err := prepararContenidoFuenteAutoridad(contenido)
	if err != nil {
		return ContenidoFuenteAutoridad{}, "", err
	}
	bytesCanonicos, err := serializarSobreContenidoPersistibleAutoridadV1(
		id, version, versionAnterior, contenido,
	)
	if err != nil {
		return ContenidoFuenteAutoridad{}, "", err
	}
	return contenido, huellaBytesFuenteAutoridad(bytesCanonicos), nil
}

func huellaValorFuenteAutoridad(valor any, maximoBytes int) (string, error) {
	canonico, err := json.Marshal(valor)
	if err != nil || len(canonico) == 0 || len(canonico) > maximoBytes {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaBytesFuenteAutoridad(canonico), nil
}

func huellaBytesFuenteAutoridad(canonico []byte) string {
	suma := sha256.Sum256(canonico)
	return hex.EncodeToString(suma[:])
}

func normalizarInstanteFuenteAutoridad(instante time.Time) time.Time {
	if instante.IsZero() {
		return time.Time{}
	}
	return instante.UTC().Truncate(time.Microsecond)
}

func instanteFuenteAutoridadCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func referenciaFuenteAutoridadValida(valor string) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximoCaracteresReferenciaDocumental {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		caracter := valor[indice]
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') {
			continue
		}
		switch caracter {
		case '-', '_', '.', ':', '/', '@', '+':
			continue
		default:
			return false
		}
	}
	return true
}

func referenciaPersonaFuenteAutoridadValida(valor string) bool {
	return referenciaOpacaContextoActorValida(valor, "per_")
}

const huellaSHA256Nula = "0000000000000000000000000000000000000000000000000000000000000000"

func esSHA256Autoridad(valor string) bool {
	return esSHA256(valor) && valor != huellaSHA256Nula
}

func textoFuenteAutoridadSeguro(valor string, maximo int, permiteEspacios bool) bool {
	if valor == "" || valor != strings.TrimSpace(valor) || len(valor) > maximo || !utf8.ValidString(valor) ||
		!norm.NFC.IsNormalString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsControl(caracter) || unicode.Is(unicode.Cf, caracter) ||
			(!permiteEspacios && unicode.IsSpace(caracter)) {
			return false
		}
	}
	return true
}
