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
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	funcionSolicitarPortalV1       = "vec_bolsa_llamamientos.solicitar_portal_candidato_v1"
	funcionResponderPortalV1       = "vec_bolsa_llamamientos.responder_llamamiento_portal_v1"
	funcionPrepararRespuestaV1     = "vec_bolsa_llamamientos.preparar_respuesta_portal_v1"
	funcionLeerPortalCandidatoV1   = "vec_bolsa_llamamientos.leer_portal_candidato_v1"
	configuracionTransaccionPortal = `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`
)

var _ puertosbolsa.RegistroPortalCandidato = (*RegistroPortalCandidatoPostgreSQL)(nil)

// RegistroPortalCandidatoPostgreSQL escribe las acciones propias del
// candidato (Bolsa 000030). Cada llamada consume su decisión AD3-84 en la
// misma transacción que la fila y su auditoría.
type RegistroPortalCandidatoPostgreSQL struct{ pool iniciadorTransacciones }

func NuevoRegistroPortalCandidatoPostgreSQL(pool *pgxpool.Pool) (*RegistroPortalCandidatoPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	return &RegistroPortalCandidatoPostgreSQL{pool: pool}, nil
}

func (r *RegistroPortalCandidatoPostgreSQL) SolicitarPortal(ctx context.Context, s puertosbolsa.SolicitudPortalCandidato) (puertosbolsa.ReciboSolicitudPortal, error) {
	var vacio puertosbolsa.ReciboSolicitudPortal
	if ctx == nil || r == nil || valorNulo(r.pool) || s.CandidatoRef == "" || s.Bolsa == "" || s.RegistradaEn.IsZero() || s.Material.ValidarEstructura() != nil {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertir(tx)
	m := s.Material
	var recibo puertosbolsa.ReciboSolicitudPortal
	err = tx.QueryRow(ctx, `SELECT reutilizada, solicitud_ref, recibo_ref, registrada_en FROM `+funcionSolicitarPortalV1+`($1::text,$2::text,$3::text,$4::text,$5::text,$6::timestamptz,$7::timestamptz,$8::text[],$9::text,$10::text,$11::timestamptz,$12::bytea,$13::bytea,$14::bytea,$15::bytea,$16::numeric,$17::numeric,$18::bytea,$19::bytea,$20::bytea,$21::bytea)`,
		s.SolicitudRef, s.ReciboRef, s.CandidatoRef, s.Bolsa, s.Tipo, utcOpcional(s.PausaHasta), utcOpcional(s.PausaMaxima), s.SituacionesAdmitidas, s.ReglaRef, s.Clave, s.RegistradaEn.UTC(),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&recibo.Reutilizada, &recibo.SolicitudRef, &recibo.ReciboRef, &recibo.RegistradaEn)
	if err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	recibo.RegistradaEn = recibo.RegistradaEn.UTC()
	return recibo, nil
}

func (r *RegistroPortalCandidatoPostgreSQL) ResponderPortal(ctx context.Context, s puertosbolsa.RespuestaPortalCandidato, plazo puertosbolsa.PlazoRespuestaPortal) (puertosbolsa.ReciboRespuestaPortal, error) {
	var vacio puertosbolsa.ReciboRespuestaPortal
	if ctx == nil || r == nil || valorNulo(r.pool) || valorNulo(plazo) || s.CandidatoRef == "" || s.Bolsa == "" || s.RespondidaEn.IsZero() || len(s.ResultadosEfectivos) == 0 || s.Material.ValidarEstructura() != nil {
		return vacio, puertosbolsa.ErrPortalCandidatoInvalido
	}
	tx, err := r.abrir(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertir(tx)
	m := s.Material
	// Primero se consume la decisión propia: sin ella la base no deja leer el
	// portal. La respuesta usa después esa misma decisión, no otra.
	if _, err := tx.Exec(ctx, `SELECT `+funcionPrepararRespuestaV1+`($1::text,$2::text,$3::bytea,$4::bytea,$5::bytea,$6::bytea,$7::numeric,$8::numeric,$9::bytea,$10::bytea,$11::bytea,$12::bytea)`,
		s.CandidatoRef, s.Bolsa, m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()); err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	// El contacto vigente se lee en la misma transacción serializable que la
	// escritura; SQL vuelve a exigir que sea ese y que la hora sea anterior.
	estados, err := leerPortalCandidato(ctx, tx, s.CandidatoRef, s.RespondidaEn, s.ResultadosEfectivos)
	if err != nil {
		return vacio, err
	}
	// Sin llamamiento abierto se llama igualmente: una repetición de la misma
	// clave devuelve su recibo y cualquier otra cosa es VBP04 en SQL.
	var contacto, vence *time.Time
	for _, e := range estados {
		if e.Bolsa == s.Bolsa && e.LlamamientoAbierto != nil {
			inicio := e.LlamamientoAbierto.ContactoEn.UTC()
			fin, err := plazo.VencimientoRespuesta(ctx, inicio)
			if err != nil || !fin.After(inicio) {
				return vacio, errors.Join(puertosbolsa.ErrPortalCandidatoNoDisponible, err)
			}
			if !s.RespondidaEn.Before(fin) {
				return vacio, puertosbolsa.ErrPortalRespuestaFueraDePlazo
			}
			fin = fin.UTC()
			contacto, vence = &inicio, &fin
		}
	}
	var recibo puertosbolsa.ReciboRespuestaPortal
	if contacto != nil {
		recibo.ContactoEn, recibo.VenceAntesDe = *contacto, *vence
	}
	err = tx.QueryRow(ctx, `SELECT reutilizada, respuesta_ref, recibo_ref, respondida_en, modo FROM `+funcionResponderPortalV1+`($1::text,$2::text,$3::text,$4::text,$5::text,$6::text,$7::text,$8::text,$9::text,$10::timestamptz,$11::timestamptz,$12::text[],$13::text,$14::text,$15::timestamptz,$16::bytea,$17::bytea,$18::bytea,$19::bytea,$20::numeric,$21::numeric,$22::bytea,$23::bytea,$24::bytea,$25::bytea)`,
		s.RespuestaRef, s.ReciboRef, s.CandidatoRef, s.Bolsa, s.Respuesta, textoOpcional(s.Causa), textoOpcional(s.JustificanteRef), textoOpcional(s.JustificanteSHA256), s.Modo,
		contacto, vence, s.ResultadosEfectivos, s.ReglaRef, s.Clave, s.RespondidaEn.UTC(),
		m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(), int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI(),
	).Scan(&recibo.Reutilizada, &recibo.RespuestaRef, &recibo.ReciboRef, &recibo.RespondidaEn, &recibo.Modo)
	if err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorPortalCandidato(ctx, err)
	}
	recibo.RespondidaEn = recibo.RespondidaEn.UTC()
	return recibo, nil
}

func (r *RegistroPortalCandidatoPostgreSQL) abrir(ctx context.Context) (pgx.Tx, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorPortalCandidato(ctx, err)
	}
	if _, err = tx.Exec(ctx, configuracionTransaccionPortal); err != nil {
		revertir(tx)
		return nil, errorPortalCandidato(ctx, err)
	}
	return tx, nil
}

