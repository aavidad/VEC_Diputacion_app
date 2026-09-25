package bootstrap

import (
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsaapplication "vec-diputacion-granada/internal/modules/bolsa/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
)

// nuevaRutaPlazoRespuestaBolsaDesarrollo compone la consulta del plazo de
// respuesta del asistente B7. Un resolutor nulo (sin catálogo de reglas)
// sigue siendo válido: la ruta responde configurada=false y el asistente
// conserva el texto libre. La ruta se protege como las demás lecturas RRHH de
// Bolsa (frontera mTLS y autoridad de rutas exactas).
func nuevaRutaPlazoRespuestaBolsaDesarrollo(resolutor *reglas.Resolutor, reloj reglas.Reloj) (vechttp.RutaExacta, error) {
	if reloj == nil {
		return vechttp.RutaExacta{}, ErrComposicionDesarrolloIncompleta
	}
	servicio, err := bolsaapplication.NuevoServicioPlazoRespuestaLlamamiento(resolutor, reloj.Ahora)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	manejador, err := bolsahttp.NuevoHandlerPlazoRespuestaLlamamiento(servicio)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaPlazoRespuestaLlamamiento, Manejador: manejador}, nil
}
