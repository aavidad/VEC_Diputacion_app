package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// RepositorioConstitucionPostgreSQL persiste la constitución de bolsas a través
// de las funciones de la migración 000007 del esquema vec_bolsa_llamamientos
// (ejecutor). Los canónicos se serializan con el mismo JSON que firma el
// dominio (HuellaCanonicaSHA256), de modo que la huella almacenada y la del
// dominio coinciden.
type RepositorioConstitucionPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.RepositorioConstitucion = (*RepositorioConstitucionPostgreSQL)(nil)

func NuevoRepositorioConstitucionPostgreSQL(pool *pgxpool.Pool) (*RepositorioConstitucionPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	return &RepositorioConstitucionPostgreSQL{pool: pool}, nil
}

type entradaConstitucionJSON struct {
	Orden            uint64 `json:"orden"`
	ParticipacionRef string `json:"participacion_ref"`
	FilaNumero       int    `json:"fila_numero"`
}

type reciboConstitucionJSON struct {
	Reutilizada        bool   `json:"reutilizada"`
	ActaRef            string `json:"acta_ref"`
	BolsaRef           string `json:"bolsa_ref"`
	VersionBolsa       uint64 `json:"version_bolsa"`
	InstantaneaRef     string `json:"instantanea_ref"`
	VersionInstantanea uint64 `json:"version_instantanea"`
	ConfirmadaEn       string `json:"confirmada_en"`
}

