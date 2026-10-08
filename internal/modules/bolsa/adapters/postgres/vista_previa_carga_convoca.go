package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

// ConsumidorVistaPreviaCargaConvocaPostgreSQL consume la concesión V3 de la
// vista previa sin enviar el fichero ni las filas a PostgreSQL.
type ConsumidorVistaPreviaCargaConvocaPostgreSQL struct {
	pool *pgxpool.Pool
}

var _ ports.ConsumidorVistaPreviaCargaConvoca = (*ConsumidorVistaPreviaCargaConvocaPostgreSQL)(nil)

func NuevoConsumidorVistaPreviaCargaConvocaPostgreSQL(pool *pgxpool.Pool) (*ConsumidorVistaPreviaCargaConvocaPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	return &ConsumidorVistaPreviaCargaConvocaPostgreSQL{pool: pool}, nil
}

func (r *ConsumidorVistaPreviaCargaConvocaPostgreSQL) ConsumirVistaPreviaCargaConvoca(
	ctx context.Context, orden ports.OrdenVistaPreviaCargaConvoca,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (ports.AcuseVistaPreviaCargaConvoca, error) {
	if r == nil || r.pool == nil || ctx == nil || orden.Validar() != nil || material.ValidarEstructura() != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, errorConsumoVistaPreviaCargaConvoca(ctx, err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, errorConsumoVistaPreviaCargaConvoca(ctx, err)
	}
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
		$1::text,$2::text,$3::text,$4::text,$5::bytea,
		$6::bytea,$7::bytea,$8::bytea,$9::bytea,$10::numeric,$11::numeric,
		$12::bytea,$13::bytea,$14::bytea,$15::bytea)`,
		orden.ActaRef, orden.ActorRef, orden.CategoriaRef, orden.HuellaFicheroSHA256,
		orden.ContextoRecursoCanonico, material.CapacidadCanonica(), material.DecisionCanonica(),
		material.MotivoCanonico(), material.ContextoActorCanonico(), material.PersonaVersion(),
		material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI(),
	).Scan(&contenido)
	if err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, errorConsumoVistaPreviaCargaConvoca(ctx, err)
	}
	acuse, err := leerAcuseVistaPreviaCargaConvoca(contenido)
	if err != nil || acuse.ValidarPara(orden, material) != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, errorConsumoVistaPreviaCargaConvoca(ctx, err)
	}
	return acuse, nil
}

func leerAcuseVistaPreviaCargaConvoca(contenido []byte) (ports.AcuseVistaPreviaCargaConvoca, error) {
	const formatoUTC = "2006-01-02T15:04:05.000000Z"
	var leido struct {
		DecisionRef    string `json:"decision_ref"`
		ActaRef        string `json:"acta_ref"`
		HuellaContexto string `json:"huella_contexto"`
		AuditoriaRef   string `json:"auditoria_ref"`
		ConsumidaEn    string `json:"consumida_en"`
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&leido); err != nil {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	instante, err := time.Parse(formatoUTC, leido.ConsumidaEn)
	if err != nil || instante.Format(formatoUTC) != leido.ConsumidaEn {
		return ports.AcuseVistaPreviaCargaConvoca{}, ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	return ports.AcuseVistaPreviaCargaConvoca{
		DecisionRef: leido.DecisionRef, ActaRef: leido.ActaRef,
		HuellaContextoSHA256: leido.HuellaContexto, AuditoriaRef: leido.AuditoriaRef,
		ConsumidaEn: instante.UTC(),
	}, nil
}

// Sólo 42501 de PostgreSQL es una denegación. Los demás fallos cierran la
// operación como indisponible, sin exponer detalles del servidor al cliente.
func errorConsumoVistaPreviaCargaConvoca(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ports.ErrVistaPreviaCargaConvocaNoDisponible
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" {
		return dominiovec.ErrAutorizacionDenegada
	}
	return ports.ErrVistaPreviaCargaConvocaNoDisponible
}
