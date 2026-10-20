package bootstrap

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"vec-diputacion-granada/config"
	publicatransitoria "vec-diputacion-granada/internal/app/composicion/publicatransitoria"
	"vec-diputacion-granada/internal/app/server"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	"vec-diputacion-granada/internal/shared/telemetria"
	vecfichero "vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La carcasa conserva la identidad validada y las consultas públicas de
// Personal. Usuarios puede componer sus audiencias y autoridad nominal con
// el gobierno técnico V3 aunque CT no esté disponible.
func nuevoServidorSinContratacionTemporalDesarrollo(
	cfg config.Config,
	composicion *ComposicionSeguridadDesarrollo,
	resolvedor vechttp.DemoIdentityResolver,
	consultaCategorias *vecfichero.ConsultaCatalogos,
	categoriasPersonal *personalapp.ServicioConsultaCategoriasProfesionales,
	consultasPersonal consultasPublicasPersonal,
	emisor vecports.EmisorIncidenciasTecnicas,
) (*http.Server, *ComposicionSeguridadDesarrollo, error) {
	var usuarios *composicionPreferenciasUsuarios
	materialUsuarios, err := abrirMaterialUsuariosAisladoDesarrollo(cfg, composicion.derivadorIdempotencia)
	if err != nil {
		slog.Warn("usuarios_no_disponibles_sin_ct", "causa", telemetria.ClaseErrorArranque(err))
	}
	if materialUsuarios != nil {
		correos, errCorreos := nuevasDependenciasCorreosUsuariosDesarrollo(cfg, composicion.emisorKMS, materialUsuarios.materiales.correos)
		var imagen *dependenciasImagenUsuariosDesarrollo
		var aspirantes *dependenciasAspirantesDesarrollo
		if errCorreos == nil {
			imagen, errCorreos = nuevasDependenciasImagenUsuariosDesarrollo(cfg, materialUsuarios.materiales.imagen)
		}
		if errCorreos == nil {
			aspirantes, errCorreos = nuevasDependenciasAspirantesDesarrollo(cfg, composicion.emisorKMS, materialUsuarios.materiales.aspirantes)
		}
		if errCorreos == nil {
			lote := materialUsuarios.materiales.preferencias
			usuarios, err = nuevasRutasUsuariosPreferenciasDesarrollo(cfg, resolvedor, composicion.derivadorIdempotencia,
				materialUsuarios.gobierno, emisor, lote[0], lote[1], lote[2], lote[3], correos, imagen, aspirantes)
		} else {
			err = errCorreos
		}
		if err != nil || usuarios == nil || usuarios.interna == nil {
			if err == nil {
				err = errComposicionUsuariosPreferencias
			}
			if usuarios != nil {
				usuarios.cerrar()
				usuarios = nil
			}
			slog.Warn("usuarios_no_disponibles_sin_ct", "causa", telemetria.ClaseErrorArranque(err))
			materialUsuarios.cerrar()
			materialUsuarios = nil
		}
	}
	completa := false
	defer func() {
		if !completa {
			if usuarios != nil {
				usuarios.cerrar()
			}
			if materialUsuarios != nil {
				materialUsuarios.cerrar()
			}
		}
	}()
	var rutasUsuarios []vechttp.RutaExacta
	var autoridad vechttp.AutoridadRutasExactas
	var registrador vecports.RegistradorAuditoriaFronteraRutaExacta
	if usuarios != nil {
		rutasUsuarios = rutaUsuariosPreferencias(usuarios)
		autoridad = autoridadExactasConUsuariosPreferencias{usuarios: usuarios}
		frontera := registradorFronterasConUsuariosPreferencias{interna: usuarios.interna.registrador}
		if usuarios.externa != nil {
			frontera.externa = usuarios.externa.registrador
			if usuarios.externa.aspirantes != nil {
				frontera.aspirantes = usuarios.externa.aspirantes.registrador
			}
		}
		registrador = frontera
	}
	api, err := newVECShellAPICompuestaConIdentidadYRutas(cfg, emisor, resolvedor, categoriasPersonal, rutasUsuarios, autoridad, registrador, nil)
	if err != nil {
		return nil, nil, marcarFalloComponenteArranque("vec_shell", err)
	}
	cfgPublica := cfg
	cfgPublica.AuthMode = config.AuthModeDisabled
	publicaBolsa, err := publicatransitoria.NuevaAPIConCatalogos(cfgPublica, consultaCategorias)
	if err != nil {
		slog.Warn("bolsa_publica_no_disponible", "causa", telemetria.ClaseErrorArranque(err))
	}
	raiz := componerRaizConPersonalPublico(composeVECShellAPIConBolsasPublicas(api, publicaBolsa, nil), consultasPersonal)
	if usuarios != nil {
		raiz = usuarios.proteger(raiz)
	}
	siguiente := raiz
	raiz = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
		if r != nil && r.URL != nil && rutaDependienteContratacionTemporal(r.URL.Path) &&
			!rutaMontadaAislada(rutasUsuarios, r.URL.Path) {
			responderContratacionTemporalNoDisponible(w, r)
			return
		}
		siguiente.ServeHTTP(w, r)
	})
	servidor, err := server.NewHTTPServer(cfg, raiz)
	if err != nil {
		return nil, nil, marcarFalloComponenteArranque("servidor_http", err)
	}
	servidor.TLSConfig, err = composicion.ConfiguracionTLS()
	if err != nil {
		return nil, nil, marcarFalloComponenteArranque("configuracion_tls", err)
	}
	detenerSellado, err := iniciarSelladoAuditoria(context.Background(), cfg)
	if err != nil {
		slog.Warn("sellado_auditoria_no_disponible", "causa", telemetria.ClaseErrorArranque(err))
	} else {
		servidor.RegisterOnShutdown(detenerSellado)
	}
	if usuarios != nil {
		servidor.RegisterOnShutdown(usuarios.cerrar)
		servidor.RegisterOnShutdown(materialUsuarios.cerrar)
	}
	completa = true
	return servidor, composicion, nil
}

func rutaMontadaAislada(rutas []vechttp.RutaExacta, ruta string) bool {
	for _, montada := range rutas {
		if montada.Ruta == ruta {
			return true
		}
	}
	return false
}

func responderContratacionTemporalNoDisponible(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusServiceUnavailable)
	if r.Method == http.MethodHead {
		return
	}
	if _, err := io.WriteString(w, `{"codigo":"servicio_no_disponible"}`); err != nil {
		slog.Warn("contratacion_temporal_respuesta_interrumpida", "causa", "escritura_respuesta_no_disponible")
	}
}

func rutaDependienteContratacionTemporal(ruta string) bool {
	for _, prefijo := range [...]string{
		"/api/vec/contratacion-temporal", "/api/vec/bolsa", "/api/vec/documentos",
		"/api/vec/auditoria", "/api/vec/calendarios", "/api/vec/reglas/vigentes",
		"/api/vec/dietas/comisiones", "/api/vec/usuarios", "/api/vec/aspirantes",
		"/api/vec/personal/asignaciones-dietas", "/api/vec/personal/relaciones-dietas",
		"/api/vec/personal/organizacion-historica",
		"/api/vec/personal/solicitudes-rectificacion-dietas", "/api/vec/cronos",
		"/api/interna/cronos", "/api/interna/personal",
	} {
		if ruta == prefijo || strings.HasPrefix(ruta, prefijo+"/") {
			return true
		}
	}
	return false
}
