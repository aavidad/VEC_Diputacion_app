package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var bolsaJustificanteValida = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,256}$`)
var participacionJustificanteValida = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,256}$`)
var huellaJustificanteValida = regexp.MustCompile(`^[0-9a-f]{64}$`)
var recursoJustificanteValido = regexp.MustCompile(`^justificante-renuncia:[0-9a-f]{64}$`)

const configurarConsultaJustificante = `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`

func errorJustificanteRenuncia(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && (pg.Code == "42501" || pg.Code == "22023" || pg.Code == "23503" || pg.Code == "P0002") {
		return errors.Join(bolsaports.ErrJustificanteRenunciaDenegado, err)
	}
	return errors.Join(bolsaports.ErrJustificanteRenunciaNoDisponible, err)
}

func errorAutoridadJustificante(err error) error {
	if err == nil {
		return bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	if errors.Is(err, vecdomain.ErrAutorizacionDenegada) || errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
		return errors.Join(bolsaports.ErrJustificanteRenunciaDenegado, err)
	}
	return errors.Join(bolsaports.ErrJustificanteRenunciaNoDisponible, err)
}

func ejecutarPreparacionJustificante(ctx context.Context, pool *pgxpool.Pool, consulta string, args ...any) ([]byte, error) {
	var contenido []byte
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx, configurarConsultaJustificante); err != nil {
			return err
		}
		var intento []byte
		if err = tx.QueryRow(ctx, consulta, args...).Scan(&intento); err != nil {
			return err
		}
		if len(intento) == 0 || len(intento) > 8192 {
			return bolsaports.ErrJustificanteRenunciaNoDisponible
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		contenido = intento
		return nil
	})
	if err != nil {
		return nil, errorJustificanteRenuncia(ctx, err)
	}
	return contenido, nil
}

