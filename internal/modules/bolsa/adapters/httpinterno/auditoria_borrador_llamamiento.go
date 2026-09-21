package httpinterno

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// El límite impide que una respuesta defectuosa convierta la bitácora de
// frontera en un acumulador sin cota. El handler propio ya limita sus JSON a
// este tamaño; este margen protege también las denegaciones previas a él.
const maximoRespuestaAuditableBorradorLlamamientoBytes = maximoRespuestaBorradorLlamamientoBytes

const tiempoMaximoAuditoriaBorradorLlamamiento = 2 * time.Second

var ErrAuditoriaBorradorLlamamientoInvalida = errors.New("bolsa http interno: auditoria de borrador de llamamiento invalida")

// ResolutorActorVerificadoBorradorLlamamiento es una frontera de composición:
// solo una identidad ya revalidada en servidor puede devolver un actor. El
// adaptador nunca mira cabeceras, certificado ni parámetros para inventarlo.
// La ausencia de resolutor conserva el actor vacío, que es válido para una
// denegación anterior a la autenticación.
type ResolutorActorVerificadoBorradorLlamamiento interface {
	ActorVerificadoParaAuditoriaBorradorLlamamiento(context.Context) (string, bool)
}

// generadorCorrelacionIntentoBorradorLlamamiento es el contrato mínimo que
// necesita esta frontera. No exige capacidad para acuñar motivos porque una
// bitácora de fallo no emite ni autoriza operaciones.
type generadorCorrelacionIntentoBorradorLlamamiento interface {
	NuevaReferenciaCorrelacionAutorizacionV2(context.Context) (string, error)
}

type auditoriaBorradorLlamamiento struct {
	siguiente   http.Handler
	registrador puertosbolsa.RegistradorIntentoBorradorLlamamiento
	generador   generadorCorrelacionIntentoBorradorLlamamiento
	actor       ResolutorActorVerificadoBorradorLlamamiento
}

// NuevaAuditoriaBorradorLlamamiento debe envolver la protección de ruta y el
// handler B-BACK. Así registra tanto una denegación temprana como un fallo del
// caso de uso ya revertido. Solo persiste fallos de las dos operaciones cerradas
// (POST colección y GET detalle); los demás recursos no le pertenecen.
func NuevaAuditoriaBorradorLlamamiento(
	siguiente http.Handler,
	registrador puertosbolsa.RegistradorIntentoBorradorLlamamiento,
	generador generadorCorrelacionIntentoBorradorLlamamiento,
	actor ResolutorActorVerificadoBorradorLlamamiento,
) (http.Handler, error) {
	if dependenciaNula(siguiente) || dependenciaNula(registrador) || dependenciaNula(generador) {
		return nil, ErrAuditoriaBorradorLlamamientoInvalida
	}
	return &auditoriaBorradorLlamamiento{siguiente: siguiente, registrador: registrador, generador: generador, actor: actor}, nil
}

func (a *auditoriaBorradorLlamamiento) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a == nil || dependenciaNula(a.siguiente) || dependenciaNula(a.registrador) || dependenciaNula(a.generador) || w == nil || r == nil {
		responderFalloAuditoriaBorradorLlamamiento(w, r)
		return
	}

	accion, clase, aplicable := intentoAuditableBorradorLlamamiento(r)
	if !aplicable {
		a.siguiente.ServeHTTP(w, r)
		return
	}

	ctxPeticion := r.Context()
	respuesta := nuevaRespuestaDiferidaBorradorLlamamiento()
	a.siguiente.ServeHTTP(respuesta, r)
	estado := respuesta.estadoFinal()
	if respuesta.excedida {
		estado = http.StatusServiceUnavailable
	}
	if !esFalloAuditableBorradorLlamamiento(estado) {
		respuesta.volcarEn(w, false, r.Method == http.MethodGet)
		return
	}

	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctxPeticion), tiempoMaximoAuditoriaBorradorLlamamiento)
	defer cancelar()
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctxAuditoria, a.generador)
	if err != nil {
		responderFalloAuditoriaBorradorLlamamiento(w, r)
		return
	}
	intento := puertosbolsa.IntentoBorradorLlamamiento{
		Correlacion: correlacion,
		Accion:      accion,
		ClaseRuta:   clase,
		Resultado:   resultadoIntentoBorradorLlamamiento(estado),
	}
	if !dependenciaNula(a.actor) {
		if actor, verificado := a.actor.ActorVerificadoParaAuditoriaBorradorLlamamiento(ctxAuditoria); verificado {
			intento.ActorVerificado = actor
		}
	}
	if intento.Validar() != nil || a.registrador.RegistrarIntentoBorradorLlamamiento(ctxAuditoria, intento) != nil {
		responderFalloAuditoriaBorradorLlamamiento(w, r)
		return
	}
	if respuesta.excedida {
		responderFalloAuditoriaBorradorLlamamiento(w, r)
		return
	}
	respuesta.volcarEn(w, false, r.Method == http.MethodGet)
}

