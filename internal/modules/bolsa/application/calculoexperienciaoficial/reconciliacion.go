package calculoexperienciaoficial

import (
	"context"
	"errors"
	"log/slog"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// IntentoReconciliacionCalculoOficial es una capacidad opaca creada antes de
// cruzar la frontera que puede enviar COMMIT. No concede permiso para repetir
// la confirmacion; solo permite consultar el mismo intento nominal.
//
// NO-GO REANUDACION EXTERNA: esta capacidad es deliberadamente in-memory y no
// puede serializarse ni reconstruirse solo desde ReferenciaOpaca. Produccion
// necesita un worker/acuse durable autenticado que recupere el intento tras un
// reinicio y vuelva a comprobar autoridad sin aceptar un identificador libre.
type IntentoReconciliacionCalculoOficial struct {
	bloqueoSerializacion
	perfil    perfilServicio
	solicitud SolicitudReconciliacionDuradera
	intencion oficial.IntencionResultadoV1
	resultado calculo.ResultadoExperienciaV1
}

func nuevoIntentoReconciliacion(
	perfil perfilServicio,
	correlacion dominiovec.ReferenciaCorrelacionAutorizacionV2,
	intencion oficial.IntencionResultadoV1,
	resultado calculo.ResultadoExperienciaV1,
) (IntentoReconciliacionCalculoOficial, error) {
	solicitud, err := NuevaSolicitudReconciliacionDuradera(correlacion, intencion)
	intento := IntentoReconciliacionCalculoOficial{
		perfil: perfil, solicitud: solicitud, intencion: intencion, resultado: resultado,
	}
	if err != nil || intento.validar() != nil {
		return IntentoReconciliacionCalculoOficial{}, ErrConfirmacionInvalida
	}
	return intento, nil
}

func (i IntentoReconciliacionCalculoOficial) validar() error {
	datos, err := i.solicitud.Datos()
	huellaIntencion, errIntencion := i.intencion.HuellaSHA256()
	huellaResultado, errResultado := i.resultado.HuellaSHA256()
	estado, fase, errEstado := estadoYFaseOficial(i.resultado)
	if (i.perfil != perfilExternoOrdinario && i.perfil != perfilInternoAlto) ||
		err != nil || errIntencion != nil || errResultado != nil || errEstado != nil ||
		i.resultado.Validar() != nil || datos.HuellaIntencionSHA256 != huellaIntencion ||
		i.intencion.HuellaResultadoSHA256() != huellaResultado ||
		i.intencion.Estado() != estado || i.intencion.Fase() != fase {
		return ErrConfirmacionInvalida
	}
	return nil
}

// ReferenciaOpaca devuelve el identificador nominal emitido antes del COMMIT.
// No contiene sujeto, convocatoria ni datos personales.
func (i IntentoReconciliacionCalculoOficial) ReferenciaOpaca() (string, error) {
	if i.validar() != nil {
		return "", ErrConfirmacionInvalida
	}
	datos, _ := i.solicitud.Datos()
	return datos.ReferenciaIntento, nil
}

// ErrorConfirmacionIndeterminada evita propagar errores de driver tras una
// frontera COMMIT ambigua y conserva la unica capacidad segura de consulta.
type ErrorConfirmacionIndeterminada struct {
	bloqueoSerializacion
	intento IntentoReconciliacionCalculoOficial
}

func nuevoErrorConfirmacionIndeterminada(
	intento IntentoReconciliacionCalculoOficial,
) *ErrorConfirmacionIndeterminada {
	return &ErrorConfirmacionIndeterminada{intento: intento}
}

func (*ErrorConfirmacionIndeterminada) Error() string {
	return "resultado oficial indeterminado; reconciliacion requerida"
}

func (*ErrorConfirmacionIndeterminada) Unwrap() []error {
	return []error{ErrResultadoConfirmacionIndeterminado, ErrReconciliacionRequerida}
}

func (e *ErrorConfirmacionIndeterminada) LogValue() slog.Value {
	return slog.StringValue(e.Error())
}

func (e *ErrorConfirmacionIndeterminada) Intento() (
	IntentoReconciliacionCalculoOficial,
	error,
) {
	if e == nil || e.intento.validar() != nil {
		return IntentoReconciliacionCalculoOficial{}, ErrConfirmacionInvalida
	}
	return e.intento, nil
}

// Reconciliar consulta un intento previo; nunca vuelve a llamar a Confirmar.
func (s *Servicio) Reconciliar(
	ctx context.Context,
	intento IntentoReconciliacionCalculoOficial,
) (ResultadoEjecucion, error) {
	if ctx == nil || !s.configuracionValida() || intento.validar() != nil ||
		intento.perfil != s.perfil {
		return ResultadoEjecucion{}, ErrServicioInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoEjecucion{}, err
	}
	resultado, err := s.reconciliar(ctx, intento)
	if err != nil {
		return ResultadoEjecucion{}, nuevoErrorConfirmacionIndeterminada(intento)
	}
	return resultado, nil
}

func (s *Servicio) reconciliar(
	ctx context.Context,
	intento IntentoReconciliacionCalculoOficial,
) (ResultadoEjecucion, error) {
	confirmacion, err := s.reconciliador.Reconciliar(ctx, intento.solicitud)
	if err != nil || confirmacion.ValidarParaReconciliacion(intento.solicitud) != nil {
		return ResultadoEjecucion{}, ErrReciboNoConfiable
	}
	return construirResultadoEjecucion(intento.resultado, confirmacion)
}

func construirResultadoEjecucion(
	resultado calculo.ResultadoExperienciaV1,
	confirmacion ResultadoConfirmacionDuradera,
) (ResultadoEjecucion, error) {
	datos, err := confirmacion.datosClonados()
	if err != nil || resultado.Validar() != nil {
		return ResultadoEjecucion{}, errors.Join(ErrReciboNoConfiable, err)
	}
	return ResultadoEjecucion{datos: &datosResultadoEjecucion{
		resultado: resultado, recibo: datos.Recibo, desenlace: datos.Desenlace,
	}}, nil
}
