package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	firma "vec-diputacion-granada/internal/modules/contrataciontemporal/application/firmaautorizacionv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const registrarFirmaConPlanSQL176 = `SELECT vec_contratacion_temporal.registrar_firma_con_plan_v2($1,$2::timestamptz,$3,$4,$5,$6,$7,$8::numeric,$9::numeric,$10,$11,$12,$13,$14,$15,$16,$17,$18::numeric,$19::numeric,$20,$21,$22,$23)::text`

var _ ports.RegistradorFirmaConPlanV2 = (*RegistroFirmasVerificadasPostgreSQL)(nil)

// Una llamada CT176 consume las dos exportaciones y el efecto en una TX.
func (r *RegistroFirmasVerificadasPostgreSQL) RegistrarFirmaConPlanV2(ctx context.Context,
	m ports.MaterialFirmaVerificadaV2, c ports.CapacidadFirmaConPlanV2) (ports.ReciboFirmaDocumento, error) {
	var cero ports.ReciboFirmaDocumento
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	canon, err := m.Canonico()
	if err != nil || !domain.InstanteUTCCanonico(m.ComprobadaEn) ||
		m.PuestoFirmanteRef != "" || m.AmbitoFirmanteRef != "" || m.ActoCompetenciaRef != "" {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	if firma.ValidarCapacidadFirmaConPlanV2(c, m) != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	interior, exterior, _, envoltorio, _ := c.ExportarParaConsumidor()
	defer clear(envoltorio)
	pInterior, err := exportacionParametrosRegistroV2(interior.ExportarMaterialParaConsumidor(), true)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer limpiarParametrosFirma172(pInterior)
	pExterior, err := exportacionParametrosRegistroV2(exterior, true)
	if err != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	defer limpiarParametrosFirma172(pExterior)
	args := append([]any{string(canon), m.ComprobadaEn, envoltorio}, pInterior...)
	args = append(args, pExterior...)
	huella, err := m.HuellaSHA256()
	if err != nil {
		return cero, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nuloRegistroTX(tx) {
		return cero, errorFirma118(ctx, err)
	}
	confirmado := false
	defer func() {
		if !confirmado {
			c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancelar()
			if err := tx.Rollback(c); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				slog.Warn("contratacion temporal: rollback de firma con plan V2 no confirmado")
			}
		}
	}()
	if _, err = tx.Exec(ctx, ajustesRegistroIncorporacionTXV2); err != nil {
		return cero, errorFirma118(ctx, err)
	}
	var contenido []byte
	if err = tx.QueryRow(ctx, registrarFirmaConPlanSQL176, args...).Scan(&contenido); err != nil {
		return cero, errorFirmaConPlanV2(ctx, err)
	}
	defer clear(contenido)
	var w reciboFirmaSQLV2
	if err := decodificarFirma118(contenido, &w); err != nil {
		return cero, errors.Join(ports.ErrResultadoFirmaDocumentoInvalido, err)
	}
	recibo, err := proyectarReciboFirmaV2(w, m, huella)
	if err != nil {
		return cero, err
	}
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	if err = tx.Commit(ctx); err != nil {
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	confirmado = true
	if err = ctx.Err(); err != nil {
		return cero, err
	}
	return recibo, nil
}

func errorFirmaConPlanV2(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "P1701" {
		return ports.ErrFirmaDocumentoEnConflicto
	}
	return errorFirma118(ctx, err)
}
