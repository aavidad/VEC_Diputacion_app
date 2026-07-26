package cobertura

import (
	"context"
	"errors"
	"sync"
	"time"
)

// EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB abre una
// transacción SERIALIZABLE READ ONLY contra el primario. El callback debe
// ejecutarse exactamente una vez, de forma síncrona y sin conservarse.
type EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB interface {
	EjecutarLecturaResultadoHistoricoTCB(
		context.Context,
		func(SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB) error,
	) error
}

// SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB es la única
// superficie que implementa el adaptador durable. Devuelve observaciones
// crudas; nunca recibe ni fabrica la unión terminal del núcleo.
type SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB interface {
	LeerResultadoHistoricoTCB(
		context.Context,
		ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	) (DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB, error)
}

// ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB es nominal:
// no tiene constructor público y solo existe durante el callback técnico.
type ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB struct {
	bloqueoSerializacionOperacionDecisionCobertura
	solicitud *SolicitudRecuperacionResultadoOperacionDecisionCobertura
}

func (c ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB) DatosLectura() (
	DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura,
	error,
) {
	if c.solicitud == nil {
		return DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido
	}
	datos, err := c.solicitud.DatosLectura()
	if err != nil {
		return DatosSolicitudRecuperacionResultadoOperacionDecisionCobertura{},
			ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido
	}
	return datos, nil
}

// DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB es una
// proyección exterior no confiable. El núcleo coteja todas las coordenadas y
// referencias antes de construir confirmado o no observable.
type DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB struct {
	bloqueoSerializacionOperacionDecisionCobertura
	Encontrado  bool
	Reserva     DatosReservaTerminalOperacionDecisionCobertura
	Recibo      ReciboOperacionDecisionCobertura
	ObservadaEn time.Time
}

type lectorResultadoHistoricoOperacionDecisionCoberturaTCB struct {
	ejecutor EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB
}

// NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB conserva en el
// núcleo la construcción de la unión. El adaptador solo aporta el ejecutor
// técnico de lectura primaria.
func NuevoLectorResultadoHistoricoOperacionDecisionCoberturaTCB(
	ejecutor EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) (LectorResultadoHistoricoOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ejecutor) {
		return nil,
			ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido
	}
	return &lectorResultadoHistoricoOperacionDecisionCoberturaTCB{
		ejecutor: ejecutor,
	}, nil
}

func (*lectorResultadoHistoricoOperacionDecisionCoberturaTCB) lectorResultadoHistoricoOperacionDecisionCoberturaSellado() {
}

func (l *lectorResultadoHistoricoOperacionDecisionCoberturaTCB) LeerResultadoHistoricoOperacionDecisionCobertura(
	ctx context.Context,
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
) (ResultadoHistoricoOperacionDecisionCobertura, error) {
	if dependenciaGobiernoOperacionCoberturaNula(ctx) ||
		dependenciaGobiernoOperacionCoberturaNula(l) ||
		dependenciaGobiernoOperacionCoberturaNula(l.ejecutor) {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{}, err
	}
	copiaSolicitud, err :=
		clonarSolicitudRecuperacionResultadoOperacionDecisionCobertura(solicitud)
	if err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido
	}
	consulta := ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{
		solicitud: &copiaSolicitud,
	}
	if _, err := consulta.DatosLectura(); err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrContratoLecturaResultadoHistoricoOperacionDecisionCoberturaInvalido
	}

	control := &controlInvocacionResultadoHistoricoOperacionDecisionCoberturaTCB{
		terminadaCh: make(chan struct{}),
	}
	ctxLectura, cancelar := context.WithCancel(ctx)
	defer cancelar()
	errEjecucion := ejecutarLecturaResultadoHistoricoTCBSegura(
		l.ejecutor,
		ctxLectura,
		func(
			sesion SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
		) (errCallback error) {
			if !control.iniciar() {
				return ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
			}
			terminada := false
			defer func() {
				if recover() == nil {
					return
				}
				if !terminada {
					control.terminar(
						ResultadoHistoricoOperacionDecisionCobertura{},
						ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
					)
				}
				errCallback =
					ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
			}()
			if dependenciaGobiernoOperacionCoberturaNula(sesion) {
				control.terminar(
					ResultadoHistoricoOperacionDecisionCobertura{},
					ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
				)
				terminada = true
				return ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
			}
			crudo, errLectura := sesion.LeerResultadoHistoricoTCB(
				ctxLectura,
				consulta,
			)
			resultado, errResultado := resultadoLecturaHistoricaTCB(
				ctxLectura,
				solicitud,
				crudo,
				errLectura,
			)
			terminadaEnPlazo := control.terminar(resultado, errResultado)
			terminada = true
			if !terminadaEnPlazo {
				return ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
			}
			return errResultado
		},
	)
	control.marcarRetornoEjecutor()
	cancelar()

	resultado, errResultado, protocoloValido, iniciada, pendiente :=
		control.cerrar()
	if errContexto := ctx.Err(); errContexto != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{}, errContexto
	}
	if pendiente || (iniciada && !protocoloValido) {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	if !iniciada {
		if errEjecucion != nil {
			return ResultadoHistoricoOperacionDecisionCobertura{},
				clasificarErrorLecturaResultadoHistoricoTCB(ctx, errEjecucion)
		}
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	if errResultado != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{}, errResultado
	}
	if errEjecucion != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			clasificarErrorLecturaResultadoHistoricoTCB(ctx, errEjecucion)
	}
	return resultado, nil
}

