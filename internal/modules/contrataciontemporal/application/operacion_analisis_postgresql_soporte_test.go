package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type preparadorObservadoIntegracionO304 struct {
	delegado          ports.PreparadorOperacionAnalisisIdempotente
	solicitud         ports.SolicitudPrepararOperacionAnalisis
	err               error
	llamadas          int
	consultas         int
	errUltimaConsulta error
}

func (p *preparadorObservadoIntegracionO304) PrepararOperacionAnalisis(
	ctx context.Context,
	solicitud ports.SolicitudPrepararOperacionAnalisis,
) (ports.PreparacionOperacionAnalisis, error) {
	p.llamadas++
	p.solicitud = solicitud
	resultado, err := p.delegado.PrepararOperacionAnalisis(ctx, solicitud)
	p.err = err
	return resultado, err
}

func (p *preparadorObservadoIntegracionO304) ConsultarOperacionAnalisisConfirmada(
	ctx context.Context,
	solicitud ports.SolicitudConsultarOperacionAnalisisConfirmada,
) (ports.ReciboOperacionAnalisis, bool, error) {
	p.consultas++
	recibo, encontrado, err :=
		p.delegado.ConsultarOperacionAnalisisConfirmada(ctx, solicitud)
	p.errUltimaConsulta = err
	return recibo, encontrado, err
}

type transaccionObservadaIntegracionO304 struct {
	delegada ports.TransaccionOperacionesAnalisis
	llamadas int
	err      error
}

func (t *transaccionObservadaIntegracionO304) ConfirmarOperacionAnalisis(
	ctx context.Context,
	orden ports.OrdenConfirmarOperacionAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	t.llamadas++
	recibo, err := t.delegada.ConfirmarOperacionAnalisis(ctx, orden)
	t.err = err
	return recibo, err
}

type transaccionRespuestaPerdidaIntegracionO304 struct {
	delegada ports.TransaccionOperacionesAnalisis
}

func (t transaccionRespuestaPerdidaIntegracionO304) ConfirmarOperacionAnalisis(
	ctx context.Context,
	orden ports.OrdenConfirmarOperacionAnalisis,
) (ports.ReciboOperacionAnalisis, error) {
	_, err := t.delegada.ConfirmarOperacionAnalisis(ctx, orden)
	if err != nil {
		return ports.ReciboOperacionAnalisis{}, err
	}
	return ports.ReciboOperacionAnalisis{},
		ports.ErrPersistenciaOperacionAnalisisNoDisponible
}
