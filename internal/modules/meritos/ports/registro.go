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
	Hecho         domain.Hecho `json:"hecho"`
	DeclaranteRef string       `json:"declarante_ref"`
}

type ResultadoOperacion struct {
	Codigo       string          `json:"codigo"`
	AuditoriaRef string          `json:"auditoria_ref"`
	Anterior     *RegistroActual `json:"anterior"`
	Recibo       *Recibo         `json:"recibo"`
}

type Cambio struct {
	Orden     OrdenOperacion
	Anterior  *RegistroActual
	Nuevo     RegistroActual
	Auditoria vec.AuditEntry
}

type Recibo struct {
	Referencia        string         `json:"referencia"`
	Accion            string         `json:"accion"`
	ActorRef          string         `json:"actor_ref"`
	ClaveIdempotencia string         `json:"clave_idempotencia"`
	HuellaComando     string         `json:"huella_comando"`
	VersionEsperada   int            `json:"version_esperada"`
	Registro          RegistroActual `json:"registro"`
	RegistradoEn      time.Time      `json:"registrado_en"`
	AuditoriaRef      string         `json:"auditoria_ref"`
	EventoRef         string         `json:"evento_ref"`
}

// Registro consume/revalida V3, carga el antecedente confiable y aplica la
// operación bajo CAS, sin lectura previa fuera de esa transacción. Idempotencia
// por actor+clave se comprueba antes del CAS; persona/declarante/origen son
// estables y el hecho original no se duplica. Confirma versión, historia,
// auditoría, outbox y recibo juntos. Un reintento consume autoridad nueva,
// audita el acceso y devuelve el recibo y antecedente histórico originales.
// Un rechazo de negocio confirmado devuelve únicamente código y auditoría:
// no hay datos de negocio ni recibo. El consumidor valida la respuesta antes
// de COMMIT; ante rollback o incertidumbre devuelve error y resultado vacío.
type Registro interface {
	EjecutarOperacion(context.Context, OrdenOperacion) (ResultadoOperacion, error)
}
