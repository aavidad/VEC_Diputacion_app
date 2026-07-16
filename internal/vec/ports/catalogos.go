package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrCatalogoNoEncontrado        = errors.New("vec: catalogo configurable no encontrado")
	ErrVersionCatalogoYaExiste     = errors.New("vec: version de catalogo ya existe")
	ErrRevisionCatalogoEnConflicto = errors.New("vec: revision de catalogo en conflicto")
	ErrSecuenciaCatalogoInvalida   = errors.New("vec: secuencia de catalogo invalida")
)

const (
	AccionCrearCatalogoConfigurable      = "vec.catalogos.crear"
	AccionActualizarCatalogoConfigurable = "vec.catalogos.actualizar"
	AccionPublicarCatalogoConfigurable   = "vec.catalogos.publicar"
	AccionRetirarCatalogoConfigurable    = "vec.catalogos.retirar"
)

type ConsultaCatalogosConfigurables interface {
	ObtenerCatalogo(context.Context, string, int) (domain.CatalogoConfigurable, error)
	ListarVersionesCatalogo(context.Context, string) ([]domain.CatalogoConfigurable, error)
}

// MetadatosFuenteCatalogos describe el adaptador de lectura sin revelar la
// procedencia interna de cada entrada ni los actores del gobierno del catalogo.
// En produccion estos datos pueden proceder del repositorio publicado; el
// adaptador de fichero los toma del envoltorio DEMO versionado. La huella de
// procedencia declarada no equivale a una firma verificable del paquete.
type MetadatosFuenteCatalogos struct {
	Revision      string
	ActualizadaEn time.Time
	Demostracion  bool
	Aviso         string
}

type ConsultaMetadatosFuenteCatalogos interface {
	ObtenerMetadatosFuenteCatalogos(context.Context) (MetadatosFuenteCatalogos, error)
}

// RepositorioGobiernoCatalogos confirma cada cambio junto con su auditoria y
// outbox. Una version publicada o retirada nunca se sobrescribe.
type RepositorioGobiernoCatalogos interface {
	ConfirmarAltaBorradorCatalogo(context.Context, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarActualizacionBorradorCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarPublicacionCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
	ConfirmarRetiradaCatalogo(context.Context, string, domain.CatalogoConfigurable, domain.AuditEntry, domain.Event, EvidenciaUsoDecisionAutorizacion) error
}
