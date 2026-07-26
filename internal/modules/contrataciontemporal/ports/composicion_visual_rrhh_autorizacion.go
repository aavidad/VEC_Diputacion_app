package ports

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const DuracionMaximaCapacidadComposicionVisualRRHH = 5 * time.Minute

var (
	ErrSolicitudComposicionVisualRRHHInvalida = errors.New(
		"contratacion temporal: solicitud de composicion visual RRHH invalida",
	)
	ErrVocabularioComposicionVisualRRHHInvalido = errors.New(
		"contratacion temporal: vocabulario de composicion visual RRHH invalido",
	)
	ErrCapacidadComposicionVisualRRHHInvalida = errors.New(
		"contratacion temporal: capacidad de composicion visual RRHH invalida",
	)
	ErrOrdenComposicionVisualRRHHInvalida = errors.New(
		"contratacion temporal: orden de composicion visual RRHH invalida",
	)
	ErrComposicionVisualRRHHNoObservable = errors.New(
		"contratacion temporal: composicion visual RRHH no observable",
	)
	ErrComposicionVisualRRHHNoDisponible = errors.New(
		"contratacion temporal: composicion visual RRHH no disponible",
	)
	ErrResultadoComposicionVisualRRHHNoConfiable = errors.New(
		"contratacion temporal: composicion visual RRHH no confiable",
	)
	ErrPublicacionesVisualesRRHHNoAtestadas = errors.New(
		"contratacion temporal: publicaciones visuales RRHH no atestadas",
	)
	ErrMaterialComposicionVisualRRHHSensible = errors.New(
		"contratacion temporal: material de composicion visual RRHH sensible",
	)
)

var patronVocabularioComposicionVisualRRHH = regexp.MustCompile(
	`^[a-z][a-z0-9_.-]{2,159}$`,
)

// VocabularioComposicionVisualRRHH mantiene acción y finalidad fuera del
// binario funcional. La raíz solo podrá construirlo con las claves exactas
// publicadas por el gobierno de autorización.
type VocabularioComposicionVisualRRHH struct {
	accion    string
	finalidad string
}

func NuevoVocabularioComposicionVisualRRHH(
	accion, finalidad string,
) (VocabularioComposicionVisualRRHH, error) {
	v := VocabularioComposicionVisualRRHH{
		accion: accion, finalidad: finalidad,
	}
	if v.validar() != nil {
		return VocabularioComposicionVisualRRHH{},
			ErrVocabularioComposicionVisualRRHHInvalido
	}
	return v, nil
}

func (v VocabularioComposicionVisualRRHH) validar() error {
	if !claveVocabularioComposicionVisualValida(v.accion) ||
		!claveVocabularioComposicionVisualValida(v.finalidad) ||
		v.accion == v.finalidad {
		return ErrVocabularioComposicionVisualRRHHInvalido
	}
	return nil
}

func claveVocabularioComposicionVisualValida(valor string) bool {
	return valor == strings.TrimSpace(valor) &&
		strings.ContainsRune(valor, '.') &&
		patronVocabularioComposicionVisualRRHH.MatchString(valor)
}

func (v VocabularioComposicionVisualRRHH) Accion() string    { return v.accion }
func (v VocabularioComposicionVisualRRHH) Finalidad() string { return v.finalidad }

type SolicitudComposicionVisualRRHH struct {
	flujoRef     string
	flujoVersion uint64
}

func NuevaSolicitudComposicionVisualRRHH(
	flujoRef string,
	flujoVersion uint64,
) (SolicitudComposicionVisualRRHH, error) {
	s := SolicitudComposicionVisualRRHH{
		flujoRef: flujoRef, flujoVersion: flujoVersion,
	}
	if s.validar() != nil {
		return SolicitudComposicionVisualRRHH{},
			ErrSolicitudComposicionVisualRRHHInvalida
	}
	return s, nil
}

func (s SolicitudComposicionVisualRRHH) validar() error {
	if !domain.ReferenciaOpacaValida(s.flujoRef) ||
		s.flujoVersion < 1 || s.flujoVersion > versionMaximaJSONSegura {
		return ErrSolicitudComposicionVisualRRHHInvalida
	}
	return nil
}

func (s SolicitudComposicionVisualRRHH) FlujoRef() string     { return s.flujoRef }
func (s SolicitudComposicionVisualRRHH) FlujoVersion() uint64 { return s.flujoVersion }
func (SolicitudComposicionVisualRRHH) String() string {
	return "[solicitud-composicion-visual-rrhh-redactada]"
}
func (SolicitudComposicionVisualRRHH) GoString() string {
	return "[solicitud-composicion-visual-rrhh-redactada]"
}
func (SolicitudComposicionVisualRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialComposicionVisualRRHHSensible
}

