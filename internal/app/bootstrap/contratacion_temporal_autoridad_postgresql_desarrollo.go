package bootstrap

import (
	"context"
	"encoding/json"
	"log/slog"
	"reflect"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	rolPropietarioContextoContratacionTemporalDesarrollo     = "vec_contexto_actor_v1_propietario"
	rolPropietarioAutorizacionContratacionTemporalDesarrollo = rolPropietarioAutorizacionPostgreSQLDesarrollo
	rolProyectorMotivosContratacionTemporalDesarrollo        = "vec_autorizacion_motivos_proyector"
	actoControlRolReincorporacionTitularDesarrollo           = "acto:ct:reincorporacion-titular:control-rol:v1"
	actoAsignacionReincorporacionTitularDesarrollo           = "acto:ct:reincorporacion-titular:asignacion:v1"
	actoSesionReincorporacionTitularDesarrollo               = "acto:ct:reincorporacion-titular:sesion:v1"
)

func publicarAutoridadPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if ctx == nil || pool == nil || soporte == nil ||
		soporte.contexto.Resultado.Validar() != nil ||
		soporte.instantanea.Validar() != nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	if err := publicarContextoPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return err
	}
	if err := publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	if err := publicarMotivosPostgreSQLContratacionTemporalDesarrollo(
		ctx, pool, soporte,
	); err != nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	if err := publicarMotivoEleccionProcedimientoRRHHPostgreSQL(ctx, pool, soporte.reloj.Ahora()); err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	return nil
}

func publicarContextoPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if soporte == nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	return publicarResultadoContextoPostgreSQLDesarrollo(
		ctx, pool, soporte.contexto.Resultado,
		operacionContextoContratacionTemporalDesarrollo(soporte),
	)
}

