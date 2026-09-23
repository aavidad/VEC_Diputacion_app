package bootstrap

import (
	"net/http"

	httpcatalogo "vec-diputacion-granada/internal/modules/dietas/adapters/httpcatalogo"
)

// nuevoManejadorCatalogoRutasDietas queda separado de bootstrap.go para que la
// composición raíz conecte explícitamente esta consulta con su autorización.
func nuevoManejadorCatalogoRutasDietas() http.Handler {
	return httpcatalogo.NuevoManejador()
}
