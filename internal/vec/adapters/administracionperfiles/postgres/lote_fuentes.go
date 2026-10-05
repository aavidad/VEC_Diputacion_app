package postgres

import (
	"context"
	"errors"
	"regexp"
)

var ErrFuenteAmbitosLoteNoDisponible = errors.New("vec.admin.perfiles.lote.fuente_ambitos.no_disponible")

var referenciaAmbitoLote = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,127}$`)
var referenciaFuenteLote = regexp.MustCompile(`^[a-z][a-z0-9_:-]{2,159}$`)
var huellaFuenteLote = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Los descriptores proceden de recibos privados de la fuente propietaria.
// AUT44 vuelve a cotejarlos con AUT37/Personal31 dentro de la transacción.
type FuenteDescriptorLote struct {
	Referencia   string `json:"referencia"`
	Version      uint64 `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type DimensionFuenteLote struct {
	Dimension string               `json:"dimension"`
	Valores   []string             `json:"valores"`
	Fuente    FuenteDescriptorLote `json:"fuente"`
}

type AmbitosFuenteLote struct {
	OrganizacionRef string
	UnidadRef       string
	Descriptores    []DimensionFuenteLote
}

type ProveedorAmbitosLote interface {
	ResolverUnidadLote(context.Context, string, string) (AmbitosFuenteLote, error)
}

type ConfiguracionProveedorAmbitosLote struct {
	OrganizacionRef string
	Unidades        []AmbitosFuenteLote
}

type proveedorAmbitosLotePrivado struct {
	organizacion string
	unidades     map[string]AmbitosFuenteLote
}

// NuevaFuenteAmbitosLotePrivada recibe material cargado de configuración
// privada por la composición. Ni una cabecera ni un DTO pueden configurarlo.
func NuevaFuenteAmbitosLotePrivada(c ConfiguracionProveedorAmbitosLote) (ProveedorAmbitosLote, error) {
	if !referenciaAmbitoLote.MatchString(c.OrganizacionRef) || len(c.Unidades) == 0 || len(c.Unidades) > 256 {
		return nil, ErrFuenteAmbitosLoteNoDisponible
	}
	r := &proveedorAmbitosLotePrivado{organizacion: c.OrganizacionRef, unidades: make(map[string]AmbitosFuenteLote, len(c.Unidades))}
	for _, unidad := range c.Unidades {
		if !ambitosFuenteLoteValidos(unidad, c.OrganizacionRef, unidad.UnidadRef) ||
			r.unidades[unidad.UnidadRef].UnidadRef != "" {
			return nil, ErrFuenteAmbitosLoteNoDisponible
		}
		r.unidades[unidad.UnidadRef] = clonarAmbitosFuenteLote(unidad)
	}
	return r, nil
}

func ambitosFuenteLoteValidos(a AmbitosFuenteLote, organizacion, unidad string) bool {
	if !referenciaAmbitoLote.MatchString(organizacion) || !referenciaAmbitoLote.MatchString(unidad) ||
		a.OrganizacionRef != organizacion || a.UnidadRef != unidad || len(a.Descriptores) != 2 {
		return false
	}
	for i, d := range a.Descriptores {
		dimension, valor := "organizacion_ref", organizacion
		if i == 1 {
			dimension, valor = "unidad_ref", unidad
		}
		if d.Dimension != dimension || len(d.Valores) != 1 || d.Valores[0] != valor ||
			!referenciaFuenteLote.MatchString(d.Fuente.Referencia) || d.Fuente.Version == 0 ||
			d.Fuente.Version > 2147483647 || !huellaFuenteLote.MatchString(d.Fuente.HuellaSHA256) {
			return false
		}
	}
	return true
}

func (p *proveedorAmbitosLotePrivado) ResolverUnidadLote(ctx context.Context, organizacion, unidad string) (AmbitosFuenteLote, error) {
	if p == nil || ctx == nil || ctx.Err() != nil || organizacion != p.organizacion {
		return AmbitosFuenteLote{}, ErrFuenteAmbitosLoteNoDisponible
	}
	v, ok := p.unidades[unidad]
	if !ok {
		return AmbitosFuenteLote{}, ErrFuenteAmbitosLoteNoDisponible
	}
	return clonarAmbitosFuenteLote(v), nil
}

func clonarAmbitosFuenteLote(a AmbitosFuenteLote) AmbitosFuenteLote {
	b := AmbitosFuenteLote{OrganizacionRef: a.OrganizacionRef, UnidadRef: a.UnidadRef,
		Descriptores: make([]DimensionFuenteLote, len(a.Descriptores))}
	for i, d := range a.Descriptores {
		b.Descriptores[i] = DimensionFuenteLote{Dimension: d.Dimension,
			Valores: append([]string(nil), d.Valores...), Fuente: d.Fuente}
	}
	return b
}