// El perfil CT130 usa una sesión/contexto propio. Su operación de registro
// incluye el perfil para que no colisione con el contexto base de la persona.
func publicarContextoPostgreSQLReincorporacionTitularDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	if soporte == nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	contexto, err := soporte.contextoReincorporacionTitular(ctx)
	if err != nil || contexto.Resultado.Validar() != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	datos, err := contexto.Vinculo.Datos()
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	operacion := referenciaAltaContratacionTemporalDesarrollo("oca_", datos.PrincipalID+"\x00"+datos.PerfilActivoRef+"\x00registro-contexto-ct130")
	return publicarResultadoContextoPostgreSQLDesarrollo(ctx, pool, contexto.Resultado, operacion)
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
		return falloPostgreSQLCTDesarrollo(nil)
	}
	actor := resultado.Contexto
	instantanea := actor.Instantanea
	manifiesto, err := dominiovec.RehidratarManifiestoProcedenciaContextoActorV1(
		resultado.ManifiestoProcedenciaCanonico,
	)
	if err != nil || manifiesto.ValidarParaContexto(actor) != nil ||
		len(instantanea.Vinculos) != 0 || len(manifiesto.Vinculos) != 0 {
		return falloPostgreSQLCTDesarrollo(err)
	}
	procedencia := manifiesto.Cuenta.AcreditacionProcedenciaComponenteContextoActorV1
	if manifiesto.Persona.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia ||
		manifiesto.Perfil.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia ||
		manifiesto.Contexto.AcreditacionProcedenciaComponenteContextoActorV1 != procedencia {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+
		rolPropietarioContextoContratacionTemporalDesarrollo); err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	if _, err = tx.Exec(ctx, `
		SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended($1,0))`,
		"vec:ct:desarrollo:autoridad:"+procedencia.ProcedenciaRef,
	); err != nil {
		return falloPostgreSQLCTDesarrollo(nil)
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
			return falloPostgreSQLCTDesarrollo(err)
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
		return falloPostgreSQLCTDesarrollo(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	return nil
}

// publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo es el paso de
// arranque del perfil dinámico (RRHH, intervención). Antes preparaba y
// publicaba siempre la semilla, lo que devolvía a activo un permiso revocado o
// restringido. Ahora solo lee: publica la semilla únicamente si el perfil no
// tiene asignación (publicación inicial transaccional) y, si la tiene, no
// escribe nada. Si la vigente no es operativa (revocada, restringida por otro
// acto, caducada o con su rol retirado) lo deja escrito en el registro y el
// arranque sigue: la guarda de origen operativo deniega sus peticiones.
func publicarAutorizacionPostgreSQLContratacionTemporalDesarrollo(
	ctx context.Context,
	pool *pgxpool.Pool,
	soporte *soporteAltaContratacionTemporalDesarrollo,
) error {
	estado, err := asegurarPerfilDinamicoCTDesarrollo(ctx, pool, soporte)
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	if estado == perfilDinamicoPendienteProvision {
		slog.Warn("perfil de Contratación temporal sin asignación operativa: sus peticiones se deniegan hasta la provisión",
			"perfil_ref", soporte.instantanea.AsignacionPerfil.PerfilActivoRef,
			"estado", string(perfilDinamicoPendienteProvision))
	}
	return nil
}

// estadoPerfilDinamicoCTDesarrollo resume, sin datos personales, qué hizo el
// arranque con un perfil dinámico de Contratación temporal.
type estadoPerfilDinamicoCTDesarrollo string

const (
	perfilDinamicoPublicadoInicial   estadoPerfilDinamicoCTDesarrollo = "publicado_inicial"
	perfilDinamicoOperativo          estadoPerfilDinamicoCTDesarrollo = "operativo"
	perfilDinamicoPendienteProvision estadoPerfilDinamicoCTDesarrollo = "pendiente_provision"
)

func asegurarPerfilDinamicoCTDesarrollo(
	ctx context.Context, pool *pgxpool.Pool, soporte *soporteAltaContratacionTemporalDesarrollo,
) (estadoPerfilDinamicoCTDesarrollo, error) {
	if ctx == nil || pool == nil || soporte == nil ||
		dependenciaEsNulaContratacionTemporalDesarrollo(soporte.reloj) {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	soporte.mu.Lock()
	plantilla := clonarInstantaneaAutorizacionAltaContratacionTemporalDesarrollo(soporte.instantanea)
	soporte.mu.Unlock()
	if plantilla.Validar() != nil {
		return "", falloPostgreSQLCTDesarrollo(nil)
	}
	autoridad := &autoridadPostgreSQLContratacionTemporalDesarrollo{pool: pool, soporte: soporte}
	comun := autoridad.autoridadComun()
	publicada, encontrada, err := leerInstantaneaPublicadaPostgreSQLDesarrollo(
		ctx, pool, plantilla.AsignacionPerfil.PerfilActivoRef,
	)
	if !encontrada {
		if err != nil {
			return "", err
		}
		comun.soloInicial = true
		preparada, err := comun.prepararInstantanea(ctx, plantilla, true)
		if err != nil || preparada.AsignacionPerfil.Version != 1 {
			return "", falloPostgreSQLCTDesarrollo(err)
		}
		if err := comun.publicarInstantanea(ctx, preparada); err != nil {
			return "", err
		}
		soporte.mu.Lock()
		soporte.instantanea = preparada
		soporte.mu.Unlock()
		return perfilDinamicoPublicadoInicial, nil
	}
	// Un documento ilegible no es operativo, pero tampoco se toca.
	if err != nil || publicada.instantanea.AsignacionPerfil.PrincipalID != plantilla.AsignacionPerfil.PrincipalID ||
		!origenOperativoPublicadoCTDesarrollo(publicada, comun.actoAsignacion, soporte.reloj.Ahora()) {
		return perfilDinamicoPendienteProvision, nil
	}
	return perfilDinamicoOperativo, nil
}

// origenOperativoPublicadoCTDesarrollo es, sobre la lectura sin bloqueo del
// arranque, el mismo criterio que comprobarOrigenOperativo aplica bajo
// bloqueo en cada preparación y publicación: activa, en vigor, con el control
// de su rol habilitado y con el puntero movido por el propio circuito.
func origenOperativoPublicadoCTDesarrollo(
	publicada instantaneaPublicadaDesarrollo, actoAsignacion string, ahora time.Time,
) bool {
	p := publicada.instantanea
	return actoAsignacion != "" && publicada.actoAsignacion == actoAsignacion &&
		p.Validar() == nil && publicada.actualizadaPor == p.AsignacionPerfil.EmitidaPor &&
		p.AsignacionPerfil.Estado == dominiovec.EstadoAsignacionPerfilActiva &&
		p.AsignacionPerfil.VigenteEn(ahora) &&
		p.ControlVigenciaVersionRol.Estado == dominiovec.EstadoControlVigenciaVersionRolHabilitada
}

type autoridadPostgreSQLContratacionTemporalDesarrollo struct {
	pool    *pgxpool.Pool
	soporte *soporteAltaContratacionTemporalDesarrollo
}

type autoridadReincorporacionTitularPostgreSQL interface {
	PrepararInstantaneaReincorporacionTitular(context.Context, dominiovec.InstantaneaAutorizacion) (dominiovec.InstantaneaAutorizacion, error)
	PublicarInstantaneaReincorporacionTitular(context.Context, dominiovec.InstantaneaAutorizacion) error
}

// El perfil CT130 dedicado sólo admite una publicación inicial ausente o el
// replay exacto de su propia instantánea central.
func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) PrepararInstantaneaReincorporacionTitular(
	ctx context.Context, solicitada dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	if a == nil || a.soporte == nil || a.soporte.reincorporacionTitular == nil ||
		ctx == nil || ctx.Err() != nil || solicitada.Validar() != nil {
		return vacia, falloPostgreSQLCTDesarrollo(nil)
	}
	semilla := a.soporte.reincorporacionTitular.instantanea
	esperada := clonarInstantaneaAutorizacionPostgreSQLDesarrollo(semilla)
	esperada.AsignacionPerfil.Ambitos = solicitada.AsignacionPerfil.Ambitos
	if !reflect.DeepEqual(solicitada, esperada) {
		return vacia, falloPostgreSQLCTDesarrollo(nil)
	}
	comun, err := a.autoridadReincorporacionTitular(ctx)
	if err != nil {
		return vacia, falloPostgreSQLCTDesarrollo(err)
	}
	preparada, err := comun.prepararInstantanea(ctx, solicitada, true)
	if err != nil {
		return vacia, falloPostgreSQLCTDesarrollo(err)
	}
	coincide, encontrada, err := instantaneaCentralCTExacta(ctx, a.pool, preparada)
	if err != nil {
		return vacia, falloPostgreSQLCTDesarrollo(err)
	}
	if encontrada && !coincide {
		if _, aprobada, err := a.preimagenReincorporacionTitularAcreditada(ctx, semilla); err != nil || !aprobada {
			return vacia, falloPostgreSQLCTDesarrollo(err)
		}
	}
	return preparada, nil
}

// Compara la preimagen efectiva bajo la autoridad central. La sesión puede
// seguir activa mientras la asignación o el control de rol dejan de estarlo.
func instantaneaCentralCTExacta(ctx context.Context, pool *pgxpool.Pool, i dominiovec.InstantaneaAutorizacion) (bool, bool, error) {
	if ctx == nil || ctx.Err() != nil || pool == nil || i.Validar() != nil || len(i.Politicas) != 0 {
		return false, false, falloPostgreSQLCTDesarrollo(nil)
	}
	if i.AsignacionPerfil.Estado != dominiovec.EstadoAsignacionPerfilActiva ||
		i.ControlVigenciaVersionRol.Estado != dominiovec.EstadoControlVigenciaVersionRolHabilitada {
		return false, false, falloPostgreSQLCTDesarrollo(nil)
	}
	huellaAsignacion, err := i.AsignacionPerfil.HuellaSHA256()
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	huellaRol, err := i.VersionRol.HuellaSHA256()
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	huellaControl, err := i.ControlVigenciaVersionRol.HuellaSHA256()
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	documentoAsignacion, err := json.Marshal(i.AsignacionPerfil)
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	documentoRol, err := json.Marshal(i.VersionRol)
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	documentoControl, err := json.Marshal(i.ControlVigenciaVersionRol)
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	actual, encontrada, err := leerAsignacionActualPostgreSQLDesarrollo(ctx, tx, i.AsignacionPerfil.PerfilActivoRef)
	if err != nil {
		return false, false, falloPostgreSQLCTDesarrollo(err)
	}
	if !encontrada {
		if err = tx.Commit(ctx); err != nil {
			return false, false, falloPostgreSQLCTDesarrollo(err)
		}
		return false, false, nil
	}
	if actual.referencia != i.AsignacionPerfil.Referencia() ||
		actual.identificador != i.AsignacionPerfil.AsignacionID ||
		actual.version != int64(i.AsignacionPerfil.Version) ||
		actual.perfilRef != i.AsignacionPerfil.PerfilActivoRef ||
		actual.principalID != i.AsignacionPerfil.PrincipalID ||
		actual.versionRolRef != i.VersionRol.Referencia() ||
		actual.huella != huellaAsignacion {
		return false, true, nil
	}
	var exacta bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		 SELECT 1 FROM vec_autorizacion.asignacion_perfil AS asignacion
		 JOIN vec_autorizacion.version_rol AS rol
		   ON rol.version_rol_ref=asignacion.version_rol_ref
		 JOIN vec_autorizacion.control_vigencia_version_rol_actual AS vigente
		   ON vigente.version_rol_ref=rol.version_rol_ref
		 JOIN vec_autorizacion.control_vigencia_version_rol AS control
		   ON control.version_rol_ref=vigente.version_rol_ref AND control.revision=vigente.revision
		 WHERE asignacion.asignacion_ref=$1 AND asignacion.huella_sha256=$2
		   AND asignacion.documento=$3::jsonb
		   AND rol.version_rol_ref=$4 AND rol.huella_sha256=$5 AND rol.documento=$6::jsonb
		   AND control.revision=$7 AND control.estado='habilitada'
		   AND control.huella_sha256=$8 AND control.documento=$9::jsonb)
		AND EXISTS (
		 SELECT 1 FROM vec_autorizacion.control_catalogo_politicas
		 WHERE control_id=true AND revision=$10 AND huella_sha256=$11)`,
		i.AsignacionPerfil.Referencia(), huellaAsignacion, documentoAsignacion,
		i.VersionRol.Referencia(), huellaRol, documentoRol,
		i.ControlVigenciaVersionRol.Revision, huellaControl, documentoControl,
		i.RevisionCatalogoPoliticas, i.CatalogoPoliticasHuellaSHA256,
	).Scan(&exacta)
	if err != nil {
		return false, true, falloPostgreSQLCTDesarrollo(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, true, falloPostgreSQLCTDesarrollo(err)
	}
	return exacta, true, nil
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

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) PublicarInstantaneaReincorporacionTitular(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	if a == nil || a.soporte == nil || a.soporte.reincorporacionTitular == nil ||
		ctx == nil || ctx.Err() != nil || instantanea.Validar() != nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	semilla := a.soporte.reincorporacionTitular.instantanea
	if instantanea.VersionRol.RolID != semilla.VersionRol.RolID ||
		!reflect.DeepEqual(instantanea.VersionRol.Concesiones, semilla.VersionRol.Concesiones) ||
		instantanea.AsignacionPerfil.PrincipalID != semilla.AsignacionPerfil.PrincipalID ||
		instantanea.AsignacionPerfil.PerfilActivoRef != semilla.AsignacionPerfil.PerfilActivoRef {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	exacta, encontrada, err := instantaneaCentralCTExacta(ctx, a.pool, instantanea)
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	comun, err := a.autoridadReincorporacionTitular(ctx)
	if err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	if !encontrada {
		comun.soloInicial = true
		return comun.publicarInstantanea(ctx, instantanea)
	}
	if exacta {
		return comun.publicarInstantaneaDesdePreimagen(ctx, instantanea, instantanea)
	}
	preimagen, aprobada, err := a.preimagenReincorporacionTitularAcreditada(ctx, semilla)
	if err != nil || !aprobada {
		return falloPostgreSQLCTDesarrollo(err)
	}
	return comun.publicarInstantaneaDesdePreimagen(ctx, instantanea, preimagen)
}

// Solo una asignación CT130 publicada por este acto puede servir como
// preimagen para otro expediente. Una restricción administrativa conserva su
// propia procedencia y no se transforma en una ampliación por la semilla.
func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) preimagenReincorporacionTitularAcreditada(
	ctx context.Context, semilla dominiovec.InstantaneaAutorizacion,
) (dominiovec.InstantaneaAutorizacion, bool, error) {
	vacia := dominiovec.InstantaneaAutorizacion{}
	if a == nil || a.pool == nil || a.soporte == nil || ctx == nil || ctx.Err() != nil || semilla.Validar() != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(nil)
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SET LOCAL ROLE `+rolPropietarioAutorizacionPostgreSQLDesarrollo); err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	actual, encontrada, err := leerAsignacionActualPostgreSQLDesarrollo(ctx, tx, semilla.AsignacionPerfil.PerfilActivoRef)
	if err != nil || !encontrada {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	var documentoAsignacion, documentoRol, documentoControl []byte
	var huellaAsignacion, huellaRol, huellaControl string
	var revisionCatalogoTexto, huellaCatalogo, actoAsignacion, actualizadaPor, actoControl string
	err = tx.QueryRow(ctx, `
		SELECT asignacion.documento, rol.documento, control.documento,
		       asignacion.huella_sha256, rol.huella_sha256, control.huella_sha256,
		       catalogo.revision::text, catalogo.huella_sha256,
		       vigente.acto_ref, vigente.actualizada_por, control_actual.acto_ref
		  FROM vec_autorizacion.asignacion_perfil_actual AS vigente
		  JOIN vec_autorizacion.asignacion_perfil AS asignacion
		    ON asignacion.perfil_activo_ref=vigente.perfil_activo_ref
		   AND asignacion.asignacion_ref=vigente.asignacion_ref
		  JOIN vec_autorizacion.version_rol AS rol
		    ON rol.version_rol_ref=asignacion.version_rol_ref
		  JOIN vec_autorizacion.control_vigencia_version_rol_actual AS control_actual
		    ON control_actual.version_rol_ref=rol.version_rol_ref
		  JOIN vec_autorizacion.control_vigencia_version_rol AS control
		    ON control.version_rol_ref=control_actual.version_rol_ref
		   AND control.revision=control_actual.revision
		  JOIN vec_autorizacion.control_catalogo_politicas AS catalogo ON catalogo.control_id=true
		 WHERE vigente.perfil_activo_ref=$1 AND vigente.asignacion_ref=$2
		 FOR SHARE OF asignacion, rol, control, control_actual, catalogo`,
		semilla.AsignacionPerfil.PerfilActivoRef, actual.referencia,
	).Scan(&documentoAsignacion, &documentoRol, &documentoControl,
		&huellaAsignacion, &huellaRol, &huellaControl,
		&revisionCatalogoTexto, &huellaCatalogo, &actoAsignacion, &actualizadaPor, &actoControl)
	if err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	revisionCatalogo, err := strconv.ParseUint(revisionCatalogoTexto, 10, 64)
	if err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	var preimagen dominiovec.InstantaneaAutorizacion
	if json.Unmarshal(documentoAsignacion, &preimagen.AsignacionPerfil) != nil ||
		json.Unmarshal(documentoRol, &preimagen.VersionRol) != nil ||
		json.Unmarshal(documentoControl, &preimagen.ControlVigenciaVersionRol) != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(nil)
	}
	preimagen.RevisionCatalogoPoliticas = revisionCatalogo
	preimagen.CatalogoPoliticasHuellaSHA256 = huellaCatalogo
	huellaAsignacionCalculada, errAsignacion := preimagen.AsignacionPerfil.HuellaSHA256()
	huellaRolCalculada, errRol := preimagen.VersionRol.HuellaSHA256()
	huellaControlCalculada, errControl := preimagen.ControlVigenciaVersionRol.HuellaSHA256()
	aprobada := preimagen.Validar() == nil && errAsignacion == nil && errRol == nil && errControl == nil &&
		preimagen.AsignacionPerfil.Referencia() == actual.referencia &&
		preimagen.AsignacionPerfil.AsignacionID == actual.identificador &&
		int64(preimagen.AsignacionPerfil.Version) == actual.version &&
		preimagen.AsignacionPerfil.PerfilActivoRef == actual.perfilRef &&
		preimagen.AsignacionPerfil.PrincipalID == actual.principalID &&
		preimagen.AsignacionPerfil.VersionRolRef == actual.versionRolRef &&
		huellaAsignacionCalculada == huellaAsignacion && huellaAsignacion == actual.huella &&
		huellaRolCalculada == huellaRol && huellaControlCalculada == huellaControl &&
		preimagen.AsignacionPerfil.Estado == dominiovec.EstadoAsignacionPerfilActiva &&
		preimagen.AsignacionPerfil.VigenteEn(a.soporte.reloj.Ahora()) &&
		preimagen.ControlVigenciaVersionRol.Estado == dominiovec.EstadoControlVigenciaVersionRolHabilitada &&
		preimagen.AsignacionPerfil.AsignacionID == semilla.AsignacionPerfil.AsignacionID &&
		preimagen.AsignacionPerfil.PrincipalID == semilla.AsignacionPerfil.PrincipalID &&
		preimagen.AsignacionPerfil.PerfilActivoRef == semilla.AsignacionPerfil.PerfilActivoRef &&
		preimagen.VersionRol.RolID == semilla.VersionRol.RolID &&
		reflect.DeepEqual(preimagen.VersionRol.Concesiones, semilla.VersionRol.Concesiones) &&
		preimagen.RevisionCatalogoPoliticas == semilla.RevisionCatalogoPoliticas &&
		preimagen.CatalogoPoliticasHuellaSHA256 == semilla.CatalogoPoliticasHuellaSHA256 &&
		actoAsignacion == actoAsignacionReincorporacionTitularDesarrollo &&
		actoControl == actoControlRolReincorporacionTitularDesarrollo &&
		actualizadaPor == preimagen.AsignacionPerfil.EmitidaPor
	if err = tx.Commit(ctx); err != nil {
		return vacia, false, falloPostgreSQLCTDesarrollo(err)
	}
	if !aprobada {
		return vacia, false, nil
	}
	return preimagen, true, nil
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) autoridadReincorporacionTitular(
	ctx context.Context,
) (autoridadPostgreSQLDesarrollo, error) {
	if a == nil || a.soporte == nil {
		return autoridadPostgreSQLDesarrollo{}, falloPostgreSQLCTDesarrollo(nil)
	}
	contexto, err := a.soporte.contextoOperativoDesarrollo(ctx)
	semilla := a.soporte.reincorporacionTitular.instantanea
	if err != nil || contexto.Resultado.Validar() != nil ||
		contexto.Resultado.Contexto.PerfilActivoRef != semilla.AsignacionPerfil.PerfilActivoRef {
		return autoridadPostgreSQLDesarrollo{}, falloPostgreSQLCTDesarrollo(err)
	}
	comun := a.autoridadComun()
	comun.vinculo = contexto.Vinculo
	comun.actoControlRol = actoControlRolReincorporacionTitularDesarrollo
	comun.actoAsignacion = actoAsignacionReincorporacionTitularDesarrollo
	comun.actoSesion = actoSesionReincorporacionTitularDesarrollo
	return comun, nil
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) PublicarInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	return a.publicarInstantanea(ctx, instantanea)
}

