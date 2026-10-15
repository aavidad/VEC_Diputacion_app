package ports

import (
	"context"
	"errors"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	EsquemaGobiernoTransicionesSituacion     = "vec.bolsa.rrhh.politica-transiciones.v1"
	AccionPublicarTransicionesSituacion      = "bolsa.politica_transiciones.publicar"
	AccionConsultarTransicionesSituacion     = "bolsa.politica_transiciones.consultar"
	AudienciaPublicarTransicionesSituacion   = "vec_bolsa_llamamientos.politica_transiciones.publicar.v1"
	FinalidadGobiernoTransicionesSituacion   = "gobierno_politica_transiciones_bolsa"
	FinalidadConsultaTransicionesSituacion   = "consultar_politica_transiciones_bolsa"
	CampoConsultaTransicionesSituacion       = "politica_transiciones"
	TipoRecursoGobiernoTransicionesSituacion = "politica_transiciones_global"
	RecursoGobiernoTransicionesSituacion     = "bolsa:politica:transiciones:global"
)

var (
	ErrGobiernoTransicionesNoDisponible = errors.New("bolsa: gobierno de transiciones no disponible")
	ErrGobiernoTransicionesConflicto    = errors.New("bolsa: version o clave de transiciones en conflicto")
	ErrGobiernoTransicionesInvalido     = errors.New("bolsa: transiciones fuera de limites")
)

// LimitesTransicionesSituacion identifica una publicación confiable del
// catálogo común. La base contrasta de nuevo estos datos en cada POST.
type LimitesTransicionesSituacion struct {
	CatalogoRef     string   `json:"catalogo_ref"`
	CatalogoVersion int      `json:"catalogo_version"`
	CatalogoSHA256  string   `json:"catalogo_sha256"`
	Transiciones    []string `json:"transiciones"`
	UnidadRef       string   `json:"-"`
	AmbitoRef       string   `json:"-"`
}

type GobiernoTransicionesSituacion struct {
	Esquema                  string                       `json:"esquema"`
	Version                  int64                        `json:"version"`
	TransicionesGuardadas    []string                     `json:"transiciones_guardadas"`
	TransicionesEfectivas    []string                     `json:"transiciones_efectivas"`
	Limites                  LimitesTransicionesSituacion `json:"limites"`
	CatalogoActivado         bool                         `json:"catalogo_activado"`
	TransicionesObligatorias []string                     `json:"transiciones_obligatorias"`
	PuedePublicar            bool                         `json:"puede_publicar"`
	ReciboRef                string                       `json:"recibo_ref,omitempty"`
	Reutilizada              bool                         `json:"reutilizada"`
}

type ComandoGobiernoTransicionesSituacion struct {
	VersionEsperada   int64
	Transiciones      []string
	Motivo            string
	ClaveIdempotencia string
	ActorRef          string
	ReciboRef         string
	CatalogoVersion   int
	CatalogoSHA256    string
	Material          puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

type RepositorioGobiernoTransicionesSituacion interface {
	Leer(context.Context, int, string) (GobiernoTransicionesSituacion, error)
	Publicar(context.Context, ComandoGobiernoTransicionesSituacion) (GobiernoTransicionesSituacion, error)
}
