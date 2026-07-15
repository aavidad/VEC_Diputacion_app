package httpseguridad

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"
)

var ErrRedNoAutorizada = errors.New("red no autorizada")

// PoliticaRed es inmutable despues de su construccion.
type PoliticaRed struct {
	superficie Superficie
	zona       ZonaRed
	prefijos   []netip.Prefix
}

// NuevaPoliticaRed construye una lista explicita. Una entrada mal formada no se
// ignora: invalida toda la politica para conservar el cierre por defecto.
func NuevaPoliticaRed(configuracion ConfiguracionSuperficie) (PoliticaRed, error) {
	if !configuracion.Superficie.Valida() || !configuracion.ZonaRed.Valida() || len(configuracion.RedesPermitidas) == 0 {
		return PoliticaRed{}, fmt.Errorf("%w: politica incompleta", ErrConfiguracionSuperficie)
	}
	if err := validarCorrespondenciaZona(configuracion.Superficie, configuracion.ZonaRed); err != nil {
		return PoliticaRed{}, err
	}
	prefijos := make([]netip.Prefix, 0, len(configuracion.RedesPermitidas))
	vistos := make(map[netip.Prefix]struct{}, len(configuracion.RedesPermitidas))
	for _, texto := range configuracion.RedesPermitidas {
		prefijo, err := netip.ParsePrefix(strings.TrimSpace(texto))
		if err != nil {
			return PoliticaRed{}, fmt.Errorf("%w: red permitida %q: %v", ErrConfiguracionSuperficie, texto, err)
		}
		prefijo = prefijo.Masked()
		if prefijo.Addr().Is4In6() {
			bitsIPv4 := prefijo.Bits() - 96
			if bitsIPv4 < 0 {
				return PoliticaRed{}, fmt.Errorf("%w: red IPv4 mapeada ambigua %q", ErrConfiguracionSuperficie, texto)
			}
			prefijo = netip.PrefixFrom(prefijo.Addr().Unmap(), bitsIPv4).Masked()
		}
		if configuracion.ZonaRed != ZonaRedPublica && prefijo.Bits() == 0 {
			return PoliticaRed{}, fmt.Errorf("%w: una zona protegida no admite una red universal", ErrConfiguracionSuperficie)
		}
		if _, existe := vistos[prefijo]; existe {
			continue
		}
		vistos[prefijo] = struct{}{}
		prefijos = append(prefijos, prefijo)
	}
	if len(prefijos) == 0 {
		return PoliticaRed{}, fmt.Errorf("%w: politica de red vacia", ErrConfiguracionSuperficie)
	}
	return PoliticaRed{
		superficie: configuracion.Superficie,
		zona:       configuracion.ZonaRed,
		prefijos:   prefijos,
	}, nil
}

// Autorizar acepta exclusivamente la direccion observada en la conexion de
// transporte. La zona pertenece a la politica fijada al arrancar y nunca puede
// ser declarada por una peticion o por una cabecera de proxy.
func (p PoliticaRed) Autorizar(direccionPar netip.Addr) error {
	if !p.superficie.Valida() || !p.zona.Valida() || !direccionPar.IsValid() {
		return ErrRedNoAutorizada
	}
	for _, prefijo := range p.prefijos {
		if prefijo.Contains(direccionPar.Unmap()) || prefijo.Contains(direccionPar) {
			return nil
		}
	}
	return ErrRedNoAutorizada
}