func publicarInstantaneaAsignacionCTSegunRuta(
	ctx context.Context, ruta string,
	autoridad autoridadAsignacionesContratacionTemporalDesarrollo,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	if autoridad == nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	if ruta == httpinterno.RutaReincorporacionesTitular || ruta == httpinterno.RutaCapacidadReincorporacionTitular {
		ct130, ok := autoridad.(autoridadReincorporacionTitularPostgreSQL)
		if !ok {
			return falloPostgreSQLCTDesarrollo(nil)
		}
		return ct130.PublicarInstantaneaReincorporacionTitular(ctx, instantanea)
	}
	return autoridad.PublicarInstantanea(ctx, instantanea)
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) prepararInstantanea(
	ctx context.Context,
	solicitada dominiovec.InstantaneaAutorizacion,
	permitirInicial bool,
) (dominiovec.InstantaneaAutorizacion, error) {
	preparada, err := a.autoridadComun().prepararInstantanea(ctx, solicitada, permitirInicial)
	if err != nil {
		return dominiovec.InstantaneaAutorizacion{}, falloPostgreSQLCTDesarrollo(err)
	}
	return preparada, nil
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) publicarInstantanea(
	ctx context.Context,
	instantanea dominiovec.InstantaneaAutorizacion,
) error {
	if err := a.autoridadComun().publicarInstantanea(ctx, instantanea); err != nil {
		return falloPostgreSQLCTDesarrollo(err)
	}
	return nil
}

