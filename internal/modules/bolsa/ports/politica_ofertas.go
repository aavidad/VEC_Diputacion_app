package ports

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionConsultarPoliticaOfertas    = "bolsa.politica_ofertas.consultar"
	AudienciaConsultarPoliticaOfertas = "vec_bolsa_llamamientos.politica_ofertas.consultar.v1"
	FinalidadConsultarPoliticaOfertas = "consultar_politica_ofertas_bolsa"
	CampoConsultarPoliticaOfertas     = "politica_ofertas"
	AccionPublicarPoliticaOfertas     = "bolsa.politica_ofertas.publicar"
	AudienciaPublicarPoliticaOfertas  = "vec_bolsa_llamamientos.politica_ofertas.publicar.v1"
	FinalidadPoliticaOfertas          = "gobierno_politica_ofertas_bolsa"
	EsquemaPoliticaOfertas            = "vec.bolsa.rrhh.politica-ofertas.v1"
)

var (
	ErrPoliticaOfertasNoDisponible = errors.New("bolsa: politica de ofertas no disponible")
	ErrPoliticaOfertasConflicto    = errors.New("bolsa: version o clave de politica en conflicto")
)

// VersionPoliticaOfertas es una publicación inmutable. Version 0 representa
// ausencia de configuración: publicar ofertas debe fallar cerrado.
type VersionPoliticaOfertas struct {
	BolsaRef     string                  `json:"bolsa_ref"`
	Version      int64                   `json:"version"`
	HuellaSHA256 string                  `json:"huella_sha256,omitempty"`
	Ejemplo      bool                    `json:"ejemplo"`
	Configurada  bool                    `json:"configurada"`
	Politica     *domain.PoliticaOfertas `json:"politica"`
	ReciboRef    string                  `json:"recibo_ref,omitempty"`
	PublicadaEn  time.Time               `json:"publicada_en,omitempty"`
	Reutilizada  bool                    `json:"reutilizada,omitempty"`
}

type ComandoPublicarPoliticaOfertas struct {
	BolsaRef, ActorRef, ClaveIdempotencia, ReciboRef string
	VersionEsperada                                  int64
	Politica                                         domain.PoliticaOfertas
	Material                                         puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type ConsultaPoliticaOfertasAutorizada struct {
	BolsaRef string
	Material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// El repositorio consume la autorización y confirma versión, recibo y
// auditoría en la misma transacción PostgreSQL. La lectura no confía en una
// versión aportada por el navegador.
type RepositorioPoliticaOfertas interface {
	Vigente(context.Context, string) (VersionPoliticaOfertas, error)
	ConsultarAutorizada(context.Context, ConsultaPoliticaOfertasAutorizada) (VersionPoliticaOfertas, error)
	Publicar(context.Context, ComandoPublicarPoliticaOfertas) (VersionPoliticaOfertas, error)
}
