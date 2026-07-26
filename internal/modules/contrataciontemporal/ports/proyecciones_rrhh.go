package ports

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	LimiteMaximoCuadroRRHH = 100

	// Estas acciones y finalidades son vocabulario cerrado del puerto. Su
	// activación productiva queda sujeta a publicación en catálogo y a una
	// política PDP aprobada por RRHH/DPD. No representan efectos de negocio.
	AccionConsultarCuadroRRHH     = "contratacion_temporal.cuadro.consultar"
	FinalidadConsultarCuadroRRHH  = "gestion_operativa_contratacion_temporal"
	AccionConsultarDetalleRRHH    = "contratacion_temporal.expediente.consultar"
	FinalidadConsultarDetalleRRHH = "tramitacion_expediente_contratacion_temporal"
)

var (
	ErrSolicitudConsultaRRHHInvalida = errors.New(
		"contratacion temporal: solicitud de consulta RRHH invalida",
	)
	ErrContextoConsultaRRHHInvalido = errors.New(
		"contratacion temporal: contexto de consulta RRHH invalido",
	)
	ErrCapacidadConsultaRRHHInvalida = errors.New(
		"contratacion temporal: capacidad de consulta RRHH invalida",
	)
	ErrOrdenConsultaRRHHInvalida = errors.New(
		"contratacion temporal: orden de consulta RRHH invalida",
	)
	ErrConsultaRRHHNoObservable = errors.New(
		"contratacion temporal: consulta RRHH no observable",
	)
	ErrConsultaRRHHNoDisponible = errors.New(
		"contratacion temporal: consulta RRHH no disponible",
	)
	ErrResultadoConsultaRRHHNoConfiable = errors.New(
		"contratacion temporal: resultado de consulta RRHH no confiable",
	)
	ErrMaterialConsultaRRHHSensible = errors.New(
		"contratacion temporal: material de consulta RRHH sensible",
	)
)

