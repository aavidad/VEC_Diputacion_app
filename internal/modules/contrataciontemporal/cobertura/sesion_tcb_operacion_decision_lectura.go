package cobertura

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida = errors.New(
		"contratacion temporal: sesion de lectura primaria de cobertura invalida",
	)
	ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible = errors.New(
		"contratacion temporal: lectura primaria de cobertura no disponible",
	)
)

// EjecutorLecturaPrimariaTCBOperacionDecisionCobertura abre una transacción
// SERIALIZABLE READ ONLY contra el primario. El callback se invoca exactamente
// una vez, de forma síncrona, y no puede conservarse.
type EjecutorLecturaPrimariaTCBOperacionDecisionCobertura interface {
	EjecutarLecturaPrimariaTCB(
		context.Context,
		func(SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
	) error
}

// SesionLecturaPrimariaTCBOperacionDecisionCobertura recibe una consulta
// nominal que contiene la huella privada. Ningún canal obtiene esa huella ni
// puede fabricar la consulta fuera del despliegue controlado.
type SesionLecturaPrimariaTCBOperacionDecisionCobertura interface {
	LeerTerminalPrimario(
		context.Context,
		ConsultaPrimariaSesionTCBOperacionDecisionCobertura,
	) (DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura, error)
}

// DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura es la vista
// defensiva disponible únicamente dentro del callback técnico.
type DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Coordenadas       DatosConsultaPrimariaOperacionDecisionCobertura
	HuellaOrdenSHA256 string
}

// ConsultaPrimariaSesionTCBOperacionDecisionCobertura es opaca y no posee
// constructor público. Se deriva exclusivamente de la solicitud nominal.
type ConsultaPrimariaSesionTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	datos *DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura
}

func (c ConsultaPrimariaSesionTCBOperacionDecisionCobertura) Datos() (
	DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura,
	error,
) {
	if c.datos == nil ||
		validarDatosConsultaPrimariaOperacionDecisionCobertura(
			c.datos.Coordenadas,
		) != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			c.datos.HuellaOrdenSHA256,
		) {
		return DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura{},
			ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
	}
	return *c.datos, nil
}

// DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura es una
// observación cruda y no confiable. El núcleo vuelve a cotejar coordenadas,
// huella y recibo antes de elevarla a un resultado de reconciliación.
type DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Encontrado          bool
	Coordenadas         DatosConsultaPrimariaOperacionDecisionCobertura
	HuellaOrdenSHA256   string
	Recibo              ReciboOperacionDecisionCobertura
	ObservadaEnPrimario time.Time
}

type reconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB struct {
	ejecutor EjecutorLecturaPrimariaTCBOperacionDecisionCobertura
}

// NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB crea la
// única adaptación nominal entre el puerto de aplicación y la lectura
// primaria técnica. La solicitud no se entrega nunca al adaptador.
func NuevoReconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB(
	ejecutor EjecutorLecturaPrimariaTCBOperacionDecisionCobertura,
) (ReconciliadorResultadoAmbiguoOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ejecutor) {
		return nil,
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return &reconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB{
		ejecutor: ejecutor,
	}, nil
}

func (r *reconciliadorResultadoAmbiguoOperacionDecisionCoberturaTCB) ReconciliarResultadoAmbiguoOperacionDecisionCobertura(
	ctx context.Context,
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
) (ResultadoReconciliacionOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(r) ||
		dependenciaGobiernoOperacionCoberturaNula(r.ejecutor) ||
		validarSolicitudReconciliacionOperacionDecisionCobertura(
			solicitud,
		) != nil {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoReconciliacionOperacionDecisionCobertura{}, err
	}
	consulta := nuevaConsultaPrimariaSesionTCBOperacionDecisionCobertura(
		solicitud,
	)
	if _, err := consulta.Datos(); err != nil {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}

	control := &controlInvocacionLecturaPrimariaTCB{
		terminadaCh: make(chan struct{}),
	}
	ctxLectura, cancelar := context.WithCancel(ctx)
	defer cancelar()
	errEjecucion := r.ejecutor.EjecutarLecturaPrimariaTCB(
		ctxLectura,
		func(sesion SesionLecturaPrimariaTCBOperacionDecisionCobertura) error {
			if !control.iniciar() {
				return ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
			}
			if dependenciaGobiernoOperacionCoberturaNula(sesion) {
				control.terminar(
					ResultadoReconciliacionOperacionDecisionCobertura{},
					false,
				)
				return ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
			}
			crudo, err := sesion.LeerTerminalPrimario(ctxLectura, consulta)
			resultado, errResultado :=
				resultadoLecturaPrimariaTCBOperacionDecisionCobertura(
					solicitud,
					crudo,
				)
			valida := err == nil && errResultado == nil
			terminadaEnPlazo := control.terminar(resultado, valida)
			if !valida || !terminadaEnPlazo {
				return ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
			}
			return nil
		},
	)
	control.marcarRetornoEjecutor()
	cancelar()
	resultado, publicable, callbackPendiente := control.cerrar()
	if callbackPendiente || errEjecucion != nil || !publicable {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrEjecucionLecturaPrimariaTCBOperacionDecisionCoberturaNoDisponible
	}
	return resultado, nil
}

func validarSolicitudReconciliacionOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
) error {
	if solicitud.datos == nil ||
		validarDatosConsultaPrimariaOperacionDecisionCobertura(
			solicitud.datos.coordenadas,
		) != nil ||
		!huellaSHA256OperacionDecisionCoberturaValida(
			solicitud.datos.huellaOrdenSHA256,
		) {
		return ErrContratoConfirmacionOperacionDecisionCoberturaInvalido
	}
	return nil
}

func nuevaConsultaPrimariaSesionTCBOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
) ConsultaPrimariaSesionTCBOperacionDecisionCobertura {
	if validarSolicitudReconciliacionOperacionDecisionCobertura(solicitud) != nil {
		return ConsultaPrimariaSesionTCBOperacionDecisionCobertura{}
	}
	datos := DatosConsultaPrimariaSesionTCBOperacionDecisionCobertura{
		Coordenadas:       solicitud.datos.coordenadas,
		HuellaOrdenSHA256: solicitud.datos.huellaOrdenSHA256,
	}
	return ConsultaPrimariaSesionTCBOperacionDecisionCobertura{datos: &datos}
}

func resultadoLecturaPrimariaTCBOperacionDecisionCobertura(
	solicitud SolicitudReconciliacionOperacionDecisionCobertura,
	crudo DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura,
) (ResultadoReconciliacionOperacionDecisionCobertura, error) {
	if validarSolicitudReconciliacionOperacionDecisionCobertura(solicitud) != nil ||
		!instanteOperacionDecisionCoberturaValido(crudo.ObservadaEnPrimario) {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
	}
	if !crudo.Encontrado {
		if !datosLecturaPrimariaAusenteSonCero(crudo) {
			return ResultadoReconciliacionOperacionDecisionCobertura{},
				ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
		}
		return NuevaResultadoReconciliacionNoConcluyenteOperacionDecisionCobertura(
			solicitud,
			crudo.ObservadaEnPrimario,
		)
	}
	if !datosConsultaPrimariaOperacionDecisionCoberturaIguales(
		crudo.Coordenadas,
		solicitud.datos.coordenadas,
	) ||
		!referenciasOperacionDecisionCoberturaIguales(
			crudo.HuellaOrdenSHA256,
			solicitud.datos.huellaOrdenSHA256,
		) {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			ErrSesionLecturaPrimariaTCBOperacionDecisionCoberturaInvalida
	}
	return NuevaResultadoReconciliacionConfirmadaOperacionDecisionCobertura(
		solicitud,
		clonarReciboOperacionDecisionCobertura(crudo.Recibo),
		crudo.ObservadaEnPrimario,
	)
}

func datosLecturaPrimariaAusenteSonCero(
	datos DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura,
) bool {
	coordenadas := datos.Coordenadas
	recibo := datos.Recibo
	return coordenadas.OrganizacionRef == "" &&
		coordenadas.ExpedienteRef == "" &&
		coordenadas.VersionExpediente == 0 &&
		coordenadas.ReservaRef == "" &&
		coordenadas.ReciboRef == "" &&
		coordenadas.CorrelacionVECRef == "" &&
		coordenadas.DecisionVECRef == "" &&
		coordenadas.RevisionCercado == 0 &&
		datos.HuellaOrdenSHA256 == "" &&
		recibo.ReciboRef == "" && recibo.ReservaRef == "" &&
		recibo.AuditoriaRef == "" && recibo.CorrelacionVECRef == "" &&
		recibo.DecisionVECRef == "" &&
		recibo.DecisionVECHuellaSHA256 == "" &&
		recibo.CodigoProbatorioVEC == "" && !recibo.ConcedidaVEC &&
		recibo.RevisionCercado == 0 &&
		recibo.AmbitoIdempotenciaHMAC == "" &&
		recibo.HuellaSemanticaHMAC == "" &&
		recibo.ConfirmadaEn.IsZero() &&
		recibo.Aplicada == nil && recibo.DenegadaVEC == nil
}

type controlInvocacionLecturaPrimariaTCB struct {
	mu                    sync.Mutex
	terminadaCh           chan struct{}
	iniciada              bool
	terminada             bool
	terminadaAntesRetorno bool
	retornada             bool
	violacion             bool
	valida                bool
	resultado             ResultadoReconciliacionOperacionDecisionCobertura
}

func (c *controlInvocacionLecturaPrimariaTCB) iniciar() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.iniciada || c.retornada {
		c.violacion = true
		return false
	}
	c.iniciada = true
	return true
}

func (c *controlInvocacionLecturaPrimariaTCB) terminar(
	resultado ResultadoReconciliacionOperacionDecisionCobertura,
	valida bool,
) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.terminada {
		c.violacion = true
		return false
	}
	c.terminadaAntesRetorno = !c.retornada
	if c.retornada {
		c.violacion = true
	}
	c.terminada = true
	c.valida = valida
	if valida {
		c.resultado = resultado
	}
	close(c.terminadaCh)
	return c.terminadaAntesRetorno
}

func (c *controlInvocacionLecturaPrimariaTCB) marcarRetornoEjecutor() {
	c.mu.Lock()
	c.retornada = true
	c.mu.Unlock()
}

func (c *controlInvocacionLecturaPrimariaTCB) cerrar() (
	ResultadoReconciliacionOperacionDecisionCobertura,
	bool,
	bool,
) {
	c.mu.Lock()
	esperar := c.iniciada && !c.terminada
	terminadaCh := c.terminadaCh
	c.mu.Unlock()
	if esperar {
		temporizador := time.NewTimer(tiempoMaximoCierreCallbackInfractor)
		defer temporizador.Stop()
		select {
		case <-terminadaCh:
		case <-temporizador.C:
			c.mu.Lock()
			c.violacion = true
			c.mu.Unlock()
			return ResultadoReconciliacionOperacionDecisionCobertura{},
				false,
				true
		}
		c.mu.Lock()
		c.violacion = true
		c.mu.Unlock()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	publicable := c.iniciada && c.terminada && c.valida &&
		c.terminadaAntesRetorno && !c.violacion
	if !publicable {
		return ResultadoReconciliacionOperacionDecisionCobertura{},
			false,
			false
	}
	return c.resultado, true, false
}
