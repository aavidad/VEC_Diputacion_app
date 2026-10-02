package administracion

import (
	"context"
	"encoding/json"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// DependenciasPerfiles reutiliza el catálogo y la autoridad de actos centrales.
// Las implementaciones deben revalidar autorización y CAS en el efecto/replay.
// Activos contiene únicamente los archivos públicos del ensamblaje ADMIN.
type DependenciasPerfiles struct {
	Sesiones api.ResolvedorSesion
	Lecturas api.FuenteLecturas
	Catalogo ports.CatalogoRolesAdministrables
	Actos    ports.AutoridadActosAdministracionPerfiles
	Auditor  api.AuditorFrontera
	Reloj    ports.Reloj
	Activos  fs.FS
}

type handlerPerfilesADMIN struct {
	api      http.Handler
	activos  fs.FS
	rutas    map[string]string
	sesiones api.ResolvedorSesion
	lecturas api.FuenteLecturas
}

func NuevoServidorConPerfiles(cfg Configuracion, deps DependenciasPerfiles) (*http.Server, error) {
	handler, err := nuevoHandlerPerfiles("https://"+cfg.Host, deps)
	if err != nil {
		return nil, err
	}
	return nuevoServidor(cfg, handler)
}

func nuevoHandlerPerfiles(origen string, deps DependenciasPerfiles) (*handlerPerfilesADMIN, error) {
	servicio, err := application.NuevoServicioAdministracionPerfiles(deps.Catalogo, deps.Actos, deps.Reloj)
	if err != nil || deps.Activos == nil {
		return nil, ErrConfiguracion
	}
	handler, err := api.NuevoHandler(origen, deps.Sesiones, deps.Lecturas, deps.Catalogo, servicio, deps.Auditor)
	if err != nil {
		return nil, ErrConfiguracion
	}
	rutas := map[string]string{
		"/administracion-perfiles/":                            "administracion-perfiles/index.html",
		"/administracion-perfiles/index.html":                  "administracion-perfiles/index.html",
		"/administracion-perfiles/administracion-perfiles.js":  "administracion-perfiles/administracion-perfiles.js",
		"/administracion-perfiles/administracion-perfiles.css": "administracion-perfiles/administracion-perfiles.css",
		"/administracion-perfiles/cliente.js":                  "administracion-perfiles/cliente.js",
		"/comun/idioma.js":                                     "comun/idioma.js",
		"/comun/textos.js":                                     "comun/textos.js",
		"/comun/tema-vec.css":                                  "comun/tema-vec.css",
		"/portal-empleado/portal.css":                          "portal-empleado/portal.css",
		"/portal-empleado/portal-componentes.css":              "portal-empleado/portal-componentes.css",
		"/textos/idiomas.json":                                 "textos/idiomas.json",
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
		fichero := "textos/" + idioma.Codigo + "/administracion-perfiles.json"
		rutas["/"+fichero] = fichero
	}
	for _, fichero := range rutas {
		if _, err := fs.Stat(deps.Activos, fichero); err != nil {
			return nil, ErrConfiguracion
		}
	}
	return &handlerPerfilesADMIN{api: handler, activos: deps.Activos, rutas: rutas, sesiones: deps.Sesiones, lecturas: deps.Lecturas}, nil
}

func (h *handlerPerfilesADMIN) atiende(ruta string) bool {
	if h == nil {
		return false
	}
	if strings.HasPrefix(ruta, api.PrefijoV1+"/") {
		return true
	}
	_, ok := h.rutas[ruta]
	return ok
}

func (h *handlerPerfilesADMIN) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	if !h.puedeLeerActivos(r.Context(), r) {
		w.WriteHeader(http.StatusForbidden)
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
	_, _ = w.Write(contenido)
}
func (h *handlerPerfilesADMIN) puedeLeerActivos(ctx context.Context, r *http.Request) bool {
	sesion, err := h.sesiones.ResolverSesionADMIN(ctx, r)
	if err != nil || sesion.Actor.Validar() != nil || sesion.InstantaneaAutorizacion.Validar() != nil || sesion.Actor.PersonaRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID || sesion.Actor.PerfilActivoRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef {
		return false
	}
	capacidades, err := h.lecturas.Capacidades(ctx, sesion.Actor)
	if err != nil || capacidades.Version != "v1" || capacidades.ActorPersonaRef != sesion.Actor.PersonaRef {
		return false
	}
	for _, accion := range capacidades.Acciones {
		if accion == "consultar" {
			return true
		}
	}
	return false
}
