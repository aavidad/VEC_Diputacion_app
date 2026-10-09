package ports

import (
	"context"
	"errors"
	"time"

	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionVincularEmisionBolsa     = "contratacion_temporal.bolsa.vincular"
	AudienciaVincularEmisionBolsa  = "vec_contratacion_temporal.vinculo_emision_bolsa.v1"
	TipoRecursoVinculoEmisionBolsa = "vinculo_bolsa_expediente"
	FinalidadVinculoEmisionBolsa   = "tramitacion_expediente_contratacion_temporal"
	EsquemaVinculoEmisionBolsa     = "vec.ct.vinculo-emision-bolsa.v1"
)

var (
	ErrVinculoEmisionBolsaInvalido     = errors.New("ct: vínculo de emisión Bolsa inválido")
	ErrVinculoEmisionBolsaNoDisponible = errors.New("ct: vínculo de emisión Bolsa no disponible")
	ErrVinculoEmisionBolsaConflicto    = errors.New("ct: vínculo de emisión Bolsa en conflicto")
)

type SolicitudVinculoEmisionBolsa struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	BolsaRef          string
	LlamamientoRef    string
	ReciboEmisionRef  string
	ClaveIdempotencia string
}

// MaterialVinculoEmisionBolsa conserva el orden exacto del canon consumido
// por CT201. No contiene actor ni perfil aportados por el navegador.
type MaterialVinculoEmisionBolsa struct {
	Esquema           string `json:"esquema"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionEsperada   uint64 `json:"version_esperada"`
	BolsaRef          string `json:"bolsa_ref"`
	LlamamientoRef    string `json:"llamamiento_ref"`
	ReciboEmisionRef  string `json:"recibo_emision_ref"`
	ClaveIdempotencia string `json:"clave_idempotencia"`
}

type ReciboVinculoEmisionBolsa struct {
	ExpedienteRef    string    `json:"expediente_ref"`
	BolsaRef         string    `json:"bolsa_ref"`
	LlamamientoRef   string    `json:"llamamiento_ref"`
	ReciboEmisionRef string    `json:"recibo_emision_ref"`
	ReciboVinculoRef string    `json:"recibo_vinculo_ref"`
	VinculadoEn      time.Time `json:"vinculado_en"`
	AuditoriaRef     string    `json:"auditoria_ref"`
	EventoRef        string    `json:"evento_ref"`
	Reutilizado      bool      `json:"reutilizado"`
}

type AutorizadorVinculoEmisionBolsa interface {
	AutorizarVinculoEmisionBolsa(context.Context, SolicitudVinculoEmisionBolsa, string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type RepositorioVinculoEmisionBolsa interface {
	CodificarMaterialVinculoEmisionBolsa(SolicitudVinculoEmisionBolsa) ([]byte, string, error)
	RegistrarVinculoEmisionBolsa(context.Context, []byte, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ReciboVinculoEmisionBolsa, error)
}

// LectorAmbitosVinculoEmisionBolsa obtiene sólo centro y categoría de la
// versión actual, después de comprobar organización y bolsa de cobertura.
type LectorAmbitosVinculoEmisionBolsa interface {
	LeerAmbitosVinculoEmisionBolsa(context.Context, SolicitudVinculoEmisionBolsa) (centroRef, categoriaRef string, err error)
}