type consultorPortal interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type estadoPortalPostgreSQL struct {
	Bolsa              string `json:"bolsa"`
	LlamamientoAbierto *struct {
		ContactoEn time.Time `json:"contacto_en"`
	} `json:"llamamiento_abierto"`
	SolicitudPendiente *struct {
		Tipo         string     `json:"tipo"`
		Recibo       string     `json:"recibo"`
		RegistradaEn time.Time  `json:"registrada_en"`
		PausaHasta   *time.Time `json:"pausa_hasta"`
	} `json:"solicitud_pendiente"`
	UltimaRespuesta *struct {
		Respuesta    string    `json:"respuesta"`
		Modo         string    `json:"modo"`
		Recibo       string    `json:"recibo"`
		RespondidaEn time.Time `json:"respondida_en"`
	} `json:"ultima_respuesta"`
}

// leerPortalCandidato solo se usa dentro de una transacción que consume una
// decisión propia del candidato (consulta o acción).
func leerPortalCandidato(ctx context.Context, tx consultorPortal, candidato string, corte time.Time, efectivos []string) ([]puertosbolsa.EstadoPortalCandidato, error) {
	var contenido []byte
	if err := tx.QueryRow(ctx, `SELECT `+funcionLeerPortalCandidatoV1+`($1::text,$2::timestamptz,$3::text[])`, candidato, corte.UTC(), efectivos).Scan(&contenido); err != nil {
		return nil, errorPortalCandidato(ctx, err)
	}
	var filas []estadoPortalPostgreSQL
	if json.Unmarshal(contenido, &filas) != nil {
		return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
	}
	estados := make([]puertosbolsa.EstadoPortalCandidato, 0, len(filas))
	for _, f := range filas {
		if f.Bolsa == "" {
			return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
		}
		e := puertosbolsa.EstadoPortalCandidato{Bolsa: f.Bolsa}
		if a := f.LlamamientoAbierto; a != nil {
			if a.ContactoEn.IsZero() || a.ContactoEn.After(corte) {
				return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
			}
			e.LlamamientoAbierto = &puertosbolsa.LlamamientoAbiertoPortal{ContactoEn: a.ContactoEn.UTC()}
		}
		if p := f.SolicitudPendiente; p != nil {
			if p.Tipo != puertosbolsa.SolicitudPortalPausa && p.Tipo != puertosbolsa.SolicitudPortalReactivacion || p.Recibo == "" || p.RegistradaEn.IsZero() {
				return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
			}
			e.SolicitudPendiente = &puertosbolsa.SolicitudPendientePortal{Tipo: p.Tipo, Recibo: p.Recibo, RegistradaEn: p.RegistradaEn.UTC(), PausaHasta: utcOpcional(p.PausaHasta)}
		}
		if u := f.UltimaRespuesta; u != nil {
			if u.Respuesta == "" || u.Recibo == "" || u.RespondidaEn.IsZero() || (u.Modo != puertosbolsa.ModoRespuestaPortalFirme && u.Modo != puertosbolsa.ModoRespuestaPortalPropuesta) {
				return nil, puertosbolsa.ErrPortalCandidatoNoDisponible
			}
			e.UltimaRespuesta = &puertosbolsa.UltimaRespuestaPortal{Respuesta: u.Respuesta, Modo: u.Modo, Recibo: u.Recibo, RespondidaEn: u.RespondidaEn.UTC()}
		}
		estados = append(estados, e)
	}
	return estados, nil
}

func utcOpcional(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.UTC()
	return &v
}

func textoOpcional(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func errorPortalCandidato(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "VBP01":
			return puertosbolsa.ErrPortalClaveReutilizada
		case "VBP02":
			return puertosbolsa.ErrPortalSolicitudPendiente
		case "VBP03":
			return puertosbolsa.ErrPortalSituacionNoAdmite
		case "VBP04", "23505":
			return puertosbolsa.ErrPortalSinLlamamientoAbierto
		case "VBP05":
			return puertosbolsa.ErrPortalRespuestaFueraDePlazo
		case "42501":
			return errors.Join(dominiovec.ErrAutorizacionDenegada, puertosbolsa.ErrPortalCandidatoInvalido)
		case "22000", "22023", "23503", "23514":
			return puertosbolsa.ErrPortalCandidatoInvalido
		}
	}
	return puertosbolsa.ErrPortalCandidatoNoDisponible
}
