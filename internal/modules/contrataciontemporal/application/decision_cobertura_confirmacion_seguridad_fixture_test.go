package application

import (
	"context"
	"sync"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type relojRevocacionConfirmacionPrueba struct {
	mu               sync.Mutex
	delegado         cobertura.RelojGobiernoOperacionCobertura
	reloj            *relojCoberturaAplicacionPrueba
	revocarEnLlamada int
	avanceRevocacion time.Duration
	llamadas         int
}

func (r *relojRevocacionConfirmacionPrueba) AhoraGobiernoOperacionCobertura(
	ctx context.Context,
) (time.Time, error) {
	r.mu.Lock()
	r.llamadas++
	llamada := r.llamadas
	if llamada == r.revocarEnLlamada {
		r.reloj.fijar(r.reloj.Ahora().Add(r.avanceRevocacion))
	}
	r.mu.Unlock()
	return r.delegado.AhoraGobiernoOperacionCobertura(ctx)
}

func (r *relojRevocacionConfirmacionPrueba) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.llamadas
}

type preparadorObservadorConfirmacionPrueba struct {
	mu          sync.Mutex
	delegado    puertosvec.PreparadorRegistroCompuestoSolicitudLigadaV3
	adulterar   bool
	correlacion string
	decisionRef string
}

func (p *preparadorObservadorConfirmacionPrueba) PrepararRegistroCompuestoSolicitudLigadaV3(
	ctx context.Context,
	solicitud dominiovec.SolicitudAutorizacionLigadaV3,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	generador puertosvec.GeneradorReferenciaDecisionAutorizacion,
) (
	dominiovec.DecisionAutorizacionLigadaV3,
	puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3,
	error,
) {
	datos, err := solicitud.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			err
	}
	correlacion, err := datos.Correlacion.ValorCanonico()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			err
	}
	decision, candidata, err :=
		p.delegado.PrepararRegistroCompuestoSolicitudLigadaV3(
			ctx,
			solicitud,
			resultado,
			generador,
		)
	if err != nil {
		return decision, candidata, err
	}
	resumen, err := candidata.Resumen()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			err
	}
	datosResumen, err := resumen.Datos()
	if err != nil {
		return dominiovec.DecisionAutorizacionLigadaV3{},
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			err
	}
	p.mu.Lock()
	p.correlacion = correlacion
	p.decisionRef = datosResumen.DecisionRef
	adulterar := p.adulterar
	p.mu.Unlock()
	if adulterar {
		return decision,
			puertosvec.CandidataRegistroDecisionAutorizacionLigadaV3{},
			nil
	}
	return decision, candidata, nil
}

func (p *preparadorObservadorConfirmacionPrueba) referencias() (
	string,
	string,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.correlacion, p.decisionRef
}

type transaccionSalidaInvalidaConfirmacionPrueba struct {
	mu       sync.Mutex
	llamadas int
}

func (t *transaccionSalidaInvalidaConfirmacionPrueba) ConfirmarOperacionDecisionCobertura(
	context.Context,
	cobertura.OrdenOperacionDecisionCobertura,
) (cobertura.ResultadoConfirmacionOperacionDecisionCobertura, error) {
	t.mu.Lock()
	t.llamadas++
	t.mu.Unlock()
	return cobertura.ResultadoConfirmacionOperacionDecisionCobertura{}, nil
}

func (t *transaccionSalidaInvalidaConfirmacionPrueba) total() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.llamadas
}
