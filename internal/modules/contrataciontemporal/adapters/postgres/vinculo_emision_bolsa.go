package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type RepositorioVinculoEmisionBolsaPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioVinculoEmisionBolsa = (*RepositorioVinculoEmisionBolsaPostgreSQL)(nil)
var _ ports.LectorAmbitosVinculoEmisionBolsa = (*RepositorioVinculoEmisionBolsaPostgreSQL)(nil)

func NuevoRepositorioVinculoEmisionBolsaPostgreSQL(pool *pgxpool.Pool) (*RepositorioVinculoEmisionBolsaPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrVinculoEmisionBolsaNoDisponible
	}
	return &RepositorioVinculoEmisionBolsaPostgreSQL{pool: pool}, nil
}

func (r *RepositorioVinculoEmisionBolsaPostgreSQL) LeerAmbitosVinculoEmisionBolsa(
	ctx context.Context, s ports.SolicitudVinculoEmisionBolsa,
) (string, string, error) {
	if ctx == nil || r == nil || r.pool == nil || s.Validar() != nil {
		return "", "", ports.ErrVinculoEmisionBolsaInvalido
	}
	var centro, categoria string
	err := r.pool.QueryRow(ctx, `SELECT centro_ref,categoria_ref FROM
		vec_contratacion_temporal.leer_ambitos_vinculo_emision_bolsa_ct_v1($1::text,$2::text,$3::numeric,$4::text)`,
		s.OrganizacionRef, s.ExpedienteRef, s.VersionEsperada, s.BolsaRef).Scan(&centro, &categoria)
	if err != nil {
		return "", "", normalizarErrorVinculoBolsa(ctx, err)
	}
	if centro == "" || categoria == "" {
		return "", "", ports.ErrVinculoEmisionBolsaNoDisponible
	}
	return centro, categoria, nil
}

func (r *RepositorioVinculoEmisionBolsaPostgreSQL) RegistrarVinculoEmisionBolsa(
	ctx context.Context, material []byte, m vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.ReciboVinculoEmisionBolsa, error) {
	var vacio ports.ReciboVinculoEmisionBolsa
	if ctx == nil || r == nil || r.pool == nil || len(material) < 128 || len(material) > 4096 || !json.Valid(material) || m.ValidarEstructura() != nil {
		return vacio, ports.ErrVinculoEmisionBolsaInvalido
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, normalizarErrorVinculoBolsa(ctx, err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
		set_config('row_security','on',true),set_config('timezone','UTC',true),
		set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
		set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacio, normalizarErrorVinculoBolsa(ctx, err)
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_vinculo_emision_bolsa_ct_v1(
		$1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,
		$8::bytea,$9::bytea,$10::bytea,$11::bytea)`, string(material),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&raw)
	if err != nil {
		return vacio, normalizarErrorVinculoBolsa(ctx, err)
	}
	var recibo ports.ReciboVinculoEmisionBolsa
	if len(raw) == 0 || len(raw) > 4096 || json.Unmarshal(raw, &recibo) != nil {
		return vacio, ports.ErrVinculoEmisionBolsaNoDisponible
	}
	recibo.VinculadoEn = recibo.VinculadoEn.UTC()
	if err = tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorVinculoBolsa(ctx, err)
	}
	return recibo, nil
}

func normalizarErrorVinculoBolsa(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ports.ErrVinculoEmisionBolsaConflicto
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "23505", "40001":
			return ports.ErrVinculoEmisionBolsaConflicto
		case "42501":
			return ports.ErrAutorizacionDenegada
		}
	}
	return ports.ErrVinculoEmisionBolsaNoDisponible
}
