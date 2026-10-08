package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// maximoFilasResumenBolsas acota lo que se acepta de la base antes de crecer.
const maximoFilasResumenBolsas = 200000

// LectorResumenBolsasPostgreSQL lee las funciones de conjunto B82 y el
// recuento agrupado B85. Todas comprueban en la base el rol de quien llama.
type LectorResumenBolsasPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.LectorResumenBolsas = (*LectorResumenBolsasPostgreSQL)(nil)

func NuevoLectorResumenBolsasPostgreSQL(pool *pgxpool.Pool) (*LectorResumenBolsasPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &LectorResumenBolsasPostgreSQL{pool: pool}, nil
}

// consultaResumenBolsas es lo que necesitan las lecturas de una transacción.
type consultaResumenBolsas interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// LeerResumen lee situaciones, políticas y recuentos en una transacción
// REPEATABLE READ de solo lectura. Una participación NULL, una bolsa sin
// recuento o cualquier fila incoherente hace fallar la lectura entera.
func (l *LectorResumenBolsasPostgreSQL) LeerResumen(ctx context.Context, corte time.Time) (ports.ResumenBolsasRRHH, error) {
	var vacio ports.ResumenBolsasRRHH
	if l == nil || l.pool == nil || ctx == nil || corte.IsZero() {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := leerSituacionesResumen(ctx, tx, corte)
	if err != nil {
		return vacio, err
	}
	politicas, err := leerPoliticasResumen(ctx, tx, corte)
	if err != nil {
		return vacio, err
	}
	conteos, err := leerLlamamientosEnCursoResumen(ctx, tx)
	if err != nil {
		return vacio, err
	}
	esperadas := make(map[string]struct{}, len(politicas))
	for _, fila := range filas {
		esperadas[fila.BolsaRef] = struct{}{}
	}
	if len(conteos) != len(esperadas) {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	for bolsaRef := range esperadas {
		if _, existe := conteos[bolsaRef]; !existe {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
	}
	if tx.Commit(ctx) != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	return ports.ResumenBolsasRRHH{Situaciones: filas, Politicas: politicas, LlamamientosEnCurso: conteos}, nil
}

func leerLlamamientosEnCursoResumen(ctx context.Context, consulta consultaResumenBolsas) (map[string]int, error) {
	filas, err := consulta.Query(ctx, `SELECT bolsa_ref,llamamientos_en_curso
		FROM vec_bolsa_llamamientos.leer_llamamientos_en_curso_bolsas_v1()`)
	if err != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	defer filas.Close()
	salida := map[string]int{}
	for filas.Next() {
		if len(salida) >= maximoFilasResumenBolsas {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		var bolsaRef string
		var total int
		if err := filas.Scan(&bolsaRef, &total); err != nil || bolsaRef == "" || total < 0 {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		if _, repetida := salida[bolsaRef]; repetida {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		salida[bolsaRef] = total
	}
	if filas.Err() != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return salida, nil
}

func leerSituacionesResumen(ctx context.Context, consulta consultaResumenBolsas, corte time.Time) ([]ports.SituacionResumenParticipacion, error) {
	filas, err := consulta.Query(ctx, `SELECT bolsa_ref,categoria_ref,confirmada_en,instantanea_ref,version_instantanea,orden,participacion_ref,situacion,desde,fecha_disponible,
		cese_fecha_efecto,cese_disponible_desde,cese_en_restriccion,cese_trabajo_cesado
		FROM vec_bolsa_llamamientos.leer_resumen_situaciones_bolsas_v1($1)`, corte.UTC())
	if err != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	defer filas.Close()
	var salida []ports.SituacionResumenParticipacion
	for filas.Next() {
		if len(salida) >= maximoFilasResumenBolsas {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		var fila ports.SituacionResumenParticipacion
		var version, orden int64
		var situacion *string
		var desde, disponible *time.Time
		var efecto, disponibleCese *time.Time
		var restringida, cesado *bool
		if err := filas.Scan(&fila.BolsaRef, &fila.CategoriaRef, &fila.ConfirmadaEn, &fila.InstantaneaRef, &version, &orden, &fila.ParticipacionRef, &situacion, &desde, &disponible,
			&efecto, &disponibleCese, &restringida, &cesado); err != nil || version <= 0 || orden <= 0 || fila.ParticipacionRef == "" || fila.BolsaRef == "" || fila.CategoriaRef == "" || fila.ConfirmadaEn.IsZero() {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		fila.VersionInstantanea, fila.Orden = uint64(version), uint64(orden)
		if situacion != nil {
			if desde == nil {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			fila.Situacion = &ports.SituacionParticipacion{ParticipacionRef: fila.ParticipacionRef, Situacion: *situacion, Desde: *desde, FechaDisponible: disponible}
		}
		if efecto != nil {
			if disponibleCese == nil || restringida == nil || cesado == nil {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			estado, presente, err := validarEstadoCese(*efecto, *disponibleCese, *restringida, *cesado, corte)
			if err != nil || !presente {
				return nil, ports.ErrResumenBolsasNoDisponible
			}
			fila.Cese = &estado
		}
		salida = append(salida, fila)
	}
	if filas.Err() != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return salida, nil
}

func leerPoliticasResumen(ctx context.Context, consulta consultaResumenBolsas, en time.Time) (map[string]dominiobolsa.PoliticaOrdenBolsa, error) {
	filas, err := consulta.Query(ctx, `SELECT bolsa_ref,politica_ref,version_politica,criterio,tipo_lista,reposicion,provisional,rotulo,actor,vigente_desde
		FROM vec_bolsa_llamamientos.leer_politicas_orden_vigentes_v1($1)`, en.UTC())
	if err != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	defer filas.Close()
	salida := map[string]dominiobolsa.PoliticaOrdenBolsa{}
	for filas.Next() {
		if len(salida) >= maximoFilasResumenBolsas {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		var p dominiobolsa.PoliticaOrdenBolsa
		var version int64
		if err := filas.Scan(&p.BolsaRef, &p.PoliticaRef, &version, &p.Criterio, &p.TipoLista, &p.Reposicion, &p.Provisional, &p.Rotulo, &p.Actor, &p.VigenteDesde); err != nil || version <= 0 {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		p.Version = uint64(version)
		if _, repetida := salida[p.BolsaRef]; repetida || p.Validar() != nil {
			return nil, ports.ErrResumenBolsasNoDisponible
		}
		salida[p.BolsaRef] = p
	}
	if filas.Err() != nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return salida, nil
}
