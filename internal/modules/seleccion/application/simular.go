package application

import (
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func Simular(c domain.Configuracion, entrada domain.Entrada, baremador ports.Baremador) (domain.Resultado, error) {
	if err := c.Validar(); err != nil {
		return domain.Resultado{}, err
	}
	if err := entrada.Validar(); err != nil {
		return domain.Resultado{}, err
	}
	meritos := map[string]domain.MeritosValorados{}
	for _, fase := range c.Fases {
		if fase.Tipo != "meritos" {
			continue
		}
		if baremador == nil {
			return domain.Resultado{}, domain.ErrConfiguracion
		}
		for _, solicitud := range entrada.Solicitudes {
			valoracion, err := baremador.Valorar(entrada.ConvocatoriaRef, entrada.BasesVersion, solicitud.Referencia)
			if err != nil {
				return domain.Resultado{}, err
			}
			meritos[solicitud.Referencia] = valoracion
		}
	}
	return domain.Evaluar(c, entrada, meritos)
}
