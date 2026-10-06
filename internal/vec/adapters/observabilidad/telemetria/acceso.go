package telemetria

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

// Esquemas de las líneas JSON que escribe el registro. Cada línea es un
// objeto completo terminado en salto de línea.
const (
	EsquemaAcceso    = "vec.acceso_http.v1"
	EsquemaEnCurso   = "vec.peticion_en_curso.v1"
	EsquemaDescartes = "vec.acceso_http_descartes.v1"
)

// Valores predeterminados. Sistemas puede cambiarlos por entorno (ver
// UmbralesDeEntorno).
const (
	UmbralLentaPredeterminado   = 300 * time.Millisecond
	UmbralEnCursoPredeterminado = 10 * time.Second
	capacidadColaPredeterminada = 8192
	// maxBloqueEscritura no supera PIPE_BUF: una escritura de ese tamaño a
	// una tubería es atómica y no se mezcla con otras líneas del proceso.
	maxBloqueEscritura = 4096
	periodoVigilancia  = time.Second
	// maxLineasPorVuelta deja respirar a la vigilancia bajo carga continua.
	maxLineasPorVuelta = 512
)

// ErrDestinoAccesoAusente indica que no se configuró dónde escribir.
var ErrDestinoAccesoAusente = errors.New("telemetria: destino del registro de acceso ausente")

// Opciones del registro de acceso.
type Opciones struct {
	// Destino recibe las líneas JSON. Lo escribe un único trabajador propio:
	// un destino lento llena la cola y descarta, nunca frena las peticiones.
	Destino io.Writer
	// Servicio es el nombre del binario (vec-server, vec-admin, vec-publico).
	Servicio string
	// Superficie distingue el portal atendido (interno, externo, publica…).
	Superficie string
	// Entorno y Version identifican el despliegue (ver EntornoDe y
	// RevisionBinario).
	Entorno string
	Version string
	// Umbrales para marcar una petición como lenta y avisar de las que
	// siguen en curso.
	Umbrales Umbrales
	// Capacidad de la cola de líneas; 0 usa la predeterminada.
	Capacidad int
	// Reloj opcional, para pruebas.
	Reloj func() time.Time
}

// Umbrales de aviso. Un valor cero usa el predeterminado.
type Umbrales struct {
	// Lenta: duración a partir de la cual la petición se marca "lenta".
	Lenta time.Duration
	// EnCurso: duración a partir de la cual una petición que todavía no ha
	// terminado deja una línea propia; se repite al triple, nueve veces…
	EnCurso time.Duration
}

// Registro escribe una línea por petición atendida y vigila las que tardan.
// Es seguro para uso concurrente.
type Registro struct {
	destino    io.Writer
	servicio   string
	superficie string
	entorno    string
	version    string
	umbrales   Umbrales
	reloj      func() time.Time

	cola        chan []byte
	descartadas atomic.Uint64
	fallos      atomic.Uint64
	activas     sync.Map // *Ficha
	enCurso     atomic.Int64

	parar     chan struct{}
	terminado chan struct{}
	pararUna  sync.Once
}

// NuevoRegistro crea el registro y arranca su trabajador.
func NuevoRegistro(o Opciones) (*Registro, error) {
	if o.Destino == nil {
		return nil, ErrDestinoAccesoAusente
	}
	if o.Umbrales.Lenta <= 0 {
		o.Umbrales.Lenta = UmbralLentaPredeterminado
	}
	if o.Umbrales.EnCurso <= 0 {
		o.Umbrales.EnCurso = UmbralEnCursoPredeterminado
	}
	if o.Capacidad <= 0 {
		o.Capacidad = capacidadColaPredeterminada
	}
	if o.Reloj == nil {
		o.Reloj = time.Now
	}
	r := &Registro{
		destino:    o.Destino,
		servicio:   valorOCampo(identificadorSeguro(o.Servicio, 40)),
		superficie: valorOCampo(identificadorSeguro(o.Superficie, 40)),
		entorno:    valorOCampo(identificadorSeguro(o.Entorno, 40)),
		version:    valorOCampo(identificadorSeguro(o.Version, 40)),
		umbrales:   o.Umbrales,
		reloj:      o.Reloj,
		cola:       make(chan []byte, o.Capacidad),
		parar:      make(chan struct{}),
		terminado:  make(chan struct{}),
	}
	go r.trabajar()
	return r, nil
}

