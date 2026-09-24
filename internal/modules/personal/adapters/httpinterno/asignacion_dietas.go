// Package httpinterno expone la asignación de Dietas que conserva Personal.
package httpinterno

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

const RutaAsignacionesDietas = "/api/vec/personal/asignaciones-dietas"
const maximoCuerpoAsignacion = 4096

type ResolutorIdentidadAsignacionDietas interface {
	ResolverIdentidadAsignacionDietas(context.Context) (core.ContextoActor, error)
}

type CasoUsoAsignacionDietas interface {
	Ejecutar(context.Context, personaldomain.SolicitudAsignacionDietas) (personalports.ResultadoAsignacionDietas, error)
}

type ManejadorAsignacionDietas struct {
	identidades ResolutorIdentidadAsignacionDietas
	casoUso     CasoUsoAsignacionDietas
	auditoria   personalports.RegistradorAuditoriaFronteraAsignacionDietas
}

var ErrManejadorAsignacionNoDisponible = errors.New("personal: manejador de asignación no disponible")

func NuevoManejadorAsignacionDietas(identidades ResolutorIdentidadAsignacionDietas, casoUso CasoUsoAsignacionDietas, auditoria personalports.RegistradorAuditoriaFronteraAsignacionDietas) (*ManejadorAsignacionDietas, error) {
	if nuloAsignacionHTTP(identidades) || nuloAsignacionHTTP(casoUso) || nuloAsignacionHTTP(auditoria) {
		return nil, ErrManejadorAsignacionNoDisponible
	}
	return &ManejadorAsignacionDietas{identidades: identidades, casoUso: casoUso, auditoria: auditoria}, nil
}

