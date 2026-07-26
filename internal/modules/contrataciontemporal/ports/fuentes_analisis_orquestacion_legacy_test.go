package ports

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// Estas ayudas conservan las regresiones históricas del puerto después de
// mover la coordinación productiva a application. No forman parte del paquete
// compilado en producción.
func presentarConfirmacionAutoridadFuenteAnalisisPrueba(
	ctx context.Context,
	presentador PresentadorAutoridadFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	material []byte,
	rol RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (ConfirmacionComprobacionAutoridadFuenteAnalisis, error) {
	comprobacion, err := NuevaComprobacionAutoridadFuenteAnalisis(
		confianza,
		material,
		rol,
		comprobadaEn,
	)
	if err != nil {
		return ConfirmacionComprobacionAutoridadFuenteAnalisis{}, err
	}
	desafio, err := comprobacion.Desafio()
	if err != nil {
		return ConfirmacionComprobacionAutoridadFuenteAnalisis{}, err
	}
	presentacion, errPresentacion :=
		presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
	if errContexto := ctx.Err(); errContexto != nil {
		return ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errContexto,
			)
	}
	confirmacion, errVerificacion :=
		comprobacion.ValidarPresentacion(presentacion, comprobadaEn)
	if errPresentacion != nil || errVerificacion != nil {
		return ConfirmacionComprobacionAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return confirmacion, nil
}

func verificarValidacionRCConFuenteO3Prueba(
	ctx context.Context,
	fuente FuentePresupuestaria,
	verificador VerificadorRespuestaFuenteAnalisis,
	publicador VerificadorPublicacionMotivoFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudValidarRC,
) (EvidenciaValidacionRCVerificadaO3, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(fuente) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(publicador) ||
		dependenciaNulaFuenteAnalisis(reloj) {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrPeticionFuenteAnalisisInvalida
	}
	material, err := MaterialAutoridadesValidacionRCO3(solicitud, confianza)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	confirmacionFuente, err :=
		presentarConfirmacionAutoridadFuenteAnalisisPrueba(
			ctx,
			fuente,
			confianza,
			material,
			RolFuentePresupuestaria,
			reloj.Ahora(),
		)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	confirmacionVerificador, err :=
		presentarConfirmacionAutoridadFuenteAnalisisPrueba(
			ctx,
			verificador,
			confianza,
			material,
			RolVerificadorRespuesta,
			reloj.Ahora(),
		)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	confirmacionPublicador, err :=
		presentarConfirmacionAutoridadFuenteAnalisisPrueba(
			ctx,
			publicador,
			confianza,
			material,
			RolPublicadorCatalogo,
			reloj.Ahora(),
		)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	comprobadaAutoridad := reloj.Ahora()
	vinculoFuente, errFuente := confirmacionFuente.VinculoPara(
		material,
		RolFuentePresupuestaria,
		comprobadaAutoridad,
	)
	vinculoVerificador, errVerificador :=
		confirmacionVerificador.VinculoPara(
			material,
			RolVerificadorRespuesta,
			comprobadaAutoridad,
		)
	vinculoPublicador, errPublicador :=
		confirmacionPublicador.VinculoPara(
			material,
			RolPublicadorCatalogo,
			comprobadaAutoridad,
		)
	if errFuente != nil || errVerificador != nil ||
		errPublicador != nil ||
		!VinculosAutoridadFuenteAnalisisO3Separados(
			vinculoFuente,
			vinculoVerificador,
			vinculoPublicador,
		) {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := fuente.ValidarRC(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrFuentePresupuestariaNoDisponible,
				errContexto,
			)
	}
	if errFuente != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrFuentePresupuestariaNoDisponible,
				errFuente,
			)
	}
	solicitudVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	confirmacion, errVerificacion :=
		verificador.VerificarRespuestaFuenteAnalisis(
			ctx,
			solicitudVerificacion,
		)
	if errVerificacion != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errVerificacion,
			)
	}
	if errContexto := ctx.Err(); errContexto != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errContexto,
			)
	}
	var confirmacionMotivo *ConfirmacionPublicacionMotivoFuenteAnalisis
	solicitudMotivo, requiereMotivo, err :=
		resultado.SolicitudPublicacionMotivo()
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	if requiereMotivo {
		obtenida, errPublicador :=
			publicador.VerificarPublicacionMotivoFuenteAnalisis(
				ctx,
				solicitudMotivo,
			)
		if errPublicador != nil {
			return EvidenciaValidacionRCVerificadaO3{},
				errorDisponibilidadFuente(
					ErrVerificacionFuenteAnalisisNoDisponible,
					errPublicador,
				)
		}
		datosMotivo, errDatos := obtenida.Datos()
		if errDatos != nil ||
			obtenida.ValidarPara(
				solicitudMotivo,
				datosMotivo.VerificadaEn,
			) != nil {
			return EvidenciaValidacionRCVerificadaO3{},
				errorDisponibilidadFuente(
					ErrVerificacionFuenteAnalisisNoDisponible,
					errDatos,
				)
		}
		confirmacionMotivo = &obtenida
	}
	return NuevaEvidenciaValidacionRCVerificadaO3(
		solicitud,
		resultado,
		confirmacion,
		confirmacionMotivo,
		confirmacionFuente,
		confirmacionVerificador,
		confirmacionPublicador,
		confianza,
		reloj.Ahora(),
	)
}

