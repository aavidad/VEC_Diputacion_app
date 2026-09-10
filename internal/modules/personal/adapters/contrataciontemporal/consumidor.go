package contrataciontemporal

import (
	"context"
	"errors"
	"reflect"

	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/modules/personal/adapters/fuenteejercicio"
)

type Consumidor struct {
	fuente      *fuenteejercicio.Fuente
	terna       fuenteejercicio.TernaEsperada
	proveedor   ProveedorNominalAlta
	transaccion TransaccionAltaPersonal
	reloj       ctports.Reloj
}

// NuevoConsumidor valida los bytes con la terna de composición. No permite
// etiquetar una fuente ya construida con otra terna ni importa datos al arrancar.
func NuevoConsumidor(contenido []byte, terna fuenteejercicio.TernaEsperada, proveedor ProveedorNominalAlta, transaccion TransaccionAltaPersonal, reloj ctports.Reloj) (*Consumidor, error) {
	if nulo(proveedor) || nulo(transaccion) || nulo(reloj) {
		return nil, ErrNoDisponible
	}
	f, err := fuenteejercicio.NuevaFuenteEjercicio(contenido, terna)
	if err != nil {
		return nil, ErrSolicitudInvalida
	}
	return &Consumidor{fuente: f, terna: terna, proveedor: proveedor, transaccion: transaccion, reloj: reloj}, nil
}

func (c *Consumidor) SolicitarAlta(ctx context.Context, s ctports.SolicitudAltaPersonalRPT) (ctports.ResultadoAltaPersonalRPT, error) {
	var cero ctports.ResultadoAltaPersonalRPT
	if c == nil || ctx == nil || c.fuente == nil || nulo(c.proveedor) || nulo(c.transaccion) || nulo(c.reloj) {
		return cero, ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if s.Validar() != nil {
		return cero, ErrSolicitudInvalida
	}
	v, err := c.fuente.Resolver(ctx, s)
	if err != nil {
		return cero, fallo(ctx, err, ErrSolicitudInvalida)
	}
	p := PreparacionAlta{Solicitud: s, Fuente: c.terna, Vinculo: v}
	a, err := c.proveedor.ResolverAutoridad(ctx, p)
	if err != nil {
		return cero, fallo(ctx, err, ErrDenegado)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	antes := c.reloj.Ahora()
	if a.validar(antes) != nil {
		return cero, ErrDenegado
	}
	a.Contexto.Resultado, err = a.Contexto.Resultado.Clonar()
	if err != nil {
		return cero, ErrDenegado
	}
	actor, _ := a.Contexto.Vinculo.Datos()
	m := MaterialAlta{Preparacion: p, OrganizacionRef: a.OrganizacionRef, ActorRef: actor.PrincipalID, PerfilRef: actor.PerfilActivoRef}
	if m.Validar() != nil {
		return cero, ErrSolicitudInvalida
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	aut, err := c.proveedor.AutorizarAlta(ctx, m)
	if err != nil {
		return cero, fallo(ctx, err, ErrDenegado)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	orden := OrdenAlta{material: m, actor: a, autorizacion: aut, preparadaEn: antes}
	if orden.ValidarEn(c.reloj.Ahora()) != nil {
		return cero, ErrDenegado
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	r, err := c.transaccion.RegistrarORecuperarAlta(ctx, orden)
	if err != nil {
		return cero, fallo(ctx, err, ErrNoDisponible)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if r.ValidarPara(orden, c.reloj.Ahora()) != nil {
		return cero, ErrReciboNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	return r.Recibo.Resultado, nil
}

func nulo(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return r.IsNil()
	}
	return false
}

func fallo(ctx context.Context, err, opaco error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	case errors.Is(err, ErrConflicto):
		return ErrConflicto
	}
	return opaco
}

var _ ctports.AltaPersonalRPT = (*Consumidor)(nil)
