package ports

import (
	"context"
	"errors"
	"reflect"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var errEmisoresMaterialConsultaRRHHNoDisponibles = errors.New(
	"contratacion temporal: emisores de material de consulta RRHH no disponibles",
)

// emisorMaterialAutorizacionAtestadaV3 conserva únicamente la forma
// estructural de la fachada VEC A2.2. No permite deducir configuración,
// audiencia ni autoridad a partir de la operación que lo consumirá.
type emisorMaterialAutorizacionAtestadaV3 interface {
	EmitirMaterialAutorizacionAtestadaV3(
		context.Context,
		dominiovec.SolicitudAutorizacionLigadaV3,
		dominiovec.ResultadoContextoActorRegistradoV2,
	) (
		dominiovec.DecisionAutorizacionLigadaV3,
		puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3,
		puertosvec.ExportadorMaterialConsumoAutorizacionAtestadaV3,
		error,
	)
}

// Los envoltorios nominales impiden intercambiar por asignación directa los
// emisores de cuadro y detalle dentro del propietario privado.
type emisorMaterialCuadroRRHH struct {
	emisor emisorMaterialAutorizacionAtestadaV3
}

type emisorMaterialDetalleRRHH struct {
	emisor emisorMaterialAutorizacionAtestadaV3
}

type emisoresMaterialConsultaRRHH struct {
	cuadro  emisorMaterialCuadroRRHH
	detalle emisorMaterialDetalleRRHH
}

func nuevosEmisoresMaterialConsultaRRHH(
	cuadro emisorMaterialAutorizacionAtestadaV3,
	detalle emisorMaterialAutorizacionAtestadaV3,
) (emisoresMaterialConsultaRRHH, error) {
	tipoCuadro, punteroCuadro, cuadroValido :=
		identidadFisicaEmisorMaterialConsultaRRHH(cuadro)
	tipoDetalle, punteroDetalle, detalleValido :=
		identidadFisicaEmisorMaterialConsultaRRHH(detalle)
	if !cuadroValido || !detalleValido ||
		tipoCuadro == tipoDetalle && punteroCuadro == punteroDetalle {
		return emisoresMaterialConsultaRRHH{},
			errEmisoresMaterialConsultaRRHHNoDisponibles
	}
	return emisoresMaterialConsultaRRHH{
		cuadro:  emisorMaterialCuadroRRHH{emisor: cuadro},
		detalle: emisorMaterialDetalleRRHH{emisor: detalle},
	}, nil
}

// identidadFisicaEmisorMaterialConsultaRRHH admite exclusivamente
// implementaciones por puntero. Una implementación por valor no posee una
// identidad física estable que permita demostrar la segregación y se rechaza
// de forma cerrada.
func identidadFisicaEmisorMaterialConsultaRRHH(
	emisor emisorMaterialAutorizacionAtestadaV3,
) (reflect.Type, uintptr, bool) {
	if emisor == nil {
		return nil, 0, false
	}
	valor := reflect.ValueOf(emisor)
	if valor.Kind() != reflect.Pointer || valor.IsNil() {
		return nil, 0, false
	}
	return valor.Type(), valor.Pointer(), true
}