func verificarCalculoCosteConFuenteO3Prueba(
	ctx context.Context,
	calculador CalculadorCostePersonal,
	verificador VerificadorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudCalcularCoste,
) (EvidenciaCalculoCosteVerificadaO3, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(calculador) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(reloj) {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrPeticionFuenteAnalisisInvalida
	}
	material, err := MaterialAutoridadesCalculoCosteO3(solicitud, confianza)
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{}, err
	}
	confirmacionFuente, err :=
		presentarConfirmacionAutoridadFuenteAnalisisPrueba(
			ctx,
			calculador,
			confianza,
			material,
			RolCalculadorCoste,
			reloj.Ahora(),
		)
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{}, err
	}
	confirmacionVerificador, err :=
		presentarConfirmacionAutoridadFuenteAnalisisPrueba(
			ctx,
			verificador,
			confianza,
			material,
			RolVerificadorRespuesta,
			reloj.Ahora(),
		)
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{}, err
	}
	comprobadaAutoridad := reloj.Ahora()
	vinculoFuente, errFuente := confirmacionFuente.VinculoPara(
		material,
		RolCalculadorCoste,
		comprobadaAutoridad,
	)
	vinculoVerificador, errVerificador :=
		confirmacionVerificador.VinculoPara(
			material,
			RolVerificadorRespuesta,
			comprobadaAutoridad,
		)
	if errFuente != nil || errVerificador != nil ||
		!VinculosAutoridadFuenteAnalisisO3Separados(
			vinculoFuente,
			vinculoVerificador,
		) {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := calculador.CalcularCoste(ctx, solicitud)
	if errContexto := ctx.Err(); errContexto != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuente(
				ErrCalculadorCosteNoDisponible,
				errContexto,
			)
	}
	if errFuente != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuente(
				ErrCalculadorCosteNoDisponible,
				errFuente,
			)
	}
	solicitudVerificacion, err := resultado.SolicitudVerificacion()
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{}, err
	}
	confirmacion, errVerificacion :=
		verificador.VerificarRespuestaFuenteAnalisis(
			ctx,
			solicitudVerificacion,
		)
	if errVerificacion != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errVerificacion,
			)
	}
	return NuevaEvidenciaCalculoCosteVerificadaO3(
		solicitud,
		resultado,
		confirmacion,
		confirmacionFuente,
		confirmacionVerificador,
		confianza,
		reloj.Ahora(),
	)
}

