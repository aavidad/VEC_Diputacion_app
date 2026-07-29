package ports

import (
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// NuevosRecursosConsultaCuadroRRHH construye el recurso autorizable del
// cuadro exclusivamente desde contexto y solicitud ya validados. El llamador
// no puede elegir módulo, tipo, ámbito, dominio ni huella.
func NuevosRecursosConsultaCuadroRRHH(
	contexto ContextoConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	instante time.Time,
) (RecursosConsultaRRHH, error) {
	if contexto.validarEn(instante) != nil {
		return RecursosConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	huella, err := huellaSolicitudCuadroRRHH(solicitud)
	if err != nil {
		return RecursosConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	recursos := RecursosConsultaRRHH{
		recurso: dominiovec.RecursoAutorizable{
			Referencia: contexto.organizacionRef,
			ModuloID:   ModuloContratacion,
			Tipo:       TipoRecursoCuadroRRHH,
			Ambitos: map[string]string{
				ambitoOrganizacionRecursoRRHH: contexto.organizacionRef,
				ambitoClaseRecursoRRHH:        string(AmbitoOrganizacionRRHH),
				ambitoReferenciaRecursoRRHH:   contexto.organizacionRef,
			},
			Atributos: map[string]string{
				atributoDominioConsultaRRHH: DominioHuellaConsultaCuadroRRHH,
				atributoHuellaConsultaRRHH:  huella,
			},
		},
	}
	if recursos.validarParaCuadro(contexto, solicitud, instante) != nil {
		return RecursosConsultaRRHH{}, ErrCapacidadConsultaRRHHInvalida
	}
	return recursos, nil
}

func (r RecursosConsultaRRHH) validarParaCuadro(
	contexto ContextoConsultaRRHH,
	solicitud SolicitudCuadroRRHH,
	instante time.Time,
) error {
	if r.validarEstructura() != nil || contexto.validarEn(instante) != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	huella, err := huellaSolicitudCuadroRRHH(solicitud)
	if err != nil {
		return ErrCapacidadConsultaRRHHInvalida
	}
	clase, ambitoRef, err := validarRecursoCapacidadConsultaRRHH(
		r.recurso,
		contexto,
		DominioHuellaConsultaCuadroRRHH,
		huella,
		AccionConsultarCuadroRRHH,
		"",
	)
	if err != nil || clase != AmbitoOrganizacionRRHH ||
		ambitoRef != contexto.organizacionRef {
		return ErrCapacidadConsultaRRHHInvalida
	}
	return nil
}
