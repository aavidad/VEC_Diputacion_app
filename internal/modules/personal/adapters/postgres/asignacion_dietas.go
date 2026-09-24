package postgres

import (
	"context"
	"errors"
	"math"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
)

const ajustesAsignacionDietas = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),set_config('idle_in_transaction_session_timeout','20s',true)`

const columnasAsignacionDietas = `recibo_ref,decision_ref,efecto_ref,consumo_huella_sha256,auditoria_ad3_ref,registrada_en,estado_local,asignacion_ref,relacion_ref,persona_ref,unidad_ref,centro_ref,administrativo_persona_ref,responsable_persona_ref,grupo_dieta,vigente_desde::text,version`

const consultaAsignacionDietasSQL = `SELECT ` + columnasAsignacionDietas + ` FROM vec_personal.consultar_asignacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const registrarInicialAsignacionDietasSQL = `SELECT ` + columnasAsignacionDietas + ` FROM vec_personal.registrar_asignacion_dietas_inicial_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const corregirAsignacionDietasSQL = `SELECT ` + columnasAsignacionDietas + ` FROM vec_personal.corregir_asignacion_dietas_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`
const corregirGrupoDietasSQL = `SELECT ` + columnasAsignacionDietas + ` FROM vec_personal.corregir_grupo_dieta_v1($1,$2,$3,$4,$5,$6::numeric,$7::numeric,$8,$9,$10,$11)`

type RepositorioAsignacionDietasPostgreSQL struct{ pool iniciadorConsultaRelacion }

var _ personalports.RepositorioAsignacionDietas = (*RepositorioAsignacionDietasPostgreSQL)(nil)

func NuevoRepositorioAsignacionDietasPostgreSQL(pool *pgxpool.Pool) (*RepositorioAsignacionDietasPostgreSQL, error) {
	return nuevoRepositorioAsignacionDietasPostgreSQL(pool)
}

func nuevoRepositorioAsignacionDietasPostgreSQL(pool iniciadorConsultaRelacion) (*RepositorioAsignacionDietasPostgreSQL, error) {
	if nuloRelacion(pool) {
		return nil, personalports.ErrAsignacionDietasNoDisponible
	}
	return &RepositorioAsignacionDietasPostgreSQL{pool: pool}, nil
}

func (r *RepositorioAsignacionDietasPostgreSQL) EjecutarAsignacionDietas(ctx context.Context, orden personalports.OrdenAsignacionDietas) (personalports.ResultadoAsignacionDietas, error) {
	var vacio personalports.ResultadoAsignacionDietas
	if r == nil || ctx == nil || nuloRelacion(r.pool) {
		return vacio, personalports.ErrAsignacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	material := orden.Material.Canonico()
	solicitud := orden.Material.Solicitud()
	if len(material) == 0 || orden.Autorizacion.ValidarEstructura() != nil || orden.Autorizacion.PersonaVersion() > math.MaxInt64 || orden.Autorizacion.PerfilVersion() > math.MaxInt64 {
		return vacio, personalports.ErrSolicitudAsignacionDietasInvalida
	}
	consulta := ""
	switch solicitud.Operacion {
	case personaldomain.ConsultarAsignacionDietas:
		consulta = consultaAsignacionDietasSQL
	case personaldomain.RegistrarInicialAsignacionDietas:
		consulta = registrarInicialAsignacionDietasSQL
	case personaldomain.CorregirAsignacionDietas:
		consulta = corregirAsignacionDietasSQL
	case personaldomain.CorregirGrupoAsignacionDietas:
		consulta = corregirGrupoDietasSQL
	default:
		return vacio, personalports.ErrSolicitudAsignacionDietasInvalida
	}
	parametros, secretos, err := parametrosConsultaRelacionPropia(material, orden.Autorizacion)
	if err != nil {
		return vacio, personalports.ErrSolicitudAsignacionDietasInvalida
	}
	defer borrarRelacion(secretos[:])
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, normalizarErrorAsignacionDietas(ctx, err)
	}
	if tx == nil {
		return vacio, personalports.ErrAsignacionDietasNoDisponible
	}
	confirmada := false
	defer func() {
		if !confirmada {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err = tx.Exec(ctx, ajustesAsignacionDietas); err != nil {
		return vacio, normalizarErrorAsignacionDietas(ctx, err)
	}
	var rpta personalports.ResultadoAsignacionDietas
	var vigenteDesde string
	err = tx.QueryRow(ctx, consulta, parametros...).Scan(
		&rpta.ReciboRef, &rpta.DecisionRef, &rpta.EfectoRef, &rpta.ConsumoHuellaSHA256, &rpta.AuditoriaRef,
		&rpta.RegistradaEn, &rpta.EstadoLocal, &rpta.Asignacion.AsignacionRef, &rpta.Asignacion.RelacionRef,
		&rpta.Asignacion.PersonaRef, &rpta.Asignacion.UnidadRef, &rpta.Asignacion.CentroRef,
		&rpta.Asignacion.AdministrativoPersonaRef, &rpta.Asignacion.ResponsablePersonaRef,
		&rpta.Asignacion.GrupoDieta, &vigenteDesde, &rpta.Asignacion.Version,
	)
	if err != nil {
		return vacio, normalizarErrorAsignacionDietas(ctx, err)
	}
	rpta.RegistradaEn = rpta.RegistradaEn.UTC()
	rpta.Asignacion.VigenteDesde, err = personaldomain.NuevaFechaCivil(vigenteDesde)
	if err != nil || !resultadoAsignacionSQLValido(rpta, solicitud, orden) {
		return vacio, personalports.ErrAsignacionDietasNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, normalizarErrorAsignacionDietas(ctx, err)
	}
	confirmada = true
	return rpta, nil
}

var reciboAsignacionPostgres = regexp.MustCompile(`^rad_[0-9a-f]{32}$`)
var huellaAsignacionPostgres = regexp.MustCompile(`^[0-9a-f]{64}$`)

func resultadoAsignacionSQLValido(r personalports.ResultadoAsignacionDietas, s personaldomain.SolicitudAsignacionDietas, orden personalports.OrdenAsignacionDietas) bool {
	resumen := orden.Autorizacion.ResumenCapacidad()
	_, offset := r.RegistradaEn.Zone()
	if r.Asignacion.Validar() != nil || r.Asignacion.RelacionRef != s.RelacionRef || r.Asignacion.PersonaRef != s.PersonaRef || r.Asignacion.UnidadRef != s.UnidadRef || !r.Asignacion.VigenteEn(s.FechaReferencia) ||
		!reciboAsignacionPostgres.MatchString(r.ReciboRef) || !huellaAsignacionPostgres.MatchString(r.ConsumoHuellaSHA256) || r.DecisionRef != resumen.DecisionRef() || r.EfectoRef != resumen.EfectoRef() || r.AuditoriaRef == "" ||
		r.RegistradaEn.IsZero() || offset != 0 || r.RegistradaEn.Nanosecond()%1000 != 0 {
		return false
	}
	if s.Operacion == personaldomain.ConsultarAsignacionDietas {
		return r.EstadoLocal == "consultada" && !r.RegistradaEn.Before(resumen.EmitidaEn()) && r.RegistradaEn.Before(resumen.ExpiraEn())
	}
	if r.EstadoLocal != "registrada" && r.EstadoLocal != "replay_confirmado" {
		return false
	}
	if r.EstadoLocal == "registrada" && (r.RegistradaEn.Before(resumen.EmitidaEn()) || !r.RegistradaEn.Before(resumen.ExpiraEn())) {
		return false
	}
	return r.Asignacion.Version == s.VersionEsperada+1 && r.Asignacion.CentroRef == s.CentroRef && r.Asignacion.AdministrativoPersonaRef == s.AdministrativoPersonaRef && r.Asignacion.ResponsablePersonaRef == s.ResponsablePersonaRef && r.Asignacion.GrupoDieta == s.GrupoDieta && r.Asignacion.VigenteDesde == s.VigenteDesde
}

func normalizarErrorAsignacionDietas(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "P7201":
			return personalports.ErrAsignacionDietasDenegada
		case "P7203":
			return personalports.ErrVersionAsignacionDietas
		case "P7204":
			return personalports.ErrClaveAsignacionDietas
		case "42501":
			return personalports.ErrAsignacionDietasDenegada
		}
	}
	return personalports.ErrAsignacionDietasNoDisponible
}