func (a *autoridadPostgreSQLContratacionTemporalDesarrollo) autoridadComun() autoridadPostgreSQLDesarrollo {
	if a == nil || a.soporte == nil {
		return autoridadPostgreSQLDesarrollo{}
	}
	if a.soporte.perfilCancelacionCentro {
		// Perfil dinámico propio de la cancelación por el centro: solo
		// continúa una asignación operativa que haya publicado él mismo.
		return autoridadPostgreSQLDesarrollo{
			pool:                  a.pool,
			vinculo:               a.soporte.contexto.Vinculo,
			prefijoBloqueo:        "vec:ct:desarrollo:autorizacion:",
			actoControlRol:        actoControlRolCancelacionCentroDesarrollo,
			actoAsignacion:        actoAsignacionCancelacionCentroDesarrollo,
			actoSesion:            actoSesionCancelacionCentroDesarrollo,
			exigirOrigenOperativo: true,
		}
	}
	// Perfiles dinámicos (RRHH, intervención, lectores y CT130): siguen
	// ajustándose por petición, pero solo encima de una asignación operativa
	// publicada por este mismo circuito. Ninguna ruta reactiva lo que una
	// revocación o restricción gobernada (otro acto) haya cerrado. Solo este
	// circuito publica en esos perfiles (acto CT o, en CT130, su acto propio).
	return autoridadPostgreSQLDesarrollo{
		pool:                  a.pool,
		vinculo:               a.soporte.contexto.Vinculo,
		prefijoBloqueo:        "vec:ct:desarrollo:autorizacion:",
		actoControlRol:        actoControlRolCTDesarrollo,
		actoAsignacion:        actoAsignacionCTDesarrollo,
		actoSesion:            "acto:ct:desarrollo:sesion:v1",
		exigirOrigenOperativo: true,
	}
}
