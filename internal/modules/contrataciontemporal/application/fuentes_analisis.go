package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// VerificarValidacionRCConFuenteO3 coordina los puertos de fuente,
// verificación y publicación. Ports conserva sólo las comprobaciones locales.
func VerificarValidacionRCConFuenteO3(
	ctx context.Context,
	fuente ports.FuentePresupuestaria,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	publicaciones ports.VerificadorPublicacionMotivoFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	solicitud ports.SolicitudValidarRC,
) (ports.EvidenciaValidacionRCVerificadaO3, error) {
	if ctx == nil || dependenciaNula(fuente) ||
		dependenciaNula(verificador) || dependenciaNula(publicaciones) ||
		dependenciaNula(reloj) || solicitud.Validar() != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	return verificarValidacionRCConFuenteO3(
		operacion,
		fuente,
		verificador,
		publicaciones,
		confianza,
		reloj,
		solicitud,
	)
}

func verificarValidacionRCConFuenteO3(
	ctx context.Context,
	fuente ports.FuentePresupuestaria,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	publicaciones ports.VerificadorPublicacionMotivoFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	solicitud ports.SolicitudValidarRC,
) (ports.EvidenciaValidacionRCVerificadaO3, error) {
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrFuentePresupuestariaNoDisponible,
				err,
			)
	}
	material, err := ports.MaterialAutoridadesValidacionRCO3(
		solicitud,
		confianza,
	)
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	fuenteConfirmada, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		fuente,
		confianza,
		material,
		ports.RolFuentePresupuestaria,
		reloj.Ahora(),
	)
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{}, err
	}
	verificadorConfirmado, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		verificador,
		confianza,
		material,
		ports.RolVerificadorRespuesta,
		reloj.Ahora(),
	)
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{}, err
	}
	publicadorConfirmado, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		publicaciones,
		confianza,
		material,
		ports.RolPublicadorCatalogo,
		reloj.Ahora(),
	)
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{}, err
	}
	comprobadaAutoridad := reloj.Ahora()
	vinculoFuenteInicial, errFuenteInicial :=
		fuenteConfirmada.VinculoPara(
			material,
			ports.RolFuentePresupuestaria,
			comprobadaAutoridad,
		)
	vinculoVerificadorInicial, errVerificadorInicial :=
		verificadorConfirmado.VinculoPara(
			material,
			ports.RolVerificadorRespuesta,
			comprobadaAutoridad,
		)
	vinculoPublicadorInicial, errPublicadorInicial :=
		publicadorConfirmado.VinculoPara(
			material,
			ports.RolPublicadorCatalogo,
			comprobadaAutoridad,
		)
	if errFuenteInicial != nil || errVerificadorInicial != nil ||
		errPublicadorInicial != nil ||
		!ports.VinculosAutoridadFuenteAnalisisO3Separados(
			vinculoFuenteInicial,
			vinculoVerificadorInicial,
			vinculoPublicadorInicial,
		) {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := fuente.ValidarRC(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrFuentePresupuestariaNoDisponible,
				errContexto,
			)
	}
	if errFuente != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrFuentePresupuestariaNoDisponible,
				errFuente,
			)
	}
	recibidaEn := reloj.Ahora()
	if resultado.ValidarPara(solicitud, recibidaEn) != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	datosResultado, err := resultado.Datos()
	vinculoFuente, errVinculo := fuenteConfirmada.VinculoPara(
		material,
		ports.RolFuentePresupuestaria,
		recibidaEn,
	)
	if err != nil || errVinculo != nil ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			vinculoFuente.AutoridadRef {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	solicitudVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	confirmacion, err := verificarRespuestaFuenteAnalisis(
		ctx,
		verificador,
		verificadorConfirmado,
		material,
		solicitudVerificacion,
		reloj,
	)
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{}, err
	}
	confirmacionMotivo, err := verificarMotivoResultadoRC(
		ctx,
		publicaciones,
		publicadorConfirmado,
		material,
		resultado,
		reloj,
	)
	if err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{}, err
	}
	comprobadaEn := reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrVerificacionFuenteAnalisisNoDisponible,
				err,
			)
	}
	return ports.NuevaEvidenciaValidacionRCVerificadaO3(
		solicitud,
		resultado,
		confirmacion,
		confirmacionMotivo,
		fuenteConfirmada,
		verificadorConfirmado,
		publicadorConfirmado,
		confianza,
		comprobadaEn,
	)
}

func VerificarCalculoCosteConFuenteO3(
	ctx context.Context,
	calculador ports.CalculadorCostePersonal,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	solicitud ports.SolicitudCalcularCoste,
) (ports.EvidenciaCalculoCosteVerificadaO3, error) {
	if ctx == nil || dependenciaNula(calculador) ||
		dependenciaNula(verificador) || dependenciaNula(reloj) ||
		solicitud.Validar() != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	return verificarCalculoCosteConFuenteO3(
		operacion,
		calculador,
		verificador,
		confianza,
		reloj,
		solicitud,
	)
}