func (r *RepositorioConstitucionPostgreSQL) Constituir(ctx context.Context, c ports.Constitucion) (ports.ReciboConstitucion, error) {
	if ctx == nil || r == nil || r.pool == nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboConstitucion{}, err
	}
	bolsa, err := c.Bolsa.ClonarCanonica()
	if err != nil || c.Instantanea.Validar() != nil || len(c.Entradas) == 0 ||
		len(c.Entradas) != len(c.Instantanea.Entradas) || c.ActaRef == "" || c.ActorRef == "" || c.CategoriaRef == "" {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	bolsaCanonica, err := json.Marshal(bolsa)
	if err != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	instantaneaCanonica, err := json.Marshal(c.Instantanea)
	if err != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	entradas := make([]entradaConstitucionJSON, len(c.Entradas))
	for i, e := range c.Entradas {
		entradas[i] = entradaConstitucionJSON{Orden: e.Orden, ParticipacionRef: e.ParticipacionRef, FilaNumero: e.FilaNumero}
	}
	entradasJSON, err := json.Marshal(entradas)
	if err != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaInvalida
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaNoDisponible
	}
	defer tx.Rollback(context.Background())
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.constituir_bolsa_v1(
		$1::text, $2::text, $3::text, $4::text, $5::bigint, $6::bytea, $7::timestamptz,
		$8::text, $9::bigint, $10::bytea, $11::timestamptz, $12::timestamptz, $13::jsonb, $14::timestamptz)`,
		c.ActaRef, c.ActorRef, c.CategoriaRef, bolsa.BolsaRef, int64(bolsa.Version), bolsaCanonica, bolsa.VigenteDesde,
		c.Instantanea.InstantaneaRef, int64(c.Instantanea.Version), instantaneaCanonica,
		c.Instantanea.ReferidaEn, c.Instantanea.GeneradaEn, entradasJSON, c.ConfirmadaEn,
	).Scan(&contenido)
	if err != nil {
		return ports.ReciboConstitucion{}, errorConstitucion(ctx, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaNoDisponible
	}
	var recibo reciboConstitucionJSON
	if json.Unmarshal(contenido, &recibo) != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaNoDisponible
	}
	confirmada, err := time.Parse("2006-01-02T15:04:05.000000Z", recibo.ConfirmadaEn)
	if err != nil {
		return ports.ReciboConstitucion{}, ports.ErrConstitucionBolsaNoDisponible
	}
	return ports.ReciboConstitucion{
		Reutilizada: recibo.Reutilizada, ActaRef: recibo.ActaRef, BolsaRef: recibo.BolsaRef,
		VersionBolsa: recibo.VersionBolsa, InstantaneaRef: recibo.InstantaneaRef,
		VersionInstantanea: recibo.VersionInstantanea, ConfirmadaEn: confirmada,
	}, nil
}

func (r *RepositorioConstitucionPostgreSQL) ListarVigentes(ctx context.Context) ([]ports.ConstitucionVigente, error) {
	if ctx == nil || r == nil || r.pool == nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := tx.Query(ctx, `SELECT acta_ref, categoria_ref, estado, instantanea_canonica, total_participaciones, confirmada_en
		FROM vec_bolsa_llamamientos.listar_constituciones_v1()`)
	if err != nil {
		return nil, errorConstitucion(ctx, err)
	}
	defer filas.Close()
	var resultado []ports.ConstitucionVigente
	for filas.Next() {
		var v ports.ConstitucionVigente
		var canon []byte
		var total int64
		if err := filas.Scan(&v.ActaRef, &v.CategoriaRef, &v.Estado, &canon, &total, &v.ConfirmadaEn); err != nil {
			return nil, ports.ErrConstitucionBolsaNoDisponible
		}
		var instantanea dominio.InstantaneaOrdenBolsa
		if json.Unmarshal(canon, &instantanea) != nil || instantanea.Validar() != nil || total < 0 {
			return nil, ports.ErrConstitucionBolsaNoDisponible
		}
		v.Instantanea = instantanea
		v.TotalParticipaciones = uint64(total)
		v.Bolsa = dominio.BolsaConstituida{BolsaRef: instantanea.BolsaRef, Version: instantanea.VersionBolsa,
			CategoriaRef: v.CategoriaRef, ListadoDefinitivoRef: instantanea.ListadoDefinitivoRef,
			VersionListado: instantanea.VersionListado, HuellaListadoSHA256: instantanea.HuellaListadoSHA256}
		resultado = append(resultado, v)
	}
	if filas.Err() != nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	return resultado, nil
}

func (r *RepositorioConstitucionPostgreSQL) Entradas(ctx context.Context, instantaneaRef string, version uint64) ([]ports.EntradaConstitucion, error) {
	if ctx == nil || r == nil || r.pool == nil || instantaneaRef == "" || version == 0 {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	filas, err := r.pool.Query(ctx, `SELECT orden, participacion_ref, fila_numero FROM vec_bolsa_llamamientos.listar_entradas_constitucion_v1($1::text, $2::bigint)`, instantaneaRef, int64(version))
	if err != nil {
		return nil, errorConstitucion(ctx, err)
	}
	defer filas.Close()
	var resultado []ports.EntradaConstitucion
	for filas.Next() {
		var e ports.EntradaConstitucion
		var orden int64
		if err := filas.Scan(&orden, &e.ParticipacionRef, &e.FilaNumero); err != nil || orden <= 0 {
			return nil, ports.ErrConstitucionBolsaNoDisponible
		}
		e.Orden = uint64(orden)
		resultado = append(resultado, e)
	}
	if filas.Err() != nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	return resultado, nil
}

func errorConstitucion(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) {
		switch errorPG.Code {
		case "22023", "23514", "23502", "23503":
			return ports.ErrConstitucionBolsaInvalida
		case "23505":
			return ports.ErrConstitucionBolsaEnConflicto
		}
	}
	return ports.ErrConstitucionBolsaNoDisponible
}

type vinculoCandidatoJSON struct {
	CandidatoRef     string `json:"candidato_ref"`
	ParticipacionRef string `json:"participacion_ref"`
}

// RegistrarVinculos registra los vínculos `can_* → participación` de un acta
// constituida (migración 000008). Idempotente: los ya registrados con el mismo
// candidato cuentan como existentes; con otro candidato, conflicto.
func (r *RepositorioConstitucionPostgreSQL) RegistrarVinculos(ctx context.Context, actaRef string, vinculos []ports.VinculoCandidato, registradaEn time.Time) (ports.ReciboVinculosCandidato, error) {
	if ctx == nil || r == nil || r.pool == nil {
		return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return ports.ReciboVinculosCandidato{}, err
	}
	if actaRef == "" || len(vinculos) == 0 || registradaEn.IsZero() {
		return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaInvalida
	}
	entradas := make([]vinculoCandidatoJSON, len(vinculos))
	for i, v := range vinculos {
		if v.CandidatoRef == "" || v.ParticipacionRef == "" {
			return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaInvalida
		}
		entradas[i] = vinculoCandidatoJSON{CandidatoRef: v.CandidatoRef, ParticipacionRef: v.ParticipacionRef}
	}
	contenidoVinculos, err := json.Marshal(entradas)
	if err != nil {
		return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaInvalida
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaNoDisponible
	}
	defer tx.Rollback(context.Background())
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.registrar_vinculos_candidato_v1($1::text, $2::jsonb, $3::timestamptz)`,
		actaRef, contenidoVinculos, registradaEn.UTC()).Scan(&contenido)
	if err != nil {
		if errors.Is(errorConstitucion(ctx, err), ports.ErrConstitucionBolsaEnConflicto) {
			return ports.ReciboVinculosCandidato{}, ports.ErrVinculoCandidatoEnConflicto
		}
		return ports.ReciboVinculosCandidato{}, errorConstitucion(ctx, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaNoDisponible
	}
	var recibo struct {
		Nuevos     uint64 `json:"nuevos"`
		Existentes uint64 `json:"existentes"`
	}
	if json.Unmarshal(contenido, &recibo) != nil {
		return ports.ReciboVinculosCandidato{}, ports.ErrConstitucionBolsaNoDisponible
	}
	return ports.ReciboVinculosCandidato{Nuevos: recibo.Nuevos, Existentes: recibo.Existentes}, nil
}

