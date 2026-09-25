// Package postgres implementa el repositorio de lectura de Calendarios sobre
// las funciones SECURITY DEFINER del esquema vec_calendarios. Nunca consulta
// sus tablas directamente ni escribe.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"vec-diputacion-granada/internal/modules/calendarios/domain"
	"vec-diputacion-granada/internal/modules/calendarios/ports"
)

var ErrNoDisponible = errors.New("calendarios postgres: no disponible")

const (
	consultaVersiones = `SELECT id, ambito_tipo, ambito_ref, anio, numero, sustituye_id, denominacion, procedencia_norma,
 procedencia_referencia, procedencia_publicada_en, sintetica, comunidad_ref, municipio_ref, conocido_desde, dias
 FROM vec_calendarios.versiones_vigentes_v1($1, $2::text[], $3::text[], $4)`
	consultaCentros = `SELECT id, ambito_tipo, ambito_ref, anio, numero, sustituye_id, denominacion, procedencia_norma,
 procedencia_referencia, procedencia_publicada_en, sintetica, comunidad_ref, municipio_ref, conocido_desde, '[]'::jsonb
 FROM vec_calendarios.centros_con_calendario_v1($1, $2)`
	maximoAmbitos        = 16
	maximoFilas          = 1000
	maximoDiasPorVersion = 400
	maximoBytesDias      = 128 << 10
)

// Iniciador es la parte de un pool que usa el adaptador: una transacción de
// solo lectura por consulta.
type Iniciador interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type Repositorio struct {
	pool Iniciador
}

var _ ports.RepositorioCalendarios = (*Repositorio)(nil)

func NuevoRepositorio(pool Iniciador) (*Repositorio, error) {
	if pool == nil || (reflect.ValueOf(pool).Kind() == reflect.Pointer && reflect.ValueOf(pool).IsNil()) {
		return nil, ErrNoDisponible
	}
	return &Repositorio{pool: pool}, nil
}

func (r *Repositorio) VersionesVigentes(ctx context.Context, c ports.ConsultaVersiones) ([]domain.VersionConDias, error) {
	if len(c.Ambitos) == 0 || len(c.Ambitos) > maximoAmbitos || c.ConocidoEn.IsZero() {
		return nil, ErrNoDisponible
	}
	tipos, refs := make([]string, len(c.Ambitos)), make([]string, len(c.Ambitos))
	for i, a := range c.Ambitos {
		if a.Validar() != nil {
			return nil, ErrNoDisponible
		}
		tipos[i], refs[i] = string(a.Tipo), a.Ref
	}
	return r.leer(ctx, consultaVersiones, c.Anio, tipos, refs, c.ConocidoEn.UTC())
}

func (r *Repositorio) CentrosConCalendario(ctx context.Context, anio int, conocidoEn time.Time) ([]domain.VersionCalendario, error) {
	if conocidoEn.IsZero() {
		return nil, ErrNoDisponible
	}
	filas, err := r.leer(ctx, consultaCentros, anio, conocidoEn.UTC())
	if err != nil {
		return nil, err
	}
	versiones := make([]domain.VersionCalendario, len(filas))
	for i, f := range filas {
		versiones[i] = f.Version
	}
	return versiones, nil
}

type diaFila struct {
	Fecha        string `json:"fecha"`
	Efecto       string `json:"efecto"`
	Denominacion string `json:"denominacion"`
}

func (r *Repositorio) leer(ctx context.Context, sql string, args ...any) (resultado []domain.VersionConDias, err error) {
	if r == nil || r.pool == nil || ctx == nil {
		return nil, ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, opaco(ctx)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	filas, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, opaco(ctx)
	}
	defer filas.Close()
	for filas.Next() {
		if len(resultado) >= maximoFilas {
			return nil, ErrNoDisponible
		}
		v, err := escanear(filas)
		if err != nil {
			return nil, opaco(ctx)
		}
		resultado = append(resultado, v)
	}
	if filas.Err() != nil {
		return nil, opaco(ctx)
	}
	filas.Close()
	if err := tx.Commit(ctx); err != nil {
		return nil, opaco(ctx)
	}
	return resultado, nil
}

func escanear(filas pgx.Rows) (domain.VersionConDias, error) {
	var (
		v                               domain.VersionCalendario
		tipo                            string
		sustituye, comunidad, municipio pgtype.Text
		publicada                       pgtype.Date
		conocido                        pgtype.Timestamptz
		dias                            []byte
	)
	if err := filas.Scan(&v.ID, &tipo, &v.Ambito.Ref, &v.Anio, &v.Numero, &sustituye, &v.Denominacion, &v.Procedencia.Norma,
		&v.Procedencia.Referencia, &publicada, &v.Procedencia.Sintetica, &comunidad, &municipio, &conocido, &dias); err != nil {
		return domain.VersionConDias{}, err
	}
	if !publicada.Valid || !conocido.Valid || len(dias) > maximoBytesDias {
		return domain.VersionConDias{}, ErrNoDisponible
	}
	v.Ambito.Tipo = domain.TipoAmbito(tipo)
	v.SustituyeID, v.ComunidadRef, v.MunicipioRef = sustituye.String, comunidad.String, municipio.String
	var err error
	t := publicada.Time
	if v.Procedencia.PublicadaEn, err = domain.NuevaFechaCivil(t.Year(), int(t.Month()), t.Day()); err != nil {
		return domain.VersionConDias{}, err
	}
	v.ConocidoDesde = conocido.Time.UTC()
	var crudos []diaFila
	if err := json.Unmarshal(dias, &crudos); err != nil || len(crudos) > maximoDiasPorVersion {
		return domain.VersionConDias{}, ErrNoDisponible
	}
	resultado := domain.VersionConDias{Version: v, Dias: make([]domain.DiaSenalado, 0, len(crudos))}
	for _, c := range crudos {
		f, err := domain.ParsearFechaCivil(c.Fecha)
		if err != nil {
			return domain.VersionConDias{}, err
		}
		resultado.Dias = append(resultado.Dias, domain.DiaSenalado{Fecha: f, Efecto: domain.Efecto(c.Efecto), Denominacion: c.Denominacion})
	}
	if err := resultado.Validar(); err != nil {
		return domain.VersionConDias{}, err
	}
	return resultado, nil
}

func opaco(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrNoDisponible
}