func (m *ManejadorAsignacionDietas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if r == nil || m == nil || nuloAsignacionHTTP(m.auditoria) {
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	actor := &actorAuditoriaAsignacion{}
	r = r.WithContext(context.WithValue(r.Context(), claveActorAuditoriaAsignacion{}, actor))
	respuesta := &respuestaRetenidaAsignacion{cabeceras: make(http.Header)}
	m.servir(respuesta, r)
	if respuesta.estado >= 400 && m.auditarRechazo(r, respuesta.estado, actor) != nil {
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	for nombre, valores := range respuesta.cabeceras {
		for _, valor := range valores {
			w.Header().Add(nombre, valor)
		}
	}
	w.WriteHeader(respuesta.estado)
	_, _ = w.Write(respuesta.cuerpo.Bytes())
}

type claveActorAuditoriaAsignacion struct{}
type actorAuditoriaAsignacion struct {
	ref     string
	recurso string
}

type respuestaRetenidaAsignacion struct {
	cabeceras http.Header
	cuerpo    bytes.Buffer
	estado    int
}

func (r *respuestaRetenidaAsignacion) Header() http.Header { return r.cabeceras }
func (r *respuestaRetenidaAsignacion) WriteHeader(estado int) {
	if r.estado == 0 {
		r.estado = estado
	}
}
func (r *respuestaRetenidaAsignacion) Write(b []byte) (int, error) {
	if r.estado == 0 {
		r.estado = http.StatusOK
	}
	return r.cuerpo.Write(b)
}

func (m *ManejadorAsignacionDietas) auditarRechazo(r *http.Request, estado int, actor *actorAuditoriaAsignacion) error {
	if actor == nil {
		return ErrManejadorAsignacionNoDisponible
	}
	ruta := personalports.RutaFronteraAsignacionDetalle
	accion := "metodo_no_admitido"
	recurso := actor.recurso
	if r.URL != nil {
		switch {
		case r.URL.Path == RutaAsignacionesDietas:
			ruta = personalports.RutaFronteraAsignacionesDietas
			if r.Method == http.MethodPost {
				accion = "registrar_inicial"
			}
		case strings.HasSuffix(r.URL.Path, "/grupo"):
			ruta = personalports.RutaFronteraAsignacionGrupo
			if referencia, _, valida := rutaAsignacion(r.URL); valida {
				recurso = referencia
			}
			if r.Method == http.MethodPut {
				accion = "grupo_corregir"
			}
		default:
			if referencia, _, valida := rutaAsignacion(r.URL); valida {
				recurso = referencia
			}
			if r.Method == http.MethodGet {
				accion = "consultar"
			} else if r.Method == http.MethodPut {
				accion = "corregir"
			}
		}
	}
	var motivo string
	switch estado {
	case http.StatusBadRequest:
		motivo = personalports.MotivoFronteraPersonalPeticion
	case http.StatusUnauthorized:
		motivo = personalports.MotivoFronteraPersonalAutenticacion
	case http.StatusForbidden:
		motivo = personalports.MotivoFronteraPersonalDenegado
	case http.StatusNotFound:
		motivo = personalports.MotivoFronteraPersonalNoEncontrada
	case http.StatusMethodNotAllowed:
		motivo = personalports.MotivoFronteraPersonalMetodo
	case http.StatusNotAcceptable:
		motivo = personalports.MotivoFronteraPersonalRepresentacion
	case http.StatusConflict:
		motivo = personalports.MotivoFronteraPersonalConflicto
	case http.StatusServiceUnavailable:
		motivo = personalports.MotivoFronteraPersonalDependencia
	default:
		return ErrManejadorAsignacionNoDisponible
	}
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	orden := personalports.OrdenAuditoriaFronteraAsignacionDietas{CorrelacionRef: correlacion, Motivo: motivo, Ruta: ruta, Accion: accion, ActorRef: actor.ref, RecursoRef: recurso, EstadoHTTP: estado}
	if orden.Validar() != nil {
		return ErrManejadorAsignacionNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	return m.auditoria.RegistrarAuditoriaFronteraAsignacionDietas(ctx, orden)
}

func (m *ManejadorAsignacionDietas) servir(w http.ResponseWriter, r *http.Request) {
	if w == nil || r == nil || m == nil || nuloAsignacionHTTP(m.identidades) || nuloAsignacionHTTP(m.casoUso) {
		responderAsignacion(w, http.StatusServiceUnavailable, map[string]string{"error": "personal.error.no_disponible"})
		return
	}
	if cabecerasLibresAsignacion(r.Header) {
		responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	if r.URL != nil && r.URL.Path == RutaAsignacionesDietas && r.URL.EscapedPath() == r.URL.Path {
		m.atenderAltaInicial(w, r)
		return
	}
	relacion, grupo, bien := rutaAsignacion(r.URL)
	if !bien {
		responderErrorAsignacion(w, http.StatusNotFound, "no_encontrada")
		return
	}
	if r.Header.Get("Accept") != "application/json" {
		responderErrorAsignacion(w, http.StatusNotAcceptable, "representacion_no_admitida")
		return
	}
	var solicitud personaldomain.SolicitudAsignacionDietas
	solicitud.RelacionRef = relacion
	switch r.Method {
	case http.MethodGet:
		if grupo || !sinCuerpoAsignacion(r) {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		fecha, unidad, bien := consultaAsignacion(r.URL)
		if !bien {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		solicitud.FechaReferencia = fecha
		solicitud.UnidadRef = unidad
		solicitud.Operacion = personaldomain.ConsultarAsignacionDietas
	case http.MethodPut:
		if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Content-Type") != "application/json; charset=utf-8" {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		var cuerpo correccionAsignacionJSON
		var persona, empleado string
		var entrada correccionGrupoAsignacionJSON
		err := decodificarAsignacion(w, r, &entrada)
		cuerpo, persona, empleado = entrada.correccionAsignacionJSON, entrada.PersonaRef, entrada.EmpleadoRef
		if err != nil || !personaldomain.ReferenciaPersonaValida(persona) || !personaldomain.ReferenciaEmpleadoValida(empleado) {
			responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
			return
		}
		solicitud = solicitudDesdeCorreccion(cuerpo, relacion, persona, empleado)
		solicitud.Operacion = personaldomain.CorregirAsignacionDietas
		if grupo {
			solicitud.Operacion = personaldomain.CorregirGrupoAsignacionDietas
		}
	default:
		w.Header().Set("Allow", "GET, PUT")
		if grupo {
			w.Header().Set("Allow", "PUT")
		}
		responderErrorAsignacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	actor, bien := m.resolverActor(w, r)
	if !bien {
		return
	}
	solicitud.Actor = actor
	resultado, err := m.casoUso.Ejecutar(r.Context(), solicitud)
	if err != nil {
		responderErrorOperacionAsignacion(w, err)
		return
	}
	estado := http.StatusOK
	if r.Method == http.MethodPut && resultado.EstadoLocal == "registrada" {
		estado = http.StatusCreated
	}
	responderAsignacion(w, estado, resultado)
}

func (m *ManejadorAsignacionDietas) atenderAltaInicial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		responderErrorAsignacion(w, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if r.URL.RawQuery != "" || r.URL.ForceQuery || r.Header.Get("Content-Type") != "application/json; charset=utf-8" || r.Header.Get("Accept") != "application/json" {
		responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	var entrada altaInicialAsignacionJSON
	if err := decodificarAsignacion(w, r, &entrada); err != nil || !personaldomain.ReferenciaRelacionValida(entrada.RelacionRef) || !personaldomain.ReferenciaPersonaValida(entrada.PersonaRef) || !personaldomain.ReferenciaEmpleadoValida(entrada.EmpleadoRef) {
		responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
		return
	}
	if registro, ok := r.Context().Value(claveActorAuditoriaAsignacion{}).(*actorAuditoriaAsignacion); ok {
		registro.recurso = entrada.RelacionRef
	}
	solicitud := solicitudDesdeCorreccion(entrada.correccionGrupoAsignacionJSON.correccionAsignacionJSON, entrada.RelacionRef, entrada.PersonaRef, entrada.EmpleadoRef)
	solicitud.Operacion = personaldomain.RegistrarInicialAsignacionDietas
	actor, bien := m.resolverActor(w, r)
	if !bien {
		return
	}
	solicitud.Actor = actor
	resultado, err := m.casoUso.Ejecutar(r.Context(), solicitud)
	if err != nil {
		responderErrorOperacionAsignacion(w, err)
		return
	}
	estado := http.StatusCreated
	if resultado.EstadoLocal == "replay_confirmado" {
		estado = http.StatusOK
	}
	responderAsignacion(w, estado, resultado)
}

func (m *ManejadorAsignacionDietas) resolverActor(w http.ResponseWriter, r *http.Request) (core.ContextoActor, bool) {
	actor, err := m.identidades.ResolverIdentidadAsignacionDietas(r.Context())
	if actor.Validar() == nil {
		if registro, ok := r.Context().Value(claveActorAuditoriaAsignacion{}).(*actorAuditoriaAsignacion); ok {
			registro.ref = actor.Principal.ID
		}
	}
	if err == nil && actor.Validar() == nil {
		return actor, true
	}
	switch {
	case errors.Is(err, personalports.ErrAutenticacionAsignacionDietasRequerida):
		responderErrorAsignacion(w, http.StatusUnauthorized, "autenticacion_requerida")
	case errors.Is(err, personalports.ErrAsignacionDietasDenegada):
		responderErrorAsignacion(w, http.StatusForbidden, "acceso_denegado")
	default:
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
	}
	return core.ContextoActor{}, false
}

func solicitudDesdeCorreccion(c correccionAsignacionJSON, relacion, persona, empleado string) personaldomain.SolicitudAsignacionDietas {
	return personaldomain.SolicitudAsignacionDietas{
		RelacionRef: relacion, PersonaRef: persona, EmpleadoRef: empleado,
		FechaReferencia:   personaldomain.FechaCivil(c.FechaReferencia),
		ClaveIdempotencia: c.ClaveIdempotencia, VersionEsperada: c.VersionEsperada,
		CentroRef: c.CentroRef, UnidadRef: c.UnidadRef,
		AdministrativoPersonaRef: c.AdministrativoPersonaRef, ResponsablePersonaRef: c.ResponsablePersonaRef,
		GrupoDieta: c.GrupoDieta, VigenteDesde: personaldomain.FechaCivil(c.VigenteDesde),
		MotivoRevision: c.MotivoRevision, ProcedenciaActoRef: c.ProcedenciaActoRef,
	}
}

type correccionAsignacionJSON struct {
	FechaReferencia          string `json:"fecha_referencia"`
	ClaveIdempotencia        string `json:"clave_idempotencia"`
	VersionEsperada          int64  `json:"version_esperada"`
	CentroRef                string `json:"centro_ref"`
	UnidadRef                string `json:"unidad_ref"`
	AdministrativoPersonaRef string `json:"administrativo_persona_ref"`
	ResponsablePersonaRef    string `json:"responsable_persona_ref"`
	GrupoDieta               int16  `json:"grupo_dieta"`
	VigenteDesde             string `json:"vigente_desde"`
	MotivoRevision           string `json:"motivo_revision"`
	ProcedenciaActoRef       string `json:"procedencia_acto_ref"`
}

type correccionGrupoAsignacionJSON struct {
	correccionAsignacionJSON
	PersonaRef  string `json:"persona_ref"`
	EmpleadoRef string `json:"empleado_ref"`
}

type altaInicialAsignacionJSON struct {
	correccionGrupoAsignacionJSON
	RelacionRef string `json:"relacion_ref"`
}

func rutaAsignacion(u *url.URL) (string, bool, bool) {
	if u == nil || u.EscapedPath() != u.Path || !strings.HasPrefix(u.Path, RutaAsignacionesDietas+"/") {
		return "", false, false
	}
	resto := strings.TrimPrefix(u.Path, RutaAsignacionesDietas+"/")
	grupo := strings.HasSuffix(resto, "/grupo")
	if grupo {
		resto = strings.TrimSuffix(resto, "/grupo")
	}
	if !personaldomain.ReferenciaRelacionValida(resto) {
		return "", false, false
	}
	return resto, grupo, true
}

func consultaAsignacion(u *url.URL) (personaldomain.FechaCivil, string, bool) {
	if u == nil || u.ForceQuery {
		return "", "", false
	}
	valores, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(valores) != 2 || len(valores["fecha_referencia"]) != 1 || len(valores["unidad_ref"]) != 1 {
		return "", "", false
	}
	fecha, err := personaldomain.NuevaFechaCivil(valores.Get("fecha_referencia"))
	unidad := valores.Get("unidad_ref")
	return fecha, unidad, err == nil && unidad != "" && len(unidad) <= 256
}

func decodificarAsignacion(w http.ResponseWriter, r *http.Request, destino any) error {
	if r.Body == nil || r.Body == http.NoBody || r.ContentLength < 1 || r.ContentLength > maximoCuerpoAsignacion || len(r.TransferEncoding) != 0 {
		return errors.New("cuerpo invalido")
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, maximoCuerpoAsignacion))
	d.DisallowUnknownFields()
	if err := d.Decode(destino); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("contenido adicional")
	}
	return nil
}

func sinCuerpoAsignacion(r *http.Request) bool {
	return r.ContentLength == 0 && len(r.TransferEncoding) == 0 && (r.Body == nil || r.Body == http.NoBody) && r.Header.Get("Content-Type") == ""
}

func cabecerasLibresAsignacion(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		minusculas := strings.ToLower(nombre)
		if minusculas == "cookie" || strings.HasPrefix(minusculas, "x-vec-") {
			return true
		}
	}
	return false
}

func responderErrorOperacionAsignacion(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, personaldomain.ErrAsignacionDietasInvalida), errors.Is(err, personalports.ErrSolicitudAsignacionDietasInvalida):
		responderErrorAsignacion(w, http.StatusBadRequest, "peticion_invalida")
	case errors.Is(err, personalports.ErrAsignacionDietasDenegada):
		responderErrorAsignacion(w, http.StatusForbidden, "acceso_denegado")
	case errors.Is(err, personalports.ErrVersionAsignacionDietas), errors.Is(err, personalports.ErrClaveAsignacionDietas):
		responderErrorAsignacion(w, http.StatusConflict, "conflicto")
	default:
		responderErrorAsignacion(w, http.StatusServiceUnavailable, "no_disponible")
	}
}

func responderErrorAsignacion(w http.ResponseWriter, estado int, codigo string) {
	responderAsignacion(w, estado, map[string]string{"error": "personal.error." + codigo})
}

func responderAsignacion(w http.ResponseWriter, estado int, valor any) {
	if w == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(valor)
}

func nuloAsignacionHTTP(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}

var _ http.Handler = (*ManejadorAsignacionDietas)(nil)