func ejecutarLecturaResultadoHistoricoTCBSegura(
	ejecutor EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	ctx context.Context,
	callback func(
		SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	) error,
) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
		}
	}()
	return ejecutor.EjecutarLecturaResultadoHistoricoTCB(ctx, callback)
}

func resultadoLecturaHistoricaTCB(
	ctx context.Context,
	solicitud SolicitudRecuperacionResultadoOperacionDecisionCobertura,
	crudo DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	errLectura error,
) (ResultadoHistoricoOperacionDecisionCobertura, error) {
	if err := ctx.Err(); err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{}, err
	}
	if errLectura != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			clasificarErrorLecturaResultadoHistoricoTCB(ctx, errLectura)
	}
	if !instanteOperacionDecisionCoberturaValido(crudo.ObservadaEn) {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	if !crudo.Encontrado {
		if crudo.Reserva !=
			(DatosReservaTerminalOperacionDecisionCobertura{}) ||
			crudo.Recibo != (ReciboOperacionDecisionCobertura{}) {
			return ResultadoHistoricoOperacionDecisionCobertura{},
				ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
		}
		resultado, err :=
			nuevoResultadoHistoricoNoObservableOperacionDecisionCobertura(
				solicitud,
				crudo.ObservadaEn,
			)
		if err != nil {
			return ResultadoHistoricoOperacionDecisionCobertura{},
				ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
		}
		return resultado, nil
	}
	resultado, err :=
		nuevoResultadoHistoricoConfirmadoOperacionDecisionCobertura(
			solicitud,
			DatosEvidenciaPersistidaResultadoOperacionDecisionCobertura{
				Reserva:     crudo.Reserva,
				Recibo:      crudo.Recibo,
				ObservadaEn: crudo.ObservadaEn,
			},
		)
	if err != nil {
		return ResultadoHistoricoOperacionDecisionCobertura{},
			ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	return resultado, nil
}

func clasificarErrorLecturaResultadoHistoricoTCB(
	ctx context.Context,
	causa error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	switch {
	case errors.Is(
		causa,
		ErrHistoriaResultadoOperacionDecisionCoberturaDivergente,
	):
		return ErrHistoriaResultadoOperacionDecisionCoberturaDivergente
	case errors.Is(
		causa,
		ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable,
	):
		return ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	case errors.Is(
		causa,
		ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible,
	):
		return ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	default:
		return ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
}

type controlInvocacionResultadoHistoricoOperacionDecisionCoberturaTCB struct {
	mu                    sync.Mutex
	terminadaCh           chan struct{}
	iniciada              bool
	terminada             bool
	terminadaAntesRetorno bool
	retornada             bool
	violacion             bool
	resultado             ResultadoHistoricoOperacionDecisionCobertura
	err                   error
}

func (c *controlInvocacionResultadoHistoricoOperacionDecisionCoberturaTCB) iniciar() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.iniciada || c.retornada {
		c.violacion = true
		return false
	}
	c.iniciada = true
	return true
}

func (c *controlInvocacionResultadoHistoricoOperacionDecisionCoberturaTCB) terminar(
	resultado ResultadoHistoricoOperacionDecisionCobertura,
	err error,
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
	c.resultado = resultado
	c.err = err
	close(c.terminadaCh)
	return c.terminadaAntesRetorno
}

func (c *controlInvocacionResultadoHistoricoOperacionDecisionCoberturaTCB) marcarRetornoEjecutor() {
	c.mu.Lock()
	c.retornada = true
	c.mu.Unlock()
}

func (c *controlInvocacionResultadoHistoricoOperacionDecisionCoberturaTCB) cerrar() (
	ResultadoHistoricoOperacionDecisionCobertura,
	error,
	bool,
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
			return ResultadoHistoricoOperacionDecisionCobertura{},
				nil, false, true, true
		}
		c.mu.Lock()
		c.violacion = true
		c.mu.Unlock()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	protocoloValido := c.iniciada && c.terminada &&
		c.terminadaAntesRetorno && !c.violacion
	return c.resultado, c.err, protocoloValido, c.iniciada, false
}
