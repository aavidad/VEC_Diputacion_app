package ports

import (
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// NuevosRecursosConsultaDetalleRRHH construye el recurso autorizable del
// detalle exclusivamente desde contexto y solicitud ya validados. El llamador
// no puede elegir módulo, tipo, ámbito, dominio ni huella.
func NuevosRecursosConsultaDetalleRRHH(
	contexto ContextoConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
	instante time.Time,
) (RecursosConsultaRRHH, error) {
	if contexto.validarEn(instante) != nil {
		return RecursosConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	huella, err := huellaSolicitudDetalleRRHH(solicitud)
	if err != nil {
		return RecursosConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	recursos := RecursosConsultaRRHH{
		recurso: dominiovec.RecursoAutorizable{
			Referencia: solicitud.expedienteRef,
			ModuloID:   ModuloContratacion,
			Tipo:       TipoRecursoExpediente,
			Ambitos: map[string]string{
				ambitoOrganizacionRecursoRRHH: contexto.organizacionRef,
				ambitoClaseRecursoRRHH:        string(AmbitoOrganizacionRRHH),
				ambitoReferenciaRecursoRRHH:   contexto.organizacionRef,
			},
			Atributos: map[string]string{
				atributoDominioConsultaRRHH: DominioHuellaConsultaDetalleRRHH,
				atributoHuellaConsultaRRHH:  huella,
			},
		},
	}
	if recursos.validarParaDetalle(contexto, solicitud, instante) != nil {
		return RecursosConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return recursos, nil
}

func (r RecursosConsultaRRHH) validarParaDetalle(
	contexto ContextoConsultaRRHH,
	solicitud SolicitudDetalleRRHH,
	instante time.Time,
) error {
	if r.validarEstructura() != nil || contexto.validarEn(instante) != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	huella, err := huellaSolicitudDetalleRRHH(solicitud)
	if err != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	clase, ambitoRef, err := validarRecursoCapacidadConsultaRRHH(
		r.recurso,
		contexto,
		DominioHuellaConsultaDetalleRRHH,
		huella,
		AccionConsultarDetalleRRHH,
		solicitud.expedienteRef,
	)
	if err != nil || clase != AmbitoOrganizacionRRHH ||
		ambitoRef != contexto.organizacionRef {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}
