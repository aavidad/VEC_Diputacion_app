package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// maximoFilasResumenBolsas acota lo que se acepta de la base antes de crecer.
const maximoFilasResumenBolsas = 200000

// LectorResumenBolsasPostgreSQL lee las dos funciones de conjunto de Bolsa
// 000082. Ambas comprueban en la base el rol de quien llama.
type LectorResumenBolsasPostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.LectorResumenBolsas = (*LectorResumenBolsasPostgreSQL)(nil)
var _ ports.LectorResumenBolsasNominal = (*LectorResumenBolsasPostgreSQL)(nil)

func NuevoLectorResumenBolsasPostgreSQL(pool *pgxpool.Pool) (*LectorResumenBolsasPostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrResumenBolsasNoDisponible
	}
	return &LectorResumenBolsasPostgreSQL{pool: pool}, nil
}

// consultaResumenBolsas es lo que necesitan las dos lecturas de una transacción.
type consultaResumenBolsas interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

// LeerResumen lee situaciones y políticas en una misma transacción
// REPEATABLE READ de solo lectura: las dos ven la misma instantánea. Una
// participación NULL (constitución sin entradas) o cualquier fila
// incoherente hace fallar la lectura entera.
func (l *LectorResumenBolsasPostgreSQL) LeerResumen(ctx context.Context, corte time.Time) ([]ports.SituacionResumenParticipacion, map[string]dominiobolsa.PoliticaOrdenBolsa, error) {
	if l == nil || l.pool == nil || ctx == nil || corte.IsZero() {
		return nil, nil, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	filas, err := leerSituacionesResumen(ctx, tx, corte)
	if err != nil {
		return nil, nil, err
	}
	politicas, err := leerPoliticasResumen(ctx, tx, corte)
	if err != nil {
		return nil, nil, err
	}
	if tx.Commit(ctx) != nil {
		return nil, nil, ports.ErrResumenBolsasNoDisponible
	}
	return filas, politicas, nil
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

// LeerResumenNominal consume la decisión, inserta la auditoría común y lee
// situaciones, políticas y contadores bajo la misma transacción. El SQL B84
// no concede lectura directa de las tablas ni del staging al LOGIN.
func (l *LectorResumenBolsasPostgreSQL) LeerResumenNominal(ctx context.Context, accion string, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResumenBolsasNominal, error) {
	var vacio ports.ResumenBolsasNominal
	if l == nil || l.pool == nil || ctx == nil || ctx.Err() != nil || material.ValidarEstructura() != nil ||
		(accion != ports.AccionRRHHBolsasConsultar && accion != ports.AccionRRHHEstadisticasConsultar) ||
		material.ResumenCapacidad().Operacion() != accion {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	audiencia, recurso := ports.AudienciaRRHHBolsasConsultar, "coleccion:bolsa:rrhh:bolsas"
	if accion == ports.AccionRRHHEstadisticasConsultar {
		audiencia, recurso = ports.AudienciaRRHHEstadisticasConsultar, "coleccion:bolsa:rrhh:estadisticas"
	}
	if material.ResumenCapacidad().AudienciaConsumo() != audiencia || material.ResumenCapacidad().EfectoRef() != recurso {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	var crudo []byte
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.consultar_resumen_rrhh_nominal_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`,
		accion, material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(),
		material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI()).Scan(&crudo)
	if err != nil {
		return vacio, errorResumenNominal(err)
	}
	if len(crudo) == 0 || len(crudo) > 32<<20 {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	resultado, err := decodificarResumenNominal(crudo)
	if err != nil || ctx.Err() != nil || tx.Commit(ctx) != nil {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	return resultado, nil
}

func errorResumenNominal(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42501" {
		return vd.ErrAutorizacionDenegada
	}
	if errors.As(err, &pg) && pg.Code == "VBR04" {
		return ports.ErrBolsaRRHHNoEncontrada
	}
	if errors.As(err, &pg) && pg.Code == "VBR09" {
		return ports.ErrCursorRRHHNoEncontrado
	}
	return ports.ErrResumenBolsasNoDisponible
}

type resumenNominalSQL struct {
	GeneradoEn  time.Time `json:"generado_en"`
	Situaciones []struct {
		BolsaRef            string     `json:"bolsa_ref"`
		CategoriaRef        string     `json:"categoria_ref"`
		ConfirmadaEn        time.Time  `json:"confirmada_en"`
		InstantaneaRef      string     `json:"instantanea_ref"`
		Version             int64      `json:"version_instantanea"`
		Orden               int64      `json:"orden"`
		ParticipacionRef    string     `json:"participacion_ref"`
		Situacion           *string    `json:"situacion"`
		Desde               *time.Time `json:"desde"`
		FechaDisponible     *time.Time `json:"fecha_disponible"`
		CeseFechaEfecto     *string    `json:"cese_fecha_efecto"`
		CeseDisponibleDesde *string    `json:"cese_disponible_desde"`
		CeseEnRestriccion   *bool      `json:"cese_en_restriccion"`
		CeseTrabajoCesado   *bool      `json:"cese_trabajo_cesado"`
	} `json:"situaciones"`
	Politicas []struct {
		BolsaRef     string    `json:"bolsa_ref"`
		PoliticaRef  string    `json:"politica_ref"`
		Version      int64     `json:"version_politica"`
		Criterio     string    `json:"criterio"`
		TipoLista    string    `json:"tipo_lista"`
		Reposicion   string    `json:"reposicion"`
		Provisional  bool      `json:"provisional"`
		Rotulo       string    `json:"rotulo"`
		Actor        string    `json:"actor"`
		VigenteDesde time.Time `json:"vigente_desde"`
	} `json:"politicas"`
	LlamamientosEnCurso             map[string]int `json:"llamamientos_en_curso"`
	HistoricoLlamamientosDisponible bool           `json:"historico_llamamientos_disponible"`
}

func decodificarResumenNominal(crudo []byte) (ports.ResumenBolsasNominal, error) {
	var s resumenNominalSQL
	var vacio ports.ResumenBolsasNominal
	if json.Unmarshal(crudo, &s) != nil || s.GeneradoEn.IsZero() || len(s.Situaciones) > maximoFilasResumenBolsas ||
		s.LlamamientosEnCurso == nil || s.HistoricoLlamamientosDisponible {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	out := ports.ResumenBolsasNominal{GeneradoEn: s.GeneradoEn, Politicas: map[string]dominiobolsa.PoliticaOrdenBolsa{},
		LlamamientosEnCurso: s.LlamamientosEnCurso}
	for _, p := range s.Politicas {
		politica := dominiobolsa.PoliticaOrdenBolsa{BolsaRef: p.BolsaRef, PoliticaRef: p.PoliticaRef, Version: uint64(p.Version),
			Criterio: p.Criterio, TipoLista: p.TipoLista, Reposicion: p.Reposicion, Provisional: p.Provisional,
			Rotulo: p.Rotulo, Actor: p.Actor, VigenteDesde: p.VigenteDesde}
		if p.Version <= 0 || politica.Validar() != nil || out.Politicas[p.BolsaRef].PoliticaRef != "" {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		out.Politicas[p.BolsaRef] = politica
	}
	for _, f := range s.Situaciones {
		if f.BolsaRef == "" || f.CategoriaRef == "" || f.ParticipacionRef == "" || f.Version <= 0 || f.Orden <= 0 ||
			f.ConfirmadaEn.IsZero() || f.Situacion == nil || f.Desde == nil || f.Desde.IsZero() {
			return vacio, ports.ErrResumenBolsasNoDisponible
		}
		fila := ports.SituacionResumenParticipacion{BolsaRef: f.BolsaRef, CategoriaRef: f.CategoriaRef,
			ConfirmadaEn: f.ConfirmadaEn, InstantaneaRef: f.InstantaneaRef, VersionInstantanea: uint64(f.Version),
			Orden: uint64(f.Orden), ParticipacionRef: f.ParticipacionRef,
			Situacion: &ports.SituacionParticipacion{ParticipacionRef: f.ParticipacionRef, Situacion: *f.Situacion,
				Desde: *f.Desde, FechaDisponible: f.FechaDisponible}}
		if f.CeseFechaEfecto != nil {
			if f.CeseDisponibleDesde == nil || f.CeseEnRestriccion == nil || f.CeseTrabajoCesado == nil {
				return vacio, ports.ErrResumenBolsasNoDisponible
			}
			efecto, e1 := time.Parse("2006-01-02", *f.CeseFechaEfecto)
			disponible, e2 := time.Parse("2006-01-02", *f.CeseDisponibleDesde)
			if e1 != nil || e2 != nil {
				return vacio, ports.ErrResumenBolsasNoDisponible
			}
			cese, presente, e3 := validarEstadoCese(efecto, disponible, *f.CeseEnRestriccion, *f.CeseTrabajoCesado, s.GeneradoEn)
			if e3 != nil || !presente {
				return vacio, ports.ErrResumenBolsasNoDisponible
			}
			fila.Cese = &cese
		}
		out.Filas = append(out.Filas, fila)
	}
	if len(out.Politicas) != len(out.LlamamientosEnCurso) {
		return vacio, ports.ErrResumenBolsasNoDisponible
	}
	return out, nil
}
