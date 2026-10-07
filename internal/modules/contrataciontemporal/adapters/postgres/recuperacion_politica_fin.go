package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionPoliticaFinAnalisisConfirmado = "vec_contratacion_temporal.consultar_politica_fin_analisis_confirmado_v1"
	funcionPoliticaFinAltaConfirmada     = "vec_contratacion_temporal.consultar_politica_fin_alta_confirmada_v1"
	maximoBytesPoliticaFinConfirmada     = 2048
)

type RecuperadorPoliticaFinPostgreSQL struct {
	pool *pgxpool.Pool
}

var _ ports.RecuperadorPoliticaFinConfirmada = (*RecuperadorPoliticaFinPostgreSQL)(nil)

func NuevoRecuperadorPoliticaFinPostgreSQL(pool *pgxpool.Pool) (*RecuperadorPoliticaFinPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrPersistenciaNoDisponible
	}
	return &RecuperadorPoliticaFinPostgreSQL{pool: pool}, nil
}

func (r *RecuperadorPoliticaFinPostgreSQL) ConsultarPoliticaFinAnalisisConfirmada(
	ctx context.Context, consulta ports.ConsultaPoliticaFinAnalisisConfirmada,
) (domain.PoliticaFin, bool, error) {
	if r == nil || r.pool == nil || ctx == nil || consulta.Validar() != nil {
		return domain.PoliticaFin{}, false, ports.ErrPreparacionOperacionAnalisisInvalida
	}
	ambitos, err := ambitosPoliticaFinConfirmada(consulta.AmbitosHMAC)
	if err != nil {
		return domain.PoliticaFin{}, false, ports.ErrPreparacionOperacionAnalisisInvalida
	}
	return r.consultar(ctx, `SELECT politica_fin FROM `+funcionPoliticaFinAnalisisConfirmado+`(
		$1::jsonb,$2::text,$3::text,$4::text,$5::text,$6::text,$7::numeric)`,
		ambitos, consulta.OrganizacionRef, consulta.ExpedienteRef,
		consulta.ActorRef, consulta.PerfilRef, string(consulta.Operacion),
		strconv.FormatUint(consulta.VersionExpediente, 10))
}

func (r *RecuperadorPoliticaFinPostgreSQL) ConsultarPoliticaFinAltaConfirmada(
	ctx context.Context, consulta ports.ConsultaPoliticaFinAltaConfirmada,
) (domain.PoliticaFin, bool, error) {
	if r == nil || r.pool == nil || ctx == nil || consulta.Validar() != nil {
		return domain.PoliticaFin{}, false, ports.ErrPreparacionAltaInvalida
	}
	ambitos, err := ambitosPoliticaFinConfirmada(consulta.AmbitosHMAC)
	if err != nil {
		return domain.PoliticaFin{}, false, ports.ErrPreparacionAltaInvalida
	}
	// El expediente todavía no se conoce: la función lo obtiene de la
	// reserva confirmada ligada al ámbito HMAC, actor, perfil y organización.
	return r.consultar(ctx, `SELECT politica_fin FROM `+funcionPoliticaFinAltaConfirmada+`(
		$1::jsonb,$2::text,$3::text,$4::text,$5::text)`,
		ambitos, consulta.OrganizacionRef, "", consulta.ActorRef, consulta.PerfilRef)
}

func ambitosPoliticaFinConfirmada(coleccion ports.ColeccionSellosHMAC) ([]byte, error) {
	datos, err := coleccion.Datos()
	if err != nil {
		return nil, err
	}
	valores := make([]string, 0, len(datos.Retenidos)+1)
	valores = append(valores, datos.Activo.Valor)
	for _, retenido := range datos.Retenidos {
		valores = append(valores, retenido.Valor)
	}
	return json.Marshal(valores)
}

func (r *RecuperadorPoliticaFinPostgreSQL) consultar(
	ctx context.Context, consulta string, argumentos ...any,
) (domain.PoliticaFin, bool, error) {
	if err := ctx.Err(); err != nil {
		return domain.PoliticaFin{}, false, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return domain.PoliticaFin{}, false, errorRecuperacionPoliticaFin(ctx)
	}
	defer revertirTransaccion(tx)
	_, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true), set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`)
	if err != nil {
		return domain.PoliticaFin{}, false, errorRecuperacionPoliticaFin(ctx)
	}
	filas, err := tx.Query(ctx, consulta, argumentos...)
	if err != nil {
		return domain.PoliticaFin{}, false, errorRecuperacionPoliticaFin(ctx)
	}
	var contenido []byte
	confirmada := filas.Next()
	if confirmada {
		err = filas.Scan(&contenido)
		if err == nil && filas.Next() {
			err = ports.ErrPersistenciaNoDisponible
		}
	}
	if err == nil {
		err = filas.Err()
	}
	filas.Close()
	if err != nil {
		return domain.PoliticaFin{}, false, errorRecuperacionPoliticaFin(ctx)
	}
	politica, err := decodificarPoliticaFinConfirmada(contenido, confirmada)
	if err != nil {
		return domain.PoliticaFin{}, false, errorRecuperacionPoliticaFin(ctx)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PoliticaFin{}, false, errorRecuperacionPoliticaFin(ctx)
	}
	return politica, confirmada, nil
}

func decodificarPoliticaFinConfirmada(contenido []byte, confirmada bool) (domain.PoliticaFin, error) {
	if !confirmada || len(contenido) == 0 {
		return domain.PoliticaFin{}, nil
	}
	if len(contenido) > maximoBytesPoliticaFinConfirmada || bytes.Equal(contenido, []byte("null")) {
		return domain.PoliticaFin{}, ports.ErrPersistenciaNoDisponible
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	var politica domain.PoliticaFin
	if err := decodificador.Decode(&politica); err != nil {
		return domain.PoliticaFin{}, ports.ErrPersistenciaNoDisponible
	}
	var extra any
	if err := decodificador.Decode(&extra); !errors.Is(err, io.EOF) || politica.Validar() != nil {
		return domain.PoliticaFin{}, ports.ErrPersistenciaNoDisponible
	}
	return politica, nil
}

func errorRecuperacionPoliticaFin(ctx context.Context) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	return ports.ErrPersistenciaNoDisponible
}
