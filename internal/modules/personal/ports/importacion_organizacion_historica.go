package ports

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La política se inyecta desde una autoridad de fuentes. Una implementación
// sin fuente/diccionario/acto/custodia aprobados debe denegar, nunca suplirlos.
type PoliticaFuentesOrganizacionHistorica interface {
	AcreditarPublicacion(context.Context, domain.ManifiestoImportacionOrganizacion) (AcreditacionFuenteOrganizacionHistorica, error)
}

type AcreditacionFuenteOrganizacionHistorica struct {
	ManifiestoHuellaSHA256   string    `json:"manifiesto_huella_sha256"`
	OrganismoRef             string    `json:"organismo_ref"`
	Tipo                     string    `json:"tipo"`
	FuenteRef                string    `json:"fuente_ref"`
	FuenteVersion            string    `json:"fuente_version"`
	FuenteHuellaSHA256       string    `json:"fuente_huella_sha256"`
	DiccionarioRef           string    `json:"diccionario_ref"`
	ActoRef                  string    `json:"acto_ref"`
	CustodiaRef              string    `json:"custodia_ref"`
	AcreditacionRef          string    `json:"acreditacion_ref"`
	AcreditacionHuellaSHA256 string    `json:"acreditacion_huella_sha256"`
	AcreditadaEn             time.Time `json:"acreditada_en"`
}

// El verificador consume el catálogo común, fijando versión, revisión y huella.
// Una coincidencia de etiqueta no crea una identidad de unidad o clasificación.
type VerificadorCatalogosImportacionOrganizacion interface {
	ValidarReferencias(context.Context, domain.ReferenciaCatalogoImportacion, domain.ReferenciaCatalogoImportacion) error
	ValidarHechos(context.Context, domain.ReferenciaCatalogoImportacion, domain.ReferenciaCatalogoImportacion, []domain.HechoImportacionOrganizacion) error
}

type AutorizadorImportacionOrganizacion interface {
	AutorizarImportacionOrganizacion(context.Context, domain.MaterialImportacionOrganizacion) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type OrdenImportacionOrganizacion struct {
	Material     domain.MaterialImportacionOrganizacion
	Autorizacion vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	Acreditacion AcreditacionFuenteOrganizacionHistorica
}

type ReciboImportacionOrganizacion struct {
	ReciboRef            string                             `json:"recibo_ref"`
	LoteRef              string                             `json:"lote_ref"`
	Fase                 domain.FaseImportacionOrganizacion `json:"fase"`
	Estado               string                             `json:"estado"`
	RevisionAnterior     int64                              `json:"revision_anterior"`
	RevisionNueva        int64                              `json:"revision_nueva"`
	ClaveIdempotencia    string                             `json:"clave_idempotencia"`
	MaterialHuellaSHA256 string                             `json:"material_huella_sha256"`
	FuenteHuellaSHA256   string                             `json:"fuente_huella_sha256"`
	ActorRef             string                             `json:"actor_ref"`
	DecisionRef          string                             `json:"decision_ref"`
	AuditoriaRef         string                             `json:"auditoria_ref"`
	RegistradoEn         time.Time                          `json:"registrado_en"`
	Replay               bool                               `json:"replay"`
}

// Ejecutar realiza el consumo V3, CAS, historia, conciliaciones, auditoría,
// recibo y outbox en una sola transacción SERIALIZABLE de PostgreSQL 18.
// Publicar comprueba además revisión por actor distinto, cobertura íntegra y
// cero incidencias bloqueantes contra los datos durables del lote.
type RepositorioImportacionOrganizacion interface {
	Ejecutar(context.Context, OrdenImportacionOrganizacion) (ReciboImportacionOrganizacion, error)
}
