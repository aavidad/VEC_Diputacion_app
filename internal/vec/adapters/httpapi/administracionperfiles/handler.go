package administracionperfiles

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

// AuditorFrontera distingue la fase técnica previa a V2 de la nominal.
// Una sesión resuelta incompatible nunca permite volver a la fase técnica.
type AuditorFrontera interface {
	RegistrarDenegacionADMIN(context.Context, DenegacionADMIN) error
}

type DenegacionADMIN struct {
	SesionResuelta  bool `json:"-"`
	Codigo          string
	Accion          string
	RecursoRef      string
	ActorPersonaRef string
	PerfilActivoRef string
	CorrelacionRef  string
	Actor           domain.ContextoActor                         `json:"-"`
	Evidencia       domain.EvidenciaSesionAdministracionPerfiles `json:"-"`
}

// FuenteLecturas debe resolver el catálogo central del actor y admitir solo
// administración de Aplicación. Sistemas queda denegado; no se infiere la
// clase desde el cliente ni desde el nombre del perfil. Debe aplicar V3,
// auditar cada lectura y denegación en la transacción común y devolver
// solo datos de la persona solicitada. La consulta de propuesta recupera los
// participantes durables también tras cerrar para recuperar el mismo recibo;
// el navegador nunca los proporciona. PuedeCerrar es solo una pista de UI:
// la autoridad durable decide y audita cada cierre y cada replay.
type FuenteLecturas interface {
	Capacidades(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (Capacidades, error)
	BuscarPersonas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, ConsultaPersonas) (PaginaPersonas, error)
	ConsultarPersona(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (FichaPersona, error)
	ListarRoles(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (Roles, error)
	ListarPropuestas(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles) (PaginaPropuestas, error)
	ConsultarPropuesta(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (Propuesta, error)
	ConsultarRecibo(context.Context, domain.ContextoActor, domain.EvidenciaSesionAdministracionPerfiles, string) (domain.ReciboAdministracionPerfiles, error)
}

type ServicioActos interface {
	AplicarOrdinario(context.Context, domain.SolicitudActoAdministracionPerfiles) (domain.ReciboAdministracionPerfiles, error)
	ProponerSensible(context.Context, domain.SolicitudActoAdministracionPerfiles) (ports.PropuestaAdministracionPerfiles, error)
	CerrarPropuestaSensible(context.Context, domain.SolicitudCierrePropuestaAdministracionPerfiles) (ports.CierrePropuestaAdministracionPerfiles, error)
}

// ServicioLotes amplía opcionalmente el servicio existente. Su ausencia
// mantiene cerrado el endpoint de lotes sin recurrir a escrituras singulares.
type ServicioLotes interface {
	AplicarLoteOrdinario(context.Context, domain.SolicitudLoteAdministracionPerfiles) (domain.ReciboLoteAdministracionPerfiles, error)
}

// Handler queda inyectable; ningún proceso lo monta en este corte.
type Handler struct {
	origen           string
	host             string
	organizacionLote string
	sesiones         ResolvedorSesion
	lecturas         FuenteLecturas
	catalogo         ports.CatalogoRolesAdministrables
	actos            ServicioActos
	lotes            ServicioLotesADMIN
	motivosLote      []MotivoLote
	soloLectura      bool
	soloMetadatos    bool
	auditor          AuditorFrontera
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

// NuevoHandlerLecturas exige la fuente autorizada y la auditoría; conserva
// cerradas las escrituras mientras no existe una autoridad durable de actos.
func NuevoHandlerLecturas(origen string, sesiones ResolvedorSesion, lecturas FuenteLecturas, auditor AuditorFrontera) (*Handler, error) {
	u, err := url.Parse(origen)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || dependenciaNula(sesiones) || dependenciaNula(lecturas) || dependenciaNula(auditor) {
		return nil, ErrConfiguracionIncompleta
	}
	return &Handler{origen: origen, host: u.Host, sesiones: sesiones, lecturas: lecturas, auditor: auditor, soloLectura: true}, nil
}

// Sólo las dos consultas nominales de usuarios: las demás rutas se rechazan
// en la frontera auditada sin ejecutar fuentes o fabricar capacidades.
func NuevoHandlerUsuariosMetadatos(origen string, sesiones ResolvedorSesion, lecturas FuenteLecturas, auditor AuditorFrontera) (*Handler, error) {
	h, err := NuevoHandlerLecturas(origen, sesiones, lecturas, auditor)
	if err != nil {
		return nil, err
	}
	h.soloMetadatos = true
	return h, nil
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
	if h == nil || h.sesiones == nil || h.lecturas == nil || (!h.soloLectura && (h.catalogo == nil || (h.actos == nil && h.lotes == nil))) || h.auditor == nil {
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
			h.denegar(w, r, http.StatusServiceUnavailable, "servicio_no_disponible")
		}
		return
	}
	if sesion.Actor.Validar() != nil || sesion.Evidencia.ValidarPara(sesion.Actor) != nil ||
		sesion.InstantaneaAutorizacion.Validar() != nil ||
		sesion.Actor.PersonaRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PrincipalID ||
		sesion.Actor.PerfilActivoRef != sesion.InstantaneaAutorizacion.AsignacionPerfil.PerfilActivoRef ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(sesion.CorrelacionRef) {
		h.denegarSesionIncompatible(w, r, sesion)
		return
	}
	if r.Method == http.MethodGet {
		h.get(w, r, sesion)
		return
	}
	if h.soloLectura {
		h.denegarActor(w, r, sesion, http.StatusServiceUnavailable, "servicio_no_disponible", "escribir", "")
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
	if r.Body != nil && r.ContentLength != 0 {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "consultar", "")
		return
	}
	p := r.URL.Path
	if h.soloMetadatos && p != PrefijoV1+"/personas" && !strings.HasPrefix(p, PrefijoV1+"/personas/") {
		h.denegarActor(w, r, s, http.StatusNotFound, "recurso_no_encontrado", "consultar", "")
		return
	}
	// La preparación sólo existe donde se montó la autoridad del lote.
	if persona, ok := rutaPreparacionLote(p); ok && h.lotes != nil {
		h.getPreparacionLote(w, r, s, persona)
		return
	}
	ctx := r.Context()
	var result any
	var err error
	switch {
	case p == PrefijoV1+"/capacidades" && r.URL.RawQuery == "":
		var capacidades Capacidades
		capacidades, err = h.lecturas.Capacidades(ctx, actor, s.Evidencia)
		capacidades.ActorPersonaRef = actor.PersonaRef
		result = capacidades
	case p == PrefijoV1+"/roles" && r.URL.RawQuery == "":
		result, err = h.lecturas.ListarRoles(ctx, actor, s.Evidencia)
	case p == PrefijoV1+"/personas":
		consulta, errConsulta := consultaPersonas(r.URL.RawQuery)
		if errConsulta != nil {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "buscar_personas", "")
			return
		}
		if h.soloMetadatos && consulta.Texto != "" {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "buscar_personas", "")
			return
		}
		result, err = h.lecturas.BuscarPersonas(ctx, actor, s.Evidencia, consulta)
		if pagina, ok := result.(PaginaPersonas); err == nil && ok && (h.soloMetadatos && pagina.Metadatos == nil || !pagina.metadatosValidos()) {
			h.denegarActor(w, r, s, http.StatusServiceUnavailable, "respuesta_incompatible", "buscar_personas", "")
			return
		}
	case strings.HasPrefix(p, PrefijoV1+"/personas/") && r.URL.RawQuery == "":
		ref := strings.TrimPrefix(p, PrefijoV1+"/personas/")
		if !refOpaca(ref, "per_") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "consultar_persona", "")
			return
		}
		var ficha FichaPersona
		ficha, err = h.lecturas.ConsultarPersona(ctx, actor, s.Evidencia, ref)
		if err == nil && ficha.referenciaEmitida() != ref {
			if h.soloMetadatos || ficha.Metadatos != nil {
				h.denegarActor(w, r, s, http.StatusServiceUnavailable, "respuesta_incompatible", "consultar_persona", ref)
				return
			}
			err = ErrConfiguracionIncompleta
		}
		if err == nil && (h.soloMetadatos && ficha.Metadatos == nil || !ficha.metadatosValidos()) {
			h.denegarActor(w, r, s, http.StatusServiceUnavailable, "respuesta_incompatible", "consultar_persona", ref)
			return
		}
		result = ficha
	case p == PrefijoV1+"/propuestas" && (r.URL.RawQuery == "" || r.URL.RawQuery == "estado=pendiente"):
		result, err = h.lecturas.ListarPropuestas(ctx, actor, s.Evidencia)
	case strings.HasPrefix(p, PrefijoV1+"/recibos/") && r.URL.RawQuery == "":
		ref := strings.TrimPrefix(p, PrefijoV1+"/recibos/")
		if !domain.ReferenciaAdministracionPerfilesValida(ref, "recibo_admin:") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "consultar_recibo", "")
			return
		}
		var recibo domain.ReciboAdministracionPerfiles
		recibo, err = h.lecturas.ConsultarRecibo(ctx, actor, s.Evidencia, ref)
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
		if errors.Is(err, domain.ErrActoAdministracionPerfilesInvalido) {
			fallo(w, http.StatusBadRequest, "solicitud_invalida")
		} else {
			falloError(w, err)
		}
		return
	}
	jsonRespuesta(w, http.StatusOK, result)
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
	return decodificarLimitado(w, r, destino, 16*1024)
}

