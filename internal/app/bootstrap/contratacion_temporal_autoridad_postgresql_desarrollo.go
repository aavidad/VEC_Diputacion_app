package bootstrap

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	rolPropietarioContextoContratacionTemporalDesarrollo     = "vec_contexto_actor_v1_propietario"
	rolPropietarioAutorizacionContratacionTemporalDesarrollo = rolPropietarioAutorizacionPostgreSQLDesarrollo
	rolProyectorMotivosContratacionTemporalDesarrollo        = "vec_autorizacion_motivos_proyector"
)

func publicarAutoridadPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || soporte == nil ||
		soporte.contexto.Resultado.Validar() != nil ||
		soporte.instantanea.Validar() != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return err
	}
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := publicarMotivosPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := publicarMotivoEleccionProcedimientoRRHHPostgreSQL(ctx, pool, soporte.reloj.Ahora()); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func publicarContextoPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if soporte == nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return publicarResultadoContextoPostgreSQLDesarrollo(
		ctx, pool, soporte.contexto.Resultado,
		operacionContextoContratacionTemporalDesarrollo(soporte),
	)
}

func operacionContextoContratacionTemporalDesarrollo(
	soporte *soporteAltaContratacionTemporalDesarrollo,
) string {
	if soporte == nil {
		return ""
	}
	base := soporte.principalID + "\x00" + soporte.certificadoSHA256
	return referenciaAltaContratacionTemporalDesarrollo("oca_", base+"\x00registro-contexto")
}

