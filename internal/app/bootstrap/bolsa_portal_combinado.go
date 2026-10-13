package bootstrap

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/separacionportales"
	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func nuevasRutasBolsaPersonalCombinado(cfg config.Config, identidad *resolvedorIdentidadDesarrollo, incidencias vecports.EmisorIncidenciasTecnicas) (bolsa, inscripcion http.Handler, cerrar func(), err error) {
	nada := func() {}
	if !superficieExternaUsuariosEnProceso(cfg) {
		return nil, nil, nada, nil
	}
	activa, err := cfg.BolsaInscripcionesExternoActivo()
	if err != nil {
		return nil, nil, nada, err
	}
	_, sinBolsa := cfg.ExternoBolsaPostgreSQL.DSN()
	if !activa && sinBolsa != nil && (identidad == nil || identidad.candidatoBolsa == nil) {
		return nil, nil, nada, nil
	}
	if identidad == nil || identidad.candidatoBolsa == nil || incidencias == nil || sinBolsa != nil {
		return nil, nil, nada, errMiBolsaNoDisponible
	}
	directorio := cfg.ExternoMaterialDir
	if !filepath.IsAbs(directorio) || directorioDentro(cfg.DevelopmentMaterialDir, directorio) || directorioDentro(directorio, cfg.DevelopmentMaterialDir) {
		return nil, nil, nada, ErrMaterialPortalExternoInvalido
	}
	if err := separacionportales.ComprobarMaterial(separacionportales.PortalExterno, directorio); err != nil {
		return nil, nil, nada, err
	}
	if err := exigirIdempotenciaPropiaPortalExterno(cfg.DevelopmentMaterialDir, directorio); err != nil {
		return nil, nil, nada, err
	}
	externa := cfg
	externa.PortalProceso = config.ValorPortalProcesoExterno
	externa.DevelopmentMaterialDir = directorio
	rutas := externa.DevelopmentPaths()
	externa.TLSCertFile, externa.TLSKeyFile = rutas.ServerCertificate, rutas.ServerPrivateKey
	material, err := cargarMaterialPortalExterno(externa)
	if err != nil {
		return nil, nil, nada, err
	}
	if !mismaIdentidadCandidataBolsa(identidad, material.identidad) {
		return nil, nil, nada, ErrMaterialPortalExternoInvalido
	}
	idempotencia, err := cargarMaterialIdempotenciaDesarrollo(directorio, filepath.Join(directorio, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return nil, nil, nada, err
	}
	derivador, err := nuevoDerivadorIdentidadOperacionDesarrollo(&idempotencia)
	idempotencia.borrar()
	if err != nil {
		return nil, nil, nada, err
	}
	derivador.espacioSeudonimos = espacioSeudonimosPortalExterno
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	defer cancelar()
	preflight, err := abrirPoolPreflightV3PortalExterno(ctx, externa.ExternoPreflightV3DatabaseURL)
	if err != nil {
		derivador.borrar()
		return nil, nil, nada, err
	}
	cerrarBase := func() { preflight.Close(); derivador.borrar() }
	bolsa, sesion, cerrarBolsa, err := nuevaMiBolsaPortalExterno(ctx, externa, material.identidad, derivador, preflight, incidencias)
	if err != nil || bolsa == nil || sesion == nil {
		cerrarBolsa()
		cerrarBase()
		if err == nil {
			err = errMiBolsaNoDisponible
		}
		return nil, nil, nada, err
	}
	inscripcion, cerrarInscripcion, err := nuevaInscripcionPortalExterno(ctx, externa, sesion, preflight)
	if err != nil {
		cerrarBolsa()
		cerrarBase()
		return nil, nil, nada, err
	}
	cerrarCartera := nada
	cartera := meritoshttp.NuevoHandlerCarteraNoDisponible()
	if cierre, fallo := montarCarteraPortalExterno(ctx, externa, sesion, derivador, preflight); fallo != nil {
		slog.Warn("cartera_meritos_combinado_no_disponible", "causa", fallo.Error())
	} else {
		cerrarCartera = cierre
		rutas := map[string]http.Handler{
			meritoshttp.RutaCartera:              sesion.cartera.proteger(sesion.cartera.manejador),
			meritoshttp.RutaRetiradasCartera:     sesion.retirada.proteger(sesion.retirada.manejador),
			meritoshttp.RutaDeclaracionesCartera: sesion.declaracion.proteger(sesion.declaracion.manejador),
		}
		cartera = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r != nil && r.URL != nil {
				if manejador := rutas[r.URL.Path]; manejador != nil {
					manejador.ServeHTTP(w, r)
					return
				}
			}
			http.NotFound(w, r)
		})
	}
	bolsaPropia := bolsa
	bolsa = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil && rutaCarteraPersonalExterna(r.URL.Path) {
			cartera.ServeHTTP(w, r)
			return
		}
		bolsaPropia.ServeHTTP(w, r)
	})
	return bolsa, inscripcion, func() { cerrarCartera(); cerrarInscripcion(); cerrarBolsa(); cerrarBase() }, nil
}

func rutaCarteraPersonalExterna(ruta string) bool {
	return ruta == meritoshttp.RutaCartera || ruta == meritoshttp.RutaRetiradasCartera ||
		ruta == meritoshttp.RutaDeclaracionesCartera
}

// El cambio de perfil conserva la identidad certificada, Persona y candidato.
// Ninguna concesión interna se suma a la autoridad externa.
func mismaIdentidadCandidataBolsa(interna, externa *resolvedorIdentidadDesarrollo) bool {
	if interna == nil || externa == nil || interna.candidatoBolsa == nil || externa.candidatoBolsa == nil {
		return false
	}
	a, b := interna.candidatoBolsa, externa.candidatoBolsa
	return a.identidad.huella != ([sha256.Size]byte{}) && a.identidad.principal.ID != "" &&
		a.personaRef != "" && a.candidatoRef != "" && a.cuentaRef != "" && b.cuentaRef != "" &&
		a.perfilRef != "" && b.perfilRef != "" && a.perfilRef != b.perfilRef &&
		// ContextoActor (CTX15) segrega las cuentas: la externa es propia.
		a.cuentaRef != b.cuentaRef &&
		a.identidad.huella == b.identidad.huella && a.identidad.principal.ID == b.identidad.principal.ID &&
		a.personaRef == b.personaRef && a.candidatoRef == b.candidatoRef
}

// Despacha las autoridades externas antes de cualquier frontera interna CT.
func componerRaizConBolsaPersonalExterna(raiz, bolsa, inscripcion http.Handler) http.Handler {
	if bolsa == nil && inscripcion == nil {
		return raiz
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != nil && r.URL != nil {
			if inscripcion != nil && rutaInscripcionExterna(r.URL.Path) {
				inscripcion.ServeHTTP(w, r)
				return
			}
			if bolsa != nil && (bolsahttp.EsRutaPortal(r.URL.Path) || rutaCarteraPersonalExterna(r.URL.Path)) {
				bolsa.ServeHTTP(w, r)
				return
			}
		}
		raiz.ServeHTTP(w, r)
	})
}