// ParticipacionesCandidato devuelve las participaciones de una persona
// candidata (la constitución más reciente primero).
func (r *RepositorioConstitucionPostgreSQL) ParticipacionesCandidato(ctx context.Context, candidatoRef string) ([]ports.ParticipacionCandidato, error) {
	if ctx == nil || r == nil || r.pool == nil || candidatoRef == "" {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	filas, err := r.pool.Query(ctx, `SELECT participacion_ref, acta_ref, bolsa_ref, version_bolsa, categoria_ref, vigente_desde, vigente_hasta, estado,
		instantanea_ref, version_instantanea, orden, total_participaciones, confirmada_en
		FROM vec_bolsa_llamamientos.listar_participaciones_candidato_v1($1::text)`, candidatoRef)
	if err != nil {
		return nil, errorConstitucion(ctx, err)
	}
	defer filas.Close()
	resultado := []ports.ParticipacionCandidato{}
	for filas.Next() {
		var p ports.ParticipacionCandidato
		var versionBolsa, versionInstantanea, orden, total int64
		var hasta *time.Time
		if err := filas.Scan(&p.ParticipacionRef, &p.ActaRef, &p.BolsaRef, &versionBolsa, &p.CategoriaRef, &p.VigenteDesde, &hasta, &p.EstadoBolsa,
			&p.InstantaneaRef, &versionInstantanea, &orden, &total, &p.ConfirmadaEn); err != nil || versionBolsa <= 0 || versionInstantanea <= 0 || orden <= 0 || total < 0 {
			return nil, ports.ErrConstitucionBolsaNoDisponible
		}
		p.VersionBolsa, p.VersionInstantanea, p.Orden, p.TotalParticipaciones = uint64(versionBolsa), uint64(versionInstantanea), uint64(orden), uint64(total)
		if hasta != nil {
			h := hasta.UTC()
			p.VigenteHasta = &h
		}
		p.VigenteDesde, p.ConfirmadaEn = p.VigenteDesde.UTC(), p.ConfirmadaEn.UTC()
		resultado = append(resultado, p)
	}
	if filas.Err() != nil {
		return nil, ports.ErrConstitucionBolsaNoDisponible
	}
	return resultado, nil
}
