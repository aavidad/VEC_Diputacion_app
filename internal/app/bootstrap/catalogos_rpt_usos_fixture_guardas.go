package bootstrap

import (
	"net"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
)

const audienciaUsosCategoriasRPTFixture = "vec_catalogos_configurables.usos_categorias.v1"

var referenciaRPTUsosFixture = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{7,127}$`)
var huellaRPTUsosFixture = regexp.MustCompile(`^[0-9a-f]{64}$`)

// admisionRPTUsosFixture conserva el destino aprobado del clon. Las referencias
// de READY sólo habilitan la comprobación de dependencias: no acreditan por sí
// solas instalación SQL, identidad, gobierno de claves ni una concesión.
type admisionRPTUsosFixture struct {
	activado                                       bool
	host                                           string
	puerto                                         uint16
	base                                           string
	huellaClonEsperada, huellaClonReady            string
	readyH6, readyAD132, readySeisRPT              string
	aprobacionRef, preimagenAsignacionHuellaSHA256 string
}

// validar comprueba el destino de todos los pools antes de abrir conexiones.
// El puerto procede de la configuración privada aprobada; ningún puerto de
// consultas MCP se interpreta como identidad del clon de recorridos.
func (a admisionRPTUsosFixture) validar(cfg config.Config, pools [3]*pgxpool.Config) error {
	ip := net.ParseIP(a.host)
	if !a.activado || !cfg.DevelopmentEnabledByDoubleKey() || ip == nil || !ip.IsLoopback() ||
		a.puerto == 0 || a.base == "" ||
		!huellaRPTUsosFixture.MatchString(a.huellaClonEsperada) || a.huellaClonReady != a.huellaClonEsperada ||
		!referenciaRPTUsosFixture.MatchString(a.readyH6) || !referenciaRPTUsosFixture.MatchString(a.readyAD132) ||
		!referenciaRPTUsosFixture.MatchString(a.readySeisRPT) ||
		!referenciaRPTUsosFixture.MatchString(a.aprobacionRef) ||
		!huellaRPTUsosFixture.MatchString(a.preimagenAsignacionHuellaSHA256) {
		return ErrSeguridadComunDesarrolloDenegada
	}
	usuarios := make(map[string]bool, len(pools))
	for _, pool := range pools {
		if pool == nil || pool.ConnConfig == nil {
			return ErrSeguridadComunDesarrolloDenegada
		}
		c := pool.ConnConfig
		if !a.destinoExacto(c.Host, c.Port) || c.Database != a.base || c.User == "" || usuarios[c.User] ||
			c.RuntimeParams["role"] != "" || c.RuntimeParams["session_authorization"] != "" ||
			c.RuntimeParams["options"] != "" || validarTLSPostgreSQLBorradores(&c.Config, true) != nil {
			return ErrSeguridadComunDesarrolloDenegada
		}
		usuarios[c.User] = true
		for _, alternativa := range c.Fallbacks {
			if alternativa == nil || !a.destinoExacto(alternativa.Host, alternativa.Port) {
				return ErrSeguridadComunDesarrolloDenegada
			}
		}
	}
	return nil
}

func (a admisionRPTUsosFixture) destinoExacto(host string, puerto uint16) bool {
	observada, esperada := net.ParseIP(host), net.ParseIP(a.host)
	return observada != nil && esperada != nil && observada.IsLoopback() &&
		observada.Equal(esperada) && puerto == a.puerto
}
