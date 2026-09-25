package supervision

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Supervisión técnica del servidor HTTP (M2a del consenso de supervisión del
// 25/09/2026). Requisito: ningún fallo sin registro y ningún dato personal en
// la capa técnica, sin retrasar la aplicación.
//
//   - Toda respuesta final con estado >= 500 declara HTTP_INTERNO_FALLIDO con
//     componente y etapa cerrados, salvo que en esa misma petición un
//     adaptador ya declarase un código específico (marca en el contexto,
//     ports.EmitirIncidenciaTecnicaEnPeticion): un fallo se cuenta una vez.
//     No se registran ruta, consulta, IP, cabeceras ni cuerpos; la
//     correlación la genera el emisor.
//   - Un pánico de handler se contiene aquí: se declara PANICO_CONTROLADO y,
//     si aún no se enviaron cabeceras, se responde un 500 de texto fijo que
//     conserva las cabeceras de seguridad ya fijadas (CSP, HSTS, etc.) y
//     retira cookies y cabeceras de contenido. Si ya se enviaron, se aborta
//     la conexión como haría net/http, sin pila.
//   - El ErrorLog de net/http se sustituye por un escritor que descarta el
//     texto original (puede incluir IP:puerto, errores TLS o pilas) y deja una
//     línea fija con el recuento de eventos agrupados; el último grupo se
//     vuelca al vencer el intervalo o al cerrar.
//
// El emisor es no bloqueante por contrato (ports.EmisorIncidenciasTecnicas):
// el coste añadido por petición es una envoltura del ResponseWriter y un
// contexto derivado.

const (
	// cuerpoPanicoControlado es la única respuesta que ve el cliente tras un
	// pánico: sin mensaje del pánico ni detalle interno.
	cuerpoPanicoControlado = "{\"error\":\"error_interno\"}\n"

	intervaloEventoServidorSaneado = 30 * time.Second
)

// SupervisarServidor envuelve el Handler de srv con la supervisión técnica y
// fija su ErrorLog saneado hacia registro. Un emisor nil conserva la
// contención de pánicos y el saneamiento sin declarar incidencias. Debe
// llamarse antes de servir. Devuelve la función idempotente que vuelca el
// último grupo pendiente del ErrorLog; se registra también en el cierre
// ordenado del servidor.
func SupervisarServidor(srv *http.Server, emisor ports.EmisorIncidenciasTecnicas, registro io.Writer) func() {
	if srv == nil {
		return func() {}
	}
	srv.Handler = SupervisarRespuestas(srv.Handler, emisor)
	escritor := nuevoEscritorErrorLogSaneado(registro, emisor)
	srv.ErrorLog = log.New(escritor, "", 0)
	srv.RegisterOnShutdown(escritor.Cerrar)
	return escritor.Cerrar
}

// SupervisarRespuestas es el middleware común de respuestas 5xx y pánicos.
func SupervisarRespuestas(siguiente http.Handler, emisor ports.EmisorIncidenciasTecnicas) http.Handler {
	if siguiente == nil {
		siguiente = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escritor := &escritorSupervisado{ResponseWriter: w}
		ctx, especificaDeclarada := ports.ConMarcaIncidenciasPeticion(r.Context())
		r = r.WithContext(ctx)
		defer func() {
			valor := recover()
			if valor == nil {
				return
			}
			if valor == http.ErrAbortHandler {
				panic(valor)
			}
			emitirIncidenciaServidor(emisor, domain.IncidenciaPanicoControlado)
			if escritor.estado != 0 {
				// Cabeceras ya enviadas: no hay respuesta coherente posible.
				panic(http.ErrAbortHandler)
			}
			responderPanicoControlado(escritor)
		}()
		siguiente.ServeHTTP(escritor, r)
		if escritor.estado >= http.StatusInternalServerError && !especificaDeclarada() {
			emitirIncidenciaServidor(emisor, domain.IncidenciaHTTPInternoFallido)
		}
	})
}