// CapacidadComposicionVisualRRHH es una concesión breve para una definición
// exacta. No contiene ni concede capacidades de efectos mostradas por la UI.
type CapacidadComposicionVisualRRHH struct {
	decisionRef     string
	correlacionRef  string
	motivoRef       string
	actorRef        string
	sesionRef       string
	perfilRef       string
	organizacionRef string
	claseAmbito     ClaseAmbitoConsultaRRHH
	ambitoRef       string
	accion          string
	finalidad       string
	flujoRef        string
	flujoVersion    uint64
	validaDesde     time.Time
	validaHasta     time.Time
}

func NuevaCapacidadComposicionVisualRRHH(
	decisionRef, correlacionRef, motivoRef string,
	contexto ContextoConsultaRRHH,
	claseAmbito ClaseAmbitoConsultaRRHH,
	ambitoRef string,
	vocabulario VocabularioComposicionVisualRRHH,
	solicitud SolicitudComposicionVisualRRHH,
	validaDesde, validaHasta time.Time,
) (CapacidadComposicionVisualRRHH, error) {
	c := CapacidadComposicionVisualRRHH{
		decisionRef: decisionRef, correlacionRef: correlacionRef,
		motivoRef: motivoRef, actorRef: contexto.actorRef,
		sesionRef: contexto.sesionRef, perfilRef: contexto.perfilRef,
		organizacionRef: contexto.organizacionRef,
		claseAmbito:     claseAmbito, ambitoRef: ambitoRef,
		accion: vocabulario.accion, finalidad: vocabulario.finalidad,
		flujoRef: solicitud.flujoRef, flujoVersion: solicitud.flujoVersion,
		validaDesde: validaDesde, validaHasta: validaHasta,
	}
	if contexto.validarEn(validaDesde) != nil ||
		validaHasta.After(contexto.validoHasta) ||
		c.validarEstructura() != nil {
		return CapacidadComposicionVisualRRHH{},
			ErrCapacidadComposicionVisualRRHHInvalida
	}
	return c, nil
}

func (c CapacidadComposicionVisualRRHH) validarEstructura() error {
	if !domain.ReferenciaOpacaValida(c.decisionRef) ||
		!domain.ReferenciaOpacaValida(c.correlacionRef) ||
		!domain.ReferenciaOpacaValida(c.motivoRef) ||
		!domain.ReferenciaOpacaValida(c.actorRef) ||
		!domain.ReferenciaOpacaValida(c.sesionRef) ||
		!domain.ReferenciaOpacaValida(c.perfilRef) ||
		!domain.ReferenciaOpacaValida(c.organizacionRef) ||
		!c.claseAmbito.valida() ||
		!domain.ReferenciaOpacaValida(c.ambitoRef) ||
		(c.claseAmbito == AmbitoOrganizacionRRHH &&
			c.ambitoRef != c.organizacionRef) ||
		!claveVocabularioComposicionVisualValida(c.accion) ||
		!claveVocabularioComposicionVisualValida(c.finalidad) ||
		c.accion == c.finalidad ||
		!domain.ReferenciaOpacaValida(c.flujoRef) ||
		c.flujoVersion < 1 || c.flujoVersion > versionMaximaJSONSegura ||
		!domain.InstanteUTCCanonico(c.validaDesde) ||
		!domain.InstanteUTCCanonico(c.validaHasta) ||
		!c.validaHasta.After(c.validaDesde) ||
		c.validaHasta.Sub(c.validaDesde) >
			DuracionMaximaCapacidadComposicionVisualRRHH {
		return ErrCapacidadComposicionVisualRRHHInvalida
	}
	return nil
}

func (c CapacidadComposicionVisualRRHH) validaPara(
	contexto ContextoConsultaRRHH,
	vocabulario VocabularioComposicionVisualRRHH,
	solicitud SolicitudComposicionVisualRRHH,
	instante time.Time,
) error {
	if vocabulario.validar() != nil || solicitud.validar() != nil ||
		c.validarEstructura() != nil || contexto.validarEn(instante) != nil ||
		c.actorRef != contexto.actorRef ||
		c.sesionRef != contexto.sesionRef ||
		c.perfilRef != contexto.perfilRef ||
		c.organizacionRef != contexto.organizacionRef ||
		c.accion != vocabulario.accion ||
		c.finalidad != vocabulario.finalidad ||
		c.flujoRef != solicitud.flujoRef ||
		c.flujoVersion != solicitud.flujoVersion ||
		instante.Before(c.validaDesde) || !instante.Before(c.validaHasta) {
		return ErrCapacidadComposicionVisualRRHHInvalida
	}
	return nil
}

