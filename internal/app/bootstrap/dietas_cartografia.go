package bootstrap

import (
	"net/http"

	"vec-diputacion-granada/config"
	httpcartografia "vec-diputacion-granada/internal/modules/dietas/adapters/httpcartografia"
	dietasosrm "vec-diputacion-granada/internal/modules/dietas/adapters/osrm"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
)

// nuevoCasoUsoCalculoRutas es la unica composicion del conector cartografico.
// Produccion puede dejarlo desconectado; la raiz aislada de presentacion exige
// despues que el resultado no sea nil.
func nuevoCasoUsoCalculoRutas(cfg config.Config) (*dietasapp.ServicioCalculoRutas, error) {
	motor, err := dietasosrm.Nuevo(dietasosrm.Configuracion{
		URLBase: cfg.OSRMBaseURL, NombreAmbito: cfg.OSRMScopeName,
		LimitesAmbito: cfg.OSRMScopeBounds, CIDRPermitidas: append([]string(nil), cfg.OSRMAllowedCIDRs...),
		VersionGrafo: cfg.OSRMGraphVersion,
	})
	if err != nil || motor == nil {
		return nil, err
	}
	return dietasapp.NuevoServicioCalculoRutas(motor)
}

func nuevoManejadorProductivoCalculoRutas(cfg config.Config) (http.Handler, error) {
	casoUso, err := nuevoCasoUsoCalculoRutas(cfg)
	if err != nil || casoUso == nil {
		return nil, err
	}
	return httpcartografia.NuevoManejador(casoUso, httpcartografia.OpcionesManejador{EnvolverEnDatos: true})
}
