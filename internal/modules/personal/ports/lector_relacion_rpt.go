package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Se reutiliza el puerto nominal RPT existente. Estas interfaces sólo conectan
// su autorización y persistencia; no crean otro dato laboral o permiso B2.
type ProveedorAutorizacionLectorRelacionRPT interface {
	AutorizarRelacionParaRPT(context.Context, domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
type OrdenLectorRelacionRPT struct {
	Material     domain.MaterialLectorRelacionRPT
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// RepositorioLectorRelacionRPT selecciona la última revisión conocida primero,
// coteja luego la terna/versión y consume autorización RPT actual junto con
// lectura, auditoría y recibo. Valida la respuesta antes de confirmar.
// La fuente B2 conserva certeza no_acreditado y cobertura no_acreditada.
type RepositorioLectorRelacionRPT interface {
	ConsultarRelacionParaRPT(context.Context, OrdenLectorRelacionRPT) (ResultadoRelacionParaRPTV1, error)
}

// IntentoLectorRelacionRPT no confía en Actor para escribir actor_ref: el
// registrador de composición conserva la identidad original de la petición.
// El motivo es cerrado; no contiene mensajes SQL ni datos laborales.
type IntentoLectorRelacionRPT struct {
	Actor       vecdomain.ContextoActor
	RelacionRef string
	Motivo      string
}
type RegistroIntentosLectorRelacionRPT interface {
	VerificarRegistroRelacionRPT(context.Context) error
	RegistrarIntentoRelacionRPT(context.Context, IntentoLectorRelacionRPT) error
}
