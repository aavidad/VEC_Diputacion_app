package ports

import (
	"context"
	"errors"
	"slices"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Cancelación del expediente antes de la fiscalización (CT122, AD3-87). Usa
// la misma mecánica que las operaciones de seguimiento: sellos HMAC de
// idempotencia y de petición, preparación de solo lectura y confirmación que
// consume la autorización atestada en la transacción del efecto.
const (
	OperacionCancelarExpediente   = "cancelar_expediente"
	TipoRecursoCancelacion        = "cancelacion_contratacion_temporal"
	FinalidadCancelarExpediente   = "cancelar_expediente_contratacion_temporal"
	AudienciaConsumoCancelacionV1 = "vec_contratacion_temporal.cancelacion_expediente.v1"

	DominioAmbitoIdempotenciaCancelacion = "vec.contratacion-temporal.cancelacion.ambito"
	DominioHuellaPeticionCancelacion     = "vec.contratacion-temporal.cancelacion.peticion"
)

var (
	// ErrCancelacionNoAdmitida: el expediente no está en curso, su fase no la
	// admite la regla vigente o ya se fiscalizó. No escribe nada.
	ErrCancelacionNoAdmitida = errors.New("contratacion temporal: cancelacion no admitida en el estado del expediente")
	// ErrCancelacionTrasFiscalizacion: el expediente ya pasó por fiscalización.
	ErrCancelacionTrasFiscalizacion = errors.New("contratacion temporal: cancelacion tras la fiscalizacion")
	ErrCancelacionYaRegistrada      = errors.New("contratacion temporal: cancelacion ya registrada")
)

// MotivoCancelacion es una entrada vigente del catálogo de motivos, con los
// canales (centro, RRHH) que pueden usarla.
type MotivoCancelacion struct {
	Clave     domain.ClaveCatalogo
	Etiqueta  string
	ClaveI18n string
	Canales   []domain.CanalCancelacion
}

func (m MotivoCancelacion) AdmiteCanal(canal domain.CanalCancelacion) bool {
	return slices.Contains(m.Canales, canal)
}

// ReglaCancelacion son las fases desde las que se admite la cancelación
// (regla c12) y los motivos vigentes del catálogo que la regla admite.
type ReglaCancelacion struct {
	Fases   []domain.ClaveFase
	Motivos []MotivoCancelacion
}

func (r ReglaCancelacion) Valida() bool {
	if !domain.FasesCancelacionValidas(r.Fases) || len(r.Motivos) == 0 || len(r.Motivos) > maximoOpcionesSeguimiento {
		return false
	}
	for _, m := range r.Motivos {
		if !m.Clave.Valida() || m.Etiqueta == "" || len(m.Canales) == 0 ||
			slices.ContainsFunc(m.Canales, func(c domain.CanalCancelacion) bool { return !c.Valido() }) {
			return false
		}
	}
	return true
}

// FuenteReglasCancelacion resuelve desde catálogos versionados las fases y
// los motivos, y la política (catálogo de motivos) que ampara la operación.
type FuenteReglasCancelacion interface {
	ReglaCancelacion(ctx context.Context, instante time.Time) (ReglaCancelacion, PoliticaOperacionSeguimiento, error)
}

// MaterialCancelacion es la intención exacta que se sella y persiste.
type MaterialCancelacion struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef string
	VersionEsperada                                     uint64
	ClaveIdempotencia                                   string
	Datos                                               domain.DatosCancelacion
}

func (m MaterialCancelacion) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		m.Datos.Validar() == nil
}

// EstadoCancelacionExpediente resume la cancelación registrada, si existe.
type EstadoCancelacionExpediente struct {
	ExpedienteRef string
	Cancelacion   *CancelacionRegistrada
}

type CancelacionRegistrada struct {
	Canal         domain.CanalCancelacion
	MotivoClave   string
	FasePrevia    string
	Observaciones string
	ReciboRef     string
	RegistradaEn  time.Time
}

type LectorEstadoCancelacion interface {
	ConsultarEstadoCancelacion(ctx context.Context, organizacionRef, expedienteRef string) (EstadoCancelacionExpediente, error)
}