func verificarCalculoCosteConFuenteO3(
	ctx context.Context,
	calculador ports.CalculadorCostePersonal,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	solicitud ports.SolicitudCalcularCoste,
) (ports.EvidenciaCalculoCosteVerificadaO3, error) {
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrCalculadorCosteNoDisponible,
				err,
			)
	}
	material, err := ports.MaterialAutoridadesCalculoCosteO3(
		solicitud,
		confianza,
	)
	if err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	fuenteConfirmada, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		calculador,
		confianza,
		material,
		ports.RolCalculadorCoste,
		reloj.Ahora(),
	)
	if err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{}, err
	}
	verificadorConfirmado, err := presentarAutoridadFuenteAnalisisO3(
		ctx,
		verificador,
		confianza,
		material,
		ports.RolVerificadorRespuesta,
		reloj.Ahora(),
	)
	if err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{}, err
	}
	comprobadaAutoridad := reloj.Ahora()
	vinculoFuenteInicial, errFuenteInicial :=
		fuenteConfirmada.VinculoPara(
			material,
			ports.RolCalculadorCoste,
			comprobadaAutoridad,
		)
	vinculoVerificadorInicial, errVerificadorInicial :=
		verificadorConfirmado.VinculoPara(
			material,
			ports.RolVerificadorRespuesta,
			comprobadaAutoridad,
		)
	if errFuenteInicial != nil || errVerificadorInicial != nil ||
		!ports.VinculosAutoridadFuenteAnalisisO3Separados(
			vinculoFuenteInicial,
			vinculoVerificadorInicial,
		) {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := calculador.CalcularCoste(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrCalculadorCosteNoDisponible,
				errContexto,
			)
	}
	if errFuente != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrCalculadorCosteNoDisponible,
				errFuente,
			)
	}
	recibidaEn := reloj.Ahora()
	if resultado.ValidarPara(solicitud, recibidaEn) != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	datosResultado, err := resultado.Datos()
	vinculoFuente, errVinculo := fuenteConfirmada.VinculoPara(
		material,
		ports.RolCalculadorCoste,
		recibidaEn,
	)
	if err != nil || errVinculo != nil ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			vinculoFuente.AutoridadRef {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	solicitudVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	confirmacion, err := verificarRespuestaFuenteAnalisis(
		ctx,
		verificador,
		verificadorConfirmado,
		material,
		solicitudVerificacion,
		reloj,
	)
	if err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{}, err
	}
	comprobadaEn := reloj.Ahora()
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrVerificacionFuenteAnalisisNoDisponible,
				err,
			)
	}
	return ports.NuevaEvidenciaCalculoCosteVerificadaO3(
		solicitud,
		resultado,
		confirmacion,
		fuenteConfirmada,
		verificadorConfirmado,
		confianza,
		comprobadaEn,
	)
}

func verificarRespuestaFuenteAnalisis(
	ctx context.Context,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	identidad ports.ConfirmacionComprobacionAutoridadFuenteAnalisis,
	material []byte,
	solicitud ports.SolicitudVerificarRespuestaFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
) (ports.ConfirmacionRespuestaFuenteAnalisis, error) {
	confirmacion, errVerificacion :=
		verificador.VerificarRespuestaFuenteAnalisis(ctx, solicitud)
	if err := ctx.Err(); err != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrVerificacionFuenteAnalisisNoDisponible,
				err,
			)
	}
	if errVerificacion != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrVerificacionFuenteAnalisisNoDisponible,
				errVerificacion,
			)
	}
	verificadaEn := reloj.Ahora()
	vinculo, errVinculo := identidad.VinculoPara(
		material,
		ports.RolVerificadorRespuesta,
		verificadaEn,
	)
	datos, errDatos := confirmacion.Datos()
	if errVinculo != nil || errDatos != nil ||
		datos.VerificadorRef != vinculo.AutoridadRef ||
		confirmacion.ValidarPara(solicitud, verificadaEn) != nil {
		return ports.ConfirmacionRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return confirmacion, nil
}

