package interna

import (
	"net/http"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// nuevaAPIInternaCT sella el primer conjunto positivo antes de construir el
// servidor C4. Las rutas heredadas del dispatcher quedan inaccesibles incluso
// si sus paquetes Go forman parte del mismo binario.
func nuevaAPIInternaCT(
	servicio *vecapp.Service,
	rutas []httpapi.RutaExacta,
	autoridad httpapi.AutoridadRutasExactas,
	auditoria vecports.RegistradorAuditoriaFronteraRutaExacta,
) (http.Handler, error) {
	if len(rutas) != 2 || rutas[0].Ruta != httpct.RutaConsultaCuadroRRHH ||
		rutas[1].Ruta != httpct.RutaConsultaDetalleRRHH ||
		interfazNulaIdentidadOffline(autoridad) ||
		interfazNulaIdentidadOffline(auditoria) {
		return nil, ErrAPIInternaNoDisponible
	}
	api, err := httpapi.NewHandlerInternoConCapacidades(servicio, httpapi.HandlerOptions{
		RutasExactas:                             rutas,
		AutoridadRutasExactas:                    autoridad,
		RegistradorAuditoriaFronteraRutasExactas: auditoria,
	}, []httpapi.CapacidadRutaInterna{
		{Metodo: http.MethodPost, Ruta: httpct.RutaConsultaCuadroRRHH},
		{Metodo: http.MethodPost, Ruta: httpct.RutaConsultaDetalleRRHH},
	})
	if err != nil {
		return nil, ErrAPIInternaNoDisponible
	}
	return api, nil
}
