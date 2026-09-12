package config

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

const (
	EnvAdministracionHTTPAddress      = "VEC_ADMIN_HTTP_ADDR"
	EnvAdministracionHTTPAllowedCIDRs = "VEC_ADMIN_HTTP_ALLOWED_CIDRS"
	EnvAdministracionAllowedOrigins   = "VEC_ADMIN_ALLOWED_ORIGINS"
	EnvAdministracionTLSCertFile      = "VEC_ADMIN_TLS_CERT_FILE"
	EnvAdministracionTLSKeyFile       = "VEC_ADMIN_TLS_KEY_FILE"
	EnvAdministracionTLSClientCAFile  = "VEC_ADMIN_TLS_CLIENT_CA_FILE"
)

var (
	ErrTransporteAdministracionDesactivado = errors.New("config: transporte de administracion desactivado")
	ErrTransporteAdministracionIncompleto  = errors.New("config: transporte de administracion incompleto")
	ErrTransporteAdministracionInvalido    = errors.New("config: transporte de administracion invalido")
)

// ConfiguracionTransporteAdministracion es una frontera independiente del
// transporte RRHH. Sus rutas TLS y redes nunca se derivan del listener común.
type ConfiguracionTransporteAdministracion struct {
	Address          string
	HTTPAllowedCIDRs []string
	AllowedOrigins   []string
	TLSCertFile      string
	TLSKeyFile       string
	TLSClientCAFile  string
}

func (c ConfiguracionTransporteAdministracion) Configurada() bool {
	c = c.normalizar()
	return c.Address != "" || len(c.HTTPAllowedCIDRs) != 0 || len(c.AllowedOrigins) != 0 || c.TLSCertFile != "" || c.TLSKeyFile != "" || c.TLSClientCAFile != ""
}

func (c ConfiguracionTransporteAdministracion) Validar() error {
	c = c.normalizar()
	if !c.Configurada() {
		return ErrTransporteAdministracionDesactivado
	}
	if c.Address == "" || len(c.HTTPAllowedCIDRs) == 0 || len(c.AllowedOrigins) == 0 || c.TLSCertFile == "" || c.TLSKeyFile == "" || c.TLSClientCAFile == "" {
		return ErrTransporteAdministracionIncompleto
	}
	if _, _, err := net.SplitHostPort(c.Address); err != nil {
		return ErrTransporteAdministracionInvalido
	}
	for _, cidr := range c.HTTPAllowedCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return ErrTransporteAdministracionInvalido
		}
	}
	for _, origin := range c.AllowedOrigins {
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return ErrTransporteAdministracionInvalido
		}
	}
	return nil
}

// ProyeccionTransporteAdministracion produce la configuración que puede ver
// el listener ADMIN. No hereda dirección, redes o materiales TLS de RRHH.
func (t ConfiguracionTransporteAdministracion) ConfiguracionServidor(base Config) (Config, error) {
	t = t.normalizar()
	if err := t.Validar(); err != nil {
		return Config{}, err
	}
	general := base.Normalize()
	if listenersEquivalentes(t.Address, general.Address) {
		return Config{}, ErrTransporteAdministracionInvalido
	}
	if general.ExecutionProfile == ExecutionProfileDevelopment && (!direccionLoopback(t.Address) || !redesLoopback(t.HTTPAllowedCIDRs)) {
		return Config{}, ErrTransporteAdministracionInvalido
	}
	proyeccion := general
	proyeccion.Address = t.Address
	proyeccion.HTTPAllowedCIDRs = append([]string(nil), t.HTTPAllowedCIDRs...)
	proyeccion.HTTPAllowedOrigins = append([]string(nil), t.AllowedOrigins...)
	proyeccion.TLSCertFile = t.TLSCertFile
	proyeccion.TLSKeyFile = t.TLSKeyFile
	return proyeccion, nil
}

func (c ConfiguracionTransporteAdministracion) normalizar() ConfiguracionTransporteAdministracion {
	c.Address = strings.TrimSpace(c.Address)
	c.TLSCertFile = strings.TrimSpace(c.TLSCertFile)
	c.TLSKeyFile = strings.TrimSpace(c.TLSKeyFile)
	c.TLSClientCAFile = strings.TrimSpace(c.TLSClientCAFile)
	c.HTTPAllowedCIDRs = normalizeOptionalCIDRs(c.HTTPAllowedCIDRs)
	c.AllowedOrigins = normalizarOrigenesAdministracion(c.AllowedOrigins)
	return c
}

func normalizarOrigenesAdministracion(entradas []string) []string {
	salida := make([]string, 0, len(entradas))
	for _, entrada := range entradas {
		if origen := strings.TrimSpace(entrada); origen != "" {
			salida = append(salida, origen)
		}
	}
	return salida
}

func listenersEquivalentes(a, b string) bool {
	hostA, puertoA, errA := net.SplitHostPort(strings.TrimSpace(a))
	hostB, puertoB, errB := net.SplitHostPort(strings.TrimSpace(b))
	if errA != nil || errB != nil || puertoA != puertoB {
		return false
	}
	ipA, ipB := net.ParseIP(strings.Trim(hostA, "[]")), net.ParseIP(strings.Trim(hostB, "[]"))
	return ipA != nil && ipB != nil && ipA.Equal(ipB)
}

func direccionLoopback(direccion string) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(direccion))
	if err != nil {
		return false
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	return ip != nil && ip.IsLoopback()
}

func redesLoopback(redes []string) bool {
	if len(redes) == 0 {
		return false
	}
	for _, valor := range redes {
		ip, red, err := net.ParseCIDR(valor)
		if err != nil || !ip.IsLoopback() || !red.Contains(ip) {
			return false
		}
		unos, bits := red.Mask.Size()
		if unos < 0 || (bits == net.IPv4len*8 && unos < 8) || (bits == net.IPv6len*8 && unos != net.IPv6len*8) {
			return false
		}
	}
	return true
}
