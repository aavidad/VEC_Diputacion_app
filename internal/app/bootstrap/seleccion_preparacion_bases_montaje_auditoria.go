package bootstrap

import (
	"net/http"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// NuevaAuditoriaFronteraPreparacionBasesV3 enlaza la DI del montaje con el
// mismo registrador común de la raíz. CT168 conserva causa fija, actor vacío
// y correlación criptográfica; el callback no lee cuerpo ni cabeceras.
func NuevaAuditoriaFronteraPreparacionBasesV3(registrador vecports.RegistradorAuditoriaFronteraRutaExacta) (func(*http.Request) error, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(registrador) {
		return nil, errMontajePreparacionBasesV3
	}
	return func(r *http.Request) error {
		if r == nil || r.URL == nil || r.Method != http.MethodPost || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
			(r.URL.Path != selhttp.RutaGuardarPreparacionBases && r.URL.Path != selhttp.RutaConsultarPreparacionBases) {
			return bolsaports.ErrPreparacionBasesNoDisponible
		}
		if err := vechttp.RegistrarDenegacionFronteraPreparacion(r.Context(), registrador,
			http.MethodPost, r.URL.Path, vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado); err != nil {
			return bolsaports.ErrPreparacionBasesNoDisponible
		}
		return nil
	}, nil
}
