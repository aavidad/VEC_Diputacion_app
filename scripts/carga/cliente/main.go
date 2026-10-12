// Command cliente mide lecturas HTTPS locales. Reutiliza el laboratorio público
// y admite el certificado y las consultas POST del banco interno de RRHH.
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	base := flag.String("url", "https://localhost:18480", "origen HTTPS (solo loopback)")
	ca := flag.String("ca", "", "CA del laboratorio")
	cert := flag.String("cert", "", "certificado sintético de cliente")
	clave := flag.String("key", "", "clave de cliente fuera de Git")
	archivo := flag.String("casos", "", "JSON privado con nombre, ruta, metodo, cuerpo y presupuesto_ms")
	niveles := flag.String("usuarios", "100,500,1000,3000", "usuarios a la vez por tanda")
	duracion := flag.Duration("duracion", 30*time.Second, "duración de cada tanda")
	pausa := flag.Duration("pausa", time.Second, "pausa entre vueltas")
	muestras := flag.Int("muestras", 0, "vueltas por usuario; 0 mide por duración")
	calientes := flag.Int("calentamientos", 0, "vueltas excluidas por usuario")
	flag.Parse()
	if err := validarDestino(*base); err != nil {
		fatal(err)
	}
	if *duracion <= 0 || *pausa < 0 || *muestras < 0 || *calientes < 0 {
		fatal(fmt.Errorf("duración/pausa/muestras/calentamientos no válidos"))
	}
	casos, err := cargarCasos(*archivo, flag.Args())
	if err != nil {
		fatal(err)
	}
	raices := x509.NewCertPool()
	pem, err := os.ReadFile(*ca)
	if err != nil || !raices.AppendCertsFromPEM(pem) {
		fatal(fmt.Errorf("CA no válida"))
	}
	conf := &tls.Config{RootCAs: raices, MinVersion: tls.VersionTLS12}
	if (*cert == "") != (*clave == "") {
		fatal(fmt.Errorf("cert y key deben ir juntos"))
	}
	if *cert != "" {
		par, err := tls.LoadX509KeyPair(*cert, *clave)
		if err != nil {
			fatal(fmt.Errorf("certificado/clave de cliente no válidos"))
		}
		conf.Certificates = []tls.Certificate{par}
	}
	var usuarios []int
	for _, texto := range strings.Split(*niveles, ",") {
		n, err := strconv.Atoi(texto)
		if err != nil || n < 1 {
			fatal(fmt.Errorf("usuarios esperado>=1 obtenido=%s", texto))
		}
		usuarios = append(usuarios, n)
	}
	correcto := true
	for _, n := range usuarios {
		if !tanda(strings.TrimSuffix(*base, "/"), conf, n, *duracion, *pausa, *muestras, *calientes, casos) {
			correcto = false
		}
	}
	if !correcto {
		os.Exit(1)
	}
}

func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }

