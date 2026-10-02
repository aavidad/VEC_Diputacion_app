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
	errCuerpoExcesivo          = errors.New("administracion de perfiles: cuerpo excesivo")
)

// SesionConfiable procede exclusivamente de la frontera mTLS y de la sesión
// ligada al perfil activo. Resolverla no concede permiso: cada lectura y acto
// requiere la decisión V3 actual de su propia autoridad.
type SesionConfiable struct {
	Actor                   domain.ContextoActor
	Evidencia               domain.EvidenciaSesionAdministracionPerfiles `json:"-"`
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
	RegistrarDenegacionADMIN(context.Context, DenegacionADMIN) error
}

type DenegacionADMIN struct {
	Codigo          string
	Accion          string
	RecursoRef      string
	ActorPersonaRef string
	PerfilActivoRef string
	CorrelacionRef  string
}

// FuenteLecturas debe aplicar V3, auditar cada lectura y denegación y devolver
// solo datos de la persona solicitada. La consulta de propuesta recupera los
// participantes durables también tras cerrar para recuperar el mismo recibo;
// el navegador nunca los proporciona. PuedeCerrar es solo una pista de UI:
// la autoridad durable decide y audita cada cierre y cada replay.
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
		r.Host != h.host || hayCabecera(r, "Authorization") || hayCabecera(r, "Cookie") ||
		hayCabecera(r, "Proxy-Authorization") ||
		hayCabecera(r, "X-Forwarded-Client-Cert") || hayCabecera(r, "X-Client-Cert") ||
		hayCabecera(r, "X-SSL-Client-Cert") || hayCabecera(r, "X-Remote-User") {
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
	if r.Method == http.MethodPost && (len(r.Header.Values("Origin")) != 1 || r.Header.Get("Origin") != h.origen ||
		len(r.Header.Values("Sec-Fetch-Site")) != 1 ||
		r.Header.Get("Sec-Fetch-Site") != "same-origin" ||
		len(r.Header.Values("Sec-Fetch-Mode")) != 1 ||
		(r.Header.Get("Sec-Fetch-Mode") != "cors" && r.Header.Get("Sec-Fetch-Mode") != "same-origin") ||
		len(r.Header.Values("Sec-Fetch-Dest")) != 1 ||
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
	if sesion.Actor.Validar() != nil || sesion.Evidencia.ValidarPara(sesion.Actor) != nil ||
		sesion.InstantaneaAutorizacion.Validar() != nil ||
		sesion.Actor.PersonaRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		sesion.Actor.PerfilActivoRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(sesion.CorrelacionRef) {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if r.Method == http.MethodGet {
		h.get(w, r, sesion)
		return
	}
	h.post(w, r, sesion)
}

func (h *Handler) origenValido(r *http.Request) bool {
	return len(r.Header.Values("Origin")) <= 1 &&
		(r.Header.Get("Origin") == "" || r.Header.Get("Origin") == h.origen) &&
		len(r.Header.Values("Sec-Fetch-Site")) <= 1 &&
		(r.Header.Get("Sec-Fetch-Site") == "" || r.Header.Get("Sec-Fetch-Site") == "same-origin")
}

func hayCabecera(r *http.Request, nombre string) bool { return len(r.Header.Values(nombre)) != 0 }

func (h *Handler) get(w http.ResponseWriter, r *http.Request, s SesionConfiable) {
	actor := s.Actor
	if r.Body != nil && r.ContentLength > 0 {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "consultar", "")
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
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "buscar_personas", "")
			return
		}
		result, err = h.lecturas.BuscarPersonas(ctx, actor, q.Get("q"), q.Get("cursor"))
	case strings.HasPrefix(p, PrefijoV1+"/personas/") && r.URL.RawQuery == "":
		ref := strings.TrimPrefix(p, PrefijoV1+"/personas/")
		if !refOpaca(ref, "per_") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "consultar_persona", "")
			return
		}
		result, err = h.lecturas.ConsultarPersona(ctx, actor, ref)
	case p == PrefijoV1+"/propuestas" && (r.URL.RawQuery == "" || r.URL.RawQuery == "estado=pendiente"):
		result, err = h.lecturas.ListarPropuestas(ctx, actor)
	case strings.HasPrefix(p, PrefijoV1+"/recibos/") && r.URL.RawQuery == "":
		ref := strings.TrimPrefix(p, PrefijoV1+"/recibos/")
		if !domain.ReferenciaAdministracionPerfilesValida(ref, "recibo_admin:") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "consultar_recibo", "")
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
		h.denegarActor(w, r, s, http.StatusNotFound, "recurso_no_encontrado", "consultar", "")
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
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "escribir", "")
		return
	}
	p := r.URL.Path
	if r.URL.RawQuery != "" {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "escribir", "")
		return
	}
	switch {
	case p == PrefijoV1+"/actos-ordinarios" || p == PrefijoV1+"/propuestas":
		var dto SolicitudActo
		if err := decodificar(w, r, &dto); err != nil {
			estado := http.StatusBadRequest
			if errors.Is(err, errCuerpoExcesivo) {
				estado = http.StatusRequestEntityTooLarge
			}
			h.denegarActor(w, r, s, estado, "solicitud_invalida", "escribir", "")
			return
		}
		prefijoOperacion := "acto_admin:"
		accion := "aplicar_ordinario"
		if p == PrefijoV1+"/propuestas" {
			prefijoOperacion, accion = "propuesta_admin:", "proponer"
		}
		if !domain.ReferenciaAdministracionPerfilesValida(dto.OperacionRef, prefijoOperacion) {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
			return
		}
		clase, err := h.catalogo.ResolverRolAdministrable(r.Context(), dto.RolVersionRef)
		if err != nil {
			falloError(w, err)
			return
		}
		solicitud := domain.SolicitudActoAdministracionPerfiles{ReferenciaActo: dto.ReferenciaActo, OperacionRef: dto.OperacionRef,
			Actor: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Operacion: domain.OperacionAdministracionPerfiles(dto.Operacion), Clase: clase.Clase,
			RolVersionRef: dto.RolVersionRef, Objetivo: dto.Objetivo.dominio(),
			Motivo: dto.Motivo.dominio(), CorrelacionRef: s.CorrelacionRef}
		if solicitud.Validar() != nil || (clase.UnidadRequerida && solicitud.Objetivo.UnidadRef == "") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
			return
		}
		if p == PrefijoV1+"/actos-ordinarios" {
			if clase.Clase != domain.ClaseControlPerfilOrdinario {
				h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
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
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
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
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "cerrar_propuesta", "")
			return
		}
		var dto SolicitudCierre
		if err := decodificar(w, r, &dto); err != nil {
			estado := http.StatusBadRequest
			if errors.Is(err, errCuerpoExcesivo) {
				estado = http.StatusRequestEntityTooLarge
			}
			h.denegarActor(w, r, s, estado, "solicitud_invalida", "cerrar_propuesta", ref)
			return
		}
		propuesta, err := h.lecturas.ConsultarPropuesta(r.Context(), s.Actor, ref)
		if err != nil {
			falloError(w, err)
			return
		}
		if propuesta.PropuestaRef != ref || propuesta.HuellaSHA256 != dto.PropuestaHuellaSHA256 {
			h.denegarActor(w, r, s, http.StatusConflict, "conflicto_estado", "cerrar_propuesta", ref)
			return
		}
		solicitud := domain.SolicitudCierrePropuestaAdministracionPerfiles{
			OperacionRef: dto.OperacionRef, PropuestaRef: ref, PropuestaHuellaSHA256: dto.PropuestaHuellaSHA256,
			ProponentePersonaRef: propuesta.ProponentePersonaRef, ObjetivoPersonaRef: propuesta.ObjetivoPersonaRef,
			Aprobador: s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
			Decision: domain.DecisionPropuestaAdministracionPerfiles(dto.Decision), Motivo: dto.Motivo.dominio(),
			CorrelacionRef: s.CorrelacionRef}
		if solicitud.Validar() != nil {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "cerrar_propuesta", ref)
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
		h.denegarActor(w, r, s, http.StatusNotFound, "recurso_no_encontrado", "escribir", "")
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

