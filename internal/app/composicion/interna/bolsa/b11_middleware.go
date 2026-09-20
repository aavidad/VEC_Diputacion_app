// Package bolsa compone la frontera interna de la consulta B11.
package bolsa

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrRutaParticipacionesPropiasB11Invalida = errors.New("composicion interna bolsa: ruta B11 invalida")

const plazoAuditoriaDenegacionB11 = 250 * time.Millisecond

// ExtractorSobrePeticionInterno es una frontera deliberadamente opaca. Su
// implementacion pertenece al proxy interno ya autenticado: esta ruta no lee
// ni acepta una cabecera, cookie, query o cuerpo como credencial.
type ExtractorSobrePeticionInterno interface {
	ExtraerSobrePeticionInterno(context.Context) ([]byte, error)
}

type autenticadorCanalB11 interface {
	AutenticarCanalTLSMutuo(tls.ConnectionState) (httpseguridad.CanalProxyAutenticado, error)
}

type resolvedorPeticionSesionB11 interface {
	ResolverYVincular(context.Context, []byte, httpseguridad.CanalProxyAutenticado, string, string, []byte) (context.Context, error)
}

// MiddlewarePeticionSesionB11 verifica exclusivamente el sobre de una
// peticion GET/HEAD de Mi bolsa y lo consume antes de invocar al handler. El
// resolvedor concreto comprueba que el canal fue emitido por la misma
// instancia de ServicioIdentidad que posee la sesion.
type MiddlewarePeticionSesionB11 struct {
	autenticador  autenticadorCanalB11
	resolvedor    resolvedorPeticionSesionB11
	extractor     ExtractorSobrePeticionInterno
	denegaciones  puertosvec.RegistradorDenegacionFronteraIdentidadV1
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2
	siguiente     http.Handler
}

func NuevoMiddlewarePeticionSesionB11(
	autenticador autenticadorCanalB11,
	resolvedor resolvedorPeticionSesionB11,
	extractor ExtractorSobrePeticionInterno,
	denegaciones puertosvec.RegistradorDenegacionFronteraIdentidadV1,
	correlaciones puertosvec.GeneradorReferenciasAutorizacionV2,
	siguiente http.Handler,
) (http.Handler, error) {
	if dependenciaB11Nula(autenticador) || dependenciaB11Nula(resolvedor) ||
		dependenciaB11Nula(extractor) || dependenciaB11Nula(denegaciones) ||
		dependenciaB11Nula(correlaciones) || dependenciaB11Nula(siguiente) {
		return nil, ErrRutaParticipacionesPropiasB11Invalida
	}
	return &MiddlewarePeticionSesionB11{
		autenticador:  autenticador,
		resolvedor:    resolvedor,
		extractor:     extractor,
		denegaciones:  denegaciones,
		correlaciones: correlaciones,
		siguiente:     siguiente,
	}, nil
}