func decodificarPreparacionJustificante(contenido []byte, salida any) error {
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(salida); err != nil {
		return errors.Join(bolsaports.ErrJustificanteRenunciaNoDisponible, err)
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		return bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	return nil
}

type ResolutorJustificanteRenunciaPostgreSQL struct {
	pool      *pgxpool.Pool
	bolsaRef  string
	autoridad bolsaports.AutoridadPreparacionJustificanteRenuncia
}

var _ bolsaports.ResolutorJustificanteRenuncia = (*ResolutorJustificanteRenunciaPostgreSQL)(nil)

func NuevoResolutorJustificanteRenunciaConPool(pool *pgxpool.Pool, bolsaRef string,
	autoridad bolsaports.AutoridadPreparacionJustificanteRenuncia) (*ResolutorJustificanteRenunciaPostgreSQL, error) {
	if pool == nil || autoridad == nil || !bolsaJustificanteValida.MatchString(bolsaRef) {
		return nil, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	return &ResolutorJustificanteRenunciaPostgreSQL{pool: pool, bolsaRef: bolsaRef, autoridad: autoridad}, nil
}

func (r *ResolutorJustificanteRenunciaPostgreSQL) ResolverJustificanteRenuncia(ctx context.Context,
	q bolsaports.SolicitudJustificanteRenuncia) (bolsaports.ActoJustificanteRenuncia, error) {
	var vacio bolsaports.ActoJustificanteRenuncia
	if ctx == nil || r == nil || r.pool == nil || r.autoridad == nil {
		return vacio, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if q.BolsaRef != r.bolsaRef {
		return vacio, bolsaports.ErrJustificanteRenunciaDenegado
	}
	candidato, err := mibolsa.CandidatoDelActo(q.ActorRegistradoV2, q.VinculoV2)
	if err != nil {
		return vacio, err
	}
	material, err := r.autoridad.PrepararJustificanteRenuncia(ctx, q.ActorRegistradoV2, q.VinculoV2, r.bolsaRef)
	if err != nil || material.ValidarEstructura() != nil {
		return vacio, errorAutoridadJustificante(err)
	}
	persona := q.ActorRegistradoV2.Contexto.PersonaRef
	contexto := q.ActorRegistradoV2.RegistroContextoRef
	contenido, err := ejecutarPreparacionJustificante(ctx, r.pool, `SELECT vec_documentos.resolver_justificante_renuncia_bolsa_v1($1::text,$2::text,$3::text,$4::text,$5::timestamptz,
	 $6::bytea,$7::bytea,$8::bytea,$9::bytea,$10::numeric,$11::numeric,$12::bytea,$13::bytea,$14::bytea,$15::bytea)`,
		persona, candidato, r.bolsaRef, contexto, q.ActorRegistradoV2.ResueltoEnAutoritativo,
		material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(),
		material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI())
	if err != nil {
		return vacio, err
	}
	var respuesta struct {
		bolsaports.ActoJustificanteRenuncia
		DecisionRef  string `json:"decision_ref"`
		AuditoriaRef string `json:"auditoria_ref"`
	}
	if err = decodificarPreparacionJustificante(contenido, &respuesta); err != nil {
		return vacio, err
	}
	a := respuesta.ActoJustificanteRenuncia
	a.BolsaRef = r.bolsaRef
	if !mibolsa.ActoJustificanteValido(a, persona, candidato, r.bolsaRef) ||
		respuesta.DecisionRef == "" || respuesta.AuditoriaRef == "" {
		return vacio, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	return a, nil
}

type ResolutorConsultaJustificanteRenunciaRRHHPostgreSQL struct {
	pool      *pgxpool.Pool
	autoridad bolsaports.AutoridadPreparacionConsultaJustificanteRenunciaRRHH
}

var _ bolsaports.ResolutorConsultaJustificanteRenunciaRRHH = (*ResolutorConsultaJustificanteRenunciaRRHHPostgreSQL)(nil)

func NuevoResolutorConsultaJustificanteRenunciaRRHHConPool(pool *pgxpool.Pool,
	autoridad bolsaports.AutoridadPreparacionConsultaJustificanteRenunciaRRHH) (*ResolutorConsultaJustificanteRenunciaRRHHPostgreSQL, error) {
	if pool == nil || autoridad == nil {
		return nil, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	return &ResolutorConsultaJustificanteRenunciaRRHHPostgreSQL{pool: pool, autoridad: autoridad}, nil
}

func (r *ResolutorConsultaJustificanteRenunciaRRHHPostgreSQL) ResolverConsultaJustificanteRenunciaRRHH(ctx context.Context,
	q bolsaports.SolicitudConsultaJustificanteRenunciaRRHH) (bolsaports.JustificanteRenunciaRRHH, error) {
	var vacio bolsaports.JustificanteRenunciaRRHH
	if ctx == nil || r == nil || r.pool == nil || r.autoridad == nil {
		return vacio, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if !bolsaJustificanteValida.MatchString(q.BolsaRef) || !participacionJustificanteValida.MatchString(q.ParticipacionRef) ||
		q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil {
		return vacio, bolsaports.ErrJustificanteRenunciaDenegado
	}
	material, err := r.autoridad.PrepararConsultaJustificanteRenunciaRRHH(ctx, q.ActorRegistradoV2,
		q.VinculoV2, q.BolsaRef, q.ParticipacionRef)
	if err != nil || material.ValidarEstructura() != nil {
		return vacio, errorAutoridadJustificante(err)
	}
	contenido, err := ejecutarPreparacionJustificante(ctx, r.pool, `SELECT vec_documentos.resolver_consulta_justificante_renuncia_bolsa_rrhh_v1($1::text,$2::text,$3::text,$4::text,$5::text,
	 $6::bytea,$7::bytea,$8::bytea,$9::bytea,$10::numeric,$11::numeric,$12::bytea,$13::bytea,$14::bytea,$15::bytea)`,
		q.BolsaRef, q.ParticipacionRef, q.ActorRegistradoV2.Contexto.PersonaRef,
		q.ActorRegistradoV2.Contexto.PerfilActivoRef, q.ActorRegistradoV2.RegistroContextoRef,
		material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(),
		material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI())
	if err != nil {
		return vacio, err
	}
	var respuesta struct {
		bolsaports.JustificanteRenunciaRRHH
		DecisionRef  string `json:"decision_ref"`
		AuditoriaRef string `json:"auditoria_ref"`
	}
	if err = decodificarPreparacionJustificante(contenido, &respuesta); err != nil {
		return vacio, err
	}
	resultado := respuesta.JustificanteRenunciaRRHH
	resultado.BolsaRef, resultado.ParticipacionRef = q.BolsaRef, q.ParticipacionRef
	if resultado.UnidadRef == "" || resultado.AmbitoRef == "" ||
		resultado.DocumentoRef == "" || !huellaJustificanteValida.MatchString(resultado.DocumentoSHA256) ||
		!recursoJustificanteValido.MatchString(resultado.RecursoRef) ||
		resultado.PersonaTitularRef == "" || respuesta.DecisionRef == "" || respuesta.AuditoriaRef == "" {
		return vacio, bolsaports.ErrJustificanteRenunciaNoDisponible
	}
	return resultado, nil
}