func valorOCampo(v string) string {
	if v == "" {
		return "desconocido"
	}
	return v
}

// Envolver devuelve el middleware de acceso. Debe quedar dentro de la
// supervisión de respuestas, para heredar su correlación; si no la hay, crea
// una propia.
func (reg *Registro) Envolver(siguiente http.Handler) http.Handler {
	if reg == nil || siguiente == nil {
		return siguiente
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		correlacion, ok := ports.CorrelacionIncidenciasPeticion(ctx)
		if !ok {
			if nuevo, err := ports.ConCorrelacionIncidenciasPeticion(ctx); err == nil {
				ctx = nuevo
				correlacion, _ = ports.CorrelacionIncidenciasPeticion(ctx)
			}
		}
		f := &Ficha{
			inicio:          reg.reloj(),
			metodo:          metodoSeguro(r.Method),
			rutaNormalizada: normalizarCamino(r.URL.Path),
			correlacion:     correlacion,
		}
		ctx = conFicha(ctx, f)
		peticion := r.WithContext(ctx)
		escritor := &escritorMedido{ResponseWriter: w}
		reg.activas.Store(f, struct{}{})
		reg.enCurso.Add(1)
		completada := false
		defer func() {
			reg.activas.Delete(f)
			reg.enCurso.Add(-1)
			// Sin recover: el pánico sigue hacia la supervisión, que lo
			// contiene y responde; aquí solo queda constancia.
			reg.escribirAcceso(peticion, f, escritor, !completada)
		}()
		siguiente.ServeHTTP(escritor, peticion)
		AnotarRuta(ctx, peticion.Pattern)
		completada = true
	})
}

// EnCurso devuelve las peticiones que se están atendiendo ahora.
func (reg *Registro) EnCurso() int64 {
	if reg == nil {
		return 0
	}
	return reg.enCurso.Load()
}

// Descartadas devuelve las líneas perdidas por cola llena desde el arranque.
func (reg *Registro) Descartadas() uint64 {
	if reg == nil {
		return 0
	}
	return reg.descartadas.Load()
}

