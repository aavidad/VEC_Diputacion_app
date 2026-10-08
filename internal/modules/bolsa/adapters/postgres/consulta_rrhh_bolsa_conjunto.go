package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// LectorBolsaRRHHConjunto reúne orden, situaciones y recuento de una bolsa
// bajo el mismo corte y la misma instantánea de PostgreSQL. B82 calcula los
// estados en conjunto; no emite una consulta por participación.
type LectorBolsaRRHHConjunto struct{ pool *pgxpool.Pool }

func NuevoLectorBolsaRRHHConjunto(pool *pgxpool.Pool) (*LectorBolsaRRHHConjunto, error) {
	if pool == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &LectorBolsaRRHHConjunto{pool: pool}, nil
}

func (l *LectorBolsaRRHHConjunto) LeerBolsa(ctx context.Context, bolsaRef string, corte time.Time) (dominiobolsa.OrdenVigenteBolsa, []ports.SituacionResumenParticipacion, int, error) {
	var vacio dominiobolsa.OrdenVigenteBolsa
	if l == nil || l.pool == nil || ctx == nil || bolsaRef == "" || corte.IsZero() {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	orden, err := leerOrdenBolsaRRHHConjunto(ctx, tx, bolsaRef, corte)
	if err != nil {
		return vacio, nil, 0, err
	}
	resumen, err := leerSituacionesResumen(ctx, tx, corte)
	if err != nil {
		return vacio, nil, 0, err
	}
	filas := make([]ports.SituacionResumenParticipacion, 0, len(orden.Posiciones))
	for _, fila := range resumen {
		if fila.BolsaRef == bolsaRef {
			filas = append(filas, fila)
		}
	}
	if len(filas) != len(orden.Posiciones) {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	conteos, err := leerLlamamientosEnCursoResumen(ctx, tx)
	if err != nil {
		return vacio, nil, 0, err
	}
	enCurso, ok := conteos[bolsaRef]
	if !ok || enCurso < 0 {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	return orden, filas, enCurso, nil
}

func leerOrdenBolsaRRHHConjunto(ctx context.Context, tx pgx.Tx, bolsaRef string, corte time.Time) (dominiobolsa.OrdenVigenteBolsa, error) {
	var salida dominiobolsa.OrdenVigenteBolsa
	filas, err := tx.Query(ctx, `SELECT politica_ref,version_politica,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde,
		participacion_ref,orden_acta,orden_vigente,situacion,razon
		FROM vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1($1,$2)`, bolsaRef, corte.UTC())
	if err != nil {
		return salida, ports.ErrConsultaOrdenVigenteNoDisponible
	}
	defer filas.Close()
	for filas.Next() {
		var posicion dominiobolsa.PosicionOrdenBolsa
		var version, ordenActa int64
		var ordenVigente *int64
		if err := filas.Scan(&salida.Politica.PoliticaRef, &version, &salida.Politica.Criterio, &salida.Politica.TipoLista,
			&salida.Politica.Reposicion, &salida.Politica.Provisional, &salida.Politica.Rotulo, &salida.Politica.Actor,
			&salida.Politica.VigenteDesde, &posicion.ParticipacionRef, &ordenActa, &ordenVigente,
			&posicion.Situacion, &posicion.Razon); err != nil || version <= 0 || ordenActa <= 0 {
			return dominiobolsa.OrdenVigenteBolsa{}, ports.ErrConsultaOrdenVigenteNoDisponible
		}
		salida.Politica.BolsaRef, salida.Politica.Version, posicion.OrdenActa = bolsaRef, uint64(version), uint64(ordenActa)
		if ordenVigente != nil {
			if *ordenVigente <= 0 {
				return dominiobolsa.OrdenVigenteBolsa{}, ports.ErrConsultaOrdenVigenteNoDisponible
			}
			v := uint64(*ordenVigente)
			posicion.OrdenVigente = &v
		}
		salida.Posiciones = append(salida.Posiciones, posicion)
	}
	if filas.Err() != nil || len(salida.Posiciones) == 0 || salida.Validar() != nil {
		return dominiobolsa.OrdenVigenteBolsa{}, ports.ErrConsultaOrdenVigenteNoDisponible
	}
	return salida, nil
}