func decodificarLimitado(w http.ResponseWriter, r *http.Request, destino any, limite int64) error {
	if r.ContentLength > limite {
		return errCuerpoExcesivo
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limite))
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

// El retorno exitoso del resolutor fija la fase aunque su sesión sea inválida.
func (h *Handler) denegarSesionIncompatible(w http.ResponseWriter, r *http.Request, s SesionConfiable) {
	accion := "consultar"
	if r.Method == http.MethodPost {
		accion = "escribir"
	}
	registro := DenegacionADMIN{Codigo: "respuesta_incompatible", Accion: accion, SesionResuelta: true}
	correlacion, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(r.Context())
	if err != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	ref, err := correlacion.ValorCanonico()
	if err != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	registro.CorrelacionRef = ref
	actor, err := s.Actor.Clonar()
	if err == nil {
		resultado, err := s.Evidencia.ResultadoContexto.Clonar()
		if err == nil {
			evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
			if evidencia.ValidarPara(actor) == nil {
				registro.Actor = actor
				registro.Evidencia = evidencia
				registro.ActorPersonaRef = actor.PersonaRef
				registro.PerfilActivoRef = actor.PerfilActivoRef
			}
		}
	}
	if h.auditor.RegistrarDenegacionADMIN(r.Context(), registro) != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	fallo(w, http.StatusServiceUnavailable, "respuesta_incompatible")
}

func (h *Handler) denegarActor(w http.ResponseWriter, r *http.Request, s SesionConfiable,
	estado int, codigo, accion, recurso string) {
	actor, err := s.Actor.Clonar()
	if err != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	resultado, err := s.Evidencia.ResultadoContexto.Clonar()
	if err != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	evidencia := domain.EvidenciaSesionAdministracionPerfiles{ResultadoContexto: resultado, Vinculo: s.Evidencia.Vinculo}
	if evidencia.ValidarPara(actor) != nil {
		fallo(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	registro := DenegacionADMIN{SesionResuelta: true, Codigo: codigo, Accion: accion, RecursoRef: recurso,
		ActorPersonaRef: actor.PersonaRef, PerfilActivoRef: actor.PerfilActivoRef,
		CorrelacionRef: s.CorrelacionRef, Actor: actor, Evidencia: evidencia}
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