func (m *MiddlewarePeticionSesionB11) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	escritor := escritorSinCookieB11{ResponseWriter: w, sinCuerpo: r != nil && r.Method == http.MethodHead}
	if m == nil || r == nil || dependenciaB11Nula(m.autenticador) ||
		dependenciaB11Nula(m.resolvedor) || dependenciaB11Nula(m.extractor) ||
		dependenciaB11Nula(m.denegaciones) || dependenciaB11Nula(m.correlaciones) ||
		dependenciaB11Nula(m.siguiente) || !peticionB11Exacta(r) {
		m.responderDenegacion(r, escritor, "")
		return
	}
	if err := r.Context().Err(); err != nil {
		m.responderDenegacion(r, escritor, "")
		return
	}
	if tieneCredencialesClienteB11(r) || !cuerpoVacioB11(r) {
		m.responderDenegacion(r, escritor, "")
		return
	}
	if r.TLS == nil {
		m.responderDenegacion(r, escritor, "")
		return
	}
	canal, err := m.autenticador.AutenticarCanalTLSMutuo(*r.TLS)
	if err != nil {
		m.responderDenegacion(r, escritor, "")
		return
	}
	sobre, err := m.extractor.ExtraerSobrePeticionInterno(r.Context())
	if err != nil || len(sobre) == 0 {
		m.responderDenegacion(r, escritor, canal.ReferenciaVinculacion())
		return
	}
	defer borrarSobreB11(sobre)
	ctx, err := m.resolvedor.ResolverYVincular(
		r.Context(), sobre, canal, r.Method, rutaParticipacionesPropiasB11, nil,
	)
	if err != nil || ctx == nil || ctx.Err() != nil {
		m.responderDenegacion(r, escritor, canal.ReferenciaVinculacion())
		return
	}
	// El sobre nunca se adjunta al request ni se reexpone: solo la capsula
	// privada creada por ServicioPeticionSesion viaja al preparador B11.
	respuesta := nuevaRespuestaDiferidaB11()
	m.siguiente.ServeHTTP(respuesta, r.WithContext(ctx))
	if respuesta.estadoDenegado() {
		m.registrarDenegacion(r.Context(), respuesta.motivo(), canal.ReferenciaVinculacion())
	}
	respuesta.enviarA(escritor)
}

func (m *MiddlewarePeticionSesionB11) responderDenegacion(r *http.Request, w http.ResponseWriter, canalRef string) {
	m.registrarDenegacion(contextoPeticionB11(r), string(puertosvec.MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida), canalRef)
	responderB11Denegado(w)
}

func (m *MiddlewarePeticionSesionB11) registrarDenegacion(ctx context.Context, motivo, canalRef string) {
	if m == nil || dependenciaB11Nula(m.denegaciones) || dependenciaB11Nula(m.correlaciones) {
		return
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoAuditoriaDenegacionB11)
	defer cancelar()
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(ctxAuditoria, m.correlaciones)
	if err != nil {
		senalAuditoriaDenegacionB11("")
		return
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		senalAuditoriaDenegacionB11("")
		return
	}
	orden := puertosvec.OrdenDenegacionFronteraIdentidadV1{
		CorrelacionRef: correlacionRef,
		Superficie:     string(puertosvec.SuperficieDenegacionFronteraIdentidadV1ExternaPersonal),
		RutaExacta:     puertosvec.RutaExactaDenegacionFronteraIdentidadV1ParticipacionesPropias,
		Accion:         string(puertosvec.AccionDenegacionFronteraIdentidadV1ConsultarParticipacionesPropias),
		Motivo:         motivo,
		CanalRef:       canalRef,
	}
	if orden.Validar() != nil {
		senalAuditoriaDenegacionB11(correlacionRef)
		return
	}
	if err := m.denegaciones.RegistrarDenegacionFronteraIdentidadV1(ctxAuditoria, orden); err != nil {
		senalAuditoriaDenegacionB11(correlacionRef)
	}
}

func senalAuditoriaDenegacionB11(correlacionRef string) {
	const codigo = "vec_b11_auditoria_denegacion_no_disponible"
	if dominiovec.ReferenciaCorrelacionAutorizacionV2Valida(correlacionRef) {
		log.Printf("%s correlacion_ref=%s", codigo, correlacionRef)
		return
	}
	log.Print(codigo)
}

func contextoPeticionB11(r *http.Request) context.Context {
	if r == nil || r.Context() == nil {
		return context.Background()
	}
	return r.Context()
}

func peticionB11Exacta(r *http.Request) bool {
	if r == nil || r.URL == nil || (r.Method != http.MethodGet && r.Method != http.MethodHead) {
		return false
	}
	return r.URL.Path == rutaParticipacionesPropiasB11 && r.URL.RawPath == "" &&
		r.URL.EscapedPath() == rutaParticipacionesPropiasB11 && r.URL.RawQuery == "" &&
		r.URL.Fragment == ""
}

