// Package httpcatalogo adapta el catálogo territorial gobernado de Dietas a
// una consulta HTTP interna. No calcula importes ni declara una liquidación.
package httpcatalogo

import (
	"encoding/json"
	"net/http"

	dietas "vec-diputacion-granada/internal/modules/dietas"
)

// Manejador sirve únicamente el catálogo que el cliente necesita para pedir
// una ruta al mediador OSRM. La autorización del actor pertenece a la frontera
// HTTP que lo compone; este adaptador sólo valida método y proyección.
type Manejador struct{}

func NuevoManejador() *Manejador { return &Manejador{} }

func (m *Manejador) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r == nil || r.URL == nil || r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "metodo no permitido"})
		return
	}
	if r.URL.RawQuery != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "peticion de catalogo invalida"})
		return
	}
	_ = json.NewEncoder(w).Encode(respuestaCatalogo{
		ProvinceRoutePoints: dietas.ProvinceRoutePointMaps(),
		ProvinceRouteMatrix: dietas.ProvinceRouteMatrixStatus(),
	})
}

type respuestaCatalogo struct {
	ProvinceRoutePoints []map[string]any `json:"province_route_points"`
	ProvinceRouteMatrix map[string]any   `json:"province_route_matrix"`
}

var _ http.Handler = (*Manejador)(nil)
