package postgres

import (
	"bytes"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// RecuperadorNecesidadAltaPostgreSQL consulta únicamente la publicación
// conservada en una alta confirmada y ligada al actor del mismo ámbito HMAC.
type RecuperadorNecesidadAltaPostgreSQL struct{ pool *pgxpool.Pool }

func NuevoRecuperadorNecesidadAltaPostgreSQL(pool *pgxpool.Pool) (*RecuperadorNecesidadAltaPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	var disponible bool
	err := pool.QueryRow(ctx, `SELECT coalesce(pg_catalog.has_function_privilege(
		current_user, pg_catalog.to_regprocedure(
		'vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)'), 'EXECUTE'), false)`).Scan(&disponible)
	if err != nil || !disponible {
		return nil, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	return &RecuperadorNecesidadAltaPostgreSQL{pool: pool}, nil
}

func (r *RecuperadorNecesidadAltaPostgreSQL) RecuperarInstantaneaNecesidadConfirmada(
	ctx context.Context, consulta ports.ConsultaInstantaneaNecesidadConfirmada,
) (ports.ResultadoInstantaneaNecesidadAlta, error) {
	if r == nil || r.pool == nil || ctx == nil || consulta.Validar() != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	pares, err := consulta.AmbitosHMAC.Datos()
	if err != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	ambitos := []string{pares.Activo.Valor}
	for _, retenido := range pares.Retenidos {
		ambitos = append(ambitos, retenido.Valor)
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	resultado := ports.ResultadoInstantaneaNecesidadAlta{Estado: ports.InstantaneaNecesidadAusente}
	for _, ambito := range ambitos {
		var estado string
		var contenido []byte
		err = tx.QueryRow(ctx, `SELECT estado, instantanea FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3($1,$2,$3,$4)`,
			ambito, consulta.OrganizacionRef, consulta.ActorRef, consulta.PerfilRef).Scan(&estado, &contenido)
		if err != nil {
			return ports.ResultadoInstantaneaNecesidadAlta{}, err
		}
		actual := ports.ResultadoInstantaneaNecesidadAlta{Estado: ports.EstadoInstantaneaNecesidadAlta(estado), Instantanea: contenido}
		if actual.Validar() != nil {
			return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
		}
		if actual.Estado == ports.InstantaneaNecesidadAusente {
			continue
		}
		if resultado.Estado == ports.InstantaneaNecesidadAusente {
			resultado = ports.ResultadoInstantaneaNecesidadAlta{Estado: actual.Estado, Instantanea: append([]byte(nil), contenido...)}
			continue
		}
		if resultado.Estado != actual.Estado || !bytes.Equal(resultado.Instantanea, actual.Instantanea) {
			return ports.ResultadoInstantaneaNecesidadAlta{}, ports.ErrFuenteNecesidadesAltaNoDisponible
		}
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ports.ResultadoInstantaneaNecesidadAlta{}, err
	}
	return resultado, nil
}
