package postgresimportacionconvoca

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	aplicacion "vec-diputacion-granada/internal/modules/bolsa/application/importacionconvoca"
)

const funcionConciliarV1 = "vec_bolsa_importacion_convoca.conciliar_v1"

var _ aplicacion.RepositorioConciliaciones = (*RepositorioConciliacionesPostgreSQL)(nil)

// RepositorioConciliacionesPostgreSQL requiere una identidad distinta de la
// importadora. No puede leer staging ni ejecutar expurgos.
type RepositorioConciliacionesPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoRepositorioConciliacionesPostgreSQL(
	pool *pgxpool.Pool,
) (*RepositorioConciliacionesPostgreSQL, error) {
	return nuevoRepositorioConciliacionesPostgreSQL(pool)
}

func nuevoRepositorioConciliacionesPostgreSQL(
	pool iniciadorTransacciones,
) (*RepositorioConciliacionesPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, ErrRepositorioNoDisponible
	}
	return &RepositorioConciliacionesPostgreSQL{pool: pool}, nil
}

type confirmacionConciliacionPostgreSQL struct {
	ImportacionRef  string `json:"importacion_ref"`
	ConciliacionRef string `json:"conciliacion_ref"`
	Resultado       string `json:"resultado"`
	RegistradaEn    string `json:"registrada_en"`
	Reutilizada     bool   `json:"reutilizada"`
}

func (r *RepositorioConciliacionesPostgreSQL) Conciliar(
	ctx context.Context,
	solicitud aplicacion.SolicitudConciliacion,
) (aplicacion.ConfirmacionConciliacion, error) {
	if ctx == nil || r == nil || valorNulo(r.pool) {
		return aplicacion.ConfirmacionConciliacion{}, ErrRepositorioNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return aplicacion.ConfirmacionConciliacion{}, err
	}
	if solicitud.Validar() != nil {
		return aplicacion.ConfirmacionConciliacion{}, aplicacion.ErrGestionDurableInvalida
	}
	var ultimo error
	for intento := 0; intento < maximoReintentos; intento++ {
		resultado, err := r.conciliarUnaVez(ctx, solicitud)
		if err == nil {
			return resultado, nil
		}
		ultimo = err
		if ctx.Err() != nil {
			return aplicacion.ConfirmacionConciliacion{}, ctx.Err()
		}
		if !esReintentable(err) {
			return aplicacion.ConfirmacionConciliacion{}, errorPostgreSQL(ctx, err)
		}
	}
	return aplicacion.ConfirmacionConciliacion{}, errorPostgreSQL(ctx, ultimo)
}

func (r *RepositorioConciliacionesPostgreSQL) conciliarUnaVez(
	ctx context.Context,
	solicitud aplicacion.SolicitudConciliacion,
) (aplicacion.ConfirmacionConciliacion, error) {
	tx, err := iniciarTransaccion(ctx, r.pool, pgx.ReadWrite)
	if err != nil {
		return aplicacion.ConfirmacionConciliacion{}, err
	}
	defer revertir(tx)
	var contenido []byte
	err = tx.QueryRow(ctx, `
		SELECT `+funcionConciliarV1+`(
			$1::text, $2::text, $3::text, $4::text, $5::text, $6::text
		)`,
		solicitud.ImportacionRef, solicitud.ConciliacionRef,
		solicitud.RegistroCorporativoRef, string(solicitud.Resultado),
		solicitud.ActorRef, solicitud.MotivoCodigo,
	).Scan(&contenido)
	if err != nil {
		return aplicacion.ConfirmacionConciliacion{}, err
	}
	defer borrarBytes(contenido)
	var datos confirmacionConciliacionPostgreSQL
	if decodificarJSONExacto(contenido, &datos) != nil {
		return aplicacion.ConfirmacionConciliacion{}, ErrResultadoNoConfiable
	}
	instante, err := time.Parse("2006-01-02T15:04:05.000000Z", datos.RegistradaEn)
	if err != nil {
		return aplicacion.ConfirmacionConciliacion{}, ErrResultadoNoConfiable
	}
	resultado := aplicacion.ConfirmacionConciliacion{
		ImportacionRef: datos.ImportacionRef, ConciliacionRef: datos.ConciliacionRef,
		Resultado:    aplicacion.ResultadoConciliado(datos.Resultado),
		RegistradaEn: instante.UTC(), Reutilizada: datos.Reutilizada,
	}
	if resultado.ValidarPara(solicitud) != nil {
		return aplicacion.ConfirmacionConciliacion{}, ErrResultadoNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return aplicacion.ConfirmacionConciliacion{}, err
	}
	return resultado, nil
}
