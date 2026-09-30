package administracionperfiles

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConfiguracionIncompleta = errors.New("administracion de perfiles: dependencias incompletas")
	ErrAutenticacionRequerida  = errors.New("administracion de perfiles: autenticacion requerida")
	ErrAccesoDenegado          = errors.New("administracion de perfiles: acceso denegado")
	ErrConflictoEstado         = errors.New("administracion de perfiles: conflicto de estado")
	ErrRecursoNoEncontrado     = errors.New("administracion de perfiles: recurso no encontrado")
)

// SesionConfiable procede exclusivamente de la frontera mTLS y de la sesión
// ligada al perfil activo. Resolverla no concede permiso: cada lectura y acto
// requiere la decisión V3 actual de su propia autoridad.
type SesionConfiable struct {
	Actor                   domain.ContextoActor
	InstantaneaAutorizacion domain.InstantaneaAutorizacion
	CorrelacionRef          string
}

// ResolvedorSesion debe cotejar certificado con CA ADMIN, revocación,
// audiencia, cuenta privilegiada, perfil nominal y sesión vigente. Ningún
// campo de identidad ni capacidad puede proceder de HTTP no confiable.
type ResolvedorSesion interface {
	ResolverSesionADMIN(context.Context, *http.Request) (SesionConfiable, error)
}

// AuditorFrontera conserva denegaciones previas a la identidad sin guardar
// cabeceras, certificado ni cuerpo. Sin recibo de auditoría se responde 503.
type AuditorFrontera interface {
	RegistrarDenegacionADMIN(context.Context, string) error
}

// FuenteLecturas debe aplicar V3, auditar cada lectura y denegación y devolver
// solo datos de la persona solicitada. La consulta de propuesta recupera los
// participantes durables para el cierre; el navegador nunca los proporciona.
type FuenteLecturas interface {
	Capacidades(context.Context, domain.ContextoActor) (Capacidades, error)
	BuscarPersonas(context.Context, domain.ContextoActor, string, string) (PaginaPersonas, error)
	ConsultarPersona(context.Context, domain.ContextoActor, string) (FichaPersona, error)
	ListarRoles(context.Context, domain.ContextoActor) (Roles, error)
	ListarPropuestas(context.Context, domain.ContextoActor) (PaginaPropuestas, error)
	ConsultarPropuesta(context.Context, domain.ContextoActor, string) (Propuesta, error)
	ConsultarRecibo(context.Context, domain.ContextoActor, string) (domain.ReciboAdministracionPerfiles, error)
}

type ServicioActos interface {
	AplicarOrdinario(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error)
	ProponerSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error)
	CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error)
}

// Handler queda inyectable; ningún proceso lo monta en este corte.
type Handler struct {
	origen   string
	host     string
	sesiones ResolvedorSesion
	lecturas FuenteLecturas
	catalogo ports.CatalogoRolesAdministrables
	actos    ServicioActos
	auditor  AuditorFrontera
}

