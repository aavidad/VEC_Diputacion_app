package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// MaterialAutoridadesValidacionRCO3 prepara una copia del material local que
// application presenta a las autoridades. No invoca fuentes ni adaptadores.
func MaterialAutoridadesValidacionRCO3(
	solicitud SolicitudValidarRC,
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

// SolicitudVerificacion devuelve una copia de la petición local ligada al
// resultado atestado. No contacta al verificador.
func (r ResultadoValidacionRC) SolicitudVerificacion() (
	SolicitudVerificarRespuestaFuenteAnalisis,
	error,
) {
	if _, err := r.Datos(); err != nil {
		return SolicitudVerificarRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return r.solicitudVerificacion(), nil
}

// SolicitudPublicacionMotivo devuelve la comprobación local necesaria para un
// rechazo RC. Una validación satisfactoria no requiere publicación.
func (r ResultadoValidacionRC) SolicitudPublicacionMotivo() (
	SolicitudVerificarPublicacionMotivoFuenteAnalisis,
	bool,
	error,
) {
	datos, err := r.Datos()
	if err != nil {
		return SolicitudVerificarPublicacionMotivoFuenteAnalisis{},
			false,
			ErrResultadoFuenteAnalisisNoConfiable
	}
	if datos.Validacion.Resultado == domain.RCValidada {
		if datos.Motivo.datos != nil {
			return SolicitudVerificarPublicacionMotivoFuenteAnalisis{},
				false,
				ErrResultadoFuenteAnalisisNoConfiable
		}
		return SolicitudVerificarPublicacionMotivoFuenteAnalisis{},
			false,
			nil
	}
	solicitud := SolicitudVerificarPublicacionMotivoFuenteAnalisis{
		Motivo:                datos.Motivo,
		HuellaRespuestaSHA256: datos.HuellaRespuestaSHA256,
		AutoridadRespuestaRef: datos.Atestacion.Metadatos.AutoridadRef,
		GeneracionRespuesta:   datos.Atestacion.Metadatos.Generacion,
	}
	if solicitud.Validar() != nil {
		return SolicitudVerificarPublicacionMotivoFuenteAnalisis{},
			false,
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return solicitud, true, nil
}

// NuevaEvidenciaValidacionRCVerificadaO3 ensambla y contrasta localmente las
// respuestas que application ya obtuvo. Las confirmaciones de autoridad son
// opacas y no permiten reconstruir claves ni credenciales.
func NuevaEvidenciaValidacionRCVerificadaO3(
	solicitud SolicitudValidarRC,
	resultado ResultadoValidacionRC,
	confirmacion ConfirmacionRespuestaFuenteAnalisis,
	confirmacionMotivo *ConfirmacionPublicacionMotivoFuenteAnalisis,
	fuente ConfirmacionComprobacionAutoridadFuenteAnalisis,
	verificador ConfirmacionComprobacionAutoridadFuenteAnalisis,
	publicador ConfirmacionComprobacionAutoridadFuenteAnalisis,
	confianza ConfianzaAutoridadesFuenteAnalisis,
	comprobadaEn time.Time,
) (EvidenciaValidacionRCVerificadaO3, error) {
	material, err := MaterialAutoridadesValidacionRCO3(
		solicitud,
		confianza,
	)
	if err != nil || resultado.ValidarPara(solicitud, comprobadaEn) != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	vinculoFuente, errFuente := fuente.validarPara(
		material,
		RolFuentePresupuestaria,
		comprobadaEn,
	)
	vinculoVerificador, errVerificador := verificador.validarPara(
		material,
		RolVerificadorRespuesta,
		comprobadaEn,
	)
	vinculoPublicador, errPublicador := publicador.validarPara(
		material,
		RolPublicadorCatalogo,
		comprobadaEn,
	)
	solicitudVerificacion, errSolicitud := resultado.SolicitudVerificacion()
	datosResultado, errResultado := resultado.Datos()
	datosConfirmacion, errConfirmacion := confirmacion.Datos()
	if errFuente != nil || errVerificador != nil ||
		errPublicador != nil || errSolicitud != nil ||
		errResultado != nil || errConfirmacion != nil ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			vinculoFuente,
			vinculoVerificador,
			vinculoPublicador,
		) ||
		datosResultado.Atestacion.Metadatos.AutoridadRef !=
			vinculoFuente.AutoridadRef ||
		datosConfirmacion.VerificadorRef !=
			vinculoVerificador.AutoridadRef ||
		confirmacion.ValidarPara(
			solicitudVerificacion,
			comprobadaEn,
		) != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	solicitudMotivo, requiereMotivo, errMotivo :=
		resultado.SolicitudPublicacionMotivo()
	if errMotivo != nil ||
		(requiereMotivo && confirmacionMotivo == nil) ||
		(!requiereMotivo && confirmacionMotivo != nil) {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	if requiereMotivo {
		datosMotivo, errDatosMotivo := confirmacionMotivo.Datos()
		if errDatosMotivo != nil ||
			datosMotivo.PublicadorRef != vinculoPublicador.AutoridadRef ||
			confirmacionMotivo.ValidarPara(
				solicitudMotivo,
				comprobadaEn,
			) != nil {
			return EvidenciaValidacionRCVerificadaO3{},
				ErrResultadoFuenteAnalisisNoConfiable
		}
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
			identidadFuente:      vinculoFuente,
			identidadVerificador: vinculoVerificador,
			identidadPublicador:  vinculoPublicador,
		},
	}
	if evidencia.validarEn(comprobadaEn) != nil {
		return EvidenciaValidacionRCVerificadaO3{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return evidencia, nil
}

func (e EvidenciaValidacionRCVerificadaO3) OrdenConsumo() (
	OrdenConsumoRespuestaFuenteAnalisis,
	error,
) {
	if e.datos == nil || e.validarEn(e.datos.confirmacionTiempo()) != nil {
		return OrdenConsumoRespuestaFuenteAnalisis{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	return e.datos.orden, nil
}

func (d datosEvidenciaValidacionRCVerificadaO3) confirmacionTiempo() time.Time {
	datos, _ := d.confirmacion.Datos()
	return datos.VerificadaEn
}

func (e EvidenciaValidacionRCVerificadaO3) Materializar() (
	domain.ValidacionRC,
	error,
) {
	if e.datos == nil {
		return domain.ValidacionRC{},
			ErrResultadoFuenteAnalisisNoConfiable
	}
	datos, err := e.datos.resultado.Datos()
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
