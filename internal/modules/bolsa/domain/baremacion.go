// Package domain contiene las reglas puras del modulo de bolsas.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBaremacionInvalida          = errors.New("bolsa: baremacion de merito invalida")
	ErrCriterioInvalido            = errors.New("bolsa: referencia de criterio invalida")
	ErrReglaCalculoInvalida        = errors.New("bolsa: referencia de regla de calculo invalida")
	ErrCalculoOficialInvalido      = errors.New("bolsa: calculo oficial invalido")
	ErrEvidenciaInvalida           = errors.New("bolsa: referencia de evidencia invalida")
	ErrValoracionEvidenciaInvalida = errors.New("bolsa: valoracion de evidencia invalida")
	ErrContenidoDecisionInvalido   = errors.New("bolsa: contenido de decision tecnica invalido")
	ErrFirmaDecisionInvalida       = errors.New("bolsa: evidencia de firma de decision invalida")
	ErrDecisionTecnicaInvalida     = errors.New("bolsa: decision tecnica invalida")
	ErrHistorialDecisionesInvalido = errors.New("bolsa: historial de decisiones invalido")
	ErrTransicionDecisionInvalida  = errors.New("bolsa: transicion de decision invalida")
	ErrDecisionSinCambios          = errors.New("bolsa: la rectificacion no cambia la valoracion")
)

const (
	// UnidadesPorPunto fija seis decimales sin recurrir nunca a coma flotante.
	// Es una invariante tecnica, no un tipo de merito ni una regla de baremo.
	UnidadesPorPunto Puntos = 1_000_000
	maximoPuntos     Puntos = 9_000_000_000_000_000

	maximoDecisionesPorMerito  = 4096
	maximoEvidenciasPorMerito  = 256
	maximoCaracteresTexto      = 8000
	maximoCaracteresReferencia = 512
)

var (
	patronReferenciaOpaca = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]*$`)
	patronClaveNegocio    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
)

// Puntos almacena micropuntos. Por ejemplo, 2,75 puntos se representan como
// 2_750_000. El dominio no admite float32 ni float64 para evitar redondeos.
type Puntos int64

func (p Puntos) Validos() bool {
	return p >= 0 && p <= maximoPuntos
}

// ReferenciaReglaCalculo identifica la regla ejecutable exacta que produjo la
// puntuacion oficial. La regla es configuracion gobernada, no codigo elegido
// por el cliente ni una constante compilada en este paquete.
type ReferenciaReglaCalculo struct {
	Clave        string `json:"clave"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaReglaCalculo) Validar() error {
	if !claveNegocioValida(r.Clave) || r.Version < 1 || !huellaSHA256Valida(r.HuellaSHA256) {
		return ErrReglaCalculoInvalida
	}
	return nil
}

// ReferenciaCriterio fija la configuracion exacta aplicada y el proceso que la
// publica. Clave y ReglaCalculo son datos gobernados: incorporar un tipo de
// merito o una formula nueva no obliga a recompilar.
type ReferenciaCriterio struct {
	ProcesoRef    string                 `json:"proceso_ref"`
	Clave         string                 `json:"clave"`
	Version       int                    `json:"version"`
	HuellaSHA256  string                 `json:"huella_sha256"`
	PuntosMaximos Puntos                 `json:"puntos_maximos"`
	ReglaCalculo  ReferenciaReglaCalculo `json:"regla_calculo"`
}

func (r ReferenciaCriterio) Validar() error {
	if !referenciaOpacaValida(r.ProcesoRef) || !claveNegocioValida(r.Clave) || r.Version < 1 ||
		!huellaSHA256Valida(r.HuellaSHA256) || !r.PuntosMaximos.Validos() || r.PuntosMaximos == 0 ||
		r.ReglaCalculo.Validar() != nil {
		return ErrCriterioInvalido
	}
	return nil
}

// CalculoOficialBaremacion es el recibo inmutable de un calculador gobernado.
// EntradaRef/HuellaEntrada permiten reproducir exactamente el calculo sin
// confiar en parametros reenviados por HTTP. ResultadoRef/HuellaResultado
// identifican el recibo completo conservado por el conector.
type CalculoOficialBaremacion struct {
	CalculoRef            string                 `json:"calculo_ref"`
	ProcesoRef            string                 `json:"proceso_ref"`
	SolicitudRef          string                 `json:"solicitud_ref"`
	SujetoRef             string                 `json:"sujeto_ref"`
	BaremacionMeritoRef   string                 `json:"baremacion_merito_ref"`
	Criterio              ReferenciaCriterio     `json:"criterio"`
	Regla                 ReferenciaReglaCalculo `json:"regla"`
	Evidencias            []EvidenciaMerito      `json:"evidencias"`
	EntradaRef            string                 `json:"entrada_ref"`
	HuellaEntradaSHA256   string                 `json:"huella_entrada_sha256"`
	PuntosCalculados      Puntos                 `json:"puntos_calculados"`
	DesgloseRef           string                 `json:"desglose_ref"`
	HuellaDesgloseSHA256  string                 `json:"huella_desglose_sha256"`
	ResultadoRef          string                 `json:"resultado_ref"`
	HuellaResultadoSHA256 string                 `json:"huella_resultado_sha256"`
	MotorCalculoRef       string                 `json:"motor_calculo_ref"`
	VersionMotorCalculo   string                 `json:"version_motor_calculo"`
	EvidenciaEjecucionRef string                 `json:"evidencia_ejecucion_ref"`
	HuellaEjecucionSHA256 string                 `json:"huella_ejecucion_sha256"`
	CalculadoEn           time.Time              `json:"calculado_en"`
}

func (c CalculoOficialBaremacion) Validar() error {
	if !referenciaOpacaValida(c.CalculoRef) || !referenciaOpacaValida(c.ProcesoRef) ||
		!referenciaOpacaValida(c.SolicitudRef) || !referenciaOpacaValida(c.SujetoRef) ||
		!referenciaOpacaValida(c.BaremacionMeritoRef) || c.Criterio.Validar() != nil ||
		c.Regla.Validar() != nil || c.Criterio.ProcesoRef != c.ProcesoRef ||
		c.Criterio.ReglaCalculo != c.Regla || !referenciaOpacaValida(c.EntradaRef) ||
		!huellaSHA256Valida(c.HuellaEntradaSHA256) || !c.PuntosCalculados.Validos() ||
		c.PuntosCalculados > c.Criterio.PuntosMaximos || !referenciaOpacaValida(c.DesgloseRef) ||
		!huellaSHA256Valida(c.HuellaDesgloseSHA256) || !referenciaOpacaValida(c.ResultadoRef) ||
		!huellaSHA256Valida(c.HuellaResultadoSHA256) || !referenciaOpacaValida(c.MotorCalculoRef) ||
		!referenciaOpacaValida(c.VersionMotorCalculo) || !referenciaOpacaValida(c.EvidenciaEjecucionRef) ||
		!huellaSHA256Valida(c.HuellaEjecucionSHA256) || c.CalculadoEn.IsZero() {
		return ErrCalculoOficialInvalido
	}
	if evidencias, err := canonizarEvidencias(c.Evidencias); err != nil || len(evidencias) == 0 ||
		len(evidencias) > maximoEvidenciasPorMerito {
		return ErrCalculoOficialInvalido
	}
	return nil
}

func (c CalculoOficialBaremacion) validarPara(procesoRef, solicitudRef, sujetoRef, baremacionRef string, criterio ReferenciaCriterio) error {
	if c.Validar() != nil || c.ProcesoRef != procesoRef || c.SolicitudRef != solicitudRef ||
		c.SujetoRef != sujetoRef || c.BaremacionMeritoRef != baremacionRef || c.Criterio != criterio {
		return ErrCalculoOficialInvalido
	}
	return nil
}

func (c CalculoOficialBaremacion) clonarCanonico() (CalculoOficialBaremacion, error) {
	canonico := c
	canonico.CalculoRef = strings.TrimSpace(c.CalculoRef)
	canonico.ProcesoRef = strings.TrimSpace(c.ProcesoRef)
	canonico.SolicitudRef = strings.TrimSpace(c.SolicitudRef)
	canonico.SujetoRef = strings.TrimSpace(c.SujetoRef)
	canonico.BaremacionMeritoRef = strings.TrimSpace(c.BaremacionMeritoRef)
	canonico.Criterio = canonizarCriterio(c.Criterio)
	canonico.Regla = canonizarReglaCalculo(c.Regla)
	var err error
	canonico.Evidencias, err = canonizarEvidencias(c.Evidencias)
	if err != nil {
		return CalculoOficialBaremacion{}, err
	}
	canonico.EntradaRef = strings.TrimSpace(c.EntradaRef)
	canonico.HuellaEntradaSHA256 = strings.TrimSpace(c.HuellaEntradaSHA256)
	canonico.DesgloseRef = strings.TrimSpace(c.DesgloseRef)
	canonico.HuellaDesgloseSHA256 = strings.TrimSpace(c.HuellaDesgloseSHA256)
	canonico.ResultadoRef = strings.TrimSpace(c.ResultadoRef)
	canonico.HuellaResultadoSHA256 = strings.TrimSpace(c.HuellaResultadoSHA256)
	canonico.MotorCalculoRef = strings.TrimSpace(c.MotorCalculoRef)
	canonico.VersionMotorCalculo = strings.TrimSpace(c.VersionMotorCalculo)
	canonico.EvidenciaEjecucionRef = strings.TrimSpace(c.EvidenciaEjecucionRef)
	canonico.HuellaEjecucionSHA256 = strings.TrimSpace(c.HuellaEjecucionSHA256)
	canonico.CalculadoEn = c.CalculadoEn.UTC()
	if err := canonico.Validar(); err != nil {
		return CalculoOficialBaremacion{}, err
	}
	return canonico, nil
}

