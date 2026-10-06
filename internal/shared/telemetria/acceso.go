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
}

// Montar envuelve el manejador del servidor con el registro de acceso. Se
// llama antes de la supervisión de respuestas para que esta quede por fuera
// y aporte la correlación que enlaza con las incidencias técnicas.
func Montar(srv *http.Server, o Opciones) {
	if srv == nil || srv.Handler == nil || o.Destino == nil {
		return
	}
	srv.Handler = Middleware(o, srv.Handler)
}

// Middleware escribe una línea por petición al terminar.
func Middleware(o Opciones, siguiente http.Handler) http.Handler {
	if o.Lenta <= 0 {
		o.Lenta = 300 * time.Millisecond
	}
	registro := slog.New(slog.NewJSONHandler(o.Destino, nil)).With(
		"servicio", o.Servicio, "superficie", o.Superficie, "entorno", o.Entorno, "version", Version())
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
		r = r.WithContext(ctx)
		e := &escritor{ResponseWriter: w}
		completada := false
		defer func() {
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
			lenta := duracion >= o.Lenta
			if lenta {
				nivel = slog.LevelWarn
			}
			if estado >= 500 {
				nivel = slog.LevelError
			}
			atributos := []slog.Attr{
				slog.String("correlacion", correlacion),
				slog.String("metodo", metodo(r.Method)),
				slog.String("ruta", ruta(r.Pattern, r.URL.Path, estado)),
				slog.Int("estado", estado),
				slog.Float64("duracion_ms", ms(duracion)),
				slog.Int64("bytes", e.bytes),
			}
			if lenta {
				atributos = append(atributos, slog.Bool("lenta", true))
			}
			if !completada {
				atributos = append(atributos, slog.Bool("interrumpida", true))
			}
			if err := ctx.Err(); errors.Is(err, context.DeadlineExceeded) {
				atributos = append(atributos, slog.String("cancelada", "plazo"))
			} else if err != nil {
				atributos = append(atributos, slog.String("cancelada", "cliente"))
			}
			registro.LogAttrs(context.Background(), nivel, "peticion", atributos...)
		}()
		siguiente.ServeHTTP(e, r)
		completada = true
	})
}

// ruta devuelve la plantilla del enrutador si la hay ("/x/{ref}") o el camino
// con cada tramo que pueda ser un valor sustituido por {valor}. Un 4xx sin
// plantilla no copia el camino: puede ser lo que escribió la persona.
func ruta(patron, camino string, estado int) string {
	if i := strings.IndexByte(patron, '/'); i >= 0 && !strings.HasSuffix(patron, "/") {
		return strings.TrimSuffix(patron[i:], "{$}")
	}
	if estado >= 400 && estado <= 499 {
		return "{sin_plantilla}"
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
	return "/" + strings.Join(tramos, "/")
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
	return "OTRO"
}

func ms(d time.Duration) float64 {
	return math.Round(float64(d)/float64(time.Millisecond)*10) / 10
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
