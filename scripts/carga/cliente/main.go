// Command cliente es un generador de carga local para medir el aguante de las
// rutas públicas de VEC. Vive fuera de la composición real: no importa ningún
// paquete del producto y solo habla HTTP con un servidor de laboratorio.
//
// Cada usuario virtual tiene su propia conexión TLS (como un navegador
// distinto), recorre en bucle el escenario elegido y espera un tiempo de
// lectura aleatorio entre peticiones. Mide percentiles por ruta, errores,
// bytes en la red y, si se indica, CPU y memoria del proceso servidor y las
// conexiones abiertas en PostgreSQL.
//
// Solo admite destinos de loopback: nunca debe apuntarse a cidonia ni a un
// servicio externo.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand/v2" // nosemgrep: math-random-used -- solo reparte rutas y pausas de la carga; no es material criptográfico
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type paso struct {
	nombre string
	ruta   string
}

// escenarios reproduce lo que pide un navegador. «pagina-bolsa» es la visita
// completa a /bolsa/ sin caché previa; «api» son solo las lecturas de datos.
func escenarios(bolsas int) map[string]func(r *rand.Rand) []paso {
	lista := func(r *rand.Rand) paso {
		n := 1 + r.IntN(bolsas)
		ruta := fmt.Sprintf("/api/publico/bolsa/bolsas/bolsa:carga-%03d/lista", n)
		if r.IntN(3) == 0 {
			ruta += "?limite=100&cursor=" + strconv.Itoa(2+r.IntN(400))
		}
		return paso{"lista", ruta}
	}
	return map[string]func(r *rand.Rand) []paso{
		"api": func(r *rand.Rand) []paso {
			return []paso{
				{"bolsas", "/api/publico/bolsa/bolsas"},
				lista(r),
				{"convocatorias", "/api/publico/bolsa/convocatorias"},
				{"categorias", "/api/publico/bolsa/categorias"},
			}
		},
		"lista": func(r *rand.Rand) []paso { return []paso{lista(r)} },
		"pagina-bolsa": func(r *rand.Rand) []paso {
			return []paso{
				{"html", "/bolsa/"},
				{"css", "/styles.css?v=20260715-theme"},
				{"css", "/comun/tema-vec.css?v=20261001-codexf-accesibilidad-v1"},
				{"css", "/bolsa/bolsa.css?v=20260925-aspecto-v1"},
				{"css", "/bolsa/bolsa_adaptable.css?v=20260924-bolsa-publica-final"},
				{"js", "/bolsa/bolsa.js?v=20261001-convoca-preparacion-v1"},
				{"js", "/bolsa/contrato-v2.js?v=20260930-codexe-publico-v2-v1"},
				{"js", "/bolsa/i18n-publica.js?v=20260930-codexe-publico-v2-v1"},
				{"js", "/bolsa/preparacion/entrada.js?v=20261001-convoca-preparacion-v1"},
				{"textos", "/textos/es/bolsas-publicas.json"},
				{"textos", "/textos/idiomas.json"},
				{"convocatorias", "/api/publico/bolsa/convocatorias"},
				{"categorias", "/api/publico/bolsa/categorias"},
				{"bolsas", "/api/publico/bolsa/bolsas"},
				lista(r),
			}
		},
	}
}

type muestra struct {
	ruta     string
	duracion time.Duration
	estado   int
	bytes    int64
	err      string
}

type resumenRuta struct {
	Ruta    string         `json:"ruta"`
	N       int            `json:"peticiones"`
	P50     float64        `json:"p50_ms"`
	P95     float64        `json:"p95_ms"`
	P99     float64        `json:"p99_ms"`
	Max     float64        `json:"max_ms"`
	Errores map[string]int `json:"errores,omitempty"`
	BytesMe float64        `json:"bytes_medios"`
}

type resumenNivel struct {
	Usuarios      int           `json:"usuarios"`
	Duracion      float64       `json:"duracion_s"`
	Peticiones    int           `json:"peticiones"`
	PorSegundo    float64       `json:"peticiones_s"`
	TasaError     float64       `json:"tasa_error"`
	MBs           float64       `json:"mb_s"`
	Rutas         []resumenRuta `json:"rutas"`
	CPUServidor   float64       `json:"cpu_servidor_pct_max"`
	RSSServidorMB float64       `json:"rss_servidor_mb_max"`
	PGConexiones  int           `json:"pg_conexiones_max"`
	PGActivas     int           `json:"pg_activas_max"`
	PGCPU         float64       `json:"pg_cpu_pct_max"`
}