// publicarResultadoContextoPostgreSQLDesarrollo materializa un contexto ya
// resuelto. La referencia de operación es de composición, nunca del cliente.
func publicarResultadoContextoPostgreSQLDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	resultado dominiovec.ResultadoContextoActorRegistradoV2,
	operacionRef string,
) error {
	if ctx == nil || pool == nil || operacionRef == "" || resultado.Validar() != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	actor := resultado.Contexto
	instantanea := actor.Instantanea
	manifiesto, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(
		resultado.ManifiestoProcedenciaCanonico,
	)
	if err != nil || manifiesto.ValidarParaContexto(actor) != nil ||
		len(instantanea.Vinculos) != 0 || len(manifiesto.Vinculos) != 0 {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	procedencia := manifiesto.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1
	if manifiesto.Persona.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia ||
		manifiesto.Perfil.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia ||
		manifiesto.Contexto.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioContextoContratacionTemporalDesarrollo); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if _, err = tx.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		"vec:ct:desarrollo:autoridad:"+procedencia.ProcedenciaRef,
	); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	consultas := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO vec_contexto_actor_v1.procedencias
		  (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
		 SELECT $1,$2,$3,$4 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.procedencias
		   WHERE procedencia_ref=$1 AND procedencia_version=$2)`,
			[]any{procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad)}},
		{`INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_versiones
		  (cuenta_ref,version,procedencia_ref,procedencia_version,
		   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
		   WHERE cuenta_ref=$1 AND version=$2)`,
			[]any{instantanea.CuentaRef, instantanea.CuentaVersion,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.proyeccion_cuenta_actual(cuenta_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual WHERE cuenta_ref=$1)`,
			[]any{instantanea.CuentaRef, instantanea.CuentaVersion}},
		{`INSERT INTO vec_contexto_actor_v1.persona_versiones
		  (persona_ref,version,procedencia_ref,procedencia_version,
		   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.persona_versiones
		   WHERE persona_ref=$1 AND version=$2)`,
			[]any{actor.PersonaRef, instantanea.PersonaVersion,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.persona_actual(persona_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.persona_actual WHERE persona_ref=$1)`,
			[]any{actor.PersonaRef, instantanea.PersonaVersion}},
		{`INSERT INTO vec_contexto_actor_v1.perfil_versiones
		  (perfil_ref,version,persona_ref,procedencia_ref,procedencia_version,
		   procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones
		   WHERE perfil_ref=$1 AND version=$2)`,
			[]any{actor.PerfilActivoRef, instantanea.PerfilVersion, actor.PersonaRef,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.perfil_actual(perfil_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.perfil_actual WHERE perfil_ref=$1)`,
			[]any{actor.PerfilActivoRef, instantanea.PerfilVersion}},
		{`INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
		  (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,
		   procedencia_version,procedencia_huella_sha256,procedencia_autoridad,
		   estado,vigente_desde,vigente_hasta)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones
		   WHERE vinculo_ref=$1 AND version=$2)`,
			[]any{instantanea.VinculoRef, instantanea.VinculoVersion,
				instantanea.CuentaRef, actor.PerfilActivoRef, actor.PersonaRef,
				procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
				procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
				string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta}},
		{`INSERT INTO vec_contexto_actor_v1.vinculo_contexto_actual(vinculo_ref,version)
		 SELECT $1,$2 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual WHERE vinculo_ref=$1)`,
			[]any{instantanea.VinculoRef, instantanea.VinculoVersion}},
		{`INSERT INTO vec_contexto_actor_v1.registros_contexto
		  (operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,
		   solicitado_en,resuelto_en,representacion_canonica,huella_sha256,
		   manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,
		   autoridad_efectiva)
		 SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13 WHERE NOT EXISTS (
		  SELECT 1 FROM vec_contexto_actor_v1.registros_contexto
		   WHERE operacion_ref=$1 OR registro_contexto_ref=$2)`,
			[]any{operacionRef, resultado.RegistroContextoRef, instantanea.CuentaRef,
				actor.PerfilActivoRef, string(actor.Principal.AuthMethod),
				string(actor.Principal.AuthAssurance), resultado.ResueltoEnAutoritativo,
				resultado.ResueltoEnAutoritativo, resultado.RepresentacionCanonica,
				resultado.HuellaSHA256, resultado.ManifiestoProcedenciaCanonico,
				resultado.ManifiestoProcedenciaHuellaSHA256,
				string(resultado.AutoridadEfectiva)}},
	}
	for _, consulta := range consultas {
		if _, err = tx.Exec(ctx, consulta.sql, consulta.args...); err != nil {
			return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
		}
	}
	var coincide bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.registros_contexto
		  WHERE operacion_ref=$1 AND registro_contexto_ref=$2 AND cuenta_ref=$3
		    AND perfil_ref=$4 AND metodo=$5 AND garantia=$6
		    AND solicitado_en=$7 AND resuelto_en=$7
		    AND representacion_canonica=$8 AND huella_sha256=$9
		    AND manifiesto_procedencia_canonico=$10
			 AND manifiesto_procedencia_huella_sha256=$11
			 AND autoridad_efectiva=$12)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.procedencias
		  WHERE procedencia_ref=$19 AND procedencia_version=$20
		    AND procedencia_huella_sha256=$21 AND procedencia_autoridad=$22)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones
		  WHERE cuenta_ref=$3 AND version=$13 AND procedencia_ref=$19
		    AND procedencia_version=$20 AND procedencia_huella_sha256=$21
		    AND procedencia_autoridad=$22 AND estado=$23
		    AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.persona_versiones
		  WHERE persona_ref=$14 AND version=$15 AND procedencia_ref=$19
		    AND procedencia_version=$20 AND procedencia_huella_sha256=$21
		    AND procedencia_autoridad=$22 AND estado=$23
		    AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones
		  WHERE perfil_ref=$4 AND version=$16 AND persona_ref=$14
		    AND procedencia_ref=$19 AND procedencia_version=$20
		    AND procedencia_huella_sha256=$21 AND procedencia_autoridad=$22
		    AND estado=$23 AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones
		  WHERE vinculo_ref=$17 AND version=$18 AND cuenta_ref=$3
		    AND perfil_ref=$4 AND persona_ref=$14 AND procedencia_ref=$19
		    AND procedencia_version=$20 AND procedencia_huella_sha256=$21
		    AND procedencia_autoridad=$22 AND estado=$23
		    AND vigente_desde=$24 AND vigente_hasta=$25)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.proyeccion_cuenta_actual
		  WHERE cuenta_ref=$3 AND version=$13)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.persona_actual
		  WHERE persona_ref=$14 AND version=$15)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.perfil_actual
		  WHERE perfil_ref=$4 AND version=$16)
		AND EXISTS (
		 SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual
		  WHERE vinculo_ref=$17 AND version=$18)`,
		operacionRef, resultado.RegistroContextoRef, instantanea.CuentaRef,
		actor.PerfilActivoRef, string(actor.Principal.AuthMethod),
		string(actor.Principal.AuthAssurance), resultado.ResueltoEnAutoritativo,
		resultado.RepresentacionCanonica, resultado.HuellaSHA256,
		resultado.ManifiestoProcedenciaCanonico,
		resultado.ManifiestoProcedenciaHuellaSHA256,
		string(resultado.AutoridadEfectiva), instantanea.CuentaVersion,
		actor.PersonaRef, instantanea.PersonaVersion, instantanea.PerfilVersion,
		instantanea.VinculoRef, instantanea.VinculoVersion,
		procedencia.ProcedenciaRef, procedencia.ProcedenciaVersion,
		procedencia.ProcedenciaHuellaSHA256, string(procedencia.ProcedenciaAutoridad),
		string(instantanea.Estado), instantanea.VigenteDesde, instantanea.VigenteHasta,
	).Scan(&coincide)
	if err != nil || !coincide {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	autoridad := &autoridadPostgreSQLContratacionTemporalDesarrollo{
		pool: pool, soporte: soporte,
	}
	instantanea, err := autoridad.prepararInstantanea(
		ctx, soporte.instantanea, true,
	)
	if err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	if err := autoridad.publicarInstantanea(ctx, instantanea); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	soporte.mu.Lock()
	soporte.instantanea = instantanea
	soporte.mu.Unlock()
	return nil
}

type autoridadPostgreSQLContratacionTemporalDesarrollo struct {
	pool    *pgxpool.Pool
	soporte *soporteAltaContratacionTemporalDesarrollo
}

// Los lectores CT existentes conservan estos nombres; la implementación es
// común y no abre ninguna autoridad adicional.
type asignacionActualPostgreSQLContratacionTemporalDesarrollo = asignacionActualPostgreSQLDesarrollo

func leerAsignacionActualPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	consultador interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	},
	perfilRef string,
) (asignacionActualPostgreSQLContratacionTemporalDesarrollo, bool, error) {
	return leerAsignacionActualPostgreSQLDesarrollo(ctx, consultador, perfilRef)
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) PrepararInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	return a.prepararInstantanea(ctx, instantanea, false)
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) PublicarInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	return a.publicarInstantanea(ctx, instantanea)
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) prepararInstantanea(
	ctx context.Context,
	solicitada dominiovec.InstantaneaAutorizacion,
	permitirInicial bool,
) (dominiovec.InstantaneaAutorizacion, error) {
	preparada, err := a.autoridadComun().prepararInstantanea(ctx, solicitada, permitirInicial)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return preparada, nil
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) publicarInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	if err := a.autoridadComun().publicarInstantanea(ctx, instantanea); err != nil {
		return errPostgreSQLContratacionTemporalDesarrolloNoDisponible
	}
	return nil
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) autoridadComun() autoridadPostgreSQLDesarrollo {
	if a == nil || a.soporte == nil {
		return autoridadPostgreSQLDesarrollo{}
	}
	return autoridadPostgreSQLDesarrollo{
		pool:           a.pool,
		vinculo:        a.soporte.contexto.Vinculo,
		prefijoBloqueo: "vec:ct:desarrollo:autorizacion:",
		actoControlRol: "acto:ct:desarrollo:control-rol:v1",
		actoAsignacion: "acto:ct:desarrollo:asignacion:v1",
		actoSesion:     "acto:ct:desarrollo:sesion:v1",
	}
}
