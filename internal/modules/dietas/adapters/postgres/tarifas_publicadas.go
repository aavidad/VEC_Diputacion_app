package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

var ErrTarifaPublicadaNoDisponible = errors.New("dietas: tarifa publicada no disponible")

const consultaTarifaPublicada = `SELECT vec_dietas.consultar_tarifa_publicada_v1($1::text,$2::date,$3::smallint,$4::text)`

type iniciadorTarifaPublicada interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type RepositorioTarifasPublicadas struct{ pool iniciadorTarifaPublicada }

func NuevoRepositorioTarifasPublicadas(pool *pgxpool.Pool) (*RepositorioTarifasPublicadas, error) {
	if pool == nil {
		return nil, ErrTarifaPublicadaNoDisponible
	}
	return &RepositorioTarifasPublicadas{pool: pool}, nil
}

// Consultar devuelve el catálogo íntegro publicado; la versión vacía se
// permite únicamente en el primer cálculo. La función SQL exige una sola
// publicación vigente y coherente para los tres grupos.
func (r *RepositorioTarifasPublicadas) Consultar(ctx context.Context, version string, grupo int, vehiculo string, fecha time.Time) (domain.CatalogoLiquidacionPublicado, error) {
	var vacio domain.CatalogoLiquidacionPublicado
	if r == nil || r.pool == nil || ctx == nil || ctx.Err() != nil ||
		(version != "" && !domain.VersionTarifaPublicadaValida(version)) || grupo < 1 || grupo > 3 ||
		(vehiculo != "propio" && vehiculo != "sin_vehiculo_propio") || fecha.IsZero() || fecha.Location() != time.UTC {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	defer func() {
		cierre, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelar()
		_ = tx.Rollback(cierre)
	}()
	if _, err := tx.Exec(ctx, `SELECT set_config('timezone','UTC',true),
		set_config('statement_timeout','5s',true),set_config('idle_in_transaction_session_timeout','10s',true)`); err != nil {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	var crudo []byte
	if err := tx.QueryRow(ctx, consultaTarifaPublicada, version, fecha.Format("2006-01-02"), int16(grupo), vehiculo).Scan(&crudo); err != nil ||
		len(crudo) == 0 || len(crudo) > 256*1024 || ctx.Err() != nil {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(crudo))
	dec.DisallowUnknownFields()
	var catalogo domain.CatalogoLiquidacionPublicado
	var extra any
	if dec.Decode(&catalogo) != nil || dec.Decode(&extra) != io.EOF || catalogo.ValidarForma() != nil ||
		catalogo.Grupo != grupo || catalogo.Vehiculo != vehiculo || catalogo.PaisISO2 != "ES" ||
		(version != "" && catalogo.VersionTarifaRef != version) ||
		fecha.Format("2006-01-02") < catalogo.VigenteDesde || fecha.Format("2006-01-02") > catalogo.VigenteHasta {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	canonico, err := json.Marshal(catalogo)
	if err != nil || !bytes.Equal(canonico, crudo) {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, ErrTarifaPublicadaNoDisponible
	}
	return catalogo, nil
}

var _ interface {
	Consultar(context.Context, string, int, string, time.Time) (domain.CatalogoLiquidacionPublicado, error)
} = (*RepositorioTarifasPublicadas)(nil)