func nuevoCliente(conf *tls.Config) (*http.Client, *http.Transport) {
	tr := &http.Transport{TLSClientConfig: conf.Clone(), ForceAttemptHTTP2: true, DisableCompression: true,
		DialContext: func(ctx context.Context, red, direccion string) (net.Conn, error) {
			host, puerto, err := net.SplitHostPort(direccion)
			if err != nil {
				return nil, err
			}
			if host == "localhost" {
				host = "127.0.0.1"
			}
			ip := net.ParseIP(host)
			if ip == nil || !ip.IsLoopback() {
				return nil, fmt.Errorf("conexión esperada=loopback obtenido=no local")
			}
			return (&net.Dialer{Timeout: 30 * time.Second}).DialContext(ctx, red, net.JoinHostPort(host, puerto))
		}}
	return &http.Client{Timeout: 30 * time.Second, Transport: tr,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, tr
}

// No conserva cuerpo, referencias ni causas de transporte que puedan contener
// datos personales. El detalle de la causa queda en el registro del servidor.
func medir(cliente *http.Client, base string, caso casoLectura) (float64, string) {
	peticion, err := http.NewRequest(caso.Metodo, base+caso.Ruta, bytes.NewReader(caso.Cuerpo))
	if err != nil {
		return 0, "peticion"
	}
	peticion.Header.Set("Accept", "application/json")
	peticion.Header.Set("Accept-Encoding", "gzip")
	if caso.Metodo == "POST" {
		peticion.Header.Set("Content-Type", "application/json")
	}
	inicio := time.Now()
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		return float64(time.Since(inicio).Microseconds()) / 1000, "red"
	}
	_, lectura := io.Copy(io.Discard, respuesta.Body)
	cierre := respuesta.Body.Close()
	ms := float64(time.Since(inicio).Microseconds()) / 1000
	if respuesta.StatusCode != http.StatusOK {
		return ms, strconv.Itoa(respuesta.StatusCode)
	}
	if lectura != nil || cierre != nil {
		return ms, "cuerpo"
	}
	return ms, ""
}

func percentil(t []float64, q float64) float64 {
	if len(t) == 0 {
		return math.NaN()
	}
	return t[int(math.Ceil(q*float64(len(t))))-1]
}

func tanda(base string, conf *tls.Config, usuarios int, duracion, pausa time.Duration, muestras, calientes int, casos []casoLectura) bool {
	var mu sync.Mutex
	tiempos := map[string][]float64{}
	errores := map[string]map[string]int{}
	inicio := time.Now()
	var grupo sync.WaitGroup
	for i := 0; i < usuarios; i++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			cliente, tr := nuevoCliente(conf)
			defer tr.CloseIdleConnections()
			for j := 0; j < calientes; j++ {
				for _, caso := range casos {
					_, fallo := medir(cliente, base, caso)
					if fallo != "" {
						mu.Lock()
						if errores[caso.Nombre] == nil {
							errores[caso.Nombre] = map[string]int{}
						}
						errores[caso.Nombre]["calentamiento:"+fallo]++
						mu.Unlock()
					}
				}
			}
			fin := time.Now().Add(duracion)
			for vuelta := 0; (muestras > 0 && vuelta < muestras) || (muestras == 0 && time.Now().Before(fin)); vuelta++ {
				for _, caso := range casos {
					ms, fallo := medir(cliente, base, caso)
					mu.Lock()
					if fallo == "" {
						tiempos[caso.Nombre] = append(tiempos[caso.Nombre], ms)
					} else {
						if errores[caso.Nombre] == nil {
							errores[caso.Nombre] = map[string]int{}
						}
						errores[caso.Nombre][fallo]++
					}
					mu.Unlock()
				}
				if pausa > 0 && (muestras == 0 || vuelta+1 < muestras) {
					time.Sleep(pausa)
				}
			}
		}()
	}
	grupo.Wait()
	fmt.Printf("\n== %d usuarios, %.3f s (incluye calentamientos)\n", usuarios, time.Since(inicio).Seconds())
	fmt.Println("| Lectura | n HTTP 200 | p50 ms | p95 ms | p99 ms | Umbral ms | Errores |")
	fmt.Println("| --- | ---: | ---: | ---: | ---: | ---: | --- |")
	correcto := true
	for _, caso := range casos {
		t := tiempos[caso.Nombre]
		sort.Float64s(t)
		fmt.Printf("| %s | %d | %.3f | %.3f | %.3f | %.0f | %v |\n", caso.Nombre, len(t), percentil(t, .5), percentil(t, .95), percentil(t, .99), caso.PresupuestoMS, errores[caso.Nombre])
		if len(t) == 0 || len(errores[caso.Nombre]) > 0 || (caso.PresupuestoMS > 0 && percentil(t, .95) >= caso.PresupuestoMS) {
			correcto = false
		}
	}
	return correcto
}
