package bootstrap

import (
	"net/http"
	"strings"

	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// catalogoValidadoresCompetentesAsignacionDietas es la única bandera de
// composición que abre el alta (POST) y la corrección completa (PUT) de la
// asignación D7. Ambas operaciones fijan centro, administrativo y responsable,
// y hoy Personal solo valida la forma de esas referencias, no su existencia ni
// su competencia. Hasta que Base publique el catálogo gobernado de validadores
// competentes y Personal lo consuma en la misma transacción, la constante queda
// vacía: las dos operaciones no se registran, la frontera las deniega por
// defecto y el montaje responde 404 si algo llegara a alcanzarlas.
//
// Solo se rellena con la referencia versionada de ese catálogo publicado, junto
// con el consumo que lo verifica; la prueba de apertura falla mientras tanto.
// La consulta (GET) y la corrección de grupo (PUT .../grupo), que no altera
// centro, administrativo ni responsable, siguen abiertas.
const catalogoValidadoresCompetentesAsignacionDietas = ""

// escrituraAsignacionDietasAbierta indica si el catálogo gobernado permite
// componer el alta y la corrección completa de la asignación D7.
func escrituraAsignacionDietasAbierta(catalogo string) bool {
	return strings.TrimSpace(catalogo) != ""
}

// escrituraAsignacionDietas identifica una petición de alta o de
// corrección completa de la asignación D7, ya sea por ruta exacta o colección.
func escrituraAsignacionDietas(ruta, metodo string) bool {
	if ruta == personalhttp.RutaAsignacionesDietas {
		return metodo == http.MethodPost
	}
	resto, ok := strings.CutPrefix(ruta, personalhttp.RutaAsignacionesDietas+"/")
	return ok && metodo == http.MethodPut && !strings.HasSuffix(resto, "/grupo")
}

// rutasAsignacionDietas registra la asignación D7 según el catálogo. Cerrada,
// no hay ruta exacta de alta y la colección responde 404 a la corrección
// completa sin llegar al manejador de Personal.
func rutasAsignacionDietas(manejador http.Handler, catalogo string) ([]vechttp.RutaExacta, []vechttp.RutaColeccion) {
	if escrituraAsignacionDietasAbierta(catalogo) {
		return []vechttp.RutaExacta{{Ruta: personalhttp.RutaAsignacionesDietas, Manejador: manejador}},
			[]vechttp.RutaColeccion{{Prefijo: personalhttp.RutaAsignacionesDietas, Manejador: manejador}}
	}
	cerrada := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || escrituraAsignacionDietas(r.URL.Path, r.Method) {
			responderDenegacionComisionesDietas(w, http.StatusNotFound)
			return
		}
		manejador.ServeHTTP(w, r)
	})
	return nil, []vechttp.RutaColeccion{{Prefijo: personalhttp.RutaAsignacionesDietas, Manejador: cerrada}}
}