func verificarMotivoResultadoRC(
	ctx context.Context,
	publicador ports.VerificadorPublicacionMotivoFuenteAnalisis,
	identidad ports.ConfirmacionComprobacionAutoridadFuenteAnalisis,
	material []byte,
	resultado ports.ResultadoValidacionRC,
	reloj ports.RelojFuenteAnalisis,
) (*ports.ConfirmacionPublicacionMotivoFuenteAnalisis, error) {
	solicitud, requerida, err := resultado.SolicitudPublicacionMotivo()
	if err != nil {
		return nil, ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	if !requerida {
		return nil, nil
	}
	confirmacion, errVerificacion :=
		publicador.VerificarPublicacionMotivoFuenteAnalisis(
			ctx,
			solicitud,
		)
	if errContexto := ctx.Err(); errContexto != nil {
		return nil, errorDisponibilidadFuenteAplicacion(
			ports.ErrVerificacionFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	comprobadaEn := reloj.Ahora()
	vinculo, errVinculo := identidad.VinculoPara(
		material,
		ports.RolPublicadorCatalogo,
		comprobadaEn,
	)
	datos, errDatos := confirmacion.Datos()
	if errVerificacion != nil || errVinculo != nil || errDatos != nil ||
		datos.PublicadorRef != vinculo.AutoridadRef ||
		confirmacion.ValidarPara(solicitud, comprobadaEn) != nil {
		return nil, errorDisponibilidadFuenteAplicacion(
			ports.ErrVerificacionFuenteAnalisisNoDisponible,
			errVerificacion,
		)
	}
	return &confirmacion, nil
}

func consumirEvidenciaFuenteAnalisisO3(
	ctx context.Context,
	consumidor ports.ConsumidorRespuestaFuenteAnalisis,
	orden ports.OrdenConsumoRespuestaFuenteAnalisis,
) (ports.ReciboConsumoRespuestaFuenteAnalisis, error) {
	if ctx == nil || dependenciaNula(consumidor) {
		return ports.ReciboConsumoRespuestaFuenteAnalisis{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConsumoRespuestaFuenteAnalisis{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrConsumoFuenteAnalisisNoDisponible,
				err,
			)
	}
	recibo, err := consumidor.ConsumirRespuestaFuenteAnalisis(ctx, orden)
	if err != nil {
		if errContexto := ctx.Err(); errContexto != nil {
			return ports.ReciboConsumoRespuestaFuenteAnalisis{},
				errorDisponibilidadFuenteAplicacion(
					ports.ErrConsumoFuenteAnalisisNoDisponible,
					errContexto,
				)
		}
		if errors.Is(err, ports.ErrRespuestaFuenteAnalisisYaConsumida) {
			return ports.ReciboConsumoRespuestaFuenteAnalisis{},
				ports.ErrRespuestaFuenteAnalisisYaConsumida
		}
		return ports.ReciboConsumoRespuestaFuenteAnalisis{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrConsumoFuenteAnalisisNoDisponible,
				err,
			)
	}
	if recibo.ValidarPara(orden) != nil {
		return ports.ReciboConsumoRespuestaFuenteAnalisis{},
			ports.ErrResultadoFuenteAnalisisNoConfiable
	}
	return recibo, nil
}

func ValidarRCConFuente(
	ctx context.Context,
	fuente ports.FuentePresupuestaria,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	publicaciones ports.VerificadorPublicacionMotivoFuenteAnalisis,
	consumidor ports.ConsumidorRespuestaFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	solicitud ports.SolicitudValidarRC,
) (domain.ValidacionRC, error) {
	if ctx == nil || dependenciaNula(consumidor) {
		return domain.ValidacionRC{}, ports.ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	evidencia, err := verificarValidacionRCConFuenteO3(
		operacion,
		fuente,
		verificador,
		publicaciones,
		confianza,
		reloj,
		solicitud,
	)
	if err != nil {
		return domain.ValidacionRC{}, err
	}
	orden, err := evidencia.OrdenConsumo()
	if err != nil {
		return domain.ValidacionRC{}, err
	}
	if _, err = consumirEvidenciaFuenteAnalisisO3(
		operacion,
		consumidor,
		orden,
	); err != nil {
		return domain.ValidacionRC{}, err
	}
	return evidencia.Materializar()
}

func CalcularCosteConFuente(
	ctx context.Context,
	calculador ports.CalculadorCostePersonal,
	verificador ports.VerificadorRespuestaFuenteAnalisis,
	consumidor ports.ConsumidorRespuestaFuenteAnalisis,
	confianza ports.ConfianzaAutoridadesFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	solicitud ports.SolicitudCalcularCoste,
) (ports.ResultadoCalculoCoste, error) {
	if ctx == nil || dependenciaNula(consumidor) {
		return ports.ResultadoCalculoCoste{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	evidencia, err := verificarCalculoCosteConFuenteO3(
		operacion,
		calculador,
		verificador,
		confianza,
		reloj,
		solicitud,
	)
	if err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	orden, err := evidencia.OrdenConsumo()
	if err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	if _, err = consumirEvidenciaFuenteAnalisisO3(
		operacion,
		consumidor,
		orden,
	); err != nil {
		return ports.ResultadoCalculoCoste{}, err
	}
	return evidencia.Resultado()
}

func errorDisponibilidadFuenteAplicacion(publico, causa error) error {
	var contexto error
	switch {
	case errors.Is(causa, context.Canceled):
		contexto = context.Canceled
	case errors.Is(causa, context.DeadlineExceeded):
		contexto = context.DeadlineExceeded
	}
	return errorFuenteAnalisisAplicacion{
		publico:  publico,
		contexto: contexto,
	}
}

type errorFuenteAnalisisAplicacion struct {
	publico  error
	contexto error
}

func (e errorFuenteAnalisisAplicacion) Error() string {
	return e.publico.Error()
}

func (e errorFuenteAnalisisAplicacion) Unwrap() []error {
	if e.contexto == nil {
		return []error{e.publico}
	}
	return []error{e.publico, e.contexto}
}