// CoincideCon compara recibos canonicos completos, incluida regla, huellas y
// tiempo, sin confundir dos localizaciones horarias del mismo instante.
func (c CalculoOficialBaremacion) CoincideCon(otro CalculoOficialBaremacion) bool {
	izquierda, err := c.clonarCanonico()
	if err != nil {
		return false
	}
	derecha, err := otro.clonarCanonico()
	if err != nil {
		return false
	}
	huellaIzquierda, err := huellaJSON(izquierda)
	if err != nil {
		return false
	}
	huellaDerecha, err := huellaJSON(derecha)
	return err == nil && huellaIzquierda == huellaDerecha
}

func (c CalculoOficialBaremacion) ClonarCanonico() (CalculoOficialBaremacion, error) {
	return c.clonarCanonico()
}

func (c CalculoOficialBaremacion) EvidenciasCanonicas() ([]EvidenciaMerito, error) {
	canonico, err := c.clonarCanonico()
	if err != nil {
		return nil, err
	}
	return canonizarEvidencias(canonico.Evidencias)
}

// ReferenciaEvidencia identifica los bytes exactos del documento evaluado.
// DocumentoRef y RepresentacionRef deben ser referencias internas opacas, sin
// DNI, nombre, correo u otros datos personales embebidos.
type ReferenciaEvidencia struct {
	DocumentoRef      string `json:"documento_ref"`
	VersionDocumento  int    `json:"version_documento"`
	RepresentacionRef string `json:"representacion_ref"`
	HuellaSHA256      string `json:"huella_sha256"`
}

func (r ReferenciaEvidencia) Validar() error {
	if !referenciaOpacaValida(r.DocumentoRef) || r.VersionDocumento < 1 ||
		!referenciaOpacaValida(r.RepresentacionRef) || !huellaSHA256Valida(r.HuellaSHA256) {
		return ErrEvidenciaInvalida
	}
	return nil
}

// EvidenciaMerito incorpora, cuando procede, el vinculo exacto con el
// documento al que subsana. Un merito puede necesitar varias evidencias
// conjuntas y estas no se convierten artificialmente en meritos distintos.
type EvidenciaMerito struct {
	Referencia    ReferenciaEvidencia  `json:"referencia"`
	SubsanacionDe *ReferenciaEvidencia `json:"subsanacion_de,omitempty"`
}

func (e EvidenciaMerito) Validar() error {
	if e.Referencia.Validar() != nil {
		return ErrEvidenciaInvalida
	}
	if e.SubsanacionDe != nil && (e.SubsanacionDe.Validar() != nil ||
		referenciasEvidenciaIguales(e.Referencia, *e.SubsanacionDe)) {
		return ErrEvidenciaInvalida
	}
	return nil
}

type EstadoValoracionEvidencia string

const (
	EstadoEvidenciaApta       EstadoValoracionEvidencia = "apta"
	EstadoEvidenciaNoApta     EstadoValoracionEvidencia = "no_apta"
	EstadoEvidenciaSubsanable EstadoValoracionEvidencia = "subsanable"
)

func (e EstadoValoracionEvidencia) Valido() bool {
	return e == EstadoEvidenciaApta || e == EstadoEvidenciaNoApta || e == EstadoEvidenciaSubsanable
}

type ResultadoSubsanacion string

const (
	ResultadoSubsanacionNoAplica  ResultadoSubsanacion = "no_aplica"
	ResultadoSubsanacionPendiente ResultadoSubsanacion = "pendiente"
	ResultadoSubsanacionAceptada  ResultadoSubsanacion = "aceptada"
	ResultadoSubsanacionRechazada ResultadoSubsanacion = "rechazada"
)

func (r ResultadoSubsanacion) Valido() bool {
	switch r {
	case ResultadoSubsanacionNoAplica, ResultadoSubsanacionPendiente,
		ResultadoSubsanacionAceptada, ResultadoSubsanacionRechazada:
		return true
	default:
		return false
	}
}

// ValoracionEvidencia deja constancia separada del juicio tecnico sobre cada
// documento, aunque el resultado y los puntos se decidan globalmente para el
// merito atomico.
type ValoracionEvidencia struct {
	Evidencia            EvidenciaMerito           `json:"evidencia"`
	Estado               EstadoValoracionEvidencia `json:"estado"`
	ResultadoSubsanacion ResultadoSubsanacion      `json:"resultado_subsanacion"`
	MotivoClave          string                    `json:"motivo_clave"`
	Motivo               string                    `json:"motivo"`
}

func (v ValoracionEvidencia) Validar() error {
	if v.Evidencia.Validar() != nil || !v.Estado.Valido() || !v.ResultadoSubsanacion.Valido() ||
		!claveNegocioValida(v.MotivoClave) || !textoValido(v.Motivo) {
		return ErrValoracionEvidenciaInvalida
	}
	if v.Evidencia.SubsanacionDe == nil {
		if (v.Estado == EstadoEvidenciaSubsanable && v.ResultadoSubsanacion != ResultadoSubsanacionPendiente) ||
			(v.Estado != EstadoEvidenciaSubsanable && v.ResultadoSubsanacion != ResultadoSubsanacionNoAplica) {
			return ErrValoracionEvidenciaInvalida
		}
		return nil
	}
	if (v.ResultadoSubsanacion == ResultadoSubsanacionAceptada && v.Estado != EstadoEvidenciaApta) ||
		(v.ResultadoSubsanacion == ResultadoSubsanacionRechazada && v.Estado != EstadoEvidenciaNoApta) ||
		(v.ResultadoSubsanacion != ResultadoSubsanacionAceptada &&
			v.ResultadoSubsanacion != ResultadoSubsanacionRechazada) {
		return ErrValoracionEvidenciaInvalida
	}
	return nil
}

// ResultadoDecisionTecnica es una invariante del procedimiento, no un
// catalogo de tipos de merito. Las categorias de meritos viven en criterios
// versionados mediante ReferenciaCriterio.
type ResultadoDecisionTecnica string

const (
	ResultadoAceptado             ResultadoDecisionTecnica = "aceptado"
	ResultadoDesestimado          ResultadoDecisionTecnica = "desestimado"
	ResultadoPendienteSubsanacion ResultadoDecisionTecnica = "pendiente_subsanacion"
)

func (r ResultadoDecisionTecnica) Valido() bool {
	return r == ResultadoAceptado || r == ResultadoDesestimado || r == ResultadoPendienteSubsanacion
}

// ClaseDecisionTecnica hace explicito por que se crea cada asiento del
// historial. Una revocacion y una rehabilitacion nunca sobrescriben la
// decision anterior: la sustituyen conservandola integra.
type ClaseDecisionTecnica string

const (
	ClaseDecisionInicial        ClaseDecisionTecnica = "inicial"
	ClaseDecisionRectificacion  ClaseDecisionTecnica = "rectificacion"
	ClaseDecisionRevocacion     ClaseDecisionTecnica = "revocacion"
	ClaseDecisionRehabilitacion ClaseDecisionTecnica = "rehabilitacion"
)

func (c ClaseDecisionTecnica) Valida() bool {
	switch c {
	case ClaseDecisionInicial, ClaseDecisionRectificacion, ClaseDecisionRevocacion,
		ClaseDecisionRehabilitacion:
		return true
	default:
		return false
	}
}

