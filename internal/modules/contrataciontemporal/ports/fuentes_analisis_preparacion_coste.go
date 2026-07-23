package ports

import "context"

// VerificarCalculoCosteConFuenteO3 conserva el resultado, la confirmación TCB
// y la orden de consumo O3-03 únicamente después de verificar las autoridades
// criptográficas separadas. Todavía no consume la respuesta.
func VerificarCalculoCosteConFuenteO3(
	ctx context.Context,
	calculador CalculadorCostePersonal,
	verificador VerificadorRespuestaFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudCalcularCoste,
) (EvidenciaCalculoCosteVerificadaO3, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(calculador) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(reloj) ||
		solicitud.Validar() != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	if err := operacion.Err(); err != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuente(
				ErrCalculadorCosteNoDisponible,
				err,
			)
	}
	datosSolicitud, errDatosSolicitud := solicitud.Datos()
	materialPeticion := materialDesafioSolicitudFuenteAnalisis(
		solicitud.datosCanonicos(),
		datosSolicitud.HuellaPeticionHMAC,
	)
	if errDatosSolicitud != nil ||
		datosSolicitud.OrganizacionRef != confianza.organizacionRef ||
		len(materialPeticion) == 0 {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrPeticionFuenteAnalisisInvalida
	}
	identidadFuente, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion,
		calculador,
		confianza,
		materialPeticion,
		RolCalculadorCoste,
		reloj.Ahora(),
	)
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{}, err
	}
	identidadVerificador, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion,
		verificador,
		confianza,
		materialPeticion,
		RolVerificadorRespuesta,
		reloj.Ahora(),
	)
	if err != nil || !autoridadesFuenteAnalisisSeparadas(
		identidadFuente,
		identidadVerificador,
	) {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := calculador.CalcularCoste(operacion, solicitud)
	if errContexto := operacion.Err(); errContexto != nil {
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
	recibidaEn := reloj.Ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuente(
				ErrCalculadorCosteNoDisponible,
				errContexto,
			)
	}
	if resultado.ValidarPara(solicitud, recibidaEn) != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datosResultado, errDatosResultado := resultado.Datos()
	if errDatosResultado != nil ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			identidadFuente.autoridadRef {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	confirmacion, err := verificarRespuestaFuenteAnalisis(
		operacion,
		verificador,
		identidadVerificador,
		resultado.solicitudVerificacion(),
		reloj,
	)
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{}, err
	}
	orden, err := nuevaOrdenConsumoResultadoCoste(
		solicitud,
		resultado,
		confirmacion,
	)
	if err != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	evidencia := EvidenciaCalculoCosteVerificadaO3{
		datos: &datosEvidenciaCalculoCosteVerificadaO3{
			solicitud:            solicitud,
			resultado:            resultado,
			confirmacion:         confirmacion,
			orden:                orden,
			identidadFuente:      identidadFuente,
			identidadVerificador: identidadVerificador,
		},
	}
	comprobadaEn := reloj.Ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errContexto,
			)
	}
	if evidencia.validarEn(comprobadaEn) != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return evidencia, nil
}