func intentoAuditableBorradorLlamamiento(r *http.Request) (puertosbolsa.AccionIntentoBorradorLlamamiento, puertosbolsa.ClaseRutaIntentoBorradorLlamamiento, bool) {
	_, clase := reconocerRutaBorradorLlamamiento(r)
	switch clase {
	case rutaBorradorLlamamientoColeccion:
		if r.Method == http.MethodPost {
			return puertosbolsa.AccionIntentoCrearBorradorLlamamiento, puertosbolsa.ClaseRutaColeccionBorradorLlamamiento, true
		}
	case rutaBorradorLlamamientoDetalle:
		if r.Method == http.MethodGet {
			return puertosbolsa.AccionIntentoConsultarBorradorLlamamiento, puertosbolsa.ClaseRutaDetalleBorradorLlamamiento, true
		}
	}
	return "", "", false
}

func esFalloAuditableBorradorLlamamiento(estado int) bool { return estado >= http.StatusBadRequest }

func resultadoIntentoBorradorLlamamiento(estado int) puertosbolsa.ResultadoIntentoBorradorLlamamiento {
	switch estado {
	case http.StatusUnauthorized:
		return puertosbolsa.ResultadoIntentoAutenticacionRequeridaBorradorLlamamiento
	case http.StatusForbidden:
		return puertosbolsa.ResultadoIntentoAccesoDenegadoBorradorLlamamiento
	case http.StatusNotFound:
		return puertosbolsa.ResultadoIntentoRecursoNoDisponibleBorradorLlamamiento
	default:
		if estado >= http.StatusInternalServerError {
			return puertosbolsa.ResultadoIntentoInfraestructuraNoDisponibleBorradorLlamamiento
		}
		return puertosbolsa.ResultadoIntentoIndeterminadoBorradorLlamamiento
	}
}

type respuestaDiferidaBorradorLlamamiento struct {
	cabecera http.Header
	cuerpo   bytes.Buffer
	estado   int
	escrita  bool
	excedida bool
}

func nuevaRespuestaDiferidaBorradorLlamamiento() *respuestaDiferidaBorradorLlamamiento {
	return &respuestaDiferidaBorradorLlamamiento{cabecera: make(http.Header)}
}

func (r *respuestaDiferidaBorradorLlamamiento) Header() http.Header { return r.cabecera }

func (r *respuestaDiferidaBorradorLlamamiento) WriteHeader(estado int) {
	if r.escrita {
		return
	}
	r.escrita, r.estado = true, estado
}

func (r *respuestaDiferidaBorradorLlamamiento) Write(contenido []byte) (int, error) {
	if !r.escrita {
		r.WriteHeader(http.StatusOK)
	}
	if len(contenido) > maximoRespuestaAuditableBorradorLlamamientoBytes-r.cuerpo.Len() {
		r.excedida = true
		return len(contenido), nil
	}
	return r.cuerpo.Write(contenido)
}

func (r *respuestaDiferidaBorradorLlamamiento) estadoFinal() int {
	if !r.escrita {
		return http.StatusOK
	}
	return r.estado
}

func (r *respuestaDiferidaBorradorLlamamiento) volcarEn(destino http.ResponseWriter, sinCuerpo, sinCookieYTrailer bool) {
	copiarCabecerasAuditoriaBorradorLlamamiento(destino.Header(), r.cabecera, sinCookieYTrailer)
	destino.WriteHeader(r.estadoFinal())
	if !sinCuerpo {
		_, _ = destino.Write(r.cuerpo.Bytes())
	}
}

func copiarCabecerasAuditoriaBorradorLlamamiento(destino, origen http.Header, sinCookieYTrailer bool) {
	for nombre := range destino {
		destino.Del(nombre)
	}
	for nombre, valores := range origen {
		if sinCookieYTrailer && (strings.EqualFold(nombre, "Set-Cookie") || strings.EqualFold(nombre, "Trailer")) {
			continue
		}
		for _, valor := range valores {
			destino.Add(nombre, valor)
		}
	}
	if sinCookieYTrailer {
		for _, valor := range origen.Values("Trailer") {
			for _, nombre := range strings.Split(valor, ",") {
				destino.Del(strings.TrimSpace(nombre))
			}
		}
		destino.Del("Set-Cookie")
		destino.Del("Trailer")
	}
}

func responderFalloAuditoriaBorradorLlamamiento(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	for nombre := range w.Header() {
		w.Header().Del(nombre)
	}
	aplicarCabeceras(w)
	w.Header().Set("Content-Length", strconv.Itoa(len(`{"error":{"codigo":"servicio_no_disponible"}}`)))
	w.WriteHeader(http.StatusServiceUnavailable)
	if r == nil || r.Method != http.MethodHead {
		_, _ = w.Write([]byte(`{"error":{"codigo":"servicio_no_disponible"}}`))
	}
}
