// Package registrocopias defines the offline executor journal contract.
// Declaracion identifies a declared CLI actor; it is never an authorization.
package registrocopias

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
)

var (
	ErrConfiguracion  = errors.New("registro_configuracion_invalida")
	ErrEntrada        = errors.New("registro_entrada_invalida")
	ErrCorrupto       = errors.New("registro_historia_incompleta")
	ErrNoExiste       = errors.New("registro_no_existe")
	ErrIO             = errors.New("registro_no_disponible")
	ErrDestinoOcupado = errors.New("registro_destino_ocupado")
)

type Declaracion struct {
	Actor       string `json:"actor_declarado"`
	Correlacion string `json:"correlacion"`
}

type Auditoria struct {
	Secuencia   uint64      `json:"secuencia"`
	Referencia  string      `json:"referencia"`
	Declaracion Declaracion `json:"declaracion"`
	Accion      string      `json:"accion"`
	Operacion   string      `json:"operacion"`
	Resultado   string      `json:"resultado"`
	Instante    string      `json:"instante"`
	Autoridad   string      `json:"autoridad"`
}

type Recibo struct {
	Referencia string                   `json:"referencia"`
	Instante   string                   `json:"instante"`
	Version    uint64                   `json:"version"`
	Estado     operacionescopias.Estado `json:"estado"`
}

type Resultado struct {
	Solicitud      operacionescopias.Solicitud `json:"solicitud"`
	Historia       []operacionescopias.Evento  `json:"historia"`
	Recibo         Recibo                      `json:"recibo"`
	Replay         bool                        `json:"replay"`
	Reconciliacion string                      `json:"reconciliacion"`
	Auditoria      Auditoria                   `json:"auditoria"`
	Operaciones    []Vista                     `json:"operaciones,omitempty"`
	Siguiente      string                      `json:"siguiente,omitempty"`
}

type Consulta struct {
	Limite  int    `json:"limite"`
	Despues string `json:"despues,omitempty"`
}
type Vista struct {
	Solicitud      operacionescopias.Solicitud `json:"solicitud"`
	Recibo         Recibo                      `json:"recibo"`
	Reconciliacion string                      `json:"reconciliacion"`
}

// Registro is for an offline declared-progress executor only. A future API must
// consume the existing central authority atomically before calling a production
// adapter. This interface does not issue or accept invented permission tokens.
type Registro interface {
	Reservar(context.Context, Declaracion, operacionescopias.Solicitud) (Resultado, error)
	Aplicar(context.Context, Declaracion, string, operacionescopias.Comando) (Resultado, error)
	Consultar(context.Context, Declaracion, string) (Resultado, error)
	Listar(context.Context, Declaracion, Consulta) (Resultado, error)
}
