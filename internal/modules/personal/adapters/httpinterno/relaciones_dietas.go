package httpinterno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

const RutaRelacionesDietas = "/api/vec/personal/relaciones-dietas"

var ErrManejadorRelacionesDietasNoDisponible = errors.New("personal: lectura de relaciones de dietas no disponible")

type ResolutorIdentidadRelacionesDietas interface {
	ResolverIdentidadRelacionesDietas(context.Context) (core.ContextoActor, personaldomain.FechaCivil, error)
}

type consultorRelacionesDietas interface {
	ConsultarPropiasParaDietas(context.Context, personaldomain.SolicitudConsultaRelacionPropia) (personalports.ResultadoConsultaRelacionPropia, error)
}

type ManejadorRelacionesDietas struct {
	identidad ResolutorIdentidadRelacionesDietas
	consulta  consultorRelacionesDietas
	auditoria personalports.RegistradorAuditoriaFronteraAsignacionDietas
}

func NuevoManejadorRelacionesDietas(identidad ResolutorIdentidadRelacionesDietas, consulta consultorRelacionesDietas, auditoria personalports.RegistradorAuditoriaFronteraAsignacionDietas) (*ManejadorRelacionesDietas, error) {
	if nuloRelacionesDietas(identidad) || nuloRelacionesDietas(consulta) || nuloRelacionesDietas(auditoria) {
		return nil, ErrManejadorRelacionesDietasNoDisponible
	}
	return &ManejadorRelacionesDietas{identidad: identidad, consulta: consulta, auditoria: auditoria}, nil
}

func (m *ManejadorRelacionesDietas) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r == nil || r.URL == nil || m == nil || nuloRelacionesDietas(m.identidad) || nuloRelacionesDietas(m.consulta) || nuloRelacionesDietas(m.auditoria) {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	if r.URL.Path != RutaRelacionesDietas || r.URL.EscapedPath() != r.URL.Path || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		m.denegar(w, r, http.StatusNotFound, "", "", "metodo_no_admitido")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		m.denegar(w, r, http.StatusMethodNotAllowed, "", "", "metodo_no_admitido")
		return
	}
	if cabeceraLibreRelacionesDietas(r.Header) || (r.Body != nil && r.Body != http.NoBody) || len(r.TransferEncoding) != 0 {
		m.denegar(w, r, http.StatusBadRequest, "", "", "consultar_relaciones")
		return
	}
	if !aceptaSoloJSONRelacionesDietas(r.Header) {
		m.denegar(w, r, http.StatusNotAcceptable, "", "", "consultar_relaciones")
		return
	}
	actor, fecha, err := m.identidad.ResolverIdentidadRelacionesDietas(r.Context())
	if err != nil || actor.Validar() != nil || fecha.Validar() != nil {
		// La frontera mTLS distingue 401/403. Una resolución de sesión que
		// falla aquí es dependencia no disponible, nunca una concesión negativa.
		m.denegar(w, r, http.StatusServiceUnavailable, "", "", "consultar_relaciones")
		return
	}
	empleados, err := actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		m.denegar(w, r, http.StatusServiceUnavailable, actor.Principal.ID, "", "consultar_relaciones")
		return
	}
	recursoRef := empleados[0]
	resultado, err := m.consulta.ConsultarPropiasParaDietas(r.Context(), personaldomain.SolicitudConsultaRelacionPropia{Actor: actor, FechaReferencia: fecha, Operacion: personaldomain.OperacionListaRelacionPropia})
	if err != nil {
		m.denegar(w, r, estadoErrorRelacionesDietas(err), actor.Principal.ID, recursoRef, "consultar_relaciones")
		return
	}
	type relacion struct {
		RelacionRef string `json:"relacion_ref"`
		UnidadRef   string `json:"unidad_ref"`
		Version     int64  `json:"version"`
	}
	salida := struct {
		RelacionesAutorizadas []relacion `json:"relaciones_autorizadas"`
		FechaReferencia       string     `json:"fecha_referencia"`
	}{RelacionesAutorizadas: make([]relacion, 0, len(resultado.Relaciones)), FechaReferencia: fecha.Texto()}
	for _, item := range resultado.Relaciones {
		if item.Validar() != nil || item.PersonaRef != actor.PersonaRef || !item.VigenteEn(fecha) {
			m.denegar(w, r, http.StatusServiceUnavailable, actor.Principal.ID, recursoRef, "consultar_relaciones")
			return
		}
		salida.RelacionesAutorizadas = append(salida.RelacionesAutorizadas, relacion{RelacionRef: item.RelacionRef, UnidadRef: item.UnidadRef, Version: item.Version})
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(salida)
}

func estadoErrorRelacionesDietas(err error) int {
	if errors.Is(err, personalports.ErrRelacionEmpleadoDenegada) {
		return http.StatusForbidden
	}
	return http.StatusServiceUnavailable
}

func cabeceraLibreRelacionesDietas(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		switch strings.ToLower(nombre) {
		case "cookie", "authorization", "x-vec-actor", "x-vec-persona", "x-vec-perfil":
			return true
		}
	}
	return false
}

func aceptaSoloJSONRelacionesDietas(cabeceras http.Header) bool {
	var valores []string
	for nombre, candidatos := range cabeceras {
		if strings.EqualFold(nombre, "Accept") {
			valores = append(valores, candidatos...)
		}
	}
	return len(valores) == 1 && valores[0] == "application/json"
}

func (m *ManejadorRelacionesDietas) denegar(w http.ResponseWriter, r *http.Request, estado int, actorRef, recursoRef, accion string) {
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	motivo := personalports.MotivoFronteraPersonalDependencia
	switch estado {
	case http.StatusBadRequest:
		motivo = personalports.MotivoFronteraPersonalPeticion
	case http.StatusUnauthorized:
		motivo = personalports.MotivoFronteraPersonalAutenticacion
		actorRef = ""
	case http.StatusForbidden:
		motivo = personalports.MotivoFronteraPersonalDenegado
	case http.StatusNotFound:
		motivo = personalports.MotivoFronteraPersonalNoEncontrada
	case http.StatusMethodNotAllowed:
		motivo = personalports.MotivoFronteraPersonalMetodo
	case http.StatusNotAcceptable:
		motivo = personalports.MotivoFronteraPersonalRepresentacion
	}
	orden := personalports.OrdenAuditoriaFronteraAsignacionDietas{CorrelacionRef: correlacion, Motivo: motivo, Ruta: personalports.RutaFronteraRelacionesDietas, Accion: accion, ActorRef: actorRef, RecursoRef: recursoRef, EstadoHTTP: estado}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	if orden.Validar() != nil || m.auditoria.RegistrarAuditoriaFronteraAsignacionDietas(ctx, orden) != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(estado)
}

func nuloRelacionesDietas(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	return (x.Kind() == reflect.Ptr || x.Kind() == reflect.Interface || x.Kind() == reflect.Func || x.Kind() == reflect.Map || x.Kind() == reflect.Slice) && x.IsNil()
}
