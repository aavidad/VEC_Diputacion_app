// Package diagnostico sirve las métricas y los perfiles de rendimiento
// (pprof) de un proceso de VEC a Sistemas, en una escucha propia separada de
// los portales.
//
// Denegación por defecto:
//   - está apagada si no se configura VEC_DIAGNOSTICO_ESCUCHA;
//   - la escucha solo puede ser una dirección de bucle local literal
//     (127.0.0.0/8 o ::1), y además se rechaza cualquier petición que no
//     llegue desde el bucle local;
//   - cada petición exige el token de VEC_DIAGNOSTICO_TOKEN_FILE (fichero
//     regular, sin permisos de grupo ni de otros, de 32 a 4096 bytes);
//   - solo admite GET y HEAD, nunca emite cookies y no usa el
//     http.DefaultServeMux (no importa net/http/pprof, que se registraría
//     ahí).
//
// Una configuración no válida deja una línea fija en el registro y la
// escucha no se abre; el proceso principal sigue funcionando.
package diagnostico

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Variables de entorno.
const (
	EnvEscucha       = "VEC_DIAGNOSTICO_ESCUCHA"
	EnvTokenFile     = "VEC_DIAGNOSTICO_TOKEN_FILE"
	minToken         = 32
	maxToken         = 4096
	maxSegundosCPU   = 60
	maxSegundosTraza = 10
)

// Errores de configuración, sin detalles de rutas ni valores.
var (
	ErrEscuchaNoLocal = errors.New("diagnostico: la escucha debe ser una direccion de bucle local")
	ErrToken          = errors.New("diagnostico: fichero de token no valido")
)

// Fuente escribe métricas en formato de texto de Prometheus.
type Fuente func(io.Writer)

// Configuracion de la superficie de diagnóstico.
type Configuracion struct {
	Escucha string
	Token   []byte
	Fuentes []Fuente
}

// Servidor es la superficie de diagnóstico en marcha.
type Servidor struct {
	srv       *http.Server
	oyente    net.Listener
	cerrarUna sync.Once
}

// LeerToken lee y comprueba el fichero del token.
func LeerToken(ruta string) ([]byte, error) {
	if ruta == "" {
		return nil, ErrToken
	}
	info, err := os.Lstat(ruta)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || info.Size() > maxToken {
		return nil, ErrToken
	}
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return nil, ErrToken
	}
	token := bytes.TrimSpace(contenido)
	if len(token) < minToken || len(token) > maxToken {
		return nil, ErrToken
	}
	for _, c := range token {
		if c < 0x21 || c > 0x7e {
			return nil, ErrToken
		}
	}
	return token, nil
}

// esEscuchaLocal admite solo IP literal de bucle local con puerto.
func esEscuchaLocal(escucha string) bool {
	direccion, err := netip.ParseAddrPort(escucha)
	return err == nil && direccion.Addr().IsLoopback() && direccion.Port() != 0
}

