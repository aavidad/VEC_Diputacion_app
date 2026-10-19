package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

var _ ports.RepositorioConfirmacionAceptacionDirecta = (*RepositorioSituacionParticipacionPostgreSQL)(nil)
var errConfirmacionConcurrente = errors.New("bolsa: confirmacion concurrente")

func (r *RepositorioSituacionParticipacionPostgreSQL) ConfirmarAceptacionDirecta(ctx context.Context,
	c ports.ComandoConfirmarAceptacionDirecta,
) (ports.RegistroConfirmacionAceptacionDirecta, error) {
	for intento := 0; intento < 3; intento++ {
		v, err := r.confirmarAceptacionUna(ctx, c)
		if !errors.Is(err, errConfirmacionConcurrente) {
			return v, err
		}
	}
	return ports.RegistroConfirmacionAceptacionDirecta{}, ports.ErrConfirmacionAceptacionDirectaConflicto
}

func (r *RepositorioSituacionParticipacionPostgreSQL) confirmarAceptacionUna(ctx context.Context,
	c ports.ComandoConfirmarAceptacionDirecta,
) (ports.RegistroConfirmacionAceptacionDirecta, error) {
	vacio := ports.RegistroConfirmacionAceptacionDirecta{}
	if r == nil || r.pool == nil || ctx == nil || c.Canon == "" || len(c.ContextoRecursoCanonico) == 0 ||
		c.Material.ValidarEstructura() != nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, errorConfirmacionDirecta(err)
	}
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
	 set_config('row_security','on',true),set_config('timezone','UTC',true),
	 set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
	 set_config('idle_in_transaction_session_timeout','20s',true)`)
	if err != nil {
		return vacio, errorConfirmacionDirecta(err)
	}
	m := c.Material
	var datos []byte
	var reutilizada bool
	err = tx.QueryRow(ctx, `SELECT recibo,reutilizada FROM vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(
	 $1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9,$10,$11,$12)`, c.Canon, c.ContextoRecursoCanonico,
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		m.PersonaVersion(), m.PerfilVersion(), m.PayloadVECAD3(), m.SobreCOSESign1(),
		m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&datos, &reutilizada)
	if err != nil {
		return vacio, errorConfirmacionDirecta(err)
	}
	v, err := decodificarConfirmacionDirecta(datos, reutilizada)
	if err != nil || v.ValidarPara(c) != nil {
		return vacio, ports.ErrSituacionParticipacionNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorConfirmacionDirecta(err)
	}
	return v, nil
}

func decodificarConfirmacionDirecta(datos []byte, reutilizada bool) (ports.RegistroConfirmacionAceptacionDirecta, error) {
	var crudo struct {
		BolsaRef               string     `json:"bolsa_ref"`
		ParticipacionRef       string     `json:"participacion_ref"`
		LlamamientoRef         string     `json:"llamamiento_ref"`
		Motivo                 string     `json:"motivo"`
		ReciboRef              string     `json:"recibo_ref"`
		Situacion              string     `json:"situacion"`
		RespuestaRef           string     `json:"respuesta_ref"`
		ResolucionRef          string     `json:"resolucion_ref"`
		CanonSHA256            string     `json:"canon_sha256"`
		Desde                  time.Time  `json:"desde"`
		SituacionEsperadaDesde time.Time  `json:"situacion_desde_esperada"`
		ResueltaEn             time.Time  `json:"resuelta_en"`
		FechaDisponible        *time.Time `json:"fecha_disponible"`
	}
	if json.Unmarshal(datos, &crudo) != nil {
		return ports.RegistroConfirmacionAceptacionDirecta{}, ports.ErrSituacionParticipacionNoDisponible
	}
	return ports.RegistroConfirmacionAceptacionDirecta{RegistroSituacionParticipacion: ports.RegistroSituacionParticipacion{
		Reutilizada: reutilizada, ReciboRef: crudo.ReciboRef, Motivo: crudo.Motivo,
		SituacionParticipacion: ports.SituacionParticipacion{ParticipacionRef: crudo.ParticipacionRef,
			Situacion: crudo.Situacion, Desde: crudo.Desde, FechaDisponible: crudo.FechaDisponible}, ResueltaEn: &crudo.ResueltaEn},
		BolsaRef: crudo.BolsaRef, LlamamientoRef: crudo.LlamamientoRef, RespuestaRef: crudo.RespuestaRef,
		ResolucionRef: crudo.ResolucionRef, SituacionEsperadaDesde: crudo.SituacionEsperadaDesde, CanonSHA256: crudo.CanonSHA256}, nil
}

func errorConfirmacionDirecta(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42501":
			return core.ErrAutorizacionDenegada
		case "VBS01", "VBS02", "23505":
			return ports.ErrConfirmacionAceptacionDirectaConflicto
		case "40001", "40P01":
			return errConfirmacionConcurrente
		}
	}
	return ports.ErrSituacionParticipacionNoDisponible
}
