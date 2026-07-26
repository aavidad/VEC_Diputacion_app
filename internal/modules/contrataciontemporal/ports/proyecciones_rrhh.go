package ports

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
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
	base64URLCursorRRHHEstricto = base64.RawURLEncoding.Strict()
	patronTextoCuadroRRHH       = regexp.MustCompile(`^[0-9A-Za-zÁÉÍÓÚÜÑáéíóúüñ/._ -]{0,80}$`)
	patronHuellaRRHH            = regexp.MustCompile(`^[0-9a-f]{64}$`)
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
	bloqueoSerializacionConsultaRRHH
	autenticacionRef          string
	autenticacionHuella       string
	sesionRef                 string
	controlSesionRef          string
	controlSesionRevision     uint64
	controlSesionHuellaSHA256 string
	actorRef                  string
	perfilRef                 string
	perfilVersion             uint64
	registroContextoRef       string
	contextoActorHuella       string
	organizacionRef           string
	resueltoEn                time.Time
	validoHasta               time.Time
}

func NuevoContextoConsultaRRHH(
	autoridad ContextoAutorizacionAltaV3,
	organizacionRef string,
	instante time.Time,
) (ContextoConsultaRRHH, error) {
	datosVinculo, err := autoridad.Vinculo.Datos()
	solicitud := SolicitudResolverContextoAutorizacionAltaV3{
		AutenticacionRef: datosVinculo.AutenticacionRef,
		SesionRef:        datosVinculo.SesionRef,
		PerfilRef:        datosVinculo.PerfilActivoRef,
	}
	if err != nil || autoridad.ValidarPara(solicitud, instante) != nil ||
		!domain.ReferenciaOpacaValida(organizacionRef) {
		return ContextoConsultaRRHH{}, ErrContextoConsultaRRHHInvalido
	}
	resultado, err := autoridad.Resultado.Clonar()
	if err != nil {
		return ContextoConsultaRRHH{}, ErrContextoConsultaRRHHInvalido
	}
	validoHasta := limiteVigenciaContextoConsultaRRHH(datosVinculo, resultado)
	resueltoEn := inicioVigenciaContextoConsultaRRHH(datosVinculo, resultado)
	c := ContextoConsultaRRHH{
		autenticacionRef:          datosVinculo.AutenticacionRef,
		autenticacionHuella:       datosVinculo.AutenticacionHuellaSHA256,
		sesionRef:                 datosVinculo.SesionRef,
		controlSesionRef:          datosVinculo.ControlSesionRef,
		controlSesionRevision:     datosVinculo.ControlSesionRevision,
		controlSesionHuellaSHA256: datosVinculo.ControlSesionHuellaSHA256,
		actorRef:                  datosVinculo.PrincipalID,
		perfilRef:                 datosVinculo.PerfilActivoRef,
		perfilVersion:             resultado.Contexto.Instantanea.PerfilVersion,
		registroContextoRef:       resultado.RegistroContextoRef,
		contextoActorHuella:       resultado.HuellaSHA256,
		organizacionRef:           organizacionRef,
		resueltoEn:                resueltoEn,
		validoHasta:               validoHasta,
	}
	if c.validarEn(instante) != nil {
		return ContextoConsultaRRHH{}, ErrContextoConsultaRRHHInvalido
	}
	return c, nil
}

func inicioVigenciaContextoConsultaRRHH(
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) time.Time {
	inicio := resultado.ResueltoEnAutoritativo
	for _, candidato := range []time.Time{
		vinculo.AutenticacionVerificadaEn,
		vinculo.SesionEmitidaEn,
		vinculo.SesionRevalidadaEn,
		resultado.Contexto.Instantanea.VigenteDesde,
	} {
		if candidato.After(inicio) {
			inicio = candidato
		}
	}
	for _, referencia := range resultado.Contexto.Instantanea.Vinculos {
		if referencia.VigenteDesde.After(inicio) {
			inicio = referencia.VigenteDesde
		}
	}
	return inicio
}

func limiteVigenciaContextoConsultaRRHH(
	vinculo dominiovec.DatosVinculoAutenticacionActorV2,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
) time.Time {
	limite := vinculo.SesionValidaHasta
	if resultado.Contexto.Instantanea.VigenteHasta.Before(limite) {
		limite = resultado.Contexto.Instantanea.VigenteHasta
	}
	for _, referencia := range resultado.Contexto.Instantanea.Vinculos {
		if referencia.VigenteHasta.Before(limite) {
			limite = referencia.VigenteHasta
		}
	}
	return limite
}

