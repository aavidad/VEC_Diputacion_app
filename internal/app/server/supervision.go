package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strconv"
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
//     componente y etapa cerrados. No se registran ruta, consulta, IP,
//     cabeceras ni cuerpos; la correlación la genera el emisor.
//   - Un pánico de handler se contiene aquí: se declara PANICO_CONTROLADO y,
//     si aún no se enviaron cabeceras, se responde un 500 de texto fijo. Si ya
//     se enviaron, se aborta la conexión como haría net/http, sin pila.
//   - El ErrorLog de net/http se sustituye por un escritor que descarta el
//     texto original (puede incluir IP:puerto, errores TLS o pilas) y deja una
//     línea fija con el recuento de eventos agrupados.
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
// llamarse antes de servir.
func SupervisarServidor(srv *http.Server, emisor ports.EmisorIncidenciasTecnicas, registro io.Writer) {
	if srv == nil {
		return
	}
	srv.Handler = SupervisarRespuestas(srv.Handler, emisor)
	srv.ErrorLog = log.New(nuevoEscritorErrorLogSaneado(registro, emisor), "", 0)
}

// SupervisarRespuestas es el middleware común de respuestas 5xx y pánicos.
func SupervisarRespuestas(siguiente http.Handler, emisor ports.EmisorIncidenciasTecnicas) http.Handler {
	if siguiente == nil {
		siguiente = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escritor := &escritorSupervisado{ResponseWriter: w}
		if emisor != nil {
			r = r.WithContext(ports.ConEmisorIncidenciasTecnicas(r.Context(), emisor))
		}
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
		if escritor.estado >= http.StatusInternalServerError {
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

func responderPanicoControlado(w *escritorSupervisado) {
	cabeceras := w.Header()
	for nombre := range cabeceras {
		delete(cabeceras, nombre)
	}
	cabeceras.Set("Content-Type", "application/json; charset=utf-8")
	cabeceras.Set("Cache-Control", "no-store")
	cabeceras.Set("X-Content-Type-Options", "nosniff")
	cabeceras.Set("Content-Length", strconv.Itoa(len(cuerpoPanicoControlado)))
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = io.WriteString(w, cuerpoPanicoControlado)
}

// escritorSupervisado observa el primer estado final sin alterar la
// respuesta. Conserva Flush y Unwrap para http.ResponseController.
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

func (w *escritorSupervisado) Flush() {
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
)

var prefijoPanicoNetHTTP = []byte("http: panic serving")

// escritorErrorLogSaneado sustituye el texto de net/http por una línea fija
// limitada a una cada intervalo, con el recuento de eventos agrupados, de modo
// que nada se pierde sin contar. Los pánicos que net/http llegue a registrar
// (fuera del middleware) declaran además PANICO_CONTROLADO.
type escritorErrorLogSaneado struct {
	mu        sync.Mutex
	destino   io.Writer
	emisor    ports.EmisorIncidenciasTecnicas
	ahora     func() time.Time
	ultimo    time.Time
	agrupados uint64
}

func nuevoEscritorErrorLogSaneado(destino io.Writer, emisor ports.EmisorIncidenciasTecnicas) *escritorErrorLogSaneado {
	return &escritorErrorLogSaneado{destino: destino, emisor: emisor, ahora: time.Now}
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
	defer e.mu.Unlock()
	ahora := e.ahora()
	if !panico && !e.ultimo.IsZero() && ahora.Sub(e.ultimo) < intervaloEventoServidorSaneado {
		e.agrupados++
		return longitud, nil
	}
	e.ultimo = ahora
	linea := mensaje + " agrupados_previos=" + strconv.FormatUint(e.agrupados, 10) + "\n"
	e.agrupados = 0
	// El ErrorLog no tiene otro destino al que informar de su propio fallo.
	_, _ = io.WriteString(e.destino, linea)
	return longitud, nil
}
