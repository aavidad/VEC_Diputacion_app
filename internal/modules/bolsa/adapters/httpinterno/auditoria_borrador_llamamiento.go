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
// handler B-BACK/B2/B3. Así registra tanto una denegación temprana como un fallo
// del caso de uso ya revertido.
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
	if _, _, ok := ReferenciasRutaOperacionesSituacion(r); ok {
		if r.Method == http.MethodPost {
			return puertosbolsa.AccionIntentoCambiarSituacionParticipacion, puertosbolsa.ClaseRutaSituacionParticipacion, true
		}
		if r.Method == http.MethodGet {
			return puertosbolsa.AccionIntentoConsultarBorradorLlamamiento, puertosbolsa.ClaseRutaSituacionParticipacion, true
		}
	}
	if _, _, ok := ReferenciasRutaContratosParticipacion(r); ok && r.Method == http.MethodGet {
		return puertosbolsa.AccionIntentoConsultarBorradorLlamamiento, puertosbolsa.ClaseRutaSituacionParticipacion, true
	}
	// Las sanciones usan la autorización de las operaciones de situación y
	// se auditan con su misma acción y clase de ruta.
	if _, _, _, ok := ReferenciasRutaSancionesParticipacion(r); ok {
		if r.Method == http.MethodPost {
			return puertosbolsa.AccionIntentoCambiarSituacionParticipacion, puertosbolsa.ClaseRutaSituacionParticipacion, true
		}
		if r.Method == http.MethodGet {
			return puertosbolsa.AccionIntentoConsultarBorradorLlamamiento, puertosbolsa.ClaseRutaSituacionParticipacion, true
		}
	}
	if _, _, _, ok := ReferenciasRutaDatosContactoParticipacion(r); ok && r.Method == http.MethodGet {
		return puertosbolsa.AccionIntentoConsultarDatosContactoParticipacion, puertosbolsa.ClaseRutaDatosContactoParticipacion, true
	}
	if _, _, _, ok := ReferenciasRutaDatosContactoParticipacion(r); ok && r.Method == http.MethodPost {
		return puertosbolsa.AccionIntentoRegistrarDatosContactoParticipacion, puertosbolsa.ClaseRutaDatosContactoParticipacion, true
	}
	// Las ofertas del art. 8.1 son una modalidad de llamamiento: se anotan
	// en la bitácora con la misma acción y clase de ruta que la emisión.
	if r.URL != nil && r.URL.RawPath == "" && (r.URL.Path == RutaOfertasPublicadas || r.URL.Path == RutaResolucionesOferta) {
		if r.Method == http.MethodGet {
			return puertosbolsa.AccionIntentoRecuperarLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento, true
		}
		if r.Method == http.MethodPost {
			return puertosbolsa.AccionIntentoEmitirLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento, true
		}
	}
	if r.URL != nil && r.URL.RawPath == "" && ((r.URL.Path == RutaPlantillaCorreoLlamamiento && r.Method == http.MethodGet) || (r.URL.Path == RutaVistaPreviaCorreoLlamamiento && r.Method == http.MethodPost)) {
		return puertosbolsa.AccionIntentoConsultarBorradorLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento, true
	}
	if r.URL != nil && r.URL.Path == RutaEmisionesLlamamiento && r.URL.RawPath == "" && r.Method == http.MethodGet {
		return puertosbolsa.AccionIntentoRecuperarLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento, true
	}
	if r.URL != nil && r.URL.Path == RutaEmisionesLlamamiento && r.URL.RawPath == "" && r.Method == http.MethodPost {
		return puertosbolsa.AccionIntentoEmitirLlamamiento, puertosbolsa.ClaseRutaEmisionesLlamamiento, true
	}
	if _, _, ok := ReferenciasRutaContactosParticipacion(r); ok && r.Method == http.MethodGet {
		return puertosbolsa.AccionIntentoConsultarBorradorLlamamiento, puertosbolsa.ClaseRutaContactosParticipacion, true
	}
	if _, _, ok := ReferenciasRutaContactosParticipacion(r); ok && r.Method == http.MethodPost {
		return puertosbolsa.AccionIntentoRegistrarContactoParticipacion, puertosbolsa.ClaseRutaContactosParticipacion, true
	}
	if _, _, ok := ReferenciasRutaSituacionParticipacion(r); ok && r.Method == http.MethodPost {
		return puertosbolsa.AccionIntentoCambiarSituacionParticipacion, puertosbolsa.ClaseRutaSituacionParticipacion, true
	}
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

func resultadoIntentoBorradorLlamamiento(estado int) puertosbolsa.ResultadoIntentoBorradorLlamamiento {
	if estado < http.StatusBadRequest {
		return puertosbolsa.ResultadoIntentoCorrectoBorradorLlamamiento
	}
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
