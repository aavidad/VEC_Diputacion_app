package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// VinculoAutoridadFuenteAnalisisO3 conserva únicamente identificadores y
// huellas ya verificados. La clave pública y la credencial firmada permanecen
// dentro de la capacidad opaca que produjo la evidencia.
type VinculoAutoridadFuenteAnalisisO3 struct {
	RaizClaveID           string
	AutoridadRef          string
	BackendRef            string
	Rol                   RolAutoridadFuenteAnalisis
	Serie                 uint64
	Generacion            uint32
	HuellaClaveSHA256     string
	CredencialEmitidaEn   time.Time
	CredencialValidaHasta time.Time
}

type EvidenciaValidacionRCVerificadaO3 struct {
	datos *datosEvidenciaValidacionRCVerificadaO3
}

type datosEvidenciaValidacionRCVerificadaO3 struct {
	solicitud            SolicitudValidarRC
	resultado            ResultadoValidacionRC
	confirmacion         ConfirmacionRespuestaFuenteAnalisis
	confirmacionMotivo   *ConfirmacionPublicacionMotivoFuenteAnalisis
	orden                OrdenConsumoRespuestaFuenteAnalisis
	identidadFuente      VinculoAutoridadFuenteAnalisisO3
	identidadVerificador VinculoAutoridadFuenteAnalisisO3
	identidadPublicador  VinculoAutoridadFuenteAnalisisO3
}

type EvidenciaCalculoCosteVerificadaO3 struct {
	datos *datosEvidenciaCalculoCosteVerificadaO3
}

type datosEvidenciaCalculoCosteVerificadaO3 struct {
	solicitud            SolicitudCalcularCoste
	resultado            ResultadoCalculoCoste
	confirmacion         ConfirmacionRespuestaFuenteAnalisis
	orden                OrdenConsumoRespuestaFuenteAnalisis
	identidadFuente      VinculoAutoridadFuenteAnalisisO3
	identidadVerificador VinculoAutoridadFuenteAnalisisO3
}

func (e EvidenciaValidacionRCVerificadaO3) ValidarEn(
	comprobadaEn time.Time,
) error {
	return e.validarEn(comprobadaEn)
}

func (e EvidenciaCalculoCosteVerificadaO3) ValidarEn(
	comprobadaEn time.Time,
) error {
	if e.datos == nil {
		return nil
	}
	return e.validarEn(comprobadaEn)
}

