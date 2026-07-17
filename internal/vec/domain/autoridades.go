package domain

import (
	"bytes"
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

const esquemaCompromisoTransicionFuenteAutoridadV1 = "vec.fuente_autoridad.compromiso_transicion.v1"

// CompromisoTransicionFuenteAutoridadV1 fija todos los datos que el
// comprobador debe atestar. Cambiar cualquiera de ellos invalida la evidencia.
type CompromisoTransicionFuenteAutoridadV1 struct {
	Esquema                    string                      `json:"esquema"`
	SolicitudRef               string                      `json:"solicitud_ref"`
	Fuente                     ReferenciaFuenteAutoridad   `json:"fuente"`
	RevisionPrevia             uint64                      `json:"revision_previa"`
	Secuencia                  uint64                      `json:"secuencia"`
	EstadoAnterior             EstadoFuenteAutoridad       `json:"estado_anterior"`
	EstadoNuevo                EstadoFuenteAutoridad       `json:"estado_nuevo"`
	Accion                     AccionActoFuenteAutoridad   `json:"accion"`
	ActorRef                   string                      `json:"actor_ref"`
	MotivoCodigo               CodigoMotivoFuenteAutoridad `json:"motivo_codigo"`
	HuellaHistoriaPreviaSHA256 string                      `json:"huella_historia_previa_sha256"`
	PreparadaEn                time.Time                   `json:"preparada_en"`
	ExpiraEn                   time.Time                   `json:"expira_en"`
}

func (c CompromisoTransicionFuenteAutoridadV1) Validar() error {
	accion, valida := accionActoParaTransicionFuenteAutoridad(c.EstadoAnterior, c.EstadoNuevo)
	if c.Esquema != esquemaCompromisoTransicionFuenteAutoridadV1 ||
		!referenciaFuenteAutoridadValida(c.SolicitudRef) || c.Fuente.Validar() != nil ||
		c.RevisionPrevia == 0 || c.Secuencia == 0 || !c.EstadoAnterior.Valido() || !c.EstadoNuevo.Valido() ||
		!valida || c.Accion != accion || !referenciaPersonaFuenteAutoridadValida(c.ActorRef) ||
		!c.MotivoCodigo.Valido() || !esSHA256Autoridad(c.HuellaHistoriaPreviaSHA256) ||
		!instanteFuenteAutoridadCanonico(c.PreparadaEn) || !instanteFuenteAutoridadCanonico(c.ExpiraEn) ||
		!c.ExpiraEn.After(c.PreparadaEn) {
		return ErrTransicionAutoridadInvalida
	}
	return nil
}

func (c CompromisoTransicionFuenteAutoridadV1) HuellaSHA256() (string, error) {
	bytesCanonicos, err := c.BytesCanonicos()
	if err != nil {
		return "", err
	}
	return huellaBytesFuenteAutoridad(bytesCanonicos), nil
}

func (c CompromisoTransicionFuenteAutoridadV1) BytesCanonicos() ([]byte, error) {
	if err := c.Validar(); err != nil {
		return nil, err
	}
	return serializarCompromisoPersistibleAutoridadV1(c)
}

// MarshalJSON fuerza a todos los conectores a usar el mismo compromiso V1
// que se firma y cuya huella se conserva. No se serializa el tipo vivo.
func (c CompromisoTransicionFuenteAutoridadV1) MarshalJSON() ([]byte, error) {
	return c.BytesCanonicos()
}

// SolicitudTransicionFuenteAutoridadV1 evita que quien integra el caso de uso
// repita actor, motivo, estado o instante entre la firma y la aplicacion.
type SolicitudTransicionFuenteAutoridadV1 struct {
	compromiso     CompromisoTransicionFuenteAutoridadV1
	bytesCanonicos []byte
}

type DatosPreparacionTransicionFuenteAutoridadV1 struct {
	EstadoNuevo  EstadoFuenteAutoridad
	ActorRef     string
	MotivoCodigo CodigoMotivoFuenteAutoridad
	SolicitudRef string
	PreparadaEn  time.Time
	ExpiraEn     time.Time
}

func nuevaSolicitudTransicionFuenteAutoridadV1(
	compromiso CompromisoTransicionFuenteAutoridadV1,
) (SolicitudTransicionFuenteAutoridadV1, error) {
	bytesCanonicos, err := compromiso.BytesCanonicos()
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, err
	}
	return SolicitudTransicionFuenteAutoridadV1{
		compromiso: compromiso, bytesCanonicos: append([]byte(nil), bytesCanonicos...),
	}, nil
}

func (s SolicitudTransicionFuenteAutoridadV1) Validar() error {
	bytesCanonicos, err := s.compromiso.BytesCanonicos()
	if err != nil || len(s.bytesCanonicos) == 0 || !bytes.Equal(bytesCanonicos, s.bytesCanonicos) {
		return ErrTransicionAutoridadInvalida
	}
	return nil
}

func (s SolicitudTransicionFuenteAutoridadV1) Compromiso() (CompromisoTransicionFuenteAutoridadV1, error) {
	if err := s.Validar(); err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	return s.compromiso, nil
}

func (s SolicitudTransicionFuenteAutoridadV1) BytesCanonicos() ([]byte, error) {
	if err := s.Validar(); err != nil {
		return nil, err
	}
	return append([]byte(nil), s.bytesCanonicos...), nil
}

// MarshalJSON evita que la opacidad de la solicitud se convierta
// accidentalmente en {}. La representación es el compromiso V1 canónico que
// puede custodiarse mientras un portafirmas completa el acto.
func (s SolicitudTransicionFuenteAutoridadV1) MarshalJSON() ([]byte, error) {
	return s.BytesCanonicos()
}

// EvidenciaActoFuenteAutoridad es una atestacion neutral producida por un
// puerto de comprobacion. Validar comprueba coherencia estructural, no firma,
// competencia ni procedencia criptografica.
type EvidenciaActoFuenteAutoridad struct {
	EvidenciaRef                string                    `json:"evidencia_ref"`
	Accion                      AccionActoFuenteAutoridad `json:"accion"`
	FuenteID                    string                    `json:"fuente_id"`
	FuenteVersion               uint64                    `json:"fuente_version"`
	HuellaContenidoSHA256       string                    `json:"huella_contenido_sha256"`
	ActoRef                     string                    `json:"acto_ref"`
	DocumentoRef                string                    `json:"documento_ref"`
	RepresentacionRef           string                    `json:"representacion_ref"`
	HuellaDocumentoSHA256       string                    `json:"huella_documento_sha256"`
	OrganoRef                   string                    `json:"organo_ref"`
	FirmasRefs                  []string                  `json:"firmas_refs"`
	ComprobadorRef              string                    `json:"comprobador_ref"`
	AtestacionRef               string                    `json:"atestacion_ref"`
	HuellaAtestacionSHA256      string                    `json:"huella_atestacion_sha256"`
	FirmaAtestacionRef          string                    `json:"firma_atestacion_ref"`
	HuellaCompromisoSHA256      string                    `json:"huella_compromiso_sha256"`
	HuellaMensajeAtestadoSHA256 string                    `json:"huella_mensaje_atestado_sha256"`
	ActoOcurridoEn              time.Time                 `json:"acto_ocurrido_en"`
	ComprobadaEn                time.Time                 `json:"comprobada_en"`
}

