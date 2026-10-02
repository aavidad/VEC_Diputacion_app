package ensayofisicopg

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

const modoInicio = "--vec-cs06-interno-iniciar"
const modoSonda = "--vec-cs06-interno-sonda"
const modoDetener = "--vec-cs06-interno-detener"

type inicioArchivado struct {
	Binario    string   `json:"binario"`
	Argumentos []string `json:"argumentos"`
	Entorno    []string `json:"entorno"`
}

// ManejarModoInterno debe invocarse antes de parsear flags por cada CLI que
// componga el ensayo. Se ejecuta dentro del contenedor con su binario propio.
// Fuera del runtime se deniega sin ejecutar comandos ni acceder a configuración.
func ManejarModoInterno(args []string, in io.Reader, out io.Writer) (int, bool) {
	if len(args) != 1 || (args[0] != modoInicio && args[0] != modoSonda && args[0] != modoDetener) {
		return 0, false
	}
	if os.Getenv("VEC_CS06_INTERNO") != "1" || !entornoInterno() {
		return 2, true
	}
	switch args[0] {
	case modoInicio:
		return iniciarDentro(), true
	case modoDetener:
		return detenerDentro(), true
	case modoSonda:
		var s puertos.SondaHTTP
		d := json.NewDecoder(io.LimitReader(in, 65536))
		d.DisallowUnknownFields()
		if d.Decode(&s) != nil {
			return 2, true
		}
		r, err := sondarDentro(context.Background(), s)
		if err != nil {
			slog.Error("cs06_sonda_fallida")
			return 1, true
		}
		if json.NewEncoder(out).Encode(r) != nil {
			return 2, true
		}
		return 0, true
	}
	return 2, true
}
func entornoInterno() bool {
	// Docker montaje read-only del verificador y raíz control propia deben existir.
	for _, p := range []string{"/verificador", "/control", "/data/PG_VERSION"} {
		if _, err := os.Stat(p); err != nil {
			slog.Error("cs06_entorno_interno_no_disponible")
			return false
		}
	}
	return true
}
func iniciarDentro() int {
	b, err := os.ReadFile("/control/inicio.json")
	if err != nil || len(b) > 65536 {
		slog.Error("cs06_inicio_no_disponible")
		return 2
	}
	var s inicioArchivado
	if json.Unmarshal(b, &s) != nil || !strings.HasPrefix(s.Binario, "/componentes/") {
		return 2
	}
	cmd := exec.Command(s.Binario, s.Argumentos...) // #nosec G204 G702 -- ruta de binario archivado previamente verificado, montada sólo en sandbox sin red/privilegios; nunca ejecutable del host.
	cmd.Env = s.Entorno
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if cmd.Start() != nil {
		return 1
	}
	if os.WriteFile("/control/proceso.pid", []byte(strconv.Itoa(cmd.Process.Pid)), 0600) != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return 1
	}
	_ = cmd.Wait()
	_ = os.WriteFile("/control/proceso.terminado", []byte("1"), 0600)
	return 0
}
func detenerDentro() int {
	b, err := os.ReadFile("/control/proceso.pid")
	if err != nil {
		slog.Error("cs06_pid_no_disponible")
		return 1
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil || pid < 2 {
		slog.Error("cs06_pid_no_admitido")
		return 1
	}
	// PID sólo procede de inicio propio y nunca atraviesa namespaces al host.
	if _, err := os.Stat("/control/proceso.terminado"); err == nil {
		return 0
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		slog.Error("cs06_proceso_no_disponible")
		return 1
	}
	if p.Signal(syscall.SIGTERM) != nil {
		return 1
	}
	return 0
}
func rutaTLSAdmitida(s string) bool {
	return strings.HasPrefix(s, "/componentes/") && filepath.Clean(s) == s && !strings.ContainsRune(s, 0)
}
func sondarDentro(ctx context.Context, s puertos.SondaHTTP) (puertos.RespuestaHTTP, error) {
	r := puertos.RespuestaHTTP{}
	if s.Puerto == 0 || !strings.HasPrefix(s.Ruta, "/") || strings.HasPrefix(s.Ruta, "//") || strings.ContainsAny(s.Ruta, "\r\n\x00") || len(s.Ruta) > 4096 || len(s.Authorization) > 16384 || strings.ContainsAny(s.Authorization, "\r\n\x00") || s.LimiteBytes < 1 || s.LimiteBytes > 1<<20 {
		return r, errRuntime
	}
	transporte := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 3 * time.Second}).DialContext, DisableKeepAlives: true}
	esquema := "http"
	if s.TLS {
		esquema = "https"
		if !rutaTLSAdmitida(s.CA) {
			return r, errRuntime
		}
		ca, err := os.ReadFile(s.CA)
		if err != nil || len(ca) > 65536 {
			return r, errRuntime
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(ca) {
			return r, errRuntime
		}
		tc := &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: pool}
		if s.Certificado != "" || s.Clave != "" {
			if !rutaTLSAdmitida(s.Certificado) || !rutaTLSAdmitida(s.Clave) {
				return r, errRuntime
			}
			cert, err := tls.LoadX509KeyPair(s.Certificado, s.Clave)
			if err != nil {
				return r, errRuntime
			}
			tc.Certificates = []tls.Certificate{cert}
		}
		transporte.TLSClientConfig = tc
	} else if s.CA != "" || s.Certificado != "" || s.Clave != "" {
		return r, errRuntime
	}
	cliente := &http.Client{Transport: transporte, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer transporte.CloseIdleConnections()
	peticion, err := http.NewRequestWithContext(ctx, http.MethodGet, esquema+"://127.0.0.1:"+strconv.Itoa(int(s.Puerto))+s.Ruta, nil)
	if err != nil {
		return r, errRuntime
	}
	if peticion.URL.Hostname() != "127.0.0.1" || peticion.URL.User != nil {
		return r, errRuntime
	}
	peticion.Header.Set("Authorization", s.Authorization)
	peticion.Header.Set("Cache-Control", "no-store")
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		return r, errRuntime
	}
	defer respuesta.Body.Close()
	h := sha256.New()
	var contenido bytes.Buffer
	n, err := io.Copy(io.MultiWriter(h, limitadorMemoria{&contenido}), io.LimitReader(respuesta.Body, s.LimiteBytes+1))
	if err != nil || n > s.LimiteBytes || len(respuesta.Cookies()) > 0 {
		return r, errRuntime
	}
	r.EstadoHTTP, r.Bytes, r.SHA256 = respuesta.StatusCode, n, hex.EncodeToString(h.Sum(nil))
	if n <= 16384 {
		r.Contenido = contenido.Bytes()
	}
	return r, nil
}

type limitadorMemoria struct{ b *bytes.Buffer }

func (l limitadorMemoria) Write(p []byte) (int, error) {
	n := len(p)
	if l.b.Len() < 16384 {
		resto := 16384 - l.b.Len()
		if len(p) > resto {
			p = p[:resto]
		}
		_, _ = l.b.Write(p)
	}
	return n, nil
}