func consumirRespuestaFuenteAnalisisPrueba(
	ctx context.Context,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	recibo, err := consumidor.ConsumirRespuestaFuenteAnalisis(ctx, orden)
	if err != nil {
		if errors.Is(err, ErrRespuestaFuenteAnalisisYaConsumida) {
			return ReciboConsumoRespuestaFuenteAnalisis{},
				ErrRespuestaFuenteAnalisisYaConsumida
		}
		return ReciboConsumoRespuestaFuenteAnalisis{},
			errorDisponibilidadFuente(
				ErrConsumoFuenteAnalisisNoDisponible,
				err,
			)
	}
	if recibo.ValidarPara(orden) != nil {
		return ReciboConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return recibo, nil
}

func validarRCConFuenteOrquestadaPrueba(
	ctx context.Context,
	fuente FuentePresupuestaria,
	verificador VerificadorRespuestaFuenteAnalisis,
	publicador VerificadorPublicacionMotivoFuenteAnalisis,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudValidarRC,
) (domain.ValidacionRC, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(consumidor) {
		return domain.ValidacionRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	evidencia, err := verificarValidacionRCConFuenteO3Prueba(
		operacion,
		fuente,
		verificador,
		publicador,
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
	if _, err := consumirRespuestaFuenteAnalisisPrueba(
		operacion,
		consumidor,
		orden,
	); err != nil {
		return domain.ValidacionRC{}, err
	}
	return evidencia.Materializar()
}

func calcularCosteConFuenteOrquestadoPrueba(
	ctx context.Context,
	calculador CalculadorCostePersonal,
	verificador VerificadorRespuestaFuenteAnalisis,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudCalcularCoste,
) (ResultadoCalculoCoste, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(consumidor) {
		return ResultadoCalculoCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	evidencia, err := verificarCalculoCosteConFuenteO3Prueba(
		operacion,
		calculador,
		verificador,
		confianza,
		reloj,
		solicitud,
	)
	if err != nil {
		return ResultadoCalculoCoste{}, err
	}
	orden, err := evidencia.OrdenConsumo()
	if err != nil {
		return ResultadoCalculoCoste{}, err
	}
	if _, err := consumirRespuestaFuenteAnalisisPrueba(
		operacion,
		consumidor,
		orden,
	); err != nil {
		return ResultadoCalculoCoste{}, err
	}
	return evidencia.Resultado()
}

func presentarYVerificarAutoridadFuenteAnalisisPrueba(
	ctx context.Context,
	presentador PresentadorAutoridadFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	material []byte,
	rol RolAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) (identidadAutoridadFuenteAnalisis, error) {
	desafio, err := nuevoDesafioAutoridadFuenteAnalisis(
		material,
		confianza.organizacionRef,
		confianza.audiencia,
		rol,
	)
	if err != nil {
		return identidadAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	presentacion, errPresentacion :=
		presentador.PresentarAutoridadFuenteAnalisis(ctx, desafio)
	if errContexto := ctx.Err(); errContexto != nil {
		return identidadAutoridadFuenteAnalisis{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errContexto,
			)
	}
	identidad, errVerificacion := confianza.verificarPresentacion(
		presentacion,
		desafio,
		rol,
		comprobadaEn,
	)
	if errPresentacion != nil || errVerificacion != nil {
		return identidadAutoridadFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return identidad, nil
}

func autoridadesFuenteAnalisisSeparadasPrueba(
	identidades ...identidadAutoridadFuenteAnalisis,
) bool {
	for indice, primera := range identidades {
		if primera.autoridadRef == "" || primera.backendRef == "" ||
			len(primera.clavePrueba) != ed25519.PublicKeySize {
			return false
		}
		for _, segunda := range identidades[indice+1:] {
			if primera.autoridadRef == segunda.autoridadRef ||
				primera.backendRef == segunda.backendRef ||
				bytes.Equal(primera.clavePrueba, segunda.clavePrueba) {
				return false
			}
		}
	}
	return true
}

func nuevaSolicitudValidarRCOrquestadaPrueba(
	ctx context.Context,
	generador GeneradorPeticionFuenteAnalisis,
	sellador SelladorPeticionFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	preparacion PreparacionSolicitudValidarRC,
) (SolicitudValidarRC, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(generador) ||
		dependenciaNulaFuenteAnalisis(sellador) ||
		dependenciaNulaFuenteAnalisis(reloj) {
		return SolicitudValidarRC{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	peticionRef, errGenerador :=
		generador.NuevaReferenciaPeticionFuenteAnalisis(
			operacion,
			TipoPeticionValidacionRC,
		)
	if errContexto := operacion.Err(); errContexto != nil {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	preparada, err := NuevaPreparacionSelladoSolicitudValidarRC(
		preparacion,
		peticionRef,
		reloj.Ahora(),
	)
	if errGenerador != nil || err != nil {
		return SolicitudValidarRC{},
			ErrPeticionFuenteAnalisisInvalida
	}
	preimagen, _ := preparada.Preimagen()
	sello, errSellador := sellador.SellarPeticionFuenteAnalisis(
		operacion,
		preimagen,
	)
	if errContexto := operacion.Err(); errContexto != nil {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	solicitud, err := preparada.Completar(sello)
	if errSellador != nil || err != nil {
		return SolicitudValidarRC{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errSellador,
		)
	}
	return solicitud, nil
}

func nuevaSolicitudCalcularCosteOrquestadaPrueba(
	ctx context.Context,
	generador GeneradorPeticionFuenteAnalisis,
	sellador SelladorPeticionFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	preparacion PreparacionSolicitudCalcularCoste,
) (SolicitudCalcularCoste, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(generador) ||
		dependenciaNulaFuenteAnalisis(sellador) ||
		dependenciaNulaFuenteAnalisis(reloj) {
		return SolicitudCalcularCoste{}, ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	peticionRef, errGenerador :=
		generador.NuevaReferenciaPeticionFuenteAnalisis(
			operacion,
			TipoPeticionCalculoCoste,
		)
	if errContexto := operacion.Err(); errContexto != nil {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	preparada, err := NuevaPreparacionSelladoSolicitudCalcularCoste(
		preparacion,
		peticionRef,
		reloj.Ahora(),
	)
	if errGenerador != nil || err != nil {
		return SolicitudCalcularCoste{},
			ErrPeticionFuenteAnalisisInvalida
	}
	preimagen, _ := preparada.Preimagen()
	sello, errSellador := sellador.SellarPeticionFuenteAnalisis(
		operacion,
		preimagen,
	)
	if errContexto := operacion.Err(); errContexto != nil {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errContexto,
		)
	}
	solicitud, err := preparada.Completar(sello)
	if errSellador != nil || err != nil {
		return SolicitudCalcularCoste{}, errorDisponibilidadFuente(
			ErrInfraestructuraFuenteAnalisisNoDisponible,
			errSellador,
		)
	}
	return solicitud, nil
}
