package application

import (
	"context"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ServicioConfirmacionIncorporacionV2 recibe material de composición confiable,
// NO DTO del navegador. No emite alta Personal, no convierte a V1 ni reintenta.
// El acreditador previo comprueba disponibilidad/origen antes del efecto; la TX
// queda obligada a repetir la lectura autorizada en el MISMO commit CT.
type ServicioConfirmacionIncorporacionV2 struct {
	proveedor   ports.ProveedorAutorizacionConfirmacionIncorporacionV2
	acreditador ports.AcreditadorPersonalIncorporacion
	tx          ports.TransaccionRegistroIncorporacionV2
	reloj       ports.Reloj
}

func NuevoServicioConfirmacionIncorporacionV2(p ports.ProveedorAutorizacionConfirmacionIncorporacionV2,
	a ports.AcreditadorPersonalIncorporacion, tx ports.TransaccionRegistroIncorporacionV2, reloj ports.Reloj,
) (*ServicioConfirmacionIncorporacionV2, error) {
	if dependenciaNula(p) || dependenciaNula(a) || dependenciaNula(tx) || dependenciaNula(reloj) {
		return nil, ErrServicioConfirmacionIncorporacionInvalido
	}
	return &ServicioConfirmacionIncorporacionV2{p, a, tx, reloj}, nil
}

func (s *ServicioConfirmacionIncorporacionV2) Confirmar(ctx context.Context, m ports.MaterialConfirmacionIncorporacionV2) (ports.ReciboRegistroIncorporacionV2, error) {
	cero := ports.ReciboRegistroIncorporacionV2{}
	if s == nil || ctx == nil || dependenciaNula(s.proveedor) || dependenciaNula(s.acreditador) || dependenciaNula(s.tx) || dependenciaNula(s.reloj) {
		return cero, ErrServicioConfirmacionIncorporacionInvalido
	}
	if e := ctx.Err(); e != nil {
		return cero, e
	}
	// No canonizar una hora no válida ni ocultar retrocesos del reloj.
	reloj := &relojRegistroV2{base: s.reloj}
	antes := reloj.Ahora()
	if ports.ValidarMaterialRegistroIncorporacionV2(m, antes) != nil {
		return cero, ErrSolicitudConfirmacionIncorporacionInvalida
	}
	a, e := s.proveedor.AutorizarConfirmacionIncorporacion(ctx, m)
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	if e != nil {
		return cero, ErrConfirmacionIncorporacionDenegada
	}
	ahora := reloj.Ahora()
	o, e := ports.NuevaOrdenConfirmacionIncorporacionV2(m, a, ahora)
	if e != nil {
		return cero, ErrConfirmacionIncorporacionDenegada
	}
	previa, e := ports.AcreditarPersonalParaIncorporacion(ctx, o, s.acreditador, reloj)
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	if e != nil {
		return cero, normalizarFalloConfirmacionIncorporacion(ctx, e)
	}
	ahora = reloj.Ahora()
	if _, e = previa.RegistroPara(o, ahora); e != nil {
		return cero, ErrResultadoConfirmacionIncorporacionNoConfiable
	}
	if e = ctx.Err(); e != nil {
		return cero, e
	}
	r, e := s.tx.RegistrarORecuperarIncorporacion(ctx, o, previa)
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	if e != nil {
		return cero, normalizarFalloConfirmacionIncorporacion(ctx, e)
	}
	ahora = reloj.Ahora()
	if r.ValidarPara(o, ahora) != nil {
		return cero, ErrResultadoConfirmacionIncorporacionNoConfiable
	}
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	e = o.ValidarEn(reloj.Ahora())
	if ce := ctx.Err(); ce != nil {
		return cero, ce
	}
	if e != nil {
		return cero, ErrResultadoConfirmacionIncorporacionNoConfiable
	}
	return r.Recibo.Copia(), nil
}

// El reloj supervisado cubre también las llamadas internas del acreditador. Un
// retroceso queda latched como hora inválida, aunque el siguiente tick se recupere.
type relojRegistroV2 struct {
	base   ports.Reloj
	ultima time.Time
	roto   bool
}

func (r *relojRegistroV2) Ahora() time.Time {
	t := r.base.Ahora()
	if r.roto || (!r.ultima.IsZero() && t.Before(r.ultima)) {
		r.roto = true
		return time.Time{}
	}
	r.ultima = t
	return t
}
