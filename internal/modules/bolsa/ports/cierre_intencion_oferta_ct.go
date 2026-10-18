package ports

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"

	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const EsquemaCierreIntencionOfertaCT = "vec.bolsa.oferta-ct.cierre-intencion.v1"

var (
	ErrCierreIntencionOfertaCTInvalido     = errors.New("bolsa: cierre de intención de oferta CT inválido")
	ErrCierreIntencionOfertaCTNoDisponible = errors.New("bolsa: resolución de intención de oferta CT no disponible")
	ErrCierreIntencionOfertaCTIncierta     = errors.New("bolsa: resolución concurrente de intención de oferta CT incierta")
	actorCierreIntencionOfertaCT           = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
	ambitoCierreIntencionOfertaCT          = regexp.MustCompile(`^[A-Za-z0-9:_-]+$`)
)

// CanonCierreIntencionOfertaCT produce los bytes exactos del material V3 que
// comparten el autorizador y el adaptador SQL. El comando llega congelado de CT.
func CanonCierreIntencionOfertaCT(actorRef string, comando json.RawMessage) (string, error) {
	if !actorCierreIntencionOfertaCT.MatchString(actorRef) || len(comando) < 2 ||
		len(comando) > 1<<20 || !json.Valid(comando) || comando[0] != '{' {
		return "", ErrCierreIntencionOfertaCTInvalido
	}
	material, err := json.Marshal(struct {
		Esquema  string          `json:"esquema"`
		ActorRef string          `json:"actor_ref"`
		Comando  json.RawMessage `json:"comando"`
	}{EsquemaCierreIntencionOfertaCT, actorRef, comando})
	if err != nil || len(material) > 1<<20 {
		return "", ErrCierreIntencionOfertaCTInvalido
	}
	return string(material), nil
}

const (
	ResultadoIntencionOfertaPublicada = "publicada"
	ResultadoIntencionOfertaCerrada   = "cerrada"
)

// ComandoResolverIntencionOfertaCT mantiene el comando completo y la
// autorización fresca de cierre. Ningún campo procede de un selector HTTP de
// candidato, bolsa o cantidad.
type ComandoResolverIntencionOfertaCT struct {
	ActorRef         string
	UnidadRef        string
	AmbitoRef        string
	Comando          json.RawMessage
	MaterialCanonico string
	Material         puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (c ComandoResolverIntencionOfertaCT) Validar() error {
	canon, err := CanonCierreIntencionOfertaCT(c.ActorRef, c.Comando)
	if err != nil || c.MaterialCanonico != canon ||
		c.Material.ValidarEstructura() != nil ||
		!ambitoCierreIntencionOfertaCT.MatchString(c.UnidadRef) ||
		!ambitoCierreIntencionOfertaCT.MatchString(c.AmbitoRef) {
		return ErrCierreIntencionOfertaCTInvalido
	}
	return nil
}

// Cierre conserva el recibo Bolsa como JSON opaco. CT lo decodifica y
// comprueba contra su reserva; el adaptador Bolsa no importa el módulo CT.
type ResultadoResolverIntencionOfertaCT struct {
	Situacion string
	Oferta    *OfertaPublicada
	Cierre    json.RawMessage
}

type RepositorioResolucionIntencionOfertaCT interface {
	Resolver(context.Context, ComandoResolverIntencionOfertaCT) (ResultadoResolverIntencionOfertaCT, error)
}
