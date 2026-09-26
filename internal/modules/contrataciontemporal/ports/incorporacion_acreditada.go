package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Confirmación de GINPIX (CT124, AD3-88): RRHH registra el número de alta de
// la ficha de la incorporación acreditada. Recurso, finalidad y audiencia
// propios; el cierre toma de aquí su número y su fecha.
const (
	OperacionConfirmarGINPIX                    = "confirmar_ginpix"
	TipoRecursoConfirmacionGINPIX               = "confirmacion_ginpix_contratacion_temporal"
	FinalidadConfirmarGINPIX                    = "confirmar_ginpix_contratacion_temporal"
	AudienciaConsumoConfirmacionGINPIXV1        = "vec_contratacion_temporal.confirmacion_ginpix.v1"
	DominioAmbitoIdempotenciaConfirmacionGINPIX = "vec.contratacion-temporal.confirmacion-ginpix.ambito"
	DominioHuellaPeticionConfirmacionGINPIX     = "vec.contratacion-temporal.confirmacion-ginpix.peticion"
)

var (
	// ErrGINPIXNoConfirmado: la regla del cierre exige GINPIX y no consta su
	// confirmación para la incorporación vigente. No se escribe nada.
	ErrGINPIXNoConfirmado = errors.New("contratacion temporal: ficha de GINPIX sin confirmar")
	// ErrGINPIXYaConfirmado: la incorporación vigente ya tiene confirmación.
	ErrGINPIXYaConfirmado = errors.New("contratacion temporal: ficha de GINPIX ya confirmada")
	// ErrGINPIXDistinto: el número o la fecha enviados no son los de la
	// confirmación registrada.
	ErrGINPIXDistinto = errors.New("contratacion temporal: numero de GINPIX distinto del confirmado")
)

// MaterialConfirmacionGINPIX es la intención exacta que se sella y persiste.
type MaterialConfirmacionGINPIX struct {
	OrganizacionRef, ExpedienteRef, ActorRef, PerfilRef string
	VersionEsperada                                     uint64
	ClaveIdempotencia                                   string
	Datos                                               domain.DatosConfirmacionGINPIX
}

func (m MaterialConfirmacionGINPIX) Valido() bool {
	return identidadOperacionValida(m.OrganizacionRef, m.ExpedienteRef, m.ActorRef, m.PerfilRef, m.VersionEsperada, m.ClaveIdempotencia) &&
		m.Datos.Validar() == nil
}

// FuenteReglaConfirmacionGINPIX resuelve la política que ampara la
// confirmación: la regla c10 del catálogo, con el motivo de su ruta.
type FuenteReglaConfirmacionGINPIX interface {
	PoliticaConfirmacionGINPIX(ctx context.Context, instante time.Time) (PoliticaOperacionSeguimiento, error)
}

// EstadoGINPIXConfirmado es la confirmación de la ficha vigente.
type EstadoGINPIXConfirmado struct {
	Numero       string
	ConfirmadaEn string
	ReciboRef    string
	RegistradaEn time.Time
}

// EstadoConfirmacionCentro es la confirmación de la incorporación por el
// centro: fecha y documento por tipo, referencia y huella.
type EstadoConfirmacionCentro struct {
	FechaIncorporacion string
	DocumentoTipo      string
	DocumentoRef       string
	DocumentoSHA256    string
	ReciboRef          string
	RegistradaEn       time.Time
}

// EstadoIncorporacionAcreditada reúne ambas confirmaciones y la no
// incorporación, si constan.
type EstadoIncorporacionAcreditada struct {
	GINPIX          *EstadoGINPIXConfirmado
	Centro          *EstadoConfirmacionCentro
	NoIncorporacion *EstadoNoIncorporacion
	// PropuestaNoIncorporacion es la propuesta pendiente de la segunda
	// persona (cuatro ojos), si la hay.
	PropuestaNoIncorporacion *EstadoPropuestaNoIncorporacion
	// Propuestas: la vigente y las sustituidas por una no incorporación (CT128).
	Propuestas []EstadoPropuestaExpediente
}

// Valido acota formato y tamaño antes de publicar la lectura.
func (e EstadoIncorporacionAcreditada) Valido() bool {
	if g := e.GINPIX; g != nil {
		if !domain.NumeroGINPIXValido(g.Numero) || !fechaCivilTextoValida(g.ConfirmadaEn) ||
			!domain.ReferenciaOpacaValida(g.ReciboRef) || !domain.InstanteUTCCanonico(g.RegistradaEn) {
			return false
		}
	}
	if c := e.Centro; c != nil {
		if !fechaCivilTextoValida(c.FechaIncorporacion) || !domain.ClaveCatalogo(c.DocumentoTipo).Valida() ||
			!domain.ReferenciaOpacaValida(c.DocumentoRef) || !huellaSHA256OperacionAnalisisValida(c.DocumentoSHA256) ||
			!domain.ReferenciaOpacaValida(c.ReciboRef) || !domain.InstanteUTCCanonico(c.RegistradaEn) {
			return false
		}
	}
	return (e.NoIncorporacion == nil || e.NoIncorporacion.Valido()) &&
		(e.PropuestaNoIncorporacion == nil || e.PropuestaNoIncorporacion.Valido()) && PropuestasExpedienteValidas(e.Propuestas)
}

func fechaCivilTextoValida(valor string) bool {
	f, err := time.Parse(time.DateOnly, valor)
	return err == nil && f.Format(time.DateOnly) == valor
}

// LectorIncorporacionAcreditada lee ambas confirmaciones del expediente. La
// composición solo la invoca tras acreditar la lectura del detalle.
type LectorIncorporacionAcreditada interface {
	ConsultarIncorporacionAcreditada(ctx context.Context, organizacionRef, expedienteRef string) (EstadoIncorporacionAcreditada, error)
}
