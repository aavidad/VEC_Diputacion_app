package application

import (
	"context"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type ResultadoContinuidadCheckpoint struct {
	ResultadoCheckpointDesarrollo
	domain.InformeContinuidadCheckpoint
}

func VerificarContinuidadCheckpoint(ctx context.Context, ancla domain.ReciboCheckpointDesarrollo, recibos []domain.ReciboCheckpointDesarrollo,
	v ports.VerificadorCheckpointDesarrollo, maxRegistros uint64, maxRecibos int,
) ResultadoContinuidadCheckpoint {
	r := ResultadoContinuidadCheckpoint{ResultadoCheckpointDesarrollo: resultadoCheckpointRechazado(),
		InformeContinuidadCheckpoint: domain.RechazoContinuidadCheckpoint("firma", "verificada_con_pin_externo", "rechazada")}
	r.Esquema = domain.EsquemaContinuidadCheckpointDesarrollo
	if maxRecibos < 1 || maxRecibos > 256 || len(recibos) < 1 || len(recibos) > maxRecibos || maxRegistros == 0 {
		r.InformeContinuidadCheckpoint = domain.RechazoContinuidadCheckpoint("limites", "validos", "invalidos")
		return r
	}
	if VerificarCheckpointDesarrollo(ctx, ancla, v, maxRegistros).Firma != "verificada_con_pin_externo" {
		return r
	}
	for _, recibo := range recibos {
		if VerificarCheckpointDesarrollo(ctx, recibo, v, maxRegistros).Firma != "verificada_con_pin_externo" {
			return r
		}
	}
	r.Firma = "verificada_con_pin_externo"
	r.InformeContinuidadCheckpoint = domain.CotejarContinuidadCheckpoint(ancla, recibos, maxRegistros, maxRecibos)
	return r
}
