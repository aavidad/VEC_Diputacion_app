package administracion

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
	adapter "vec-diputacion-granada/internal/modules/administracion/adapters/httpcopias"
	d "vec-diputacion-granada/internal/vec/domain"

	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

// NuevoServidorCopias monta la vista K y su API dentro de la frontera F.
// La composición recibe la sesión ADMIN central; no abre cuentas ni perfiles.
// Sin proveedores de sesión, autorización y auditoría no se crea el servidor.
func NuevoServidorCopias(cfg Configuracion, deps DependenciasCopias, publicos fs.FS, auditor AuditorLecturasCopias) (*http.Server, error) {
	if app.Ausente(auditor) || app.Ausente(publicos) || deps.ResolverSesion == nil || deps.AuditorFrontera == nil || app.Ausente(deps.Autoridad) || app.Ausente(deps.Lecturas) {
		return nil, p.ErrNoDisponible
	}
	api, err := NuevoHandlerCopias("https://"+cfg.Host, deps)
	if err != nil {
		return nil, err
	}
	superficie := nuevaSuperficieCopias(cfg, deps, publicos, auditor, api)
	return nuevoServidor(cfg, superficie, func(ctx context.Context) error { return deps.AuditorFrontera(ctx, denegacionSuperficie(p.Sesion{})) })
}

func nuevaSuperficieCopias(cfg Configuracion, deps DependenciasCopias, publicos fs.FS, auditor AuditorLecturasCopias, api http.Handler) http.Handler {
	catalogos := catalogosPublicosCopias(publicos)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/admin/copias/v1") {
			api.ServeHTTP(w, r)
			return
		}
		archivo, admitido := publicoCopias(r.URL.Path, catalogos)
		if !admitido {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet || !consultaPublicaCopias(r.URL, catalogos) {
			rechazarSuperficie(w, r, deps, p.Sesion{}, http.StatusBadRequest)
			return
		}
		for _, k := range []string{"Cookie", "Authorization", "Proxy-Authorization", "X-Remote-User", "X-Forwarded-Client-Cert", "X-Client-Cert", "X-SSL-Client-Cert"} {
			if len(r.Header.Values(k)) != 0 {
				rechazarSuperficie(w, r, deps, p.Sesion{}, http.StatusForbidden)
				return
			}
		}
		if len(r.Header.Values("Origin")) > 1 || r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "https://"+cfg.Host || len(r.Header.Values("Sec-Fetch-Site")) > 1 || r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin" {
			rechazarSuperficie(w, r, deps, p.Sesion{}, http.StatusForbidden)
			return
		}
		s, err := deps.ResolverSesion(r.Context(), r)
		if err == nil && !sesionCopiasValida(s) {
			err = p.ErrNoDisponible
		}
		if err == nil {
			err = deps.Autoridad.AutorizarCopias(r.Context(), s, p.Consultar, "copias")
		}
		if err != nil {
			if deps.AuditorFrontera(r.Context(), denegacionSuperficie(s)) != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}
		if auditor.RegistrarLecturaCopias(r.Context(), s, "superficie", "copias", "consultado") != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		b, err := leerPublicoCopias(publicos, archivo)
		if err != nil || len(b) > 2<<20 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(archivo)))
		w.Header().Set("Cache-Control", "no-store, no-transform")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Referrer-Policy", "no-referrer")
		_, _ = w.Write(b)
	})
}

func publicoCopias(url string, catalogos map[string]string) (string, bool) {
	if archivo, ok := catalogos[url]; ok {
		return archivo, true
	}
	switch url {
	case "/admin/copias/":
		return "admin/copias/index.html", true
	case "/admin/copias/montaje.js", "/admin/copias/vista.js", "/admin/copias/cliente-http.js", "/admin/copias/contratos.js", "/admin/copias/estilos.css", "/admin/copias/i18n.js", "/admin/copias/vista-configuracion.js", "/admin/copias/vista-recuperacion.js", "/admin/copias/vista-soporte.js", "/comun/textos.js", "/comun/idioma.js", "/comun/iconos-vec.js", "/comun/tema-vec.css", "/portal-empleado/portal.css", "/portal-empleado/portal-componentes.css", "/textos/idiomas.json":
		return strings.TrimPrefix(url, "/"), true
	default:
		return "", false
	}
}

