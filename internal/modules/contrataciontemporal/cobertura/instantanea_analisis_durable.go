package cobertura

import (
	"context"
	"errors"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrSolicitudInstantaneaAnalisisDurableInvalida = errors.New(
		"contratacion temporal: solicitud de instantanea de analisis durable invalida",
	)
	ErrInstantaneaAnalisisDurableNoDisponible = errors.New(
		"contratacion temporal: instantanea de analisis durable no disponible",
	)
	ErrInstantaneaAnalisisDurableNoConfiable = errors.New(
		"contratacion temporal: instantanea de analisis durable no confiable",
	)
)

// SolicitudInstantaneaAnalisisDurableO3 identifica únicamente el agregado
// que debe leer la infraestructura. No contiene análisis, recibos ni huellas
// aportables por un cliente.
type SolicitudInstantaneaAnalisisDurableO3 struct {
	bloqueoSerializacionOperacionDecisionCobertura
	organizacionRef string
	expedienteRef   string
	versionEsperada uint64
}

func NuevaSolicitudInstantaneaAnalisisDurableO3(
	organizacionRef string,
	expedienteRef string,
	versionEsperada uint64,
) (SolicitudInstantaneaAnalisisDurableO3, error) {
	solicitud := SolicitudInstantaneaAnalisisDurableO3{
		organizacionRef: organizacionRef,
		expedienteRef:   expedienteRef,
		versionEsperada: versionEsperada,
	}
	if solicitud.validar() != nil {
		return SolicitudInstantaneaAnalisisDurableO3{},
			ErrSolicitudInstantaneaAnalisisDurableInvalida
	}
	return solicitud, nil
}

func (s SolicitudInstantaneaAnalisisDurableO3) validar() error {
	if !domain.ReferenciaOpacaValida(s.organizacionRef) ||
		!domain.ReferenciaOpacaValida(s.expedienteRef) ||
		s.versionEsperada < 2 ||
		s.versionEsperada >= MaximoEnteroSeguroOperacionDecisionCobertura {
		return ErrSolicitudInstantaneaAnalisisDurableInvalida
	}
	return nil
}

// Coordenadas entrega al adaptador de salida la clave mínima de lectura. El
// método no concede autoridad ni expone el contenido de la instantánea.
func (s SolicitudInstantaneaAnalisisDurableO3) Coordenadas() (
	organizacionRef string,
	expedienteRef string,
	versionEsperada uint64,
	err error,
) {
	if s.validar() != nil {
		return "", "", 0, ErrSolicitudInstantaneaAnalisisDurableInvalida
	}
	return s.organizacionRef, s.expedienteRef, s.versionEsperada, nil
}

// LectorExpedienteAnalisisDurableO3 es un puerto funcional de salida. La
// composición del lado servidor debe inyectar solo el adaptador durable;
// ningún adaptador HTTP, CLI, MCP o de escritorio debe implementarlo.
type LectorExpedienteAnalisisDurableO3 interface {
	LeerExpedienteAnalisisDurableO3(
		context.Context,
		SolicitudInstantaneaAnalisisDurableO3,
	) (domain.Expediente, error)
}

// InstantaneaAnalisisDurableO3 acredita que el agregado leído coincide con la
// identidad solicitada y conserva un análisis O3 íntegro que habilita avance.
// No tiene constructor público desde datos: solo Obtener... puede materializar
// la capacidad tras invocar y verificar el lector server-side.
type InstantaneaAnalisisDurableO3 struct {
	bloqueoSerializacionOperacionDecisionCobertura
	expediente           *domain.Expediente
	analisisRef          string
	analisisHuellaSHA256 string
}

