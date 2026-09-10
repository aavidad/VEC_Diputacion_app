package incorporacionejercicio

import (
	"context"
	"sync"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/adapters/contrataciontemporal"
)

// capturaAlta se crea por operación, no es un almacén ni fallback histórico.
// Un solo delegado y ninguna publicación hasta éxito, validación y cancelación.
type capturaAlta struct {
	mu           sync.Mutex
	delegado     personal.TransaccionAltaPersonal
	reloj        ct.Reloj
	usada, lista bool
	orden        personal.OrdenAlta
	resultado    personal.ResultadoTransaccionAlta
}

func (c *capturaAlta) RegistrarORecuperarAlta(ctx context.Context, o personal.OrdenAlta) (personal.ResultadoTransaccionAlta, error) {
	var cero personal.ResultadoTransaccionAlta
	if c == nil || ctx == nil || nulo(c.delegado) || nulo(c.reloj) {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	c.mu.Lock()
	if c.usada {
		c.mu.Unlock()
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	c.usada = true
	c.mu.Unlock()
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	antes := c.reloj.Ahora()
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	if o.ValidarEn(antes) != nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	r, e := c.delegado.RegistrarORecuperarAlta(ctx, o)
	if e != nil {
		return cero, fallo(ctx, e)
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	despues := c.reloj.Ahora()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	if despues.Before(antes) || r.ValidarPara(o, despues) != nil {
		return cero, ct.ErrComposicionIncorporacionAplicacion
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	c.orden = o
	c.resultado = r
	c.lista = true
	return r, nil
}
func (c *capturaAlta) tomar(ctx context.Context) (personal.OrdenAlta, personal.ResultadoTransaccionAlta, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	o, r := c.orden, c.resultado
	c.orden = personal.OrdenAlta{}
	c.resultado = personal.ResultadoTransaccionAlta{}
	lista := c.lista
	c.lista = false
	if e := ctx.Err(); e != nil {
		return personal.OrdenAlta{}, personal.ResultadoTransaccionAlta{}, e
	}
	if !lista {
		return personal.OrdenAlta{}, personal.ResultadoTransaccionAlta{}, ct.ErrComposicionIncorporacionAplicacion
	}
	return o, r, nil
}
