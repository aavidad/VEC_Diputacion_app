package domain

// RestriccionesProyeccionAutorizacionV3 es una copia de las restricciones de
// una decisión nominal ligada a la solicitud exacta. Las listas nunca comparten
// memoria con la decisión sellada.
type RestriccionesProyeccionAutorizacionV3 struct {
	CamposPermitidos []string
	Obligaciones     []string
}

func (d DecisionAutorizacionLigadaV3) RestriccionesProyeccionPara(
	solicitud SolicitudAutorizacionLigadaV3,
) (RestriccionesProyeccionAutorizacionV3, error) {
	if d.ValidarPara(solicitud) != nil || d.datos == nil || !d.datos.concedida {
		return RestriccionesProyeccionAutorizacionV3{}, ErrAutorizacionDenegada
	}
	return RestriccionesProyeccionAutorizacionV3{
		CamposPermitidos: append([]string(nil), d.datos.camposPermitidos...),
		Obligaciones:     append([]string(nil), d.datos.obligaciones...),
	}, nil
}

// ExigirProyeccionPara exige exactamente los campos que consume la proyección.
// Falla cerrado ante campos ausentes, futuros, repetidos o sin concesión y
// ante cualquier obligación que el consumidor no soporte.
// La ejecución de las obligaciones sigue siendo responsabilidad del consumidor.
func (d DecisionAutorizacionLigadaV3) ExigirProyeccionPara(
	solicitud SolicitudAutorizacionLigadaV3,
	camposRequeridos []string,
	obligacionesSoportadas []string,
) error {
	restricciones, err := d.RestriccionesProyeccionPara(solicitud)
	if err != nil || len(camposRequeridos) == 0 ||
		len(restricciones.CamposPermitidos) != len(camposRequeridos) ||
		!listaAutorizacionValida(camposRequeridos, false, false) ||
		!listaAutorizacionValida(obligacionesSoportadas, false, false) {
		return ErrAutorizacionDenegada
	}
	permitidos := make(map[string]struct{}, len(restricciones.CamposPermitidos))
	for _, campo := range restricciones.CamposPermitidos {
		permitidos[campo] = struct{}{}
	}
	for _, campo := range camposRequeridos {
		if _, existe := permitidos[campo]; !existe {
			return ErrAutorizacionDenegada
		}
	}
	soportadas := make(map[string]struct{}, len(obligacionesSoportadas))
	for _, obligacion := range obligacionesSoportadas {
		soportadas[obligacion] = struct{}{}
	}
	for _, obligacion := range restricciones.Obligaciones {
		if _, existe := soportadas[obligacion]; !existe {
			return ErrAutorizacionDenegada
		}
	}
	return nil
}
