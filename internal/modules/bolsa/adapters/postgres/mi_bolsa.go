package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

const funcionConsultarMiBolsaV1 = "vec_bolsa_llamamientos.consultar_mi_bolsa_v1"

var _ puertosbolsa.ConsultaMiBolsa = (*ConsultaMiBolsaPostgreSQL)(nil)

type ConsultaMiBolsaPostgreSQL struct{ pool iniciadorTransacciones }

func NuevaConsultaMiBolsaPostgreSQL(pool *pgxpool.Pool) (*ConsultaMiBolsaPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, puertosbolsa.ErrMaterialMiBolsaNoDisponible
	}
	return &ConsultaMiBolsaPostgreSQL{pool: pool}, nil
}

func (r *ConsultaMiBolsaPostgreSQL) ConsultarMiBolsa(ctx context.Context, s puertosbolsa.SolicitudConsultaMiBolsa) (puertosbolsa.InstantaneaMiBolsa, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) || s.CandidatoRef == "" || s.ConsultadaEn.IsZero() || s.Material.ValidarEstructura() != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, puertosbolsa.ErrConsultaMiBolsaInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errorMiBolsa(ctx, err)
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errorMiBolsa(ctx, err)
	}
	m := s.Material
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT `+funcionConsultarMiBolsaV1+`($1::text,$2::timestamptz,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`,
		s.CandidatoRef, s.ConsultadaEn.UTC(), m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()).Scan(&contenido)
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errorMiBolsa(ctx, err)
	}
	defer borrarBytesPostgreSQL(contenido)
	resultado, err := decodificarInstantaneaMiBolsa(contenido, s.ConsultadaEn)
	if err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, err
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.InstantaneaMiBolsa{}, errorMiBolsa(ctx, err)
	}
	return resultado, nil
}

type respuestaMiBolsaPostgreSQL struct {
	ConsultadaEn    time.Time                        `json:"consultada_en"`
	Participaciones []participacionMiBolsaPostgreSQL `json:"participaciones"`
}
type participacionMiBolsaPostgreSQL struct {
	Bolsa             string                              `json:"bolsa"`
	Categoria         string                              `json:"categoria"`
	Version           uint64                              `json:"version"`
	OrdenInicial      uint64                              `json:"orden_inicial"`
	TotalInstantanea  uint64                              `json:"total_instantanea"`
	EstadoBolsa       string                              `json:"estado_bolsa"`
	VigenteDesde      time.Time                           `json:"vigente_desde"`
	VigenteHasta      *time.Time                          `json:"vigente_hasta"`
	SituacionActual   *situacionActualMiBolsaPostgreSQL   `json:"situacion_actual"`
	UltimoLlamamiento *ultimoLlamamientoMiBolsaPostgreSQL `json:"ultimo_llamamiento"`
}

type ultimoLlamamientoMiBolsaPostgreSQL struct {
	EmitidoEn time.Time `json:"emitido_en"`
	Canal     string    `json:"canal"`
	Resultado string    `json:"resultado"`
}

type situacionActualMiBolsaPostgreSQL struct {
	Estado          string     `json:"estado"`
	Desde           time.Time  `json:"desde"`
	Hasta           *time.Time `json:"hasta"`
	FechaDisponible *time.Time `json:"fecha_disponible"`
}

func decodificarInstantaneaMiBolsa(contenido []byte, esperada time.Time) (puertosbolsa.InstantaneaMiBolsa, error) {
	var respuesta respuestaMiBolsaPostgreSQL
	if json.Unmarshal(contenido, &respuesta) != nil || !respuesta.ConsultadaEn.Equal(esperada.UTC()) {
		return puertosbolsa.InstantaneaMiBolsa{}, puertosbolsa.ErrResultadoMiBolsaInvalido
	}
	resultado := puertosbolsa.InstantaneaMiBolsa{ConsultadaEn: respuesta.ConsultadaEn.UTC(), Participaciones: make([]puertosbolsa.ParticipacionMiBolsa, 0, len(respuesta.Participaciones))}
	for _, origen := range respuesta.Participaciones {
		p := puertosbolsa.ParticipacionMiBolsa{Bolsa: origen.Bolsa, Categoria: origen.Categoria, Version: origen.Version, OrdenInicial: origen.OrdenInicial, TotalInstantanea: origen.TotalInstantanea, EstadoBolsa: origen.EstadoBolsa, VigenteDesde: origen.VigenteDesde.UTC()}
		if origen.VigenteHasta != nil {
			hasta := origen.VigenteHasta.UTC()
			p.VigenteHasta = &hasta
		}
		if origen.SituacionActual != nil {
			s := origen.SituacionActual
			actual := &puertosbolsa.SituacionActualMiBolsa{Estado: s.Estado, Desde: s.Desde.UTC()}
			if s.Hasta != nil {
				hasta := s.Hasta.UTC()
				actual.Hasta = &hasta
			}
			if s.FechaDisponible != nil {
				fecha := s.FechaDisponible.UTC()
				actual.FechaDisponible = &fecha
			}
			p.SituacionActual = actual
		}
		if origen.UltimoLlamamiento != nil {
			l := origen.UltimoLlamamiento
			if l.EmitidoEn.IsZero() || l.EmitidoEn.After(esperada) || l.Canal != "correo" || (l.Resultado != "enviado" && l.Resultado != "no_enviado") {
				return puertosbolsa.InstantaneaMiBolsa{}, puertosbolsa.ErrResultadoMiBolsaInvalido
			}
			p.UltimoLlamamiento = &puertosbolsa.UltimoLlamamientoMiBolsa{EmitidoEn: l.EmitidoEn.UTC(), Canal: l.Canal, Resultado: l.Resultado}
		}
		if p.Bolsa == "" || p.Categoria == "" || p.Version == 0 || p.OrdenInicial == 0 || p.TotalInstantanea < p.OrdenInicial || p.EstadoBolsa == "" || p.VigenteDesde.IsZero() || (p.VigenteHasta != nil && !p.VigenteHasta.After(p.VigenteDesde)) {
			return puertosbolsa.InstantaneaMiBolsa{}, puertosbolsa.ErrResultadoMiBolsaInvalido
		}
		if s := p.SituacionActual; s != nil && (s.Estado == "" || s.Desde.IsZero() || s.Desde.After(esperada) || s.Hasta != nil && s.Hasta.Before(s.Desde)) {
			return puertosbolsa.InstantaneaMiBolsa{}, puertosbolsa.ErrResultadoMiBolsaInvalido
		}
		resultado.Participaciones = append(resultado.Participaciones, p)
	}
	return resultado, nil
}

func errorMiBolsa(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "40001", "40P01", "55P03", "57014":
			return puertosbolsa.ErrMaterialMiBolsaNoDisponible
		case "22000", "22023", "23503", "23514", "42501", "55000":
			return puertosbolsa.ErrConsultaMiBolsaInvalida
		}
	}
	return puertosbolsa.ErrMaterialMiBolsaNoDisponible
}
