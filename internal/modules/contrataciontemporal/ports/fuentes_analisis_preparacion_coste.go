package ports

import "time"

func MaterialAutoridadesCalculoCosteO3(
	solicitud SolicitudCalcularCoste,
	confianza ConfianzaAutoridadesFuenteAnalisis,
) ([]byte, error) {
	datos, err := solicitud.Datos()
	material := materialDesafioSolicitudFuenteAnalisis(
		solicitud.datosCanonicos(),
		datos.HuellaPeticionHMAC,
	)
	if err != nil || solicitud.Validar() != nil ||
		confianza.Validar() != nil ||
		datos.OrganizacionRef != confianza.organizacionRef ||
		len(material) == 0 {
		return nil, ErrPeticionFuenteAnalisisInvalida
	}
	return append([]byte(nil), material...), nil
}

func (r ResultadoCalculoCoste) SolicitudVerificacion() (
	SolicitudVerificarRespuestaFuenteAnalisis,
	error,
) {
	if _, err := r.Datos(); err != nil {
		return SolicitudVerificarRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return r.solicitudVerificacion(), nil
}

func NuevaEvidenciaCalculoCosteVerificadaO3(
	solicitud SolicitudCalcularCoste,
	resultado ResultadoCalculoCoste,
	confirmacion ConfirmacionRespuestaFuenteAnalisis,
	fuente ConfirmacionComprobacionAutoridadFuenteAnalisis,
	verificador ConfirmacionComprobacionAutoridadFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	comprobadaEn time.Time,
) (EvidenciaCalculoCosteVerificadaO3, error) {
	material, err := MaterialAutoridadesCalculoCosteO3(
		solicitud,
		confianza,
	)
	if err != nil || resultado.ValidarPara(solicitud, comprobadaEn) != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	vinculoFuente, errFuente := fuente.validarPara(
		material,
		RolCalculadorCoste,
		comprobadaEn,
	)
	vinculoVerificador, errVerificador := verificador.validarPara(
		material,
		RolVerificadorRespuesta,
		comprobadaEn,
	)
	solicitudVerificacion, errSolicitud := resultado.SolicitudVerificacion()
	datosResultado, errResultado := resultado.Datos()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	if errFuente != nil || errVerificador != nil ||
		errSolicitud != nil || errResultado != nil ||
		errConfirmacion != nil ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			vinculoFuente,
			vinculoVerificador,
		) ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			vinculoFuente.AutoridadRef ||
		datosConfirmacion.VerificadorRef !=
			vinculoVerificador.AutoridadRef ||
		confirmacion.ValidarPara(
			solicitudVerificacion,
			comprobadaEn,
		) != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
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
			identidadFuente:      vinculoFuente,
			identidadVerificador: vinculoVerificador,
		},
	}
	if evidencia.validarEn(comprobadaEn) != nil {
		return EvidenciaCalculoCosteVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return evidencia, nil
}

func (e EvidenciaCalculoCosteVerificadaO3) OrdenConsumo() (
	OrdenConsumoRespuestaFuenteAnalisis,
	error,
) {
	if e.datos == nil {
		return OrdenConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return e.datos.orden, nil
}

func (e EvidenciaCalculoCosteVerificadaO3) Resultado() (
	ResultadoCalculoCoste,
	error,
) {
	if e.datos == nil {
		return ResultadoCalculoCoste{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	if _, err := e.datos.resultado.Datos(); err != nil {
		return ResultadoCalculoCoste{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return e.datos.resultado.clonar(), nil
}
