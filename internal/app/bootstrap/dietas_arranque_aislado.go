package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

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
	cartografiaValida := true
	// El shell construye su manejador OSRM más tarde. Se valida aquí para
	// impedir que un campo cartográfico incompleto pare el arranque común.
	if _, err := nuevoCasoUsoCalculoRutas(cfg); err != nil {
		registrarFalloDietasArranque("cartografia", err)
		cartografiaValida = false
		m.cfgShell.OSRMBaseURL = ""
		m.cfgShell.OSRMScopeName = ""
		m.cfgShell.OSRMScopeBounds = ""
		m.cfgShell.OSRMAllowedCIDRs = nil
		m.cfgShell.OSRMGraphVersion = ""
	}
	var err error
	if cartografiaValida {
		m.rutas, m.cerrarRutas, err = nuevasRutasDietasConMaterialComunDesarrollo(cfg, resolvedor, derivador, material.rutas)
		if err != nil {
			registrarFalloDietasArranque("rutas", err)
			m.rutas = nil
		}
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
	if m.comisiones != nil && m.rutas == nil {
		m.rutas = autoridadRutasDietasNoDisponible{}
	}
	return m
}

// La autoridad cerrada permite que la ausencia local de cartografía devuelva
// 503 en las dos rutas, incluso cuando el resto de Dietas está operativo.
type autoridadRutasDietasNoDisponible struct{}

func (autoridadRutasDietasNoDisponible) AutorizarPeticionRutaDietas(context.Context, *http.Request) error {
	return vechttp.ErrRutaDietasNoDisponible
}

// publicarMaterialDietasAisladoDesarrollo da a Dietas un plazo propio.
// Si alguna audiencia falla, no entrega ni siquiera los proveedores ya creados.
func publicarMaterialDietasAisladoDesarrollo(
	plazo time.Duration,
	publicar func(context.Context, string) (*proveedorMaterialAltaContratacionTemporalDesarrollo, error),
) (materialDietasDesdeCTDesarrollo, error) {
	if publicar == nil || plazo <= 0 {
		return materialDietasDesdeCTDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazo)
	defer cancelar()
	descriptores := descriptoresMaterialDietasDesarrollo()
	proveedores := make(map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo, len(descriptores))
	for _, descriptor := range descriptores {
		proveedor, err := publicar(ctx, descriptor.Audiencia)
		if err != nil {
			return materialDietasDesdeCTDesarrollo{}, err
		}
		if err := ctx.Err(); err != nil {
			return materialDietasDesdeCTDesarrollo{}, err
		}
		if proveedor == nil {
			return materialDietasDesdeCTDesarrollo{}, ErrComposicionDesarrolloIncompleta
		}
		proveedores[descriptor.Audiencia] = proveedor
	}
	if len(proveedores) == 0 {
		return materialDietasDesdeCTDesarrollo{}, ErrComposicionDesarrolloIncompleta
	}
	return materialDietasDesdeCTDesarrollo{
		personal:    proveedores[audienciaConsumoPersonalDietasDesarrollo],
		crear:       proveedores[audienciaConsumoCrearDietasDesarrollo],
		consultar:   proveedores[audienciaConsumoConsultarDietasDesarrollo],
		adicionales: proveedores,
	}, nil
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