func (e EvidenciaActoFuenteAutoridad) Validar() error {
	if !referenciaFuenteAutoridadValida(e.EvidenciaRef) || !e.Accion.Valida() ||
		!esClaveDocumentalCanonica(e.FuenteID) || e.FuenteVersion < 1 || !esSHA256Autoridad(e.HuellaContenidoSHA256) ||
		!referenciaFuenteAutoridadValida(e.ActoRef) || !referenciaFuenteAutoridadValida(e.DocumentoRef) ||
		!referenciaFuenteAutoridadValida(e.RepresentacionRef) || !esSHA256Autoridad(e.HuellaDocumentoSHA256) ||
		!referenciaFuenteAutoridadValida(e.OrganoRef) || len(e.FirmasRefs) == 0 ||
		len(e.FirmasRefs) > maximoFirmasActoFuenteAutoridad ||
		!referenciaFuenteAutoridadValida(e.ComprobadorRef) || !referenciaFuenteAutoridadValida(e.AtestacionRef) ||
		!esSHA256Autoridad(e.HuellaAtestacionSHA256) || !referenciaFuenteAutoridadValida(e.FirmaAtestacionRef) ||
		!esSHA256Autoridad(e.HuellaCompromisoSHA256) || !esSHA256Autoridad(e.HuellaMensajeAtestadoSHA256) ||
		!instanteFuenteAutoridadCanonico(e.ActoOcurridoEn) ||
		!instanteFuenteAutoridadCanonico(e.ComprobadaEn) || e.ComprobadaEn.Before(e.ActoOcurridoEn) {
		return ErrEvidenciaActoAutoridadInvalida
	}
	vistas := make(map[string]struct{}, len(e.FirmasRefs))
	for _, firma := range e.FirmasRefs {
		if !referenciaFuenteAutoridadValida(firma) {
			return ErrEvidenciaActoAutoridadInvalida
		}
		if _, repetida := vistas[firma]; repetida {
			return ErrEvidenciaActoAutoridadInvalida
		}
		vistas[firma] = struct{}{}
	}
	return nil
}

func (e EvidenciaActoFuenteAutoridad) ClonarCanonica() (EvidenciaActoFuenteAutoridad, error) {
	if err := e.Validar(); err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	clon := e
	clon.FirmasRefs = append([]string(nil), e.FirmasRefs...)
	sort.Strings(clon.FirmasRefs)
	return clon, nil
}

const esquemaMensajeAtestacionActoFuenteAutoridadV1 = "vec.fuente_autoridad.mensaje_atestacion_acto.v1"

// MensajeAtestacionActoFuenteAutoridadV1 es el mensaje completo que cubre la
// atestacion externa. Excluye unicamente el sobre criptografico que lo firma
// para evitar una dependencia circular.
type MensajeAtestacionActoFuenteAutoridadV1 struct {
	Esquema               string                                `json:"esquema"`
	Compromiso            CompromisoTransicionFuenteAutoridadV1 `json:"compromiso"`
	EvidenciaRef          string                                `json:"evidencia_ref"`
	ActoRef               string                                `json:"acto_ref"`
	DocumentoRef          string                                `json:"documento_ref"`
	RepresentacionRef     string                                `json:"representacion_ref"`
	HuellaDocumentoSHA256 string                                `json:"huella_documento_sha256"`
	OrganoRef             string                                `json:"organo_ref"`
	FirmasRefs            []string                              `json:"firmas_refs"`
	ComprobadorRef        string                                `json:"comprobador_ref"`
	ActoOcurridoEn        time.Time                             `json:"acto_ocurrido_en"`
	ComprobadaEn          time.Time                             `json:"comprobada_en"`
}

// DatosMensajeAtestacionActoFuenteAutoridadV1 son hechos producidos por el
// comprobador. No incluyen campos derivados ni el sobre que todavía debe
// firmar el mensaje.
type DatosMensajeAtestacionActoFuenteAutoridadV1 struct {
	EvidenciaRef          string
	ActoRef               string
	DocumentoRef          string
	RepresentacionRef     string
	HuellaDocumentoSHA256 string
	OrganoRef             string
	FirmasRefs            []string
	ComprobadorRef        string
	ActoOcurridoEn        time.Time
	ComprobadaEn          time.Time
}

type DatosSobreAtestacionActoFuenteAutoridadV1 struct {
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	FirmaAtestacionRef     string
}

// PrepararMensajeAtestacionActoFuenteAutoridadV1 construye el único mensaje
// que un conector puede firmar. El adaptador no serializa el compromiso ni
// repite actor, recurso, revisión o acción.
func PrepararMensajeAtestacionActoFuenteAutoridadV1(
	solicitud SolicitudTransicionFuenteAutoridadV1,
	datos DatosMensajeAtestacionActoFuenteAutoridadV1,
) (MensajeAtestacionActoFuenteAutoridadV1, error) {
	compromiso, err := solicitud.Compromiso()
	if err != nil {
		return MensajeAtestacionActoFuenteAutoridadV1{}, err
	}
	mensaje := MensajeAtestacionActoFuenteAutoridadV1{
		Esquema: esquemaMensajeAtestacionActoFuenteAutoridadV1, Compromiso: compromiso,
		EvidenciaRef: datos.EvidenciaRef, ActoRef: datos.ActoRef,
		DocumentoRef: datos.DocumentoRef, RepresentacionRef: datos.RepresentacionRef,
		HuellaDocumentoSHA256: datos.HuellaDocumentoSHA256, OrganoRef: datos.OrganoRef,
		FirmasRefs: append([]string(nil), datos.FirmasRefs...), ComprobadorRef: datos.ComprobadorRef,
		ActoOcurridoEn: normalizarInstanteFuenteAutoridad(datos.ActoOcurridoEn),
		ComprobadaEn:   normalizarInstanteFuenteAutoridad(datos.ComprobadaEn),
	}
	if _, err := mensaje.BytesCanonicos(); err != nil {
		return MensajeAtestacionActoFuenteAutoridadV1{}, err
	}
	return mensaje, nil
}

