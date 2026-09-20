// La cuenta de ejecución B11 invoca exclusivamente la fachada nominal que
// consume y revalida la autorización AD3 antes de leer las participaciones.
package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionConsultarParticipacionesPropiasB11V1 = "vec_bolsa_llamamientos.consultar_participaciones_propias_v1"
	consultaCanonicaParticipacionesPropiasB11V1 = `{"esquema":"vec.bolsa.consulta-participaciones-propias.b11.v1","version":1}`
	maximoBytesRespuestaParticipacionesPropias  = 2 * 1024 * 1024
	maximaProfundidadParticipacionesPropias     = 16
)

var (
	ErrFuenteParticipacionesPropiasPostgreSQLNoDisponible = errors.New("bolsa: fuente PostgreSQL de participaciones propias no disponible")
	ErrConsultaParticipacionesPropiasPostgreSQLEnCurso    = errors.New("bolsa: consulta PostgreSQL de participaciones propias en curso")
)

var _ puertosbolsa.ConsultaParticipacionesPropiasPersistente = (*ConsultaParticipacionesPropiasPostgreSQL)(nil)

// ConsultaParticipacionesPropiasPostgreSQL conserva la unidad transaccional
// de registro/revalidación AD3 y la lectura B11 dentro de una única fachada.
type ConsultaParticipacionesPropiasPostgreSQL struct{ pool iniciadorTransacciones }

func NuevaConsultaParticipacionesPropiasPostgreSQL(pool *pgxpool.Pool) (*ConsultaParticipacionesPropiasPostgreSQL, error) {
	return nuevaConsultaParticipacionesPropiasPostgreSQL(pool)
}

func nuevaConsultaParticipacionesPropiasPostgreSQL(pool iniciadorTransacciones) (*ConsultaParticipacionesPropiasPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrFuenteParticipacionesPropiasPostgreSQLNoDisponible
	}
	return &ConsultaParticipacionesPropiasPostgreSQL{pool: pool}, nil
}

func (r *ConsultaParticipacionesPropiasPostgreSQL) ConsultarParticipacionesPropias(
	ctx context.Context,
	consulta puertosbolsa.ConsultaParticipacionesPropias,
	material puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3,
) (puertosbolsa.ResultadoParticipacionesPropias, error) {
	vacio := puertosbolsa.ResultadoParticipacionesPropias{}
	if ctx == nil || consulta.Validar() != nil || material.ValidarEstructura() != nil {
		return vacio, puertosbolsa.ErrConsultaParticipacionesPropiasInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if r == nil || valorNulo(r.pool) {
		return vacio, ErrFuenteParticipacionesPropiasPostgreSQLNoDisponible
	}
	recurso, err := puertosbolsa.RecursoAutorizableParticipacionesPropias(consulta)
	if err != nil {
		return vacio, puertosbolsa.ErrConsultaParticipacionesPropiasInvalida
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != puertosbolsa.AccionConsultarParticipacionesPropias ||
		resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		resumen.AudienciaConsumo() != puertosbolsa.AudienciaParticipacionesPropias {
		return vacio, dominiovec.ErrAutorizacionDenegada
	}

	tx, err := r.iniciar(ctx)
	if err != nil {
		return vacio, err
	}
	defer revertir(tx)

	capacidad, decision, motivo, contexto := material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico()
	payload, sobre, evidencia, raiz := material.PayloadVECAD3(), material.SobreCOSESign1(), material.EvidenciaVerificacion(), material.RaizPublicaSPKI()
	defer borrarBytesPostgreSQL(capacidad, decision, motivo, contexto, payload, sobre, evidencia, raiz)

	consultaCanonica := []byte(consultaCanonicaParticipacionesPropiasB11V1)
	var respuesta []byte
	err = tx.QueryRow(ctx, `SELECT `+funcionConsultarParticipacionesPropiasB11V1+`(
  $1::bytea,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`,
		consultaCanonica, capacidad, decision, motivo, contexto,
		material.PersonaVersion(), material.PerfilVersion(), payload, sobre, evidencia, raiz,
	).Scan(&respuesta)
	defer borrarBytesPostgreSQL(respuesta)
	if err != nil {
		return vacio, errorPostgreSQLParticipacionesPropias(ctx, err)
	}
	resultado, err := decodificarParticipacionesPropiasPostgreSQL(respuesta, consulta)
	if err != nil {
		return vacio, err
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, errorPostgreSQLParticipacionesPropias(ctx, err)
	}
	return resultado, nil
}

func (r *ConsultaParticipacionesPropiasPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorPostgreSQLParticipacionesPropias(ctx, err)
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		revertir(tx)
		return nil, errorPostgreSQLParticipacionesPropias(ctx, err)
	}
	return tx, nil
}

func decodificarParticipacionesPropiasPostgreSQL(contenido []byte, consulta puertosbolsa.ConsultaParticipacionesPropias) (puertosbolsa.ResultadoParticipacionesPropias, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesRespuestaParticipacionesPropias || !utf8.Valid(contenido) || validarJSONParticipacionesPropiasNoAmbiguo(contenido) != nil {
		return puertosbolsa.ResultadoParticipacionesPropias{}, puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	var resultado puertosbolsa.ResultadoParticipacionesPropias
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&resultado); err != nil {
		return puertosbolsa.ResultadoParticipacionesPropias{}, puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return puertosbolsa.ResultadoParticipacionesPropias{}, puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	resultado, err := resultado.ClonarValidadoPara(consulta)
	if err != nil {
		return puertosbolsa.ResultadoParticipacionesPropias{}, puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	return resultado, nil
}

func validarJSONParticipacionesPropiasNoAmbiguo(contenido []byte) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	d.UseNumber()
	if err := consumirValorJSONParticipacionesPropias(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); !errors.Is(err, io.EOF) {
		return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	return nil
}

func consumirValorJSONParticipacionesPropias(d *json.Decoder, profundidad int) error {
	if profundidad > maximaProfundidadParticipacionesPropias {
		return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	token, err := d.Token()
	if err != nil {
		return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	delimitador, compuesto := token.(json.Delim)
	if !compuesto {
		return nil
	}
	switch delimitador {
	case '{':
		claves := map[string]struct{}{}
		for d.More() {
			tokenClave, err := d.Token()
			clave, esCadena := tokenClave.(string)
			if err != nil || !esCadena {
				return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
			}
			if _, existe := claves[clave]; existe {
				return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
			}
			claves[clave] = struct{}{}
			if err := consumirValorJSONParticipacionesPropias(d, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil || cierre != json.Delim('}') {
			return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
		}
	case '[':
		for d.More() {
			if err := consumirValorJSONParticipacionesPropias(d, profundidad+1); err != nil {
				return err
			}
		}
		cierre, err := d.Token()
		if err != nil || cierre != json.Delim(']') {
			return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
		}
	default:
		return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
	}
	return nil
}

func errorPostgreSQLParticipacionesPropias(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42501":
			return dominiovec.ErrAutorizacionDenegada
		case "40001", "40P01", "55P03", "57014":
			return ErrConsultaParticipacionesPropiasPostgreSQLEnCurso
		case "22000", "22023", "23503", "23514", "55000":
			return puertosbolsa.ErrResultadoParticipacionesPropiasInvalido
		}
	}
	return ErrFuenteParticipacionesPropiasPostgreSQLNoDisponible
}
