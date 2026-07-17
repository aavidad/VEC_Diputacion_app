// La cuenta de ejecucion del panel solo puede invocar la funcion SECURITY
// DEFINER de contrato cerrado. No recibe SELECT sobre tablas de convocatorias,
// bolsas, llamamientos, auditoria ni identidad.
package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const funcionConsultarPanelInternoV1 = "vec_bolsa_panel.consultar_panel_interno_v1"

var _ puertosbolsa.ConsultaPanelInterno = (*ConsultaPanelInternoPostgreSQL)(nil)

// ConsultaPanelInternoPostgreSQL consume una autorizacion V2 y obtiene una
// instantanea agregada dentro de la misma transaccion durable.
type ConsultaPanelInternoPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevaConsultaPanelInternoPostgreSQL(
	pool *pgxpool.Pool,
) (*ConsultaPanelInternoPostgreSQL, error) {
	return nuevaConsultaPanelInternoPostgreSQL(pool)
}

func nuevaConsultaPanelInternoPostgreSQL(
	pool iniciadorTransacciones,
) (*ConsultaPanelInternoPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrFuentePanelInternoPostgreSQLNoDisponible
	}
	return &ConsultaPanelInternoPostgreSQL{pool: pool}, nil
}

func (r *ConsultaPanelInternoPostgreSQL) ConsultarPanel(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConsultaPanelInterno,
) (puertosbolsa.InstantaneaPanelInterno, error) {
	if ctx == nil {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrConsultaPanelInternoInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	if r == nil || valorNulo(r.pool) {
		return puertosbolsa.InstantaneaPanelInterno{}, ErrFuentePanelInternoPostgreSQLNoDisponible
	}
	operacion, prueba, decision, motivo, correlacion, err :=
		serializarConsultaPanelInternoPostgreSQL(solicitud)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	defer borrarBytesPostgreSQL(operacion, prueba, decision, motivo)

	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	defer revertir(tx)

	var respuesta []byte
	err = tx.QueryRow(ctx, `
		SELECT panel_canonico
		FROM `+funcionConsultarPanelInternoV1+`(
			$1::jsonb, $2::jsonb, $3::bytea, $4::bytea, $5::text
		)`, operacion, prueba, decision, motivo, correlacion,
	).Scan(&respuesta)
	defer borrarBytesPostgreSQL(respuesta)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errorPostgreSQLPanelInterno(ctx, err)
	}
	resultado, err := decodificarPanelInternoPostgreSQL(respuesta, solicitud)
	if err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, err
	}
	// La respuesta queda completamente revalidada antes del COMMIT. Una fila
	// ambigua, de demostracion o ligada a otra decision nunca se confirma.
	if _, err := resultado.ClonarValidadaPara(solicitud); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, puertosbolsa.ErrResultadoPanelInternoInvalido
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.InstantaneaPanelInterno{}, errorPostgreSQLPanelInterno(ctx, err)
	}
	return resultado, nil
}

func (r *ConsultaPanelInternoPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return nil, errorPostgreSQLPanelInterno(ctx, err)
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertir(tx)
		return nil, errorPostgreSQLPanelInterno(ctx, err)
	}
	return tx, nil
}
