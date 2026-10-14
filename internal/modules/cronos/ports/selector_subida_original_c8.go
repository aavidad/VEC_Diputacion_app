package ports

import (
	"context"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// SelectorSubidaOriginalC8 procede de la ruta Cronos. Se cierra al construir
// el subidor de esta petición y nunca se sustituye por la clave idempotente.
type SelectorSubidaOriginalC8 struct {
	SolicitudRef    string
	VersionEsperada int64
}

type EstadoPropietarioSubidaOriginalC8 struct {
	SolicitudRef string                     `json:"solicitud_ref"`
	Version      int64                      `json:"version"`
	PersonaRef   string                     `json:"persona_ref"`
	EmpleadoRef  string                     `json:"empleado_ref"`
	Auditoria    AuditoriaLecturaOriginalC8 `json:"auditoria"`
}

// La captura procede de sesión certificada y permiso actual. El SQL usa el
// LOGIN documental, exige CRN22 y registra la lectura en la misma transacción.
type CapturadorPreparacionSubidaOriginalC8 interface {
	CapturarPreparacionSubidaOriginal(context.Context, vecdomain.ResultadoContextoActorRegistradoV2,
		vecdomain.VinculoAutenticacionActorV2, SelectorSubidaOriginalC8) (CapturaLecturaOriginalC8, error)
}

type FuenteActoSubidaOriginalC8 interface {
	ConsultarActoSubidaOriginal(context.Context, SelectorSubidaOriginalC8,
		CapturaLecturaOriginalC8) (EstadoPropietarioSubidaOriginalC8, error)
}

type ActoSubidaOriginalC8 struct {
	Recurso       vecdomain.RecursoAutorizable
	PersonaRef    string
	Finalidad     string
	Clasificacion string
	VersionDueno  uint64
}