var (
	patronTextoCuadroRRHH = regexp.MustCompile(`^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$`)
	patronCursorRRHH      = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	patronHuellaRRHH      = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type ClaseAmbitoConsultaRRHH string

const (
	AmbitoOrganizacionRRHH  ClaseAmbitoConsultaRRHH = "organizacion"
	AmbitoCentroRRHH        ClaseAmbitoConsultaRRHH = "centro"
	AmbitoUnidadGestionRRHH ClaseAmbitoConsultaRRHH = "unidad_gestion"
)

func (c ClaseAmbitoConsultaRRHH) valida() bool {
	return c == AmbitoOrganizacionRRHH ||
		c == AmbitoCentroRRHH ||
		c == AmbitoUnidadGestionRRHH
}

// ContextoConsultaRRHH solo puede proceder de una autoridad ligada al canal.
// Sus campos son privados, no se serializa y su representación está redactada.
type ContextoConsultaRRHH struct {
	autenticacionRef string
	sesionRef        string
	actorRef         string
	perfilRef        string
	organizacionRef  string
	resueltoEn       time.Time
	validoHasta      time.Time
}

func NuevoContextoConsultaRRHH(
	autenticacionRef, sesionRef, actorRef, perfilRef, organizacionRef string,
	resueltoEn, validoHasta time.Time,
) (ContextoConsultaRRHH, error) {
	c := ContextoConsultaRRHH{
		autenticacionRef: autenticacionRef,
		sesionRef:        sesionRef,
		actorRef:         actorRef,
		perfilRef:        perfilRef,
		organizacionRef:  organizacionRef,
		resueltoEn:       resueltoEn,
		validoHasta:      validoHasta,
	}
	if c.validarEn(resueltoEn) != nil {
		return ContextoConsultaRRHH{}, ErrContextoConsultaRRHHInvalido
	}
	return c, nil
}

func (c ContextoConsultaRRHH) validarEn(instante time.Time) error {
	if !domain.ReferenciaOpacaValida(c.autenticacionRef) ||
		!domain.ReferenciaOpacaValida(c.sesionRef) ||
		!domain.ReferenciaOpacaValida(c.actorRef) ||
		!domain.ReferenciaOpacaValida(c.perfilRef) ||
		!domain.ReferenciaOpacaValida(c.organizacionRef) ||
		!domain.InstanteUTCCanonico(c.resueltoEn) ||
		!domain.InstanteUTCCanonico(c.validoHasta) ||
		!domain.InstanteUTCCanonico(instante) ||
		!c.validoHasta.After(c.resueltoEn) ||
		instante.Before(c.resueltoEn) || !instante.Before(c.validoHasta) {
		return ErrContextoConsultaRRHHInvalido
	}
	return nil
}

func (c ContextoConsultaRRHH) AutenticacionRef() string { return c.autenticacionRef }
func (c ContextoConsultaRRHH) SesionRef() string        { return c.sesionRef }
func (c ContextoConsultaRRHH) ActorRef() string         { return c.actorRef }
func (c ContextoConsultaRRHH) PerfilRef() string        { return c.perfilRef }
func (c ContextoConsultaRRHH) OrganizacionRef() string  { return c.organizacionRef }
func (c ContextoConsultaRRHH) ResueltoEn() time.Time    { return c.resueltoEn }
func (c ContextoConsultaRRHH) ValidoHasta() time.Time   { return c.validoHasta }
func (ContextoConsultaRRHH) String() string             { return "[contexto-consulta-rrhh-redactado]" }
func (ContextoConsultaRRHH) GoString() string           { return "[contexto-consulta-rrhh-redactado]" }
func (ContextoConsultaRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}

type SolicitudCuadroRRHH struct {
	texto       string
	estadoClave domain.EstadoOperativo
	faseClave   domain.ClaveFase
	limite      uint16
	cursor      string
}

func NuevaSolicitudCuadroRRHH(
	texto string,
	estadoClave domain.EstadoOperativo,
	faseClave domain.ClaveFase,
	limite uint16,
	cursor string,
) (SolicitudCuadroRRHH, error) {
	s := SolicitudCuadroRRHH{
		texto:       texto,
		estadoClave: estadoClave,
		faseClave:   faseClave,
		limite:      limite,
		cursor:      cursor,
	}
	if s.validar() != nil {
		return SolicitudCuadroRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	return s, nil
}

func (s SolicitudCuadroRRHH) validar() error {
	if s.texto != strings.TrimSpace(s.texto) ||
		!utf8.ValidString(s.texto) ||
		!patronTextoCuadroRRHH.MatchString(s.texto) ||
		(s.estadoClave != "" && !s.estadoClave.Valido()) ||
		(s.faseClave != "" && !s.faseClave.Valida()) ||
		s.limite < 1 || s.limite > LimiteMaximoCuadroRRHH ||
		(s.cursor != "" && !cursorRRHHValido(s.cursor)) {
		return ErrSolicitudConsultaRRHHInvalida
	}
	return nil
}

func cursorRRHHValido(cursor string) bool {
	return len(cursor) >= 32 && len(cursor) <= 2048 &&
		patronCursorRRHH.MatchString(cursor)
}

func (s SolicitudCuadroRRHH) Texto() string                       { return s.texto }
func (s SolicitudCuadroRRHH) EstadoClave() domain.EstadoOperativo { return s.estadoClave }
func (s SolicitudCuadroRRHH) FaseClave() domain.ClaveFase         { return s.faseClave }
func (s SolicitudCuadroRRHH) Limite() uint16                      { return s.limite }
func (s SolicitudCuadroRRHH) Cursor() string                      { return s.cursor }
func (SolicitudCuadroRRHH) String() string                        { return "[solicitud-cuadro-rrhh-redactada]" }
func (SolicitudCuadroRRHH) GoString() string                      { return "[solicitud-cuadro-rrhh-redactada]" }
func (SolicitudCuadroRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}

type SolicitudDetalleRRHH struct {
	expedienteRef    string
	versionObservada uint64
}

// NuevaSolicitudDetalleRRHH interpreta versionObservada=0 como primera carga.
// Cualquier versión no nula exige una lectura exactamente coincidente.
func NuevaSolicitudDetalleRRHH(
	expedienteRef string,
	versionObservada uint64,
) (SolicitudDetalleRRHH, error) {
	s := SolicitudDetalleRRHH{
		expedienteRef:    expedienteRef,
		versionObservada: versionObservada,
	}
	if s.validar() != nil {
		return SolicitudDetalleRRHH{}, ErrSolicitudConsultaRRHHInvalida
	}
	return s, nil
}

func (s SolicitudDetalleRRHH) validar() error {
	if !domain.ReferenciaOpacaValida(s.expedienteRef) ||
		s.versionObservada > 9_007_199_254_740_991 {
		return ErrSolicitudConsultaRRHHInvalida
	}
	return nil
}

func (s SolicitudDetalleRRHH) ExpedienteRef() string    { return s.expedienteRef }
func (s SolicitudDetalleRRHH) VersionObservada() uint64 { return s.versionObservada }
func (SolicitudDetalleRRHH) String() string             { return "[solicitud-detalle-rrhh-redactada]" }
func (SolicitudDetalleRRHH) GoString() string           { return "[solicitud-detalle-rrhh-redactada]" }
func (SolicitudDetalleRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
