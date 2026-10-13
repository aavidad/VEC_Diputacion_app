package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/ports"
)

// AppendIntentoAuditoriaEnTx une el asiento nominal a una lectura del llamador.
// No confirma ni reintenta: el propietario confirma juntos lectura y auditoría.
func AppendIntentoAuditoriaEnTx(ctx context.Context, tx pgx.Tx, orden ports.OrdenIntentoAuditoria, proceso, canal string) (ports.AcuseIntentoAuditoria, error) {
	if ctx == nil || ctx.Err() != nil || valorNuloPostgreSQL(tx) {
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	datos, err := orden.Datos()
	if err != nil || datos.Datos.Proceso != proceso || datos.Datos.Canal != canal {
		return ports.AcuseIntentoAuditoria{}, ports.ErrOrdenIntentoAuditoriaInvalida
	}
	acuse, err := appendDatosIntentoAuditoriaEnTx(ctx, tx, datos)
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	if err := acuse.ValidarPara(orden); err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	return acuse, nil
}

func appendDatosIntentoAuditoriaEnTx(ctx context.Context, tx pgx.Tx, datos ports.DatosOrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	contexto, vinculo, material, err := serializarIntentoAuditoria(datos)
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	var acuse ports.AcuseIntentoAuditoria
	if err := tx.QueryRow(ctx, consultaAppendIntentoAuditoria, contexto, vinculo, material).Scan(
		&acuse.AuditoriaRef, &acuse.Secuencia, &acuse.HuellaSHA256,
		&acuse.CorrelacionRef, &acuse.RegistradaEn,
	); err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	acuse.RegistradaEn = acuse.RegistradaEn.UTC()
	return acuse, nil
}
