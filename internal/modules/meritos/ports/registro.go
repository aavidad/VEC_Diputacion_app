package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	vec "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrRegistroNoDisponible = errors.New("meritos.error.registro_no_disponible")
	ErrConflictoVersion     = errors.New("meritos.error.conflicto_version")
	ErrClaveReutilizada     = errors.New("meritos.error.clave_reutilizada")
)

// OrdenOperacion liga clave, actor, contenido y versión esperada a la solicitud
// V3 exacta. El contenido inicial no es una revisión ni acreditación efectiva.
type OrdenOperacion struct {
	Accion            string
	ActorRef          string
	ClaveIdempotencia string
	HuellaComando     string
	VersionEsperada   int
	Hecho             domain.Hecho
	Motivo            vec.ReferenciaEntradaCatalogo
	FechaCorte        string
	Autorizacion      AutorizacionOperacion
}

type RegistroActual struct {
	Hecho         domain.Hecho
	DeclaranteRef string
}

type PreparacionOperacion struct {
	Actual         *RegistroActual
	ReciboAnterior *Recibo
}

type Cambio struct {
	Orden     OrdenOperacion
	Anterior  *RegistroActual
	Nuevo     RegistroActual
	Auditoria vec.AuditEntry
}

type Recibo struct {
	Referencia        string
	Accion            string
	ActorRef          string
	ClaveIdempotencia string
	HuellaComando     string
	VersionEsperada   int
	Registro          RegistroActual
	RegistradoEn      time.Time
	AuditoriaRef      string
	EventoRef         string
}

// Registro no tiene implementación runtime en RUM02.
// PrepararOperacion lee únicamente bajo concesión positiva exacta y vigente;
// no consume la concesión ni confirma un efecto. La idempotencia se comprueba
// antes del CAS: una clave con otra huella devuelve ErrClaveReutilizada.
// ConfirmarCambio revalida/consume V3 bajo bloqueo y CAS del estado anterior,
// mantiene declarante y origen estables, impide duplicar persona/fuente/origen/
// tipo y añade versión, historia, auditoría, outbox y recibo en una transacción.
// RecuperarCambio exige nueva autoridad vigente, audita el acceso en transacción
// y devuelve el recibo original sin otra versión ni otro evento de negocio.
// Ningún método devuelve recibo ante rollback, resultado incierto o commit fallido.
type Registro interface {
	PrepararOperacion(context.Context, OrdenOperacion) (PreparacionOperacion, error)
	ConfirmarCambio(context.Context, Cambio) (Recibo, error)
	RecuperarCambio(context.Context, OrdenOperacion) (Recibo, error)
}