func (e EvidenciaValidacionRCVerificadaO3) validarEn(
	comprobadaEn time.Time,
) error {
	if e.datos == nil ||
		e.datos.solicitud.Validar() != nil ||
		e.datos.resultado.ValidarPara(
			e.datos.solicitud,
			comprobadaEn,
		) != nil ||
		e.datos.confirmacion.ValidarPara(
			e.datos.resultado.solicitudVerificacion(),
			comprobadaEn,
		) != nil ||
		validarOrdenConsumoRespuesta(
			datosOrdenConsumo(e.datos.orden),
			e.datos.resultado.solicitudVerificacion(),
		) != nil ||
		!vinculoAutoridadAnalisisValido(
			e.datos.identidadFuente,
			RolFuentePresupuestaria,
		) ||
		!vinculoAutoridadAnalisisValido(
			e.datos.identidadVerificador,
			RolVerificadorRespuesta,
		) ||
		!vinculoAutoridadAnalisisValido(
			e.datos.identidadPublicador,
			RolPublicadorCatalogo,
		) ||
		!vinculoAutoridadFuenteAnalisisO3VigenteEn(
			e.datos.identidadFuente,
			comprobadaEn,
		) ||
		!vinculoAutoridadFuenteAnalisisO3VigenteEn(
			e.datos.identidadVerificador,
			comprobadaEn,
		) ||
		!vinculoAutoridadFuenteAnalisisO3VigenteEn(
			e.datos.identidadPublicador,
			comprobadaEn,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			e.datos.identidadFuente,
			e.datos.identidadVerificador,
			e.datos.identidadPublicador,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errResultado := e.datos.resultado.Datos()
	confirmacion, errConfirmacion := e.datos.confirmacion.Datos()
	orden, errOrden := e.datos.orden.Datos()
	if errResultado != nil || errConfirmacion != nil || errOrden != nil ||
		resultado.Atestacion.Metadatos.AutoridadRef !=
			e.datos.identidadFuente.AutoridadRef ||
		confirmacion.VerificadorRef !=
			e.datos.identidadVerificador.AutoridadRef ||
		orden.HuellaRespuestaSHA256 !=
			resultado.HuellaRespuestaSHA256 {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	if resultado.Validacion.Resultado == domain.RCValidada {
		if e.datos.confirmacionMotivo != nil ||
			orden.ConfirmacionPublicacion != nil {
			return ErrResultadoFuenteAnalisisNoConfiable
		}
		return nil
	}
	if e.datos.confirmacionMotivo == nil ||
		orden.ConfirmacionPublicacion == nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	datosMotivo, errMotivo := e.datos.confirmacionMotivo.Datos()
	if errMotivo != nil ||
		datosMotivo.PublicadorRef !=
			e.datos.identidadPublicador.AutoridadRef {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	solicitudMotivo := SolicitudVerificarPublicacionMotivoFuenteAnalisis{
		Motivo:                resultado.Motivo,
		HuellaRespuestaSHA256: resultado.HuellaRespuestaSHA256,
		AutoridadRespuestaRef: resultado.Atestacion.Metadatos.AutoridadRef,
		GeneracionRespuesta:   resultado.Atestacion.Metadatos.Generacion,
	}
	if e.datos.confirmacionMotivo.ValidarPara(
		solicitudMotivo,
		comprobadaEn,
	) != nil {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func (e EvidenciaCalculoCosteVerificadaO3) validarEn(
	comprobadaEn time.Time,
) error {
	if e.datos == nil ||
		e.datos.solicitud.Validar() != nil ||
		e.datos.resultado.ValidarPara(
			e.datos.solicitud,
			comprobadaEn,
		) != nil ||
		e.datos.confirmacion.ValidarPara(
			e.datos.resultado.solicitudVerificacion(),
			comprobadaEn,
		) != nil ||
		validarOrdenConsumoRespuesta(
			datosOrdenConsumo(e.datos.orden),
			e.datos.resultado.solicitudVerificacion(),
		) != nil ||
		!vinculoAutoridadAnalisisValido(
			e.datos.identidadFuente,
			RolCalculadorCoste,
		) ||
		!vinculoAutoridadAnalisisValido(
			e.datos.identidadVerificador,
			RolVerificadorRespuesta,
		) ||
		!vinculoAutoridadFuenteAnalisisO3VigenteEn(
			e.datos.identidadFuente,
			comprobadaEn,
		) ||
		!vinculoAutoridadFuenteAnalisisO3VigenteEn(
			e.datos.identidadVerificador,
			comprobadaEn,
		) ||
		!vinculosAutoridadFuenteAnalisisO3Separados(
			e.datos.identidadFuente,
			e.datos.identidadVerificador,
		) {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	resultado, errResultado := e.datos.resultado.Datos()
	confirmacion, errConfirmacion := e.datos.confirmacion.Datos()
	orden, errOrden := e.datos.orden.Datos()
	if errResultado != nil || errConfirmacion != nil || errOrden != nil ||
		resultado.Atestacion.Metadatos.AutoridadRef !=
			e.datos.identidadFuente.AutoridadRef ||
		confirmacion.VerificadorRef !=
			e.datos.identidadVerificador.AutoridadRef ||
		orden.HuellaRespuestaSHA256 !=
			resultado.HuellaRespuestaSHA256 {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func vinculoAutoridadFuenteAnalisisO3VigenteEn(
	identidad VinculoAutoridadFuenteAnalisisO3,
	comprobadaEn time.Time,
) bool {
	return instanteFuenteAnalisisCanonico(comprobadaEn) &&
		!comprobadaEn.Before(identidad.CredencialEmitidaEn) &&
		comprobadaEn.Before(identidad.CredencialValidaHasta)
}

func datosOrdenConsumo(
	orden OrdenConsumoRespuestaFuenteAnalisis,
) DatosOrdenConsumoRespuestaFuenteAnalisis {
	datos, _ := orden.Datos()
	return datos
}
