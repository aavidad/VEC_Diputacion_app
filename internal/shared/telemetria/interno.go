package telemetria

import (
	"context"
	"expvar"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"vec-diputacion-granada/internal/shared/plazoarranque"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Métricas del proceso con expvar (JSON en /debug/vars de la escucha
// interna): peticiones por ruta y clase de estado, milisegundos acumulados
// por ruta, lentas, en curso y el estado de cada pool de PostgreSQL.
var (
	peticiones = expvar.NewMap("vec_http_peticiones")
	duraciones = expvar.NewMap("vec_http_ms")
	lentas     = expvar.NewInt("vec_http_lentas")
	enCurso    = expvar.NewInt("vec_http_en_curso")

	claves    sync.Map // claves ya vistas en los mapas
	numClaves atomic.Int64

	pools    sync.Map // *pgxpool.Pool -> nombre del rol
	numPools atomic.Int64
)

func init() {
	expvar.Publish("vec_bd_pools", expvar.Func(estadoPools))
}

// maxClaves acota la cardinalidad: las rutas nuevas por encima se suman en
// "otras".
const maxClaves = 1000

func clave(k string) string {
	if _, ok := claves.Load(k); ok {
		return k
	}
	if numClaves.Load() >= maxClaves {
		return "otras"
	}
	if _, cargada := claves.LoadOrStore(k, struct{}{}); !cargada {
		numClaves.Add(1)
	}
	return k
}

func observar(metodo, ruta string, estado int, d time.Duration, lenta bool) {
	base := metodo + " " + ruta
	peticiones.Add(clave(base+" "+string(rune('0'+estado/100))+"xx"), 1)
	duraciones.AddFloat(clave(base), segundos(d)*1000)
	if lenta {
		lentas.Add(1)
	}
}

func registrarPool(pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	if _, ok := pools.Load(pool); ok || numPools.Load() >= 256 {
		return
	}
	nombre := "pool"
	if c := pool.Config(); c != nil && c.ConnConfig != nil && esTramoFijo(c.ConnConfig.User) && c.ConnConfig.User != "" {
		nombre = c.ConnConfig.User // rol de base de datos, no una persona
	}
	if _, cargado := pools.LoadOrStore(pool, nombre); !cargado {
		numPools.Add(1)
	}
}

func estadoPools() any {
	salida := map[string]map[string]any{}
	pools.Range(func(k, v any) bool {
		s := k.(*pgxpool.Pool).Stat()
		nombre := v.(string)
		for i := 2; salida[nombre] != nil; i++ {
			nombre = v.(string) + "_" + strconv.Itoa(i)
		}
		salida[nombre] = map[string]any{
			"maximo": s.MaxConns(), "abiertas": s.TotalConns(), "en_uso": s.AcquiredConns(), "libres": s.IdleConns(),
			"prestamos": s.AcquireCount(), "prestamos_con_espera": s.EmptyAcquireCount(),
			"prestamos_cancelados": s.CanceledAcquireCount(), "espera_ms": s.AcquireDuration().Milliseconds(),
		}
		return true
	})
	return salida
}

// arrancarDiagnostico abre la escucha interna para Sistemas con /debug/vars
// (expvar) y /debug/pprof. Está apagada si escucha está vacía y solo admite
// una IP de bucle local literal; además rechaza peticiones que no lleguen del
// bucle local y todo lo que no sea GET o HEAD. Usa su propio ServeMux: los
// portales nunca la sirven.
func arrancarDiagnostico(escucha string, avisos io.Writer) func() {
	if escucha == "" {
		return func() {}
	}
	avisar := func(texto string) {
		if avisos != nil {
			_, _ = io.WriteString(avisos, "telemetria: "+texto+"\n")
		}
	}
	direccion, err := netip.ParseAddrPort(escucha)
	if err != nil || !direccion.Addr().IsLoopback() || direccion.Port() == 0 {
		avisar("diagnostico no arrancado: la escucha debe ser 127.0.0.1:puerto o [::1]:puerto")
		return func() {}
	}
	oyente, err := net.Listen("tcp", escucha)
	if err != nil {
		avisar("diagnostico no arrancado: escucha no disponible")
		return func() {}
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/vars", variables)
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	srv := &http.Server{
		Handler:           soloLocal(mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      2 * time.Minute, // perfiles de CPU de hasta 60 s
		ErrorLog:          log.New(io.Discard, "", 0),
	}
	go func() { _ = srv.Serve(oyente) }()
	avisar("diagnostico escuchando en bucle local")
	return func() {
		ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancelar()
		_ = srv.Shutdown(ctx)
	}
}

func soloLocal(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remota, err := netip.ParseAddrPort(r.RemoteAddr)
		switch {
		case err != nil || !remota.Addr().Unmap().IsLoopback() || !hostLocal(r.Host):
			// El Host también debe ser una IP de bucle local: un nombre
			// permitiría a una página web llegar aquí por rebinding de DNS.
			http.Error(w, "prohibido", http.StatusForbidden)
		case r.Method != http.MethodGet && r.Method != http.MethodHead:
			http.Error(w, "metodo no admitido", http.StatusMethodNotAllowed)
		default:
			w.Header().Set("Cache-Control", "no-store")
			siguiente.ServeHTTP(w, r)
		}
	})
}

// variables sirve las variables de expvar como /debug/vars, salvo "cmdline":
// los argumentos del proceso no son métricas y podrían llevar rutas.
func variables(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = io.WriteString(w, "{")
	primera := true
	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key == "cmdline" {
			return
		}
		if !primera {
			_, _ = io.WriteString(w, ",")
		}
		primera = false
		_, _ = io.WriteString(w, strconv.Quote(kv.Key)+":"+kv.Value.String())
	})
	_, _ = io.WriteString(w, "}\n")
}

// hostLocal admite solo una IP literal de bucle local, con o sin puerto.
func hostLocal(host string) bool {
	if direccion, err := netip.ParseAddrPort(host); err == nil {
		return direccion.Addr().Unmap().IsLoopback()
	}
	direccion, err := netip.ParseAddr(strings.Trim(host, "[]"))
	return err == nil && direccion.Unmap().IsLoopback()
}
