package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func NuevaSolicitudValidarRC(
	ctx context.Context,
	generador ports.GeneradorPeticionFuenteAnalisis,
	sellador ports.SelladorPeticionFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	preparacion ports.PreparacionSolicitudValidarRC,
) (ports.SolicitudValidarRC, error) {
	if ctx == nil || dependenciaNula(generador) ||
		dependenciaNula(sellador) || dependenciaNula(reloj) ||
		preparacion.Validar() != nil {
		return ports.SolicitudValidarRC{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	solicitadaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ports.SolicitudValidarRC{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				err,
			)
	}
	peticionRef, errGenerador :=
		generador.NuevaReferenciaPeticionFuenteAnalisis(
			operacion,
			ports.TipoPeticionValidacionRC,
		)
	if err := operacion.Err(); err != nil {
		return ports.SolicitudValidarRC{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				err,
			)
	}
	preparada, err := ports.NuevaPreparacionSelladoSolicitudValidarRC(
		preparacion,
		peticionRef,
		solicitadaEn,
	)
	if errGenerador != nil || err != nil {
		return ports.SolicitudValidarRC{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				errGenerador,
			)
	}
	preimagen, err := preparada.Preimagen()
	if err != nil {
		return ports.SolicitudValidarRC{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	sello, errSellador := sellador.SellarPeticionFuenteAnalisis(
		operacion,
		preimagen,
	)
	if err := operacion.Err(); err != nil {
		return ports.SolicitudValidarRC{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				err,
			)
	}
	solicitud, err := preparada.Completar(sello)
	if errSellador != nil || err != nil {
		return ports.SolicitudValidarRC{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				errSellador,
			)
	}
	return solicitud, nil
}

func NuevaSolicitudCalcularCoste(
	ctx context.Context,
	generador ports.GeneradorPeticionFuenteAnalisis,
	sellador ports.SelladorPeticionFuenteAnalisis,
	reloj ports.RelojFuenteAnalisis,
	preparacion ports.PreparacionSolicitudCalcularCoste,
) (ports.SolicitudCalcularCoste, error) {
	if ctx == nil || dependenciaNula(generador) ||
		dependenciaNula(sellador) || dependenciaNula(reloj) ||
		preparacion.Validar() != nil {
		return ports.SolicitudCalcularCoste{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	operacion, cancelar := context.WithTimeout(
		ctx,
		ports.TiempoMaximoFuenteAnalisis,
	)
	defer cancelar()
	solicitadaEn := reloj.Ahora()
	if err := operacion.Err(); err != nil {
		return ports.SolicitudCalcularCoste{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				err,
			)
	}
	peticionRef, errGenerador :=
		generador.NuevaReferenciaPeticionFuenteAnalisis(
			operacion,
			ports.TipoPeticionCalculoCoste,
		)
	if err := operacion.Err(); err != nil {
		return ports.SolicitudCalcularCoste{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				err,
			)
	}
	preparada, err :=
		ports.NuevaPreparacionSelladoSolicitudCalcularCoste(
			preparacion,
			peticionRef,
			solicitadaEn,
		)
	if errGenerador != nil || err != nil {
		return ports.SolicitudCalcularCoste{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				errGenerador,
			)
	}
	preimagen, err := preparada.Preimagen()
	if err != nil {
		return ports.SolicitudCalcularCoste{},
			ports.ErrPeticionFuenteAnalisisInvalida
	}
	sello, errSellador := sellador.SellarPeticionFuenteAnalisis(
		operacion,
		preimagen,
	)
	if err := operacion.Err(); err != nil {
		return ports.SolicitudCalcularCoste{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				err,
			)
	}
	solicitud, err := preparada.Completar(sello)
	if errSellador != nil || err != nil {
		return ports.SolicitudCalcularCoste{},
			errorDisponibilidadFuenteAplicacion(
				ports.ErrInfraestructuraFuenteAnalisisNoDisponible,
				errSellador,
			)
	}
	return solicitud, nil
}