func (c ContextoConsultaRRHH) validarEn(instante time.Time) error {
	if !domain.ReferenciaOpacaValida(c.autenticacionRef) ||
		!patronHuellaRRHH.MatchString(c.autenticacionHuella) ||
		!domain.ReferenciaOpacaValida(c.sesionRef) ||
		!domain.ReferenciaOpacaValida(c.controlSesionRef) ||
		c.controlSesionRevision == 0 ||
		!huellaSHA256RRHHValida(c.controlSesionHuellaSHA256) ||
		!domain.ReferenciaOpacaValida(c.actorRef) ||
		!domain.ReferenciaOpacaValida(c.perfilRef) ||
		c.perfilVersion == 0 ||
		c.perfilVersion > versionMaximaJSONSegura ||
		!domain.ReferenciaOpacaValida(c.registroContextoRef) ||
		!patronHuellaRRHH.MatchString(c.contextoActorHuella) ||
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

func huellaSHA256RRHHValida(valor string) bool {
	return patronHuellaRRHH.MatchString(valor) &&
		valor != strings.Repeat("0", 64)
}

func (c ContextoConsultaRRHH) AutenticacionRef() string { return c.autenticacionRef }
func (c ContextoConsultaRRHH) SesionRef() string        { return c.sesionRef }
func (c ContextoConsultaRRHH) ControlSesionRef() string { return c.controlSesionRef }
func (c ContextoConsultaRRHH) ControlSesionRevision() uint64 {
	return c.controlSesionRevision
}
func (c ContextoConsultaRRHH) ControlSesionHuellaSHA256() string {
	return c.controlSesionHuellaSHA256
}
func (c ContextoConsultaRRHH) ActorRef() string        { return c.actorRef }
func (c ContextoConsultaRRHH) PerfilRef() string       { return c.perfilRef }
func (c ContextoConsultaRRHH) PerfilVersion() uint64   { return c.perfilVersion }
func (c ContextoConsultaRRHH) OrganizacionRef() string { return c.organizacionRef }
func (c ContextoConsultaRRHH) ResueltoEn() time.Time   { return c.resueltoEn }
func (c ContextoConsultaRRHH) ValidoHasta() time.Time  { return c.validoHasta }

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
	if len(cursor) != 43 {
		return false
	}
	material, err := base64URLCursorRRHHEstricto.DecodeString(cursor)
	defer clear(material)
	return err == nil &&
		len(material) == 32 &&
		base64.RawURLEncoding.EncodeToString(material) == cursor
}

func (s SolicitudCuadroRRHH) Texto() string                       { return s.texto }
func (s SolicitudCuadroRRHH) EstadoClave() domain.EstadoOperativo { return s.estadoClave }
func (s SolicitudCuadroRRHH) FaseClave() domain.ClaveFase         { return s.faseClave }
func (s SolicitudCuadroRRHH) Limite() uint16                      { return s.limite }
func (s SolicitudCuadroRRHH) Cursor() string                      { return s.cursor }
func (s SolicitudCuadroRRHH) HuellaCanonicaSHA256() (string, error) {
	return huellaSolicitudCuadroRRHH(s)
}
func (s SolicitudCuadroRRHH) FiltrosHuellaSHA256() (string, error) {
	return huellaFiltrosCuadroRRHH(s)
}
func (SolicitudCuadroRRHH) String() string   { return "[solicitud-cuadro-rrhh-redactada]" }
func (SolicitudCuadroRRHH) GoString() string { return "[solicitud-cuadro-rrhh-redactada]" }
func (s SolicitudCuadroRRHH) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, s.String())
}
func (s SolicitudCuadroRRHH) LogValue() slog.Value {
	return slog.StringValue(s.String())
}
func (SolicitudCuadroRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}

func (p PaginaCuadroRRHH) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, p.String())
}
func (p PaginaCuadroRRHH) LogValue() slog.Value {
	return slog.StringValue(p.String())
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
func (s SolicitudDetalleRRHH) HuellaCanonicaSHA256() (string, error) {
	return huellaSolicitudDetalleRRHH(s)
}
func (SolicitudDetalleRRHH) String() string   { return "[solicitud-detalle-rrhh-redactada]" }
func (SolicitudDetalleRRHH) GoString() string { return "[solicitud-detalle-rrhh-redactada]" }
func (SolicitudDetalleRRHH) MarshalJSON() ([]byte, error) {
	return nil, ErrMaterialConsultaRRHHSensible
}
