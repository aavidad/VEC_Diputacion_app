package application

import (
	"context"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func EmitirCheckpointDesarrollo(ctx context.Context, c domain.CheckpointDesarrollo, f ports.FirmadorCheckpointDesarrollo, t ports.SelladorCheckpointDesarrollo, maxRegistros uint64) (domain.ReciboCheckpointDesarrollo, error) {
	if ctx == nil || ctx.Err() != nil || f == nil || t == nil || maxRegistros == 0 || c.Cobertura.Registros > maxRegistros {
		return domain.ReciboCheckpointDesarrollo{}, domain.ErrCheckpointInvalido
	}
	if _, err := c.Canonico(); err != nil {
		return domain.ReciboCheckpointDesarrollo{}, err
	}
	sello, err := t.SellarCheckpoint(ctx, c)
	if err != nil {
		return domain.ReciboCheckpointDesarrollo{}, err
	}
	recibo := domain.ReciboCheckpointDesarrollo{Checkpoint: c, TSA: sello, PinSPKISHA256: f.PinCheckpoint()}
	firmado, err := f.FirmarCheckpoint(ctx, recibo)
	if err != nil {
		return domain.ReciboCheckpointDesarrollo{}, err
	}
	// Un puerto intercambiable no puede cambiar el contenido solicitado.
	a, _ := recibo.CanonicoParaFirma()
	b, e := firmado.CanonicoParaFirma()
	if e != nil || string(a) != string(b) || firmado.FirmaBase64 == "" {
		return domain.ReciboCheckpointDesarrollo{}, domain.ErrCheckpointInvalido
	}
	return firmado, nil
}

type ResultadoCheckpointDesarrollo struct {
	ConsumosHistoricosSinFechaLigada bool   `json:"consumos_historicos_sin_fecha_ligada"`
	FechaConsumoLigadaCotejada       bool   `json:"fecha_consumo_ligada_cotejada"`
	Esquema                          string `json:"esquema"`
	Modo                             string `json:"modo"`
	Firma                            string `json:"firma"`
	IntegridadCadena                 string `json:"integridad_cadena"`
	OrigenExtraccion                 string `json:"origen_extraccion"`
	TSA                              string `json:"tsa"`
	TiempoIndependiente              bool   `json:"tiempo_independiente"`
	FirmaLegal                       bool   `json:"firma_legal"`
}

// resultadoCheckpointRechazado traduce el fallo al informe cerrado del canal,
// sin exponer detalles del proveedor ni el material recibido.
func resultadoCheckpointRechazado() ResultadoCheckpointDesarrollo {
	return ResultadoCheckpointDesarrollo{Esquema: domain.EsquemaCheckpointDesarrollo, Modo: "DESARROLLO", Firma: "rechazada", IntegridadCadena: "no_evaluada", OrigenExtraccion: "no_acreditado", TSA: "no_verificada_offline"}
}

func VerificarCheckpointDesarrollo(ctx context.Context, r domain.ReciboCheckpointDesarrollo, v ports.VerificadorCheckpointDesarrollo, maxRegistros uint64) ResultadoCheckpointDesarrollo {
	o := resultadoCheckpointRechazado()
	if ctx == nil || ctx.Err() != nil || v == nil || maxRegistros == 0 || r.Checkpoint.Cobertura.Registros > maxRegistros {
		return o
	}
	if _, err := r.CanonicoParaFirma(); err != nil {
		return resultadoCheckpointRechazado()
	}
	if err := v.VerificarCheckpoint(ctx, r); err != nil {
		return resultadoCheckpointRechazado()
	}
	o.Firma = "verificada_con_pin_externo"
	return o
}