// Cerrar vacía la cola y detiene el trabajador. Es idempotente.
func (reg *Registro) Cerrar(ctx context.Context) error {
	if reg == nil {
		return nil
	}
	reg.pararUna.Do(func() { close(reg.parar) })
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-reg.terminado:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type lineaAcceso struct {
	Esquema      string  `json:"esquema"`
	Instante     string  `json:"instante"`
	Nivel        string  `json:"nivel"`
	Servicio     string  `json:"servicio"`
	Superficie   string  `json:"superficie"`
	Entorno      string  `json:"entorno"`
	Version      string  `json:"version"`
	Correlacion  string  `json:"correlacion"`
	Metodo       string  `json:"metodo"`
	Ruta         string  `json:"ruta"`
	Estado       int     `json:"estado"`
	DuracionMS   float64 `json:"duracion_ms"`
	Bytes        int64   `json:"bytes"`
	Lenta        bool    `json:"lenta,omitempty"`
	Interrumpida bool    `json:"interrumpida,omitempty"`
	Cancelada    string  `json:"cancelada,omitempty"`
	Causa        string  `json:"causa,omitempty"`
	EtapaFallo   string  `json:"etapa_fallo,omitempty"`
	Incidencia   string  `json:"incidencia,omitempty"`
	Componente   string  `json:"componente,omitempty"`
	EtapaInc     string  `json:"etapa_incidencia,omitempty"`
}

func (reg *Registro) escribirAcceso(r *http.Request, f *Ficha, e *escritorMedido, interrumpida bool) {
	duracion := reg.reloj().Sub(f.inicio)
	estado := e.estado
	if estado == 0 {
		estado = http.StatusOK
		if interrumpida {
			estado = http.StatusInternalServerError
		}
	}
	f.mu.Lock()
	linea := lineaAcceso{
		Esquema:      EsquemaAcceso,
		Instante:     formatoInstante(f.inicio),
		Servicio:     reg.servicio,
		Superficie:   reg.superficie,
		Entorno:      reg.entorno,
		Version:      reg.version,
		Correlacion:  f.correlacion,
		Metodo:       f.metodo,
		Ruta:         f.rutaFinal(estado),
		Estado:       estado,
		DuracionMS:   milisegundos(duracion),
		Bytes:        e.bytes,
		Interrumpida: interrumpida,
		Causa:        f.causa,
		EtapaFallo:   f.etapaFallo,
		Incidencia:   f.incidencia,
		Componente:   f.componente,
		EtapaInc:     f.etapaInc,
	}
	f.mu.Unlock()
	switch err := r.Context().Err(); {
	case errors.Is(err, context.DeadlineExceeded):
		linea.Cancelada = "plazo"
	case err != nil:
		linea.Cancelada = "cliente"
	}
	linea.Lenta = duracion >= reg.umbrales.Lenta
	switch {
	case estado >= http.StatusInternalServerError || interrumpida:
		linea.Nivel = "error"
	case linea.Lenta:
		linea.Nivel = "aviso"
	default:
		linea.Nivel = "info"
	}
	reg.encolar(linea)
}

// rutaFinal exige f.mu tomado. Con plantilla anotada manda la plantilla; sin
// ella, solo se escribe el camino normalizado si la respuesta la produjo una
// ruta servida (2xx, 3xx o 5xx).
func (f *Ficha) rutaFinal(estado int) string {
	if f.ruta != "" {
		return f.ruta
	}
	if estado == http.StatusNotFound {
		return rutaNoEncontrada
	}
	if estado >= 400 && estado <= 499 {
		// Sin plantilla, un 4xx puede venir de una frontera previa al
		// enrutador (sesión, permisos) con un camino que escribió la persona.
		return rutaSinPlantilla
	}
	return f.rutaNormalizada
}

func (reg *Registro) encolar(v any) {
	contenido, err := json.Marshal(v)
	if err != nil {
		reg.fallos.Add(1)
		return
	}
	contenido = append(contenido, '\n')
	select {
	case reg.cola <- contenido:
	default:
		reg.descartadas.Add(1)
	}
}

// trabajar es el único escritor del destino. Agrupa las líneas disponibles
// en bloques de hasta maxBloqueEscritura bytes y, cada segundo, revisa las
// peticiones que siguen en curso.
func (reg *Registro) trabajar() {
	defer close(reg.terminado)
	vigilancia := time.NewTicker(periodoVigilancia)
	defer vigilancia.Stop()
	bloque := make([]byte, 0, maxBloqueEscritura)
	var informadas uint64
	for {
		select {
		case linea := <-reg.cola:
			bloque = reg.vaciarCola(bloque, linea)
		case <-vigilancia.C:
			reg.vigilarEnCurso()
		case <-reg.parar:
			for len(reg.cola) > 0 {
				bloque = reg.vaciarCola(bloque, <-reg.cola)
			}
			reg.informarDescartes(&informadas)
			return
		}
		reg.informarDescartes(&informadas)
	}
}

// vaciarCola escribe la línea recibida y las que ya esperan en la cola,
// agrupadas sin superar maxBloqueEscritura salvo una línea más larga, que
// sale sola. Devuelve el bloque vacío para reutilizarlo.
func (reg *Registro) vaciarCola(bloque, linea []byte) []byte {
	bloque = bloque[:0]
	for leidas := 1; ; leidas++ {
		if len(bloque) > 0 && len(bloque)+len(linea) > maxBloqueEscritura {
			reg.escribir(bloque)
			bloque = bloque[:0]
		}
		bloque = append(bloque, linea...)
		if leidas < maxLineasPorVuelta {
			select {
			case linea = <-reg.cola:
				continue
			default:
			}
		}
		reg.escribir(bloque)
		return bloque[:0]
	}
}

func (reg *Registro) escribir(bloque []byte) {
	if _, err := reg.destino.Write(bloque); err != nil {
		reg.fallos.Add(1)
	}
}

func (reg *Registro) informarDescartes(informadas *uint64) {
	total := reg.descartadas.Load()
	if total == *informadas {
		return
	}
	nuevas := total - *informadas
	*informadas = total
	linea := `{"esquema":"` + EsquemaDescartes + `","instante":"` + formatoInstante(reg.reloj()) +
		`","nivel":"aviso","servicio":"` + reg.servicio + `","descartadas":` + strconv.FormatUint(nuevas, 10) + "}\n"
	reg.escribir([]byte(linea))
}

type lineaEnCurso struct {
	Esquema        string  `json:"esquema"`
	Instante       string  `json:"instante"`
	Nivel          string  `json:"nivel"`
	Servicio       string  `json:"servicio"`
	Superficie     string  `json:"superficie"`
	Entorno        string  `json:"entorno"`
	Version        string  `json:"version"`
	Correlacion    string  `json:"correlacion"`
	Metodo         string  `json:"metodo"`
	Ruta           string  `json:"ruta"`
	Llegada        string  `json:"llegada"`
	TranscurridoMS float64 `json:"transcurrido_ms"`
}

// vigilarEnCurso deja una línea por cada petición que supera el umbral de
// "en curso" y la repite al triple de tiempo (10 s, 30 s, 90 s…), de modo que
// una petición atascada se ve mientras ocurre y no solo cuando termina.
func (reg *Registro) vigilarEnCurso() {
	ahora := reg.reloj()
	reg.activas.Range(func(clave, _ any) bool {
		f := clave.(*Ficha)
		transcurrido := ahora.Sub(f.inicio)
		f.mu.Lock()
		siguiente := reg.umbrales.EnCurso
		for i := 0; i < f.avisosEnCurso && siguiente < math.MaxInt64/3; i++ {
			siguiente *= 3
		}
		if transcurrido < siguiente {
			f.mu.Unlock()
			return true
		}
		f.avisosEnCurso++
		linea := lineaEnCurso{
			Esquema:        EsquemaEnCurso,
			Instante:       formatoInstante(ahora),
			Nivel:          "aviso",
			Servicio:       reg.servicio,
			Superficie:     reg.superficie,
			Entorno:        reg.entorno,
			Version:        reg.version,
			Correlacion:    f.correlacion,
			Metodo:         f.metodo,
			Ruta:           f.rutaNormalizada,
			Llegada:        formatoInstante(f.inicio),
			TranscurridoMS: milisegundos(transcurrido),
		}
		f.mu.Unlock()
		if contenido, err := json.Marshal(linea); err == nil {
			reg.escribir(append(contenido, '\n'))
		}
		return true
	})
}

func formatoInstante(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// milisegundos redondea a décimas de milisegundo.
func milisegundos(d time.Duration) float64 {
	return math.Round(float64(d)/float64(time.Millisecond)*10) / 10
}

func metodoSeguro(m string) string {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace:
		return m
	default:
		return "OTRO"
	}
}

// escritorMedido observa el estado final y los bytes del cuerpo sin cambiar
// la respuesta. Conserva Flush, ReadFrom (sendfile) y Unwrap para
// http.ResponseController.
type escritorMedido struct {
	http.ResponseWriter
	estado int
	bytes  int64
}

func (w *escritorMedido) WriteHeader(estado int) {
	if w.estado == 0 && (estado < 100 || estado > 199 || estado == http.StatusSwitchingProtocols) {
		w.estado = estado
	}
	w.ResponseWriter.WriteHeader(estado)
}

func (w *escritorMedido) Write(contenido []byte) (int, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(contenido)
	w.bytes += int64(n)
	return n, err
}

func (w *escritorMedido) ReadFrom(origen io.Reader) (int64, error) {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	var n int64
	var err error
	if lector, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		n, err = lector.ReadFrom(origen)
	} else {
		n, err = io.Copy(w.ResponseWriter, origen)
	}
	w.bytes += n
	return n, err
}

func (w *escritorMedido) Flush() {
	_ = w.FlushError()
}

func (w *escritorMedido) FlushError() error {
	if w.estado == 0 {
		w.estado = http.StatusOK
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *escritorMedido) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
