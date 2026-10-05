package administracion

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"mime"
	"net"
	"net/http"
	"path"
	"strings"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/httpapi/adminselector"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasPerfiles reutiliza el catálogo y la autoridad de actos centrales.
// Las implementaciones deben revalidar autorización y CAS en el efecto/replay.
// Activos contiene únicamente los archivos públicos del ensamblaje ADMIN.
type DependenciasPerfiles struct {
	// ContextoConexion se obtiene del resolver ADMIN, nunca del cliente.
	ContextoConexion      func(context.Context, net.Conn) context.Context
	Sesiones              api.ResolvedorSesion
	Lecturas              api.FuenteLecturas
	Catalogo              ports.CatalogoRolesAdministrables
	Actos                 ports.AutoridadActosAdministracionPerfiles
	Auditor               api.AuditorFrontera
	Reloj                 ports.Reloj
	Activos               fs.FS
	ObservadorSelector    adminselector.FuenteObservacion
	FuenteSeleccion       FuenteSeleccionAuditadaADMIN
	AudienciaSelector     string
	SoloUsuariosMetadatos bool
	Lote                  *LoteADMIN
}

type handlerPerfilesADMIN struct {
	contextoConexion  func(context.Context, net.Conn) context.Context
	api               http.Handler
	activos           fs.FS
	rutas             map[string]string
	sesiones          api.ResolvedorSesion
	lecturas          api.FuenteLecturas
	auditor           api.AuditorFrontera
	selector          *adminselector.Handler
	observador        adminselector.FuenteObservacion
	fuenteSeleccion   FuenteSeleccionAuditadaADMIN
	host              hostAdmin
	audienciaSelector string
	reloj             ports.Reloj
}

func NuevoServidorConPerfiles(cfg Configuracion, deps DependenciasPerfiles) (*http.Server, error) {
	host, hostValido := analizarHostAdmin(cfg.Host)
	if !hostValido {
		return nil, ErrConfiguracion
	}
	handler, err := nuevoHandlerPerfiles(host, deps)
	if err != nil {
		return nil, err
	}
	return nuevoServidor(cfg, handler)
}

func nuevoHandlerPerfiles(host hostAdmin, deps DependenciasPerfiles) (*handlerPerfilesADMIN, error) {
	servicio, err := application.NuevoServicioAdministracionPerfiles(deps.Catalogo, deps.Actos, deps.Reloj)
	if err != nil || deps.Activos == nil {
		return nil, ErrConfiguracion
	}
	handler, err := api.NuevoHandler(host.origen(), deps.Sesiones, deps.Lecturas, deps.Catalogo, servicio, deps.Auditor)
	if err != nil {
		return nil, ErrConfiguracion
	}
	return montarActivosPerfiles(handler, deps, host)
}

// NuevoServidorConLecturas monta la consulta ADMIN sin autoridad de escritura.
// Los POST permanecen denegados y auditados por el adaptador HTTP común.
func NuevoServidorConLecturas(cfg Configuracion, deps DependenciasPerfiles) (*http.Server, error) {
	if deps.ContextoConexion == nil || deps.Reloj == nil || deps.Catalogo != nil || deps.Actos != nil {
		return nil, ErrConfiguracion
	}
	if deps.Lecturas == nil {
		deps.Lecturas = lecturasNoDisponibles{}
	}
	host, hostValido := analizarHostAdmin(cfg.Host)
	if !hostValido {
		return nil, ErrConfiguracion
	}
	constructor := api.NuevoHandlerLecturas
	if deps.SoloUsuariosMetadatos {
		constructor = api.NuevoHandlerUsuariosMetadatos
	}
	if deps.Lote != nil {
		// El lote sólo se monta junto a las lecturas nominales de usuarios.
		if !deps.SoloUsuariosMetadatos || dependenciaComposicionNula(deps.Lote.Catalogo) || dependenciaComposicionNula(deps.Lote.Servicio) {
			return nil, ErrConfiguracion
		}
		lote := *deps.Lote
		constructor = func(origen string, sesiones api.ResolvedorSesion, lecturas api.FuenteLecturas, auditor api.AuditorFrontera) (*api.Handler, error) {
			return api.NuevoHandlerUsuariosMetadatosConLote(origen, lote.Organizacion, sesiones, lecturas, lote.Catalogo, lote.Servicio, auditor)
		}
	}
	handler, err := constructor(host.origen(), deps.Sesiones, deps.Lecturas, deps.Auditor)
	if err != nil {
		return nil, ErrConfiguracion
	}
	montaje, err := montarActivosPerfiles(handler, deps, host)
	if err != nil {
		return nil, err
	}
	return nuevoServidor(cfg, montaje)
}

