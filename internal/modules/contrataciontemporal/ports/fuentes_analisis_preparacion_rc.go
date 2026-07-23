package ports

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// VerificarValidacionRCConFuenteO3 obtiene una respuesta de la fuente y
// conserva, en una capacidad opaca, el resultado, las confirmaciones y la
// orden de consumo ya contrastados con las autoridades criptográficas O3-03.
// Todavía no consume la respuesta.
func VerificarValidacionRCConFuenteO3(
	ctx context.Context,
	fuente FuentePresupuestaria,
	verificador VerificadorRespuestaFuenteAnalisis,
	publicaciones VerificadorPublicacionMotivoFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	reloj RelojFuenteAnalisis,
	solicitud SolicitudValidarRC,
) (EvidenciaValidacionRCVerificadaO3, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(fuente) ||
		dependenciaNulaFuenteAnalisis(verificador) ||
		dependenciaNulaFuenteAnalisis(publicaciones) ||
		dependenciaNulaFuenteAnalisis(reloj) ||
		solicitud.Validar() != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	if err := operacion.Err(); err != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrFuentePresupuestariaNoDisponible,
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
		return EvidenciaValidacionRCVerificadaO3{},
			ErrPeticionFuenteAnalisisInvalida
	}
	identidadFuente, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion,
		fuente,
		confianza,
		materialPeticion,
		RolFuentePresupuestaria,
		reloj.Ahora(),
	)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	identidadVerificador, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion,
		verificador,
		confianza,
		materialPeticion,
		RolVerificadorRespuesta,
		reloj.Ahora(),
	)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	identidadPublicador, err := presentarYVerificarAutoridadFuenteAnalisis(
		operacion,
		publicaciones,
		confianza,
		materialPeticion,
		RolPublicadorCatalogo,
		reloj.Ahora(),
	)
	if err != nil || !autoridadesFuenteAnalisisSeparadas(
		identidadFuente,
		identidadVerificador,
		identidadPublicador,
	) {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errFuente := fuente.ValidarRC(operacion, solicitud)
	if errContexto := operacion.Err(); errContexto != nil {
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
	recibidaEn := reloj.Ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrFuentePresupuestariaNoDisponible,
				errContexto,
			)
	}
	if resultado.ValidarPara(solicitud, recibidaEn) != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datosResultado, errDatosResultado := resultado.Datos()
	if errDatosResultado != nil ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			identidadFuente.autoridadRef {
		return EvidenciaValidacionRCVerificadaO3{},
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
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	confirmacionMotivo, err := verificarMotivoResultadoRC(
		operacion,
		publicaciones,
		identidadPublicador,
		resultado,
		reloj,
	)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{}, err
	}
	orden, err := nuevaOrdenConsumoResultadoRC(
		solicitud,
		resultado,
		confirmacion,
		confirmacionMotivo,
	)
	if err != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	evidencia := EvidenciaValidacionRCVerificadaO3{
		datos: &datosEvidenciaValidacionRCVerificadaO3{
			solicitud:            solicitud,
			resultado:            resultado,
			confirmacion:         confirmacion,
			confirmacionMotivo:   confirmacionMotivo,
			orden:                orden,
			identidadFuente:      identidadFuente,
			identidadVerificador: identidadVerificador,
			identidadPublicador:  identidadPublicador,
		},
	}
	comprobadaEn := reloj.Ahora()
	if errContexto := operacion.Err(); errContexto != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			errorDisponibilidadFuente(
				ErrVerificacionFuenteAnalisisNoDisponible,
				errContexto,
			)
	}
	if evidencia.validarEn(comprobadaEn) != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return evidencia, nil
}

func materializarValidacionRCEvidenciaO3(
	evidencia EvidenciaValidacionRCVerificadaO3,
) (domain.ValidacionRC, error) {
	if evidencia.datos == nil {
		return domain.ValidacionRC{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datos, err := evidencia.datos.resultado.Datos()
	if err != nil {
		return domain.ValidacionRC{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	validacion, err := materializarMotivoValidacionRC(
		datos.Validacion,
		datos.Motivo,
	)
	if err != nil {
		return domain.ValidacionRC{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return clonarValidacionRC(validacion), nil
}
