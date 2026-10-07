package httpinterno

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
)

// RutaFichaPropia sirve a la persona empleada su propia ficha. No admite
// referencias: persona, perfil y empleado los fija la frontera del servidor.
// Solo admite una fecha civil de referencia opcional para los servicios.
const RutaFichaPropia = "/api/interna/personal/mi-ficha"

// PerfilAceptacionFichaPropiaExportacion negocia sólo metadatos de representación.
const PerfilAceptacionFichaPropiaExportacion = `application/json; profile="urn:vec:personal:ficha-propia:exportacion:v1"`

// Preferencia de representación; no constituye autorización para la historia.
const PreferenciaHistoriaServiciosFichaPropia = "vec-personal-historia-servicios-v1"
const PreferenciaHistoriaRelacionesFichaPropia = "vec-personal-historia-relaciones-v1"

// margenConocidoFichaPropia deja fuera de la foto los hechos de los últimos
// instantes para que el corte nunca supere el reloj de la base de datos.
const margenConocidoFichaPropia = time.Second

// Estos motivos de transporte se propagan como estados cerrados. ServeHTTP
// registra cada rechazo previo a la consulta sin guardar la query.
const (
	MotivoFichaPropiaQueryNoEncontrada = http.StatusNotFound
	MotivoFichaPropiaQueryInvalida     = http.StatusBadRequest
)

var ErrManejadorFichaPropiaNoDisponible = errors.New("personal: ficha propia HTTP no disponible")

// ResolutorActorFichaPropia devuelve el contexto de actor que la frontera
// capturó antes de validar esta petición (alcance {empleado}). No vuelve a
// resolver identidad ni acepta un actor procedente del transporte.
type ResolutorActorFichaPropia interface {
	ResolverActorFichaPropia(context.Context) (core.ContextoActor, error)
}

type consultorFichaPropia interface {
	Consultar(context.Context, personaldomain.SolicitudFichaPropia) (personalports.ResultadoFichaPropia, error)
}

type ManejadorFichaPropia struct {
	actor                        ResolutorActorFichaPropia
	consulta                     consultorFichaPropia
	registro                     personalports.RegistroIntentosFichaPropia
	ahora                        func() time.Time
	zona                         *time.Location
	exportacionDisponible        bool
	historiaDisponible           bool
	historiaRelacionesDisponible bool
}

// NuevoManejadorFichaPropia fija la fecha de efectos en la zona indicada
// (la del organismo) y el instante de conocimiento en UTC.
func NuevoManejadorFichaPropia(actor ResolutorActorFichaPropia, consulta consultorFichaPropia, registro personalports.RegistroIntentosFichaPropia, ahora func() time.Time, zona *time.Location, exportacionDisponible ...bool) (*ManejadorFichaPropia, error) {
	if nuloRelacionesDietas(actor) || nuloRelacionesDietas(consulta) || nuloRelacionesDietas(registro) || ahora == nil || zona == nil || len(exportacionDisponible) > 3 {
		return nil, ErrManejadorFichaPropiaNoDisponible
	}
	disponible := len(exportacionDisponible) >= 1 && exportacionDisponible[0]
	return &ManejadorFichaPropia{actor: actor, consulta: consulta, registro: registro, ahora: ahora, zona: zona, exportacionDisponible: disponible, historiaDisponible: len(exportacionDisponible) >= 2 && exportacionDisponible[1], historiaRelacionesDisponible: len(exportacionDisponible) == 3 && exportacionDisponible[2]}, nil
}

