package ports

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrCatalogoNoEncontrado        = errors.New("vec: catalogo configurable no encontrado")
	ErrVersionCatalogoYaExiste     = errors.New("vec: version de catalogo ya existe")
	ErrRevisionCatalogoEnConflicto = errors.New("vec: revision de catalogo en conflicto")
	ErrSecuenciaCatalogoInvalida   = errors.New("vec: secuencia de catalogo invalida")
)

type ConsultaCatalogosConfigurables interface {
	ObtenerCatalogo(context.Context, string, int) (domain.CatalogoConfigurable, error)
	ListarVersionesCatalogo(context.Context, string) ([]domain.CatalogoConfigurable, error)
}

// RepositorioGobiernoCatalogos confirma cada cambio junto con su auditoria y
// outbox. Una version publicada o retirada nunca se sobrescribe.
type RepositorioGobiernoCatalogos interface {
	ConfirmarAltaBorradorCatalogo(context.Context, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event) error
	ConfirmarActualizacionBorradorCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event) error
	ConfirmarPublicacionCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event) error
	ConfirmarRetiradaCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event) error
}