// ObtenerInstantaneaAnalisisDurableO3 lee primero la autoridad durable y
// deriva dentro de esta frontera la referencia y la huella O3. Los errores
// privados del adaptador no atraviesan la frontera.
func ObtenerInstantaneaAnalisisDurableO3(
	ctx context.Context,
	lector LectorExpedienteAnalisisDurableO3,
	solicitud SolicitudInstantaneaAnalisisDurableO3,
) (InstantaneaAnalisisDurableO3, error) {
	if dependenciaInstantaneaAnalisisDurableNula(ctx) ||
		dependenciaInstantaneaAnalisisDurableNula(lector) ||
		solicitud.validar() != nil {
		return InstantaneaAnalisisDurableO3{},
			ErrSolicitudInstantaneaAnalisisDurableInvalida
	}
	if err := ctx.Err(); err != nil {
		return InstantaneaAnalisisDurableO3{},
			errors.Join(ErrInstantaneaAnalisisDurableNoDisponible, err)
	}
	expediente, err := lector.LeerExpedienteAnalisisDurableO3(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return InstantaneaAnalisisDurableO3{},
			errors.Join(
				ErrInstantaneaAnalisisDurableNoDisponible,
				errContexto,
			)
	}
	if err != nil {
		return InstantaneaAnalisisDurableO3{},
			ErrInstantaneaAnalisisDurableNoDisponible
	}
	instantanea, err := nuevaInstantaneaAnalisisDurableO3(
		expediente,
		solicitud,
	)
	if err != nil {
		return InstantaneaAnalisisDurableO3{},
			ErrInstantaneaAnalisisDurableNoConfiable
	}
	return instantanea, nil
}

func nuevaInstantaneaAnalisisDurableO3(
	expediente domain.Expediente,
	solicitud SolicitudInstantaneaAnalisisDurableO3,
) (InstantaneaAnalisisDurableO3, error) {
	analisisRef, analisisHuella, err :=
		identidadAnalisisDurableO3(expediente, solicitud)
	if err != nil {
		return InstantaneaAnalisisDurableO3{},
			ErrInstantaneaAnalisisDurableNoConfiable
	}
	clon := expediente.Clonar()
	return InstantaneaAnalisisDurableO3{
		expediente:           &clon,
		analisisRef:          analisisRef,
		analisisHuellaSHA256: analisisHuella,
	}, nil
}

func identidadAnalisisDurableO3(
	expediente domain.Expediente,
	solicitud SolicitudInstantaneaAnalisisDurableO3,
) (string, string, error) {
	organizacionRef, expedienteRef, versionEsperada, err :=
		solicitud.Coordenadas()
	if err != nil ||
		expediente.Validar() != nil ||
		expediente.OrganizacionRef != organizacionRef ||
		expediente.Referencia != expedienteRef ||
		expediente.Version != versionEsperada ||
		expediente.Analisis == nil ||
		!expediente.Analisis.HabilitaAvance() ||
		expediente.Analisis.ActuacionRegistro == nil {
		return "", "", ErrInstantaneaAnalisisDurableNoConfiable
	}
	vinculo := expediente.Analisis.ActuacionRegistro
	if vinculo.AccionClave != domain.ClaveCatalogo(ports.AccionRegistrarAnalisis) &&
		vinculo.AccionClave != domain.ClaveCatalogo(ports.AccionRectificarAnalisis) {
		return "", "", ErrInstantaneaAnalisisDurableNoConfiable
	}
	huella, err := ports.HuellaAnalisisRRHHRehidratadoO3(
		*expediente.Analisis,
	)
	if err != nil || !domain.ReferenciaOpacaValida(vinculo.ReciboRef) ||
		!huellaSHA256OperacionDecisionCoberturaValida(huella) {
		return "", "", ErrInstantaneaAnalisisDurableNoConfiable
	}
	return vinculo.ReciboRef, huella, nil
}

// DesplegarPara entrega una copia defensiva solo al consumidor que conserva
// la solicitud exacta. No crea un DTO transferible ni expone memoria interna.
func (i InstantaneaAnalisisDurableO3) DesplegarPara(
	solicitud SolicitudInstantaneaAnalisisDurableO3,
) (
	expediente domain.Expediente,
	analisisRef string,
	analisisHuellaSHA256 string,
	err error,
) {
	if i.expediente == nil {
		return domain.Expediente{}, "", "",
			ErrInstantaneaAnalisisDurableNoConfiable
	}
	refCalculada, huellaCalculada, err := identidadAnalisisDurableO3(
		*i.expediente,
		solicitud,
	)
	if err != nil ||
		!referenciasOperacionDecisionCoberturaIguales(
			i.analisisRef,
			refCalculada,
		) ||
		!referenciasOperacionDecisionCoberturaIguales(
			i.analisisHuellaSHA256,
			huellaCalculada,
		) {
		return domain.Expediente{}, "", "",
			ErrInstantaneaAnalisisDurableNoConfiable
	}
	return i.expediente.Clonar(), i.analisisRef, i.analisisHuellaSHA256, nil
}

func dependenciaInstantaneaAnalisisDurableNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