func (c CapacidadComposicionVisualRRHH) DecisionRef() string {
	return c.decisionRef
}
func (c CapacidadComposicionVisualRRHH) CorrelacionRef() string {
	return c.correlacionRef
}
func (c CapacidadComposicionVisualRRHH) OrganizacionRef() string {
	return c.organizacionRef
}
func (c CapacidadComposicionVisualRRHH) ClaseAmbito() ClaseAmbitoConsultaRRHH {
	return c.claseAmbito
}
func (c CapacidadComposicionVisualRRHH) AmbitoRef() string {
	return c.ambitoRef
}
func (c CapacidadComposicionVisualRRHH) FlujoRef() string {
	return c.flujoRef
}
func (c CapacidadComposicionVisualRRHH) FlujoVersion() uint64 {
	return c.flujoVersion
}
func (c CapacidadComposicionVisualRRHH) ValidaHasta() time.Time {
	return c.validaHasta
}
func (CapacidadComposicionVisualRRHH) String() string {
	return "[capacidad-composicion-visual-rrhh-redactada]"
}
func (CapacidadComposicionVisualRRHH) GoString() string {
	return "[capacidad-composicion-visual-rrhh-redactada]"
}
func (CapacidadComposicionVisualRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialComposicionVisualRRHHSensible
}

type OrdenConsultaComposicionVisualRRHH struct {
	contexto    ContextoConsultaRRHH
	capacidad   CapacidadComposicionVisualRRHH
	vocabulario VocabularioComposicionVisualRRHH
	solicitud   SolicitudComposicionVisualRRHH
	instante    time.Time
}

func NuevaOrdenConsultaComposicionVisualRRHH(
	contexto ContextoConsultaRRHH,
	capacidad CapacidadComposicionVisualRRHH,
	vocabulario VocabularioComposicionVisualRRHH,
	solicitud SolicitudComposicionVisualRRHH,
	instante time.Time,
) (OrdenConsultaComposicionVisualRRHH, error) {
	if capacidad.validaPara(
		contexto, vocabulario, solicitud, instante,
	) != nil {
		return OrdenConsultaComposicionVisualRRHH{},
			ErrOrdenComposicionVisualRRHHInvalida
	}
	return OrdenConsultaComposicionVisualRRHH{
		contexto: contexto, capacidad: capacidad,
		vocabulario: vocabulario, solicitud: solicitud, instante: instante,
	}, nil
}

func (o OrdenConsultaComposicionVisualRRHH) Contexto() ContextoConsultaRRHH {
	return o.contexto
}
func (o OrdenConsultaComposicionVisualRRHH) Capacidad() CapacidadComposicionVisualRRHH {
	return o.capacidad
}
func (o OrdenConsultaComposicionVisualRRHH) Vocabulario() VocabularioComposicionVisualRRHH {
	return o.vocabulario
}
func (o OrdenConsultaComposicionVisualRRHH) Solicitud() SolicitudComposicionVisualRRHH {
	return o.solicitud
}
func (o OrdenConsultaComposicionVisualRRHH) Instante() time.Time { return o.instante }
func (OrdenConsultaComposicionVisualRRHH) String() string {
	return "[orden-consulta-composicion-visual-rrhh-redactada]"
}
func (OrdenConsultaComposicionVisualRRHH) GoString() string {
	return "[orden-consulta-composicion-visual-rrhh-redactada]"
}
func (OrdenConsultaComposicionVisualRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialComposicionVisualRRHHSensible
}

type AutorizadorComposicionVisualRRHH interface {
	AutorizarComposicionVisualRRHH(
		context.Context,
		ContextoConsultaRRHH,
		VocabularioComposicionVisualRRHH,
		SolicitudComposicionVisualRRHH,
		time.Time,
	) (CapacidadComposicionVisualRRHH, error)
}

type SesionComposicionVisualRRHH interface {
	ConsultarComposicionVisualYRegistrar(
		context.Context,
		OrdenConsultaComposicionVisualRRHH,
	) (ComposicionVisualRRHH, error)
}

// AutoridadPublicacionesVisualesRRHH es independiente de la sesión fuente.
// Debe cotejar el registro gobernado en su transacción durable, revalidar la
// vigencia de la capacidad y registrar allí el resultado de la atestación.
type AutoridadPublicacionesVisualesRRHH interface {
	AtestarPublicacionesVisualesYRegistrar(
		context.Context,
		SolicitudAtestacionPublicacionesVisualesRRHH,
	) error
}

var (
	_ json.Marshaler = SolicitudComposicionVisualRRHH{}
	_ json.Marshaler = CapacidadComposicionVisualRRHH{}
	_ json.Marshaler = OrdenConsultaComposicionVisualRRHH{}
	_ fmt.Stringer   = SolicitudComposicionVisualRRHH{}
	_ fmt.Stringer   = CapacidadComposicionVisualRRHH{}
	_ fmt.Stringer   = OrdenConsultaComposicionVisualRRHH{}
)