func tieneCredencialesClienteB11(r *http.Request) bool {
	return r == nil || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != ""
}

func cuerpoVacioB11(r *http.Request) bool {
	if r == nil || r.ContentLength != 0 {
		return false
	}
	if r.Body == nil {
		return true
	}
	contenido, err := io.ReadAll(io.LimitReader(r.Body, 1))
	_ = r.Body.Close()
	r.Body = http.NoBody
	return err == nil && len(contenido) == 0
}

func responderB11Denegado(w http.ResponseWriter) {
	if w == nil {
		return
	}
	borrarCookiesYTrailersB11(w.Header())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = io.WriteString(w, `{"error":"autenticacion_requerida"}`)
}

type respuestaDiferidaB11 struct {
	cabeceras http.Header
	cuerpo    bytes.Buffer
	estado    int
}

func nuevaRespuestaDiferidaB11() *respuestaDiferidaB11 {
	return &respuestaDiferidaB11{cabeceras: make(http.Header)}
}

func (r *respuestaDiferidaB11) Header() http.Header { return r.cabeceras }

func (r *respuestaDiferidaB11) WriteHeader(estado int) {
	if r.estado == 0 {
		r.estado = estado
	}
}

func (r *respuestaDiferidaB11) Write(cuerpo []byte) (int, error) {
	if r.estado == 0 {
		r.estado = http.StatusOK
	}
	return r.cuerpo.Write(cuerpo)
}

func (r *respuestaDiferidaB11) estadoDenegado() bool {
	return r != nil && (r.estado == http.StatusUnauthorized || r.estado == http.StatusForbidden)
}

func (r *respuestaDiferidaB11) motivo() string {
	if r != nil && r.estado == http.StatusForbidden {
		return string(puertosvec.MotivoDenegacionFronteraIdentidadV1AccesoDenegado)
	}
	return string(puertosvec.MotivoDenegacionFronteraIdentidadV1AutenticacionRequerida)
}

func (r *respuestaDiferidaB11) enviarA(destino http.ResponseWriter) {
	if r == nil || destino == nil {
		return
	}
	for nombre, valores := range r.cabeceras {
		if cabeceraCookieOTrailerB11(nombre) {
			continue
		}
		destino.Header()[nombre] = append([]string(nil), valores...)
	}
	borrarCookiesYTrailersB11(destino.Header())
	estado := r.estado
	if estado == 0 {
		estado = http.StatusOK
	}
	destino.WriteHeader(estado)
	_, _ = destino.Write(r.cuerpo.Bytes())
}

func borrarSobreB11(sobre []byte) {
	for indice := range sobre {
		sobre[indice] = 0
	}
}

type escritorSinCookieB11 struct {
	http.ResponseWriter
	sinCuerpo bool
}

func (e escritorSinCookieB11) WriteHeader(estado int) {
	borrarCookiesYTrailersB11(e.Header())
	e.ResponseWriter.WriteHeader(estado)
}

func (e escritorSinCookieB11) Write(cuerpo []byte) (int, error) {
	borrarCookiesYTrailersB11(e.Header())
	if e.sinCuerpo {
		return len(cuerpo), nil
	}
	return e.ResponseWriter.Write(cuerpo)
}

func borrarCookiesYTrailersB11(cabeceras http.Header) {
	for nombre := range cabeceras {
		if cabeceraCookieOTrailerB11(nombre) {
			delete(cabeceras, nombre)
		}
	}
}

func cabeceraCookieOTrailerB11(nombre string) bool {
	return strings.EqualFold(nombre, "Set-Cookie") ||
		strings.EqualFold(nombre, "Trailer") ||
		(len(nombre) >= len(http.TrailerPrefix) &&
			strings.EqualFold(nombre[:len(http.TrailerPrefix)], http.TrailerPrefix))
}

func dependenciaB11Nula(valor any) bool {
	if valor == nil {
		return true
	}
	v := reflect.ValueOf(valor)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