func (m *ManejadorFichaPropia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if r == nil || r.URL == nil || m == nil || nuloRelacionesDietas(m.actor) || nuloRelacionesDietas(m.consulta) || nuloRelacionesDietas(m.registro) {
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	// La frontera ya debe haber capturado identidad y correlación comunes.
	// Sin ese contexto no atribuimos al cliente ningún intento.
	actor, err := m.actor.ResolverActorFichaPropia(r.Context())
	if err != nil || actor.Validar() != nil {
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	if err := m.registro.VerificarRegistroFichaPropia(r.Context()); err != nil {
		m.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible")
		return
	}
	if r.URL.Path != RutaFichaPropia || r.URL.RawPath != "" {
		m.denegar(w, r, http.StatusNotFound, "no_encontrada")
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		m.denegar(w, r, http.StatusMethodNotAllowed, "metodo_no_permitido")
		return
	}
	if cabeceraLibreRelacionesDietas(r.Header) || (r.Body != nil && r.Body != http.NoBody) || len(r.TransferEncoding) != 0 || r.ContentLength > 0 {
		m.denegar(w, r, http.StatusBadRequest, "peticion_invalida")
		return
	}
	referencia, estado := fechaReferenciaFichaPropia(r.URL)
	if estado != 0 {
		codigo := "peticion_invalida"
		if estado == http.StatusNotFound {
			codigo = "no_encontrada"
		}
		m.denegar(w, r, estado, codigo)
		return
	}
	ahora := m.ahora().UTC()
	vigente, err := personaldomain.NuevaFechaCivil(ahora.In(m.zona).Format("2006-01-02"))
	if err != nil {
		m.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible")
		return
	}
	if referencia != "" {
		vigente = referencia
	}
	corte := personaldomain.CorteEmpleadoB2{VigenteEn: vigente, ConocidoEn: ahora.Add(-margenConocidoFichaPropia).Truncate(time.Microsecond)}
	resultado, err := m.consulta.Consultar(r.Context(), personaldomain.SolicitudFichaPropia{Corte: corte, Actor: actor})
	switch {
	case err == nil:
	case errors.Is(err, personaldomain.ErrFichaPropiaSinEmpleado):
		responderFichaPropia(w, http.StatusForbidden, "sin_empleado", nil)
		return
	case errors.Is(err, personaldomain.ErrFichaPropiaAmbigua):
		responderFichaPropia(w, http.StatusForbidden, "empleado_ambiguo", nil)
		return
	case errors.Is(err, personaldomain.ErrFichaPropiaDenegada):
		responderFichaPropia(w, http.StatusForbidden, "acceso_denegado", nil)
		return
	case errors.Is(err, personaldomain.ErrFichaPropiaExcedeLimite):
		// La ficha existe, pero no cabe en la pantalla: estado propio (422),
		// que el portal muestra en sus apartados en lugar de ocultarlos.
		responderFichaPropia(w, http.StatusUnprocessableEntity, "excede_limite", nil)
		return
	case r.Context().Err() != nil:
		return
	default:
		responderFichaPropia(w, http.StatusServiceUnavailable, "no_disponible", nil)
		return
	}
	// Solo lo que la pantalla necesita: sin referencias de persona, empleado,
	// decisión ni auditoría. El recibo es opaco.
	datos := map[string]any{
		"ficha":         resultado.Ficha,
		"recibo_ref":    resultado.Evidencia.ReciboRef,
		"consultada_en": resultado.Evidencia.ConsultadaEn.UTC().Format("2006-01-02T15:04:05.000000Z"),
	}
	// Los clientes anteriores conservan su sobre de tres claves exactas.
	acepta := r.Header.Values("Accept")
	if len(acepta) == 1 && acepta[0] == PerfilAceptacionFichaPropiaExportacion {
		datos["exportacion_servicios_disponible"] = m.exportacionDisponible
	}
	preferencia := r.Header.Values("Prefer")
	if len(preferencia) == 1 && (preferencia[0] == PreferenciaHistoriaServiciosFichaPropia || preferencia[0] == PreferenciaHistoriaServiciosFichaPropia+", "+PreferenciaHistoriaRelacionesFichaPropia) {
		datos["historia_servicios_disponible"] = m.historiaDisponible
	}
	if len(preferencia) == 1 && (preferencia[0] == PreferenciaHistoriaRelacionesFichaPropia || preferencia[0] == PreferenciaHistoriaServiciosFichaPropia+", "+PreferenciaHistoriaRelacionesFichaPropia) {
		datos["historia_relaciones_disponible"] = m.historiaRelacionesDisponible
	}
	w.Header().Add("Vary", "Prefer")
	w.Header().Add("Vary", "Accept")
	responderFichaPropia(w, http.StatusOK, "", map[string]any{"data": datos})
}

// La única entrada temporal del navegador es la fecha civil. El instante de
// conocimiento sigue derivándose del reloj servidor en cada consulta.
func fechaReferenciaFichaPropia(u *url.URL) (personaldomain.FechaCivil, int) {
	if u.RawQuery == "" && !u.ForceQuery {
		return "", 0
	}
	if len(u.RawQuery) > 80 {
		return "", MotivoFichaPropiaQueryNoEncontrada
	}
	valores, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(valores) != 1 || len(valores["fecha_referencia"]) != 1 {
		return "", MotivoFichaPropiaQueryNoEncontrada
	}
	texto := valores.Get("fecha_referencia")
	fecha, err := personaldomain.NuevaFechaCivil(texto)
	if err != nil || (len(texto) == 10 && texto[:4] == "0000") || u.RawQuery != "fecha_referencia="+texto {
		return "", MotivoFichaPropiaQueryInvalida
	}
	return fecha, 0
}

// denegar registra sólo rechazos anteriores a la consulta. El contexto conserva
// la identidad original y la correlación que preparó la frontera del servidor.
// Si no hay acuse durable, la respuesta no revela el motivo original.
func (m *ManejadorFichaPropia) denegar(w http.ResponseWriter, r *http.Request, estado int, motivo string) {
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(2*time.Second))
	defer cancelar()
	motivoRegistro := "entrada_invalida"
	if estado == http.StatusServiceUnavailable {
		motivoRegistro = "no_disponible"
	}
	if err := m.registro.RegistrarIntentoFichaPropia(ctx, personalports.IntentoFichaPropia{Motivo: motivoRegistro}); err != nil {
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
