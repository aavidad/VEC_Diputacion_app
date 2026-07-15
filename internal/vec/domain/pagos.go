package domain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var (
	ErrDineroCobroInvalido                  = errors.New("vec: importe de cobro invalido")
	ErrReferenciaTarifaCobroInvalida        = errors.New("vec: referencia de tarifa de cobro invalida")
	ErrOrdenCobroInvalida                   = errors.New("vec: orden de cobro invalida")
	ErrTransicionCobroInvalida              = errors.New("vec: transicion de cobro invalida")
	ErrEvidenciaCobroInvalida               = errors.New("vec: evidencia de cobro invalida")
	ErrEvidenciaCobroConflictiva            = errors.New("vec: evidencia de cobro conflictiva")
	ErrCoincidenciaCobroInvalida            = errors.New("vec: el resultado no coincide con la orden de cobro")
	ErrDatoTarjetaProhibido                 = errors.New("vec: dato de tarjeta prohibido")
	ErrDevolucionCobroInvalida              = errors.New("vec: devolucion de cobro invalida")
	ErrConciliacionCobroInvalida            = errors.New("vec: conciliacion de cobro invalida")
	ErrSerializacionEvidenciaCobroProhibida = errors.New("vec: serializacion de evidencia interna de cobro prohibida")
	ErrSerializacionOrdenCobroProhibida     = errors.New("vec: serializacion directa de orden de cobro prohibida")
	ErrContextoAutorizacionCobroInvalido    = errors.New("vec: contexto de autorizacion de cobro invalido")
	ErrComandoCobroInvalido                 = errors.New("vec: comando de cobro invalido")
	ErrSerializacionAutorizacionCobro       = errors.New("vec: serializacion directa de autorizacion de cobro prohibida")
)

const (
	maximoCaracteresConceptoCobro  = 280
	maximoHechosOrdenCobro         = 10_000
	vigenciaMaximaOrdenCobro       = 366 * 24 * time.Hour
	desfaseMaximoEvidenciaCobro    = 2 * time.Minute
	antiguedadMaximaEvidenciaCobro = 15 * time.Minute
	vigenciaMaximaUsoContextoCobro = time.Minute
	audienciaEvidenciaCobro        = "vec.cobros"
	versionEsquemaIntegridadCobro  = 1
	dominioHMACAltaCobro           = "pagos-v1"
	dominioHMACDevolucionCobro     = "devoluciones-v1"
)

