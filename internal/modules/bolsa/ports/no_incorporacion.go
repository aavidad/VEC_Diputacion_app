package ports

import (
	"context"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
)

// PlazosNoIncorporacion son las fechas que el calendario del catálogo de
// Bolsa calcula para la clave de consecuencia del evento, con la regla del
// recurso con que se calcularon. La consecuencia (baja, suspensión o
// ninguna) no viaja: la resuelve la base con la política publicada al
// arrancar (Bolsa 000042), que además comprueba esa regla. Nil si el catálogo
// no reconoce la clave: el evento queda sin efecto para revisión.
type PlazosNoIncorporacion struct {
	RecursoVence             string  `json:"recurso_vence"`
	RecursoReglaRef          string  `json:"recurso_regla_ref"`
	RecursoReglaHuellaSHA256 string  `json:"recurso_regla_huella_sha256"`
	SuspensionHasta          *string `json:"suspension_hasta"`
}

// EventoNoIncorporacionRecibido conserva el contenido exacto y su huella.
type EventoNoIncorporacionRecibido struct {
	Evento         dominiobolsa.EventoNoIncorporacion
	Contenido      []byte
	HuellaSHA256   string
	OrigenCreadaEn time.Time
	OrigenPosicion int64
	Plazos         *PlazosNoIncorporacion
}

// Resultados de la evaluación en Bolsa. Solo «aplicada» tiene efecto y
// habilita el siguiente llamamiento; «llamamiento_ajeno» es definitiva; las
// demás quedan para revisión de RRHH y se reevalúan en cada pasada.
const (
	EstadoNoIncorporacionAplicada         = "aplicada"
	EstadoNoIncorporacionLlamamientoAjeno = "llamamiento_ajeno"
)

// ResultadoRegistroNoIncorporacion: Estado dice qué efecto tuvo («aplicada»,
// «participacion_no_constituida», «sin_aceptacion»...).
type ResultadoRegistroNoIncorporacion struct {
	Reutilizado      bool
	Estado           string
	ParticipacionRef string
}

// PendienteNoIncorporacion es un evento de la bandeja aún sin aplicar, con lo
// necesario para recalcular sus plazos.
type PendienteNoIncorporacion struct {
	EventoRef         string
	ConsecuenciaClave string
	FechaNotificacion string
}

// LimitePendientesNoIncorporacion acota cada reevaluación.
const LimitePendientesNoIncorporacion = 100

// BuzonNoIncorporaciones es la bandeja idempotente de Bolsa 000042, que solo
// alcanza el rol propio del relevo.
type BuzonNoIncorporaciones interface {
	CursorNoIncorporaciones(context.Context) (CursorContratosParticipacion, bool, error)
	RegistrarNoIncorporacion(context.Context, EventoNoIncorporacionRecibido) (ResultadoRegistroNoIncorporacion, error)
	PendientesNoIncorporacion(ctx context.Context, limite int) ([]PendienteNoIncorporacion, error)
	ReevaluarNoIncorporacion(ctx context.Context, eventoRef string, plazos *PlazosNoIncorporacion) (ResultadoRegistroNoIncorporacion, error)
}

// ResolvedorSancionNoIncorporacion resuelve la consecuencia con el catálogo
// de sanciones de Bolsa (b24.sancion.*).
type ResolvedorSancionNoIncorporacion interface {
	ResolverSancion(ctx context.Context, clave string, notificadaEn time.Time) (ResolucionCatalogoSancion, error)
}

// ConsecuenciaPoliticaNoIncorporacion es una entrada b24.sancion.* tal como
// se publica en la política de la base.
type ConsecuenciaPoliticaNoIncorporacion struct {
	Etiqueta          string `json:"etiqueta"`
	Efecto            string `json:"efecto"`
	ReglaRef          string `json:"regla_ref"`
	ReglaHuellaSHA256 string `json:"regla_huella_sha256"`
	ConPlazo          bool   `json:"con_plazo"`
	OrdenFinal        bool   `json:"orden_final"`
	FinAutomatico     bool   `json:"fin_automatico"`
}

// PoliticaNoIncorporacion traslada a la base (Bolsa 000042) las
// consecuencias del catálogo y la regla del recurso, al arrancar.
type PoliticaNoIncorporacion struct {
	CatalogoRef              string
	CatalogoSHA256           string
	Consecuencias            map[string]ConsecuenciaPoliticaNoIncorporacion
	RecursoReglaRef          string
	RecursoReglaHuellaSHA256 string
}

// PublicadorPoliticaNoIncorporacion registra una versión nueva si difiere
// de la vigente y devuelve la versión vigente.
type PublicadorPoliticaNoIncorporacion interface {
	PublicarPoliticaNoIncorporacion(context.Context, PoliticaNoIncorporacion) (int64, error)
}
