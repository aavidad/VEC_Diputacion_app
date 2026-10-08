package rptpublica

import (
	"context"
	"crypto/subtle"
	"reflect"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

// VerificadorAlta consume la lectura pública que conserva Personal. Comprueba
// sólo código y publicación; no conoce ocupantes, vacantes ni titularidad.
type VerificadorAlta struct {
	fuente personalports.ConsultaRPTPublica
}

func NuevoVerificadorAlta(fuente personalports.ConsultaRPTPublica) (*VerificadorAlta, error) {
	if fuente == nil || (reflect.ValueOf(fuente).Kind() == reflect.Pointer && reflect.ValueOf(fuente).IsNil()) {
		return nil, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	return &VerificadorAlta{fuente: fuente}, nil
}

func (v *VerificadorAlta) VerificarPuestoRPTAlta(
	ctx context.Context, solicitud ports.SolicitudVerificarPuestoRPTAlta,
) (ports.PuestoRPTAltaVerificado, error) {
	if v == nil || v.fuente == nil || ctx == nil || solicitud.Validar() != nil {
		return ports.PuestoRPTAltaVerificado{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	catalogo, err := v.fuente.ObtenerRPTPublica(ctx)
	if err != nil || catalogo.Validar() != nil || ctx.Err() != nil ||
		catalogo.Esquema != "vec.catalogo.rpt.v1" ||
		catalogo.Fuente.Importacion != solicitud.CatalogoRef || solicitud.CatalogoVersion != 1 ||
		subtle.ConstantTimeCompare([]byte(catalogo.Fuente.HuellaSHA256), []byte(solicitud.CatalogoHuellaSHA256)) != 1 {
		return ports.PuestoRPTAltaVerificado{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	resultado := ports.PuestoRPTAltaVerificado{
		CatalogoRef: solicitud.CatalogoRef, CatalogoVersion: solicitud.CatalogoVersion,
		CatalogoHuellaSHA256: solicitud.CatalogoHuellaSHA256, PuestoCodigo: solicitud.PuestoCodigo,
	}
	for _, puesto := range catalogo.Puestos {
		if puesto.Codigo == solicitud.PuestoCodigo {
			resultado.ExisteEnPublicacion = true
			break
		}
	}
	return resultado, nil
}