var (
	monedaCobroValida          = regexp.MustCompile(`^[A-Z]{3}$`)
	idOrdenCobroOpaco          = regexp.MustCompile(`^cob_[A-Za-z0-9_-]{22,128}$`)
	idDevolucionOpaco          = regexp.MustCompile(`^dev_[A-Za-z0-9_-]{22,128}$`)
	idSesionCobroOpaco         = regexp.MustCompile(`^ses_[A-Za-z0-9_-]{22,128}$`)
	idAtestacionOpaco          = regexp.MustCompile(`^aut_[A-Za-z0-9_-]{22,128}$`)
	idPersonaCobroOpaco        = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
	idPerfilCobroOpaco         = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,128}$`)
	idRepresentacionCobroOpaca = regexp.MustCompile(`^rep_[A-Za-z0-9_-]{22,128}$`)
	huellaNulaCobro            = strings.Repeat("0", 64)
)

// DineroCobro representa dinero exclusivamente en la unidad menor de la
// moneda. No admite cero porque las exenciones y no sujeciones son decisiones
// administrativas, no cobros ficticios.
type DineroCobro struct {
	UnidadesMenores int64  `json:"unidades_menores"`
	Moneda          string `json:"moneda"`
}

func (d DineroCobro) Validar() error {
	if d.UnidadesMenores <= 0 || !monedaCobroValida.MatchString(d.Moneda) {
		return ErrDineroCobroInvalido
	}
	return nil
}

func (d DineroCobro) Igual(otro DineroCobro) bool {
	return d.UnidadesMenores == otro.UnidadesMenores && d.Moneda == otro.Moneda
}

// ReferenciaTarifaCobro fija la version exacta de la tarifa y su contenido.
// Una orden nunca se recalcula contra la version que resulte ser la ultima.
type ReferenciaTarifaCobro struct {
	TarifaID        string `json:"tarifa_id"`
	Version         int    `json:"version"`
	HuellaSHA256    string `json:"huella_sha256"`
	ReglaCalculoRef string `json:"regla_calculo_ref"`
}

func (r ReferenciaTarifaCobro) Validar() error {
	if !esClaveDocumentalCanonica(r.TarifaID) || r.Version < 1 ||
		!esSHA256(r.HuellaSHA256) || !referenciaCobroValida(r.ReglaCalculoRef) {
		return ErrReferenciaTarifaCobroInvalida
	}
	return nil
}

func (r ReferenciaTarifaCobro) Referencia() string {
	return fmt.Sprintf("tarifa:%s:v%d", r.TarifaID, r.Version)
}

type EstadoCobro string

const (
	EstadoCobroCreada               EstadoCobro = "creada"
	EstadoCobroEnviadaPasarela      EstadoCobro = "enviada_a_pasarela"
	EstadoCobroResultadoPendiente   EstadoCobro = "resultado_pendiente"
	EstadoCobroConfirmada           EstadoCobro = "confirmada"
	EstadoCobroConciliada           EstadoCobro = "conciliada"
	EstadoCobroRechazada            EstadoCobro = "rechazada"
	EstadoCobroCancelada            EstadoCobro = "cancelada"
	EstadoCobroCaducada             EstadoCobro = "caducada"
	EstadoCobroResultadoDesconocido EstadoCobro = "resultado_desconocido"
	EstadoCobroDevolucionSolicitada EstadoCobro = "devolucion_solicitada"
	EstadoCobroDevolucionRechazada  EstadoCobro = "devolucion_rechazada"
	EstadoCobroDevuelta             EstadoCobro = "devuelta"
	EstadoCobroDevolucionConciliada EstadoCobro = "devolucion_conciliada"
	EstadoCobroIncidenciaBloqueada  EstadoCobro = "incidencia_bloqueada"
)

func (e EstadoCobro) Valido() bool {
	switch e {
	case EstadoCobroCreada, EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente,
		EstadoCobroConfirmada, EstadoCobroConciliada, EstadoCobroRechazada,
		EstadoCobroCancelada, EstadoCobroCaducada, EstadoCobroResultadoDesconocido,
		EstadoCobroDevolucionSolicitada, EstadoCobroDevolucionRechazada,
		EstadoCobroDevuelta, EstadoCobroDevolucionConciliada, EstadoCobroIncidenciaBloqueada:
		return true
	default:
		return false
	}
}

type TipoHechoCobro string

const (
	HechoCobroOrdenCreada                    TipoHechoCobro = "orden_creada"
	HechoCobroOperacionEnviada               TipoHechoCobro = "operacion_enviada"
	HechoCobroResultadoPendiente             TipoHechoCobro = "resultado_pendiente"
	HechoCobroResultadoDesconocido           TipoHechoCobro = "resultado_desconocido"
	HechoCobroConfirmado                     TipoHechoCobro = "cobro_confirmado"
	HechoCobroRechazado                      TipoHechoCobro = "cobro_rechazado"
	HechoCobroCancelado                      TipoHechoCobro = "orden_cancelada"
	HechoCobroCaducado                       TipoHechoCobro = "orden_caducada"
	HechoCobroConciliado                     TipoHechoCobro = "cobro_conciliado"
	HechoCobroDevolucionSolicitada           TipoHechoCobro = "devolucion_solicitada"
	HechoCobroDevolucionResultadoPendiente   TipoHechoCobro = "devolucion_resultado_pendiente"
	HechoCobroDevolucionResultadoDesconocido TipoHechoCobro = "devolucion_resultado_desconocido"
	HechoCobroDevolucionRechazada            TipoHechoCobro = "devolucion_rechazada"
	HechoCobroDevuelto                       TipoHechoCobro = "cobro_devuelto"
	HechoCobroDevolucionConciliada           TipoHechoCobro = "devolucion_conciliada"
	HechoCobroIncidenciaDetectada            TipoHechoCobro = "incidencia_detectada"
	HechoCobroEvidenciaAdicional             TipoHechoCobro = "evidencia_adicional"
)

func (t TipoHechoCobro) Valido() bool {
	switch t {
	case HechoCobroOrdenCreada, HechoCobroOperacionEnviada, HechoCobroResultadoPendiente,
		HechoCobroResultadoDesconocido, HechoCobroConfirmado, HechoCobroRechazado,
		HechoCobroCancelado, HechoCobroCaducado, HechoCobroConciliado,
		HechoCobroDevolucionSolicitada, HechoCobroDevolucionResultadoPendiente,
		HechoCobroDevolucionResultadoDesconocido,
		HechoCobroDevolucionRechazada, HechoCobroDevuelto, HechoCobroDevolucionConciliada,
		HechoCobroIncidenciaDetectada, HechoCobroEvidenciaAdicional:
		return true
	default:
		return false
	}
}

// HechoCobro es una entrada probatoria de solo adicion. La proyeccion puede
// reconstruirse comprobando la secuencia y los estados anterior/posterior.
type HechoCobro struct {
	VersionEsquemaIntegridad    int                               `json:"version_esquema_integridad"`
	Secuencia                   int64                             `json:"secuencia"`
	Tipo                        TipoHechoCobro                    `json:"tipo"`
	EstadoAnterior              EstadoCobro                       `json:"estado_anterior,omitempty"`
	EstadoPosterior             EstadoCobro                       `json:"estado_posterior"`
	EvidenciaRef                string                            `json:"evidencia_ref"`
	EvidenciaRelacionadaRef     string                            `json:"evidencia_relacionada_ref,omitempty"`
	HuellaEvidenciaSHA256       string                            `json:"huella_evidencia_sha256"`
	HuellaMensajeOriginalSHA256 string                            `json:"huella_mensaje_original_sha256,omitempty"`
	IndiceIdempotenciaHMAC      string                            `json:"-"`
	ActorRef                    string                            `json:"actor_ref"`
	PerfilActivoRef             string                            `json:"perfil_activo_ref"`
	AccionAutorizada            AccionCobro                       `json:"accion_autorizada"`
	AutorizacionRef             string                            `json:"autorizacion_ref"`
	HuellaDecisionSHA256        string                            `json:"huella_decision_sha256"`
	AutorizacionEmitidaEn       time.Time                         `json:"autorizacion_emitida_en"`
	AutorizacionValidaHasta     time.Time                         `json:"autorizacion_valida_hasta"`
	AutorizacionEvaluadaEn      time.Time                         `json:"autorizacion_evaluada_en"`
	AtestacionAutenticacionRef  string                            `json:"atestacion_autenticacion_ref"`
	AtestacionEmitidaEn         time.Time                         `json:"atestacion_emitida_en"`
	AtestacionValidaHasta       time.Time                         `json:"atestacion_valida_hasta"`
	AutenticacionVerificadaEn   time.Time                         `json:"autenticacion_verificada_en"`
	SesionRef                   string                            `json:"sesion_ref"`
	HuellaSesionHMAC            string                            `json:"-"`
	MetodoAutenticacion         AuthMethod                        `json:"metodo_autenticacion"`
	GarantiaAutenticacion       AuthAssurance                     `json:"garantia_autenticacion"`
	CorrelacionRef              string                            `json:"correlacion_ref"`
	ConectorID                  string                            `json:"conector_id,omitempty"`
	VersionConector             int                               `json:"version_conector,omitempty"`
	OperacionProveedorRef       string                            `json:"operacion_proveedor_ref,omitempty"`
	DevolucionRef               string                            `json:"devolucion_ref,omitempty"`
	ConciliacionRef             string                            `json:"conciliacion_ref,omitempty"`
	Importe                     DineroCobro                       `json:"importe"`
	CodigoResultado             string                            `json:"codigo_resultado,omitempty"`
	VerificacionEvidenciaRef    string                            `json:"verificacion_evidencia_ref,omitempty"`
	HuellaVerificacionSHA256    string                            `json:"huella_verificacion_sha256,omitempty"`
	MetodoVerificacionEvidencia MetodoAutenticacionEvidenciaCobro `json:"metodo_verificacion_evidencia,omitempty"`
	AudienciaEvidencia          string                            `json:"audiencia_evidencia,omitempty"`
	EvidenciaEmitidaEn          time.Time                         `json:"evidencia_emitida_en,omitempty"`
	EvidenciaRecibidaEn         time.Time                         `json:"evidencia_recibida_en,omitempty"`
	EvidenciaVerificadaEn       time.Time                         `json:"evidencia_verificada_en,omitempty"`
	Motivo                      string                            `json:"motivo"`
	OcurridoEn                  time.Time                         `json:"ocurrido_en"`
	HuellaInstantaneaAltaSHA256 string                            `json:"huella_instantanea_alta_sha256,omitempty"`
	HuellaEstadoAnteriorSHA256  string                            `json:"huella_estado_anterior_sha256"`
	HuellaEstadoPosteriorSHA256 string                            `json:"huella_estado_posterior_sha256"`
}

func (h HechoCobro) Validar() error {
	if h.VersionEsquemaIntegridad != versionEsquemaIntegridadCobro || h.Secuencia < 1 ||
		!h.Tipo.Valido() || !h.EstadoPosterior.Valido() ||
		!referenciaCobroValida(h.EvidenciaRef) || !esSHA256(h.HuellaEvidenciaSHA256) ||
		!idPersonaCobroOpaco.MatchString(h.ActorRef) || !idPerfilCobroOpaco.MatchString(h.PerfilActivoRef) ||
		!h.AccionAutorizada.Valida() || !TuplaHechoCobroValida(h.Tipo, h.EstadoPosterior, h.AccionAutorizada) ||
		!referenciaCobroValida(h.AutorizacionRef) || !esSHA256(h.HuellaDecisionSHA256) ||
		h.AutorizacionEmitidaEn.IsZero() || h.AutorizacionValidaHasta.IsZero() ||
		!h.AutorizacionValidaHasta.After(h.AutorizacionEmitidaEn) ||
		h.AutorizacionEvaluadaEn.Before(h.AutorizacionEmitidaEn) ||
		!h.AutorizacionEvaluadaEn.Before(h.AutorizacionValidaHasta) ||
		!idAtestacionOpaco.MatchString(h.AtestacionAutenticacionRef) ||
		h.AtestacionEmitidaEn.IsZero() || h.AtestacionValidaHasta.IsZero() ||
		!h.AtestacionValidaHasta.After(h.AtestacionEmitidaEn) ||
		h.AutenticacionVerificadaEn.Before(h.AtestacionEmitidaEn) ||
		!h.AutenticacionVerificadaEn.Before(h.AtestacionValidaHasta) ||
		!idSesionCobroOpaco.MatchString(h.SesionRef) || !esHuellaSesionCobro(h.HuellaSesionHMAC) ||
		!metodoAutenticacionCobroPermitido(h.MetodoAutenticacion) ||
		!garantiaAutenticacionCobroPermitida(h.GarantiaAutenticacion) ||
		!referenciaCobroValida(h.CorrelacionRef) || h.Importe.Validar() != nil ||
		!textoCobroValido(h.Motivo, maximoCaracteresReferenciaDocumental) || h.OcurridoEn.IsZero() {
		return ErrEvidenciaCobroInvalida
	}
	if h.Secuencia == 1 {
		if h.Tipo != HechoCobroOrdenCreada || h.EstadoAnterior != "" || h.EstadoPosterior != EstadoCobroCreada {
			return ErrEvidenciaCobroInvalida
		}
	} else if !h.EstadoAnterior.Valido() {
		return ErrEvidenciaCobroInvalida
	}
	tieneConector := h.ConectorID != "" || h.VersionConector != 0 || h.OperacionProveedorRef != ""
	if tieneConector && (!esClaveDocumentalCanonica(h.ConectorID) || h.VersionConector < 1 ||
		!referenciaCobroValida(h.OperacionProveedorRef)) {
		return ErrEvidenciaCobroInvalida
	}
	if h.DevolucionRef != "" && !idDevolucionOpaco.MatchString(h.DevolucionRef) {
		return ErrEvidenciaCobroInvalida
	}
	if h.IndiceIdempotenciaHMAC != "" && !esHuellaHMACSHA256(h.IndiceIdempotenciaHMAC) {
		return ErrEvidenciaCobroInvalida
	}
	if h.ConciliacionRef != "" && !referenciaCobroValida(h.ConciliacionRef) {
		return ErrEvidenciaCobroInvalida
	}
	if h.CodigoResultado != "" && !textoCobroValido(h.CodigoResultado, 128) {
		return ErrEvidenciaCobroInvalida
	}
	if h.HuellaMensajeOriginalSHA256 != "" && !esSHA256(h.HuellaMensajeOriginalSHA256) {
		return ErrEvidenciaCobroInvalida
	}
	if h.EvidenciaRelacionadaRef != "" && !referenciaCobroValida(h.EvidenciaRelacionadaRef) {
		return ErrEvidenciaCobroInvalida
	}
	if (h.Tipo == HechoCobroIncidenciaDetectada) != (h.EvidenciaRelacionadaRef != "") {
		return ErrEvidenciaCobroInvalida
	}
	if h.Tipo == HechoCobroIncidenciaDetectada {
		if !strings.HasPrefix(h.EvidenciaRef, "incidencia:") || h.EvidenciaRelacionadaRef == h.EvidenciaRef ||
			strings.HasPrefix(h.EvidenciaRelacionadaRef, "incidencia:") {
			return ErrEvidenciaCobroInvalida
		}
	} else if strings.HasPrefix(h.EvidenciaRef, "incidencia:") {
		return ErrEvidenciaCobroInvalida
	}
	if !esSHA256(h.HuellaEstadoAnteriorSHA256) || !esSHA256(h.HuellaEstadoPosteriorSHA256) {
		return ErrEvidenciaCobroInvalida
	}
	if h.Tipo == HechoCobroOrdenCreada {
		if !esSHA256(h.HuellaInstantaneaAltaSHA256) {
			return ErrEvidenciaCobroInvalida
		}
	} else if h.HuellaInstantaneaAltaSHA256 != "" {
		return ErrEvidenciaCobroInvalida
	}
	esHechoDevolucion := h.Tipo == HechoCobroDevolucionSolicitada || h.Tipo == HechoCobroDevolucionResultadoPendiente ||
		h.Tipo == HechoCobroDevolucionResultadoDesconocido || h.Tipo == HechoCobroDevolucionRechazada ||
		h.Tipo == HechoCobroDevuelto || h.Tipo == HechoCobroDevolucionConciliada
	if h.Tipo != HechoCobroIncidenciaDetectada && h.Tipo != HechoCobroEvidenciaAdicional && esHechoDevolucion != (h.DevolucionRef != "") {
		return ErrEvidenciaCobroInvalida
	}
	esHechoConciliacion := h.Tipo == HechoCobroConciliado || h.Tipo == HechoCobroDevolucionConciliada
	if h.Tipo != HechoCobroIncidenciaDetectada && h.Tipo != HechoCobroEvidenciaAdicional &&
		esHechoConciliacion != (h.ConciliacionRef != "") {
		return ErrEvidenciaCobroInvalida
	}
	requiereIdempotencia := h.Tipo == HechoCobroOrdenCreada || h.Tipo == HechoCobroDevolucionSolicitada
	if requiereIdempotencia != (h.IndiceIdempotenciaHMAC != "") {
		return ErrEvidenciaCobroInvalida
	}
	if requiereIdempotencia {
		dominio := dominioHMACAltaCobro
		if h.Tipo == HechoCobroDevolucionSolicitada {
			dominio = dominioHMACDevolucionCobro
		}
		if !esHuellaHMACCobroDeDominio(h.IndiceIdempotenciaHMAC, dominio) {
			return ErrEvidenciaCobroInvalida
		}
	}
	if !matrizCamposHechoCobroValida(h) {
		return ErrEvidenciaCobroInvalida
	}
	return nil
}

func (HechoCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (*HechoCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (HechoCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (HechoCobro) String() string     { return "[HECHO-COBRO-INTERNO]" }
func (h HechoCobro) GoString() string { return h.String() }
func (h HechoCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, h.String())
}

type AltaOrdenCobro struct {
	ID                     string
	IndiceIdempotenciaHMAC string
	ExpedienteRef          string
	SolicitudRef           string
	LiquidacionRef         string
	Tarifa                 ReferenciaTarifaCobro
	SujetoRef              string
	RepresentacionRef      string
	Importe                DineroCobro
	Concepto               string
	Finalidad              string
	CorrelacionRef         string
	CreadaEn               time.Time
	CaducaEn               time.Time
	EvidenciaCreacionRef   string
	HuellaEvidenciaSHA256  string
	Motivo                 string
}

// BytesCanonicosIdempotenciaAltaCobro fija el significado funcional de una
// peticion antes de reservarla. Excluye identificadores y tiempos generados,
// pero incluye todos los datos que cambiarian el cobro.
func BytesCanonicosIdempotenciaAltaCobro(alta AltaOrdenCobro) ([]byte, error) {
	if alta.Tarifa.Validar() != nil || alta.Importe.Validar() != nil ||
		!referenciaCobroValida(alta.ExpedienteRef) || !referenciaCobroValida(alta.SolicitudRef) ||
		!referenciaCobroValida(alta.LiquidacionRef) || !idPersonaCobroOpaco.MatchString(alta.SujetoRef) ||
		(alta.RepresentacionRef != "" && !idRepresentacionCobroOpaca.MatchString(alta.RepresentacionRef)) ||
		!textoCobroValido(alta.Concepto, maximoCaracteresConceptoCobro) ||
		!textoCobroValido(alta.Finalidad, maximoCaracteresReferenciaDocumental) {
		return nil, ErrOrdenCobroInvalida
	}
	valor := struct {
		VersionEsquema                                                            int
		ExpedienteRef, SolicitudRef, LiquidacionRef, SujetoRef, RepresentacionRef string
		Tarifa                                                                    ReferenciaTarifaCobro
		Importe                                                                   DineroCobro
		Concepto, Finalidad                                                       string
	}{
		VersionEsquema: versionEsquemaIntegridadCobro,
		ExpedienteRef:  alta.ExpedienteRef, SolicitudRef: alta.SolicitudRef,
		LiquidacionRef: alta.LiquidacionRef, SujetoRef: alta.SujetoRef,
		RepresentacionRef: alta.RepresentacionRef, Tarifa: alta.Tarifa, Importe: alta.Importe,
		Concepto: alta.Concepto, Finalidad: alta.Finalidad,
	}
	bytes, err := json.Marshal(valor)
	if err != nil {
		return nil, ErrOrdenCobroInvalida
	}
	return append([]byte(nil), bytes...), nil
}

type AccionCobro string

const (
	AccionCobroCrearOrden          AccionCobro = "cobros.orden.crear"
	AccionCobroIniciarOperacion    AccionCobro = "cobros.operacion.iniciar"
	AccionCobroProcesarResultado   AccionCobro = "cobros.resultado.procesar"
	AccionCobroSolicitarDevolucion AccionCobro = "cobros.devolucion.solicitar"
	AccionCobroProcesarDevolucion  AccionCobro = "cobros.devolucion.procesar"
	AccionCobroConciliar           AccionCobro = "cobros.conciliar"
	AccionCobroCancelar            AccionCobro = "cobros.cancelar"
	AccionCobroCaducar             AccionCobro = "cobros.caducar"
)

func (a AccionCobro) Valida() bool {
	switch a {
	case AccionCobroCrearOrden, AccionCobroIniciarOperacion, AccionCobroProcesarResultado,
		AccionCobroSolicitarDevolucion, AccionCobroProcesarDevolucion, AccionCobroConciliar,
		AccionCobroCancelar, AccionCobroCaducar:
		return true
	default:
		return false
	}
}

type especificacionAutorizacionCobro struct {
	garantiaMinima AuthAssurance
	campos         []string
}

// especificacionesAutorizacionCobro es la unica lista positiva de alcance del
// modulo. Una decision sin todos estos campos, con uno adicional o emitida
// para una accion desconocida se deniega. Los nombres describen capacidades
// del dominio, no columnas de una base de datos concreta.
var especificacionesAutorizacionCobro = map[AccionCobro]especificacionAutorizacionCobro{
	AccionCobroCrearOrden: {
		garantiaMinima: AuthAssuranceSubstantial,
		campos:         []string{"auditoria", "evento_outbox", "orden.alta", "orden.historial"},
	},
	AccionCobroIniciarOperacion: {
		garantiaMinima: AuthAssuranceSubstantial,
		campos:         []string{"auditoria", "evento_outbox", "orden.historial", "orden.operacion_pasarela"},
	},
	AccionCobroProcesarResultado: {
		garantiaMinima: AuthAssuranceHigh,
		campos:         []string{"auditoria", "evento_outbox", "orden.historial", "orden.resultado"},
	},
	AccionCobroSolicitarDevolucion: {
		garantiaMinima: AuthAssuranceHigh,
		campos:         []string{"auditoria", "evento_outbox", "orden.devolucion", "orden.historial"},
	},
	AccionCobroProcesarDevolucion: {
		garantiaMinima: AuthAssuranceHigh,
		campos:         []string{"auditoria", "evento_outbox", "orden.devolucion", "orden.historial"},
	},
	AccionCobroConciliar: {
		garantiaMinima: AuthAssuranceHigh,
		campos:         []string{"auditoria", "evento_outbox", "orden.conciliacion", "orden.historial"},
	},
	AccionCobroCancelar: {
		garantiaMinima: AuthAssuranceHigh,
		campos:         []string{"auditoria", "evento_outbox", "orden.estado", "orden.historial"},
	},
	AccionCobroCaducar: {
		garantiaMinima: AuthAssuranceHigh,
		campos:         []string{"auditoria", "evento_outbox", "orden.estado", "orden.historial"},
	},
}

// CamposRequeridosAccionCobro permite configurar el motor de autorizacion con
// la misma lista cerrada que aplica el dominio. Devuelve siempre una copia.
func CamposRequeridosAccionCobro(accion AccionCobro) ([]string, bool) {
	especificacion, existe := especificacionesAutorizacionCobro[accion]
	if !existe {
		return nil, false
	}
	return append([]string(nil), especificacion.campos...), true
}

// ResultadoVerificacionAutenticacionCobro es la proyeccion minima que entrega
// una autoridad de identidad tras comprobar la sesion. No es una concesion de
// pago. La implementacion de VerificadorAutenticacionCobro es un limite de
// confianza y debe proceder del servicio de identidad, nunca de la peticion.
type ResultadoVerificacionAutenticacionCobro struct {
	PrincipalRef     string
	Metodo           AuthMethod
	Garantia         AuthAssurance
	AutenticacionRef string
	SesionRef        string
	HuellaSesionHMAC string
	EmitidaEn        time.Time
	ValidaHasta      time.Time
}

func (ResultadoVerificacionAutenticacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (*ResultadoVerificacionAutenticacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionCobro
}
func (ResultadoVerificacionAutenticacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (ResultadoVerificacionAutenticacionCobro) String() string {
	return "[RESULTADO-AUTENTICACION-COBRO-INTERNO]"
}
func (r ResultadoVerificacionAutenticacionCobro) GoString() string { return r.String() }
func (r ResultadoVerificacionAutenticacionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}

func (r ResultadoVerificacionAutenticacionCobro) validar(instante time.Time) error {
	if !idPersonaCobroOpaco.MatchString(r.PrincipalRef) || !metodoAutenticacionCobroPermitido(r.Metodo) ||
		!garantiaAutenticacionCobroPermitida(r.Garantia) || !idAtestacionOpaco.MatchString(r.AutenticacionRef) ||
		!idSesionCobroOpaco.MatchString(r.SesionRef) || !esHuellaSesionCobro(r.HuellaSesionHMAC) ||
		r.EmitidaEn.IsZero() || r.ValidaHasta.IsZero() || !r.ValidaHasta.After(r.EmitidaEn) ||
		instante.IsZero() || instante.UTC().Before(r.EmitidaEn.UTC()) || !instante.UTC().Before(r.ValidaHasta.UTC()) {
		return ErrContextoAutorizacionCobroInvalido
	}
	return nil
}

// SolicitudVerificacionAutenticacionCobro identifica una sesion por una
// referencia opaca y una huella HMAC de dominio separado. Nunca transporta el
// token, la cookie ni material de autenticacion reutilizable.
type SolicitudVerificacionAutenticacionCobro struct {
	SesionRef        string
	HuellaSesionHMAC string
	Instante         time.Time
}

func (SolicitudVerificacionAutenticacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (*SolicitudVerificacionAutenticacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionCobro
}
func (SolicitudVerificacionAutenticacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (SolicitudVerificacionAutenticacionCobro) String() string {
	return "[SOLICITUD-AUTENTICACION-COBRO-INTERNA]"
}
func (s SolicitudVerificacionAutenticacionCobro) GoString() string { return s.String() }
func (s SolicitudVerificacionAutenticacionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}

func (s SolicitudVerificacionAutenticacionCobro) validar() error {
	if !idSesionCobroOpaco.MatchString(s.SesionRef) || !esHuellaSesionCobro(s.HuellaSesionHMAC) || s.Instante.IsZero() {
		return ErrContextoAutorizacionCobroInvalido
	}
	return nil
}

// VerificadorAutenticacionCobro es el puerto minimo pendiente de cablear en el
// servicio de aplicacion con la identidad opaca ya validada. El dominio no
// puede demostrar el origen real de una implementacion inyectada: la raiz de
// composicion debe impedir verificadores de cabeceras o datos aportados por el
// llamador. Esta interfaz evita aceptar un nivel de garantia suelto como si
// fuera prueba suficiente, pero no finge resolver por si sola ese limite.
type VerificadorAutenticacionCobro interface {
	VerificarAutenticacionCobro(
		context.Context,
		SolicitudVerificacionAutenticacionCobro,
	) (ResultadoVerificacionAutenticacionCobro, error)
}

type datosAtestacionAutenticacionCobro struct {
	principalRef     string
	metodo           AuthMethod
	garantia         AuthAssurance
	autenticacionRef string
	sesionRef        string
	huellaSesionHMAC string
	emitidaEn        time.Time
	validaHasta      time.Time
	verificadaEn     time.Time
}

// AtestacionAutenticacionCobro es opaca y no serializable. Su valor cero
// deniega. Conserva el resultado de una verificacion de sesion, no datos
// recibidos directamente en la operacion de pago.
type AtestacionAutenticacionCobro struct {
	datos *datosAtestacionAutenticacionCobro
}

func NuevaAtestacionAutenticacionCobro(
	ctx context.Context,
	verificador VerificadorAutenticacionCobro,
	sesionRef, huellaSesionHMAC string,
	instante time.Time,
) (AtestacionAutenticacionCobro, error) {
	solicitud := SolicitudVerificacionAutenticacionCobro{
		SesionRef: sesionRef, HuellaSesionHMAC: huellaSesionHMAC, Instante: instante.UTC(),
	}
	if ctx == nil || verificador == nil || solicitud.validar() != nil {
		return AtestacionAutenticacionCobro{}, ErrContextoAutorizacionCobroInvalido
	}
	if err := ctx.Err(); err != nil {
		return AtestacionAutenticacionCobro{}, err
	}
	resultado, err := verificador.VerificarAutenticacionCobro(ctx, solicitud)
	if contextoErr := ctx.Err(); contextoErr != nil {
		return AtestacionAutenticacionCobro{}, contextoErr
	}
	if err != nil || resultado.validar(instante) != nil || resultado.SesionRef != solicitud.SesionRef ||
		resultado.HuellaSesionHMAC != solicitud.HuellaSesionHMAC {
		return AtestacionAutenticacionCobro{}, ErrContextoAutorizacionCobroInvalido
	}
	return AtestacionAutenticacionCobro{datos: &datosAtestacionAutenticacionCobro{
		principalRef: resultado.PrincipalRef, metodo: resultado.Metodo, garantia: resultado.Garantia,
		autenticacionRef: resultado.AutenticacionRef, sesionRef: resultado.SesionRef,
		huellaSesionHMAC: resultado.HuellaSesionHMAC, emitidaEn: resultado.EmitidaEn.UTC(),
		validaHasta: resultado.ValidaHasta.UTC(), verificadaEn: instante.UTC(),
	}}, nil
}

func (a AtestacionAutenticacionCobro) validar(instante time.Time) error {
	if a.datos == nil || instante.IsZero() {
		return ErrContextoAutorizacionCobroInvalido
	}
	d := a.datos
	resultado := ResultadoVerificacionAutenticacionCobro{
		PrincipalRef: d.principalRef, Metodo: d.metodo, Garantia: d.garantia,
		AutenticacionRef: d.autenticacionRef, SesionRef: d.sesionRef, HuellaSesionHMAC: d.huellaSesionHMAC,
		EmitidaEn: d.emitidaEn, ValidaHasta: d.validaHasta,
	}
	if resultado.validar(instante) != nil || !d.verificadaEn.Equal(instante.UTC()) {
		return ErrContextoAutorizacionCobroInvalido
	}
	return nil
}

func (AtestacionAutenticacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (*AtestacionAutenticacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionCobro
}
func (AtestacionAutenticacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (AtestacionAutenticacionCobro) String() string     { return "[ATESTACION-AUTENTICACION-COBRO]" }
func (a AtestacionAutenticacionCobro) GoString() string { return a.String() }
func (a AtestacionAutenticacionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}

// DatosContextoAutorizacionCobro es la proyeccion de solo lectura que permite
// auditar una autorizacion ya validada. No sirve para construirla.
type DatosContextoAutorizacionCobro struct {
	DecisionRef          string
	ActorRef             string
	PerfilActivoRef      string
	Accion               AccionCobro
	RecursoRef           string
	Finalidad            string
	CorrelacionRef       string
	Garantia             AuthAssurance
	Metodo               AuthMethod
	AutenticacionRef     string
	SesionRef            string
	HuellaSesionHMAC     string
	CamposPermitidos     []string
	HuellaDecisionSHA256 string
	VigenteDesde         time.Time
	VigenteHasta         time.Time
	EvaluadaEn           time.Time
}

// ContextoAutorizacionCobro es una capacidad opaca: el valor cero y los
// literales externos fallan cerrados. La huella interna liga la decision
// completa, pero acredita integridad dentro del proceso, no la procedencia
// criptografica de esa decision.
type ContextoAutorizacionCobro struct {
	datos *datosContextoAutorizacionCobro
}

type datosContextoAutorizacionCobro struct {
	DatosContextoAutorizacionCobro
	decision   DecisionAutorizacion
	atestacion datosAtestacionAutenticacionCobro
}

// NuevoContextoAutorizacionCobro aplica el contrato positivo y exacto. El
// futuro servicio de aplicacion debe invocarlo en la misma operacion en que
// obtiene la decision del Autorizador, el recurso resuelto por el servidor y la
// atestacion del servicio de identidad; nunca debe aceptar ninguno de esos
// valores desde HTTP, CLI o mensajeria. Mientras ese cableado no exista, este
// constructor no constituye por si solo una frontera de produccion infabricable.
func NuevoContextoAutorizacionCobro(
	decision DecisionAutorizacion,
	atestacion AtestacionAutenticacionCobro,
	recurso RecursoAutorizable,
	evaluadaEn time.Time,
) (ContextoAutorizacionCobro, error) {
	accion := AccionCobro(decision.Accion)
	especificacion, conocida := especificacionesAutorizacionCobro[accion]
	huellaContextoEsperada, errContexto := recurso.HuellaContextoAutorizacionSHA256()
	datosVinculo, errVinculo := decision.VinculoAutenticacionActor.Datos()
	if decision.ValidarEvidenciaInstantanea() != nil || !decision.Concedida || !decision.VigenteEn(evaluadaEn) ||
		!conocida || errContexto != nil || errVinculo != nil ||
		recurso.ModuloID != "pagos" || recurso.Tipo != "orden_cobro" ||
		decision.RecursoRef != recurso.Referencia || decision.ModuloID != recurso.ModuloID ||
		decision.TipoRecurso != recurso.Tipo || decision.ContextoRecursoHuellaSHA256 != huellaContextoEsperada ||
		atestacion.validar(evaluadaEn) != nil ||
		!idPersonaCobroOpaco.MatchString(decision.PrincipalID) ||
		!idPerfilCobroOpaco.MatchString(decision.PerfilActivoRef) ||
		atestacion.datos.principalRef != decision.PrincipalID ||
		atestacion.datos.autenticacionRef != datosVinculo.AutenticacionRef ||
		atestacion.datos.sesionRef != datosVinculo.SesionRef ||
		atestacion.datos.metodo != datosVinculo.MetodoObservado ||
		atestacion.datos.garantia != datosVinculo.GarantiaObservada ||
		decision.EmitidaEn.UTC().Before(atestacion.datos.emitidaEn) ||
		!atestacion.datos.garantia.Cumple(decision.GarantiaMinima) ||
		!decision.GarantiaMinima.Cumple(especificacion.garantiaMinima) ||
		len(decision.Obligaciones) != 0 ||
		!mismosCamposCobro(decision.CamposPermitidos, especificacion.campos) {
		return ContextoAutorizacionCobro{}, ErrContextoAutorizacionCobroInvalido
	}
	decisionCopia := clonarDecisionAutorizacionCobro(decision)
	huellaDecision, err := huellaDecisionAutorizacionCobro(decisionCopia)
	if err != nil {
		return ContextoAutorizacionCobro{}, ErrContextoAutorizacionCobroInvalido
	}
	vigenteHasta := decision.ValidaHasta.UTC()
	if atestacion.datos.validaHasta.Before(vigenteHasta) {
		vigenteHasta = atestacion.datos.validaHasta
	}
	limiteUso := evaluadaEn.UTC().Add(vigenciaMaximaUsoContextoCobro)
	if limiteUso.Before(vigenteHasta) {
		vigenteHasta = limiteUso
	}
	datos := DatosContextoAutorizacionCobro{
		DecisionRef: decision.DecisionRef, ActorRef: decision.PrincipalID,
		PerfilActivoRef: decision.PerfilActivoRef, Accion: accion, RecursoRef: decision.RecursoRef,
		Finalidad: decision.Finalidad, CorrelacionRef: decision.CorrelacionRef,
		Garantia: atestacion.datos.garantia, Metodo: atestacion.datos.metodo,
		AutenticacionRef:     atestacion.datos.autenticacionRef,
		SesionRef:            atestacion.datos.sesionRef,
		HuellaSesionHMAC:     atestacion.datos.huellaSesionHMAC,
		CamposPermitidos:     append([]string(nil), especificacion.campos...),
		HuellaDecisionSHA256: huellaDecision, VigenteDesde: decision.EmitidaEn.UTC(),
		VigenteHasta: vigenteHasta, EvaluadaEn: evaluadaEn.UTC(),
	}
	contexto := ContextoAutorizacionCobro{datos: &datosContextoAutorizacionCobro{
		DatosContextoAutorizacionCobro: datos, decision: decisionCopia,
		atestacion: *atestacion.datos,
	}}
	if err := contexto.validarEstructura(); err != nil {
		return ContextoAutorizacionCobro{}, err
	}
	return contexto, nil
}

func (c ContextoAutorizacionCobro) Datos() (DatosContextoAutorizacionCobro, error) {
	if err := c.validarEstructura(); err != nil {
		return DatosContextoAutorizacionCobro{}, err
	}
	resultado := c.datos.DatosContextoAutorizacionCobro
	resultado.CamposPermitidos = append([]string(nil), resultado.CamposPermitidos...)
	return resultado, nil
}

// CoincideExactamenteConDecision permite a un puerto de persistencia cruzar
// una evidencia reforzada con la decision inmutable que origino este contexto
// sin exponer esa decision interna. No compara huellas de esquemas distintos:
// vuelve a calcular dentro del dominio la misma representacion de cobros usada
// al crear el contexto. Cualquier diferencia, incluso en controles de rol,
// catalogo, sesion o actor, falla cerrada.
func (c ContextoAutorizacionCobro) CoincideExactamenteConDecision(
	decision DecisionAutorizacion,
) bool {
	if c.validarEstructura() != nil || decision.ValidarEvidenciaInstantanea() != nil {
		return false
	}
	huella, err := huellaDecisionAutorizacionCobro(decision)
	return err == nil && huella == c.datos.HuellaDecisionSHA256
}

func (ContextoAutorizacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (*ContextoAutorizacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionAutorizacionCobro
}
func (ContextoAutorizacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionAutorizacionCobro
}
func (ContextoAutorizacionCobro) String() string     { return "[AUTORIZACION-COBRO-INTERNA]" }
func (c ContextoAutorizacionCobro) GoString() string { return c.String() }
func (c ContextoAutorizacionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

type DatosComandoDevolucionCobro struct {
	OrdenRef                string
	VersionOrden            int
	HuellaOrdenSHA256       string
	ConectorID              string
	VersionConector         int
	OperacionProveedorRef   string
	DevolucionRef           string
	IndiceIdempotenciaHMAC  string
	Importe                 DineroCobro
	Motivo                  string
	DecisionAutorizacionRef string
	CorrelacionRef          string
}

type DatosComandoInicioOperacionCobro struct {
	OrdenRef                string
	VersionOrden            int
	HuellaOrdenSHA256       string
	LiquidacionRef          string
	IndiceIdempotenciaHMAC  string
	Importe                 DineroCobro
	Concepto                string
	CaducaEn                time.Time
	RetornoUsuarioRef       string
	NotificacionServidorRef string
	DecisionAutorizacionRef string
	CorrelacionRef          string
}

type ComandoInicioOperacionCobro struct {
	datos DatosComandoInicioOperacionCobro
}

func (c ComandoInicioOperacionCobro) Datos() (DatosComandoInicioOperacionCobro, error) {
	if err := c.Validar(); err != nil {
		return DatosComandoInicioOperacionCobro{}, err
	}
	return c.datos, nil
}

func (c ComandoInicioOperacionCobro) Validar() error {
	d := c.datos
	if !idOrdenCobroOpaco.MatchString(d.OrdenRef) || d.VersionOrden < 1 || !esSHA256(d.HuellaOrdenSHA256) ||
		!referenciaCobroValida(d.LiquidacionRef) ||
		!esHuellaHMACCobroDeDominio(d.IndiceIdempotenciaHMAC, dominioHMACAltaCobro) ||
		d.Importe.Validar() != nil || !textoCobroValido(d.Concepto, maximoCaracteresConceptoCobro) ||
		d.CaducaEn.IsZero() || !referenciaCobroValida(d.RetornoUsuarioRef) ||
		!referenciaCobroValida(d.NotificacionServidorRef) || !referenciaCobroValida(d.DecisionAutorizacionRef) ||
		!referenciaCobroValida(d.CorrelacionRef) {
		return ErrComandoCobroInvalido
	}
	return nil
}

type ComandoDevolucionCobro struct{ datos DatosComandoDevolucionCobro }

func (c ComandoDevolucionCobro) Datos() (DatosComandoDevolucionCobro, error) {
	if err := c.Validar(); err != nil {
		return DatosComandoDevolucionCobro{}, err
	}
	return c.datos, nil
}

func (c ComandoDevolucionCobro) Validar() error {
	d := c.datos
	if !idOrdenCobroOpaco.MatchString(d.OrdenRef) || d.VersionOrden < 1 || !esSHA256(d.HuellaOrdenSHA256) ||
		!esClaveDocumentalCanonica(d.ConectorID) || d.VersionConector < 1 ||
		!referenciaCobroValida(d.OperacionProveedorRef) || !idDevolucionOpaco.MatchString(d.DevolucionRef) ||
		!esHuellaHMACCobroDeDominio(d.IndiceIdempotenciaHMAC, dominioHMACDevolucionCobro) || d.Importe.Validar() != nil ||
		!textoCobroValido(d.Motivo, maximoCaracteresReferenciaDocumental) ||
		!referenciaCobroValida(d.DecisionAutorizacionRef) || !referenciaCobroValida(d.CorrelacionRef) {
		return ErrComandoCobroInvalido
	}
	return nil
}

type DatosComandoConciliacionCobro struct {
	OrdenRef                string
	VersionOrden            int
	HuellaOrdenSHA256       string
	ConectorID              string
	VersionConector         int
	OperacionProveedorRef   string
	DevolucionRef           string
	Tipo                    TipoConciliacionCobro
	Importe                 DineroCobro
	ReferenciaCierre        string
	DecisionAutorizacionRef string
	CorrelacionRef          string
}

type ComandoConciliacionCobro struct{ datos DatosComandoConciliacionCobro }

func (c ComandoConciliacionCobro) Datos() (DatosComandoConciliacionCobro, error) {
	if err := c.Validar(); err != nil {
		return DatosComandoConciliacionCobro{}, err
	}
	return c.datos, nil
}

func (c ComandoConciliacionCobro) Validar() error {
	d := c.datos
	if !idOrdenCobroOpaco.MatchString(d.OrdenRef) || d.VersionOrden < 1 || !esSHA256(d.HuellaOrdenSHA256) ||
		!esClaveDocumentalCanonica(d.ConectorID) || d.VersionConector < 1 ||
		!referenciaCobroValida(d.OperacionProveedorRef) || !d.Tipo.Valido() || d.Importe.Validar() != nil ||
		!referenciaCobroValida(d.ReferenciaCierre) || !referenciaCobroValida(d.DecisionAutorizacionRef) ||
		!referenciaCobroValida(d.CorrelacionRef) ||
		(d.Tipo == TipoConciliacionCobroDevolucion && !idDevolucionOpaco.MatchString(d.DevolucionRef)) ||
		(d.Tipo == TipoConciliacionCobroIngreso && d.DevolucionRef != "") {
		return ErrComandoCobroInvalido
	}
	return nil
}

func (ComandoDevolucionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (ComandoConciliacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (ComandoInicioOperacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (*ComandoDevolucionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (*ComandoConciliacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (*ComandoInicioOperacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (ComandoDevolucionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (ComandoConciliacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (ComandoInicioOperacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (ComandoDevolucionCobro) String() string          { return "[COMANDO-DEVOLUCION-INTERNO]" }
func (ComandoConciliacionCobro) String() string        { return "[COMANDO-CONCILIACION-INTERNO]" }
func (ComandoInicioOperacionCobro) String() string     { return "[COMANDO-INICIO-INTERNO]" }
func (c ComandoDevolucionCobro) GoString() string      { return c.String() }
func (c ComandoConciliacionCobro) GoString() string    { return c.String() }
func (c ComandoInicioOperacionCobro) GoString() string { return c.String() }
func (c ComandoDevolucionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ComandoConciliacionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}
func (c ComandoInicioOperacionCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (c ContextoAutorizacionCobro) validarEstructura() error {
	if c.datos == nil {
		return ErrContextoAutorizacionCobroInvalido
	}
	d := c.datos
	especificacion, conocida := especificacionesAutorizacionCobro[d.Accion]
	huellaDecision, err := huellaDecisionAutorizacionCobro(d.decision)
	if !conocida || err != nil || huellaDecision != d.HuellaDecisionSHA256 ||
		(AtestacionAutenticacionCobro{datos: &d.atestacion}).validar(d.EvaluadaEn) != nil ||
		d.atestacion.principalRef != d.ActorRef || d.atestacion.metodo != d.Metodo ||
		d.atestacion.garantia != d.Garantia || d.atestacion.autenticacionRef != d.AutenticacionRef ||
		d.atestacion.sesionRef != d.SesionRef || d.atestacion.huellaSesionHMAC != d.HuellaSesionHMAC ||
		d.VigenteHasta.After(d.atestacion.validaHasta) || d.VigenteDesde.Before(d.atestacion.emitidaEn) ||
		d.decision.ValidarEvidenciaInstantanea() != nil || !d.decision.Concedida ||
		d.decision.DecisionRef != d.DecisionRef || d.decision.PrincipalID != d.ActorRef ||
		d.decision.PerfilActivoRef != d.PerfilActivoRef || d.decision.Accion != string(d.Accion) ||
		d.decision.RecursoRef != d.RecursoRef || d.decision.Finalidad != d.Finalidad ||
		d.decision.CorrelacionRef != d.CorrelacionRef || len(d.decision.Obligaciones) != 0 ||
		d.decision.ModuloID != "pagos" || d.decision.TipoRecurso != "orden_cobro" ||
		!mismosCamposCobro(d.decision.CamposPermitidos, especificacion.campos) ||
		!mismosCamposCobro(d.CamposPermitidos, especificacion.campos) ||
		!d.Garantia.Cumple(d.decision.GarantiaMinima) ||
		!d.decision.GarantiaMinima.Cumple(especificacion.garantiaMinima) ||
		!d.Accion.Valida() || !referenciaCobroValida(d.DecisionRef) ||
		!idPersonaCobroOpaco.MatchString(d.ActorRef) || !idPerfilCobroOpaco.MatchString(d.PerfilActivoRef) ||
		!referenciaCobroValida(d.RecursoRef) ||
		!textoCobroValido(d.Finalidad, maximoCaracteresReferenciaDocumental) ||
		!referenciaCobroValida(d.CorrelacionRef) || !garantiaAutenticacionCobroPermitida(d.Garantia) ||
		!metodoAutenticacionCobroPermitido(d.Metodo) || !idAtestacionOpaco.MatchString(d.AutenticacionRef) ||
		!idSesionCobroOpaco.MatchString(d.SesionRef) || !esHuellaSesionCobro(d.HuellaSesionHMAC) ||
		!esSHA256(d.HuellaDecisionSHA256) ||
		d.VigenteDesde.IsZero() || d.VigenteHasta.IsZero() ||
		!d.VigenteHasta.After(d.VigenteDesde) ||
		d.VigenteHasta.Sub(d.EvaluadaEn) > vigenciaMaximaUsoContextoCobro ||
		d.EvaluadaEn.Before(d.VigenteDesde) || !d.EvaluadaEn.Before(d.VigenteHasta) ||
		!d.decision.EmitidaEn.UTC().Equal(d.VigenteDesde) || d.VigenteHasta.After(d.decision.ValidaHasta.UTC()) {
		return ErrContextoAutorizacionCobroInvalido
	}
	return nil
}

// ValidarEn comprueba el alcance y la vigencia en el instante efectivo de la
// operacion. No existe una variante sin tiempo: reutilizar la hora de emision
// permitiria aceptar capacidades caducadas desde una cache. El instante debe
// proceder del reloj confiable del servidor, nunca de la peticion.
func (c ContextoAutorizacionCobro) ValidarEn(accion AccionCobro, recurso, finalidad, correlacion string, instante time.Time) error {
	especificacion, conocida := especificacionesAutorizacionCobro[accion]
	if c.datos == nil || !conocida || instante.IsZero() {
		return ErrContextoAutorizacionCobroInvalido
	}
	d := c.datos
	instante = instante.UTC()
	if c.validarEstructura() != nil || !accion.Valida() || d.Accion != accion ||
		d.RecursoRef != recurso || d.Finalidad != finalidad || d.CorrelacionRef != correlacion ||
		!d.Garantia.Cumple(especificacion.garantiaMinima) || instante.Before(d.EvaluadaEn) ||
		!instante.Before(d.VigenteHasta) || instante.Sub(d.EvaluadaEn) > vigenciaMaximaUsoContextoCobro {
		return ErrContextoAutorizacionCobroInvalido
	}
	return nil
}

func mismosCamposCobro(recibidos, requeridos []string) bool {
	if len(recibidos) == 0 || len(recibidos) != len(requeridos) {
		return false
	}
	vistos := make(map[string]struct{}, len(recibidos))
	for _, campo := range recibidos {
		if !esClaveDocumentalCanonica(campo) {
			return false
		}
		if _, repetido := vistos[campo]; repetido {
			return false
		}
		vistos[campo] = struct{}{}
	}
	for _, requerido := range requeridos {
		if _, existe := vistos[requerido]; !existe {
			return false
		}
	}
	return true
}

func clonarDecisionAutorizacionCobro(decision DecisionAutorizacion) DecisionAutorizacion {
	copia := decision
	copia.PoliticasEvaluadasRefs = append([]string(nil), decision.PoliticasEvaluadasRefs...)
	copia.PoliticasRefs = append([]string(nil), decision.PoliticasRefs...)
	copia.CamposPermitidos = append([]string(nil), decision.CamposPermitidos...)
	copia.Obligaciones = append([]string(nil), decision.Obligaciones...)
	if decision.PoliticasHuellasSHA256 != nil {
		copia.PoliticasHuellasSHA256 = make(map[string]string, len(decision.PoliticasHuellasSHA256))
		for referencia, huella := range decision.PoliticasHuellasSHA256 {
			copia.PoliticasHuellasSHA256[referencia] = huella
		}
	}
	if decision.PoliticasEvaluadasHuellasSHA256 != nil {
		copia.PoliticasEvaluadasHuellasSHA256 = make(map[string]string, len(decision.PoliticasEvaluadasHuellasSHA256))
		for referencia, huella := range decision.PoliticasEvaluadasHuellasSHA256 {
			copia.PoliticasEvaluadasHuellasSHA256[referencia] = huella
		}
	}
	return copia
}

func huellaDecisionAutorizacionCobro(decision DecisionAutorizacion) (string, error) {
	if decision.ValidarEvidenciaInstantanea() != nil {
		return "", ErrContextoAutorizacionCobroInvalido
	}
	bytes, err := json.Marshal(decision)
	if err != nil {
		return "", ErrContextoAutorizacionCobroInvalido
	}
	huella := sha256.Sum256(bytes)
	return hex.EncodeToString(huella[:]), nil
}

// OrdenCobro no contiene PAN, CVV, PIN, criptogramas ni cargas opacas del
// proveedor. Solo conserva referencias y evidencias verificadas.
type OrdenCobro struct {
	VersionEsquemaIntegridad         int                   `json:"-"`
	ID                               string                `json:"id"`
	Version                          int                   `json:"version"`
	IndiceIdempotenciaHMAC           string                `json:"-"`
	ExpedienteRef                    string                `json:"expediente_ref"`
	SolicitudRef                     string                `json:"solicitud_ref"`
	LiquidacionRef                   string                `json:"liquidacion_ref"`
	Tarifa                           ReferenciaTarifaCobro `json:"tarifa"`
	SujetoRef                        string                `json:"sujeto_ref"`
	RepresentacionRef                string                `json:"representacion_ref,omitempty"`
	Importe                          DineroCobro           `json:"importe"`
	Concepto                         string                `json:"concepto"`
	Finalidad                        string                `json:"finalidad"`
	CorrelacionRef                   string                `json:"correlacion_ref"`
	Estado                           EstadoCobro           `json:"estado"`
	ConectorID                       string                `json:"conector_id,omitempty"`
	VersionConector                  int                   `json:"version_conector,omitempty"`
	OperacionProveedorRef            string                `json:"operacion_proveedor_ref,omitempty"`
	CreadaEn                         time.Time             `json:"creada_en"`
	CaducaEn                         time.Time             `json:"caduca_en"`
	ConfirmadaEn                     time.Time             `json:"confirmada_en,omitempty"`
	ConciliadaEn                     time.Time             `json:"conciliada_en,omitempty"`
	ConciliacionRef                  string                `json:"conciliacion_ref,omitempty"`
	DevolucionRef                    string                `json:"devolucion_ref,omitempty"`
	IndiceIdempotenciaDevolucionHMAC string                `json:"-"`
	DevolucionSolicitadaEn           time.Time             `json:"devolucion_solicitada_en,omitempty"`
	DevueltaEn                       time.Time             `json:"devuelta_en,omitempty"`
	DevolucionConciliadaEn           time.Time             `json:"devolucion_conciliada_en,omitempty"`
	DevolucionConciliacionRef        string                `json:"devolucion_conciliacion_ref,omitempty"`
	UltimaActualizacionEn            time.Time             `json:"ultima_actualizacion_en"`
	Historial                        []HechoCobro          `json:"historial"`
	HuellaInstantaneaAltaSHA256      string                `json:"-"`
	HuellaEstadoSHA256               string                `json:"-"`
}

type instantaneaAltaOrdenCobro struct {
	VersionEsquemaIntegridad int
	ID                       string
	IndiceIdempotenciaHMAC   string
	ExpedienteRef            string
	SolicitudRef             string
	LiquidacionRef           string
	Tarifa                   ReferenciaTarifaCobro
	SujetoRef                string
	RepresentacionRef        string
	Importe                  DineroCobro
	Concepto                 string
	Finalidad                string
	CorrelacionRef           string
	CreadaEn                 string
	CaducaEn                 string
}

func (o OrdenCobro) calcularHuellaInstantaneaAlta() (string, error) {
	return huellaCanonicaCobro(instantaneaAltaOrdenCobro{
		VersionEsquemaIntegridad: o.VersionEsquemaIntegridad,
		ID:                       o.ID, IndiceIdempotenciaHMAC: o.IndiceIdempotenciaHMAC,
		ExpedienteRef: o.ExpedienteRef, SolicitudRef: o.SolicitudRef, LiquidacionRef: o.LiquidacionRef,
		Tarifa: o.Tarifa, SujetoRef: o.SujetoRef, RepresentacionRef: o.RepresentacionRef,
		Importe: o.Importe, Concepto: o.Concepto, Finalidad: o.Finalidad, CorrelacionRef: o.CorrelacionRef,
		CreadaEn: o.CreadaEn.UTC().Format(time.RFC3339Nano), CaducaEn: o.CaducaEn.UTC().Format(time.RFC3339Nano),
	})
}

func calcularHuellaHechoCobro(hecho HechoCobro) (string, error) {
	copia := hecho
	copia.HuellaEstadoPosteriorSHA256 = ""
	// Las HMAC se omiten en la representacion JSON publica para evitar
	// filtraciones, pero forman parte obligatoria de la cadena probatoria.
	return huellaCanonicaCobro(struct {
		Hecho                  hechoCobroCanonico
		IndiceIdempotenciaHMAC string
		HuellaSesionHMAC       string
	}{
		Hecho: hechoCobroCanonico(copia), IndiceIdempotenciaHMAC: hecho.IndiceIdempotenciaHMAC,
		HuellaSesionHMAC: hecho.HuellaSesionHMAC,
	})
}

type hechoCobroCanonico HechoCobro

func huellaCanonicaCobro(valor any) (string, error) {
	bytes, err := json.Marshal(valor)
	if err != nil {
		return "", err
	}
	huella := sha256.Sum256(bytes)
	return hex.EncodeToString(huella[:]), nil
}

func NuevaOrdenCobro(alta AltaOrdenCobro, autorizacion ContextoAutorizacionCobro) (OrdenCobro, error) {
	if contieneDatoTarjetaCobro(alta.Concepto, alta.Finalidad, alta.Motivo, alta.LiquidacionRef) {
		return OrdenCobro{}, ErrDatoTarjetaProhibido
	}
	if !idOrdenCobroOpaco.MatchString(alta.ID) || !esHuellaHMACCobroDeDominio(alta.IndiceIdempotenciaHMAC, dominioHMACAltaCobro) ||
		!referenciaCobroValida(alta.ExpedienteRef) || !referenciaCobroValida(alta.SolicitudRef) ||
		!referenciaCobroValida(alta.LiquidacionRef) || alta.Tarifa.Validar() != nil ||
		!idPersonaCobroOpaco.MatchString(alta.SujetoRef) ||
		(alta.RepresentacionRef != "" && !idRepresentacionCobroOpaca.MatchString(alta.RepresentacionRef)) ||
		alta.Importe.Validar() != nil || !textoCobroValido(alta.Concepto, maximoCaracteresConceptoCobro) ||
		!textoCobroValido(alta.Finalidad, maximoCaracteresReferenciaDocumental) ||
		!referenciaCobroValida(alta.CorrelacionRef) ||
		alta.CreadaEn.IsZero() || !alta.CaducaEn.After(alta.CreadaEn) || alta.CaducaEn.Sub(alta.CreadaEn) > vigenciaMaximaOrdenCobro ||
		!referenciaEvidenciaEntradaCobroValida(alta.EvidenciaCreacionRef) || !esSHA256(alta.HuellaEvidenciaSHA256) ||
		!textoCobroValido(alta.Motivo, maximoCaracteresReferenciaDocumental) {
		return OrdenCobro{}, ErrOrdenCobroInvalida
	}
	if err := autorizacion.ValidarEn(AccionCobroCrearOrden, alta.LiquidacionRef, alta.Finalidad, alta.CorrelacionRef, alta.CreadaEn); err != nil ||
		autorizacion.datos.EvaluadaEn.After(alta.CreadaEn) || alta.CreadaEn.Sub(autorizacion.datos.EvaluadaEn) > time.Minute {
		return OrdenCobro{}, ErrContextoAutorizacionCobroInvalido
	}
	alta.CreadaEn = alta.CreadaEn.UTC()
	alta.CaducaEn = alta.CaducaEn.UTC()
	orden := OrdenCobro{
		VersionEsquemaIntegridad: versionEsquemaIntegridadCobro,
		ID:                       alta.ID,
		Version:                  1,
		IndiceIdempotenciaHMAC:   alta.IndiceIdempotenciaHMAC,
		ExpedienteRef:            alta.ExpedienteRef,
		SolicitudRef:             alta.SolicitudRef,
		LiquidacionRef:           alta.LiquidacionRef,
		Tarifa:                   alta.Tarifa,
		SujetoRef:                alta.SujetoRef,
		RepresentacionRef:        alta.RepresentacionRef,
		Importe:                  alta.Importe,
		Concepto:                 alta.Concepto,
		Finalidad:                alta.Finalidad,
		CorrelacionRef:           alta.CorrelacionRef,
		Estado:                   EstadoCobroCreada,
		CreadaEn:                 alta.CreadaEn,
		CaducaEn:                 alta.CaducaEn,
		UltimaActualizacionEn:    alta.CreadaEn,
	}
	huellaAlta, err := orden.calcularHuellaInstantaneaAlta()
	if err != nil {
		return OrdenCobro{}, ErrOrdenCobroInvalida
	}
	hecho := HechoCobro{
		VersionEsquemaIntegridad: versionEsquemaIntegridadCobro,
		Secuencia:                1, Tipo: HechoCobroOrdenCreada, EstadoPosterior: EstadoCobroCreada,
		EvidenciaRef: alta.EvidenciaCreacionRef, HuellaEvidenciaSHA256: alta.HuellaEvidenciaSHA256,
		IndiceIdempotenciaHMAC: alta.IndiceIdempotenciaHMAC, ActorRef: autorizacion.datos.ActorRef,
		PerfilActivoRef: autorizacion.datos.PerfilActivoRef, AccionAutorizada: autorizacion.datos.Accion,
		AutorizacionRef:            autorizacion.datos.DecisionRef,
		HuellaDecisionSHA256:       autorizacion.datos.HuellaDecisionSHA256,
		AutorizacionEmitidaEn:      autorizacion.datos.VigenteDesde,
		AutorizacionValidaHasta:    autorizacion.datos.VigenteHasta,
		AutorizacionEvaluadaEn:     autorizacion.datos.EvaluadaEn,
		AtestacionAutenticacionRef: autorizacion.datos.AutenticacionRef,
		AtestacionEmitidaEn:        autorizacion.datos.atestacion.emitidaEn,
		AtestacionValidaHasta:      autorizacion.datos.atestacion.validaHasta,
		AutenticacionVerificadaEn:  autorizacion.datos.atestacion.verificadaEn,
		SesionRef:                  autorizacion.datos.SesionRef, HuellaSesionHMAC: autorizacion.datos.HuellaSesionHMAC,
		MetodoAutenticacion: autorizacion.datos.Metodo, GarantiaAutenticacion: autorizacion.datos.Garantia,
		CorrelacionRef: alta.CorrelacionRef,
		Importe:        alta.Importe, Motivo: alta.Motivo, OcurridoEn: alta.CreadaEn,
		HuellaInstantaneaAltaSHA256: huellaAlta, HuellaEstadoAnteriorSHA256: huellaNulaCobro,
	}
	huellaEstado, err := calcularHuellaHechoCobro(hecho)
	if err != nil {
		return OrdenCobro{}, ErrOrdenCobroInvalida
	}
	hecho.HuellaEstadoPosteriorSHA256 = huellaEstado
	orden.Historial = []HechoCobro{hecho}
	orden.HuellaInstantaneaAltaSHA256 = huellaAlta
	orden.HuellaEstadoSHA256 = huellaEstado
	if err := orden.Validar(); err != nil {
		return OrdenCobro{}, err
	}
	return orden, nil
}

func (o OrdenCobro) Validar() error {
	if o.VersionEsquemaIntegridad != versionEsquemaIntegridadCobro ||
		!idOrdenCobroOpaco.MatchString(o.ID) || o.Version < 1 || !esHuellaHMACCobroDeDominio(o.IndiceIdempotenciaHMAC, dominioHMACAltaCobro) ||
		!referenciaCobroValida(o.ExpedienteRef) || !referenciaCobroValida(o.SolicitudRef) ||
		!referenciaCobroValida(o.LiquidacionRef) || o.Tarifa.Validar() != nil ||
		!idPersonaCobroOpaco.MatchString(o.SujetoRef) ||
		(o.RepresentacionRef != "" && !idRepresentacionCobroOpaca.MatchString(o.RepresentacionRef)) ||
		o.Importe.Validar() != nil || !textoCobroValido(o.Concepto, maximoCaracteresConceptoCobro) ||
		!textoCobroValido(o.Finalidad, maximoCaracteresReferenciaDocumental) ||
		!referenciaCobroValida(o.CorrelacionRef) || !o.Estado.Valido() || o.CreadaEn.IsZero() ||
		!o.CaducaEn.After(o.CreadaEn) || o.CaducaEn.Sub(o.CreadaEn) > vigenciaMaximaOrdenCobro ||
		o.UltimaActualizacionEn.Before(o.CreadaEn) ||
		len(o.Historial) == 0 || len(o.Historial) > maximoHechosOrdenCobro || o.Version != len(o.Historial) ||
		contieneDatoTarjetaCobro(o.Concepto, o.Finalidad, o.LiquidacionRef) {
		return ErrOrdenCobroInvalida
	}
	huellaAlta, err := o.calcularHuellaInstantaneaAlta()
	if err != nil || huellaAlta != o.HuellaInstantaneaAltaSHA256 || !esSHA256(o.HuellaEstadoSHA256) {
		return ErrOrdenCobroInvalida
	}
	estado := EstadoCobro("")
	instante := time.Time{}
	huellaEstado := huellaNulaCobro
	vistas := make(map[string]string, len(o.Historial))
	conectorID := ""
	versionConector := 0
	operacionProveedorRef := ""
	confirmadaEn := time.Time{}
	conciliadaEn := time.Time{}
	conciliacionRef := ""
	devolucionRef := ""
	indiceIdempotenciaDevolucionHMAC := ""
	devolucionSolicitadaEn := time.Time{}
	devueltaEn := time.Time{}
	devolucionConciliadaEn := time.Time{}
	devolucionConciliacionRef := ""
	for indice, hecho := range o.Historial {
		if err := hecho.Validar(); err != nil || hecho.Secuencia != int64(indice+1) ||
			hecho.EstadoAnterior != estado || (!instante.IsZero() && hecho.OcurridoEn.Before(instante)) ||
			!transicionHechoCobroPermitida(estado, hecho.EstadoPosterior, hecho.Tipo) || !hecho.Importe.Igual(o.Importe) ||
			hecho.HuellaEstadoAnteriorSHA256 != huellaEstado {
			return ErrOrdenCobroInvalida
		}
		if indice == 0 && hecho.HuellaInstantaneaAltaSHA256 != huellaAlta {
			return ErrOrdenCobroInvalida
		}
		huellaCalculada, err := calcularHuellaHechoCobro(hecho)
		if err != nil || huellaCalculada != hecho.HuellaEstadoPosteriorSHA256 {
			return ErrOrdenCobroInvalida
		}
		if _, existe := vistas[hecho.EvidenciaRef]; existe {
			return ErrOrdenCobroInvalida
		}
		vistas[hecho.EvidenciaRef] = hecho.HuellaEvidenciaSHA256
		if hecho.Tipo == HechoCobroOperacionEnviada {
			conectorID, versionConector, operacionProveedorRef = hecho.ConectorID, hecho.VersionConector, hecho.OperacionProveedorRef
		} else if hechoCobroEsRemoto(hecho) && conectorID != "" && (hecho.ConectorID != conectorID || hecho.VersionConector != versionConector ||
			hecho.OperacionProveedorRef != operacionProveedorRef) {
			return ErrOrdenCobroInvalida
		}
		switch hecho.Tipo {
		case HechoCobroOrdenCreada:
			if hecho.IndiceIdempotenciaHMAC != o.IndiceIdempotenciaHMAC {
				return ErrOrdenCobroInvalida
			}
		case HechoCobroConfirmado:
			confirmadaEn = hecho.OcurridoEn
		case HechoCobroConciliado:
			conciliadaEn, conciliacionRef = hecho.OcurridoEn, hecho.ConciliacionRef
		case HechoCobroDevolucionSolicitada:
			devolucionRef, devolucionSolicitadaEn = hecho.DevolucionRef, hecho.OcurridoEn
			indiceIdempotenciaDevolucionHMAC = hecho.IndiceIdempotenciaHMAC
		case HechoCobroDevolucionResultadoPendiente, HechoCobroDevolucionResultadoDesconocido,
			HechoCobroDevolucionRechazada, HechoCobroDevuelto, HechoCobroDevolucionConciliada:
			if hecho.DevolucionRef != devolucionRef {
				return ErrOrdenCobroInvalida
			}
			if hecho.Tipo == HechoCobroDevuelto {
				devueltaEn = hecho.OcurridoEn
			}
			if hecho.Tipo == HechoCobroDevolucionConciliada {
				devolucionConciliadaEn, devolucionConciliacionRef = hecho.OcurridoEn, hecho.ConciliacionRef
			}
		}
		estado = hecho.EstadoPosterior
		instante = hecho.OcurridoEn
		huellaEstado = hecho.HuellaEstadoPosteriorSHA256
	}
	if estado != o.Estado || !instante.Equal(o.UltimaActualizacionEn) || huellaEstado != o.HuellaEstadoSHA256 {
		return ErrOrdenCobroInvalida
	}
	if o.ConectorID != conectorID || o.VersionConector != versionConector || o.OperacionProveedorRef != operacionProveedorRef {
		return ErrOrdenCobroInvalida
	}
	if conectorID != "" && (!esClaveDocumentalCanonica(o.ConectorID) || o.VersionConector < 1 ||
		!referenciaCobroValida(o.OperacionProveedorRef)) {
		return ErrOrdenCobroInvalida
	}
	if !o.ConfirmadaEn.Equal(confirmadaEn) {
		return ErrOrdenCobroInvalida
	}
	if !o.ConciliadaEn.Equal(conciliadaEn) || o.ConciliacionRef != conciliacionRef {
		return ErrOrdenCobroInvalida
	}
	tieneDevolucion := !devolucionSolicitadaEn.IsZero()
	if tieneDevolucion != (o.DevolucionRef != "" && !o.DevolucionSolicitadaEn.IsZero()) ||
		(tieneDevolucion && (!idDevolucionOpaco.MatchString(o.DevolucionRef) || o.DevolucionRef != devolucionRef ||
			!o.DevolucionSolicitadaEn.Equal(devolucionSolicitadaEn) ||
			o.IndiceIdempotenciaDevolucionHMAC != indiceIdempotenciaDevolucionHMAC)) ||
		(!tieneDevolucion && o.IndiceIdempotenciaDevolucionHMAC != "") {
		return ErrOrdenCobroInvalida
	}
	tieneDevuelta := !devueltaEn.IsZero()
	if tieneDevuelta != !o.DevueltaEn.IsZero() || !o.DevueltaEn.Equal(devueltaEn) {
		return ErrOrdenCobroInvalida
	}
	if !o.DevolucionConciliadaEn.Equal(devolucionConciliadaEn) ||
		o.DevolucionConciliacionRef != devolucionConciliacionRef {
		return ErrOrdenCobroInvalida
	}
	return nil
}

func (o OrdenCobro) Clonar() OrdenCobro {
	clon := o
	clon.Historial = append([]HechoCobro(nil), o.Historial...)
	return clon
}

// VistaTitularOrdenCobro es la proyeccion minima que un adaptador puede
// convertir expresamente en DTO. El agregado completo nunca es un DTO HTTP.
type VistaTitularOrdenCobro struct {
	OrdenRef       string      `json:"orden_ref"`
	Estado         EstadoCobro `json:"estado"`
	Importe        DineroCobro `json:"importe"`
	CreadaEn       time.Time   `json:"creada_en"`
	CaducaEn       time.Time   `json:"caduca_en"`
	ConfirmadaEn   time.Time   `json:"confirmada_en,omitempty"`
	UltimoCambioEn time.Time   `json:"ultimo_cambio_en"`
}

func (o OrdenCobro) VistaTitular() (VistaTitularOrdenCobro, error) {
	if err := o.Validar(); err != nil {
		return VistaTitularOrdenCobro{}, err
	}
	return VistaTitularOrdenCobro{
		OrdenRef: o.ID, Estado: o.Estado, Importe: o.Importe, CreadaEn: o.CreadaEn,
		CaducaEn: o.CaducaEn, ConfirmadaEn: o.ConfirmadaEn, UltimoCambioEn: o.UltimaActualizacionEn,
	}, nil
}

func (o OrdenCobro) ControlConcurrencia() (version int, huellaEstadoSHA256 string, err error) {
	if err = o.Validar(); err != nil {
		return 0, "", err
	}
	return o.Version, o.HuellaEstadoSHA256, nil
}

func (OrdenCobro) MarshalJSON() ([]byte, error) { return nil, ErrSerializacionOrdenCobroProhibida }
func (*OrdenCobro) UnmarshalJSON([]byte) error  { return ErrSerializacionOrdenCobroProhibida }
func (OrdenCobro) MarshalText() ([]byte, error) { return nil, ErrSerializacionOrdenCobroProhibida }
func (OrdenCobro) String() string               { return "[ORDEN-COBRO-INTERNA]" }
func (o OrdenCobro) GoString() string           { return o.String() }
func (o OrdenCobro) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, o.String())
}

type EvidenciaInicioOperacionCobro struct{ datos datosEvidenciaCobro }

// ControlEvidenciaInicioOperacionCobro expone al puerto remoto solo los datos
// imprescindibles para impedir mezclar evidencia y origen de dos conectores.
type ControlEvidenciaInicioOperacionCobro struct {
	ConectorID               string
	VersionConector          int
	OrdenRef                 string
	LiquidacionRef           string
	OperacionProveedorRef    string
	Importe                  DineroCobro
	Concepto                 string
	VerificacionRef          string
	HuellaVerificacionSHA256 string
	RecibidaEn               time.Time
}

func (e EvidenciaInicioOperacionCobro) Control() (ControlEvidenciaInicioOperacionCobro, error) {
	if err := e.Validar(); err != nil {
		return ControlEvidenciaInicioOperacionCobro{}, err
	}
	return ControlEvidenciaInicioOperacionCobro{
		ConectorID: e.datos.conectorID, VersionConector: e.datos.versionConector,
		OrdenRef: e.datos.ordenRef, LiquidacionRef: e.datos.liquidacionRef,
		OperacionProveedorRef: e.datos.operacionProveedorRef, Importe: e.datos.importe,
		Concepto: e.datos.concepto, VerificacionRef: e.datos.verificacionRef,
		HuellaVerificacionSHA256: e.datos.huellaVerificacionSHA256,
		RecibidaEn:               e.datos.recibidaEn,
	}, nil
}

type EvidenciaResultadoCobro struct {
	datos     datosEvidenciaCobro
	resultado ResultadoOperacionCobro
}
type EvidenciaResultadoDevolucionCobro struct {
	datos         datosEvidenciaCobro
	devolucionRef string
	resultado     ResultadoDevolucionCobro
}
type EvidenciaConciliacionCobro struct {
	datos           datosEvidenciaCobro
	tipo            TipoConciliacionCobro
	conciliacionRef string
}

type MetodoAutenticacionEvidenciaCobro string

const (
	MetodoAutenticacionCobroFirmaMensaje        MetodoAutenticacionEvidenciaCobro = "firma_mensaje"
	MetodoAutenticacionCobroTLSMutuo            MetodoAutenticacionEvidenciaCobro = "tls_mutuo"
	MetodoAutenticacionCobroFirmaYTLSMutuo      MetodoAutenticacionEvidenciaCobro = "firma_mensaje_y_tls_mutuo"
	MetodoAutenticacionCobroConsultaAutenticada MetodoAutenticacionEvidenciaCobro = "consulta_canal_autenticado"
	metodoAutenticacionCobroDecisionInterna     MetodoAutenticacionEvidenciaCobro = "decision_interna"
)

func (m MetodoAutenticacionEvidenciaCobro) Valido() bool {
	return m == MetodoAutenticacionCobroFirmaMensaje || m == MetodoAutenticacionCobroTLSMutuo ||
		m == MetodoAutenticacionCobroFirmaYTLSMutuo || m == MetodoAutenticacionCobroConsultaAutenticada
}

type datosEvidenciaCobro struct {
	evidenciaRef                string
	evidenciaRelacionadaRef     string
	huellaSHA256                string
	huellaMensajeOriginalSHA256 string
	conectorID                  string
	versionConector             int
	ordenRef                    string
	liquidacionRef              string
	operacionProveedorRef       string
	devolucionRef               string
	conciliacionRef             string
	indiceIdempotenciaHMAC      string
	importe                     DineroCobro
	concepto                    string
	codigo                      string
	metodoAutenticacion         MetodoAutenticacionEvidenciaCobro
	audiencia                   string
	verificacionRef             string
	huellaVerificacionSHA256    string
	emitidaEn                   time.Time
	recibidaEn                  time.Time
	verificadaEn                time.Time
	ocurridoEn                  time.Time
}

type DatosEvidenciaServidorCobro struct {
	EvidenciaRef             string
	HuellaSHA256             string
	ConectorID               string
	VersionConector          int
	OrdenRef                 string
	LiquidacionRef           string
	OperacionProveedorRef    string
	Importe                  DineroCobro
	Concepto                 string
	Codigo                   string
	MetodoAutenticacion      MetodoAutenticacionEvidenciaCobro
	Audiencia                string
	VerificacionRef          string
	HuellaVerificacionSHA256 string
	EmitidaEn                time.Time
	RecibidaEn               time.Time
	VerificadaEn             time.Time
}

func (d DatosEvidenciaServidorCobro) validar() (datosEvidenciaCobro, error) {
	if !referenciaEvidenciaEntradaCobroValida(d.EvidenciaRef) || !esSHA256(d.HuellaSHA256) ||
		!esClaveDocumentalCanonica(d.ConectorID) || d.VersionConector < 1 ||
		!idOrdenCobroOpaco.MatchString(d.OrdenRef) || !referenciaCobroValida(d.LiquidacionRef) ||
		!referenciaCobroValida(d.OperacionProveedorRef) || d.Importe.Validar() != nil ||
		!textoCobroValido(d.Concepto, maximoCaracteresConceptoCobro) || !textoCobroValido(d.Codigo, 128) ||
		!d.MetodoAutenticacion.Valido() || d.MetodoAutenticacion == metodoAutenticacionCobroDecisionInterna ||
		d.Audiencia != audienciaEvidenciaCobro || !referenciaCobroValida(d.VerificacionRef) ||
		!esSHA256(d.HuellaVerificacionSHA256) || d.EmitidaEn.IsZero() ||
		d.RecibidaEn.IsZero() || d.RecibidaEn.Before(d.EmitidaEn) ||
		d.VerificadaEn.IsZero() || d.VerificadaEn.Before(d.RecibidaEn) ||
		d.VerificadaEn.Sub(d.RecibidaEn) > desfaseMaximoEvidenciaCobro ||
		contieneDatoTarjetaCobro(d.Concepto, d.Codigo, string(d.MetodoAutenticacion), d.Audiencia, d.LiquidacionRef) {
		return datosEvidenciaCobro{}, ErrEvidenciaCobroInvalida
	}
	return datosEvidenciaCobro{
		evidenciaRef: d.EvidenciaRef, huellaMensajeOriginalSHA256: d.HuellaSHA256,
		conectorID: d.ConectorID, versionConector: d.VersionConector,
		ordenRef: d.OrdenRef, liquidacionRef: d.LiquidacionRef,
		operacionProveedorRef: d.OperacionProveedorRef, importe: d.Importe, concepto: d.Concepto,
		codigo: d.Codigo, metodoAutenticacion: d.MetodoAutenticacion,
		audiencia: d.Audiencia, verificacionRef: d.VerificacionRef,
		huellaVerificacionSHA256: d.HuellaVerificacionSHA256,
		emitidaEn:                d.EmitidaEn.UTC(), recibidaEn: d.RecibidaEn.UTC(), verificadaEn: d.VerificadaEn.UTC(),
	}, nil
}

type instantaneaEvidenciaCobro struct {
	Tipo                  string
	Resultado             string
	EvidenciaRef          string
	HuellaMensajeOriginal string
	ConectorID            string
	VersionConector       int
	OrdenRef              string
	LiquidacionRef        string
	OperacionProveedorRef string
	DevolucionRef         string
	ConciliacionRef       string
	Importe               DineroCobro
	Concepto              string
	Codigo                string
	MetodoAutenticacion   MetodoAutenticacionEvidenciaCobro
	Audiencia             string
	VerificacionRef       string
	HuellaVerificacion    string
	EmitidaEn             string
	RecibidaEn            string
	VerificadaEn          string
}

func sellarEvidenciaCobro(datos datosEvidenciaCobro, tipo, resultado string) (datosEvidenciaCobro, error) {
	huella, err := huellaCanonicaCobro(instantaneaEvidenciaCobro{
		Tipo: tipo, Resultado: resultado, EvidenciaRef: datos.evidenciaRef,
		HuellaMensajeOriginal: datos.huellaMensajeOriginalSHA256,
		ConectorID:            datos.conectorID, VersionConector: datos.versionConector,
		OrdenRef: datos.ordenRef, LiquidacionRef: datos.liquidacionRef,
		OperacionProveedorRef: datos.operacionProveedorRef, DevolucionRef: datos.devolucionRef,
		ConciliacionRef: datos.conciliacionRef, Importe: datos.importe, Concepto: datos.concepto,
		Codigo: datos.codigo, MetodoAutenticacion: datos.metodoAutenticacion, Audiencia: datos.audiencia,
		VerificacionRef: datos.verificacionRef, HuellaVerificacion: datos.huellaVerificacionSHA256,
		EmitidaEn: datos.emitidaEn.UTC().Format(time.RFC3339Nano), RecibidaEn: datos.recibidaEn.UTC().Format(time.RFC3339Nano),
		VerificadaEn: datos.verificadaEn.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return datosEvidenciaCobro{}, ErrEvidenciaCobroInvalida
	}
	datos.huellaSHA256 = huella
	return datos, nil
}

// NuevaEvidenciaInicioOperacionCobroVerificada no verifica
// criptografia. Es una fabrica de frontera para implementaciones del puerto
// VerificadorPasarelaCobro; los adaptadores HTTP no deben invocarla.
func NuevaEvidenciaInicioOperacionCobroVerificada(datos DatosEvidenciaServidorCobro) (EvidenciaInicioOperacionCobro, error) {
	validos, err := datos.validar()
	if err != nil {
		return EvidenciaInicioOperacionCobro{}, err
	}
	validos, err = sellarEvidenciaCobro(validos, "inicio_operacion", "iniciada")
	if err != nil {
		return EvidenciaInicioOperacionCobro{}, err
	}
	return EvidenciaInicioOperacionCobro{datos: validos}, nil
}

type ResultadoOperacionCobro string

const (
	ResultadoOperacionCobroPendiente   ResultadoOperacionCobro = "pendiente"
	ResultadoOperacionCobroConfirmado  ResultadoOperacionCobro = "confirmado"
	ResultadoOperacionCobroRechazado   ResultadoOperacionCobro = "rechazado"
	ResultadoOperacionCobroDesconocido ResultadoOperacionCobro = "desconocido"
)

func (r ResultadoOperacionCobro) Valido() bool {
	return r == ResultadoOperacionCobroPendiente || r == ResultadoOperacionCobroConfirmado ||
		r == ResultadoOperacionCobroRechazado || r == ResultadoOperacionCobroDesconocido
}

// NuevaEvidenciaResultadoCobroVerificada no constituye por si misma una prueba
// criptografica. Valida exactamente la salida de un verificador confiable, sin
// corregirla, y sella todos sus campos para detectar cambios posteriores.
func NuevaEvidenciaResultadoCobroVerificada(datos DatosEvidenciaServidorCobro, resultado ResultadoOperacionCobro) (EvidenciaResultadoCobro, error) {
	validos, err := datos.validar()
	if err != nil || !resultado.Valido() {
		return EvidenciaResultadoCobro{}, ErrEvidenciaCobroInvalida
	}
	validos, err = sellarEvidenciaCobro(validos, "resultado_cobro", string(resultado))
	if err != nil {
		return EvidenciaResultadoCobro{}, err
	}
	return EvidenciaResultadoCobro{datos: validos, resultado: resultado}, nil
}

type ResultadoDevolucionCobro string

const (
	ResultadoDevolucionCobroPendiente   ResultadoDevolucionCobro = "pendiente"
	ResultadoDevolucionCobroConfirmada  ResultadoDevolucionCobro = "confirmada"
	ResultadoDevolucionCobroRechazada   ResultadoDevolucionCobro = "rechazada"
	ResultadoDevolucionCobroDesconocido ResultadoDevolucionCobro = "desconocido"
)

func (r ResultadoDevolucionCobro) Valido() bool {
	return r == ResultadoDevolucionCobroPendiente || r == ResultadoDevolucionCobroConfirmada ||
		r == ResultadoDevolucionCobroRechazada || r == ResultadoDevolucionCobroDesconocido
}

func NuevaEvidenciaResultadoDevolucionCobroVerificada(datos DatosEvidenciaServidorCobro, devolucionRef string, resultado ResultadoDevolucionCobro) (EvidenciaResultadoDevolucionCobro, error) {
	validos, err := datos.validar()
	if err != nil || !idDevolucionOpaco.MatchString(devolucionRef) || !resultado.Valido() {
		return EvidenciaResultadoDevolucionCobro{}, ErrEvidenciaCobroInvalida
	}
	validos.devolucionRef = devolucionRef
	validos, err = sellarEvidenciaCobro(validos, "resultado_devolucion", string(resultado))
	if err != nil {
		return EvidenciaResultadoDevolucionCobro{}, err
	}
	return EvidenciaResultadoDevolucionCobro{datos: validos, devolucionRef: devolucionRef, resultado: resultado}, nil
}

type TipoConciliacionCobro string

const (
	TipoConciliacionCobroIngreso    TipoConciliacionCobro = "ingreso"
	TipoConciliacionCobroDevolucion TipoConciliacionCobro = "devolucion"
)

func (t TipoConciliacionCobro) Valido() bool {
	return t == TipoConciliacionCobroIngreso || t == TipoConciliacionCobroDevolucion
}

func NuevaEvidenciaConciliacionCobroVerificada(datos DatosEvidenciaServidorCobro, tipo TipoConciliacionCobro, conciliacionRef, devolucionRef string) (EvidenciaConciliacionCobro, error) {
	validos, err := datos.validar()
	if err != nil || !tipo.Valido() || !referenciaCobroValida(conciliacionRef) ||
		(tipo == TipoConciliacionCobroIngreso && devolucionRef != "") ||
		(tipo == TipoConciliacionCobroDevolucion && !idDevolucionOpaco.MatchString(devolucionRef)) {
		return EvidenciaConciliacionCobro{}, ErrConciliacionCobroInvalida
	}
	validos.conciliacionRef = conciliacionRef
	validos.devolucionRef = devolucionRef
	validos, err = sellarEvidenciaCobro(validos, "conciliacion", string(tipo))
	if err != nil {
		return EvidenciaConciliacionCobro{}, err
	}
	return EvidenciaConciliacionCobro{datos: validos, tipo: tipo, conciliacionRef: conciliacionRef}, nil
}

func (e EvidenciaInicioOperacionCobro) Validar() error {
	return e.datos.validarSello("inicio_operacion", "iniciada")
}
func (e EvidenciaResultadoCobro) Validar() error {
	if err := e.datos.validarSello("resultado_cobro", string(e.resultado)); err != nil || !e.resultado.Valido() {
		return ErrEvidenciaCobroInvalida
	}
	return nil
}
func (e EvidenciaResultadoDevolucionCobro) Validar() error {
	if err := e.datos.validarSello("resultado_devolucion", string(e.resultado)); err != nil ||
		!idDevolucionOpaco.MatchString(e.devolucionRef) || e.datos.devolucionRef != e.devolucionRef || !e.resultado.Valido() {
		return ErrEvidenciaCobroInvalida
	}
	return nil
}
func (e EvidenciaConciliacionCobro) Validar() error {
	if err := e.datos.validarSello("conciliacion", string(e.tipo)); err != nil || !e.tipo.Valido() ||
		!referenciaCobroValida(e.conciliacionRef) || e.datos.conciliacionRef != e.conciliacionRef ||
		(e.tipo == TipoConciliacionCobroIngreso && e.datos.devolucionRef != "") ||
		(e.tipo == TipoConciliacionCobroDevolucion && !idDevolucionOpaco.MatchString(e.datos.devolucionRef)) {
		return ErrConciliacionCobroInvalida
	}
	return nil
}

func (d datosEvidenciaCobro) validarInterna() error {
	_, err := (DatosEvidenciaServidorCobro{
		EvidenciaRef: d.evidenciaRef, HuellaSHA256: d.huellaMensajeOriginalSHA256, ConectorID: d.conectorID,
		VersionConector: d.versionConector, OrdenRef: d.ordenRef, LiquidacionRef: d.liquidacionRef,
		OperacionProveedorRef: d.operacionProveedorRef, Importe: d.importe, Concepto: d.concepto, Codigo: d.codigo,
		MetodoAutenticacion: d.metodoAutenticacion, Audiencia: d.audiencia,
		VerificacionRef: d.verificacionRef, HuellaVerificacionSHA256: d.huellaVerificacionSHA256,
		EmitidaEn: d.emitidaEn, RecibidaEn: d.recibidaEn, VerificadaEn: d.verificadaEn,
	}).validar()
	return err
}

func (d datosEvidenciaCobro) validarSello(tipo, resultado string) error {
	if err := d.validarInterna(); err != nil || !esSHA256(d.huellaSHA256) {
		return ErrEvidenciaCobroInvalida
	}
	sellados, err := sellarEvidenciaCobro(d, tipo, resultado)
	if err != nil || sellados.huellaSHA256 != d.huellaSHA256 {
		return ErrEvidenciaCobroInvalida
	}
	return nil
}

func (e EvidenciaInicioOperacionCobro) String() string       { return "[EVIDENCIA-COBRO-INTERNA]" }
func (e EvidenciaResultadoCobro) String() string             { return "[EVIDENCIA-COBRO-INTERNA]" }
func (e EvidenciaResultadoDevolucionCobro) String() string   { return "[EVIDENCIA-COBRO-INTERNA]" }
func (e EvidenciaConciliacionCobro) String() string          { return "[EVIDENCIA-COBRO-INTERNA]" }
func (e EvidenciaInicioOperacionCobro) GoString() string     { return e.String() }
func (e EvidenciaResultadoCobro) GoString() string           { return e.String() }
func (e EvidenciaResultadoDevolucionCobro) GoString() string { return e.String() }
func (e EvidenciaConciliacionCobro) GoString() string        { return e.String() }
func (e EvidenciaInicioOperacionCobro) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, e.String())
}
func (e EvidenciaResultadoCobro) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, e.String()) }
func (e EvidenciaResultadoDevolucionCobro) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, e.String())
}
func (e EvidenciaConciliacionCobro) Format(s fmt.State, _ rune) { _, _ = io.WriteString(s, e.String()) }
func (EvidenciaInicioOperacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaResultadoCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaResultadoDevolucionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaConciliacionCobro) MarshalJSON() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (*EvidenciaInicioOperacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (*EvidenciaResultadoCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (*EvidenciaResultadoDevolucionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (*EvidenciaConciliacionCobro) UnmarshalJSON([]byte) error {
	return ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaInicioOperacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaResultadoCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaResultadoDevolucionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}
func (EvidenciaConciliacionCobro) MarshalText() ([]byte, error) {
	return nil, ErrSerializacionEvidenciaCobroProhibida
}

func (o OrdenCobro) PrepararInicioOperacion(retornoUsuarioRef, notificacionServidorRef string, instante time.Time, autorizacion ContextoAutorizacionCobro) (ComandoInicioOperacionCobro, error) {
	if err := o.Validar(); err != nil || o.Estado != EstadoCobroCreada ||
		!referenciaCobroValida(retornoUsuarioRef) || !referenciaCobroValida(notificacionServidorRef) ||
		instante.IsZero() || instante.Before(o.UltimaActualizacionEn) {
		return ComandoInicioOperacionCobro{}, ErrComandoCobroInvalido
	}
	if err := autorizacion.ValidarEn(AccionCobroIniciarOperacion, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return ComandoInicioOperacionCobro{}, err
	}
	if !instante.Before(o.CaducaEn) {
		return ComandoInicioOperacionCobro{}, ErrTransicionCobroInvalida
	}
	comando := ComandoInicioOperacionCobro{datos: DatosComandoInicioOperacionCobro{
		OrdenRef: o.ID, VersionOrden: o.Version, HuellaOrdenSHA256: o.HuellaEstadoSHA256,
		LiquidacionRef: o.LiquidacionRef, IndiceIdempotenciaHMAC: o.IndiceIdempotenciaHMAC,
		Importe: o.Importe, Concepto: o.Concepto, CaducaEn: o.CaducaEn,
		RetornoUsuarioRef: retornoUsuarioRef, NotificacionServidorRef: notificacionServidorRef,
		DecisionAutorizacionRef: autorizacion.datos.DecisionRef, CorrelacionRef: o.CorrelacionRef,
	}}
	if err := comando.Validar(); err != nil {
		return ComandoInicioOperacionCobro{}, err
	}
	return comando, nil
}

func (o OrdenCobro) RegistrarEnvio(evidencia EvidenciaInicioOperacionCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error) {
	if !textoCobroValido(motivo, maximoCaracteresReferenciaDocumental) {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	if err := evidencia.Validar(); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := autorizacion.ValidarEn(AccionCobroIniciarOperacion, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := o.coincideEvidencia(evidencia.datos, false, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if repetida, err := o.comprobarEvidenciaRepetida(evidencia.datos, HechoCobroOperacionEnviada); repetida {
		return o.Clonar(), true, nil
	} else if err != nil {
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Inicio de operacion incompatible", instante)
	}
	if o.Estado != EstadoCobroCreada {
		return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
	}
	nueva, repetida, err := o.aplicarHecho(nuevoHechoCobro(o, HechoCobroOperacionEnviada,
		EstadoCobroEnviadaPasarela, evidencia.datos, autorizacion, motivo))
	if err == nil && !repetida {
		nueva.ConectorID = evidencia.datos.conectorID
		nueva.VersionConector = evidencia.datos.versionConector
		nueva.OperacionProveedorRef = evidencia.datos.operacionProveedorRef
		err = nueva.Validar()
	}
	return nueva, repetida, err
}

func (o OrdenCobro) AplicarResultadoServidor(evidencia EvidenciaResultadoCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error) {
	if !textoCobroValido(motivo, maximoCaracteresReferenciaDocumental) {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	if err := evidencia.Validar(); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := autorizacion.ValidarEn(AccionCobroProcesarResultado, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := o.coincideEvidencia(evidencia.datos, true, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if repetida, err := o.comprobarEvidenciaRepetida(evidencia.datos, tipoHechoResultadoCobro(evidencia.resultado)); repetida {
		return o.Clonar(), true, nil
	} else if err != nil {
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Evidencia reutilizada con otro contenido", instante)
	}
	if o.Estado == EstadoCobroConfirmada || o.Estado == EstadoCobroConciliada || estadoCobroTieneDevolucion(o.Estado) {
		if evidencia.resultado == ResultadoOperacionCobroConfirmado {
			return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
		}
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Resultado posterior incompatible con cobro confirmado", instante)
	}
	if o.Estado == EstadoCobroRechazada {
		if evidencia.resultado == ResultadoOperacionCobroRechazado {
			return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
		}
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Resultado posterior incompatible con rechazo", instante)
	}
	if (o.Estado == EstadoCobroResultadoPendiente && evidencia.resultado == ResultadoOperacionCobroPendiente) ||
		(o.Estado == EstadoCobroResultadoDesconocido && evidencia.resultado == ResultadoOperacionCobroDesconocido) {
		return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
	}
	if o.Estado == EstadoCobroCancelada || o.Estado == EstadoCobroCaducada {
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Resultado posterior incompatible con cierre local", instante)
	}
	tipo, estado := HechoCobroResultadoPendiente, EstadoCobroResultadoPendiente
	switch evidencia.resultado {
	case ResultadoOperacionCobroConfirmado:
		tipo, estado = HechoCobroConfirmado, EstadoCobroConfirmada
	case ResultadoOperacionCobroRechazado:
		tipo, estado = HechoCobroRechazado, EstadoCobroRechazada
	case ResultadoOperacionCobroDesconocido:
		tipo, estado = HechoCobroResultadoDesconocido, EstadoCobroResultadoDesconocido
	}
	nueva, repetida, err := o.aplicarHecho(nuevoHechoCobro(o, tipo, estado, evidencia.datos, autorizacion, motivo))
	if err == nil && !repetida && estado == EstadoCobroConfirmada {
		nueva.ConfirmadaEn = evidencia.datos.recibidaEn
	}
	if err == nil && !repetida {
		err = nueva.Validar()
	}
	return nueva, repetida, err
}

type SolicitudDevolucionOrdenCobro struct {
	DevolucionRef          string
	EvidenciaRef           string
	HuellaEvidenciaSHA256  string
	IndiceIdempotenciaHMAC string
	Motivo                 string
	SolicitadaEn           time.Time
}

// BytesCanonicosIdempotenciaDevolucionCobro fija todos los datos que pueden
// cambiar el significado administrativo o economico de una devolucion. Omite
// exclusivamente el instante, que procede del servidor.
func BytesCanonicosIdempotenciaDevolucionCobro(o OrdenCobro, solicitud SolicitudDevolucionOrdenCobro) ([]byte, error) {
	if o.Validar() != nil || !idDevolucionOpaco.MatchString(solicitud.DevolucionRef) ||
		!referenciaEvidenciaEntradaCobroValida(solicitud.EvidenciaRef) ||
		!esSHA256(solicitud.HuellaEvidenciaSHA256) ||
		!textoCobroValido(solicitud.Motivo, maximoCaracteresReferenciaDocumental) {
		return nil, ErrDevolucionCobroInvalida
	}
	valor := struct {
		VersionEsquema  int
		OrdenRef        string
		LiquidacionRef  string
		DevolucionRef   string
		EvidenciaRef    string
		HuellaEvidencia string
		Importe         DineroCobro
		Motivo          string
	}{
		VersionEsquema: versionEsquemaIntegridadCobro, OrdenRef: o.ID, LiquidacionRef: o.LiquidacionRef,
		DevolucionRef: solicitud.DevolucionRef, EvidenciaRef: solicitud.EvidenciaRef,
		HuellaEvidencia: solicitud.HuellaEvidenciaSHA256, Importe: o.Importe, Motivo: solicitud.Motivo,
	}
	bytes, err := json.Marshal(valor)
	if err != nil {
		return nil, ErrDevolucionCobroInvalida
	}
	return append([]byte(nil), bytes...), nil
}

func (o OrdenCobro) SolicitarDevolucion(solicitud SolicitudDevolucionOrdenCobro, autorizacion ContextoAutorizacionCobro) (OrdenCobro, ComandoDevolucionCobro, bool, error) {
	if err := o.Validar(); err != nil || !idDevolucionOpaco.MatchString(solicitud.DevolucionRef) ||
		!referenciaEvidenciaEntradaCobroValida(solicitud.EvidenciaRef) || !esSHA256(solicitud.HuellaEvidenciaSHA256) ||
		!esHuellaHMACCobroDeDominio(solicitud.IndiceIdempotenciaHMAC, dominioHMACDevolucionCobro) ||
		!textoCobroValido(solicitud.Motivo, maximoCaracteresReferenciaDocumental) ||
		solicitud.SolicitadaEn.IsZero() || solicitud.SolicitadaEn.Before(o.UltimaActualizacionEn) ||
		contieneDatoTarjetaCobro(solicitud.Motivo) {
		return OrdenCobro{}, ComandoDevolucionCobro{}, false, ErrDevolucionCobroInvalida
	}
	if err := autorizacion.ValidarEn(AccionCobroSolicitarDevolucion, o.ID, o.Finalidad, o.CorrelacionRef, solicitud.SolicitadaEn); err != nil {
		return OrdenCobro{}, ComandoDevolucionCobro{}, false, err
	}
	if autorizacion.datos.EvaluadaEn.After(solicitud.SolicitadaEn) ||
		solicitud.SolicitadaEn.Sub(autorizacion.datos.EvaluadaEn) > time.Minute {
		return OrdenCobro{}, ComandoDevolucionCobro{}, false, ErrContextoAutorizacionCobroInvalido
	}
	datos := datosEvidenciaCobro{
		evidenciaRef: solicitud.EvidenciaRef, huellaSHA256: solicitud.HuellaEvidenciaSHA256,
		ordenRef: o.ID, liquidacionRef: o.LiquidacionRef,
		importe: o.Importe, concepto: o.Concepto, codigo: "devolucion_solicitada",
		metodoAutenticacion: metodoAutenticacionCobroDecisionInterna,
		audiencia:           audienciaEvidenciaCobro, emitidaEn: solicitud.SolicitadaEn.UTC(), recibidaEn: solicitud.SolicitadaEn.UTC(),
	}
	datos.devolucionRef = solicitud.DevolucionRef
	datos.indiceIdempotenciaHMAC = solicitud.IndiceIdempotenciaHMAC
	nueva, repetida, err := o.Clonar(), false, error(nil)
	for _, anterior := range o.Historial {
		if anterior.Tipo != HechoCobroDevolucionSolicitada ||
			anterior.IndiceIdempotenciaHMAC != solicitud.IndiceIdempotenciaHMAC {
			continue
		}
		if anterior.DevolucionRef == solicitud.DevolucionRef && anterior.EvidenciaRef == solicitud.EvidenciaRef &&
			anterior.HuellaEvidenciaSHA256 == solicitud.HuellaEvidenciaSHA256 &&
			anterior.Motivo == solicitud.Motivo {
			repetida = true
			break
		}
		datos.ocurridoEn = solicitud.SolicitadaEn.UTC()
		bloqueada, repetidaIncidencia, errorIncidencia := o.bloquearPorIncidencia(
			datos, autorizacion, "Idempotencia de devolucion reutilizada con otros datos", solicitud.SolicitadaEn,
		)
		return bloqueada, ComandoDevolucionCobro{}, repetidaIncidencia, errorIncidencia
	}
	if !repetida {
		nueva, repetida, err = o.aplicarHecho(nuevoHechoCobro(o, HechoCobroDevolucionSolicitada,
			EstadoCobroDevolucionSolicitada, datos, autorizacion, solicitud.Motivo))
	}
	if errors.Is(err, ErrEvidenciaCobroConflictiva) {
		bloqueada, repetidaIncidencia, errorIncidencia := o.bloquearPorIncidencia(datos, autorizacion, "Solicitud de devolucion reutilizada con otros datos", solicitud.SolicitadaEn)
		return bloqueada, ComandoDevolucionCobro{}, repetidaIncidencia, errorIncidencia
	}
	if err == nil && !repetida {
		nueva.DevolucionRef = solicitud.DevolucionRef
		nueva.IndiceIdempotenciaDevolucionHMAC = solicitud.IndiceIdempotenciaHMAC
		nueva.DevolucionSolicitadaEn = solicitud.SolicitadaEn.UTC()
		err = nueva.Validar()
	}
	if err != nil {
		return OrdenCobro{}, ComandoDevolucionCobro{}, false, err
	}
	comando := ComandoDevolucionCobro{datos: DatosComandoDevolucionCobro{
		OrdenRef: nueva.ID, VersionOrden: nueva.Version, HuellaOrdenSHA256: nueva.HuellaEstadoSHA256,
		ConectorID: nueva.ConectorID, VersionConector: nueva.VersionConector,
		OperacionProveedorRef: nueva.OperacionProveedorRef, DevolucionRef: nueva.DevolucionRef,
		IndiceIdempotenciaHMAC: nueva.IndiceIdempotenciaDevolucionHMAC, Importe: nueva.Importe,
		Motivo: solicitud.Motivo, DecisionAutorizacionRef: autorizacion.datos.DecisionRef,
		CorrelacionRef: nueva.CorrelacionRef,
	}}
	if err := comando.Validar(); err != nil {
		return OrdenCobro{}, ComandoDevolucionCobro{}, false, err
	}
	return nueva, comando, repetida, nil
}

func (o OrdenCobro) AplicarResultadoDevolucionServidor(evidencia EvidenciaResultadoDevolucionCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error) {
	if !textoCobroValido(motivo, maximoCaracteresReferenciaDocumental) {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	if err := evidencia.Validar(); err != nil || evidencia.devolucionRef != o.DevolucionRef {
		return OrdenCobro{}, false, ErrDevolucionCobroInvalida
	}
	if err := autorizacion.ValidarEn(AccionCobroProcesarDevolucion, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := o.coincideEvidencia(evidencia.datos, true, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	tipoEsperado := tipoHechoResultadoDevolucionCobro(evidencia.resultado)
	if repetida, err := o.comprobarEvidenciaRepetida(evidencia.datos, tipoEsperado); repetida {
		return o.Clonar(), true, nil
	} else if err != nil {
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Evidencia de devolucion reutilizada", instante)
	}
	if o.Estado == EstadoCobroDevuelta || o.Estado == EstadoCobroDevolucionConciliada {
		if evidencia.resultado == ResultadoDevolucionCobroConfirmada {
			return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
		}
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Resultado incompatible con devolucion confirmada", instante)
	}
	if o.Estado == EstadoCobroDevolucionRechazada {
		if evidencia.resultado == ResultadoDevolucionCobroRechazada {
			return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
		}
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Resultado incompatible con devolucion rechazada", instante)
	}
	tipo, estado := HechoCobroDevolucionResultadoDesconocido, EstadoCobroDevolucionSolicitada
	switch evidencia.resultado {
	case ResultadoDevolucionCobroConfirmada:
		tipo, estado = HechoCobroDevuelto, EstadoCobroDevuelta
	case ResultadoDevolucionCobroRechazada:
		tipo, estado = HechoCobroDevolucionRechazada, EstadoCobroDevolucionRechazada
	case ResultadoDevolucionCobroPendiente:
		tipo, estado = HechoCobroDevolucionResultadoPendiente, EstadoCobroDevolucionSolicitada
	}
	nueva, repetida, err := o.aplicarHecho(nuevoHechoCobro(o, tipo, estado, evidencia.datos, autorizacion, motivo))
	if err == nil && !repetida && estado == EstadoCobroDevuelta {
		nueva.DevueltaEn = evidencia.datos.recibidaEn
	}
	if err == nil && !repetida {
		err = nueva.Validar()
	}
	return nueva, repetida, err
}

func (o OrdenCobro) PrepararConciliacion(tipo TipoConciliacionCobro, referenciaCierre string, instante time.Time, autorizacion ContextoAutorizacionCobro) (ComandoConciliacionCobro, error) {
	if err := o.Validar(); err != nil || !tipo.Valido() || !referenciaCobroValida(referenciaCierre) ||
		instante.IsZero() || instante.Before(o.UltimaActualizacionEn) {
		return ComandoConciliacionCobro{}, ErrConciliacionCobroInvalida
	}
	if err := autorizacion.ValidarEn(AccionCobroConciliar, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return ComandoConciliacionCobro{}, err
	}
	devolucionRef := ""
	if tipo == TipoConciliacionCobroIngreso {
		if o.Estado != EstadoCobroConfirmada {
			return ComandoConciliacionCobro{}, ErrTransicionCobroInvalida
		}
	} else {
		if o.Estado != EstadoCobroDevuelta {
			return ComandoConciliacionCobro{}, ErrTransicionCobroInvalida
		}
		devolucionRef = o.DevolucionRef
	}
	comando := ComandoConciliacionCobro{datos: DatosComandoConciliacionCobro{
		OrdenRef: o.ID, VersionOrden: o.Version, HuellaOrdenSHA256: o.HuellaEstadoSHA256,
		ConectorID: o.ConectorID, VersionConector: o.VersionConector,
		OperacionProveedorRef: o.OperacionProveedorRef, DevolucionRef: devolucionRef,
		Tipo: tipo, Importe: o.Importe, ReferenciaCierre: referenciaCierre,
		DecisionAutorizacionRef: autorizacion.datos.DecisionRef, CorrelacionRef: o.CorrelacionRef,
	}}
	if err := comando.Validar(); err != nil {
		return ComandoConciliacionCobro{}, err
	}
	return comando, nil
}

func (o OrdenCobro) AplicarConciliacionServidor(evidencia EvidenciaConciliacionCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error) {
	if !textoCobroValido(motivo, maximoCaracteresReferenciaDocumental) {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	if err := evidencia.Validar(); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := autorizacion.ValidarEn(AccionCobroConciliar, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if err := o.coincideEvidencia(evidencia.datos, true, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	tipo, estado := HechoCobroConciliado, EstadoCobroConciliada
	if evidencia.tipo == TipoConciliacionCobroDevolucion {
		tipo, estado = HechoCobroDevolucionConciliada, EstadoCobroDevolucionConciliada
		if evidencia.datos.devolucionRef != o.DevolucionRef {
			return OrdenCobro{}, false, ErrCoincidenciaCobroInvalida
		}
	}
	if repetida, err := o.comprobarEvidenciaRepetida(evidencia.datos, tipo); repetida {
		return o.Clonar(), true, nil
	} else if err != nil {
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Evidencia de conciliacion reutilizada", instante)
	}
	if evidencia.tipo == TipoConciliacionCobroIngreso && o.ConciliacionRef != "" {
		if o.ConciliacionRef == evidencia.conciliacionRef {
			return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
		}
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Dos referencias de conciliacion de ingreso", instante)
	}
	if evidencia.tipo == TipoConciliacionCobroDevolucion && o.DevolucionConciliacionRef != "" {
		if o.DevolucionConciliacionRef == evidencia.conciliacionRef {
			return o.conservarEvidenciaAdicional(evidencia.datos, autorizacion, motivo)
		}
		return o.bloquearPorIncidencia(evidencia.datos, autorizacion, "Dos referencias de conciliacion de devolucion", instante)
	}
	nueva, repetida, err := o.aplicarHecho(nuevoHechoCobro(o, tipo, estado, evidencia.datos, autorizacion, motivo))
	if err == nil && !repetida {
		if evidencia.tipo == TipoConciliacionCobroIngreso {
			nueva.ConciliadaEn = evidencia.datos.recibidaEn
			nueva.ConciliacionRef = evidencia.conciliacionRef
		} else {
			nueva.DevolucionConciliadaEn = evidencia.datos.recibidaEn
			nueva.DevolucionConciliacionRef = evidencia.conciliacionRef
		}
		err = nueva.Validar()
	}
	return nueva, repetida, err
}

func (o OrdenCobro) Cancelar(evidenciaRef, huella, motivo string, instante time.Time, autorizacion ContextoAutorizacionCobro) (OrdenCobro, bool, error) {
	return o.aplicarDecisionLocal(HechoCobroCancelado, EstadoCobroCancelada, evidenciaRef, huella,
		motivo, instante, autorizacion, AccionCobroCancelar)
}

func (o OrdenCobro) Caducar(evidenciaRef, huella, motivo string, instante time.Time, autorizacion ContextoAutorizacionCobro) (OrdenCobro, bool, error) {
	if instante.Before(o.CaducaEn) {
		return OrdenCobro{}, false, ErrTransicionCobroInvalida
	}
	return o.aplicarDecisionLocal(HechoCobroCaducado, EstadoCobroCaducada, evidenciaRef, huella,
		motivo, instante, autorizacion, AccionCobroCaducar)
}

func (o OrdenCobro) aplicarDecisionLocal(tipo TipoHechoCobro, estado EstadoCobro, evidenciaRef, huella,
	motivo string, instante time.Time, autorizacion ContextoAutorizacionCobro, accion AccionCobro) (OrdenCobro, bool, error) {
	if err := o.Validar(); err != nil || !referenciaEvidenciaEntradaCobroValida(evidenciaRef) || !esSHA256(huella) ||
		!textoCobroValido(motivo, maximoCaracteresReferenciaDocumental) ||
		instante.IsZero() || instante.Before(o.UltimaActualizacionEn) || contieneDatoTarjetaCobro(motivo) {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	if err := autorizacion.ValidarEn(accion, o.ID, o.Finalidad, o.CorrelacionRef, instante); err != nil {
		return OrdenCobro{}, false, err
	}
	if autorizacion.datos.EvaluadaEn.After(instante) || instante.Sub(autorizacion.datos.EvaluadaEn) > time.Minute {
		return OrdenCobro{}, false, ErrContextoAutorizacionCobroInvalido
	}
	datos := datosEvidenciaCobro{
		evidenciaRef: evidenciaRef, huellaSHA256: huella,
		ordenRef: o.ID, liquidacionRef: o.LiquidacionRef, importe: o.Importe, concepto: o.Concepto,
		codigo: string(tipo), metodoAutenticacion: metodoAutenticacionCobroDecisionInterna,
		audiencia: audienciaEvidenciaCobro,
		emitidaEn: instante.UTC(), recibidaEn: instante.UTC(),
	}
	nueva, repetida, err := o.aplicarHecho(nuevoHechoCobro(o, tipo, estado, datos, autorizacion, motivo))
	if errors.Is(err, ErrEvidenciaCobroConflictiva) {
		return o.bloquearPorIncidencia(datos, autorizacion, "Decision local reutilizada con otros datos", instante)
	}
	return nueva, repetida, err
}

func (o OrdenCobro) coincideEvidencia(datos datosEvidenciaCobro, exigeOperacion bool, evaluadaEn time.Time) error {
	if err := o.Validar(); err != nil {
		return err
	}
	if datos.ordenRef != o.ID || datos.liquidacionRef != o.LiquidacionRef ||
		!datos.importe.Igual(o.Importe) || datos.concepto != o.Concepto {
		return ErrCoincidenciaCobroInvalida
	}
	if exigeOperacion && (datos.conectorID != o.ConectorID || datos.versionConector != o.VersionConector ||
		datos.operacionProveedorRef != o.OperacionProveedorRef) {
		return ErrCoincidenciaCobroInvalida
	}
	if !exigeOperacion && o.Estado != EstadoCobroCreada &&
		(datos.conectorID != o.ConectorID || datos.versionConector != o.VersionConector ||
			datos.operacionProveedorRef != o.OperacionProveedorRef) {
		return ErrCoincidenciaCobroInvalida
	}
	if datos.recibidaEn.Before(o.CreadaEn) || datos.recibidaEn.After(evaluadaEn.Add(desfaseMaximoEvidenciaCobro)) ||
		evaluadaEn.Sub(datos.recibidaEn) > antiguedadMaximaEvidenciaCobro ||
		datos.recibidaEn.Sub(datos.emitidaEn) > antiguedadMaximaEvidenciaCobro {
		return ErrEvidenciaCobroInvalida
	}
	if !exigeOperacion && o.Estado == EstadoCobroCreada && !datos.recibidaEn.Before(o.CaducaEn) {
		return ErrTransicionCobroInvalida
	}
	return nil
}

func nuevoHechoCobro(orden OrdenCobro, tipo TipoHechoCobro, estado EstadoCobro, datos datosEvidenciaCobro, autorizacion ContextoAutorizacionCobro, motivo string) HechoCobro {
	var actorRef, perfilRef, autorizacionRef, atestacionRef, sesionRef, huellaSesionHMAC string
	var huellaDecision string
	var accion AccionCobro
	var metodo AuthMethod
	var garantia AuthAssurance
	var autorizacionEmitidaEn, autorizacionValidaHasta, autorizacionEvaluadaEn time.Time
	var atestacionEmitidaEn, atestacionValidaHasta, autenticacionVerificadaEn time.Time
	if autorizacion.datos != nil {
		actorRef = autorizacion.datos.ActorRef
		perfilRef = autorizacion.datos.PerfilActivoRef
		autorizacionRef = autorizacion.datos.DecisionRef
		huellaDecision = autorizacion.datos.HuellaDecisionSHA256
		autorizacionEmitidaEn = autorizacion.datos.VigenteDesde
		autorizacionValidaHasta = autorizacion.datos.VigenteHasta
		autorizacionEvaluadaEn = autorizacion.datos.EvaluadaEn
		atestacionRef = autorizacion.datos.AutenticacionRef
		atestacionEmitidaEn = autorizacion.datos.atestacion.emitidaEn
		atestacionValidaHasta = autorizacion.datos.atestacion.validaHasta
		autenticacionVerificadaEn = autorizacion.datos.atestacion.verificadaEn
		sesionRef = autorizacion.datos.SesionRef
		huellaSesionHMAC = autorizacion.datos.HuellaSesionHMAC
		metodo = autorizacion.datos.Metodo
		garantia = autorizacion.datos.Garantia
		accion = autorizacion.datos.Accion
	}
	ocurridoEn := datos.recibidaEn
	if !datos.ocurridoEn.IsZero() {
		ocurridoEn = datos.ocurridoEn
	}
	hecho := HechoCobro{
		VersionEsquemaIntegridad: orden.VersionEsquemaIntegridad,
		Secuencia:                int64(len(orden.Historial) + 1), Tipo: tipo, EstadoAnterior: orden.Estado,
		EstadoPosterior: estado, EvidenciaRef: datos.evidenciaRef,
		EvidenciaRelacionadaRef: datos.evidenciaRelacionadaRef,
		HuellaEvidenciaSHA256:   datos.huellaSHA256, ActorRef: actorRef,
		HuellaMensajeOriginalSHA256: datos.huellaMensajeOriginalSHA256,
		IndiceIdempotenciaHMAC:      datos.indiceIdempotenciaHMAC,
		PerfilActivoRef:             perfilRef,
		AccionAutorizada:            accion,
		AutorizacionRef:             autorizacionRef,
		HuellaDecisionSHA256:        huellaDecision,
		AutorizacionEmitidaEn:       autorizacionEmitidaEn,
		AutorizacionValidaHasta:     autorizacionValidaHasta,
		AutorizacionEvaluadaEn:      autorizacionEvaluadaEn,
		AtestacionAutenticacionRef:  atestacionRef,
		AtestacionEmitidaEn:         atestacionEmitidaEn,
		AtestacionValidaHasta:       atestacionValidaHasta,
		AutenticacionVerificadaEn:   autenticacionVerificadaEn,
		SesionRef:                   sesionRef,
		HuellaSesionHMAC:            huellaSesionHMAC,
		MetodoAutenticacion:         metodo,
		GarantiaAutenticacion:       garantia,
		CorrelacionRef:              orden.CorrelacionRef,
		ConectorID:                  datos.conectorID, VersionConector: datos.versionConector,
		OperacionProveedorRef: datos.operacionProveedorRef, DevolucionRef: datos.devolucionRef,
		ConciliacionRef: datos.conciliacionRef, Importe: orden.Importe,
		CodigoResultado: datos.codigo, Motivo: motivo, OcurridoEn: ocurridoEn,
	}
	if datos.huellaMensajeOriginalSHA256 != "" {
		hecho.VerificacionEvidenciaRef = datos.verificacionRef
		hecho.HuellaVerificacionSHA256 = datos.huellaVerificacionSHA256
		hecho.MetodoVerificacionEvidencia = datos.metodoAutenticacion
		hecho.AudienciaEvidencia = datos.audiencia
		hecho.EvidenciaEmitidaEn = datos.emitidaEn
		hecho.EvidenciaRecibidaEn = datos.recibidaEn
		hecho.EvidenciaVerificadaEn = datos.verificadaEn
	}
	return hecho
}

func (o OrdenCobro) aplicarHecho(hecho HechoCobro) (OrdenCobro, bool, error) {
	if err := o.Validar(); err != nil {
		return OrdenCobro{}, false, ErrTransicionCobroInvalida
	}
	hecho.HuellaEstadoAnteriorSHA256 = o.HuellaEstadoSHA256
	huellaPosterior, err := calcularHuellaHechoCobro(hecho)
	if err != nil {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	hecho.HuellaEstadoPosteriorSHA256 = huellaPosterior
	if err := hecho.Validar(); err != nil {
		return OrdenCobro{}, false, err
	}
	for _, anterior := range o.Historial {
		if anterior.EvidenciaRef == hecho.EvidenciaRef {
			if mismaIdentidadEvidenciaCobro(anterior, hecho) {
				return o.Clonar(), true, nil
			}
			return OrdenCobro{}, false, ErrEvidenciaCobroConflictiva
		}
	}
	if hecho.EstadoAnterior != o.Estado || hecho.Secuencia != int64(len(o.Historial)+1) ||
		!transicionHechoCobroPermitida(o.Estado, hecho.EstadoPosterior, hecho.Tipo) ||
		hecho.OcurridoEn.Before(o.UltimaActualizacionEn) {
		return OrdenCobro{}, false, ErrTransicionCobroInvalida
	}
	nueva := o.Clonar()
	nueva.Version++
	nueva.Estado = hecho.EstadoPosterior
	nueva.UltimaActualizacionEn = hecho.OcurridoEn.UTC()
	nueva.Historial = append(nueva.Historial, hecho)
	nueva.HuellaEstadoSHA256 = hecho.HuellaEstadoPosteriorSHA256
	return nueva, false, nil
}

func (o OrdenCobro) comprobarEvidenciaRepetida(datos datosEvidenciaCobro, tipo TipoHechoCobro) (bool, error) {
	candidata := nuevoHechoCobro(o, tipo, o.Estado, datos, ContextoAutorizacionCobro{}, "Comparacion interna")
	for _, anterior := range o.Historial {
		if anterior.EvidenciaRef == datos.evidenciaRef {
			if mismaIdentidadEvidenciaCobro(anterior, candidata) {
				return true, nil
			}
			return false, ErrEvidenciaCobroConflictiva
		}
	}
	return false, nil
}

// mismaIdentidadEvidenciaCobro compara todo el significado probado, no solo
// una referencia y una huella declarada. Omite deliberadamente la nueva
// decision de autorizacion y el texto operativo del reintento.
func mismaIdentidadEvidenciaCobro(anterior, candidata HechoCobro) bool {
	return anterior.Tipo == candidata.Tipo && anterior.EvidenciaRef == candidata.EvidenciaRef &&
		anterior.HuellaEvidenciaSHA256 == candidata.HuellaEvidenciaSHA256 &&
		anterior.HuellaMensajeOriginalSHA256 == candidata.HuellaMensajeOriginalSHA256 &&
		anterior.ConectorID == candidata.ConectorID && anterior.VersionConector == candidata.VersionConector &&
		anterior.OperacionProveedorRef == candidata.OperacionProveedorRef &&
		anterior.DevolucionRef == candidata.DevolucionRef && anterior.ConciliacionRef == candidata.ConciliacionRef &&
		anterior.IndiceIdempotenciaHMAC == candidata.IndiceIdempotenciaHMAC &&
		anterior.Importe.Igual(candidata.Importe) &&
		anterior.CodigoResultado == candidata.CodigoResultado && anterior.OcurridoEn.Equal(candidata.OcurridoEn)
}

func (o OrdenCobro) conservarEvidenciaAdicional(datos datosEvidenciaCobro, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error) {
	return o.aplicarHecho(nuevoHechoCobro(o, HechoCobroEvidenciaAdicional, o.Estado, datos,
		autorizacion, "Evidencia adicional: "+motivo))
}

func (o OrdenCobro) bloquearPorIncidencia(datos datosEvidenciaCobro, autorizacion ContextoAutorizacionCobro, motivo string, detectadaEn time.Time) (OrdenCobro, bool, error) {
	if o.Estado == EstadoCobroIncidenciaBloqueada {
		return OrdenCobro{}, false, ErrTransicionCobroInvalida
	}
	if detectadaEn.IsZero() || detectadaEn.Before(o.UltimaActualizacionEn) {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	datos.evidenciaRelacionadaRef = datos.evidenciaRef
	// La incidencia es un hecho nuevo con referencia propia. La referencia
	// conflictiva se conserva aparte para que ni un replay alterado pueda
	// impedir el bloqueo seguro por colision con una evidencia anterior.
	referenciaIncidencia, valida := referenciaIncidenciaCobro(datos.huellaSHA256)
	if !valida {
		return OrdenCobro{}, false, ErrEvidenciaCobroInvalida
	}
	datos.evidenciaRef = referenciaIncidencia
	datos.devolucionRef = o.DevolucionRef
	datos.conciliacionRef = ""
	datos.indiceIdempotenciaHMAC = ""
	datos.codigo = "resultado_conflictivo"
	datos.ocurridoEn = detectadaEn.UTC()
	return o.aplicarHecho(nuevoHechoCobro(o, HechoCobroIncidenciaDetectada, EstadoCobroIncidenciaBloqueada,
		datos, autorizacion, "Incidencia bloqueante: "+motivo))
}

func referenciaIncidenciaCobro(huella string) (string, bool) {
	if !esSHA256(huella) {
		return "", false
	}
	var codificada strings.Builder
	codificada.Grow(len("incidencia:") + len(huella))
	codificada.WriteString("incidencia:")
	for _, caracter := range huella {
		if caracter >= '0' && caracter <= '9' {
			codificada.WriteByte(byte('a' + caracter - '0'))
			continue
		}
		codificada.WriteByte(byte('k' + caracter - 'a'))
	}
	return codificada.String(), true
}

func tipoHechoResultadoCobro(resultado ResultadoOperacionCobro) TipoHechoCobro {
	switch resultado {
	case ResultadoOperacionCobroConfirmado:
		return HechoCobroConfirmado
	case ResultadoOperacionCobroRechazado:
		return HechoCobroRechazado
	case ResultadoOperacionCobroDesconocido:
		return HechoCobroResultadoDesconocido
	default:
		return HechoCobroResultadoPendiente
	}
}

func tipoHechoResultadoDevolucionCobro(resultado ResultadoDevolucionCobro) TipoHechoCobro {
	switch resultado {
	case ResultadoDevolucionCobroConfirmada:
		return HechoCobroDevuelto
	case ResultadoDevolucionCobroRechazada:
		return HechoCobroDevolucionRechazada
	case ResultadoDevolucionCobroPendiente:
		return HechoCobroDevolucionResultadoPendiente
	default:
		return HechoCobroDevolucionResultadoDesconocido
	}
}

func transicionHechoCobroPermitida(anterior, posterior EstadoCobro, tipo TipoHechoCobro) bool {
	if anterior == "" {
		return posterior == EstadoCobroCreada && tipo == HechoCobroOrdenCreada
	}
	if tipo == HechoCobroIncidenciaDetectada {
		return anterior.Valido() && anterior != EstadoCobroIncidenciaBloqueada && posterior == EstadoCobroIncidenciaBloqueada
	}
	if tipo == HechoCobroEvidenciaAdicional {
		return anterior.Valido() && anterior != EstadoCobroIncidenciaBloqueada && posterior == anterior
	}
	permitidas := map[TipoHechoCobro]struct {
		desde []EstadoCobro
		hacia EstadoCobro
	}{
		HechoCobroOperacionEnviada:               {[]EstadoCobro{EstadoCobroCreada}, EstadoCobroEnviadaPasarela},
		HechoCobroResultadoPendiente:             {[]EstadoCobro{EstadoCobroEnviadaPasarela, EstadoCobroResultadoDesconocido}, EstadoCobroResultadoPendiente},
		HechoCobroResultadoDesconocido:           {[]EstadoCobro{EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente}, EstadoCobroResultadoDesconocido},
		HechoCobroConfirmado:                     {[]EstadoCobro{EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente, EstadoCobroResultadoDesconocido}, EstadoCobroConfirmada},
		HechoCobroRechazado:                      {[]EstadoCobro{EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente, EstadoCobroResultadoDesconocido}, EstadoCobroRechazada},
		HechoCobroCancelado:                      {[]EstadoCobro{EstadoCobroCreada, EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente}, EstadoCobroCancelada},
		HechoCobroCaducado:                       {[]EstadoCobro{EstadoCobroCreada, EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente}, EstadoCobroCaducada},
		HechoCobroConciliado:                     {[]EstadoCobro{EstadoCobroConfirmada}, EstadoCobroConciliada},
		HechoCobroDevolucionSolicitada:           {[]EstadoCobro{EstadoCobroConfirmada, EstadoCobroConciliada, EstadoCobroDevolucionRechazada}, EstadoCobroDevolucionSolicitada},
		HechoCobroDevolucionResultadoPendiente:   {[]EstadoCobro{EstadoCobroDevolucionSolicitada}, EstadoCobroDevolucionSolicitada},
		HechoCobroDevolucionResultadoDesconocido: {[]EstadoCobro{EstadoCobroDevolucionSolicitada}, EstadoCobroDevolucionSolicitada},
		HechoCobroDevolucionRechazada:            {[]EstadoCobro{EstadoCobroDevolucionSolicitada}, EstadoCobroDevolucionRechazada},
		HechoCobroDevuelto:                       {[]EstadoCobro{EstadoCobroDevolucionSolicitada}, EstadoCobroDevuelta},
		HechoCobroDevolucionConciliada:           {[]EstadoCobro{EstadoCobroDevuelta}, EstadoCobroDevolucionConciliada},
	}
	regla, existe := permitidas[tipo]
	if !existe || regla.hacia != posterior {
		return false
	}
	for _, estado := range regla.desde {
		if estado == anterior {
			return true
		}
	}
	return false
}

// TuplaHechoCobroValida es la lista positiva de semanticas publicables. Evita
// que un hecho o un evento aislado combinen un nombre valido con el estado o
// la accion de otra operacion.
func TuplaHechoCobroValida(tipo TipoHechoCobro, estado EstadoCobro, accion AccionCobro) bool {
	switch tipo {
	case HechoCobroOrdenCreada:
		return estado == EstadoCobroCreada && accion == AccionCobroCrearOrden
	case HechoCobroOperacionEnviada:
		return estado == EstadoCobroEnviadaPasarela && accion == AccionCobroIniciarOperacion
	case HechoCobroResultadoPendiente:
		return estado == EstadoCobroResultadoPendiente && accion == AccionCobroProcesarResultado
	case HechoCobroResultadoDesconocido:
		return estado == EstadoCobroResultadoDesconocido && accion == AccionCobroProcesarResultado
	case HechoCobroConfirmado:
		return estado == EstadoCobroConfirmada && accion == AccionCobroProcesarResultado
	case HechoCobroRechazado:
		return estado == EstadoCobroRechazada && accion == AccionCobroProcesarResultado
	case HechoCobroCancelado:
		return estado == EstadoCobroCancelada && accion == AccionCobroCancelar
	case HechoCobroCaducado:
		return estado == EstadoCobroCaducada && accion == AccionCobroCaducar
	case HechoCobroConciliado:
		return estado == EstadoCobroConciliada && accion == AccionCobroConciliar
	case HechoCobroDevolucionConciliada:
		return estado == EstadoCobroDevolucionConciliada && accion == AccionCobroConciliar
	case HechoCobroDevolucionSolicitada:
		return estado == EstadoCobroDevolucionSolicitada && accion == AccionCobroSolicitarDevolucion
	case HechoCobroDevolucionResultadoPendiente, HechoCobroDevolucionResultadoDesconocido:
		return estado == EstadoCobroDevolucionSolicitada && accion == AccionCobroProcesarDevolucion
	case HechoCobroDevolucionRechazada:
		return estado == EstadoCobroDevolucionRechazada && accion == AccionCobroProcesarDevolucion
	case HechoCobroDevuelto:
		return estado == EstadoCobroDevuelta && accion == AccionCobroProcesarDevolucion
	case HechoCobroIncidenciaDetectada:
		if estado != EstadoCobroIncidenciaBloqueada {
			return false
		}
		switch accion {
		case AccionCobroIniciarOperacion, AccionCobroProcesarResultado, AccionCobroSolicitarDevolucion,
			AccionCobroProcesarDevolucion, AccionCobroConciliar, AccionCobroCancelar, AccionCobroCaducar:
			return true
		default:
			return false
		}
	case HechoCobroEvidenciaAdicional:
		return tuplaEvidenciaAdicionalCobroValida(estado, accion)
	default:
		return false
	}
}

func tuplaEvidenciaAdicionalCobroValida(estado EstadoCobro, accion AccionCobro) bool {
	switch accion {
	case AccionCobroIniciarOperacion:
		switch estado {
		case EstadoCobroEnviadaPasarela, EstadoCobroResultadoPendiente, EstadoCobroResultadoDesconocido,
			EstadoCobroConfirmada, EstadoCobroConciliada, EstadoCobroRechazada, EstadoCobroCancelada,
			EstadoCobroCaducada, EstadoCobroDevolucionSolicitada, EstadoCobroDevolucionRechazada,
			EstadoCobroDevuelta, EstadoCobroDevolucionConciliada:
			return true
		default:
			return false
		}
	case AccionCobroProcesarResultado:
		switch estado {
		case EstadoCobroResultadoPendiente, EstadoCobroResultadoDesconocido, EstadoCobroConfirmada,
			EstadoCobroConciliada, EstadoCobroRechazada, EstadoCobroDevolucionSolicitada,
			EstadoCobroDevolucionRechazada, EstadoCobroDevuelta, EstadoCobroDevolucionConciliada:
			return true
		default:
			return false
		}
	case AccionCobroProcesarDevolucion:
		switch estado {
		case EstadoCobroDevolucionSolicitada, EstadoCobroDevolucionRechazada, EstadoCobroDevuelta,
			EstadoCobroDevolucionConciliada:
			return true
		default:
			return false
		}
	case AccionCobroConciliar:
		return estado == EstadoCobroConciliada || estado == EstadoCobroDevolucionConciliada
	default:
		return false
	}
}

func estadoCobroTieneDevolucion(estado EstadoCobro) bool {
	return estado == EstadoCobroDevolucionSolicitada || estado == EstadoCobroDevolucionRechazada ||
		estado == EstadoCobroDevuelta || estado == EstadoCobroDevolucionConciliada
}

func hechoCobroEsRemoto(h HechoCobro) bool {
	switch h.Tipo {
	case HechoCobroOperacionEnviada, HechoCobroResultadoPendiente, HechoCobroResultadoDesconocido,
		HechoCobroConfirmado, HechoCobroRechazado, HechoCobroConciliado,
		HechoCobroDevolucionResultadoPendiente, HechoCobroDevolucionResultadoDesconocido,
		HechoCobroDevolucionRechazada, HechoCobroDevuelto, HechoCobroDevolucionConciliada,
		HechoCobroEvidenciaAdicional:
		return true
	case HechoCobroIncidenciaDetectada:
		switch h.AccionAutorizada {
		case AccionCobroIniciarOperacion, AccionCobroProcesarResultado,
			AccionCobroProcesarDevolucion, AccionCobroConciliar:
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// matrizCamposHechoCobroValida define campos obligatorios y prohibidos por
// procedencia. Un hecho remoto siempre conserva la atestacion exacta del
// verificador; un hecho local no puede aparentar haber sido emitido por una
// pasarela.
func matrizCamposHechoCobroValida(h HechoCobro) bool {
	remoto := hechoCobroEsRemoto(h)
	tieneConector := h.ConectorID != "" || h.VersionConector != 0 || h.OperacionProveedorRef != ""
	tieneVerificacion := h.VerificacionEvidenciaRef != "" || h.HuellaVerificacionSHA256 != "" ||
		h.MetodoVerificacionEvidencia != "" || h.AudienciaEvidencia != "" ||
		!h.EvidenciaEmitidaEn.IsZero() || !h.EvidenciaRecibidaEn.IsZero() || !h.EvidenciaVerificadaEn.IsZero()
	if !remoto {
		return !tieneConector && h.HuellaMensajeOriginalSHA256 == "" && !tieneVerificacion
	}
	if !tieneConector || !esClaveDocumentalCanonica(h.ConectorID) || h.VersionConector < 1 ||
		!referenciaCobroValida(h.OperacionProveedorRef) || !esSHA256(h.HuellaMensajeOriginalSHA256) ||
		!referenciaCobroValida(h.VerificacionEvidenciaRef) || !esSHA256(h.HuellaVerificacionSHA256) ||
		!h.MetodoVerificacionEvidencia.Valido() || h.AudienciaEvidencia != audienciaEvidenciaCobro ||
		h.EvidenciaEmitidaEn.IsZero() || h.EvidenciaRecibidaEn.Before(h.EvidenciaEmitidaEn) ||
		h.EvidenciaVerificadaEn.Before(h.EvidenciaRecibidaEn) ||
		h.EvidenciaVerificadaEn.Sub(h.EvidenciaRecibidaEn) > desfaseMaximoEvidenciaCobro {
		return false
	}
	if h.Tipo == HechoCobroIncidenciaDetectada {
		return !h.OcurridoEn.Before(h.EvidenciaRecibidaEn)
	}
	return h.OcurridoEn.Equal(h.EvidenciaRecibidaEn)
}

func metodoAutenticacionCobroPermitido(metodo AuthMethod) bool {
	switch metodo {
	case AuthMethodCertificate, AuthMethodDNIe, AuthMethodSSO, AuthMethodClave, AuthMethodKerberos:
		return true
	default:
		return false
	}
}

func garantiaAutenticacionCobroPermitida(garantia AuthAssurance) bool {
	switch garantia {
	case AuthAssuranceLow, AuthAssuranceSubstantial, AuthAssuranceHigh:
		return true
	default:
		return false
	}
}

func esHuellaSesionCobro(valor string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] == "sesion-v1" && esSHA256(partes[2])
}

func esHuellaHMACCobroDeDominio(valor, dominio string) bool {
	partes := strings.Split(valor, ":")
	return len(partes) == 3 && partes[0] == "hmac-sha256" && partes[1] == dominio && esSHA256(partes[2])
}

func referenciaCobroValida(valor string) bool {
	return referenciaDocumentalValida(valor) && !contieneDatoTarjetaCobro(valor)
}

func referenciaEvidenciaEntradaCobroValida(valor string) bool {
	return referenciaCobroValida(valor) && !strings.HasPrefix(valor, "incidencia:")
}

func textoCobroValido(valor string, maximo int) bool {
	return valor == strings.TrimSpace(valor) && valor != "" && len(valor) <= maximo &&
		textoDocumentalValido(valor) && !contieneDatoTarjetaCobro(valor)
}

func contieneDatoTarjetaCobro(valores ...string) bool {
	for _, valor := range valores {
		minusculas := strings.Map(func(caracter rune) rune {
			if unicode.Is(unicode.Cf, caracter) {
				return -1
			}
			return unicode.ToLower(caracter)
		}, valor)
		palabras := strings.FieldsFunc(minusculas, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
		for _, palabra := range palabras {
			switch palabra {
			case "pan", "cvv", "cvc", "cvn", "pin", "criptograma", "cryptogram", "tarjeta", "card", "cardnumber":
				return true
			}
		}
		digitos := make([]byte, 0, 32)
		comprobar := func() bool {
			for longitud := 13; longitud <= 19 && longitud <= len(digitos); longitud++ {
				for inicio := 0; inicio+longitud <= len(digitos); inicio++ {
					if numeroTarjetaLuhnValido(digitos[inicio : inicio+longitud]) {
						return true
					}
				}
			}
			return false
		}
		for _, caracter := range valor {
			if numero, esDigito := valorDigitoDecimalCobro(caracter); esDigito {
				digitos = append(digitos, byte('0'+numero))
				continue
			}
			if (unicode.IsSpace(caracter) || unicode.Is(unicode.Dash, caracter) ||
				unicode.Is(unicode.Cf, caracter) || caracter == '.') && len(digitos) > 0 {
				continue
			}
			if comprobar() {
				return true
			}
			digitos = digitos[:0]
		}
		if comprobar() {
			return true
		}
	}
	return false
}

func valorDigitoDecimalCobro(caracter rune) (byte, bool) {
	switch {
	case caracter >= '0' && caracter <= '9':
		return byte(caracter - '0'), true
	case caracter >= '\u0660' && caracter <= '\u0669':
		return byte(caracter - '\u0660'), true
	case caracter >= '\u06f0' && caracter <= '\u06f9':
		return byte(caracter - '\u06f0'), true
	case caracter >= '\uff10' && caracter <= '\uff19':
		return byte(caracter - '\uff10'), true
	default:
		return 0, false
	}
}

func numeroTarjetaLuhnValido(digitos []byte) bool {
	suma := 0
	par := len(digitos)%2 == 0
	for indice, caracter := range digitos {
		numero := int(caracter - '0')
		if (indice%2 == 0) == par {
			numero *= 2
			if numero > 9 {
				numero -= 9
			}
		}
		suma += numero
	}
	return suma > 0 && suma%10 == 0
}

// Mantiene la importacion de encoding/json ligada al contrato de rechazo de
// serializacion y permite una comprobacion estatica de la interfaz.
var (
	_ json.Marshaler = EvidenciaInicioOperacionCobro{}
	_ json.Marshaler = EvidenciaResultadoCobro{}
	_ json.Marshaler = EvidenciaResultadoDevolucionCobro{}
	_ json.Marshaler = EvidenciaConciliacionCobro{}
)
