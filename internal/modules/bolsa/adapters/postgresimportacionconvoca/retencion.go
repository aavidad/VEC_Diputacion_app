package postgresimportacionconvoca

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
)

const (
	funcionCambiarBloqueoV1 = "vec_bolsa_importacion_convoca.cambiar_bloqueo_retencion_v1"
	funcionExpurgarV1       = "vec_bolsa_importacion_convoca.expurgar_staging_vencido_v1"
)

var _ aplicacion.RepositorioRetencion = (*RepositorioRetencionPostgreSQL)(nil)

// RepositorioRetencionPostgreSQL requiere identidad de conservación. No puede
// importar, recuperar staging ni conciliar.
type RepositorioRetencionPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoRepositorioRetencionPostgreSQL(
	pool *pgxpool.Pool,
) (*RepositorioRetencionPostgreSQL, error) {
	return nuevoRepositorioRetencionPostgreSQL(pool)
}

func nuevoRepositorioRetencionPostgreSQL(
	pool iniciadorTransacciones,
) (*RepositorioRetencionPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrRepositorioNoDisponible
	}
	return &RepositorioRetencionPostgreSQL{pool: pool}, nil
}

type confirmacionBloqueoPostgreSQL struct {
	ImportacionRef string `json:"importacion_ref"`
	DecisionRef    string `json:"decision_ref"`
	Bloqueado      bool   `json:"bloqueado"`
	RegistradaEn   string `json:"registrada_en"`
	Reutilizada    bool   `json:"reutilizada"`
}

type resultadoExpurgoPostgreSQL struct {
	EjecucionRef string `json:"ejecucion_ref"`
	Lotes        int    `json:"lotes"`
	Filas        int    `json:"filas"`
	EjecutadaEn  string `json:"ejecutada_en"`
	Reutilizada  bool   `json:"reutilizada"`
}

func (r *RepositorioRetencionPostgreSQL) CambiarBloqueo(
	ctx context.Context,
	solicitud aplicacion.SolicitudCambioBloqueoRetencion,
) (aplicacion.ConfirmacionCambioBloqueo, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) {
		return aplicacion.ConfirmacionCambioBloqueo{}, ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, err
	}
	if solicitud.Validar() != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, aplicacion.ErrGestionDurableInvalida
	}
	var ultimo error
	for intento := 0; intento < maximoReintentos; intento++ {
		resultado, err := r.cambiarBloqueoUnaVez(ctx, solicitud)
		if err == nil {
			return resultado, nil
		}
		ultimo = err
		if ctx.Err() != nil {
			return aplicacion.ConfirmacionCambioBloqueo{}, ctx.Err()
		}
		if !esReintentable(err) {
			return aplicacion.ConfirmacionCambioBloqueo{}, errorPostgreSQL(ctx, err)
		}
	}
	return aplicacion.ConfirmacionCambioBloqueo{}, errorPostgreSQL(ctx, ultimo)
}

func (r *RepositorioRetencionPostgreSQL) cambiarBloqueoUnaVez(
	ctx context.Context,
	solicitud aplicacion.SolicitudCambioBloqueoRetencion,
) (aplicacion.ConfirmacionCambioBloqueo, error) {
	tx, err := iniciarTransaccion(ctx, r.pool, pgx.ReadWrite)
	if err != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, err
	}
	defer revertir(tx)
	var contenido []byte
	err = tx.QueryRow(ctx, `
		SELECT `+funcionCambiarBloqueoV1+`(
			$1::text, $2::text, $3::text, $4::text, $5::boolean
		)`,
		solicitud.ImportacionRef, solicitud.DecisionRef, solicitud.ActorRef,
		solicitud.MotivoCodigo, solicitud.Bloqueado,
	).Scan(&contenido)
	if err != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, err
	}
	defer borrarBytes(contenido)
	var datos confirmacionBloqueoPostgreSQL
	if decodificarJSONExacto(contenido, &datos) != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, ErrResultadoNoConfiable
	}
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", datos.RegistradaEn)
	if err != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, ErrResultadoNoConfiable
	}
	resultado := aplicacion.ConfirmacionCambioBloqueo{
		ImportacionRef: datos.ImportacionRef, DecisionRef: datos.DecisionRef,
		Bloqueado: datos.Bloqueado, RegistradaEn: instante.UTC(),
		Reutilizada: datos.Reutilizada,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, ErrResultadoNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return aplicacion.ConfirmacionCambioBloqueo{}, err
	}
	return resultado, nil
}

func (r *RepositorioRetencionPostgreSQL) ExpurgarVencidos(
	ctx context.Context,
	solicitud aplicacion.SolicitudExpurgoStaging,
) (aplicacion.ResultadoExpurgoStaging, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) {
		return aplicacion.ResultadoExpurgoStaging{}, ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return aplicacion.ResultadoExpurgoStaging{}, err
	}
	if solicitud.Validar() != nil {
		return aplicacion.ResultadoExpurgoStaging{}, aplicacion.ErrGestionDurableInvalida
	}
	var ultimo error
	for intento := 0; intento < maximoReintentos; intento++ {
		resultado, err := r.expurgarUnaVez(ctx, solicitud)
		if err == nil {
			return resultado, nil
		}
		ultimo = err
		if ctx.Err() != nil {
			return aplicacion.ResultadoExpurgoStaging{}, ctx.Err()
		}
		if !esReintentable(err) {
			return aplicacion.ResultadoExpurgoStaging{}, errorPostgreSQL(ctx, err)
		}
	}
	return aplicacion.ResultadoExpurgoStaging{}, errorPostgreSQL(ctx, ultimo)
}

func (r *RepositorioRetencionPostgreSQL) expurgarUnaVez(
	ctx context.Context,
	solicitud aplicacion.SolicitudExpurgoStaging,
) (aplicacion.ResultadoExpurgoStaging, error) {
	tx, err := iniciarTransaccion(ctx, r.pool, pgx.ReadWrite)
	if err != nil {
		return aplicacion.ResultadoExpurgoStaging{}, err
	}
	defer revertir(tx)
	var contenido []byte
	err = tx.QueryRow(ctx, `
		SELECT `+funcionExpurgarV1+`(
			$1::text, $2::text, $3::text, $4::bigint, $5::integer
		)`,
		solicitud.EjecucionRef, solicitud.ActorRef, solicitud.PoliticaRef,
		solicitud.PoliticaVersion, solicitud.Limite,
	).Scan(&contenido)
	if err != nil {
		return aplicacion.ResultadoExpurgoStaging{}, err
	}
	defer borrarBytes(contenido)
	var datos resultadoExpurgoPostgreSQL
	if decodificarJSONExacto(contenido, &datos) != nil {
		return aplicacion.ResultadoExpurgoStaging{}, ErrResultadoNoConfiable
	}
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", datos.EjecutadaEn)
	if err != nil {
		return aplicacion.ResultadoExpurgoStaging{}, ErrResultadoNoConfiable
	}
	resultado := aplicacion.ResultadoExpurgoStaging{
		EjecucionRef: datos.EjecucionRef, Lotes: datos.Lotes, Filas: datos.Filas,
		EjecutadaEn: instante.UTC(), Reutilizada: datos.Reutilizada,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return aplicacion.ResultadoExpurgoStaging{}, ErrResultadoNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return aplicacion.ResultadoExpurgoStaging{}, err
	}
	return resultado, nil
}