// emitirIncidenciaServidor declara el código en el único componente/etapa
// que el catálogo v1 admite para una petición HTTP.
func emitirIncidenciaServidor(emisor ports.EmisorIncidenciasTecnicas, codigo domain.CodigoIncidenciaTecnica) {
	if emisor == nil {
		return
	}
	emisor.Emitir(domain.SolicitudIncidenciaTecnica{
		Codigo:     codigo,
		Componente: domain.ComponenteIncidenciaHTTP,
		Etapa:      domain.EtapaIncidenciaPeticion,
	})
}

// cabecerasRetiradasTrasPanico describen el contenido o el estado que el
// handler preparaba y ya no se sirve. Las de seguridad (CSP, HSTS,
// X-Frame-Options, Referrer-Policy, CORP/COOP…) se conservan.
var cabecerasRetiradasTrasPanico = []string{
	"Set-Cookie", "Content-Type", "Content-Length", "Content-Encoding", "Content-Disposition",
	"Content-Language", "Content-Location", "Content-Range", "Content-Md5", "Etag", "Last-Modified",
	"Expires", "Accept-Ranges", "Location", "Refresh", "Link", "Trailer", "Transfer-Encoding",
}

func responderPanicoControlado(w *escritorSupervisado) {
	cabeceras := w.Header()
	for _, nombre := range cabecerasRetiradasTrasPanico {
		cabeceras.Del(nombre)
	}
	for nombre := range cabeceras {
		// Trailers anunciados con el prefijo de net/http.
		if strings.HasPrefix(nombre, http.TrailerPrefix) {
			delete(cabeceras, nombre)
		}
	}
	cabeceras.Set("Content-Type", "application/json; charset=utf-8")
	cabeceras.Set("Cache-Control", "no-store")
	cabeceras.Set("X-Content-Type-Options", "nosniff")
	cabeceras.Set("Content-Length", strconv.Itoa(len(cuerpoPanicoControlado)))
	w.WriteHeader(http.StatusInternalServerError)
	// El pánico ya se declaró; el cliente puede haberse desconectado.
	_, _ = io.WriteString(w, cuerpoPanicoControlado)
}

// escritorSupervisado observa el primer estado final sin alterar la
// respuesta. Conserva Flush, ReadFrom (sendfile) y Unwrap para
// http.ResponseController.
type escritorSupervisado struct {
	http.ResponseWriter
	estado int
}

func (w *escritorSupervisado) WriteHeader(estado int) {
	if w.estado == 0 && !esRespuestaInformativa(estado) {
		w.estado = estado
	}
	w.ResponseWriter.WriteHeader(estado)
}

func (w *escritorSupervisado) Write(contenido []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	return w.ResponseWriter.Write(contenido)
}

// ReadFrom conserva la vía rápida de net/http (sendfile/splice de
// http.ServeContent y FileServer) delegando en el ReaderFrom del
// ResponseWriter subyacente cuando lo implementa.
func (w *escritorSupervisado) ReadFrom(origen io.Reader) (int64, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	if lector, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return lector.ReadFrom(origen)
	}
	return io.Copy(w.ResponseWriter, origen)
}

func (w *escritorSupervisado) Flush() {
	// http.Flusher no admite devolver el error del cliente desconectado.
	_ = w.FlushError()
}

func (w *escritorSupervisado) FlushError() error {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *escritorSupervisado) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Mensajes fijos del ErrorLog saneado. Solo se distingue la clase del evento
// por un prefijo estable de net/http; el resto del texto nunca se copia.
const (
	mensajeErrorLogPanico = "servidor: panico de handler contenido por net/http"
	mensajeErrorLogEvento = "servidor: evento HTTP/TLS rechazado"
	mensajeErrorLogGrupo  = "servidor: eventos HTTP/TLS rechazados agrupados"
)

var prefijoPanicoNetHTTP = []byte("http: panic serving")

// escritorErrorLogSaneado sustituye el texto de net/http por una línea fija
// limitada a una cada intervalo, con el recuento de eventos agrupados, de modo
// que nada se pierde sin contar: el grupo pendiente se vuelca al vencer el
// intervalo (temporizador) o al cerrar. Los pánicos que net/http llegue a
// registrar (fuera del middleware) declaran además PANICO_CONTROLADO. La
// escritura al destino nunca se hace con el mutex tomado: un destino lento no
// bloquea a otras conexiones que registran a la vez.
type escritorErrorLogSaneado struct {
	mu        sync.Mutex
	destino   io.Writer
	emisor    ports.EmisorIncidenciasTecnicas
	ahora     func() time.Time
	programar func(time.Duration, func()) *time.Timer
	ultimo    time.Time
	agrupados uint64
	volcado   *time.Timer
	cerrado   bool
}

