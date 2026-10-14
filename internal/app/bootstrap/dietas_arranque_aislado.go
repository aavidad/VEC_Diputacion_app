package bootstrap

import (
	"errors"
	"log/slog"
	"net/http"

	"vec-diputacion-granada/config"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
)

// montajeDietasAislado conserva sólo las capacidades que llegaron a montarse.
// Una configuración defectuosa de Dietas no autoriza ninguna ruta del módulo.
type montajeDietasAislado struct {
	rutas       vechttp.AutoridadPeticionRutasDietas
	comisiones  *autoridadComisionesDietasDesarrollo
	cerrarRutas func()
	denegar     bool
	cfgShell    config.Config
}

func componerDietasAisladasDesarrollo(
	cfg config.Config,
	resolvedor vechttp.DemoIdentityResolver,
	derivador *derivadorIdentidadOperacionDesarrollo,
	material materialDietasDesdeCTDesarrollo,
) montajeDietasAislado {
	m := montajeDietasAislado{cerrarRutas: func() {}, cfgShell: cfg}
	// El shell construye su manejador OSRM más tarde. Se valida aquí para
	// impedir que un campo cartográfico incompleto pare el arranque común.
	if _, err := nuevoCasoUsoCalculoRutas(cfg); err != nil {
		registrarFalloDietasArranque("cartografia", err)
		m.denegar = true
		m.cfgShell.OSRMBaseURL = ""
		m.cfgShell.OSRMScopeName = ""
		m.cfgShell.OSRMScopeBounds = ""
		m.cfgShell.OSRMAllowedCIDRs = nil
		m.cfgShell.OSRMGraphVersion = ""
		return m
	}
	var err error
	m.rutas, m.cerrarRutas, err = nuevasRutasDietasDesarrollo(cfg, resolvedor, derivador)
	if err != nil {
		registrarFalloDietasArranque("rutas", err)
		m.rutas = nil
		m.denegar = true
		return m
	}
	if m.cerrarRutas == nil {
		m.cerrarRutas = func() {}
	}
	m.comisiones, err = nuevasComisionesDietasDesarrollo(cfg, resolvedor, derivador, material)
	if err != nil {
		registrarFalloDietasArranque("comisiones", err)
		m.cerrarRutas()
		m.cerrarRutas = func() {}
		m.rutas = nil
		m.comisiones = nil
		m.denegar = true
	}
	return m
}

// El mensaje del error puede contener rutas, DSN o material privado. Sólo se
// registran etiquetas del código y categorías cerradas.
func registrarFalloDietasArranque(componente string, err error) {
	if err == nil {
		return
	}
	causa := "dependencia_no_disponible"
	switch {
	case componente == "cartografia":
		causa = "configuracion_cartografia_invalida"
	case errors.Is(err, config.ErrConfiguracionDietasBorradoresSelector):
		causa = "selector_invalido"
	case errors.Is(err, config.ErrConfiguracionDietasBorradoresActivacion),
		errors.Is(err, config.ErrConfiguracionDietasBorradoresIncompleta),
		errors.Is(err, config.ErrConfiguracionDietasBorradoresNoSeparada):
		causa = "configuracion_incompleta"
	case errors.Is(err, vechttp.ErrRutaDietasNoDisponible):
		causa = "rutas_no_disponibles"
	}
	slog.Error("dietas: componente no disponible", "componente", componente, "causa", causa)
}

func denegarDietasNoDisponibles(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && (esRutaComisionesDietas(r.URL.Path) ||
			r.URL.Path == "/api/vec/dietas/route-catalog" || r.URL.Path == "/api/vec/dietas/road-route") {
			responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
}