// ReferenciaDecision enlaza una rectificacion con la decision exacta que
// sustituye. La huella impide que una referencia estable apunte despues a un
// contenido distinto.
type ReferenciaDecision struct {
	ID           string `json:"id"`
	Numero       int    `json:"numero"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaDecision) Validar() error {
	if !referenciaOpacaValida(r.ID) || r.Numero < 1 || !huellaSHA256Valida(r.HuellaSHA256) {
		return ErrDecisionTecnicaInvalida
	}
	return nil
}

// ContenidoDecisionTecnica es el contenido administrativo que se firma. Para
// obtener una firma valida se calcula primero HuellaContenidoSHA256 y se
// entrega esa huella al conector de firma.
type ContenidoDecisionTecnica struct {
	ID                           string                   `json:"id"`
	Numero                       int                      `json:"numero"`
	Clase                        ClaseDecisionTecnica     `json:"clase"`
	ProcesoRef                   string                   `json:"proceso_ref"`
	SolicitudRef                 string                   `json:"solicitud_ref"`
	SujetoRef                    string                   `json:"sujeto_ref"`
	BaremacionMeritoRef          string                   `json:"baremacion_merito_ref"`
	VersionAnteriorBaremacion    uint64                   `json:"version_anterior_baremacion"`
	VersionBaremacion            uint64                   `json:"version_baremacion"`
	HuellaEstadoAnteriorSHA256   string                   `json:"huella_estado_anterior_sha256"`
	HuellaEstadoResultanteSHA256 string                   `json:"huella_estado_resultante_sha256"`
	Criterio                     ReferenciaCriterio       `json:"criterio"`
	CalculoOficial               CalculoOficialBaremacion `json:"calculo_oficial"`
	ValoracionesEvidencia        []ValoracionEvidencia    `json:"valoraciones_evidencia"`
	PuntosDeclarados             Puntos                   `json:"puntos_declarados"`
	PuntosReconocidos            Puntos                   `json:"puntos_reconocidos"`
	Resultado                    ResultadoDecisionTecnica `json:"resultado"`
	DecisorRef                   string                   `json:"decisor_ref"`
	PerfilDecisorClave           string                   `json:"perfil_decisor_clave"`
	MotivoClave                  string                   `json:"motivo_clave"`
	Motivo                       string                   `json:"motivo"`
	FuentesNormativasRefs        []string                 `json:"fuentes_normativas_refs"`
	AutorizacionRef              string                   `json:"autorizacion_ref"`
	FinalidadClave               string                   `json:"finalidad_clave"`
	CorrelacionRef               string                   `json:"correlacion_ref"`
	DecididaEn                   time.Time                `json:"decidida_en"`
	Sustituye                    *ReferenciaDecision      `json:"sustituye,omitempty"`
}

func (c ContenidoDecisionTecnica) Validar() error {
	if !referenciaOpacaValida(c.ID) || c.Numero < 1 || !c.Clase.Valida() ||
		!referenciaOpacaValida(c.ProcesoRef) || !referenciaOpacaValida(c.SolicitudRef) ||
		!referenciaOpacaValida(c.SujetoRef) || !referenciaOpacaValida(c.BaremacionMeritoRef) ||
		c.VersionAnteriorBaremacion < 1 || c.VersionBaremacion != c.VersionAnteriorBaremacion+1 ||
		c.VersionBaremacion != uint64(c.Numero)+1 || !huellaSHA256Valida(c.HuellaEstadoAnteriorSHA256) ||
		!huellaSHA256Valida(c.HuellaEstadoResultanteSHA256) || c.Criterio.Validar() != nil ||
		c.Criterio.ProcesoRef != c.ProcesoRef ||
		c.CalculoOficial.validarPara(c.ProcesoRef, c.SolicitudRef, c.SujetoRef, c.BaremacionMeritoRef, c.Criterio) != nil ||
		!c.PuntosDeclarados.Validos() || !c.PuntosReconocidos.Validos() ||
		c.PuntosReconocidos > c.CalculoOficial.PuntosCalculados || !c.Resultado.Valido() ||
		!referenciaOpacaValida(c.DecisorRef) || !claveNegocioValida(c.PerfilDecisorClave) ||
		!claveNegocioValida(c.MotivoClave) || !textoValido(c.Motivo) ||
		!referenciaOpacaValida(c.AutorizacionRef) || !claveNegocioValida(c.FinalidadClave) ||
		!referenciaOpacaValida(c.CorrelacionRef) || c.DecididaEn.IsZero() {
		return ErrContenidoDecisionInvalido
	}
	valoraciones, err := canonizarValoracionesEvidencia(c.ValoracionesEvidencia)
	if err != nil || len(valoraciones) == 0 || len(valoraciones) > maximoEvidenciasPorMerito {
		return ErrContenidoDecisionInvalido
	}
	if _, err := canonizarReferencias(c.FuentesNormativasRefs); err != nil {
		return ErrContenidoDecisionInvalido
	}
	if !mismoConjuntoEvidencias(c.CalculoOficial.Evidencias, evidenciasDeValoraciones(valoraciones)) ||
		c.DecididaEn.Before(c.CalculoOficial.CalculadoEn) {
		return ErrContenidoDecisionInvalido
	}
	if (c.Resultado == ResultadoDesestimado || c.Resultado == ResultadoPendienteSubsanacion) && c.PuntosReconocidos != 0 {
		return ErrContenidoDecisionInvalido
	}
	if c.Resultado == ResultadoAceptado && !contieneEstadoEvidencia(valoraciones, EstadoEvidenciaApta) {
		return ErrContenidoDecisionInvalido
	}
	if c.Resultado == ResultadoPendienteSubsanacion && !contieneEstadoEvidencia(valoraciones, EstadoEvidenciaSubsanable) {
		return ErrContenidoDecisionInvalido
	}
	if c.Clase == ClaseDecisionInicial {
		if c.Numero != 1 || c.Sustituye != nil {
			return ErrContenidoDecisionInvalido
		}
	} else if c.Numero < 2 || c.Sustituye == nil || c.Sustituye.Validar() != nil {
		return ErrContenidoDecisionInvalido
	}
	if c.Clase == ClaseDecisionRevocacion && c.Resultado == ResultadoAceptado {
		return ErrContenidoDecisionInvalido
	}
	if c.Clase == ClaseDecisionRehabilitacion && c.Resultado != ResultadoAceptado {
		return ErrContenidoDecisionInvalido
	}
	return nil
}

func (c ContenidoDecisionTecnica) clonarCanonico() (ContenidoDecisionTecnica, error) {
	canonico := c
	canonico.ID = strings.TrimSpace(c.ID)
	canonico.ProcesoRef = strings.TrimSpace(c.ProcesoRef)
	canonico.SolicitudRef = strings.TrimSpace(c.SolicitudRef)
	canonico.SujetoRef = strings.TrimSpace(c.SujetoRef)
	canonico.BaremacionMeritoRef = strings.TrimSpace(c.BaremacionMeritoRef)
	canonico.HuellaEstadoAnteriorSHA256 = strings.TrimSpace(c.HuellaEstadoAnteriorSHA256)
	canonico.HuellaEstadoResultanteSHA256 = strings.TrimSpace(c.HuellaEstadoResultanteSHA256)
	canonico.DecisorRef = strings.TrimSpace(c.DecisorRef)
	canonico.MotivoClave = strings.TrimSpace(c.MotivoClave)
	canonico.Motivo = strings.TrimSpace(c.Motivo)
	canonico.AutorizacionRef = strings.TrimSpace(c.AutorizacionRef)
	canonico.FinalidadClave = strings.TrimSpace(c.FinalidadClave)
	canonico.CorrelacionRef = strings.TrimSpace(c.CorrelacionRef)
	canonico.Criterio = canonizarCriterio(c.Criterio)
	var err error
	canonico.CalculoOficial, err = c.CalculoOficial.clonarCanonico()
	if err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	canonico.ValoracionesEvidencia, err = canonizarValoracionesEvidencia(c.ValoracionesEvidencia)
	if err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	canonico.FuentesNormativasRefs, err = canonizarReferencias(c.FuentesNormativasRefs)
	if err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	canonico.DecididaEn = c.DecididaEn.UTC()
	if c.Sustituye != nil {
		sustituye := *c.Sustituye
		sustituye.ID = strings.TrimSpace(sustituye.ID)
		sustituye.HuellaSHA256 = strings.TrimSpace(sustituye.HuellaSHA256)
		canonico.Sustituye = &sustituye
	}
	if err := canonico.Validar(); err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	return canonico, nil
}

func (c ContenidoDecisionTecnica) HuellaContenidoSHA256() (string, error) {
	canonico, err := c.clonarCanonico()
	if err != nil {
		return "", err
	}
	return huellaJSON(canonico)
}

func (c ContenidoDecisionTecnica) ClonarCanonico() (ContenidoDecisionTecnica, error) {
	return c.clonarCanonico()
}

// FirmaDecisionTecnica conserva la evidencia verificable de la firma. La
// validacion criptografica real pertenece al conector de firma; el dominio
// exige sus referencias y vincula el resultado con la huella del contenido.
type FirmaDecisionTecnica struct {
	FirmanteRef                            string    `json:"firmante_ref"`
	PerfilFirmanteClave                    string    `json:"perfil_firmante_clave"`
	PoliticaFirmaRef                       string    `json:"politica_firma_ref"`
	PoliticaFirmaVersion                   int       `json:"politica_firma_version"`
	HuellaPoliticaFirmaSHA256              string    `json:"huella_politica_firma_sha256"`
	PerfilFirmaAlcanzadoClave              string    `json:"perfil_firma_alcanzado_clave"`
	RequiereFirmaInteractiva               bool      `json:"requiere_firma_interactiva"`
	RequiereValidacionServidor             bool      `json:"requiere_validacion_servidor"`
	RequiereSelloTiempo                    bool      `json:"requiere_sello_tiempo"`
	RequiereAumentoLongevidad              bool      `json:"requiere_aumento_longevidad"`
	SesionFirmaInteractivaRef              string    `json:"sesion_firma_interactiva_ref"`
	HuellaEvidenciaFirmaInteractivaSHA256  string    `json:"huella_evidencia_firma_interactiva_sha256"`
	DocumentoFirmableRef                   string    `json:"documento_firmable_ref"`
	VersionDocumentoFirmable               string    `json:"version_documento_firmable"`
	HuellaDocumentoFirmableSHA256          string    `json:"huella_documento_firmable_sha256"`
	EvidenciaCustodiaRef                   string    `json:"evidencia_custodia_ref"`
	FirmaRef                               string    `json:"firma_ref"`
	HuellaFirmaSHA256                      string    `json:"huella_firma_sha256"`
	DocumentoFirmadoRef                    string    `json:"documento_firmado_ref"`
	HuellaDocumentoSHA256                  string    `json:"huella_documento_sha256"`
	DocumentoFirmadoCustodiadoRef          string    `json:"documento_firmado_custodiado_ref"`
	VersionDocumentoFirmadoCustodiado      string    `json:"version_documento_firmado_custodiado"`
	EvidenciaRecuperacionFirmadoRef        string    `json:"evidencia_recuperacion_firmado_ref"`
	HuellaEvidenciaRecuperacionSHA256      string    `json:"huella_evidencia_recuperacion_sha256"`
	EvidenciaCustodiaDocumentoFirmadoRef   string    `json:"evidencia_custodia_documento_firmado_ref"`
	EvidenciaRetencionDocumentoFirmadoRef  string    `json:"evidencia_retencion_documento_firmado_ref"`
	PoliticaRetencionDocumentoFirmadoRef   string    `json:"politica_retencion_documento_firmado_ref"`
	DocumentoFirmadoRetenidoHasta          time.Time `json:"documento_firmado_retenido_hasta"`
	ManifiestoProbatorioRef                string    `json:"manifiesto_probatorio_ref"`
	HuellaManifiestoProbatorioSHA256       string    `json:"huella_manifiesto_probatorio_sha256"`
	SelloManifiestoProbatorioHMACSHA256    string    `json:"sello_manifiesto_probatorio_hmac_sha256"`
	HuellaContenidoSHA256                  string    `json:"huella_contenido_sha256"`
	ValidacionInicialFirmaRef              string    `json:"validacion_inicial_firma_ref"`
	HuellaValidacionInicialSHA256          string    `json:"huella_validacion_inicial_sha256"`
	ValidadaInicialEn                      time.Time `json:"validada_inicial_en"`
	ValidacionFirmaRef                     string    `json:"validacion_firma_ref"`
	HuellaValidacionSHA256                 string    `json:"huella_validacion_sha256"`
	ValidadaEn                             time.Time `json:"validada_en"`
	SelloTiempoRef                         string    `json:"sello_tiempo_ref,omitempty"`
	HuellaSelloTiempoSHA256                string    `json:"huella_sello_tiempo_sha256,omitempty"`
	PoliticaSelloTiempoRef                 string    `json:"politica_sello_tiempo_ref,omitempty"`
	PoliticaSelloTiempoVersion             int       `json:"politica_sello_tiempo_version,omitempty"`
	HuellaPoliticaSelloTiempoSHA256        string    `json:"huella_politica_sello_tiempo_sha256,omitempty"`
	ValidacionSelloTiempoRef               string    `json:"validacion_sello_tiempo_ref,omitempty"`
	HuellaValidacionSelloTiempoSHA256      string    `json:"huella_validacion_sello_tiempo_sha256,omitempty"`
	SelladaEn                              time.Time `json:"sellada_en,omitempty"`
	ValidacionDocumentoSelladoRef          string    `json:"validacion_documento_sellado_ref,omitempty"`
	HuellaValidacionDocumentoSelladoSHA256 string    `json:"huella_validacion_documento_sellado_sha256,omitempty"`
	ValidadoDocumentoSelladoEn             time.Time `json:"validado_documento_sellado_en,omitempty"`
	NivelLongevidadClave                   string    `json:"nivel_longevidad_clave,omitempty"`
	AumentoLongevidadRef                   string    `json:"aumento_longevidad_ref,omitempty"`
	HuellaAumentoLongevidadSHA256          string    `json:"huella_aumento_longevidad_sha256,omitempty"`
	PoliticaLongevidadRef                  string    `json:"politica_longevidad_ref,omitempty"`
	PoliticaLongevidadVersion              int       `json:"politica_longevidad_version,omitempty"`
	HuellaPoliticaLongevidadSHA256         string    `json:"huella_politica_longevidad_sha256,omitempty"`
	ValidacionLongevidadRef                string    `json:"validacion_longevidad_ref,omitempty"`
	HuellaValidacionLongevidadSHA256       string    `json:"huella_validacion_longevidad_sha256,omitempty"`
	AumentadaEn                            time.Time `json:"aumentada_en,omitempty"`
	FirmadaEn                              time.Time `json:"firmada_en"`
}

func (f FirmaDecisionTecnica) validarPara(contenido ContenidoDecisionTecnica) error {
	if !referenciaOpacaValida(f.FirmanteRef) || !claveNegocioValida(f.PerfilFirmanteClave) ||
		!referenciaOpacaValida(f.PoliticaFirmaRef) || f.PoliticaFirmaVersion < 1 ||
		!huellaSHA256Valida(f.HuellaPoliticaFirmaSHA256) || !perfilFirmaDecisionValido(f.PerfilFirmaAlcanzadoClave) ||
		!f.RequiereFirmaInteractiva ||
		!f.RequiereValidacionServidor || !referenciaOpacaValida(f.SesionFirmaInteractivaRef) ||
		!huellaSHA256Valida(f.HuellaEvidenciaFirmaInteractivaSHA256) ||
		!referenciaOpacaValida(f.DocumentoFirmableRef) || !referenciaOpacaValida(f.VersionDocumentoFirmable) ||
		!huellaSHA256Valida(f.HuellaDocumentoFirmableSHA256) || !referenciaOpacaValida(f.EvidenciaCustodiaRef) ||
		!referenciaOpacaValida(f.FirmaRef) ||
		!huellaSHA256Valida(f.HuellaFirmaSHA256) || !referenciaOpacaValida(f.DocumentoFirmadoRef) ||
		!huellaSHA256Valida(f.HuellaDocumentoSHA256) ||
		!referenciaOpacaValida(f.DocumentoFirmadoCustodiadoRef) ||
		!referenciaOpacaValida(f.VersionDocumentoFirmadoCustodiado) ||
		!referenciaOpacaValida(f.EvidenciaRecuperacionFirmadoRef) ||
		!huellaSHA256Valida(f.HuellaEvidenciaRecuperacionSHA256) ||
		!referenciaOpacaValida(f.EvidenciaCustodiaDocumentoFirmadoRef) ||
		!referenciaOpacaValida(f.EvidenciaRetencionDocumentoFirmadoRef) ||
		!referenciaOpacaValida(f.PoliticaRetencionDocumentoFirmadoRef) ||
		f.DocumentoFirmadoRetenidoHasta.IsZero() ||
		!f.DocumentoFirmadoRetenidoHasta.After(f.ValidadaEn) ||
		!referenciaOpacaValida(f.ManifiestoProbatorioRef) ||
		!huellaSHA256Valida(f.HuellaManifiestoProbatorioSHA256) ||
		!huellaHMACSHA256DominioValida(f.SelloManifiestoProbatorioHMACSHA256) ||
		!huellaSHA256Valida(f.HuellaContenidoSHA256) ||
		!referenciaOpacaValida(f.ValidacionInicialFirmaRef) ||
		!huellaSHA256Valida(f.HuellaValidacionInicialSHA256) || f.ValidadaInicialEn.IsZero() ||
		f.ValidadaInicialEn.Before(f.FirmadaEn) ||
		!referenciaOpacaValida(f.ValidacionFirmaRef) || !huellaSHA256Valida(f.HuellaValidacionSHA256) ||
		f.ValidadaEn.IsZero() || f.ValidadaEn.Before(f.ValidadaInicialEn) ||
		f.FirmadaEn.IsZero() || f.FirmadaEn.Before(contenido.DecididaEn) ||
		f.FirmanteRef != contenido.DecisorRef || f.PerfilFirmanteClave != contenido.PerfilDecisorClave {
		return ErrFirmaDecisionInvalida
	}
	if !f.perfilFirmaCoherente() || !f.evidenciaSelloCoherente() || !f.evidenciaLongevidadCoherente() {
		return ErrFirmaDecisionInvalida
	}
	if f.RequiereAumentoLongevidad && f.ValidadaEn.Before(f.AumentadaEn) {
		return ErrFirmaDecisionInvalida
	}
	if !f.RequiereAumentoLongevidad && f.RequiereSelloTiempo && f.ValidadaEn.Before(f.SelladaEn) {
		return ErrFirmaDecisionInvalida
	}
	huellaContenido, err := contenido.HuellaContenidoSHA256()
	if err != nil || f.HuellaContenidoSHA256 != huellaContenido {
		return ErrFirmaDecisionInvalida
	}
	return nil
}

func (f FirmaDecisionTecnica) clonarCanonica() FirmaDecisionTecnica {
	canonica := f
	canonica.FirmanteRef = strings.TrimSpace(f.FirmanteRef)
	canonica.PoliticaFirmaRef = strings.TrimSpace(f.PoliticaFirmaRef)
	canonica.HuellaPoliticaFirmaSHA256 = strings.TrimSpace(f.HuellaPoliticaFirmaSHA256)
	canonica.PerfilFirmaAlcanzadoClave = strings.TrimSpace(f.PerfilFirmaAlcanzadoClave)
	canonica.SesionFirmaInteractivaRef = strings.TrimSpace(f.SesionFirmaInteractivaRef)
	canonica.HuellaEvidenciaFirmaInteractivaSHA256 = strings.TrimSpace(f.HuellaEvidenciaFirmaInteractivaSHA256)
	canonica.DocumentoFirmableRef = strings.TrimSpace(f.DocumentoFirmableRef)
	canonica.VersionDocumentoFirmable = strings.TrimSpace(f.VersionDocumentoFirmable)
	canonica.HuellaDocumentoFirmableSHA256 = strings.TrimSpace(f.HuellaDocumentoFirmableSHA256)
	canonica.EvidenciaCustodiaRef = strings.TrimSpace(f.EvidenciaCustodiaRef)
	canonica.FirmaRef = strings.TrimSpace(f.FirmaRef)
	canonica.HuellaFirmaSHA256 = strings.TrimSpace(f.HuellaFirmaSHA256)
	canonica.DocumentoFirmadoRef = strings.TrimSpace(f.DocumentoFirmadoRef)
	canonica.HuellaDocumentoSHA256 = strings.TrimSpace(f.HuellaDocumentoSHA256)
	canonica.DocumentoFirmadoCustodiadoRef = strings.TrimSpace(f.DocumentoFirmadoCustodiadoRef)
	canonica.VersionDocumentoFirmadoCustodiado = strings.TrimSpace(f.VersionDocumentoFirmadoCustodiado)
	canonica.EvidenciaRecuperacionFirmadoRef = strings.TrimSpace(f.EvidenciaRecuperacionFirmadoRef)
	canonica.HuellaEvidenciaRecuperacionSHA256 = strings.TrimSpace(f.HuellaEvidenciaRecuperacionSHA256)
	canonica.EvidenciaCustodiaDocumentoFirmadoRef = strings.TrimSpace(f.EvidenciaCustodiaDocumentoFirmadoRef)
	canonica.EvidenciaRetencionDocumentoFirmadoRef = strings.TrimSpace(f.EvidenciaRetencionDocumentoFirmadoRef)
	canonica.PoliticaRetencionDocumentoFirmadoRef = strings.TrimSpace(f.PoliticaRetencionDocumentoFirmadoRef)
	canonica.DocumentoFirmadoRetenidoHasta = f.DocumentoFirmadoRetenidoHasta.UTC()
	canonica.ManifiestoProbatorioRef = strings.TrimSpace(f.ManifiestoProbatorioRef)
	canonica.HuellaManifiestoProbatorioSHA256 = strings.TrimSpace(f.HuellaManifiestoProbatorioSHA256)
	canonica.SelloManifiestoProbatorioHMACSHA256 = strings.TrimSpace(f.SelloManifiestoProbatorioHMACSHA256)
	canonica.HuellaContenidoSHA256 = strings.TrimSpace(f.HuellaContenidoSHA256)
	canonica.ValidacionInicialFirmaRef = strings.TrimSpace(f.ValidacionInicialFirmaRef)
	canonica.HuellaValidacionInicialSHA256 = strings.TrimSpace(f.HuellaValidacionInicialSHA256)
	canonica.ValidadaInicialEn = f.ValidadaInicialEn.UTC()
	canonica.ValidacionFirmaRef = strings.TrimSpace(f.ValidacionFirmaRef)
	canonica.HuellaValidacionSHA256 = strings.TrimSpace(f.HuellaValidacionSHA256)
	canonica.ValidadaEn = f.ValidadaEn.UTC()
	canonica.SelloTiempoRef = strings.TrimSpace(f.SelloTiempoRef)
	canonica.HuellaSelloTiempoSHA256 = strings.TrimSpace(f.HuellaSelloTiempoSHA256)
	canonica.PoliticaSelloTiempoRef = strings.TrimSpace(f.PoliticaSelloTiempoRef)
	canonica.HuellaPoliticaSelloTiempoSHA256 = strings.TrimSpace(f.HuellaPoliticaSelloTiempoSHA256)
	canonica.ValidacionSelloTiempoRef = strings.TrimSpace(f.ValidacionSelloTiempoRef)
	canonica.HuellaValidacionSelloTiempoSHA256 = strings.TrimSpace(f.HuellaValidacionSelloTiempoSHA256)
	if !f.SelladaEn.IsZero() {
		canonica.SelladaEn = f.SelladaEn.UTC()
	}
	canonica.ValidacionDocumentoSelladoRef = strings.TrimSpace(f.ValidacionDocumentoSelladoRef)
	canonica.HuellaValidacionDocumentoSelladoSHA256 = strings.TrimSpace(f.HuellaValidacionDocumentoSelladoSHA256)
	if !f.ValidadoDocumentoSelladoEn.IsZero() {
		canonica.ValidadoDocumentoSelladoEn = f.ValidadoDocumentoSelladoEn.UTC()
	}
	canonica.NivelLongevidadClave = strings.TrimSpace(f.NivelLongevidadClave)
	canonica.AumentoLongevidadRef = strings.TrimSpace(f.AumentoLongevidadRef)
	canonica.HuellaAumentoLongevidadSHA256 = strings.TrimSpace(f.HuellaAumentoLongevidadSHA256)
	canonica.PoliticaLongevidadRef = strings.TrimSpace(f.PoliticaLongevidadRef)
	canonica.HuellaPoliticaLongevidadSHA256 = strings.TrimSpace(f.HuellaPoliticaLongevidadSHA256)
	canonica.ValidacionLongevidadRef = strings.TrimSpace(f.ValidacionLongevidadRef)
	canonica.HuellaValidacionLongevidadSHA256 = strings.TrimSpace(f.HuellaValidacionLongevidadSHA256)
	if !f.AumentadaEn.IsZero() {
		canonica.AumentadaEn = f.AumentadaEn.UTC()
	}
	canonica.FirmadaEn = f.FirmadaEn.UTC()
	return canonica
}

// DecisionTecnica es un asiento firmado e inmutable por huella. Cualquier
// cambio posterior invalida Validar y debe expresarse como un asiento nuevo.
type DecisionTecnica struct {
	Contenido    ContenidoDecisionTecnica `json:"contenido"`
	Firma        FirmaDecisionTecnica     `json:"firma"`
	HuellaSHA256 string                   `json:"huella_sha256"`
}

func ConstituirDecisionFirmada(contenido ContenidoDecisionTecnica, firma FirmaDecisionTecnica) (DecisionTecnica, error) {
	canonico, err := contenido.clonarCanonico()
	if err != nil {
		return DecisionTecnica{}, err
	}
	firmaCanonica := firma.clonarCanonica()
	if err := firmaCanonica.validarPara(canonico); err != nil {
		return DecisionTecnica{}, err
	}
	decision := DecisionTecnica{Contenido: canonico, Firma: firmaCanonica}
	huella, err := decision.calcularHuellaSHA256()
	if err != nil {
		return DecisionTecnica{}, err
	}
	decision.HuellaSHA256 = huella
	return decision, nil
}

func (d DecisionTecnica) Referencia() ReferenciaDecision {
	return ReferenciaDecision{ID: d.Contenido.ID, Numero: d.Contenido.Numero, HuellaSHA256: d.HuellaSHA256}
}

func (d DecisionTecnica) Validar() error {
	canonico, err := d.Contenido.clonarCanonico()
	if err != nil || d.Firma.validarPara(canonico) != nil || !huellaSHA256Valida(d.HuellaSHA256) {
		return ErrDecisionTecnicaInvalida
	}
	huella, err := d.calcularHuellaSHA256()
	if err != nil || huella != d.HuellaSHA256 {
		return ErrDecisionTecnicaInvalida
	}
	return nil
}

func (d DecisionTecnica) calcularHuellaSHA256() (string, error) {
	contenido, err := d.Contenido.clonarCanonico()
	if err != nil {
		return "", err
	}
	firma := d.Firma.clonarCanonica()
	if err := firma.validarPara(contenido); err != nil {
		return "", err
	}
	return huellaJSON(struct {
		Contenido ContenidoDecisionTecnica `json:"contenido"`
		Firma     FirmaDecisionTecnica     `json:"firma"`
	}{Contenido: contenido, Firma: firma})
}

func (d DecisionTecnica) clonarCanonica() (DecisionTecnica, error) {
	contenido, err := d.Contenido.clonarCanonico()
	if err != nil {
		return DecisionTecnica{}, err
	}
	clon := DecisionTecnica{
		Contenido:    contenido,
		Firma:        d.Firma.clonarCanonica(),
		HuellaSHA256: strings.TrimSpace(d.HuellaSHA256),
	}
	if err := clon.Validar(); err != nil {
		return DecisionTecnica{}, err
	}
	return clon, nil
}

// AltaMeritoBaremable contiene las referencias minimas para crear un merito
// atomico. EvidenciasIniciales puede contener varios documentos que, solo en
// conjunto, acreditan el mismo merito y no deben puntuar por separado.
type AltaMeritoBaremable struct {
	ID                  string                   `json:"id"`
	ProcesoRef          string                   `json:"proceso_ref"`
	SolicitudRef        string                   `json:"solicitud_ref"`
	SujetoRef           string                   `json:"sujeto_ref"`
	Criterio            ReferenciaCriterio       `json:"criterio"`
	EvidenciasIniciales []EvidenciaMerito        `json:"evidencias_iniciales"`
	PuntosDeclarados    Puntos                   `json:"puntos_declarados"`
	CalculoOficial      CalculoOficialBaremacion `json:"calculo_oficial"`
	CreadaEn            time.Time                `json:"creada_en"`
}

// PropuestaDecisionTecnica expresa una valoracion antes de ser firmada. La
// clase, numero, merito y referencia sustituida los determina el agregado.
type PropuestaDecisionTecnica struct {
	ID                    string                   `json:"id"`
	CalculoOficial        CalculoOficialBaremacion `json:"calculo_oficial"`
	PuntosReconocidos     Puntos                   `json:"puntos_reconocidos"`
	Resultado             ResultadoDecisionTecnica `json:"resultado"`
	DecisorRef            string                   `json:"decisor_ref"`
	PerfilDecisorClave    string                   `json:"perfil_decisor_clave"`
	ValoracionesEvidencia []ValoracionEvidencia    `json:"valoraciones_evidencia"`
	MotivoClave           string                   `json:"motivo_clave"`
	Motivo                string                   `json:"motivo"`
	FuentesNormativasRefs []string                 `json:"fuentes_normativas_refs"`
	AutorizacionRef       string                   `json:"autorizacion_ref"`
	FinalidadClave        string                   `json:"finalidad_clave"`
	CorrelacionRef        string                   `json:"correlacion_ref"`
	DecididaEn            time.Time                `json:"decidida_en"`
}

// BaremacionMerito es el historial de un unico merito bajo un unico criterio.
// Puede reunir varias evidencias y Decisiones solo crece por incorporacion.
type BaremacionMerito struct {
	ID                  string                   `json:"id"`
	ProcesoRef          string                   `json:"proceso_ref"`
	SolicitudRef        string                   `json:"solicitud_ref"`
	SujetoRef           string                   `json:"sujeto_ref"`
	Criterio            ReferenciaCriterio       `json:"criterio"`
	EvidenciasIniciales []EvidenciaMerito        `json:"evidencias_iniciales"`
	PuntosDeclarados    Puntos                   `json:"puntos_declarados"`
	CalculoInicial      CalculoOficialBaremacion `json:"calculo_inicial"`
	CreadaEn            time.Time                `json:"creada_en"`
	Decisiones          []DecisionTecnica        `json:"decisiones"`
}

func NuevaBaremacionMerito(alta AltaMeritoBaremable) (BaremacionMerito, error) {
	baremacion := BaremacionMerito{
		ID:                  strings.TrimSpace(alta.ID),
		ProcesoRef:          strings.TrimSpace(alta.ProcesoRef),
		SolicitudRef:        strings.TrimSpace(alta.SolicitudRef),
		SujetoRef:           strings.TrimSpace(alta.SujetoRef),
		Criterio:            alta.Criterio,
		EvidenciasIniciales: alta.EvidenciasIniciales,
		PuntosDeclarados:    alta.PuntosDeclarados,
		CalculoInicial:      alta.CalculoOficial,
		CreadaEn:            alta.CreadaEn.UTC(),
		Decisiones:          []DecisionTecnica{},
	}
	baremacion.Criterio = canonizarCriterio(baremacion.Criterio)
	var err error
	baremacion.CalculoInicial, err = alta.CalculoOficial.clonarCanonico()
	if err != nil {
		return BaremacionMerito{}, err
	}
	baremacion.EvidenciasIniciales, err = canonizarEvidencias(alta.EvidenciasIniciales)
	if err != nil {
		return BaremacionMerito{}, err
	}
	if err := baremacion.Validar(); err != nil {
		return BaremacionMerito{}, err
	}
	return baremacion, nil
}

func (b BaremacionMerito) Validar() error {
	if !referenciaOpacaValida(b.ID) || !referenciaOpacaValida(b.ProcesoRef) ||
		!referenciaOpacaValida(b.SolicitudRef) || !referenciaOpacaValida(b.SujetoRef) ||
		b.Criterio.Validar() != nil || b.Criterio.ProcesoRef != b.ProcesoRef ||
		b.CalculoInicial.validarPara(b.ProcesoRef, b.SolicitudRef, b.SujetoRef, b.ID, b.Criterio) != nil ||
		!b.PuntosDeclarados.Validos() || b.CreadaEn.IsZero() || b.CreadaEn.Before(b.CalculoInicial.CalculadoEn) ||
		len(b.Decisiones) > maximoDecisionesPorMerito {
		return ErrBaremacionInvalida
	}
	evidenciasIniciales, err := canonizarEvidencias(b.EvidenciasIniciales)
	if err != nil || len(evidenciasIniciales) == 0 || len(evidenciasIniciales) > maximoEvidenciasPorMerito {
		return ErrBaremacionInvalida
	}
	if !mismoConjuntoEvidencias(evidenciasIniciales, b.CalculoInicial.Evidencias) {
		return ErrBaremacionInvalida
	}
	vistos := make(map[string]struct{}, len(b.Decisiones))
	huellaAnterior, err := b.huellaEstadoAdministrativoBase()
	if err != nil {
		return ErrHistorialDecisionesInvalido
	}
	for indice := range b.Decisiones {
		actual := b.Decisiones[indice]
		huellaResultante, err := huellaEstadoResultanteDesde(huellaAnterior, actual.Contenido)
		if err != nil {
			return ErrHistorialDecisionesInvalido
		}
		if actual.Validar() != nil || actual.Contenido.ProcesoRef != b.ProcesoRef ||
			actual.Contenido.SolicitudRef != b.SolicitudRef || actual.Contenido.SujetoRef != b.SujetoRef ||
			actual.Contenido.BaremacionMeritoRef != b.ID ||
			actual.Contenido.Criterio != b.Criterio || actual.Contenido.VersionAnteriorBaremacion != uint64(indice+1) ||
			actual.Contenido.VersionBaremacion != uint64(indice+2) ||
			actual.Contenido.HuellaEstadoAnteriorSHA256 != huellaAnterior ||
			actual.Contenido.HuellaEstadoResultanteSHA256 != huellaResultante ||
			actual.Contenido.PuntosDeclarados != b.PuntosDeclarados || actual.Contenido.Numero != indice+1 {
			return ErrHistorialDecisionesInvalido
		}
		if _, existe := vistos[actual.Contenido.ID]; existe {
			return ErrHistorialDecisionesInvalido
		}
		vistos[actual.Contenido.ID] = struct{}{}
		if indice == 0 {
			if actual.Contenido.Clase != ClaseDecisionInicial || actual.Contenido.Sustituye != nil ||
				!actual.Contenido.CalculoOficial.CoincideCon(b.CalculoInicial) ||
				actual.Contenido.DecididaEn.Before(b.CreadaEn) ||
				!mismoConjuntoEvidencias(evidenciasIniciales, evidenciasDeValoraciones(actual.Contenido.ValoracionesEvidencia)) {
				return ErrHistorialDecisionesInvalido
			}
			huellaAnterior = huellaResultante
			continue
		}
		anterior := b.Decisiones[indice-1]
		if actual.Contenido.Sustituye == nil || *actual.Contenido.Sustituye != anterior.Referencia() ||
			actual.Contenido.DecididaEn.Before(anterior.Firma.FirmadaEn) ||
			validarEvolucionEvidencias(anterior.Contenido.ValoracionesEvidencia, actual.Contenido.ValoracionesEvidencia) != nil ||
			validarCambioDecision(anterior, actual) != nil {
			return ErrHistorialDecisionesInvalido
		}
		huellaAnterior = huellaResultante
	}
	return nil
}

func (b BaremacionMerito) PrepararDecisionInicial(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error) {
	if err := b.Validar(); err != nil || len(b.Decisiones) != 0 || !propuesta.CalculoOficial.CoincideCon(b.CalculoInicial) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	return b.prepararContenido(ClaseDecisionInicial, nil, propuesta)
}

// PrepararRectificacion corrige puntos, motivacion, fuentes o valoraciones
// documentales. Los cambios que retiran o conceden una aceptacion se tipifican
// expresamente mediante PrepararRevocacion o PrepararRehabilitacion.
func (b BaremacionMerito) PrepararRectificacion(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error) {
	ultima, existe := b.UltimaDecision()
	if err := b.Validar(); err != nil || !existe ||
		(ultima.Contenido.Resultado == ResultadoAceptado && propuesta.Resultado != ResultadoAceptado) ||
		(ultima.Contenido.Resultado != ResultadoAceptado && propuesta.Resultado == ResultadoAceptado) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	referencia := ultima.Referencia()
	contenido, err := b.prepararContenido(ClaseDecisionRectificacion, &referencia, propuesta)
	if err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	if !contenidoAdministrativoCambia(ultima.Contenido, contenido) {
		return ContenidoDecisionTecnica{}, ErrDecisionSinCambios
	}
	return contenido, nil
}

// PrepararRevocacion retira una aceptacion previa sin borrar la decision que
// la concedio. La propuesta debe incluir la nueva valoracion de cada evidencia
// y puede dejar el merito desestimado o pendiente de subsanacion.
func (b BaremacionMerito) PrepararRevocacion(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error) {
	ultima, existe := b.UltimaDecision()
	if err := b.Validar(); err != nil || !existe || ultima.Contenido.Resultado != ResultadoAceptado ||
		propuesta.Resultado == ResultadoAceptado {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	referencia := ultima.Referencia()
	return b.prepararContenido(ClaseDecisionRevocacion, &referencia, propuesta)
}

// PrepararRehabilitacion acepta un merito antes desestimado o pendiente. Los
// puntos y valoraciones se aportan completos para que la firma cubra el nuevo
// juicio tecnico, incluida una posible evidencia de subsanacion.
func (b BaremacionMerito) PrepararRehabilitacion(propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error) {
	ultima, existe := b.UltimaDecision()
	if err := b.Validar(); err != nil || !existe || ultima.Contenido.Resultado == ResultadoAceptado ||
		propuesta.Resultado != ResultadoAceptado {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	referencia := ultima.Referencia()
	return b.prepararContenido(ClaseDecisionRehabilitacion, &referencia, propuesta)
}

func (b BaremacionMerito) prepararContenido(clase ClaseDecisionTecnica, sustituye *ReferenciaDecision, propuesta PropuestaDecisionTecnica) (ContenidoDecisionTecnica, error) {
	calculo, err := propuesta.CalculoOficial.clonarCanonico()
	if err != nil || calculo.validarPara(b.ProcesoRef, b.SolicitudRef, b.SujetoRef, b.ID, b.Criterio) != nil {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	valoraciones, err := canonizarValoracionesEvidencia(propuesta.ValoracionesEvidencia)
	if err != nil || !mismoConjuntoEvidencias(calculo.Evidencias, evidenciasDeValoraciones(valoraciones)) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	huellaAnterior, err := b.huellaEstadoAdministrativoHasta(len(b.Decisiones))
	if err != nil {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	contenido := ContenidoDecisionTecnica{
		ID:                           strings.TrimSpace(propuesta.ID),
		Numero:                       len(b.Decisiones) + 1,
		Clase:                        clase,
		ProcesoRef:                   b.ProcesoRef,
		SolicitudRef:                 b.SolicitudRef,
		SujetoRef:                    b.SujetoRef,
		BaremacionMeritoRef:          b.ID,
		VersionAnteriorBaremacion:    uint64(len(b.Decisiones) + 1),
		VersionBaremacion:            uint64(len(b.Decisiones) + 2),
		HuellaEstadoAnteriorSHA256:   huellaAnterior,
		HuellaEstadoResultanteSHA256: strings.Repeat("0", sha256.Size*2),
		Criterio:                     b.Criterio,
		CalculoOficial:               calculo,
		ValoracionesEvidencia:        valoraciones,
		PuntosDeclarados:             b.PuntosDeclarados,
		PuntosReconocidos:            propuesta.PuntosReconocidos,
		Resultado:                    propuesta.Resultado,
		DecisorRef:                   strings.TrimSpace(propuesta.DecisorRef),
		PerfilDecisorClave:           propuesta.PerfilDecisorClave,
		MotivoClave:                  propuesta.MotivoClave,
		Motivo:                       strings.TrimSpace(propuesta.Motivo),
		FuentesNormativasRefs:        propuesta.FuentesNormativasRefs,
		AutorizacionRef:              strings.TrimSpace(propuesta.AutorizacionRef),
		FinalidadClave:               propuesta.FinalidadClave,
		CorrelacionRef:               strings.TrimSpace(propuesta.CorrelacionRef),
		DecididaEn:                   propuesta.DecididaEn.UTC(),
		Sustituye:                    sustituye,
	}
	canonico, err := contenido.clonarCanonico()
	if err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	huellaResultante, err := b.huellaEstadoAdministrativoCon(canonico)
	if err != nil {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	canonico.HuellaEstadoResultanteSHA256 = huellaResultante
	canonico, err = canonico.clonarCanonico()
	if err != nil {
		return ContenidoDecisionTecnica{}, err
	}
	if canonico.DecididaEn.Before(b.CreadaEn) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	if canonico.DecididaEn.Before(canonico.CalculoOficial.CalculadoEn) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	if ultima, existe := b.UltimaDecision(); existe && canonico.DecididaEn.Before(ultima.Firma.FirmadaEn) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	if ultima, existe := b.UltimaDecision(); existe {
		if err := validarEvolucionEvidencias(ultima.Contenido.ValoracionesEvidencia, canonico.ValoracionesEvidencia); err != nil {
			return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
		}
	} else if !mismoConjuntoEvidencias(b.EvidenciasIniciales, evidenciasDeValoraciones(canonico.ValoracionesEvidencia)) {
		return ContenidoDecisionTecnica{}, ErrTransicionDecisionInvalida
	}
	return canonico, nil
}

// IncorporarDecision es la unica transicion del agregado. Solo incorpora una
// decision ya firmada y devuelve una copia nueva con un asiento adicional.
func (b BaremacionMerito) IncorporarDecision(decision DecisionTecnica) (BaremacionMerito, error) {
	if err := b.Validar(); err != nil || decision.Validar() != nil || len(b.Decisiones) >= maximoDecisionesPorMerito {
		return BaremacionMerito{}, ErrTransicionDecisionInvalida
	}
	clon, err := b.ClonarCanonica()
	if err != nil {
		return BaremacionMerito{}, err
	}
	decisionCanonica, err := decision.clonarCanonica()
	if err != nil {
		return BaremacionMerito{}, err
	}
	clon.Decisiones = append(clon.Decisiones, decisionCanonica)
	if err := clon.Validar(); err != nil {
		return BaremacionMerito{}, ErrTransicionDecisionInvalida
	}
	return clon, nil
}

func (b BaremacionMerito) UltimaDecision() (DecisionTecnica, bool) {
	if len(b.Decisiones) == 0 {
		return DecisionTecnica{}, false
	}
	clon, err := b.Decisiones[len(b.Decisiones)-1].clonarCanonica()
	if err != nil {
		return DecisionTecnica{}, false
	}
	return clon, true
}

func (b BaremacionMerito) HistorialDecisiones() ([]DecisionTecnica, error) {
	if err := b.Validar(); err != nil {
		return nil, err
	}
	historial := make([]DecisionTecnica, len(b.Decisiones))
	for indice := range b.Decisiones {
		clon, err := b.Decisiones[indice].clonarCanonica()
		if err != nil {
			return nil, err
		}
		historial[indice] = clon
	}
	return historial, nil
}

func (b BaremacionMerito) ClonarCanonica() (BaremacionMerito, error) {
	clon := b
	clon.ID = strings.TrimSpace(b.ID)
	clon.ProcesoRef = strings.TrimSpace(b.ProcesoRef)
	clon.SolicitudRef = strings.TrimSpace(b.SolicitudRef)
	clon.SujetoRef = strings.TrimSpace(b.SujetoRef)
	clon.Criterio = canonizarCriterio(b.Criterio)
	var err error
	clon.CalculoInicial, err = b.CalculoInicial.clonarCanonico()
	if err != nil {
		return BaremacionMerito{}, err
	}
	clon.EvidenciasIniciales, err = canonizarEvidencias(b.EvidenciasIniciales)
	if err != nil {
		return BaremacionMerito{}, err
	}
	clon.CreadaEn = b.CreadaEn.UTC()
	clon.Decisiones = make([]DecisionTecnica, len(b.Decisiones))
	for indice := range b.Decisiones {
		decision, err := b.Decisiones[indice].clonarCanonica()
		if err != nil {
			return BaremacionMerito{}, err
		}
		clon.Decisiones[indice] = decision
	}
	if err := clon.Validar(); err != nil {
		return BaremacionMerito{}, err
	}
	return clon, nil
}

func (b BaremacionMerito) HuellaEstadoSHA256() (string, error) {
	canonico, err := b.ClonarCanonica()
	if err != nil {
		return "", err
	}
	return huellaJSON(canonico)
}

// HuellaEstadoAdministrativoSHA256 encadena el estado que debe quedar cubierto
// por la firma sin crear una autorreferencia criptografica. La base cubre todos
// los datos de alta; cada eslabon cubre la huella anterior y el nuevo contenido
// administrativo salvo sus dos campos de enlace. La validacion es lineal.
func (b BaremacionMerito) HuellaEstadoAdministrativoSHA256() (string, error) {
	if err := b.Validar(); err != nil {
		return "", err
	}
	return b.huellaEstadoAdministrativoHasta(len(b.Decisiones))
}

func (b BaremacionMerito) huellaEstadoAdministrativoCon(contenido ContenidoDecisionTecnica) (string, error) {
	anterior, err := b.huellaEstadoAdministrativoHasta(len(b.Decisiones))
	if err != nil {
		return "", err
	}
	return huellaEstadoResultanteDesde(anterior, contenido)
}

func (b BaremacionMerito) huellaEstadoAdministrativoHasta(limite int) (string, error) {
	if limite < 0 || limite > len(b.Decisiones) {
		return "", ErrHistorialDecisionesInvalido
	}
	huella, err := b.huellaEstadoAdministrativoBase()
	if err != nil {
		return "", err
	}
	for indice := 0; indice < limite; indice++ {
		huella, err = huellaEstadoResultanteDesde(huella, b.Decisiones[indice].Contenido)
		if err != nil {
			return "", err
		}
	}
	return huella, nil
}

func (b BaremacionMerito) huellaEstadoAdministrativoBase() (string, error) {
	calculo, err := b.CalculoInicial.clonarCanonico()
	if err != nil {
		return "", err
	}
	evidencias, err := canonizarEvidencias(b.EvidenciasIniciales)
	if err != nil {
		return "", err
	}
	return huellaJSON(struct {
		ID                  string                   `json:"id"`
		ProcesoRef          string                   `json:"proceso_ref"`
		SolicitudRef        string                   `json:"solicitud_ref"`
		SujetoRef           string                   `json:"sujeto_ref"`
		Criterio            ReferenciaCriterio       `json:"criterio"`
		EvidenciasIniciales []EvidenciaMerito        `json:"evidencias_iniciales"`
		PuntosDeclarados    Puntos                   `json:"puntos_declarados"`
		CalculoInicial      CalculoOficialBaremacion `json:"calculo_inicial"`
		CreadaEn            time.Time                `json:"creada_en"`
	}{
		ID: strings.TrimSpace(b.ID), ProcesoRef: strings.TrimSpace(b.ProcesoRef),
		SolicitudRef: strings.TrimSpace(b.SolicitudRef), SujetoRef: strings.TrimSpace(b.SujetoRef),
		Criterio: canonizarCriterio(b.Criterio), EvidenciasIniciales: evidencias,
		PuntosDeclarados: b.PuntosDeclarados, CalculoInicial: calculo, CreadaEn: b.CreadaEn.UTC(),
	})
}

func huellaEstadoResultanteDesde(huellaAnterior string, contenido ContenidoDecisionTecnica) (string, error) {
	if !huellaSHA256Valida(huellaAnterior) {
		return "", ErrHistorialDecisionesInvalido
	}
	canonico, err := contenido.clonarCanonico()
	if err != nil {
		return "", err
	}
	canonico.HuellaEstadoAnteriorSHA256 = ""
	canonico.HuellaEstadoResultanteSHA256 = ""
	return huellaJSON(struct {
		HuellaAnterior string                   `json:"huella_anterior_sha256"`
		Contenido      ContenidoDecisionTecnica `json:"contenido"`
	}{HuellaAnterior: huellaAnterior, Contenido: canonico})
}

func validarCambioDecision(anterior, actual DecisionTecnica) error {
	if !contenidoAdministrativoCambia(anterior.Contenido, actual.Contenido) {
		return ErrDecisionSinCambios
	}
	switch {
	case anterior.Contenido.Resultado == ResultadoAceptado && actual.Contenido.Resultado != ResultadoAceptado:
		if actual.Contenido.Clase != ClaseDecisionRevocacion {
			return ErrTransicionDecisionInvalida
		}
	case anterior.Contenido.Resultado != ResultadoAceptado && actual.Contenido.Resultado == ResultadoAceptado:
		if actual.Contenido.Clase != ClaseDecisionRehabilitacion {
			return ErrTransicionDecisionInvalida
		}
	case anterior.Contenido.Resultado != ResultadoAceptado && actual.Contenido.Resultado != ResultadoAceptado,
		anterior.Contenido.Resultado == ResultadoAceptado && actual.Contenido.Resultado == ResultadoAceptado:
		if actual.Contenido.Clase != ClaseDecisionRectificacion {
			return ErrTransicionDecisionInvalida
		}
	default:
		return ErrTransicionDecisionInvalida
	}
	return nil
}

func contenidoAdministrativoCambia(anterior, actual ContenidoDecisionTecnica) bool {
	if anterior.Resultado != actual.Resultado || !anterior.CalculoOficial.CoincideCon(actual.CalculoOficial) ||
		anterior.PuntosReconocidos != actual.PuntosReconocidos || anterior.MotivoClave != actual.MotivoClave ||
		anterior.Motivo != actual.Motivo || anterior.FinalidadClave != actual.FinalidadClave {
		return true
	}
	if !cadenasIguales(anterior.FuentesNormativasRefs, actual.FuentesNormativasRefs) {
		return true
	}
	return !valoracionesEvidenciaIguales(anterior.ValoracionesEvidencia, actual.ValoracionesEvidencia)
}

func canonizarReglaCalculo(regla ReferenciaReglaCalculo) ReferenciaReglaCalculo {
	regla.Clave = strings.TrimSpace(regla.Clave)
	regla.HuellaSHA256 = strings.TrimSpace(regla.HuellaSHA256)
	return regla
}

func canonizarCriterio(criterio ReferenciaCriterio) ReferenciaCriterio {
	criterio.ProcesoRef = strings.TrimSpace(criterio.ProcesoRef)
	criterio.Clave = strings.TrimSpace(criterio.Clave)
	criterio.HuellaSHA256 = strings.TrimSpace(criterio.HuellaSHA256)
	criterio.ReglaCalculo = canonizarReglaCalculo(criterio.ReglaCalculo)
	return criterio
}

func canonizarEvidencias(evidencias []EvidenciaMerito) ([]EvidenciaMerito, error) {
	if len(evidencias) == 0 || len(evidencias) > maximoEvidenciasPorMerito {
		return nil, ErrEvidenciaInvalida
	}
	canonicas := make([]EvidenciaMerito, len(evidencias))
	for indice, evidencia := range evidencias {
		canonica := evidencia
		canonica.Referencia = canonizarReferenciaEvidencia(evidencia.Referencia)
		if evidencia.SubsanacionDe != nil {
			referencia := canonizarReferenciaEvidencia(*evidencia.SubsanacionDe)
			canonica.SubsanacionDe = &referencia
		}
		if err := canonica.Validar(); err != nil {
			return nil, err
		}
		canonicas[indice] = canonica
	}
	ordenarEvidencias(canonicas)
	for indice := 1; indice < len(canonicas); indice++ {
		if referenciasEvidenciaIguales(canonicas[indice-1].Referencia, canonicas[indice].Referencia) {
			return nil, ErrEvidenciaInvalida
		}
	}
	return canonicas, nil
}

func canonizarValoracionesEvidencia(valoraciones []ValoracionEvidencia) ([]ValoracionEvidencia, error) {
	if len(valoraciones) == 0 || len(valoraciones) > maximoEvidenciasPorMerito {
		return nil, ErrValoracionEvidenciaInvalida
	}
	canonicas := make([]ValoracionEvidencia, len(valoraciones))
	for indice, valoracion := range valoraciones {
		evidencias, err := canonizarEvidencias([]EvidenciaMerito{valoracion.Evidencia})
		if err != nil {
			return nil, ErrValoracionEvidenciaInvalida
		}
		canonica := valoracion
		canonica.Evidencia = evidencias[0]
		canonica.MotivoClave = strings.TrimSpace(valoracion.MotivoClave)
		canonica.Motivo = strings.TrimSpace(valoracion.Motivo)
		if err := canonica.Validar(); err != nil {
			return nil, err
		}
		canonicas[indice] = canonica
	}
	ordenarValoraciones(canonicas)
	for indice := 1; indice < len(canonicas); indice++ {
		if referenciasEvidenciaIguales(canonicas[indice-1].Evidencia.Referencia, canonicas[indice].Evidencia.Referencia) {
			return nil, ErrValoracionEvidenciaInvalida
		}
	}
	return canonicas, nil
}

func canonizarReferencias(referencias []string) ([]string, error) {
	if len(referencias) == 0 || len(referencias) > 256 {
		return nil, ErrContenidoDecisionInvalido
	}
	canonicas := append([]string(nil), referencias...)
	for indice := range canonicas {
		canonicas[indice] = strings.TrimSpace(canonicas[indice])
		if !referenciaOpacaValida(canonicas[indice]) {
			return nil, ErrContenidoDecisionInvalido
		}
	}
	ordenarCadenas(canonicas)
	for indice := 1; indice < len(canonicas); indice++ {
		if canonicas[indice] == canonicas[indice-1] {
			return nil, ErrContenidoDecisionInvalido
		}
	}
	return canonicas, nil
}

func validarEvolucionEvidencias(anteriores, actuales []ValoracionEvidencia) error {
	previas, err := canonizarValoracionesEvidencia(anteriores)
	if err != nil {
		return err
	}
	nuevas, err := canonizarValoracionesEvidencia(actuales)
	if err != nil || len(nuevas) < len(previas) {
		return ErrTransicionDecisionInvalida
	}
	previasPorClave := make(map[string]EvidenciaMerito, len(previas))
	nuevasPorClave := make(map[string]EvidenciaMerito, len(nuevas))
	for _, valoracion := range previas {
		previasPorClave[claveReferenciaEvidencia(valoracion.Evidencia.Referencia)] = valoracion.Evidencia
	}
	for _, valoracion := range nuevas {
		nuevasPorClave[claveReferenciaEvidencia(valoracion.Evidencia.Referencia)] = valoracion.Evidencia
	}
	for clave := range previasPorClave {
		if _, existe := nuevasPorClave[clave]; !existe {
			return ErrTransicionDecisionInvalida
		}
	}
	for clave, evidencia := range nuevasPorClave {
		if previa, existia := previasPorClave[clave]; existia {
			if !evidenciasMeritoIguales(previa, evidencia) {
				return ErrTransicionDecisionInvalida
			}
			continue
		}
		if evidencia.SubsanacionDe == nil {
			return ErrTransicionDecisionInvalida
		}
		if _, existe := previasPorClave[claveReferenciaEvidencia(*evidencia.SubsanacionDe)]; !existe {
			return ErrTransicionDecisionInvalida
		}
	}
	return nil
}

func evidenciasDeValoraciones(valoraciones []ValoracionEvidencia) []EvidenciaMerito {
	evidencias := make([]EvidenciaMerito, len(valoraciones))
	for indice := range valoraciones {
		evidencias[indice] = valoraciones[indice].Evidencia
	}
	return evidencias
}

func mismoConjuntoEvidencias(izquierda, derecha []EvidenciaMerito) bool {
	a, err := canonizarEvidencias(izquierda)
	if err != nil {
		return false
	}
	b, err := canonizarEvidencias(derecha)
	if err != nil || len(a) != len(b) {
		return false
	}
	for indice := range a {
		if !evidenciasMeritoIguales(a[indice], b[indice]) {
			return false
		}
	}
	return true
}

func valoracionesEvidenciaIguales(izquierda, derecha []ValoracionEvidencia) bool {
	a, err := canonizarValoracionesEvidencia(izquierda)
	if err != nil {
		return false
	}
	b, err := canonizarValoracionesEvidencia(derecha)
	if err != nil || len(a) != len(b) {
		return false
	}
	for indice := range a {
		if !evidenciasMeritoIguales(a[indice].Evidencia, b[indice].Evidencia) ||
			a[indice].Estado != b[indice].Estado ||
			a[indice].ResultadoSubsanacion != b[indice].ResultadoSubsanacion ||
			a[indice].MotivoClave != b[indice].MotivoClave || a[indice].Motivo != b[indice].Motivo {
			return false
		}
	}
	return true
}

func evidenciasMeritoIguales(izquierda, derecha EvidenciaMerito) bool {
	if !referenciasEvidenciaIguales(izquierda.Referencia, derecha.Referencia) {
		return false
	}
	if izquierda.SubsanacionDe == nil || derecha.SubsanacionDe == nil {
		return izquierda.SubsanacionDe == nil && derecha.SubsanacionDe == nil
	}
	return referenciasEvidenciaIguales(*izquierda.SubsanacionDe, *derecha.SubsanacionDe)
}

func referenciasEvidenciaIguales(izquierda, derecha ReferenciaEvidencia) bool {
	return izquierda == derecha
}

func canonizarReferenciaEvidencia(referencia ReferenciaEvidencia) ReferenciaEvidencia {
	canonica := referencia
	canonica.DocumentoRef = strings.TrimSpace(referencia.DocumentoRef)
	canonica.RepresentacionRef = strings.TrimSpace(referencia.RepresentacionRef)
	canonica.HuellaSHA256 = strings.TrimSpace(referencia.HuellaSHA256)
	return canonica
}

func claveReferenciaEvidencia(referencia ReferenciaEvidencia) string {
	return referencia.DocumentoRef + "\x00" + referencia.RepresentacionRef + "\x00" +
		referencia.HuellaSHA256 + "\x00" + strconv.Itoa(referencia.VersionDocumento)
}

func contieneEstadoEvidencia(valoraciones []ValoracionEvidencia, estado EstadoValoracionEvidencia) bool {
	for _, valoracion := range valoraciones {
		if valoracion.Estado == estado {
			return true
		}
	}
	return false
}

func ordenarEvidencias(evidencias []EvidenciaMerito) {
	sort.Slice(evidencias, func(i, j int) bool {
		return claveReferenciaEvidencia(evidencias[i].Referencia) < claveReferenciaEvidencia(evidencias[j].Referencia)
	})
}

func ordenarValoraciones(valoraciones []ValoracionEvidencia) {
	sort.Slice(valoraciones, func(i, j int) bool {
		return claveReferenciaEvidencia(valoraciones[i].Evidencia.Referencia) < claveReferenciaEvidencia(valoraciones[j].Evidencia.Referencia)
	})
}

func ordenarCadenas(valores []string) {
	sort.Strings(valores)
}

func cadenasIguales(izquierda, derecha []string) bool {
	a, err := canonizarReferencias(izquierda)
	if err != nil {
		return false
	}
	b, err := canonizarReferencias(derecha)
	if err != nil || len(a) != len(b) {
		return false
	}
	for indice := range a {
		if a[indice] != b[indice] {
			return false
		}
	}
	return true
}

func huellaJSON(valor any) (string, error) {
	contenido, err := json.Marshal(valor)
	if err != nil {
		return "", err
	}
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:]), nil
}

func referenciaOpacaValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && len(valor) > 0 && len(valor) <= maximoCaracteresReferencia &&
		patronReferenciaOpaca.MatchString(valor)
}

func claveNegocioValida(valor string) bool {
	return valor == strings.TrimSpace(valor) && len(valor) <= 128 && patronClaveNegocio.MatchString(valor)
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != sha256.Size*2 || strings.ToLower(valor) != valor {
		return false
	}
	_, err := hex.DecodeString(valor)
	return err == nil
}

func huellaHMACSHA256DominioValida(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" &&
		claveNegocioValida(partes[1]) && huellaSHA256Valida(partes[2])
}

func referenciaYHuellaOpcionalesCoherentes(referencia, huella string) bool {
	referencia = strings.TrimSpace(referencia)
	huella = strings.TrimSpace(huella)
	if referencia == "" || huella == "" {
		return referencia == "" && huella == ""
	}
	return referenciaOpacaValida(referencia) && huellaSHA256Valida(huella)
}

func textoValido(valor string) bool {
	valor = strings.TrimSpace(valor)
	if valor == "" || len(valor) > maximoCaracteresTexto {
		return false
	}
	for _, caracter := range valor {
		if caracter < 32 && caracter != '\n' && caracter != '\r' && caracter != '\t' {
			return false
		}
	}
	return true
}