func montarActivosPerfiles(handler http.Handler, deps DependenciasPerfiles, host hostAdmin) (*handlerPerfilesADMIN, error) {
	if handler == nil || deps.Activos == nil || deps.ContextoConexion == nil || host.nombre == "" || host.autoridad == "" {
		return nil, ErrConfiguracion
	}
	rutas := map[string]string{
		"/admin/usuarios/":                             "admin/usuarios/index.html",
		"/admin/usuarios/index.html":                   "admin/usuarios/index.html",
		"/admin/usuarios/entry.js":                     "admin/usuarios/entry.js",
		"/admin/usuarios/vista.js":                     "admin/usuarios/vista.js",
		"/admin/usuarios/render.js":                    "admin/usuarios/render.js",
		"/admin/usuarios/contratos.js":                 "admin/usuarios/contratos.js",
		"/admin/usuarios/metadatos.js":                 "admin/usuarios/metadatos.js",
		"/admin/usuarios/cliente.js":                   "admin/usuarios/cliente.js",
		"/admin/usuarios/lecturas-http.js":             "admin/usuarios/lecturas-http.js",
		"/admin/usuarios/propuestas.js":                "admin/usuarios/propuestas.js",
		"/admin/usuarios/propuestas-contratos.js":      "admin/usuarios/propuestas-contratos.js",
		"/admin/usuarios/usuarios.css":                 "admin/usuarios/usuarios.css",
		"/administracion-perfiles/":                    "admin/usuarios/index.html",
		"/administracion-perfiles/index.html":          "admin/usuarios/index.html",
		"/administracion-perfiles/selector-perfil.js":  "administracion-perfiles/selector-perfil.js",
		"/administracion-perfiles/selector-perfil.css": "administracion-perfiles/selector-perfil.css",
		"/favicon.svg":                                 "favicon.svg",
		"/comun/idioma.js":                             "comun/idioma.js",
		"/comun/textos.js":                             "comun/textos.js",
		"/comun/tema-vec.css":                          "comun/tema-vec.css",
		"/portal-empleado/portal.css":                  "portal-empleado/portal.css",
		"/portal-empleado/portal-componentes.css":      "portal-empleado/portal-componentes.css",
		"/portal-empleado/portal-flujos.css":           "portal-empleado/portal-flujos.css",
		"/portal-empleado/portal-patrones.css":         "portal-empleado/portal-patrones.css",
		"/textos/idiomas.json":                         "textos/idiomas.json",
	}
	indice, err := fs.ReadFile(deps.Activos, "textos/idiomas.json")
	if err != nil || len(indice) > 32768 {
		return nil, ErrConfiguracion
	}
	var idiomas struct {
		Idiomas []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if json.Unmarshal(indice, &idiomas) != nil || len(idiomas.Idiomas) == 0 || len(idiomas.Idiomas) > 32 {
		return nil, ErrConfiguracion
	}
	for _, idioma := range idiomas.Idiomas {
		if len(idioma.Codigo) < 2 || len(idioma.Codigo) > 16 {
			return nil, ErrConfiguracion
		}
		for _, c := range idioma.Codigo {
			if (c < 'a' || c > 'z') && c != '-' {
				return nil, ErrConfiguracion
			}
		}
		for _, catalogo := range []string{"admin-usuarios", "admin-selector"} {
			fichero := "textos/" + idioma.Codigo + "/" + catalogo + ".json"
			rutas["/"+fichero] = fichero
		}
	}
	for _, fichero := range rutas {
		if _, err := fs.Stat(deps.Activos, fichero); err != nil {
			return nil, ErrConfiguracion
		}
	}
	var selector *adminselector.Handler
	if deps.ObservadorSelector != nil || deps.FuenteSeleccion != nil {
		var err error
		selector, err = adminselector.NuevoHandler(host.origen(), deps.AudienciaSelector, deps.ObservadorSelector, seleccionAuditadaADMIN{fuente: deps.FuenteSeleccion}, deps.Auditor, deps.Reloj)
		if err != nil {
			return nil, ErrConfiguracion
		}
	}
	return &handlerPerfilesADMIN{api: handler, activos: deps.Activos, rutas: rutas, sesiones: deps.Sesiones, lecturas: deps.Lecturas, auditor: deps.Auditor, contextoConexion: deps.ContextoConexion, selector: selector, observador: deps.ObservadorSelector, fuenteSeleccion: deps.FuenteSeleccion, host: host, audienciaSelector: deps.AudienciaSelector, reloj: deps.Reloj}, nil
}

func (h *handlerPerfilesADMIN) atiende(ruta string) bool {
	if h == nil {
		return false
	}
	if strings.HasPrefix(ruta, api.PrefijoV1+"/") {
		return true
	}
	if strings.HasPrefix(ruta, adminselector.PrefijoV1+"/") {
		return true
	}
	_, ok := h.rutas[ruta]
	return ok
}

func (h *handlerPerfilesADMIN) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, adminselector.PrefijoV1+"/") {
		if h.selector == nil {
			w.WriteHeader(h.denegarActivos(r.Context(), http.StatusServiceUnavailable, "servicio_no_disponible"))
			return
		}
		h.selector.ServeHTTP(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, api.PrefijoV1+"/") {
		h.api.ServeHTTP(w, r)
		return
	}
	fichero, ok := h.rutas[r.URL.Path]
	if !ok || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	// La frontera externa ya cotejó CA/CRL/Host/red. Una hoja admitida sigue
	// sin conceder acceso ADMIN: la sesión y capacidad de lectura son centrales.
	if estado := h.estadoLecturaActivos(r.Context(), r); estado != http.StatusOK {
		w.WriteHeader(estado)
		return
	}
	contenido, err := fs.ReadFile(h.activos, fichero)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	tipo := mime.TypeByExtension(path.Ext(fichero))
	if tipo == "" {
		tipo = "application/octet-stream"
	}
	w.Header().Set("Content-Type", tipo)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; font-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	// #nosec G705 -- Activos públicos del ensamblaje, con ruta fija; no hay contenido ni plantilla del usuario.
	_, _ = w.Write(contenido)
}
func (h *handlerPerfilesADMIN) estadoLecturaActivos(ctx context.Context, r *http.Request) int {
	for _, nombre := range []string{"Authorization", "Proxy-Authorization", "Cookie", "X-Remote-User", "X-Forwarded-Client-Cert", "X-Client-Cert", "X-SSL-Client-Cert"} {
		if len(r.Header.Values(nombre)) != 0 {
			return h.denegarActivos(ctx, http.StatusUnauthorized, "autenticacion_requerida")
		}
	}
	if h.observador != nil {
		return h.estadoActivosSelector(ctx, r)
	}
	sesion, err := h.sesiones.ResolverSesionADMIN(ctx, r)
	if err != nil {
		estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
		if errors.Is(err, api.ErrAutenticacionRequerida) {
			estado, codigo = http.StatusUnauthorized, "autenticacion_requerida"
		}
		if errors.Is(err, api.ErrAccesoDenegado) {
			estado, codigo = http.StatusForbidden, "acceso_denegado"
		}
		return h.denegarActivos(ctx, estado, codigo)
	}
	if sesion.Actor.Validar() != nil || sesion.Evidencia.ValidarPara(sesion.Actor) != nil ||
		sesion.InstantaneaAutorizacion.Validar() != nil ||
		sesion.Actor.PersonaRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID || sesion.Actor.PerfilActivoRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef {
		return h.denegarActivos(ctx, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
	capacidades, err := h.lecturas.Capacidades(ctx, sesion.Actor, sesion.Evidencia)
	// FuenteLecturas conserva la auditoría de esta lectura, incluida denegación.
	if err != nil {
		if errors.Is(err, api.ErrAccesoDenegado) {
			return http.StatusForbidden
		}
		return http.StatusServiceUnavailable
	}
	if capacidades.Version != "1" || capacidades.ActorPersonaRef != sesion.Actor.PersonaRef {
		return http.StatusServiceUnavailable
	}
	for _, accion := range capacidades.Acciones {
		if accion == "consultar" {
			return http.StatusOK
		}
	}
	return http.StatusForbidden
}
func (h *handlerPerfilesADMIN) denegarActivos(ctx context.Context, estado int, codigo string) int {
	if h.auditor == nil || h.auditor.RegistrarDenegacionADMIN(ctx, api.DenegacionADMIN{Codigo: codigo, Accion: "consultar", RecursoRef: "administracion:perfiles:activos"}) != nil {
		return http.StatusServiceUnavailable
	}
	return estado
}
