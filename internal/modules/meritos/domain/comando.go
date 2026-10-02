package domain

import (
	"encoding/json"
	"sort"

	vec "vec-diputacion-granada/internal/vec/domain"
)

const EsquemaComandoHecho = "vec.meritos.hecho.operacion.v1"

// ComandoHecho fija la preimagen que liga contenido, motivo y versión esperada
// al recurso nominal. No contiene permisos, decisiones ni material de identidad.
type ComandoHecho struct {
	Esquema           string                        `json:"esquema"`
	Accion            string                        `json:"accion"`
	ActorRef          string                        `json:"actor_ref"`
	ClaveIdempotencia string                        `json:"clave_idempotencia"`
	VersionEsperada   int                           `json:"version_esperada"`
	Hecho             Hecho                         `json:"hecho"`
	Motivo            vec.ReferenciaEntradaCatalogo `json:"motivo"`
	FechaCorte        string                        `json:"fecha_corte"`
}

func (c ComandoHecho) RepresentacionCanonica() ([]byte, error) {
	if c.Esquema != EsquemaComandoHecho || !ReferenciaValida(c.ActorRef) || !ReferenciaValida(c.ClaveIdempotencia) ||
		c.VersionEsperada < 0 || c.VersionEsperada >= 1<<30 || c.Hecho.Version != c.VersionEsperada+1 ||
		c.Hecho.Validar() != nil || c.Hecho.Estado != Declarado || c.Hecho.Revision != nil ||
		!vec.ReferenciaMotivoAutorizacionV2Valida(c.Motivo) || c.Motivo.CatalogoVersion > 1<<31-1 || !fechaValida(c.FechaCorte) ||
		(c.Hecho.Horas != nil && *c.Hecho.Horas > 1<<31-1) {
		return nil, ErrHecho
	}
	switch c.Accion {
	case "meritos.hecho.declarar", "meritos.hecho.verificar", "meritos.hecho.rechazar", "meritos.hecho.rectificar":
	default:
		return nil, ErrHecho
	}
	c.Hecho.Evidencias = append([]vec.ReferenciaDocumento{}, c.Hecho.Evidencias...)
	for _, ev := range c.Hecho.Evidencias {
		if ev.Version > 1<<31-1 {
			return nil, ErrHecho
		}
	}
	sort.Slice(c.Hecho.Evidencias, func(i, j int) bool {
		if c.Hecho.Evidencias[i].ID == c.Hecho.Evidencias[j].ID {
			return c.Hecho.Evidencias[i].Version < c.Hecho.Evidencias[j].Version
		}
		return c.Hecho.Evidencias[i].ID < c.Hecho.Evidencias[j].ID
	})
	return json.Marshal(c)
}
