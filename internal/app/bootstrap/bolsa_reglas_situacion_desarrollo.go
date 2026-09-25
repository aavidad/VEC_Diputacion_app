package bootstrap

import (
	"net/http"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	reglasbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/reglas"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/reglas"
)

// rutaReglasSituacionBolsaDesarrollo va bajo la misma frontera mTLS de
// consulta RRHH que el resto de lecturas de Bolsa.
const rutaReglasSituacionBolsaDesarrollo = bolsahttp.RutaReglasSituacion

// componerReglasSituacionBolsaDesarrollo engancha el catálogo de reglas de
// Bolsa al cambio de situación: restringe las transiciones del servicio ya
// compuesto y publica la lectura que usa la pantalla de RRHH. Con resolutor
// nulo (sin catálogo) la ruta responde «configuradas: false» y el servicio
// conserva su tabla compilada.
func componerReglasSituacionBolsaDesarrollo(resolutor *reglas.Resolutor, mutador http.Handler) (vechttp.RutaExacta, error) {
	consulta := reglasbolsa.NuevasReglasSituacion(resolutor)
	if participacion, ok := mutador.(*manejadorParticipacionBolsaDesarrollo); ok && participacion != nil && consulta.Configurada() {
		participacion.servicioSituacion.EstablecerReglasTransiciones(consulta)
	}
	manejador, err := bolsahttp.NuevoHandlerReglasSituacion(consulta)
	if err != nil {
		return vechttp.RutaExacta{}, err
	}
	return vechttp.RutaExacta{Ruta: bolsahttp.RutaReglasSituacion, Manejador: manejador}, nil
}