func denegacionSuperficie(s p.Sesion) adapter.Denegacion {
	return adapter.Denegacion{Codigo: "acceso_denegado", Accion: "superficie", RecursoRef: "copias", ActorPersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef, CorrelacionRef: s.CorrelacionRef}
}

func sesionCopiasValida(s p.Sesion) bool {
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	return s.Actor.Validar() == nil && s.Instantanea.Validar() == nil && s.Actor.Instantanea.VigenteEn(ahora) && s.Instantanea.AsignacionPerfil.VigenteEn(ahora) && s.Instantanea.ControlVigenciaVersionRol.Estado == d.EstadoControlVigenciaVersionRolHabilitada && s.Actor.Principal.AuthMethod == d.AuthMethodCertificate && s.Actor.Principal.AuthAssurance == d.AuthAssuranceHigh && s.Actor.PersonaRef == s.Instantanea.AsignacionPerfil.PrincipalID && s.Actor.PerfilActivoRef == s.Instantanea.AsignacionPerfil.PerfilActivoRef && d.ReferenciaCorrelacionAutorizacionV2Valida(s.CorrelacionRef)
}

func rechazarSuperficie(w http.ResponseWriter, r *http.Request, d DependenciasCopias, s p.Sesion, status int) {
	if d.AuditorFrontera(r.Context(), denegacionSuperficie(s)) != nil {
		status = http.StatusServiceUnavailable
	}
	w.WriteHeader(status)
}

func leerPublicoCopias(publicos fs.FS, archivo string) ([]byte, error) {
	f, err := publicos.Open(archivo)
	if err != nil {
		return nil, p.ErrNoDisponible
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || !st.Mode().IsRegular() || st.Size() > 2<<20 {
		return nil, p.ErrNoDisponible
	}
	b, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil || len(b) > 2<<20 {
		return nil, p.ErrNoDisponible
	}
	return b, nil
}

func catalogosPublicosCopias(publicos fs.FS) map[string]string {
	b, err := leerPublicoCopias(publicos, "textos/idiomas.json")
	if err != nil {
		return nil
	}
	var indice struct {
		Idiomas []struct {
			Codigo string `json:"codigo"`
		} `json:"idiomas"`
	}
	if json.Unmarshal(b, &indice) != nil || len(indice.Idiomas) == 0 || len(indice.Idiomas) > 64 {
		return nil
	}
	patron := regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)
	m := map[string]string{}
	for _, idioma := range indice.Idiomas {
		if !patron.MatchString(idioma.Codigo) {
			return nil
		}
		archivo := "textos/" + idioma.Codigo + "/copias-admin.json"
		m["/"+archivo] = archivo
	}
	return m
}
func consultaPublicaCopias(u *url.URL, catalogos map[string]string) bool {
	if u.RawQuery == "" {
		return true
	}
	versiones := map[string]string{
		"/portal-empleado/portal.css":             "20261001-codexf-accesibilidad-v1",
		"/comun/tema-vec.css":                     "20261001-codexf-accesibilidad-v1",
		"/portal-empleado/portal-componentes.css": "20261001-f-cronos-movimientos-v1",
		"/admin/copias/montaje.js":                "20261002-codexk-admin-montaje-v1",
	}
	for _, nombre := range []string{"vista.js", "cliente-http.js", "contratos.js", "estilos.css", "i18n.js", "vista-configuracion.js", "vista-recuperacion.js", "vista-soporte.js"} {
		versiones["/admin/copias/"+nombre] = "20261001-cs09-copias-ux-v2"
	}
	if version, ok := versiones[u.Path]; ok && u.RawQuery == "v="+version {
		return true
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || u.Path != "/admin/copias/" || len(q) != 1 || len(q["lang"]) != 1 {
		return false
	}
	_, ok := catalogos["/textos/"+q.Get("lang")+"/copias-admin.json"]
	return ok
}
