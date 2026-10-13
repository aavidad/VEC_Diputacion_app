package bootstrap

import (
	"context"
	"net/http"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// El proceso combinado conserva la sesión y los LOGIN del aspirante. El
// interno separado no abre conexiones externas ni compone estas rutas.
func nuevaInscripcionCombinadaDesarrollo(cfg config.Config, preferencias *composicionPreferenciasUsuarios) (http.Handler, func(), error) {
	nada := func() {}
	if !superficieExternaUsuariosEnProceso(cfg) {
		return nil, nada, nil
	}
	activo, err := cfg.BolsaInscripcionesExternoActivo()
	if err != nil {
		return nil, nada, err
	}
	if !activo {
		return nil, nada, nil
	}
	if preferencias == nil || preferencias.externa == nil {
		return nil, nada, errMontajeInscripcionBolsa
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	defer cancelar()
	preflight, err := abrirPoolPreflightV3PortalExterno(ctx, cfg.ExternoPreflightV3DatabaseURL)
	if err != nil {
		return nil, nada, err
	}
	manejador, cerrar, err := nuevaInscripcionPortalExterno(ctx, cfg, preferencias.externa, preflight)
	if err != nil {
		preflight.Close()
		return nil, nada, err
	}
	return manejador, func() { cerrar(); preflight.Close() }, nil
}

// Despacha fuera de las fronteras internas: el constructor externo acredita
// su propia sesión nominal y sólo reconoce las rutas del aspirante.
func componerRaizConInscripcionExterna(raiz, inscripcion http.Handler) http.Handler {
	if inscripcion == nil {
		return raiz
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && rutaInscripcionExterna(r.URL.Path) {
			inscripcion.ServeHTTP(w, r)
			return
		}
		raiz.ServeHTTP(w, r)
	})
}