func NuevoHandler(origen string, sesiones ResolvedorSesion, lecturas FuenteLecturas,
	catalogo ports.CatalogoRolesAdministrables, actos ServicioActos, auditor AuditorFrontera) (*Handler, error) {
	u, err := url.Parse(origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" ||
		dependenciaNula(sesiones) || dependenciaNula(lecturas) ||
		dependenciaNula(catalogo) || dependenciaNula(actos) || dependenciaNula(auditor) {
		return nil, ErrConfiguracionIncompleta
	}
	return &Handler{origen: origen, host: u.Host, sesiones: sesiones,
		lecturas: lecturas, catalogo: catalogo, actos: actos, auditor: auditor}, nil
}

func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.sesiones == nil || h.lecturas == nil || h.catalogo == nil || h.actos == nil || h.auditor == nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r == nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 ||
		r.Host != h.host || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" ||
		r.Header.Get("Proxy-Authorization") != "" ||
		r.Header.Get("X-Forwarded-Client-Cert") != "" || r.Header.Get("X-Client-Cert") != "" ||
		r.Header.Get("X-SSL-Client-Cert") != "" || r.Header.Get("X-Remote-User") != "" {
		h.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	if !h.origenValido(r) {
		h.denegar(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		h.denegar(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.Method == http.MethodPost && (r.Header.Get("Origin") != h.origen ||
		r.Header.Get("Sec-Fetch-Site") != "same-origin" ||
		(r.Header.Get("Sec-Fetch-Mode") != "cors" && r.Header.Get("Sec-Fetch-Mode") != "same-origin") ||
		r.Header.Get("Sec-Fetch-Dest") != "empty") {
		h.denegar(w, r, http.StatusForbidden, "acceso_denegado")
		return
	}
	sesion, err := h.sesiones.ResolverSesionADMIN(r.Context(), r)
	if err != nil {
		switch {
		case errors.Is(err, ErrAutenticacionRequerida):
			h.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida")
		case errors.Is(err, ErrAccesoDenegado), errors.Is(err, domain.ErrAutorizacionDenegada):
			h.denegar(w, r, http.StatusForbidden, "acceso_denegado")
		default:
			fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		}
		return
	}
	if sesion.Actor.Validar() != nil || sesion.InstantaneaAutorizacion.Validar() != nil ||
		sesion.Actor.PersonaRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		sesion.Actor.PerfilActivoRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(sesion.CorrelacionRef) {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodGet {
		h.get(w, r, sesion.Actor)
		return
	}
	h.post(w, r, sesion)
}

func (h *Handler) origenValido(r *http.Request) bool {
	return len(r.Header.Values("Origin")) <= 1 &&
		(r.Header.Get("Origin") == "" || r.Header.Get("Origin") == h.origen) &&
		(r.Header.Get("Sec-Fetch-Site") == "" || r.Header.Get("Sec-Fetch-Site") == "same-origin")
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, actor domain.ContextoActor) {
	if r.Body != nil && r.ContentLength > 0 {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	p := r.URL.Path
	ctx := r.Context()
	var result any
	var err error
	switch {
	case p == PrefijoV1+"/capacidades" && r.URL.RawQuery == "":
		var capacidades Capacidades
		capacidades, err = h.lecturas.Capacidades(ctx, actor)
		capacidades.ActorPersonaRef = actor.PersonaRef
		result = capacidades
	case p == PrefijoV1+"/roles" && r.URL.RawQuery == "":
		result, err = h.lecturas.ListarRoles(ctx, actor)
	case p == PrefijoV1+"/personas":
		q := r.URL.Query()
		if len(q) > 2 || (len(q) == 2 && q["cursor"] == nil) || len(q["q"]) != 1 || len(q["q"][0]) < 2 || len(q["q"][0]) > 80 ||
			len(q["cursor"]) > 1 || len(q["cursor"]) == 1 && len(q["cursor"][0]) > 256 {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		result, err = h.lecturas.BuscarPersonas(ctx, actor, q.Get("q"), q.Get("cursor"))
	case strings.HasPrefix(p, PrefijoV1+"/personas/") && r.URL.RawQuery == "":
		ref := strings.TrimPrefix(p, PrefijoV1+"/personas/")
		if !refOpaca(ref, "per_") {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		result, err = h.lecturas.ConsultarPersona(ctx, actor, ref)
	case p == PrefijoV1+"/propuestas" && (r.URL.RawQuery == "" || r.URL.RawQuery == "estado=pendiente"):
		result, err = h.lecturas.ListarPropuestas(ctx, actor)
	case strings.HasPrefix(p, PrefijoV1+"/recibos/") && r.URL.RawQuery == "":
		ref := strings.TrimPrefix(p, PrefijoV1+"/recibos/")
		if !domain.ReferenciaAdministracionPerfilesValida(ref, "recibo_admin:") {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		var recibo domain.ReciboAdministracionPerfiles
		recibo, err = h.lecturas.ConsultarRecibo(ctx, actor, ref)
		if err == nil && (recibo.Validar() != nil || recibo.ReciboRef != ref) {
			err = ErrConfiguracionIncompleta
		}
		result = struct {
			Recibo Recibo `json:"recibo"`
		}{reciboDTO(recibo)}
	default:
		fallo(w, http.StatusNotFound, "recurso_no_encontrado")
		return
	}
	if err != nil {
		falloError(w, err)
		return
	}
	jsonRespuesta(w, http.StatusOK, result)
}

func (h *Handler) post(w http.ResponseWriter, r *http.Request, s SesionConfiable) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	p := r.URL.Path
	if r.URL.RawQuery != "" {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return
	}
	switch {
	case p == PrefijoV1+"/actos-ordinarios" || p == PrefijoV1+"/propuestas":
		var dto SolicitudActo
		if !decodificar(w, r, &dto) {
			return
		}
		clase, err := h.catalogo.ResolverRolAdministrable(r.Context(), dto.RolVersionRef)
		if err != nil {
			falloError(w, err)
			return
		}
		solicitud := domain.SolicitudActoAdministracionPerfiles{OperacionRef: dto.OperacionRef,
			Actor: s.Actor, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Operacion: domain.OperacionAdministracionPerfiles(dto.Operacion), Clase: clase.Clase,
			RolVersionRef: dto.RolVersionRef, Objetivo: dto.Objetivo.dominio(),
			Motivo: dto.Motivo.dominio(), CorrelacionRef: s.CorrelacionRef}
		if solicitud.Validar() != nil {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		if p == PrefijoV1+"/actos-ordinarios" {
			if clase.Clase != domain.ClaseControlPerfilOrdinario {
				fallo(w, http.StatusBadRequest, "solicitud_invalida")
				return
			}
			recibo, err := h.actos.AplicarOrdinario(r.Context(), solicitud)
			if err != nil {
				falloError(w, err)
				return
			}
			if recibo.Validar() != nil || recibo.OperacionRef != dto.OperacionRef {
				fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
				return
			}
			jsonRespuesta(w, http.StatusOK, struct {
				Recibo Recibo `json:"recibo"`
			}{reciboDTO(recibo)})
			return
		}
		if !clase.Clase.RequiereDobleControl() {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		propuesta, err := h.actos.ProponerSensible(r.Context(), solicitud)
		if err != nil {
			falloError(w, err)
			return
		}
		if propuesta.ValidarPara(solicitud) != nil {
			fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		propuestaDTO := struct {
			PropuestaRef string `json:"propuesta_ref"`
			HuellaSHA256 string `json:"huella_sha256"`
			CaducaEn     any    `json:"caduca_en"`
		}{propuesta.PropuestaRef, propuesta.HuellaSHA256, propuesta.CaducaEn}
		jsonRespuesta(w, http.StatusOK, struct {
			Propuesta any `json:"propuesta"`
		}{propuestaDTO})
	case strings.HasPrefix(p, PrefijoV1+"/propuestas/") && strings.HasSuffix(p, "/cierre"):
		ref := strings.TrimSuffix(strings.TrimPrefix(p, PrefijoV1+"/propuestas/"), "/cierre")
		if !domain.ReferenciaAdministracionPerfilesValida(ref, "propuesta_admin:") {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		var dto SolicitudCierre
		if !decodificar(w, r, &dto) {
			return
		}
		propuesta, err := h.lecturas.ConsultarPropuesta(r.Context(), s.Actor, ref)
		if err != nil {
			falloError(w, err)
			return
		}
		if !propuesta.PuedeCerrar {
			fallo(w, http.StatusForbidden, "acceso_denegado")
			return
		}
		if propuesta.PropuestaRef != ref || propuesta.HuellaSHA256 != dto.PropuestaHuellaSHA256 {
			fallo(w, http.StatusConflict, "conflicto_estado")
			return
		}
		solicitud := domain.SolicitudCierrePropuestaAdministracionPerfiles{
			OperacionRef: dto.OperacionRef, PropuestaRef: ref, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
			ProponentePersonaRef: propuesta.ProponentePersonaRef, ObjetivoPersonaRef: propuesta.ObjetivoPersonaRef,
			Aprobador: s.Actor, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Decision: domain.DecisionPropuestaAdministracionPerfiles(dto.Decision), Motivo: dto.Motivo.dominio(),
			CorrelacionRef: s.CorrelacionRef}
		if solicitud.Validar() != nil {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
			return
		}
		cierre, err := h.actos.CerrarPropuestaSensible(r.Context(), solicitud)
		if err != nil {
			falloError(w, err)
			return
		}
		if cierre.ValidarPara(solicitud) != nil {
			fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		var recibo *Recibo
		if cierre.Recibo != nil {
			r := reciboDTO(*cierre.Recibo)
			recibo = &r
		}
		cierreDTO := struct {
			OperacionRef       string  `json:"operacion_ref"`
			PropuestaRef       string  `json:"propuesta_ref"`
			Decision           string  `json:"decision"`
			HuellaCierreSHA256 string  `json:"huella_cierre_sha256"`
			ConfirmadoEn       any     `json:"confirmado_en"`
			Recibo             *Recibo `json:"recibo,omitempty"`
		}{
			cierre.OperacionRef, cierre.PropuestaRef, string(cierre.Decision), cierre.HuellaCierreSHA256, cierre.ConfirmadoEn, recibo}
		jsonRespuesta(w, http.StatusOK, struct {
			Cierre any `json:"cierre"`
		}{cierreDTO})
	default:
		fallo(w, http.StatusNotFound, "recurso_no_encontrado")
	}
}

func refOpaca(v, prefix string) bool {
	if !strings.HasPrefix(v, prefix) || len(v) < len(prefix)+22 || len(v) > len(prefix)+128 {
		return false
	}
	for _, c := range v[len(prefix):] {
		if c < 'a' || c > 'z' {
			if c < 'A' || c > 'Z' {
				if c < '0' || c > '9' {
					if c != '_' && c != '-' {
						return false
					}
				}
			}
		}
	}
	return true
}

func decodificar(w http.ResponseWriter, r *http.Request, destino any) bool {
	if r.ContentLength > 16*1024 {
		fallo(w, http.StatusRequestEntityTooLarge, "solicitud_invalida")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return false
	}
	var sobrante any
	if err := dec.Decode(&sobrante); err != io.EOF {
		fallo(w, http.StatusBadRequest, "solicitud_invalida")
		return false
	}
	return true
}

func falloError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrAutenticacionRequerida):
		fallo(w, http.StatusUnauthorized, "autenticacion_requerida")
	case errors.Is(err, ErrAccesoDenegado), errors.Is(err, domain.ErrAutorizacionDenegada):
		fallo(w, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, ErrConflictoEstado):
		fallo(w, http.StatusConflict, "conflicto_estado")
	case errors.Is(err, ErrRecursoNoEncontrado):
		fallo(w, http.StatusNotFound, "recurso_no_encontrado")
	default:
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
	}
}

func (h *Handler) denegar(w http.ResponseWriter, r *http.Request, estado int, codigo string) {
	if r == nil || h.auditor.RegistrarDenegacionADMIN(r.Context(), codigo) != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	fallo(w, estado, codigo)
}

func fallo(w http.ResponseWriter, estado int, codigo string) {
	jsonRespuesta(w, estado, struct {
		Error struct {
			Codigo    string `json:"codigo"`
			ClaveI18N string `json:"clave_i18n"`
		} `json:"error"`
	}{
		Error: struct {
			Codigo    string `json:"codigo"`
			ClaveI18N string `json:"clave_i18n"`
		}{codigo, "api.admin.perfiles.error." + codigo}})
}

func jsonRespuesta(w http.ResponseWriter, estado int, v any) {
	for _, cabecera := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Access-Control-Allow-Headers", "Access-Control-Allow-Methods", "Access-Control-Expose-Headers", "Location"} {
		w.Header().Del(cabecera)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(v)
}
