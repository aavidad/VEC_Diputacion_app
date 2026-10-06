// Package telemetria escribe el registro técnico de acceso de los procesos
// HTTP de VEC: una línea JSON (log/slog) por petición con ruta, estado y
// duración, para que Sistemas encuentre las peticiones lentas o fallidas.
//
// Es independiente de la auditoría de uso de datos: no guarda identidad,
// consulta, cabeceras ni cuerpos, y de la ruta solo la plantilla o el camino
// sin valores.
package telemetria

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// Revision se marca al compilar:
//
//	-ldflags "-X vec-diputacion-granada/internal/shared/telemetria.Revision=<hash>"
//
// Los guiones de despliegue compilan con -buildvcs=false, así que sin la
// marca el binario no sabe de qué revisión procede.
var Revision string

// Version devuelve la revisión marcada o, si falta, la que Go incrusta desde
// Git; lo no conforme queda como "desconocida".
func Version() string {
	if v := domain.NormalizarVersionBinario(strings.TrimSpace(Revision)); v != domain.VersionBinarioDesconocida {
		return v
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" {
				return domain.NormalizarVersionBinario(s.Value)
			}
		}
	}
	return domain.VersionBinarioDesconocida
}

// Entorno usa el valor declarado (VEC_ENTORNO) si existe y, si no, el perfil
// de ejecución con el que arranca el proceso.
func Entorno(declarado, perfil string) string {
	if declarado = strings.TrimSpace(declarado); declarado != "" {
		return string(domain.NormalizarEntornoIncidenciaTecnica(declarado))
	}
	switch strings.ToLower(strings.TrimSpace(perfil)) {
	case "desarrollo", "cidonia":
		return "desarrollo"
	case "presentacion_rrhh", "presentacion":
		return "presentacion"
	case "pruebas":
		return "pruebas"
	case "produccion":
		return "produccion"
	}
	return "desconocido"
}

// UmbralLenta es la duración a partir de la cual una petición se registra
// con nivel WARN y lenta=true. VEC_TELEMETRIA_LENTA_MS la cambia.
func UmbralLenta(getenv func(string) string) time.Duration {
	if n, err := strconv.Atoi(strings.TrimSpace(getenv("VEC_TELEMETRIA_LENTA_MS"))); err == nil && n > 0 && n <= 600_000 {
		return time.Duration(n) * time.Millisecond
	}
	return 300 * time.Millisecond
}

// Opciones del registro de acceso.
type Opciones struct {
	Destino    io.Writer // salida de las líneas JSON (stderr en los binarios)
	Servicio   string    // vec-server, vec-admin, vec-publico
	Superficie string    // interno, externo, integrada, administracion, publica
	Entorno    string
	Lenta      time.Duration
	// Consultas: a partir de cuántas consultas SQL la petición se marca
	// lenta aunque sea rápida (señal de una consulta por fila).
	Consultas int64
	// Diagnostico es la escucha interna de /debug/vars y /debug/pprof
	// (VEC_DIAGNOSTICO_ESCUCHA); vacía, no se abre.
	Diagnostico string
}

// UmbralConsultas lee VEC_TELEMETRIA_LENTA_CONSULTAS (20 por defecto).
func UmbralConsultas(getenv func(string) string) int64 {
	if n, err := strconv.Atoi(strings.TrimSpace(getenv("VEC_TELEMETRIA_LENTA_CONSULTAS"))); err == nil && n > 0 {
		return int64(n)
	}
	return 20
}

// Montar envuelve el manejador del servidor con el registro de acceso. Se
// llama antes de la supervisión de respuestas para que esta quede por fuera
// y aporte la correlación que enlaza con las incidencias técnicas.
func Montar(srv *http.Server, o Opciones) {
	if srv == nil || srv.Handler == nil || o.Destino == nil {
		return
	}
	srv.Handler = Middleware(o, srv.Handler)
	srv.RegisterOnShutdown(arrancarDiagnostico(o.Diagnostico, o.Destino))
}

