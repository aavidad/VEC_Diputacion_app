package httpcopias

import (
	"context"
	"net/http"
	"net/url"
	"time"

	app "vec-diputacion-granada/internal/modules/administracion/application/httpcopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	d "vec-diputacion-granada/internal/vec/domain"
)

const PrefijoV1 = "/api/admin/copias/v1"
const maxCuerpo = 16 * 1024

type ResolverSesion func(context.Context, *http.Request) (p.Sesion, error)
type Denegacion struct{ Codigo, Accion, RecursoRef, ActorPersonaRef, PerfilActivoRef, CorrelacionRef string }

// AuditorFrontera debe conservar denegados y errores mediante la autoridad
// común, con un plazo propio acotado incluso si el cliente se desconecta.
// Antes de sesión utiliza la frontera sin fabricar identidad nominal.
type AuditorFrontera func(context.Context, Denegacion) error

// Existing ADMIN composition must recheck CA/CRL/network/session before invoking this handler.
type Handler struct {
	origen, host string
	resolver     ResolverSesion
	auditor      AuditorFrontera
	servicio     *app.Servicio
}

func Nuevo(origen string, resolver ResolverSesion, auditor AuditorFrontera, servicio *app.Servicio) (*Handler, error) {
	u, err := url.Parse(origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || resolver == nil || auditor == nil || servicio == nil || app.Ausente(servicio.Autoridad) {
		return nil, p.ErrNoDisponible
	}
	return &Handler{origen: origen, host: u.Host, resolver: resolver, auditor: auditor, servicio: servicio}, nil
}
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.resolver == nil || h.auditor == nil || h.servicio == nil {
		fallo(w, p.ErrNoDisponible)
		return
	}
	if r == nil {
		fallo(w, p.ErrNoDisponible)
		return
	}
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || r.Host != h.host {
		h.denegar(w, r, p.Sesion{}, p.ErrAutenticacion, "", "")
		return
	}
	for _, key := range []string{"Authorization", "Cookie", "Proxy-Authorization", "X-Forwarded-Client-Cert", "X-Client-Cert", "X-SSL-Client-Cert", "X-Remote-User"} {
		if len(r.Header.Values(key)) != 0 {
			h.denegar(w, r, p.Sesion{}, p.ErrAutenticacion, "", "")
			return
		}
	}
	if len(r.Header.Values("Origin")) > 1 || r.Header.Get("Origin") != "" && r.Header.Get("Origin") != h.origen || len(r.Header.Values("Sec-Fetch-Site")) > 1 || r.Header.Get("Sec-Fetch-Site") != "" && r.Header.Get("Sec-Fetch-Site") != "same-origin" {
		h.denegar(w, r, p.Sesion{}, p.ErrDenegado, "", "")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.denegarStatus(w, r, p.Sesion{}, 405, "metodo_no_permitido", "", "")
		return
	}
	if r.Method == http.MethodPost {
		for key, want := range map[string]string{"Origin": h.origen, "Sec-Fetch-Site": "same-origin", "Sec-Fetch-Dest": "empty"} {
			if len(r.Header.Values(key)) != 1 || r.Header.Get(key) != want {
				h.denegar(w, r, p.Sesion{}, p.ErrDenegado, "", "")
				return
			}
		}
		if len(r.Header.Values("Sec-Fetch-Mode")) != 1 || r.Header.Get("Sec-Fetch-Mode") != "cors" && r.Header.Get("Sec-Fetch-Mode") != "same-origin" {
			h.denegar(w, r, p.Sesion{}, p.ErrDenegado, "", "")
			return
		}
	}
	ses, err := h.resolver(r.Context(), r)
	if err != nil {
		h.denegar(w, r, p.Sesion{}, err, "", "")
		return
	}
	if ses.Actor.Validar() != nil || !ses.Actor.Instantanea.VigenteEn(time.Now().UTC().Truncate(time.Microsecond)) || ses.Instantanea.Validar() != nil || !ses.Instantanea.AsignacionPerfil.VigenteEn(time.Now()) || ses.Instantanea.ControlVigenciaVersionRol.Estado != d.EstadoControlVigenciaVersionRolHabilitada || ses.Actor.Principal.AuthMethod != d.AuthMethodCertificate || ses.Actor.Principal.AuthAssurance != d.AuthAssuranceHigh || ses.Actor.PersonaRef != ses.Instantanea.AsignacionPerfil.PrincipalID || ses.Actor.PerfilActivoRef != ses.Instantanea.AsignacionPerfil.PerfilActivoRef || !d.ReferenciaCorrelacionAutorizacionV2Valida(ses.CorrelacionRef) {
		h.denegar(w, r, p.Sesion{}, p.ErrNoDisponible, "", "")
		return
	}
	if r.Context().Err() != nil {
		h.denegar(w, r, ses, p.ErrNoDisponible, "", "")
		return
	}
	if r.Method == http.MethodGet {
		h.get(w, r, ses)
		return
	}
	h.post(w, r, ses)
}
func (h *Handler) autorizado(w http.ResponseWriter, r *http.Request, s p.Sesion, op p.Operacion, ref string, dependency any) bool {
	if err := h.servicio.Autorizar(r.Context(), s, op, ref); err != nil {
		h.denegar(w, r, s, err, string(op), ref)
		return false
	}
	if r.Context().Err() != nil || app.Ausente(dependency) {
		h.denegar(w, r, s, p.ErrNoDisponible, string(op), ref)
		return false
	}
	return true
}
