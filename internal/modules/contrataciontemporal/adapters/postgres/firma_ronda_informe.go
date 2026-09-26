package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// consultarInicioRondaInformeSQL123 lee de CT123 la versión en la que se
// emitió el informe nuevo vigente del expediente (0 si no lo hay).
const consultarInicioRondaInformeSQL123 = `SELECT vec_contratacion_temporal.inicio_ronda_informe_nuevo_v1($1, $2)::bigint`

var _ ports.FuenteRondaFirmaInforme = (*RegistroFirmasDocumentoPostgreSQL)(nil)

// InicioRondaInformeNuevo devuelve la versión del expediente desde la que se
// firma el informe nuevo tras subsanar. Solo lee del propio módulo.
func (r *RegistroFirmasDocumentoPostgreSQL) InicioRondaInformeNuevo(ctx context.Context, organizacionRef, expedienteRef string) (uint64, error) {
	if ctx == nil || r == nil || nuloRegistroTX(r.pool) {
		return 0, ports.ErrRegistroFirmaDocumentoNoDisponible
	}
	if !domain.ReferenciaOpacaValida(organizacionRef) || !domain.ReferenciaOpacaValida(expedienteRef) {
		return 0, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil || nuloRegistroTX(tx) {
		return 0, errorFirma118(ctx, err)
	}
	defer func() {
		c, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(c)
	}()
	var version int64
	if err = tx.QueryRow(ctx, consultarInicioRondaInformeSQL123, organizacionRef, expedienteRef).Scan(&version); err != nil {
		return 0, errorFirma118(ctx, err)
	}
	if version < 0 {
		return 0, ports.ErrResultadoFirmaDocumentoInvalido
	}
	return uint64(version), nil
}