// ConstituirEvidenciaAtestadaV1 incorpora el sobre criptográfico después de
// firmar/verificar el mensaje y calcula todos los campos derivados.
func (m MensajeAtestacionActoFuenteAutoridadV1) ConstituirEvidenciaAtestadaV1(
	sobre DatosSobreAtestacionActoFuenteAutoridadV1,
) (EvidenciaActoFuenteAutoridad, error) {
	if !referenciaFuenteAutoridadValida(sobre.AtestacionRef) ||
		!esSHA256Autoridad(sobre.HuellaAtestacionSHA256) ||
		!referenciaFuenteAutoridadValida(sobre.FirmaAtestacionRef) {
		return EvidenciaActoFuenteAutoridad{}, ErrEvidenciaActoAutoridadInvalida
	}
	huellaCompromiso, err := m.Compromiso.HuellaSHA256()
	if err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	huellaMensaje, err := m.HuellaSHA256()
	if err != nil {
		return EvidenciaActoFuenteAutoridad{}, err
	}
	evidencia := EvidenciaActoFuenteAutoridad{
		EvidenciaRef: m.EvidenciaRef, Accion: m.Compromiso.Accion,
		FuenteID: m.Compromiso.Fuente.FuenteID, FuenteVersion: m.Compromiso.Fuente.Version,
		HuellaContenidoSHA256: m.Compromiso.Fuente.HuellaContenidoSHA256,
		ActoRef:               m.ActoRef, DocumentoRef: m.DocumentoRef, RepresentacionRef: m.RepresentacionRef,
		HuellaDocumentoSHA256: m.HuellaDocumentoSHA256, OrganoRef: m.OrganoRef,
		FirmasRefs: append([]string(nil), m.FirmasRefs...), ComprobadorRef: m.ComprobadorRef,
		AtestacionRef: sobre.AtestacionRef, HuellaAtestacionSHA256: sobre.HuellaAtestacionSHA256,
		FirmaAtestacionRef: sobre.FirmaAtestacionRef, HuellaCompromisoSHA256: huellaCompromiso,
		HuellaMensajeAtestadoSHA256: huellaMensaje,
		ActoOcurridoEn:              m.ActoOcurridoEn, ComprobadaEn: m.ComprobadaEn,
	}
	return evidencia.ClonarCanonica()
}

func (m MensajeAtestacionActoFuenteAutoridadV1) Validar() error {
	if m.Esquema != esquemaMensajeAtestacionActoFuenteAutoridadV1 || m.Compromiso.Validar() != nil ||
		!referenciaFuenteAutoridadValida(m.EvidenciaRef) || !referenciaFuenteAutoridadValida(m.ActoRef) ||
		!referenciaFuenteAutoridadValida(m.DocumentoRef) || !referenciaFuenteAutoridadValida(m.RepresentacionRef) ||
		!esSHA256Autoridad(m.HuellaDocumentoSHA256) || !referenciaFuenteAutoridadValida(m.OrganoRef) ||
		len(m.FirmasRefs) == 0 || len(m.FirmasRefs) > maximoFirmasActoFuenteAutoridad ||
		!referenciaFuenteAutoridadValida(m.ComprobadorRef) || !instanteFuenteAutoridadCanonico(m.ActoOcurridoEn) ||
		!instanteFuenteAutoridadCanonico(m.ComprobadaEn) || m.ComprobadaEn.Before(m.ActoOcurridoEn) ||
		m.ComprobadaEn.Before(m.Compromiso.PreparadaEn) || !m.ComprobadaEn.Before(m.Compromiso.ExpiraEn) {
		return ErrEvidenciaActoAutoridadInvalida
	}
	vistas := make(map[string]struct{}, len(m.FirmasRefs))
	for _, firma := range m.FirmasRefs {
		if !referenciaFuenteAutoridadValida(firma) {
			return ErrEvidenciaActoAutoridadInvalida
		}
		if _, repetida := vistas[firma]; repetida {
			return ErrEvidenciaActoAutoridadInvalida
		}
		vistas[firma] = struct{}{}
	}
	return nil
}

func (m MensajeAtestacionActoFuenteAutoridadV1) BytesCanonicos() ([]byte, error) {
	if err := m.Validar(); err != nil {
		return nil, err
	}
	m.FirmasRefs = append([]string(nil), m.FirmasRefs...)
	sort.Strings(m.FirmasRefs)
	return serializarMensajeAtestacionPersistibleAutoridadV1(m)
}

// MarshalJSON evita que el orden recibido de las firmas u otro detalle del
// tipo vivo produzca unos bytes distintos de los entregados a Portafirmas.
func (m MensajeAtestacionActoFuenteAutoridadV1) MarshalJSON() ([]byte, error) {
	return m.BytesCanonicos()
}

func (m MensajeAtestacionActoFuenteAutoridadV1) HuellaSHA256() (string, error) {
	bytesCanonicos, err := m.BytesCanonicos()
	if err != nil {
		return "", err
	}
	return huellaBytesFuenteAutoridad(bytesCanonicos), nil
}

func mensajeAtestacionActoFuenteAutoridad(
	compromiso CompromisoTransicionFuenteAutoridadV1,
	evidencia EvidenciaActoFuenteAutoridad,
) MensajeAtestacionActoFuenteAutoridadV1 {
	return MensajeAtestacionActoFuenteAutoridadV1{
		Esquema: esquemaMensajeAtestacionActoFuenteAutoridadV1, Compromiso: compromiso,
		EvidenciaRef: evidencia.EvidenciaRef, ActoRef: evidencia.ActoRef,
		DocumentoRef: evidencia.DocumentoRef, RepresentacionRef: evidencia.RepresentacionRef,
		HuellaDocumentoSHA256: evidencia.HuellaDocumentoSHA256, OrganoRef: evidencia.OrganoRef,
		FirmasRefs: append([]string(nil), evidencia.FirmasRefs...), ComprobadorRef: evidencia.ComprobadorRef,
		ActoOcurridoEn: evidencia.ActoOcurridoEn, ComprobadaEn: evidencia.ComprobadaEn,
	}
}

type TransicionFuenteAutoridad struct {
	Secuencia                    uint64                       `json:"secuencia"`
	EstadoAnterior               EstadoFuenteAutoridad        `json:"estado_anterior"`
	EstadoNuevo                  EstadoFuenteAutoridad        `json:"estado_nuevo"`
	ActorRef                     string                       `json:"actor_ref"`
	MotivoCodigo                 CodigoMotivoFuenteAutoridad  `json:"motivo_codigo"`
	SolicitudRef                 string                       `json:"solicitud_ref"`
	PreparadaEn                  time.Time                    `json:"preparada_en"`
	ExpiraEn                     time.Time                    `json:"expira_en"`
	RegistradaEn                 time.Time                    `json:"registrada_en"`
	Evidencia                    EvidenciaActoFuenteAutoridad `json:"evidencia"`
	HuellaHistoriaAnteriorSHA256 string                       `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256    string                       `json:"huella_historia_nueva_sha256"`
}

// EdicionBorradorFuenteAutoridad conserva todos los actores que alteraron el
// borrador y encadena la huella anterior con la nueva.
type EdicionBorradorFuenteAutoridad struct {
	RevisionAnterior              uint64                      `json:"revision_anterior"`
	RevisionNueva                 uint64                      `json:"revision_nueva"`
	ActorRef                      string                      `json:"actor_ref"`
	MotivoCodigo                  CodigoMotivoFuenteAutoridad `json:"motivo_codigo"`
	RegistradaEn                  time.Time                   `json:"registrada_en"`
	HuellaContenidoAnteriorSHA256 string                      `json:"huella_contenido_anterior_sha256"`
	HuellaContenidoNuevaSHA256    string                      `json:"huella_contenido_nueva_sha256"`
	HuellaHistoriaAnteriorSHA256  string                      `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256     string                      `json:"huella_historia_nueva_sha256"`
}

