package ports

import (
	"context"
	"errors"
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
	identidadFuente      identidadAutoridadFuenteAnalisis
	identidadVerificador identidadAutoridadFuenteAnalisis
	identidadPublicador  identidadAutoridadFuenteAnalisis
}

type EvidenciaCalculoCosteVerificadaO3 struct {
	datos *datosEvidenciaCalculoCosteVerificadaO3
}

type datosEvidenciaCalculoCosteVerificadaO3 struct {
	solicitud            SolicitudCalcularCoste
	resultado            ResultadoCalculoCoste
	confirmacion         ConfirmacionRespuestaFuenteAnalisis
	orden                OrdenConsumoRespuestaFuenteAnalisis
	identidadFuente      identidadAutoridadFuenteAnalisis
	identidadVerificador identidadAutoridadFuenteAnalisis
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
		!identidadAutoridadFuenteAnalisisValida(
			e.datos.identidadFuente,
			RolFuentePresupuestaria,
		) ||
		!identidadAutoridadFuenteAnalisisValida(
			e.datos.identidadVerificador,
			RolVerificadorRespuesta,
		) ||
		!identidadAutoridadFuenteAnalisisValida(
			e.datos.identidadPublicador,
			RolPublicadorCatalogo,
		) ||
		!identidadAutoridadFuenteAnalisisVigenteEn(
			e.datos.identidadFuente,
			comprobadaEn,
		) ||
		!identidadAutoridadFuenteAnalisisVigenteEn(
			e.datos.identidadVerificador,
			comprobadaEn,
		) ||
		!identidadAutoridadFuenteAnalisisVigenteEn(
			e.datos.identidadPublicador,
			comprobadaEn,
		) ||
		!autoridadesFuenteAnalisisSeparadas(
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
			e.datos.identidadFuente.autoridadRef ||
		confirmacion.VerificadorRef !=
			e.datos.identidadVerificador.autoridadRef ||
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
			e.datos.identidadPublicador.autoridadRef {
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
		!identidadAutoridadFuenteAnalisisValida(
			e.datos.identidadFuente,
			RolCalculadorCoste,
		) ||
		!identidadAutoridadFuenteAnalisisValida(
			e.datos.identidadVerificador,
			RolVerificadorRespuesta,
		) ||
		!identidadAutoridadFuenteAnalisisVigenteEn(
			e.datos.identidadFuente,
			comprobadaEn,
		) ||
		!identidadAutoridadFuenteAnalisisVigenteEn(
			e.datos.identidadVerificador,
			comprobadaEn,
		) ||
		!autoridadesFuenteAnalisisSeparadas(
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
			e.datos.identidadFuente.autoridadRef ||
		confirmacion.VerificadorRef !=
			e.datos.identidadVerificador.autoridadRef ||
		orden.HuellaRespuestaSHA256 !=
			resultado.HuellaRespuestaSHA256 {
		return ErrResultadoFuenteAnalisisNoConfiable
	}
	return nil
}

func consumirEvidenciaFuenteAnalisisO3(
	ctx context.Context,
	consumidor ConsumidorRespuestaFuenteAnalisis,
	orden OrdenConsumoRespuestaFuenteAnalisis,
) (ReciboConsumoRespuestaFuenteAnalisis, error) {
	if ctx == nil || dependenciaNulaFuenteAnalisis(consumidor) {
		return ReciboConsumoRespuestaFuenteAnalisis{},
			ErrPeticionFuenteAnalisisInvalida
	}
	if err := ctx.Err(); err != nil {
		return ReciboConsumoRespuestaFuenteAnalisis{},
			errorDisponibilidadFuente(
				ErrConsumoFuenteAnalisisNoDisponible,
				err,
			)
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	recibo, err := consumidor.ConsumirRespuestaFuenteAnalisis(
		operacion,
		orden,
	)
	if err != nil {
		if errContexto := operacion.Err(); errContexto != nil {
			return ReciboConsumoRespuestaFuenteAnalisis{},
				errorDisponibilidadFuente(
					ErrConsumoFuenteAnalisisNoDisponible,
					errContexto,
				)
		}
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

func identidadAutoridadFuenteAnalisisValida(
	identidad identidadAutoridadFuenteAnalisis,
	rol RolAutoridadFuenteAnalisis,
) bool {
	return identidad.rol == rol &&
		domain.ReferenciaOpacaValida(identidad.raizClaveID) &&
		domain.ReferenciaOpacaValida(identidad.autoridadRef) &&
		domain.ReferenciaOpacaValida(identidad.backendRef) &&
		identidad.serie > 0 &&
		identidad.serie <= maximoEnteroSeguroFuenteAnalisis &&
		identidad.generacion > 0 &&
		huellaSHA256FuenteAnalisisValida(
			identidad.huellaClavePruebaSHA256,
		) &&
		instanteFuenteAnalisisCanonico(
			identidad.credencialEmitidaEn,
		) &&
		instanteFuenteAnalisisCanonico(
			identidad.credencialValidaHasta,
		) &&
		identidad.credencialValidaHasta.After(
			identidad.credencialEmitidaEn,
		)
}

func identidadAutoridadFuenteAnalisisVigenteEn(
	identidad identidadAutoridadFuenteAnalisis,
	comprobadaEn time.Time,
) bool {
	return instanteFuenteAnalisisCanonico(comprobadaEn) &&
		!comprobadaEn.Before(identidad.credencialEmitidaEn) &&
		comprobadaEn.Before(identidad.credencialValidaHasta)
}

func vinculoAutoridadFuenteAnalisisO3(
	identidad identidadAutoridadFuenteAnalisis,
) VinculoAutoridadFuenteAnalisisO3 {
	return VinculoAutoridadFuenteAnalisisO3{
		RaizClaveID:           identidad.raizClaveID,
		AutoridadRef:          identidad.autoridadRef,
		BackendRef:            identidad.backendRef,
		Rol:                   identidad.rol,
		Serie:                 identidad.serie,
		Generacion:            identidad.generacion,
		HuellaClaveSHA256:     identidad.huellaClavePruebaSHA256,
		CredencialEmitidaEn:   identidad.credencialEmitidaEn,
		CredencialValidaHasta: identidad.credencialValidaHasta,
	}
}

func datosOrdenConsumo(
	orden OrdenConsumoRespuestaFuenteAnalisis,
) DatosOrdenConsumoRespuestaFuenteAnalisis {
	datos, _ := orden.Datos()
	return datos
}
