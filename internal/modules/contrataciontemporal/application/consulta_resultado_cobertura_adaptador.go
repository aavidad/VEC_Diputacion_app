package application

import "context"

// ConsultarParaAdaptador mantiene dentro de aplicación la apertura del
// resultado nominal. La frontera recibe únicamente la proyección mínima y no
// puede fabricar un resultado confirmado.
func (s *ServicioConsultaResultadoCobertura) ConsultarParaAdaptador(
	ctx context.Context,
	solicitud SolicitudConsultaResultadoCobertura,
) (DatosConsultaResultadoCoberturaParaAdaptador, error) {
	resultado, err := s.Consultar(ctx, solicitud)
	if err != nil {
		return DatosConsultaResultadoCoberturaParaAdaptador{}, err
	}
	datos, valida := resultado.DatosParaAdaptador()
	if !valida {
		return DatosConsultaResultadoCoberturaParaAdaptador{},
			ErrConsultaResultadoCoberturaNoConfiable
	}
	return datos, nil
}