// FuenteAutoridadVersionada registra autoridad documental, no una regla de
// negocio. Las revisiones anteriores viven en el repositorio append-only.
type FuenteAutoridadVersionada struct {
	ID                           string                           `json:"id"`
	Version                      uint64                           `json:"version"`
	Revision                     uint64                           `json:"revision"`
	VersionAnterior              ReferenciaLinajeFuenteAutoridad  `json:"version_anterior,omitempty"`
	Contenido                    ContenidoFuenteAutoridad         `json:"contenido"`
	HuellaContenidoInicialSHA256 string                           `json:"huella_contenido_inicial_sha256"`
	HuellaHistoriaInicialSHA256  string                           `json:"huella_historia_inicial_sha256"`
	Estado                       EstadoFuenteAutoridad            `json:"estado"`
	CreadaPor                    string                           `json:"creada_por"`
	CreadaEn                     time.Time                        `json:"creada_en"`
	MotivoCreacionCodigo         CodigoMotivoFuenteAutoridad      `json:"motivo_creacion_codigo"`
	EdicionesBorrador            []EdicionBorradorFuenteAutoridad `json:"ediciones_borrador,omitempty"`
	Transiciones                 []TransicionFuenteAutoridad      `json:"transiciones,omitempty"`
}

type DatosAltaFuenteAutoridadV1 struct {
	ID                   string
	Contenido            ContenidoFuenteAutoridad
	CreadaPor            string
	CreadaEn             time.Time
	MotivoCreacionCodigo CodigoMotivoFuenteAutoridad
}

func NuevaFuenteAutoridadBorradorV1(datos DatosAltaFuenteAutoridadV1) (FuenteAutoridadVersionada, error) {
	return nuevaFuenteAutoridadBorradorVersionada(
		datos.ID, 1, ReferenciaLinajeFuenteAutoridad{}, datos.Contenido, datos.CreadaPor,
		datos.MotivoCreacionCodigo, datos.CreadaEn,
	)
}

func nuevaFuenteAutoridadBorradorVersionada(
	id string,
	version uint64,
	versionAnterior ReferenciaLinajeFuenteAutoridad,
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	registradaEn time.Time,
) (FuenteAutoridadVersionada, error) {
	registradaEn = normalizarInstanteFuenteAutoridad(registradaEn)
	contenido, err := normalizarContenidoFuenteAutoridad(contenido)
	if err != nil || !referenciaPersonaFuenteAutoridadValida(actorRef) || !motivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(registradaEn) || contenido.ConocidaEn.After(registradaEn) {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	contenido, huellaInicial, err := prepararHuellaContenidoFuenteAutoridad(
		id, version, versionAnterior, contenido,
	)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	huellaHistoriaInicial, err := huellaHistoriaInicialFuenteAutoridad(
		id, version, versionAnterior, huellaInicial, actorRef, motivoCodigo, registradaEn,
	)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	fuente := FuenteAutoridadVersionada{
		ID: id, Version: version, Revision: 1, VersionAnterior: versionAnterior,
		Contenido: contenido, HuellaContenidoInicialSHA256: huellaInicial,
		HuellaHistoriaInicialSHA256: huellaHistoriaInicial,
		Estado:                      EstadoFuenteAutoridadBorrador, CreadaPor: actorRef, CreadaEn: registradaEn,
		MotivoCreacionCodigo: motivoCodigo,
	}
	return fuente.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) ReferenciaExacta() (ReferenciaFuenteAutoridad, error) {
	_, huella, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return ReferenciaFuenteAutoridad{}, err
	}
	referencia := ReferenciaFuenteAutoridad{FuenteID: f.ID, Version: f.Version, HuellaContenidoSHA256: huella}
	if err := referencia.Validar(); err != nil {
		return ReferenciaFuenteAutoridad{}, err
	}
	return referencia, nil
}

func (f FuenteAutoridadVersionada) ReferenciaLinajeExacta() (ReferenciaLinajeFuenteAutoridad, error) {
	canonica, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil || canonica.Estado == EstadoFuenteAutoridadBorrador {
		return ReferenciaLinajeFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
	}
	bytesEstado, err := serializarEstadoPersistibleFuenteAutoridadV1(canonica)
	if err != nil {
		return ReferenciaLinajeFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
	}
	referencia := ReferenciaLinajeFuenteAutoridad{
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: canonica.ID, Version: canonica.Version, HuellaContenidoSHA256: huellaContenido,
		},
		Revision: canonica.Revision, Estado: canonica.Estado,
		HuellaHistoriaSHA256: canonica.huellaHistoriaActual(),
		HuellaEstadoSHA256:   huellaBytesFuenteAutoridad(bytesEstado),
	}
	if err := referencia.Validar(); err != nil {
		return ReferenciaLinajeFuenteAutoridad{}, err
	}
	return referencia, nil
}

// Citar solo expone preceptos que existen en una version ya publicada. Una
// suspension o derogacion no borra la cita historica; un borrador nunca es
// fuente citable.
func (f FuenteAutoridadVersionada) Citar(preceptos ...string) (CitaFuenteAutoridad, error) {
	canonico, huella, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil || canonico.Estado == EstadoFuenteAutoridadBorrador {
		return CitaFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
	}
	disponibles := make(map[string]struct{}, len(canonico.Contenido.Preceptos))
	for _, precepto := range canonico.Contenido.Preceptos {
		disponibles[precepto.Clave] = struct{}{}
	}
	for _, precepto := range preceptos {
		if _, existe := disponibles[precepto]; !existe {
			return CitaFuenteAutoridad{}, ErrReferenciaAutoridadInvalida
		}
	}
	cita := CitaFuenteAutoridad{
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: canonico.ID, Version: canonico.Version, HuellaContenidoSHA256: huella,
		},
		Preceptos: append([]string(nil), preceptos...),
	}
	return cita.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) Validar() error {
	_, _, err := f.prepararCanonicaSinEstadoPersistible()
	return err
}

func (f FuenteAutoridadVersionada) ClonarCanonica() (FuenteAutoridadVersionada, error) {
	clon, _, err := f.prepararCanonicaSinEstadoPersistible()
	return clon, err
}

func (f FuenteAutoridadVersionada) HuellaContenidoSHA256() (string, error) {
	_, huella, err := f.prepararCanonicaSinEstadoPersistible()
	return huella, err
}

func (f FuenteAutoridadVersionada) HuellaEstadoSHA256() (string, error) {
	_, _, estadoCanonico, err := f.prepararCanonica()
	if err != nil {
		return "", err
	}
	return huellaBytesFuenteAutoridad(estadoCanonico), nil
}

// MarshalJSON impide persistir por accidente la estructura viva. Todo JSON
// del agregado cruza el contrato congelado EstadoPersistibleV1.
func (f FuenteAutoridadVersionada) MarshalJSON() ([]byte, error) {
	return f.EstadoPersistibleV1()
}

// UnmarshalJSON impide que un adaptador eluda por accidente la rehidratacion
// estricta V1 mediante encoding/json. Solo se acepta el estado canonico exacto.
func (f *FuenteAutoridadVersionada) UnmarshalJSON(datos []byte) error {
	if f == nil {
		return ErrEstadoPersistibleFuenteAutoridadInvalido
	}
	rehidratada, err := RehidratarFuenteAutoridadV1(datos)
	if err != nil {
		return err
	}
	*f = rehidratada
	return nil
}

