package httpcartografia

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"strings"

	"vec-diputacion-granada/config"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
)

var ErrSuperficiePresentacionInvalida = errors.New("http cartografia: superficie de presentacion invalida")

type superficiePresentacion struct {
	rutas           *Manejador
	redesPermitidas []netip.Prefix
}

// NuevaSuperficiePresentacion compone una frontera no autoritativa y sin
// identidad. Solo admite la consulta cartografica exacta y su healthcheck; no
// persiste, no concede permisos y no transporta credenciales ambientales.
func NuevaSuperficiePresentacion(cfg config.Config, calculador dietasapp.CasoUsoCalculoRutas) (http.Handler, error) {
	cfg = cfg.Normalize()
	if !cfg.RRHHPresentationEnabledByDoubleGuard() || cfg.AuthMode != config.AuthModeDisabled ||
		cfg.StorageMode != config.StorageModeMemory || cfg.FakeCredentialsPath != "" ||
		cfg.PersonalCatalogPath != "" || !cfg.PersonalCatalogInMemory || !direccionPrivadaLiteral(cfg.Address) {
		return nil, ErrSuperficiePresentacionInvalida
	}
	redes, err := parsearRedesEntrada(cfg.HTTPAllowedCIDRs)
	if err != nil {
		return nil, errors.Join(ErrSuperficiePresentacionInvalida, err)
	}
	manejador, err := NuevoManejador(calculador, OpcionesManejador{})
	if err != nil {
		return nil, errors.Join(ErrSuperficiePresentacionInvalida, err)
	}
	return &superficiePresentacion{rutas: manejador, redesPermitidas: redes}, nil
}

func (s *superficiePresentacion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	escritor := &escritorSinEstadoAmbiental{ResponseWriter: w}
	prepararCabecerasPresentacion(escritor)
	if r == nil || r.URL == nil {
		escribirError(escritor, http.StatusBadRequest, "peticion no valida")
		return
	}
	if !s.remotoPermitido(r.RemoteAddr) {
		escribirError(escritor, http.StatusForbidden, "acceso no permitido")
		return
	}
	if credencialAmbiental(r.Header) || len(r.Trailer) != 0 || contieneCabecera(r.Header, "Trailer") {
		escribirError(escritor, http.StatusBadRequest, "peticion no valida")
		return
	}
	if r.URL.RawPath != "" || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.URL.EscapedPath() != r.URL.Path || strings.ContainsRune(r.URL.Path, '\\') {
		escribirError(escritor, http.StatusNotFound, "ruta no encontrada")
		return
	}
	switch r.URL.Path {
	case "/healthz":
		s.servirSalud(escritor, r)
	case RutaPresentacion:
		s.rutas.ServeHTTP(escritor, r)
	default:
		escribirError(escritor, http.StatusNotFound, "ruta no encontrada")
	}
}

func (s *superficiePresentacion) servirSalud(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		escribirError(w, http.StatusMethodNotAllowed, "metodo no permitido")
		return
	}
	if r.URL.RawQuery != "" {
		escribirError(w, http.StatusBadRequest, "peticion no valida")
		return
	}
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status": "ok", "mode": "presentacion_cartografica_no_autoritativa",
		})
	}
}

func (s *superficiePresentacion) remotoPermitido(remoto string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoto))
	if err != nil {
		return false
	}
	direccion, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil || direccion.Is4In6() {
		return false
	}
	direccion = direccion.Unmap()
	for _, red := range s.redesPermitidas {
		if red.Contains(direccion) {
			return true
		}
	}
	return false
}

func direccionPrivadaLiteral(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	if err != nil || strings.TrimSpace(host) == "" {
		return false
	}
	ip, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil || ip.IsUnspecified() || ip.Is4In6() {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
}

func parsearRedesEntrada(valores []string) ([]netip.Prefix, error) {
	if len(valores) == 0 || len(valores) > 32 {
		return nil, errors.New("debe enumerarse al menos una red de entrada")
	}
	locales := []netip.Prefix{
		netip.MustParsePrefix("127.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fc00::/7"), netip.MustParsePrefix("fe80::/10"),
	}
	resultado := make([]netip.Prefix, 0, len(valores))
	vistas := make(map[netip.Prefix]struct{}, len(valores))
	for _, valor := range valores {
		if valor == "" || valor != strings.TrimSpace(valor) {
			return nil, errors.New("red de entrada no canonica")
		}
		red, err := netip.ParsePrefix(valor)
		if err != nil || red.Addr().Is4In6() || red.Masked().String() != valor || red.Bits() == 0 {
			return nil, errors.New("red de entrada no valida")
		}
		red = red.Masked()
		contenida := false
		for _, local := range locales {
			if red.Addr().BitLen() == local.Addr().BitLen() && red.Bits() >= local.Bits() && local.Contains(red.Addr()) {
				contenida = true
				break
			}
		}
		if !contenida {
			return nil, errors.New("la red de entrada no es local")
		}
		if _, repetida := vistas[red]; repetida {
			return nil, errors.New("red de entrada repetida")
		}
		vistas[red] = struct{}{}
		resultado = append(resultado, red)
	}
	return resultado, nil
}

func credencialAmbiental(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		normalizado := strings.ToLower(strings.TrimSpace(nombre))
		if normalizado == "cookie" || normalizado == "authorization" ||
			normalizado == "proxy-authorization" || normalizado == "forwarded" ||
			normalizado == "via" || normalizado == "remote-user" || normalizado == "x-remote-user" ||
			strings.HasPrefix(normalizado, "x-vec-") || strings.HasPrefix(normalizado, "x-auth-") ||
			strings.HasPrefix(normalizado, "x-forwarded-") {
			return true
		}
	}
	return false
}

func contieneCabecera(cabeceras http.Header, nombreBuscado string) bool {
	for nombre := range cabeceras {
		if strings.EqualFold(strings.TrimSpace(nombre), nombreBuscado) {
			return true
		}
	}
	return false
}

func prepararCabecerasPresentacion(w http.ResponseWriter) {
	prepararRespuestaJSON(w)
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
}

type escritorSinEstadoAmbiental struct {
	http.ResponseWriter
	escrito bool
}

func (w *escritorSinEstadoAmbiental) WriteHeader(estado int) {
	if w.escrito {
		return
	}
	w.escrito = true
	w.limpiar()
	w.ResponseWriter.WriteHeader(estado)
}

func (w *escritorSinEstadoAmbiental) Write(contenido []byte) (int, error) {
	if !w.escrito {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(contenido)
}

func (w *escritorSinEstadoAmbiental) limpiar() {
	for nombre := range w.Header() {
		normalizado := strings.ToLower(strings.TrimSpace(nombre))
		if normalizado == "set-cookie" || strings.HasPrefix(normalizado, "access-control-") {
			w.Header().Del(nombre)
		}
	}
}