func main() {
	base := flag.String("url", "https://localhost:18480", "origen del laboratorio (solo loopback)")
	ca := flag.String("ca", "", "certificado de la CA del laboratorio")
	niveles := flag.String("usuarios", "100,500,1000,3000", "usuarios virtuales por nivel")
	duracion := flag.Duration("duracion", 30*time.Second, "duración de cada nivel")
	calentamiento := flag.Duration("rampa", 5*time.Second, "tiempo para incorporar a todos los usuarios")
	escenario := flag.String("escenario", "api", "api | lista | pagina-bolsa")
	pausaMin := flag.Duration("pausa-min", 500*time.Millisecond, "pausa mínima entre peticiones de un usuario")
	pausaMax := flag.Duration("pausa-max", 1500*time.Millisecond, "pausa máxima entre peticiones de un usuario")
	bolsas := flag.Int("bolsas", 40, "bolsas sintéticas del laboratorio")
	gzip := flag.Bool("gzip", true, "enviar Accept-Encoding: gzip como un navegador")
	pid := flag.Int("pid", 0, "pid del servidor para medir CPU y memoria")
	contenedorPG := flag.String("pg-contenedor", "", "contenedor Docker de PostgreSQL para contar conexiones")
	salida := flag.String("json", "", "fichero donde guardar el resumen JSON")
	flag.Parse()

	destino, err := url.Parse(*base)
	if err != nil || !esLoopback(destino.Hostname()) {
		fmt.Fprintln(os.Stderr, "el destino debe ser loopback (127.0.0.1, ::1 o localhost)")
		os.Exit(2)
	}
	raices := x509.NewCertPool()
	if *ca != "" {
		pem, err := os.ReadFile(*ca)
		if err != nil || !raices.AppendCertsFromPEM(pem) {
			fmt.Fprintln(os.Stderr, "CA no válida")
			os.Exit(2)
		}
	}
	gen, ok := escenarios(*bolsas)[*escenario]
	if !ok {
		fmt.Fprintln(os.Stderr, "escenario desconocido")
		os.Exit(2)
	}
	var resultados []resumenNivel
	for _, texto := range strings.Split(*niveles, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(texto))
		if err != nil || n < 1 || n > 20000 {
			fmt.Fprintln(os.Stderr, "nivel de usuarios no válido:", texto)
			os.Exit(2)
		}
		r := ejecutarNivel(destino.String(), raices, n, *duracion, *calentamiento, gen, *pausaMin, *pausaMax, *gzip, *pid, *contenedorPG)
		imprimir(r)
		resultados = append(resultados, r)
		time.Sleep(3 * time.Second)
	}
	if *salida != "" {
		datos, _ := json.MarshalIndent(map[string]any{"escenario": *escenario, "niveles": resultados}, "", "  ")
		if err := os.WriteFile(*salida, datos, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, "no se pudo escribir el resumen:", err)
		}
	}
}

func esLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func ejecutarNivel(base string, raices *x509.CertPool, usuarios int, duracion, rampa time.Duration,
	gen func(*rand.Rand) []paso, pausaMin, pausaMax time.Duration, gzip bool, pid int, contenedor string) resumenNivel {
	ctx, cancelar := context.WithTimeout(context.Background(), rampa+duracion)
	defer cancelar()
	inicioMedida := time.Now().Add(rampa)
	muestras := make(chan muestra, 65536)
	var grupo sync.WaitGroup
	var recogidas []muestra
	recogida := make(chan struct{})
	go func() {
		for m := range muestras {
			recogidas = append(recogidas, m)
		}
		close(recogida)
	}()
	monitor := iniciarMonitor(ctx, pid, contenedor)
	for i := 0; i < usuarios; i++ {
		grupo.Add(1)
		retardo := time.Duration(int64(rampa) * int64(i) / int64(usuarios))
		go func(semilla uint64) {
			defer grupo.Done()
			usuarioVirtual(ctx, base, raices, retardo, inicioMedida, gen, pausaMin, pausaMax, gzip, semilla, muestras)
		}(uint64(i + 1))
	}
	grupo.Wait()
	close(muestras)
	<-recogida
	pico := monitor()
	return resumir(usuarios, duracion, recogidas, pico)
}

