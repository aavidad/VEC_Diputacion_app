package bootstrap

import (
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
)

// rutaReglasSituacionBolsaDesarrollo va bajo la misma frontera mTLS de
// consulta RRHH que el resto de lecturas de Bolsa.
const rutaReglasSituacionBolsaDesarrollo = bolsahttp.RutaReglasSituacion

// componerReglasSituacionBolsaDesarrollo engancha el catálogo de reglas de
// Bolsa al cambio de situación: restringe con él el servicio ya
// compuesto y publica la lectura que usa la pantalla de RRHH, con los
// destinos que el servicio admitirá. Con resolutor nulo (sin catálogo) la
// ruta responde «configuradas: false» y el servicio aplica la política que ya
// tenga la base (o la compilada, si la base no tiene la migración 000032).
// El arranque nunca publica una nueva versión.
func componerReglasSituacionBolsaDesarrollo(resolutor *reglas.Resolutor, mutador http.Handler) (vechttp.RutaExacta, error) {
	consulta := reglasbolsa.NuevasReglasSituacion(resolutor)
	var servicio *aplicacionbolsa.ServicioSituacionParticipacion
	if participacion, ok := mutador.(*manejadorParticipacionBolsaDesarrollo); ok && participacion != nil {
		servicio = participacion.servicioSituacion
	}
	var fuente bolsahttp.FuenteTransicionesSituacion
	if servicio != nil {
		if consulta.Configurada() {
			servicio.EstablecerReglasTransiciones(consulta)
		}
		fuente = servicio
	}
	manejador, err := bolsahttp.NuevoHandlerReglasSituacionConTransiciones(consulta, fuente)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaReglasSituacion, Manejador: manejador}, nil
}