func (f FuenteAutoridadVersionada) prepararCanonica() (FuenteAutoridadVersionada, string, []byte, error) {
	return f.prepararCanonicaSegun(true)
}

func (f FuenteAutoridadVersionada) prepararCanonicaSinEstadoPersistible() (
	FuenteAutoridadVersionada,
	string,
	error,
) {
	canonica, huellaContenido, _, err := f.prepararCanonicaSegun(false)
	return canonica, huellaContenido, err
}

func (f FuenteAutoridadVersionada) prepararCanonicaSegun(
	incluirEstadoPersistible bool,
) (FuenteAutoridadVersionada, string, []byte, error) {
	contenido, huellaContenido, err := prepararHuellaContenidoFuenteAutoridad(
		f.ID, f.Version, f.VersionAnterior, f.Contenido,
	)
	huellaHistoriaInicial, errHistoria := huellaHistoriaInicialFuenteAutoridad(
		f.ID, f.Version, f.VersionAnterior, f.HuellaContenidoInicialSHA256,
		f.CreadaPor, f.MotivoCreacionCodigo, f.CreadaEn,
	)
	if err != nil || f.Revision == 0 || !f.Estado.Valido() ||
		!esSHA256Autoridad(f.HuellaContenidoInicialSHA256) ||
		errHistoria != nil || f.HuellaHistoriaInicialSHA256 != huellaHistoriaInicial ||
		!referenciaPersonaFuenteAutoridadValida(f.CreadaPor) ||
		!instanteFuenteAutoridadCanonico(f.CreadaEn) || f.Contenido.ConocidaEn.After(f.CreadaEn) ||
		!f.MotivoCreacionCodigo.Valido() || len(f.EdicionesBorrador) > maximoEdicionesFuenteAutoridad ||
		len(f.Transiciones) > maximoTransicionesFuenteAutoridad ||
		f.Revision != uint64(1+len(f.EdicionesBorrador)+len(f.Transiciones)) {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}

	clon := f
	clon.Contenido = contenido
	clon.EdicionesBorrador = append([]EdicionBorradorFuenteAutoridad(nil), f.EdicionesBorrador...)
	clon.Transiciones = make([]TransicionFuenteAutoridad, len(f.Transiciones))

	huellaEncadenada := f.HuellaContenidoInicialSHA256
	huellaHistoria := f.HuellaHistoriaInicialSHA256
	ultimoRegistro := f.CreadaEn
	editores := make(map[string]struct{}, len(f.EdicionesBorrador))
	for indice, edicion := range f.EdicionesBorrador {
		revisionAnterior := uint64(indice + 1)
		if edicion.RevisionAnterior != revisionAnterior || edicion.RevisionNueva != revisionAnterior+1 ||
			!referenciaPersonaFuenteAutoridadValida(edicion.ActorRef) || !edicion.MotivoCodigo.Valido() ||
			!instanteFuenteAutoridadCanonico(edicion.RegistradaEn) || !edicion.RegistradaEn.After(ultimoRegistro) ||
			!esSHA256Autoridad(edicion.HuellaContenidoAnteriorSHA256) ||
			!esSHA256Autoridad(edicion.HuellaContenidoNuevaSHA256) ||
			!esSHA256Autoridad(edicion.HuellaHistoriaAnteriorSHA256) ||
			!esSHA256Autoridad(edicion.HuellaHistoriaNuevaSHA256) ||
			edicion.HuellaContenidoAnteriorSHA256 != huellaEncadenada ||
			edicion.HuellaHistoriaAnteriorSHA256 != huellaHistoria {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		huellaHistoriaNueva, err := huellaHistoriaEdicionBorradorFuenteAutoridad(edicion)
		if err != nil || edicion.HuellaHistoriaNuevaSHA256 != huellaHistoriaNueva {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		huellaEncadenada = edicion.HuellaContenidoNuevaSHA256
		huellaHistoria = huellaHistoriaNueva
		ultimoRegistro = edicion.RegistradaEn
		editores[edicion.ActorRef] = struct{}{}
	}
	if huellaEncadenada != huellaContenido {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}

	if (f.Estado == EstadoFuenteAutoridadBorrador) != (len(f.Transiciones) == 0) {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}
	estado := EstadoFuenteAutoridadBorrador
	referenciasSolicitud := make(map[string]struct{}, len(f.Transiciones))
	referenciasEvidencia := make(map[string]struct{}, len(f.Transiciones))
	referenciasAtestacion := make(map[string]struct{}, len(f.Transiciones))
	for indice, transicion := range f.Transiciones {
		secuencia := uint64(indice + 1)
		if transicion.Secuencia != secuencia || transicion.EstadoAnterior != estado ||
			!transicionPermitidaFuenteAutoridad(transicion.EstadoAnterior, transicion.EstadoNuevo) ||
			!referenciaPersonaFuenteAutoridadValida(transicion.ActorRef) || !transicion.MotivoCodigo.Valido() ||
			!referenciaFuenteAutoridadValida(transicion.SolicitudRef) ||
			!instanteFuenteAutoridadCanonico(transicion.PreparadaEn) || !transicion.PreparadaEn.After(ultimoRegistro) ||
			!instanteFuenteAutoridadCanonico(transicion.ExpiraEn) || !transicion.ExpiraEn.After(transicion.PreparadaEn) ||
			!instanteFuenteAutoridadCanonico(transicion.RegistradaEn) || !transicion.RegistradaEn.After(ultimoRegistro) ||
			transicion.RegistradaEn.Before(transicion.Evidencia.ComprobadaEn) ||
			!transicion.RegistradaEn.Before(transicion.ExpiraEn) ||
			!esSHA256Autoridad(transicion.HuellaHistoriaAnteriorSHA256) ||
			!esSHA256Autoridad(transicion.HuellaHistoriaNuevaSHA256) ||
			transicion.HuellaHistoriaAnteriorSHA256 != huellaHistoria {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		if indice == 0 {
			if transicion.ActorRef == f.CreadaPor {
				return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
			}
			if _, edito := editores[transicion.ActorRef]; edito {
				return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
			}
		}
		compromiso, err := construirCompromisoTransicionFuenteAutoridad(
			f.ID, f.Version, huellaContenido,
			uint64(1+len(f.EdicionesBorrador)+indice), secuencia,
			transicion.EstadoAnterior, transicion.EstadoNuevo, transicion.ActorRef,
			transicion.MotivoCodigo, huellaHistoria, transicion.SolicitudRef,
			transicion.PreparadaEn, transicion.ExpiraEn,
		)
		if err != nil || validarEvidenciaTransicionFuenteAutoridad(transicion.Evidencia, compromiso) != nil ||
			!registrarReferenciaFuenteAutoridadUnica(referenciasSolicitud, transicion.SolicitudRef) ||
			!registrarReferenciasEvidenciaFuenteAutoridad(
				transicion.Evidencia, referenciasEvidencia, referenciasAtestacion,
			) {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		huellaHistoriaNueva, err := huellaHistoriaTransicionFuenteAutoridad(transicion, compromiso)
		if err != nil || transicion.HuellaHistoriaNuevaSHA256 != huellaHistoriaNueva {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		evidencia, err := transicion.Evidencia.ClonarCanonica()
		if err != nil {
			return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
		}
		clon.Transiciones[indice] = transicion
		clon.Transiciones[indice].Evidencia = evidencia
		estado = transicion.EstadoNuevo
		huellaHistoria = huellaHistoriaNueva
		ultimoRegistro = transicion.RegistradaEn
	}
	if estado != f.Estado {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}

	if !incluirEstadoPersistible {
		return clon, huellaContenido, nil, nil
	}
	estadoCanonico, err := serializarEstadoPersistibleFuenteAutoridadV1(clon)
	if err != nil {
		return FuenteAutoridadVersionada{}, "", nil, ErrFuenteAutoridadInvalida
	}
	return clon, huellaContenido, estadoCanonico, nil
}

func registrarReferenciasEvidenciaFuenteAutoridad(
	evidencia EvidenciaActoFuenteAutoridad,
	evidencias, atestaciones map[string]struct{},
) bool {
	if !registrarReferenciaFuenteAutoridadUnica(evidencias, evidencia.EvidenciaRef) ||
		!registrarReferenciaFuenteAutoridadUnica(atestaciones, evidencia.AtestacionRef) {
		return false
	}
	return true
}

func registrarReferenciaFuenteAutoridadUnica(vistas map[string]struct{}, referencia string) bool {
	if _, repetida := vistas[referencia]; repetida {
		return false
	}
	vistas[referencia] = struct{}{}
	return true
}

func (f FuenteAutoridadVersionada) ActualizarBorrador(
	revisionEsperada uint64,
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	instante time.Time,
) (FuenteAutoridadVersionada, error) {
	canonico, huellaAnterior, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	if canonico.Estado != EstadoFuenteAutoridadBorrador {
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	if revisionEsperada != canonico.Revision {
		return FuenteAutoridadVersionada{}, ErrRevisionAutoridadEnConflicto
	}
	if len(canonico.EdicionesBorrador) >= maximoEdicionesFuenteAutoridad {
		return FuenteAutoridadVersionada{}, ErrLimiteAutoridadAlcanzado
	}
	contenido, err = normalizarContenidoFuenteAutoridad(contenido)
	instante = normalizarInstanteFuenteAutoridad(instante)
	if err != nil || !referenciaPersonaFuenteAutoridadValida(actorRef) || !motivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(instante) || !instante.After(canonico.ultimaMutacionEn()) ||
		contenido.ConocidaEn.After(instante) {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	contenido, huellaNueva, err := prepararHuellaContenidoFuenteAutoridad(
		canonico.ID, canonico.Version, canonico.VersionAnterior, contenido,
	)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	if huellaNueva == huellaAnterior {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	edicion := EdicionBorradorFuenteAutoridad{
		RevisionAnterior: canonico.Revision, RevisionNueva: canonico.Revision + 1,
		ActorRef: actorRef, MotivoCodigo: motivoCodigo, RegistradaEn: instante,
		HuellaContenidoAnteriorSHA256: huellaAnterior, HuellaContenidoNuevaSHA256: huellaNueva,
		HuellaHistoriaAnteriorSHA256: canonico.huellaHistoriaActual(),
	}
	huellaHistoriaNueva, err := huellaHistoriaEdicionBorradorFuenteAutoridad(edicion)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	edicion.HuellaHistoriaNuevaSHA256 = huellaHistoriaNueva
	canonico.EdicionesBorrador = append(canonico.EdicionesBorrador, edicion)
	canonico.Contenido = contenido
	canonico.Revision++
	return canonico.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) NuevaVersionV1(
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	instante time.Time,
) (FuenteAutoridadVersionada, error) {
	canonico, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	instante = normalizarInstanteFuenteAutoridad(instante)
	if canonico.Estado == EstadoFuenteAutoridadBorrador || canonico.Version == ^uint64(0) ||
		!referenciaPersonaFuenteAutoridadValida(actorRef) ||
		!motivoCodigo.Valido() || !instanteFuenteAutoridadCanonico(instante) ||
		!instante.After(canonico.ultimaMutacionEn()) {
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	contenidoNormalizado, err := normalizarContenidoFuenteAutoridad(contenido)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	_, bytesContenidoActual, err := prepararContenidoFuenteAutoridad(canonico.Contenido)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	_, bytesContenidoNuevo, err := prepararContenidoFuenteAutoridad(contenidoNormalizado)
	if err != nil || bytes.Equal(bytesContenidoActual, bytesContenidoNuevo) {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	bytesEstado, err := serializarEstadoPersistibleFuenteAutoridadV1(canonico)
	if err != nil {
		return FuenteAutoridadVersionada{}, ErrFuenteAutoridadInvalida
	}
	linaje := ReferenciaLinajeFuenteAutoridad{
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: canonico.ID, Version: canonico.Version, HuellaContenidoSHA256: huellaContenido,
		},
		Revision: canonico.Revision, Estado: canonico.Estado,
		HuellaHistoriaSHA256: canonico.huellaHistoriaActual(),
		HuellaEstadoSHA256:   huellaBytesFuenteAutoridad(bytesEstado),
	}
	return nuevaFuenteAutoridadBorradorVersionada(
		canonico.ID, canonico.Version+1, linaje, contenidoNormalizado, actorRef, motivoCodigo, instante,
	)
}

// PrepararSolicitudTransicionV1 devuelve una solicitud y sus bytes canónicos
// firmables. El adaptador no construye JSON ni repite los parámetros al
// aplicar el acto.
func (f FuenteAutoridadVersionada) PrepararSolicitudTransicionV1(
	datos DatosPreparacionTransicionFuenteAutoridadV1,
) (SolicitudTransicionFuenteAutoridadV1, error) {
	canonico, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, err
	}
	datos.PreparadaEn = normalizarInstanteFuenteAutoridad(datos.PreparadaEn)
	datos.ExpiraEn = normalizarInstanteFuenteAutoridad(datos.ExpiraEn)
	compromiso, err := canonico.prepararCompromisoCanonico(
		huellaContenido, datos,
	)
	if err != nil {
		return SolicitudTransicionFuenteAutoridadV1{}, err
	}
	return nuevaSolicitudTransicionFuenteAutoridadV1(compromiso)
}

func (f FuenteAutoridadVersionada) AplicarTransicionV1(
	solicitud SolicitudTransicionFuenteAutoridadV1,
	evidencia EvidenciaActoFuenteAutoridad,
	registradaEn time.Time,
) (FuenteAutoridadVersionada, error) {
	canonico, huellaContenido, err := f.prepararCanonicaSinEstadoPersistible()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	compromiso, err := solicitud.Compromiso()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	esperado, err := canonico.prepararCompromisoCanonico(huellaContenido, DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: compromiso.EstadoNuevo, ActorRef: compromiso.ActorRef,
		MotivoCodigo: compromiso.MotivoCodigo, SolicitudRef: compromiso.SolicitudRef,
		PreparadaEn: compromiso.PreparadaEn, ExpiraEn: compromiso.ExpiraEn,
	})
	if err != nil || compromiso != esperado {
		return FuenteAutoridadVersionada{}, ErrSolicitudAutoridadObsoleta
	}
	bytesEsperados, err := esperado.BytesCanonicos()
	bytesSolicitud, errSolicitud := solicitud.BytesCanonicos()
	if err != nil || errSolicitud != nil || !bytes.Equal(bytesEsperados, bytesSolicitud) {
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	evidenciaCanonica, err := evidencia.ClonarCanonica()
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	if validarEvidenciaTransicionFuenteAutoridad(evidenciaCanonica, compromiso) != nil {
		return FuenteAutoridadVersionada{}, ErrEvidenciaActoAutoridadInvalida
	}
	registradaEn = normalizarInstanteFuenteAutoridad(registradaEn)
	if !instanteFuenteAutoridadCanonico(registradaEn) || registradaEn.Before(evidenciaCanonica.ComprobadaEn) ||
		!registradaEn.Before(compromiso.ExpiraEn) || !registradaEn.After(canonico.ultimaMutacionEn()) {
		if !registradaEn.Before(compromiso.ExpiraEn) {
			return FuenteAutoridadVersionada{}, ErrSolicitudAutoridadExpirada
		}
		return FuenteAutoridadVersionada{}, ErrTransicionAutoridadInvalida
	}
	transicion := TransicionFuenteAutoridad{
		Secuencia: uint64(len(canonico.Transiciones) + 1), EstadoAnterior: canonico.Estado,
		EstadoNuevo: compromiso.EstadoNuevo, ActorRef: compromiso.ActorRef,
		MotivoCodigo: compromiso.MotivoCodigo, SolicitudRef: compromiso.SolicitudRef,
		PreparadaEn: compromiso.PreparadaEn, ExpiraEn: compromiso.ExpiraEn, RegistradaEn: registradaEn,
		Evidencia: evidenciaCanonica, HuellaHistoriaAnteriorSHA256: canonico.huellaHistoriaActual(),
	}
	huellaHistoriaNueva, err := huellaHistoriaTransicionFuenteAutoridad(transicion, compromiso)
	if err != nil {
		return FuenteAutoridadVersionada{}, err
	}
	transicion.HuellaHistoriaNuevaSHA256 = huellaHistoriaNueva
	canonico.Estado = compromiso.EstadoNuevo
	canonico.Revision++
	canonico.Transiciones = append(canonico.Transiciones, transicion)
	return canonico.ClonarCanonica()
}

func (f FuenteAutoridadVersionada) prepararCompromisoCanonico(
	huellaContenido string,
	datos DatosPreparacionTransicionFuenteAutoridadV1,
) (CompromisoTransicionFuenteAutoridadV1, error) {
	if len(f.Transiciones) >= maximoTransicionesFuenteAutoridad ||
		!transicionPermitidaFuenteAutoridad(f.Estado, datos.EstadoNuevo) ||
		!referenciaPersonaFuenteAutoridadValida(datos.ActorRef) || !datos.MotivoCodigo.Valido() ||
		!referenciaFuenteAutoridadValida(datos.SolicitudRef) ||
		!instanteFuenteAutoridadCanonico(datos.PreparadaEn) || !datos.PreparadaEn.After(f.ultimaMutacionEn()) ||
		!instanteFuenteAutoridadCanonico(datos.ExpiraEn) || !datos.ExpiraEn.After(datos.PreparadaEn) {
		return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	for _, transicion := range f.Transiciones {
		if transicion.SolicitudRef == datos.SolicitudRef {
			return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
		}
	}
	if f.Estado == EstadoFuenteAutoridadBorrador {
		if datos.ActorRef == f.CreadaPor || f.fueEditorBorrador(datos.ActorRef) {
			return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
		}
	}
	return construirCompromisoTransicionFuenteAutoridad(
		f.ID, f.Version, huellaContenido, f.Revision, uint64(len(f.Transiciones)+1),
		f.Estado, datos.EstadoNuevo, datos.ActorRef, datos.MotivoCodigo, f.huellaHistoriaActual(),
		datos.SolicitudRef, datos.PreparadaEn, datos.ExpiraEn,
	)
}

func construirCompromisoTransicionFuenteAutoridad(
	ID string,
	version uint64,
	huellaContenido string,
	revisionPrevia, secuencia uint64,
	estadoAnterior, estadoNuevo EstadoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	huellaHistoriaPrevia string,
	solicitudRef string,
	preparadaEn, expiraEn time.Time,
) (CompromisoTransicionFuenteAutoridadV1, error) {
	accion, valida := accionActoParaTransicionFuenteAutoridad(estadoAnterior, estadoNuevo)
	if !valida {
		return CompromisoTransicionFuenteAutoridadV1{}, ErrTransicionAutoridadInvalida
	}
	compromiso := CompromisoTransicionFuenteAutoridadV1{
		Esquema: esquemaCompromisoTransicionFuenteAutoridadV1, SolicitudRef: solicitudRef,
		Fuente: ReferenciaFuenteAutoridad{
			FuenteID: ID, Version: version, HuellaContenidoSHA256: huellaContenido,
		},
		RevisionPrevia: revisionPrevia, Secuencia: secuencia,
		EstadoAnterior: estadoAnterior, EstadoNuevo: estadoNuevo, Accion: accion,
		ActorRef: actorRef, MotivoCodigo: motivoCodigo,
		HuellaHistoriaPreviaSHA256: huellaHistoriaPrevia, PreparadaEn: preparadaEn, ExpiraEn: expiraEn,
	}
	if err := compromiso.Validar(); err != nil {
		return CompromisoTransicionFuenteAutoridadV1{}, err
	}
	return compromiso, nil
}

func validarEvidenciaTransicionFuenteAutoridad(
	evidencia EvidenciaActoFuenteAutoridad,
	compromiso CompromisoTransicionFuenteAutoridadV1,
) error {
	huellaCompromiso, err := compromiso.HuellaSHA256()
	huellaMensaje, errMensaje := mensajeAtestacionActoFuenteAutoridad(compromiso, evidencia).HuellaSHA256()
	if err != nil || evidencia.Validar() != nil || evidencia.Accion != compromiso.Accion ||
		evidencia.FuenteID != compromiso.Fuente.FuenteID || evidencia.FuenteVersion != compromiso.Fuente.Version ||
		evidencia.HuellaContenidoSHA256 != compromiso.Fuente.HuellaContenidoSHA256 ||
		evidencia.HuellaCompromisoSHA256 != huellaCompromiso || errMensaje != nil ||
		evidencia.HuellaMensajeAtestadoSHA256 != huellaMensaje ||
		evidencia.ComprobadaEn.Before(compromiso.PreparadaEn) ||
		!evidencia.ComprobadaEn.Before(compromiso.ExpiraEn) {
		return ErrEvidenciaActoAutoridadInvalida
	}
	return nil
}

func (f FuenteAutoridadVersionada) ultimaMutacionEn() time.Time {
	if len(f.Transiciones) != 0 {
		return f.Transiciones[len(f.Transiciones)-1].RegistradaEn
	}
	if len(f.EdicionesBorrador) != 0 {
		return f.EdicionesBorrador[len(f.EdicionesBorrador)-1].RegistradaEn
	}
	return f.CreadaEn
}

func (f FuenteAutoridadVersionada) huellaHistoriaActual() string {
	if len(f.Transiciones) != 0 {
		return f.Transiciones[len(f.Transiciones)-1].HuellaHistoriaNuevaSHA256
	}
	if len(f.EdicionesBorrador) != 0 {
		return f.EdicionesBorrador[len(f.EdicionesBorrador)-1].HuellaHistoriaNuevaSHA256
	}
	return f.HuellaHistoriaInicialSHA256
}

func (f FuenteAutoridadVersionada) fueEditorBorrador(actorRef string) bool {
	for _, edicion := range f.EdicionesBorrador {
		if edicion.ActorRef == actorRef {
			return true
		}
	}
	return false
}

func huellaHistoriaInicialFuenteAutoridad(
	id string,
	version uint64,
	versionAnterior ReferenciaLinajeFuenteAutoridad,
	huellaContenidoInicial, actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	registradaEn time.Time,
) (string, error) {
	if !esClaveDocumentalCanonica(id) || version < 1 || !esSHA256Autoridad(huellaContenidoInicial) ||
		!referenciaPersonaFuenteAutoridadValida(actorRef) || !motivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(registradaEn) {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaValorFuenteAutoridad(historiaInicialPersistibleAutoridadV1{
		Esquema: "vec.fuente_autoridad.historia_inicial.v1", ID: id, Version: uint64(version),
		VersionAnteriorFuenteID:       versionAnterior.Fuente.FuenteID,
		VersionAnteriorNumero:         versionAnterior.Fuente.Version,
		VersionAnteriorHuellaSHA256:   versionAnterior.Fuente.HuellaContenidoSHA256,
		VersionAnteriorRevision:       versionAnterior.Revision,
		VersionAnteriorEstado:         string(versionAnterior.Estado),
		VersionAnteriorHistoriaSHA256: versionAnterior.HuellaHistoriaSHA256,
		VersionAnteriorEstadoSHA256:   versionAnterior.HuellaEstadoSHA256,
		HuellaContenidoInicialSHA256:  huellaContenidoInicial, ActorRef: actorRef,
		MotivoCodigo: string(motivoCodigo), RegistradaEn: textoInstantePersistibleAutoridadV1(registradaEn),
	}, maximoBytesSobreContenido)
}

func huellaHistoriaEdicionBorradorFuenteAutoridad(edicion EdicionBorradorFuenteAutoridad) (string, error) {
	if edicion.RevisionAnterior == 0 || edicion.RevisionNueva != edicion.RevisionAnterior+1 ||
		!referenciaPersonaFuenteAutoridadValida(edicion.ActorRef) || !edicion.MotivoCodigo.Valido() ||
		!instanteFuenteAutoridadCanonico(edicion.RegistradaEn) ||
		!esSHA256Autoridad(edicion.HuellaContenidoAnteriorSHA256) ||
		!esSHA256Autoridad(edicion.HuellaContenidoNuevaSHA256) ||
		!esSHA256Autoridad(edicion.HuellaHistoriaAnteriorSHA256) {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaValorFuenteAutoridad(historiaEdicionPersistibleAutoridadV1{
		Esquema:          "vec.fuente_autoridad.historia_edicion.v1",
		RevisionAnterior: edicion.RevisionAnterior, RevisionNueva: edicion.RevisionNueva,
		ActorRef: edicion.ActorRef, MotivoCodigo: string(edicion.MotivoCodigo),
		RegistradaEn:                  textoInstantePersistibleAutoridadV1(edicion.RegistradaEn),
		HuellaContenidoAnteriorSHA256: edicion.HuellaContenidoAnteriorSHA256,
		HuellaContenidoNuevaSHA256:    edicion.HuellaContenidoNuevaSHA256,
		HuellaHistoriaAnteriorSHA256:  edicion.HuellaHistoriaAnteriorSHA256,
	}, maximoBytesSobreContenido)
}

func huellaHistoriaTransicionFuenteAutoridad(
	transicion TransicionFuenteAutoridad,
	compromiso CompromisoTransicionFuenteAutoridadV1,
) (string, error) {
	huellaCompromiso, err := compromiso.HuellaSHA256()
	if err != nil || transicion.HuellaHistoriaAnteriorSHA256 != compromiso.HuellaHistoriaPreviaSHA256 ||
		transicion.Evidencia.HuellaCompromisoSHA256 != huellaCompromiso {
		return "", ErrFuenteAutoridadInvalida
	}
	return huellaValorFuenteAutoridad(historiaTransicionPersistibleAutoridadV1{
		Esquema:                      "vec.fuente_autoridad.historia_transicion.v1",
		HuellaHistoriaAnteriorSHA256: transicion.HuellaHistoriaAnteriorSHA256,
		HuellaCompromisoSHA256:       huellaCompromiso, EvidenciaRef: transicion.Evidencia.EvidenciaRef,
		HuellaMensajeAtestadoSHA256: transicion.Evidencia.HuellaMensajeAtestadoSHA256,
		AtestacionRef:               transicion.Evidencia.AtestacionRef,
		HuellaAtestacionSHA256:      transicion.Evidencia.HuellaAtestacionSHA256,
		FirmaAtestacionRef:          transicion.Evidencia.FirmaAtestacionRef,
		RegistradaEn:                textoInstantePersistibleAutoridadV1(transicion.RegistradaEn),
	}, maximoBytesSobreContenido)
}

func transicionPermitidaFuenteAutoridad(anterior, nueva EstadoFuenteAutoridad) bool {
	switch anterior {
	case EstadoFuenteAutoridadBorrador:
		return nueva == EstadoFuenteAutoridadPublicada
	case EstadoFuenteAutoridadPublicada:
		return nueva == EstadoFuenteAutoridadSuspendida || nueva == EstadoFuenteAutoridadDerogada
	case EstadoFuenteAutoridadSuspendida:
		return nueva == EstadoFuenteAutoridadPublicada || nueva == EstadoFuenteAutoridadDerogada
	default:
		return false
	}
}

func accionActoParaTransicionFuenteAutoridad(anterior, nueva EstadoFuenteAutoridad) (AccionActoFuenteAutoridad, bool) {
	switch {
	case anterior == EstadoFuenteAutoridadBorrador && nueva == EstadoFuenteAutoridadPublicada:
		return AccionActoPublicarFuenteAutoridad, true
	case anterior == EstadoFuenteAutoridadPublicada && nueva == EstadoFuenteAutoridadSuspendida:
		return AccionActoSuspenderFuenteAutoridad, true
	case anterior == EstadoFuenteAutoridadSuspendida && nueva == EstadoFuenteAutoridadPublicada:
		return AccionActoLevantarSuspensionFuenteAutoridad, true
	case (anterior == EstadoFuenteAutoridadPublicada || anterior == EstadoFuenteAutoridadSuspendida) && nueva == EstadoFuenteAutoridadDerogada:
		return AccionActoDerogarFuenteAutoridad, true
	default:
		return "", false
	}
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
