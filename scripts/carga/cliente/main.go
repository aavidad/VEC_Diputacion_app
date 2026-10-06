// Command cliente lanza N usuarios contra un servidor local durante un tiempo
// y da p50/p95/p99 y errores por ruta. Cada usuario usa su propia conexión
// TLS, como un navegador distinto. Solo admite destinos de loopback.
//
//	go run ./scripts/carga/cliente -ca ca.crt -usuarios 100,1000 /api/publico/bolsa/bolsas ...
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	base := flag.String("url", "https://localhost:18480", "origen (solo loopback)")
	ca := flag.String("ca", "", "CA del laboratorio")
	niveles := flag.String("usuarios", "100,500,1000,3000", "usuarios a la vez por tanda")
	duracion := flag.Duration("duracion", 30*time.Second, "duración de cada tanda")
	pausa := flag.Duration("pausa", time.Second, "pausa de cada usuario entre vueltas")
	flag.Parse()
	rutas := flag.Args()
	destino, err := url.Parse(*base)
	if err != nil || len(rutas) == 0 || !(destino.Hostname() == "localhost" || net.ParseIP(destino.Hostname()).IsLoopback()) {
		fmt.Fprintln(os.Stderr, "uso: cliente -ca CA [-usuarios 100,1000] RUTA... (destino solo loopback)")
		os.Exit(2)
	}
	raices := x509.NewCertPool()
	if pem, err := os.ReadFile(*ca); err != nil || !raices.AppendCertsFromPEM(pem) {
		fmt.Fprintln(os.Stderr, "CA no válida")
		os.Exit(2)
	}
	for _, texto := range strings.Split(*niveles, ",") {
		n, err := strconv.Atoi(texto)
		if err != nil || n < 1 {
			fmt.Fprintln(os.Stderr, "usuarios no válidos:", texto)
			os.Exit(2)
		}
		tanda(*base, raices, n, *duracion, *pausa, rutas)
	}
}

func tanda(base string, raices *x509.CertPool, usuarios int, duracion, pausa time.Duration, rutas []string) {
	var mu sync.Mutex
	tiempos := map[string][]float64{}
	errores := map[string]map[string]int{}
	fin := time.Now().Add(duracion)
	var grupo sync.WaitGroup
	for i := 0; i < usuarios; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			cliente := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{
				TLSClientConfig: &tls.Config{RootCAs: raices}, ForceAttemptHTTP2: true, DisableCompression: true,
			}}
			for time.Now().Before(fin) {
				for _, ruta := range rutas {
					peticion, _ := http.NewRequest(http.MethodGet, base+ruta, nil)
					peticion.Header.Set("Accept-Encoding", "gzip")
					inicio := time.Now()
					respuesta, err := cliente.Do(peticion)
					fallo := ""
					if err != nil {
						fallo = "red"
					} else {
						_, _ = io.Copy(io.Discard, respuesta.Body)
						_ = respuesta.Body.Close()
						if respuesta.StatusCode >= 400 {
							fallo = strconv.Itoa(respuesta.StatusCode)
						}
					}
					mu.Lock()
					tiempos[ruta] = append(tiempos[ruta], float64(time.Since(inicio).Microseconds())/1000)
					if fallo != "" {
						if errores[ruta] == nil {
							errores[ruta] = map[string]int{}
						}
						errores[ruta][fallo]++
					}
					mu.Unlock()
				}
				time.Sleep(pausa)
			}
		}()
	}
	grupo.Wait()
	total := 0
	for _, t := range tiempos {
		total += len(t)
	}
	fmt.Printf("\n== %d usuarios: %.0f peticiones/s\n", usuarios, float64(total)/duracion.Seconds())
	for _, ruta := range rutas {
		t := tiempos[ruta]
		sort.Float64s(t)
		p := func(q float64) float64 { return t[int(float64(len(t)-1)*q)] }
		fmt.Printf("%-60s n=%-7d p50=%-7.1f p95=%-7.1f p99=%-7.1f ms errores=%v\n", ruta, len(t), p(.5), p(.95), p(.99), errores[ruta])
	}
}
