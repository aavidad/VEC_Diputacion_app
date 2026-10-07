package administracion

import (
	"net/http"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

// DependenciasCopias reuses the existing ADMIN resolver and audit authority via
// small typed callbacks; no new session, role assignment or certificate policy.
type DependenciasCopias struct {
	ResolverSesion  adapter.ResolverSesion
	AuditorFrontera adapter.AuditorFrontera
	Autoridad       p.Autorizador
	Lecturas        p.Consultas
	Cambios         p.Cambios
	Control         p.Control
	Opciones        p.FuenteOpciones
	FuenteRevision  p.FuenteRevision
}

// NuevoHandlerCopias is mounted only behind the existing live ADMIN TLS/CRL
// verification. Missing business ports remain unavailable, never successful.
func NuevoHandlerCopias(origen string, deps DependenciasCopias) (http.Handler, error) {
	return adapter.Nuevo(origen, deps.ResolverSesion, deps.AuditorFrontera,
		&app.Servicio{Autoridad: deps.Autoridad, Lecturas: deps.Lecturas, Cambios: deps.Cambios, Control: deps.Control, Opciones: deps.Opciones, FuenteRevision: deps.FuenteRevision})
}
