package postgresimportacionconvoca

import (
	"context"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain/importacionconvoca"
)

// PrepararCargaAutorizada reutiliza exactamente la protección y serialización
// del importador CLI. El llamante debe entregar ambos JSON a una sola operación
// transaccional y borrar sus bytes tras usarla; esta función no escribe.
func PrepararCargaAutorizada(ctx context.Context, protector ProtectorStagingConvoca, lote dominio.LoteValidado) ([]byte, []byte, error) {
	if ctx == nil || valorNulo(protector) || lote.Validar() != nil {
		return nil, nil, ErrLoteNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	protegido, err := protector.ProtegerStaging(ctx, SolicitudProteccionStaging{
		ImportacionRef:      lote.Acta.ImportacionRef,
		HuellaFicheroSHA256: lote.Acta.HuellaFicheroSHA256,
		Esquema:             lote.Acta.Esquema,
		Filas:               clonarFilasDominio(lote.Aceptadas),
	})
	if err != nil {
		borrarFilasProtegidas(protegido.Filas)
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		return nil, nil, ErrProteccionNoDisponible
	}
	defer borrarFilasProtegidas(protegido.Filas)
	if validarCorrespondenciaProteccion(lote.Aceptadas, protegido.Filas) != nil {
		return nil, nil, ErrMaterialNoConfiable
	}
	actaJSON, err := serializarActa(lote.Acta)
	if err != nil {
		return nil, nil, err
	}
	filasJSON, err := serializarFilasProtegidas(protegido.Filas)
	if err != nil {
		borrarBytes(actaJSON)
		return nil, nil, err
	}
	return actaJSON, filasJSON, nil
}
