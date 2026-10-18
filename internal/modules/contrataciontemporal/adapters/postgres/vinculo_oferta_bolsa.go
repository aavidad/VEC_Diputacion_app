package postgres

import (
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// decodificarVinculoOfertaCT exige que el recibo SQL conserve el mismo hecho
// publicado por Bolsa. La base verifica además B120 y su propio outbox.
func decodificarVinculoOfertaCT(raw []byte, reserva ports.ReservaSeleccionOfertaCT,
	oferta ports.OfertaPublicadaSeleccionCT,
) (ports.ReciboSeleccionOfertaCT, error) {
	var recibo ports.ReciboSeleccionOfertaCT
	if json.Unmarshal(raw, &recibo) != nil ||
		recibo.ValidarPara(reserva.Intencion, reserva.Comando) != nil ||
		recibo.OfertaRef != oferta.OfertaRef ||
		recibo.ReciboOfertaRef != oferta.ReciboOfertaRef ||
		recibo.NumeroPlazas != oferta.NumeroPlazas ||
		!recibo.PublicadaEn.Equal(oferta.PublicadaEn) ||
		!recibo.VenceAntesDe.Equal(oferta.VenceAntesDe) {
		return ports.ReciboSeleccionOfertaCT{}, errRegistroSeleccionOfertaPostgreSQL
	}
	return recibo, nil
}
