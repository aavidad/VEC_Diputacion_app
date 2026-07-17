package calculoexperienciaoficial

import (
	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// DatosOrdenConfiable solo admite capacidades obtenidas por la frontera de
// identidad y referencias exactas ya resueltas. No contiene DNI ni permite
// seleccionar el perfil de proteccion del servicio.
type DatosOrdenConfiable struct {
	bloqueoSerializacion
	ContextoActor             dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Selector                  puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo
	Motivo                    dominiovec.ReferenciaEntradaCatalogo
	CorrelacionLectura        dominiovec.ReferenciaCorrelacionAutorizacionV2
	CorrelacionEscritura      dominiovec.ReferenciaCorrelacionAutorizacionV2
	Causa                     oficial.CausaGobernadaV1
	MotorEsperado             oficial.VinculoMotorV1
	TipoEfecto                oficial.TipoEfectoV1
	Predecesor                *oficial.VinculoPredecesorV1
}

type OrdenCalculoExperienciaOficial struct {
	bloqueoSerializacion
	datos *DatosOrdenConfiable
}

func NuevaOrdenConfiable(
	datos DatosOrdenConfiable,
) (OrdenCalculoExperienciaOficial, error) {
	actor, err := datos.ContextoActor.Clonar()
	if err != nil || datos.VinculoAutenticacionActor.ValidarPara(actor) != nil ||
		validarDatosOrdenEstaticos(datos) != nil {
		return OrdenCalculoExperienciaOficial{}, ErrOrdenInvalida
	}
	datos.ContextoActor = actor
	if datos.Predecesor != nil {
		predecesor := *datos.Predecesor
		datos.Predecesor = &predecesor
	}
	return OrdenCalculoExperienciaOficial{datos: &datos}, nil
}

func (o OrdenCalculoExperienciaOficial) datosClonados() (DatosOrdenConfiable, error) {
	if o.datos == nil {
		return DatosOrdenConfiable{}, ErrOrdenInvalida
	}
	datos := *o.datos
	actor, err := datos.ContextoActor.Clonar()
	if err != nil {
		return DatosOrdenConfiable{}, ErrOrdenInvalida
	}
	datos.ContextoActor = actor
	if datos.Predecesor != nil {
		predecesor := *datos.Predecesor
		datos.Predecesor = &predecesor
	}
	if validarDatosOrdenEstaticos(datos) != nil {
		return DatosOrdenConfiable{}, ErrOrdenInvalida
	}
	return datos, nil
}

type datosResultadoEjecucion struct {
	resultado calculo.ResultadoExperienciaV1
	recibo    oficial.ReciboV1
	desenlace DesenlaceConfirmacionDuradera
}

type ResultadoEjecucion struct {
	bloqueoSerializacion
	datos *datosResultadoEjecucion
}

func (r ResultadoEjecucion) Resultado() (calculo.ResultadoExperienciaV1, error) {
	if r.datos == nil || r.datos.resultado.Validar() != nil {
		return calculo.ResultadoExperienciaV1{}, ErrResultadoNoConfiable
	}
	return r.datos.resultado, nil
}

func (r ResultadoEjecucion) Recibo() (oficial.ReciboV1, error) {
	if r.datos == nil || r.datos.recibo.Validar() != nil {
		return oficial.ReciboV1{}, ErrReciboNoConfiable
	}
	return r.datos.recibo, nil
}

func (r ResultadoEjecucion) Desenlace() (DesenlaceConfirmacionDuradera, error) {
	if r.datos == nil || !r.datos.desenlace.valido() {
		return "", ErrReciboNoConfiable
	}
	return r.datos.desenlace, nil
}

func (d DesenlaceConfirmacionDuradera) valido() bool {
	return d == ConfirmacionCreada || d == ConfirmacionReutilizada
}
