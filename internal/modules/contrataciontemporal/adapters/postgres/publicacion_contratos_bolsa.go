package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ports.LectorPublicacionContratosBolsa = (*LectorPublicacionContratosBolsaPostgreSQL)(nil)

// LectorPublicacionContratosBolsaPostgreSQL lee CT113 con el rol ejecutor de
// CT. No escribe: la historia y el outbox ya los escribió CT75.
type LectorPublicacionContratosBolsaPostgreSQL struct {
	pool *pgxpool.Pool
}

func NuevoLectorPublicacionContratosBolsaPostgreSQL(pool *pgxpool.Pool) (*LectorPublicacionContratosBolsaPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrPublicacionContratosBolsaNoDisponible
	}
	return &LectorPublicacionContratosBolsaPostgreSQL{pool: pool}, nil
}

func (l *LectorPublicacionContratosBolsaPostgreSQL) LeerContratosBolsa(ctx context.Context, desde ports.CursorPublicacionContratosBolsa, limite int) ([]ports.EventoContratoBolsaPublicado, error) {
	if l == nil || l.pool == nil || ctx == nil || limite < 1 || limite > ports.LimiteLecturaContratosBolsa ||
		(desde.Vacio() && desde.OrigenRef != "") || len(desde.OrigenRef) > 512 {
		return nil, ports.ErrPublicacionContratosBolsaNoDisponible
	}
	var desdeEn *time.Time
	var desdeRef *string
	if !desde.Vacio() {
		en, ref := desde.CreadaEn.UTC(), desde.OrigenRef
		desdeEn, desdeRef = &en, &ref
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrPublicacionContratosBolsaNoDisponible, err)
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT evento_ref, evento::text, huella_sha256, origen_ref, origen_creada_en FROM vec_contratacion_temporal.leer_contratos_bolsa_v1($1,$2,$3)`, desdeEn, desdeRef, limite)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrPublicacionContratosBolsaNoDisponible, err)
	}
	defer filas.Close()
	eventos := make([]ports.EventoContratoBolsaPublicado, 0, limite)
	for filas.Next() {
		var e ports.EventoContratoBolsaPublicado
		var contenido string
		if err := filas.Scan(&e.EventoRef, &contenido, &e.HuellaSHA256, &e.OrigenRef, &e.OrigenCreadaEn); err != nil {
			return nil, fmt.Errorf("%w: %w", ports.ErrPublicacionContratosBolsaNoDisponible, err)
		}
		// La huella la calcula PostgreSQL sobre este mismo texto: se coteja
		// antes de entregarlo para no propagar un contenido alterado.
		suma := sha256.Sum256([]byte(contenido))
		if hex.EncodeToString(suma[:]) != e.HuellaSHA256 || e.EventoRef == "" || e.OrigenRef == "" {
			return nil, fmt.Errorf("%w: %w", ports.ErrPublicacionContratosBolsaNoDisponible, errors.New("huella de evento incoherente"))
		}
		e.Contenido = []byte(contenido)
		eventos = append(eventos, e)
	}
	if err := filas.Err(); err != nil {
		return nil, fmt.Errorf("%w: %w", ports.ErrPublicacionContratosBolsaNoDisponible, err)
	}
	return eventos, nil
}
