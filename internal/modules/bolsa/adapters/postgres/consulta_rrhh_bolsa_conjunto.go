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

func (l *LectorBolsaRRHHConjunto) LeerBolsa(ctx context.Context, bolsaRef string, corte time.Time) (dominiobolsa.OrdenVigenteBolsa, []ports.SituacionBolsaRRHH, int, error) {
	var vacio dominiobolsa.OrdenVigenteBolsa
	if l == nil || l.pool == nil || ctx == nil || bolsaRef == "" || corte.IsZero() {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, nil, 0, err
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacio, nil, 0, errorLecturaBolsaRRHHConjunto(ctx, ports.ErrResumenBolsasNoDisponible)
	}
	defer tx.Rollback(context.Background())
	orden, err := leerOrdenBolsaRRHHConjunto(ctx, tx, bolsaRef, corte)
	if err != nil {
		return vacio, nil, 0, errorLecturaBolsaRRHHConjunto(ctx, err)
	}
	filas, err := leerSituacionesBolsaRRHHConjunto(ctx, tx, bolsaRef, corte)
	if err != nil {
		return vacio, nil, 0, errorLecturaBolsaRRHHConjunto(ctx, err)
	}
	if len(filas) != len(orden.Posiciones) {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	conteos, err := leerLlamamientosEnCursoResumen(ctx, tx)
	if err != nil {
		return vacio, nil, 0, errorLecturaBolsaRRHHConjunto(ctx, err)
	}
	enCurso, ok := conteos[bolsaRef]
	if !ok || enCurso < 0 {
		return vacio, nil, 0, ports.ErrResumenBolsasNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return vacio, nil, 0, errorLecturaBolsaRRHHConjunto(ctx, ports.ErrResumenBolsasNoDisponible)
	}
	return orden, filas, enCurso, nil
}

func errorLecturaBolsaRRHHConjunto(ctx context.Context, fallo error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fallo
}

func leerSituacionesBolsaRRHHConjunto(ctx context.Context, tx pgx.Tx, bolsaRef string, corte time.Time) ([]ports.SituacionBolsaRRHH, error) {
	filas, err := tx.Query(ctx, `SELECT bolsa_ref,categoria_ref,confirmada_en,instantanea_ref,version_instantanea,orden,
		participacion_ref,situacion,desde,fecha_disponible,cese_fecha_efecto,cese_disponible_desde,
		cese_en_restriccion,cese_trabajo_cesado,cese_pendiente,pendiente_desde,fila_numero
		FROM vec_bolsa_llamamientos.leer_situaciones_bolsa_rrhh_v1($1,$2)`, bolsaRef, corte.UTC())
	if err != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	defer filas.Close()
	salida := make([]ports.SituacionBolsaRRHH, 0)
	for filas.Next() {
		if len(salida) >= 20000 {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		var fila ports.SituacionBolsaRRHH
		var version, orden int64
		var numero int
		var situacion *string
		var desde, disponible *time.Time
		var efecto, disponibleCese, pendienteDesde *time.Time
		var restringida, cesado *bool
		var pendiente bool
		if err := filas.Scan(&fila.BolsaRef, &fila.CategoriaRef, &fila.ConfirmadaEn, &fila.InstantaneaRef,
			&version, &orden, &fila.ParticipacionRef, &situacion, &desde, &disponible,
			&efecto, &disponibleCese, &restringida, &cesado, &pendiente, &pendienteDesde, &numero); err != nil ||
			fila.BolsaRef != bolsaRef || version <= 0 || orden <= 0 || numero <= 0 || fila.ParticipacionRef == "" ||
			fila.CategoriaRef == "" || fila.InstantaneaRef == "" || fila.ConfirmadaEn.IsZero() || situacion == nil || desde == nil {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		fila.VersionInstantanea, fila.Orden = uint64(version), uint64(orden)
		fila.Situacion = &ports.SituacionParticipacion{ParticipacionRef: fila.ParticipacionRef, Situacion: *situacion, Desde: desde.UTC(), FechaDisponible: disponible}
		if pendiente {
			if pendienteDesde == nil || pendienteDesde.IsZero() || pendienteDesde.After(corte) || efecto != nil || disponibleCese != nil ||
				restringida == nil || !*restringida || cesado == nil || *cesado {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			fila.Cese = &ports.EstadoCese{CesePendiente: true, PendienteDesde: pendienteDesde.UTC()}
		} else {
			if pendienteDesde != nil {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			if efecto == nil && disponibleCese == nil && restringida == nil && cesado == nil {
				// No hay cese para esta participación.
			} else if efecto != nil && disponibleCese != nil && restringida != nil && cesado != nil {
				estado, presente, err := validarEstadoCese(*efecto, *disponibleCese, *restringida, *cesado, corte)
				if err != nil || !presente {
					return nil, ports.ErrResumenBolsasNoDisponible
				}
				fila.Cese = &estado
			} else {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
		}
		fila.FilaNumero = numero
		salida = append(salida, fila)
	}
	if filas.Err() != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return salida, nil
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