func nuevoEscritorErrorLogSaneado(destino io.Writer, emisor ports.EmisorIncidenciasTecnicas) *escritorErrorLogSaneado {
	return &escritorErrorLogSaneado{destino: destino, emisor: emisor, ahora: time.Now, programar: time.AfterFunc}
}

func (e *escritorErrorLogSaneado) Write(entrada []byte) (int, error) {
	longitud := len(entrada)
	panico := bytes.HasPrefix(entrada, prefijoPanicoNetHTTP)
	if panico && e.emisor != nil {
		emitirIncidenciaServidor(e.emisor, domain.IncidenciaPanicoControlado)
	}
	if e.destino == nil {
		return longitud, nil
	}
	mensaje := mensajeErrorLogEvento
	if panico {
		mensaje = mensajeErrorLogPanico
	}
	e.mu.Lock()
	ahora := e.ahora()
	if !panico && !e.cerrado && !e.ultimo.IsZero() && ahora.Sub(e.ultimo) < intervaloEventoServidorSaneado {
		e.agrupados++
		if e.volcado == nil {
			e.volcado = e.programar(intervaloEventoServidorSaneado-ahora.Sub(e.ultimo), e.volcarGrupo)
		}
		e.mu.Unlock()
		return longitud, nil
	}
	e.ultimo = ahora
	linea := mensaje + " agrupados_previos=" + strconv.FormatUint(e.agrupados, 10) + "\n"
	e.agrupados = 0
	e.detenerVolcadoBloqueado()
	e.mu.Unlock()
	e.escribir(linea)
	return longitud, nil
}

// volcarGrupo escribe el grupo pendiente cuando vence el intervalo sin que
// llegue otro evento que lo arrastre.
func (e *escritorErrorLogSaneado) volcarGrupo() {
	e.mu.Lock()
	e.volcado = nil
	if e.agrupados == 0 {
		e.mu.Unlock()
		return
	}
	linea := mensajeErrorLogGrupo + " agrupados=" + strconv.FormatUint(e.agrupados, 10) + "\n"
	e.agrupados = 0
	e.ultimo = e.ahora()
	e.mu.Unlock()
	e.escribir(linea)
}

// Cerrar vuelca el grupo pendiente y detiene el temporizador. Es idempotente;
// los eventos posteriores se escriben sin agrupar.
func (e *escritorErrorLogSaneado) Cerrar() {
	e.mu.Lock()
	e.cerrado = true
	e.detenerVolcadoBloqueado()
	pendientes := e.agrupados
	e.agrupados = 0
	e.mu.Unlock()
	if pendientes > 0 && e.destino != nil {
		e.escribir(mensajeErrorLogGrupo + " agrupados=" + strconv.FormatUint(pendientes, 10) + "\n")
	}
}

// detenerVolcadoBloqueado exige e.mu tomado.
func (e *escritorErrorLogSaneado) detenerVolcadoBloqueado() {
	if e.volcado != nil {
		e.volcado.Stop()
		e.volcado = nil
	}
}

// escribir entrega una línea completa en una sola escritura, sin mutex
// tomado. log.Logger ya serializa las llamadas a Write; el volcado del
// temporizador y el cierre escriben líneas breves completas.
func (e *escritorErrorLogSaneado) escribir(linea string) {
	// El ErrorLog no tiene otro destino al que informar de su propio fallo.
	_, _ = io.WriteString(e.destino, linea)
}

// esRespuestaInformativa reconoce las respuestas 1xx provisionales (salvo 101,
// que es final): no fijan el estado observado. Misma regla que el paquete
// server, repetida aquí para no arrastrar ese paquete ni ser arrastrado por él.
func esRespuestaInformativa(estado int) bool {
	return estado >= 100 && estado <= 199 && estado != http.StatusSwitchingProtocols
}
