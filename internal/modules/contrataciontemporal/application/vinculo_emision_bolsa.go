package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var ErrServicioVinculoEmisionBolsaNoDisponible = errors.New("ct: servicio de vínculo de emisión Bolsa no disponible")

type ServicioVinculoEmisionBolsa struct {
	autorizador ports.AutorizadorVinculoEmisionBolsa
	repositorio ports.RepositorioVinculoEmisionBolsa
}

func NuevoServicioVinculoEmisionBolsa(a ports.AutorizadorVinculoEmisionBolsa, r ports.RepositorioVinculoEmisionBolsa) (*ServicioVinculoEmisionBolsa, error) {
	if dependenciaNula(a) || dependenciaNula(r) {
		return nil, ErrServicioVinculoEmisionBolsaNoDisponible
	}
	return &ServicioVinculoEmisionBolsa{autorizador: a, repositorio: r}, nil
}

func (s *ServicioVinculoEmisionBolsa) Vincular(ctx context.Context, solicitud ports.SolicitudVinculoEmisionBolsa) (ports.ReciboVinculoEmisionBolsa, error) {
	var vacio ports.ReciboVinculoEmisionBolsa
	if ctx == nil || s == nil || solicitud.Validar() != nil {
		return vacio, ports.ErrVinculoEmisionBolsaInvalido
	}
	_, material, materialSHA256, err := ports.NuevoMaterialVinculoEmisionBolsa(solicitud)
	if err != nil {
		return vacio, err
	}
	defer clear(material)
	autorizacion, err := s.autorizador.AutorizarVinculoEmisionBolsa(ctx, solicitud, materialSHA256)
	if err != nil {
		return vacio, err
	}
	recibo, err := s.repositorio.RegistrarVinculoEmisionBolsa(ctx, material, autorizacion)
	if err != nil {
		return vacio, err
	}
	if recibo.ValidarPara(solicitud) != nil {
		return vacio, ports.ErrVinculoEmisionBolsaNoDisponible
	}
	return recibo, nil
}
