// Package sintetico is an offline, in-memory demonstration only. It deliberately
// does not implement ConsumidorV3 and never claims SQL commit, KMS or fsync.
package sintetico

import (
	"context"
	"sync"
	"time"
	dominio "vec-diputacion-granada/internal/modules/administracion/domain/ordenescopias"
	puerto "vec-diputacion-granada/internal/modules/administracion/ports/ordenescopias"
)

type Entorno struct {
	mu       sync.Mutex
	orden    dominio.Orden
	aceptada *puerto.Aceptacion
}

func Nuevo(o dominio.Orden) (*Entorno, error) {
	if _, e := o.Datos(); e != nil {
		return nil, dominio.ErrOrden
	}
	return &Entorno{orden: o}, nil
}
func (e *Entorno) LeerOrdenComprometida(ctx context.Context, ref string) (dominio.Orden, error) {
	d, err := e.orden.Datos()
	if ctx.Err() != nil || err != nil || ref != d.Orden {
		return dominio.Orden{}, dominio.ErrOrden
	}
	return e.orden, nil
}
func (e *Entorno) ValidarActual(ctx context.Context, o dominio.Orden, t time.Time) error {
	h, err := o.SHA256()
	expected, x := e.orden.SHA256()
	if ctx.Err() != nil || err != nil || x != nil || h != expected || o.ValidarEn(t) != nil {
		return dominio.ErrOrden
	}
	return nil
}
func (e *Entorno) AceptarOrden(ctx context.Context, o dominio.Orden) (puerto.Aceptacion, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	h, err := o.SHA256()
	original, x := e.orden.SHA256()
	d, y := o.Datos()
	if ctx.Err() != nil || err != nil || x != nil || y != nil || h != original {
		return puerto.Aceptacion{}, dominio.ErrOrden
	}
	if e.aceptada != nil {
		r := *e.aceptada
		r.Replay = true
		return r, nil
	}
	r := puerto.Aceptacion{Orden: d.Orden, SHA256: h, Recibo: "recibo:sintetico:" + h}
	e.aceptada = &r
	return r, nil
}
