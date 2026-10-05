package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var ErrLecturaUsuariosAdministrablesNoDisponible = errors.New("vec.usuarios_administrables.lectura.no_disponible")

// Los filtros son selección de la petición; organización y unidad proceden
// exclusivamente de la configuración privada del adaptador.
type FiltrosUsuariosAdministrables struct {
	PerfilRef string
	UnidadRef string
	Estado    string
	Cursor    string
}

type PerfilUsuarioAdministrable struct {
	PerfilRef     string
	RolVersionRef string
	Version       uint64
	Estado        string
	VigenteDesde  time.Time
	VigenteHasta  time.Time
}

// DenominacionVersion es nil sólo si CA comprobó que no hay denominación.
// Esta fuente no lee ni proyecta nombre, cuenta, acto o historia.
type UsuarioAdministrable struct {
	PersonaRef          string
	UnidadRef           string
	DenominacionVersion *uint64
	Perfiles            []PerfilUsuarioAdministrable
}

type PaginaUsuariosAdministrables struct {
	Personas        []UsuarioAdministrable
	SiguienteCursor string
}

type FuenteUsuariosAdministrables interface {
	ListarUsuarios(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, FiltrosUsuariosAdministrables) (PaginaUsuariosAdministrables, error)
	ConsultarUsuario(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (*UsuarioAdministrable, error)
}

// EmisionUsuariosAdministrables es el contrato de entrada del emisor central.
// Material y recurso se construyen juntos; la correlación procede del canal
// interno y no de un campo libre del cliente.
type EmisionUsuariosAdministrables struct {
	Material    []byte
	Recurso     domain.RecursoAutorizable
	Accion      string
	Audiencia   string
	Correlacion domain.ReferenciaCorrelacionAutorizacionV2
}

// La implementación real pertenece a la composición de autorización V3.
// La fuente PostgreSQL no crea decisiones, firmas, raíces ni perfiles.
type EmisorLecturaUsuariosAdministrables interface {
	EmitirLecturaUsuariosAdministrables(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, domain.InstantaneaAutorizacion, EmisionUsuariosAdministrables) (ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