// Arrancar abre la escucha y sirve en segundo plano.
func Arrancar(c Configuracion) (*Servidor, error) {
	if !esEscuchaLocal(c.Escucha) {
		return nil, ErrEscuchaNoLocal
	}
	if len(c.Token) < minToken {
		return nil, ErrToken
	}
	oyente, err := net.Listen("tcp", c.Escucha)
	if err != nil {
		return nil, fmt.Errorf("diagnostico: escucha no disponible")
	}
	s := &Servidor{oyente: oyente}
	s.srv = &http.Server{
		Handler:           Manejador(c.Token, c.Fuentes),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      (maxSegundosCPU + 30) * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 << 10,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	go func() { _ = s.srv.Serve(oyente) }()
	return s, nil
}

// Direccion devuelve la dirección real de escucha.
func (s *Servidor) Direccion() string {
	if s == nil {
		return ""
	}
	return s.oyente.Addr().String()
}

// Cerrar detiene la escucha. Es idempotente.
func (s *Servidor) Cerrar(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var err error
	s.cerrarUna.Do(func() { err = s.srv.Shutdown(ctx) })
	return err
}

// Manejador devuelve el manejador protegido. Se exporta para probarlo.
func Manejador(token []byte, fuentes []Fuente) http.Handler {
	huella := sha256.Sum256(token)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		var b bytes.Buffer
		for _, f := range fuentes {
			if f != nil {
				f(&b)
			}
		}
		_, _ = w.Write(b.Bytes())
	})
	mux.HandleFunc("/debug/pprof/", indicePerfiles)
	mux.HandleFunc("/debug/pprof/profile", perfilCPU)
	mux.HandleFunc("/debug/pprof/trace", trazaEjecucion)
	for _, nombre := range []string{"heap", "allocs", "goroutine", "threadcreate", "block", "mutex"} {
		mux.HandleFunc("/debug/pprof/"+nombre, perfilNombrado(nombre))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cabeceras := w.Header()
		cabeceras.Set("Cache-Control", "no-store")
		cabeceras.Set("X-Content-Type-Options", "nosniff")
		if !desdeBucleLocal(r.RemoteAddr) {
			http.Error(w, "prohibido", http.StatusForbidden)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			cabeceras.Set("Allow", "GET, HEAD")
			http.Error(w, "metodo no admitido", http.StatusMethodNotAllowed)
			return
		}
		if !tokenValido(r.Header.Get("Authorization"), huella) {
			cabeceras.Set("WWW-Authenticate", `Bearer realm="vec-diagnostico"`)
			http.Error(w, "no autorizado", http.StatusUnauthorized)
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func desdeBucleLocal(remota string) bool {
	direccion, err := netip.ParseAddrPort(remota)
	return err == nil && direccion.Addr().Unmap().IsLoopback()
}

// tokenValido compara huellas de longitud fija en tiempo constante.
func tokenValido(cabecera string, huella [32]byte) bool {
	recibido, ok := strings.CutPrefix(cabecera, "Bearer ")
	if !ok || recibido == "" {
		return false
	}
	h := sha256.Sum256([]byte(recibido))
	return subtle.ConstantTimeCompare(h[:], huella[:]) == 1
}

func indicePerfiles(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/debug/pprof/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, strings.Join([]string{
		"/debug/pprof/profile?seconds=30   CPU (1-60 s)",
		"/debug/pprof/trace?seconds=5      traza de ejecución (1-10 s)",
		"/debug/pprof/heap                 memoria en uso",
		"/debug/pprof/allocs               reservas de memoria",
		"/debug/pprof/goroutine?debug=2    pilas de todas las gorrutinas",
		"/debug/pprof/threadcreate         hilos del sistema",
		"/debug/pprof/block, /mutex        vacíos salvo que se activen",
		"",
	}, "\n"))
}

func segundos(r *http.Request, predeterminado, maximo int) (int, bool) {
	valor := r.URL.Query().Get("seconds")
	if valor == "" {
		return predeterminado, true
	}
	n, err := strconv.Atoi(valor)
	if err != nil || n < 1 || n > maximo {
		return 0, false
	}
	return n, true
}

func perfilCPU(w http.ResponseWriter, r *http.Request) {
	n, ok := segundos(r, 30, maxSegundosCPU)
	if !ok {
		http.Error(w, "seconds fuera de rango", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="cpu.pprof"`)
	if err := pprof.StartCPUProfile(w); err != nil {
		w.Header().Del("Content-Disposition")
		http.Error(w, "ya hay un perfil de CPU en curso", http.StatusConflict)
		return
	}
	esperar(r.Context(), time.Duration(n)*time.Second)
	pprof.StopCPUProfile()
}

func trazaEjecucion(w http.ResponseWriter, r *http.Request) {
	n, ok := segundos(r, 1, maxSegundosTraza)
	if !ok {
		http.Error(w, "seconds fuera de rango", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", `attachment; filename="traza.out"`)
	if err := trace.Start(w); err != nil {
		w.Header().Del("Content-Disposition")
		http.Error(w, "ya hay una traza en curso", http.StatusConflict)
		return
	}
	esperar(r.Context(), time.Duration(n)*time.Second)
	trace.Stop()
}

func esperar(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

func perfilNombrado(nombre string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		perfil := pprof.Lookup(nombre)
		if perfil == nil {
			http.NotFound(w, r)
			return
		}
		debug, _ := strconv.Atoi(r.URL.Query().Get("debug"))
		if debug < 0 || debug > 2 {
			debug = 0
		}
		if debug > 0 {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="`+nombre+`.pprof"`)
		}
		_ = perfil.WriteTo(w, debug)
	}
}

// MontarDesdeEntorno arranca la superficie si VEC_DIAGNOSTICO_ESCUCHA está
// configurada. Cualquier fallo deja una línea fija en avisos y no abre nada.
// Devuelve la función de cierre (no hace nada si no arrancó).
func MontarDesdeEntorno(getenv func(string) string, servicio string, fuentes []Fuente, avisos io.Writer) func() {
	nada := func() {}
	if getenv == nil || strings.TrimSpace(getenv(EnvEscucha)) == "" {
		return nada
	}
	avisar := func(texto string) {
		if avisos != nil {
			_, _ = io.WriteString(avisos, servicio+": "+texto+"\n")
		}
	}
	token, err := LeerToken(strings.TrimSpace(getenv(EnvTokenFile)))
	if err != nil {
		avisar("diagnostico no arrancado: fichero de token no valido")
		return nada
	}
	s, err := Arrancar(Configuracion{Escucha: strings.TrimSpace(getenv(EnvEscucha)), Token: token, Fuentes: fuentes})
	switch {
	case errors.Is(err, ErrEscuchaNoLocal):
		avisar("diagnostico no arrancado: la escucha debe ser 127.0.0.1:puerto o [::1]:puerto")
		return nada
	case err != nil:
		avisar("diagnostico no arrancado: escucha no disponible")
		return nada
	}
	avisar("diagnostico escuchando en bucle local")
	return func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = s.Cerrar(ctx)
	}
}
