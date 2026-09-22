package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

type ConsultaOrdenVigentePostgreSQL struct{ pool *pgxpool.Pool }

var _ puertosbolsa.ConsultaOrdenVigente = (*ConsultaOrdenVigentePostgreSQL)(nil)

func NuevaConsultaOrdenVigentePostgreSQL(pool *pgxpool.Pool) (*ConsultaOrdenVigentePostgreSQL, error) {
	if pool == nil {
		return nil, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	return &ConsultaOrdenVigentePostgreSQL{pool: pool}, nil
}

func (c *ConsultaOrdenVigentePostgreSQL) ConsultarOrdenVigente(ctx context.Context, bolsaRef string) (dominiobolsa.OrdenVigenteBolsa, error) {
	if c == nil || c.pool == nil || ctx == nil || bolsaRef == "" {
		return dominiobolsa.OrdenVigenteBolsa{}, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	filas, err := c.pool.Query(ctx, `SELECT politica_ref,version_politica,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,participacion_ref,orden_acta,orden_vigente,situacion,razon FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1($1,clock_timestamp())`, bolsaRef)
	if err != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, errorOrdenVigente(err)
	}
	defer filas.Close()
	var salida dominiobolsa.OrdenVigenteBolsa
	for filas.Next() {
		var posicion dominiobolsa.PosicionOrdenBolsa
		var ordenVigente *int64
		var version, ordenActa int64
		if err := filas.Scan(&salida.Politica.PoliticaRef, &version, &salida.Politica.Criterio, &salida.Politica.TipoLista, &salida.Politica.Reposicion, &salida.Politica.Provisional, &salida.Politica.Rotulo, &salida.Politica.Actor, &salida.Politica.VigenteDesde, &posicion.ParticipacionRef, &ordenActa, &ordenVigente, &posicion.Situacion, &posicion.Razon); err != nil {
			return dominiobolsa.OrdenVigenteBolsa{}, errorOrdenVigente(err)
		}
		salida.Politica.BolsaRef, salida.Politica.Version, posicion.OrdenActa = bolsaRef, uint64(version), uint64(ordenActa)
		if ordenVigente != nil {
			valor := uint64(*ordenVigente)
			posicion.OrdenVigente = &valor
		}
		salida.Posiciones = append(salida.Posiciones, posicion)
	}
	if err := filas.Err(); err != nil || len(salida.Posiciones) == 0 || salida.Validar() != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	return salida, nil
}

func errorOrdenVigente(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
	}
	return puertosbolsa.ErrConsultaOrdenVigenteNoDisponible
}
