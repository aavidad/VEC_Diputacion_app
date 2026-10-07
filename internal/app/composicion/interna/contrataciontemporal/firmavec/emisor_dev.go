package firmavec

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// NuevaEmisionFirmaVecDesarrollo compone el registro VEC con la selección
// sellada por AUT56 y la captura del mismo emisor V3. El emisor común debe
// recibir su PDP, confianza y material del root; aquí no se crean autoridades.
func NuevaEmisionFirmaVecDesarrollo(a AutoridadSesionFirmanteV2,
	emisor firmaemisorv2.EmisorComunV3ConCaptura, autorizacion vp.FuenteAutorizacion,
	reloj vp.Reloj, motivo core.ReferenciaEntradaCatalogo,
	admision *vecapp.AdmisionGarantiaFirmaVecDesarrollo,
) (*firmaemisorv2.Emisor, *FuenteNominalFirmaVecV2, error) {
	if dependenciaNula(a) || dependenciaNula(emisor) || dependenciaNula(autorizacion) ||
		dependenciaNula(reloj) || admision == nil || !core.ReferenciaMotivoAutorizacionV2Valida(motivo) {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	fuente, err := NuevaFuenteNominalFirmaVecV2(a)
	if err != nil {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	resultado, err := firmaemisorv2.NuevoEmisorFirmaVecDesarrollo(fuente, emisor, motivo, reloj, autorizacion, admision)
	if err != nil {
		return nil, nil, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	return resultado, fuente, nil
}
