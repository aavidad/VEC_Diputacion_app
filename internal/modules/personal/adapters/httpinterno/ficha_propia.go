package httpinterno

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// RutaFichaPropia sirve a la persona empleada su propia ficha. No admite
// parámetros: persona, perfil y empleado los fija la frontera del servidor.
const RutaFichaPropia = "/api/interna/personal/mi-ficha"

// margenConocidoFichaPropia deja fuera de la foto los hechos de los últimos
// instantes para que el corte nunca supere el reloj de la base de datos.
const margenConocidoFichaPropia = time.Second

var ErrManejadorFichaPropiaNoDisponible = errors.New("personal: ficha propia HTTP no disponible")

// ResolutorActorFichaPropia devuelve el contexto de actor que la frontera
// registró para esta misma petición (alcance {empleado}).
type ResolutorActorFichaPropia interface {
	ResolverActorFichaPropia(context.Context) (core.ContextoActor, error)
}

type consultorFichaPropia interface {
	Consultar(context.Context, personaldomain.SolicitudFichaPropia) (personalports.ResultadoFichaPropia, error)
}

type ManejadorFichaPropia struct {
	actor    ResolutorActorFichaPropia
	consulta consultorFichaPropia
	registro personalports.RegistroDenegacionFichaPropia
	ahora    func() time.Time
	zona     *time.Location
}

// NuevoManejadorFichaPropia fija la fecha de efectos en la zona indicada
// (la del organismo) y el instante de conocimiento en UTC.
func NuevoManejadorFichaPropia(actor ResolutorActorFichaPropia, consulta consultorFichaPropia, registro personalports.RegistroDenegacionFichaPropia, ahora func() time.Time, zona *time.Location) (*ManejadorFichaPropia, error) {
	if nuloRelacionesDietas(actor) || nuloRelacionesDietas(consulta) || nuloRelacionesDietas(registro) || ahora == nil || zona == nil {
		return nil, ErrManejadorFichaPropiaNoDisponible
	}
	return &ManejadorFichaPropia{actor: actor, consulta: consulta, registro: registro, ahora: ahora, zona: zona}, nil
}

func (m *ManejadorFichaPropia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if r == nil || r.URL == nil || m == nil || nuloRelacionesDietas(m.actor) || nuloRelacionesDietas(m.consulta) || nuloRelacionesDietas(m.registro) {
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	if r.URL.Path != RutaFichaPropia || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery {
		m.denegar(w, r, http.StatusNotFound, "no_encontrada", "")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		m.denegar(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido", "")
		return
	}
	if cabeceraLibreRelacionesDietas(r.Header) || (r.Body != nil && r.Body != http.NoBody) || len(r.TransferEncoding) != 0 || r.ContentLength > 0 {
		m.denegar(w, r, http.StatusBadRequest, "peticion_invalida", "")
		return
	}
	actor, err := m.actor.ResolverActorFichaPropia(r.Context())
	if err != nil || actor.Validar() != nil {
		m.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible", "")
		return
	}
	ahora := m.ahora().UTC()
	vigente, err := personaldomain.NuevaFechaCivil(ahora.In(m.zona).Format("2006-01-02"))
	if err != nil {
		m.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible", actor.Principal.ID)
		return
	}
	corte := personaldomain.CorteEmpleadoB2{VigenteEn: vigente, ConocidoEn: ahora.Add(-margenConocidoFichaPropia).Truncate(time.Microsecond)}
	resultado, err := m.consulta.Consultar(r.Context(), personaldomain.SolicitudFichaPropia{Corte: corte, Actor: actor})
	switch {
	case err == nil:
	case errors.Is(err, personaldomain.ErrFichaPropiaSinEmpleado):
		m.denegar(w, r, http.StatusForbidden, "sin_empleado", actor.Principal.ID)
		return
	case errors.Is(err, personaldomain.ErrFichaPropiaAmbigua):
		m.denegar(w, r, http.StatusForbidden, "empleado_ambiguo", actor.Principal.ID)
		return
	case errors.Is(err, personaldomain.ErrFichaPropiaDenegada):
		m.denegar(w, r, http.StatusForbidden, "acceso_denegado", actor.Principal.ID)
		return
	case errors.Is(err, personaldomain.ErrFichaPropiaExcedeLimite):
		// La ficha existe, pero no cabe en la pantalla: estado propio (422),
		// que el portal muestra en sus apartados en lugar de ocultarlos.
		m.denegar(w, r, http.StatusUnprocessableEntity, "excede_limite", actor.Principal.ID)
		return
	case r.Context().Err() != nil:
		return
	default:
		m.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible", actor.Principal.ID)
		return
	}
	// Solo lo que la pantalla necesita: sin referencias de persona, empleado,
	// decisión ni auditoría. El recibo es opaco.
	responderFichaPropia(w, http.StatusOK, "", map[string]any{"data": map[string]any{
		"ficha":         resultado.Ficha,
		"recibo_ref":    resultado.Evidencia.ReciboRef,
		"consultada_en": resultado.Evidencia.ConsultadaEn.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}})
}

// denegar registra la denegación antes de responder; si no se confirma,
// la respuesta es dependencia no disponible, nunca el motivo original.
func (m *ManejadorFichaPropia) denegar(w http.ResponseWriter, r *http.Request, estado int, motivo, actorRef string) {
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	if err := m.registro.RegistrarDenegacionFichaPropia(ctx, personalports.DenegacionFichaPropia{CorrelacionRef: correlacion, Motivo: motivo, EstadoHTTP: estado, ActorRef: actorRef}); err != nil {
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	codigo := motivo
	if estado == http.StatusServiceUnavailable {
		codigo = "no_disponible"
	}
	responderFichaPropia(w, estado, codigo, nil)
}

func responderFichaPropia(w http.ResponseWriter, estado int, codigo string, datos any) {
	for _, k := range []string{"Set-Cookie", "Access-Control-Allow-Origin", "Access-Control-Allow-Credentials", "Location", "Content-Encoding"} {
		w.Header().Del(k)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; frame-ancestors 'none'")
	if datos == nil {
		datos = map[string]string{"error": codigo}
	}
	contenido, err := json.Marshal(datos)
	if err != nil {
		estado = http.StatusServiceUnavailable
		contenido = []byte(`{"error":"no_disponible"}`)
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}