func usuarioVirtual(ctx context.Context, base string, raices *x509.CertPool, retardo time.Duration, inicioMedida time.Time,
	gen func(*rand.Rand) []paso, pausaMin, pausaMax time.Duration, gzip bool, semilla uint64, muestras chan<- muestra) {
	transporte := &http.Transport{
		TLSClientConfig:     &tls.Config{RootCAs: raices, MinVersion: tls.VersionTLS12},
		ForceAttemptHTTP2:   true,
		MaxIdleConnsPerHost: 2,
		DisableCompression:  true, // se cuentan los bytes tal como viajan
		DialContext:         (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
	}
	defer transporte.CloseIdleConnections()
	cliente := &http.Client{Transport: transporte, Timeout: 30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r := rand.New(rand.NewPCG(semilla, semilla*7919))
	select {
	case <-time.After(retardo):
	case <-ctx.Done():
		return
	}
	for ctx.Err() == nil {
		for _, p := range gen(r) {
			if ctx.Err() != nil {
				return
			}
			m := pedir(ctx, cliente, base, p, gzip)
			if m.err == "cancelado" {
				return
			}
			if time.Now().After(inicioMedida) {
				muestras <- m
			}
		}
		pausa := pausaMin
		if pausaMax > pausaMin {
			pausa += time.Duration(r.Int64N(int64(pausaMax - pausaMin)))
		}
		if pausa > 0 {
			select {
			case <-time.After(pausa):
			case <-ctx.Done():
				return
			}
		}
	}
}

func pedir(ctx context.Context, cliente *http.Client, base string, p paso, gzip bool) muestra {
	peticion, err := http.NewRequestWithContext(ctx, http.MethodGet, base+p.ruta, nil)
	if err != nil {
		return muestra{ruta: p.nombre, err: "peticion"}
	}
	peticion.Header.Set("Accept-Language", "es")
	if gzip {
		peticion.Header.Set("Accept-Encoding", "gzip")
	}
	inicio := time.Now()
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		if ctx.Err() != nil {
			return muestra{err: "cancelado"}
		}
		return muestra{ruta: p.nombre, duracion: time.Since(inicio), err: clasificar(err)}
	}
	n, err := io.Copy(io.Discard, respuesta.Body)
	_ = respuesta.Body.Close()
	m := muestra{ruta: p.nombre, duracion: time.Since(inicio), estado: respuesta.StatusCode, bytes: n}
	if err != nil {
		if ctx.Err() != nil {
			return muestra{err: "cancelado"}
		}
		m.err = clasificar(err)
	}
	return m
}

func clasificar(err error) string {
	var ne net.Error
	switch {
	case errors.As(err, &ne) && ne.Timeout():
		return "tiempo_agotado"
	case strings.Contains(err.Error(), "connection refused"):
		return "conexion_rechazada"
	case strings.Contains(err.Error(), "reset"):
		return "conexion_reiniciada"
	case strings.Contains(err.Error(), "EOF"):
		return "eof"
	default:
		return "red"
	}
}

func resumir(usuarios int, duracion time.Duration, ms []muestra, pico picoRecursos) resumenNivel {
	porRuta := map[string][]muestra{}
	errores := 0
	var bytes int64
	for _, m := range ms {
		porRuta[m.ruta] = append(porRuta[m.ruta], m)
		if m.err != "" || m.estado >= 400 || m.estado == 0 {
			errores++
		}
		bytes += m.bytes
	}
	r := resumenNivel{Usuarios: usuarios, Duracion: duracion.Seconds(), Peticiones: len(ms),
		PorSegundo: float64(len(ms)) / duracion.Seconds(), MBs: float64(bytes) / duracion.Seconds() / 1e6,
		CPUServidor: pico.cpu, RSSServidorMB: pico.rssMB, PGConexiones: pico.pgConexiones,
		PGActivas: pico.pgActivas, PGCPU: pico.pgCPU}
	if len(ms) > 0 {
		r.TasaError = float64(errores) / float64(len(ms))
	}
	nombres := make([]string, 0, len(porRuta))
	for k := range porRuta {
		nombres = append(nombres, k)
	}
	sort.Strings(nombres)
	for _, k := range nombres {
		lista := porRuta[k]
		d := make([]float64, len(lista))
		errs := map[string]int{}
		var b int64
		for i, m := range lista {
			d[i] = float64(m.duracion.Microseconds()) / 1000
			b += m.bytes
			switch {
			case m.err != "":
				errs[m.err]++
			case m.estado >= 400:
				errs["http_"+strconv.Itoa(m.estado)]++
			}
		}
		sort.Float64s(d)
		rr := resumenRuta{Ruta: k, N: len(d), P50: percentil(d, 50), P95: percentil(d, 95),
			P99: percentil(d, 99), Max: d[len(d)-1], BytesMe: float64(b) / float64(len(d))}
		if len(errs) > 0 {
			rr.Errores = errs
		}
		r.Rutas = append(r.Rutas, rr)
	}
	return r
}

func percentil(ordenados []float64, p float64) float64 {
	if len(ordenados) == 0 {
		return 0
	}
	i := int(float64(len(ordenados)-1) * p / 100)
	return ordenados[i]
}

func imprimir(r resumenNivel) {
	fmt.Printf("\n== %d usuarios: %d peticiones (%.0f/s), errores %.2f %%, %.1f MB/s; servidor CPU máx %.0f %% RSS máx %.0f MB; PG conexiones máx %d (activas %d), CPU PG máx %.0f %%\n",
		r.Usuarios, r.Peticiones, r.PorSegundo, r.TasaError*100, r.MBs, r.CPUServidor, r.RSSServidorMB, r.PGConexiones, r.PGActivas, r.PGCPU)
	fmt.Printf("%-14s %8s %9s %9s %9s %9s %10s  %s\n", "ruta", "n", "p50 ms", "p95 ms", "p99 ms", "máx ms", "bytes", "errores")
	for _, x := range r.Rutas {
		fmt.Printf("%-14s %8d %9.1f %9.1f %9.1f %9.1f %10.0f  %v\n", x.Ruta, x.N, x.P50, x.P95, x.P99, x.Max, x.BytesMe, x.Errores)
	}
}

type picoRecursos struct {
	cpu, rssMB, pgCPU       float64
	pgConexiones, pgActivas int
}

// iniciarMonitor muestrea cada segundo la CPU y la memoria del servidor (por
// /proc) y, si hay contenedor, las conexiones de PostgreSQL y su CPU. Devuelve
// una función que detiene el muestreo y entrega los picos.
func iniciarMonitor(ctx context.Context, pid int, contenedor string) func() picoRecursos {
	var pico picoRecursos
	var mu sync.Mutex
	var parar atomic.Bool
	hecho := make(chan struct{})
	go func() {
		defer close(hecho)
		ticks := int64(100)
		cpuAnterior, _ := cpuProceso(pid)
		momentoAnterior := time.Now()
		for !parar.Load() && ctx.Err() == nil {
			time.Sleep(time.Second)
			if pid > 0 {
				cpu, rss := cpuProceso(pid)
				ahora := time.Now()
				pct := float64(cpu-cpuAnterior) / float64(ticks) / ahora.Sub(momentoAnterior).Seconds() * 100
				cpuAnterior, momentoAnterior = cpu, ahora
				mu.Lock()
				pico.cpu = max(pico.cpu, pct)
				pico.rssMB = max(pico.rssMB, rss)
				mu.Unlock()
			}
			if contenedor != "" {
				total, activas := conexionesPG(contenedor)
				mu.Lock()
				pico.pgConexiones = max(pico.pgConexiones, total)
				pico.pgActivas = max(pico.pgActivas, activas)
				mu.Unlock()
			}
		}
	}()
	var cpuPG chan float64
	if contenedor != "" {
		cpuPG = make(chan float64, 1)
		go func() {
			var m float64
			for !parar.Load() && ctx.Err() == nil {
				m = max(m, cpuContenedor(contenedor))
			}
			cpuPG <- m
		}()
	}
	return func() picoRecursos {
		parar.Store(true)
		<-hecho
		if cpuPG != nil {
			v := <-cpuPG
			mu.Lock()
			pico.pgCPU = v
			mu.Unlock()
		}
		mu.Lock()
		defer mu.Unlock()
		return pico
	}
}

func cpuProceso(pid int) (int64, float64) {
	if pid <= 0 {
		return 0, 0
	}
	datos, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0, 0
	}
	texto := string(datos)
	cierre := strings.LastIndexByte(texto, ')')
	if cierre < 0 {
		return 0, 0
	}
	campos := strings.Fields(texto[cierre+2:])
	if len(campos) < 22 {
		return 0, 0
	}
	utime, _ := strconv.ParseInt(campos[11], 10, 64)
	stime, _ := strconv.ParseInt(campos[12], 10, 64)
	rssPaginas, _ := strconv.ParseInt(campos[21], 10, 64)
	return utime + stime, float64(rssPaginas*int64(os.Getpagesize())) / 1e6
}

func conexionesPG(contenedor string) (int, int) {
	salida, err := exec.Command("docker", "exec", contenedor, "psql", "-XAt", "-U", "postgres", "-d", "postgres", "-c",
		"SELECT count(*) FILTER (WHERE backend_type='client backend'), count(*) FILTER (WHERE backend_type='client backend' AND state='active') FROM pg_stat_activity").Output()
	if err != nil {
		return 0, 0
	}
	partes := strings.Split(strings.TrimSpace(string(salida)), "|")
	if len(partes) != 2 {
		return 0, 0
	}
	total, _ := strconv.Atoi(partes[0])
	activas, _ := strconv.Atoi(partes[1])
	// La propia consulta de muestreo cuenta como una conexión activa.
	return total - 1, activas - 1
}

func cpuContenedor(contenedor string) float64 {
	salida, err := exec.Command("docker", "stats", "--no-stream", "--format", "{{.CPUPerc}}", contenedor).Output()
	if err != nil {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(string(salida)), "%"), 64)
	return v
}