func decodificar(w http.ResponseWriter, r *http.Request, destino any) error {
	if r.ContentLength > 16*1024 {
		return errCuerpoExcesivo
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(destino); err != nil {
		var exceso *http.MaxBytesError
		if errors.As(err, &exceso) {
			return errCuerpoExcesivo
		}
		return err
	}
	var sobrante any
	if err := dec.Decode(&sobrante); err != io.EOF {
		var exceso *http.MaxBytesError
		if errors.As(err, &exceso) {
			return errCuerpoExcesivo
		}
		return ErrConfiguracionIncompleta
	}
	return nil
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
	if r == nil || h.auditor.RegistrarDenegacionADMIN(r.Context(), DenegacionADMIN{Codigo: codigo}) != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	fallo(w, estado, codigo)
}

func (h *Handler) denegarActor(w http.ResponseWriter, r *http.Request, s SesionConfiable,
	estado int, codigo, accion, recurso string) {
	registro := DenegacionADMIN{Codigo: codigo, Accion: accion, RecursoRef: recurso,
		ActorPersonaRef: s.Actor.PersonaRef, PerfilActivoRef: s.Actor.PerfilActivoRef,
		CorrelacionRef: s.CorrelacionRef}
	if h.auditor.RegistrarDenegacionADMIN(r.Context(), registro) != nil {
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