// Middleware escribe una línea por petición al terminar.
func Middleware(o Opciones, siguiente http.Handler) http.Handler {
	if o.Lenta <= 0 {
		o.Lenta = 300 * time.Millisecond
	}
	if o.Consultas <= 0 {
		o.Consultas = 20
	}
	// Nombres de campo de las convenciones semánticas de OpenTelemetry
	// (service.*, deployment.*, http.*, url.*, error.type); los propios de
	// VEC van en el espacio vec.*.
	registro := slog.New(slog.NewJSONHandler(o.Destino, nil)).With(
		"service.name", o.Servicio, "service.version", Version(),
		"deployment.environment.name", o.Entorno, "vec.superficie", o.Superficie)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		ctx := r.Context()
		correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
		if !ok {
			if c, err := ports.ConCorrelacionIncidenciasPeticion(ctx); err == nil {
				ctx = c
				correlacion, _ = ports.CorrelacionIncidenciasPeticion(ctx)
			}
		}
		bd := &medida{}
		r = r.WithContext(context.WithValue(ctx, claveMedida{}, bd))
		e := &escritor{ResponseWriter: w}
		completada := false
		enCurso.Add(1)
		defer func() {
			enCurso.Add(-1)
			// Sin recover: un pánico sigue hasta la supervisión, que responde.
			duracion := time.Since(inicio)
			estado := e.estado
			if estado == 0 {
				estado = http.StatusOK
			}
			if !completada {
				estado = http.StatusInternalServerError
			}
			nivel := slog.LevelInfo
			consultas := bd.consultas.Load()
			lenta := duracion >= o.Lenta || consultas >= o.Consultas
			if lenta {
				nivel = slog.LevelWarn
			}
			if estado >= 500 {
				nivel = slog.LevelError
			}
			ruta := rutaOCamino(r.Pattern, r.URL.Path, estado)
			atributos := []slog.Attr{
				slog.String("http.request.method", metodo(r.Method)),
				ruta,
				slog.Int("http.response.status_code", estado),
				slog.Float64("http.server.request.duration", segundos(duracion)),
				slog.Int64("http.response.body.size", e.bytes),
				slog.String("vec.correlacion", correlacion),
				slog.Int64("vec.bd.consultas", consultas),
				slog.Float64("vec.bd.duracion", segundos(time.Duration(bd.bd.Load()))),
			}
			if espera := time.Duration(bd.espera.Load()); espera > 0 {
				atributos = append(atributos, slog.Float64("vec.bd.espera_conexion", segundos(espera)))
			}
			bd.mu.Lock()
			if bd.ultimoErr != "" {
				atributos = append(atributos, slog.String("vec.bd.error", bd.ultimoErr))
			}
			if estado >= 500 {
				// error.type: la clase del error de base de datos si la hay; si
				// no, el código de estado, como pide OpenTelemetry.
				tipo := strconv.Itoa(estado)
				if bd.ultimoErr != "" {
					tipo = bd.ultimoErr
				}
				atributos = append(atributos, slog.String("error.type", tipo))
			}
			if lenta {
				// El análisis del texto SQL solo se hace en las lentas.
				atributos = append(atributos, slog.Bool("vec.lenta", true))
				if bd.sqlMaxima != "" {
					atributos = append(atributos, slog.String("vec.bd.consulta_mas_lenta", operacion(bd.sqlMaxima)),
						slog.Float64("vec.bd.consulta_mas_lenta.duracion", segundos(bd.maxima)))
				}
			}
			bd.mu.Unlock()
			observar(metodo(r.Method), ruta.Value.String(), estado, duracion, lenta)
			if !completada {
				atributos = append(atributos, slog.Bool("vec.interrumpida", true))
			}
			if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
				atributos = append(atributos, slog.String("vec.cancelada", "plazo"))
			} else if err != nil {
				atributos = append(atributos, slog.String("vec.cancelada", "cliente"))
			}
			registro.LogAttrs(context.Background(), nivel, "http.server.request", atributos...)
		}()
		siguiente.ServeHTTP(e, r)
		completada = true
	})
}

// rutaOCamino sigue a OpenTelemetry: http.route solo cuando el enrutador
// dio la plantilla ("/x/{ref}"); si no, url.path depurado, con cada tramo que
// pueda ser un valor cambiado por {valor}. Un 4xx sin plantilla no copia el
// camino: puede ser lo que escribió la persona.
//
// Solo se conservan tramos de minúsculas, guion, guion bajo y punto. Los
// caminos de VEC llevan referencias opacas (con cifras o «:»), nunca nombres,
// un camino que no existe responde 404 u otro 4xx y los estáticos salen de una
// lista positiva. Una ruta nueva con texto libre en el camino debe registrarse
// con patrón en un http.ServeMux para que se vea su plantilla.
func rutaOCamino(patron, camino string, estado int) slog.Attr {
	if i := strings.IndexByte(patron, '/'); i >= 0 && !strings.HasSuffix(patron, "/") {
		return slog.String("http.route", strings.TrimSuffix(patron[i:], "{$}"))
	}
	if estado >= 400 && estado <= 499 {
		return slog.String("url.path", "{sin_plantilla}")
	}
	tramos := strings.Split(strings.TrimPrefix(camino, "/"), "/")
	if len(tramos) > 16 {
		tramos = append(tramos[:16], "{mas}")
	}
	for i, t := range tramos {
		if !esTramoFijo(t) {
			tramos[i] = "{valor}"
		}
	}
	return slog.String("url.path", "/"+strings.Join(tramos, "/"))
}

// esTramoFijo admite minúsculas, guion, guion bajo y punto (hasta 40) o
// versiones v1..v99; cifras, mayúsculas y otros signos indican un valor.
func esTramoFijo(t string) bool {
	if len(t) > 40 {
		return false
	}
	if len(t) >= 2 && len(t) <= 3 && t[0] == 'v' && strings.Trim(t[1:], "0123456789") == "" {
		return true
	}
	return strings.Trim(t, "abcdefghijklmnopqrstuvwxyz-_.") == ""
}

func metodo(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions:
		return m
	}
	return "_OTHER" // valor de OpenTelemetry para métodos no conocidos
}

// segundos redondea a la décima de milisegundo; OpenTelemetry mide las
// duraciones en segundos.
func segundos(d time.Duration) float64 {
	return math.Round(d.Seconds()*1e4) / 1e4
}

// escritor observa estado y bytes sin cambiar la respuesta; conserva Flush,
// ReadFrom (sendfile) y Unwrap para http.ResponseController.
type escritor struct {
	http.ResponseWriter
	estado int
	bytes  int64
}

func (w *escritor) WriteHeader(estado int) {
	if w.estado == 0 && (estado >= 200 || estado == http.StatusSwitchingProtocols) {
		w.estado = estado
	}
	w.ResponseWriter.WriteHeader(estado)
}

func (w *escritor) Write(b []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

func (w *escritor) ReadFrom(origen io.Reader) (int64, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	n, err := io.Copy(w.ResponseWriter, origen)
	w.bytes += n
	return n, err
}

func (w *escritor) Flush() {
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *escritor) Unwrap() http.ResponseWriter { return w.ResponseWriter }
