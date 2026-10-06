package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	core "vec-diputacion-granada/internal/vec/domain"

	"vec-diputacion-granada/internal/shared/telemetria"
)

type consultaSnapshotContextoExterno interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// HuellaSnapshotContextoExterno consulta la preimagen y el canon CTX15 en
// una transacción del propietario. También sirve al provisionador Usuarios;
// no conecta AUT16, no consulta proyecciones internas y no escribe.
func HuellaSnapshotContextoExterno(ctx context.Context, q consultaSnapshotContextoExterno, s SnapshotContextoExterno, version int64, huella string) (string, error) {
	if ctx == nil || ctx.Err() != nil || q == nil || !s.validar() || version < 0 || version >= 1<<31 ||
		(version == 0 && huella != "") || (version > 0 && !huellaPreimagenMiBolsaPortalExterno.MatchString(huella)) {
		return "", ErrProvisionCandidatoExterno
	}
	var encontrada int64
	var previa string
	err := q.QueryRow(ctx, `SELECT version::bigint,huella_sha256 FROM vec_contexto_actor_v1.preimagen_snapshot_contexto_externo_v1($1)`, s.ProvisionRef).Scan(&encontrada, &previa)
	if errors.Is(err, pgx.ErrNoRows) {
		if version != 0 || huella != "" {
			return "", ErrProvisionCandidatoExterno
		}
	} else if err != nil || encontrada != version || previa != huella {
		return "", ErrProvisionCandidatoExterno
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", ErrProvisionCandidatoExterno
	}
	var nueva string
	if q.QueryRow(ctx, `SELECT vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1($1::jsonb,$2::numeric)`, string(b), version+1).Scan(&nueva) != nil || !huellaPreimagenMiBolsaPortalExterno.MatchString(nueva) {
		return "", ErrProvisionCandidatoExterno
	}
	return nueva, nil
}

// PublicarSnapshotContextoExterno consume la huella aprobada y el CAS bajo
// el propietario CTX15. No confirma la transacción: el llamante la controla.
func PublicarSnapshotContextoExterno(ctx context.Context, q consultaSnapshotContextoExterno, s SnapshotContextoExterno, version int64, previa, aprobada string) error {
	huella, err := HuellaSnapshotContextoExterno(ctx, q, s, version, previa)
	if err != nil || huella != aprobada {
		return ErrProvisionCandidatoExterno
	}
	b, err := json.Marshal(s)
	if err != nil {
		return ErrProvisionCandidatoExterno
	}
	var ref, confirmada string
	var publicada int64
	err = q.QueryRow(ctx, `SELECT provision_ref,version::bigint,huella_sha256 FROM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1($1::jsonb,$2::numeric,$3,$4)`, string(b), version, huellaNulaProvisionExterna(previa), aprobada).Scan(&ref, &publicada, &confirmada)
	if err != nil || ref != s.ProvisionRef || publicada != version+1 || confirmada != aprobada {
		return ErrProvisionCandidatoExterno
	}
	return nil
}

// CompletarPlanProvisionCandidatoExterno solo abre una transacción de
// lectura para la huella canónica CTX15; los demás planes son puros.
func CompletarPlanProvisionCandidatoExterno(ctx context.Context, cfg config.Config, dsn string, p PlanProvisionCandidatoExterno) (resultado PlanProvisionCandidatoExterno, fallo error) {
	if !procesoInternoProvisionExterna(cfg) || ctx == nil || ctx.Err() != nil || p.resumen.HuellaSHA256 == "" {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	if p.resumen.Fase != "contexto" {
		return p, nil
	}
	pool, err := abrirPoolProvisionCandidatoExterno(ctx, dsn)
	if err != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	defer func() {
		if revertirProvisionExterna(ctx, tx) != nil {
			resultado, fallo = PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
		}
	}()
	if prepararCanalProvisionExterna(ctx, tx, "contexto") != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	var s SnapshotContextoExterno
	if json.Unmarshal(p.snapshot, &s) != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	p.huellaContexto, err = HuellaSnapshotContextoExterno(ctx, tx, s, p.preimagen.VersionContexto, p.preimagen.HuellaContexto)
	if err != nil {
		return PlanProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
	}
	if err := p.actualizarHuella(); err != nil {
		return PlanProvisionCandidatoExterno{}, err
	}
	return p, nil
}

// EjecutarProvisionCandidatoExterno publica solo la fase aprobada. Un error
// al confirmar no se reintenta: AUT16 no ofrece replay, y se debe reconciliar
// la preimagen por el canal del propietario antes de preparar otro plan.
func EjecutarProvisionCandidatoExterno(ctx context.Context, cfg config.Config, dsn string, p PlanProvisionCandidatoExterno, aprobar, preimagen string) (resultado ResumenProvisionCandidatoExterno, fallo error) {
	var vacio ResumenProvisionCandidatoExterno
	if ctx == nil || ctx.Err() != nil || !procesoInternoProvisionExterna(cfg) || !huellaPreimagenMiBolsaPortalExterno.MatchString(aprobar) ||
		aprobar != p.resumen.HuellaSHA256 || preimagen != p.resumen.PreimagenSHA256 ||
		(p.resumen.Fase == "contexto" && !huellaPreimagenMiBolsaPortalExterno.MatchString(p.huellaContexto)) {
		return vacio, ErrProvisionCandidatoExterno
	}
	pool, err := abrirPoolProvisionCandidatoExterno(ctx, dsn)
	if err != nil {
		return vacio, ErrProvisionCandidatoExterno
	}
	defer pool.Close()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return vacio, ErrProvisionCandidatoExterno
	}
	defer func() {
		if revertirProvisionExterna(ctx, tx) != nil {
			resultado, fallo = ResumenProvisionCandidatoExterno{}, ErrProvisionCandidatoExterno
		}
	}()
	if prepararCanalProvisionExterna(ctx, tx, p.resumen.Fase) != nil {
		return vacio, ErrProvisionCandidatoExterno
	}
	switch p.resumen.Fase {
	case "contexto":
		var s SnapshotContextoExterno
		if json.Unmarshal(p.snapshot, &s) != nil {
			return vacio, ErrProvisionCandidatoExterno
		}
		err = PublicarSnapshotContextoExterno(ctx, tx, s, p.preimagen.VersionContexto, p.preimagen.HuellaContexto, p.huellaContexto)
	case "autorizacion":
		err = publicarAutorizacionCandidatoExterno(ctx, tx, p)
	case "motivos":
		err = publicarMotivosCandidatoExterno(ctx, tx, p)
	case "identidad":
		err = publicarIdentidadCandidatoExterno(ctx, tx, p)
	default:
		err = ErrProvisionCandidatoExterno
	}
	if err != nil || tx.Commit(ctx) != nil {
		return vacio, ErrProvisionCandidatoExterno
	}
	p.resumen.Estado = "confirmado"
	return p.resumen, nil
}

func revertirProvisionExterna(ctx context.Context, tx pgx.Tx) error {
	limite, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	if err := tx.Rollback(limite); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		return ErrProvisionCandidatoExterno
	}
	return nil
}

func abrirPoolProvisionCandidatoExterno(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" {
		return nil, ErrProvisionCandidatoExterno
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User == "" || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, ErrProvisionCandidatoExterno
	}
	c.MaxConns, c.MinConns = 1, 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-provisionar-candidato-externo", "search_path": "pg_catalog", "timezone": "UTC", "statement_timeout": "15s", "lock_timeout": "3s", "idle_in_transaction_session_timeout": "20s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	telemetria.Instrumentar(c) // consultas por petición en el registro de acceso
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, ErrProvisionCandidatoExterno
	}
	return pool, nil
}

// La sesión interna tiene una única membresía directa nominal y ninguna
// cadena privilegiada. AUT16 prohíbe SET ROLE; CTX15 exige su propietario.
func prepararCanalProvisionExterna(ctx context.Context, tx pgx.Tx, fase string) error {
	rol, set := "", false
	switch fase {
	case "identidad":
		rol, set = "vec_identidad_sesiones_v1_propietario", true
	case "contexto":
		rol, set = "vec_contexto_actor_v1_propietario", true
	case "autorizacion":
		rol = "vec_autorizacion_publicador_candidato_externo"
	case "motivos":
		rol, set = rolProyectorMotivosContratacionTemporalDesarrollo, true
	default:
		return ErrProvisionCandidatoExterno
	}
	var valido bool
	err := tx.QueryRow(ctx, `SELECT coalesce(bool_and(session_user=current_user AND current_setting('role')='none'
 AND l.rolcanlogin AND l.rolinherit AND l.rolconfig IS NULL
 AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls)
 AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members WHERE member=l.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=l.oid AND roleid=g.oid
   AND NOT admin_option AND inherit_option AND set_option=$2)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members WHERE member=g.oid)),false)
 FROM pg_catalog.pg_roles l,pg_catalog.pg_roles g WHERE l.rolname=session_user AND g.rolname=$1`, rol, set).Scan(&valido)
	if err != nil || !valido {
		return ErrProvisionCandidatoExterno
	}
	if set {
		// rol procede únicamente del switch cerrado anterior.
		_, err = tx.Exec(ctx, `SET LOCAL ROLE `+pgx.Identifier{rol}.Sanitize())
	}
	if err != nil {
		return ErrProvisionCandidatoExterno
	}
	return nil
}

// PrepararCanalContextoExterno acredita el canal interno nominal CTX15.
// Usuarios lo reutiliza en su propia transacción y con su propia credencial.
func PrepararCanalContextoExterno(ctx context.Context, tx pgx.Tx) error {
	if ctx == nil || ctx.Err() != nil || tx == nil {
		return ErrProvisionCandidatoExterno
	}
	return prepararCanalProvisionExterna(ctx, tx, "contexto")
}

func huellaNulaProvisionExterna(h string) any {
	if h == "" {
		return nil
	}
	return h
}

func publicarAutorizacionCandidatoExterno(ctx context.Context, tx pgx.Tx, p PlanProvisionCandidatoExterno) error {
	var ref, hr, hc string
	var version, revision int64
	err := tx.QueryRow(ctx, `SELECT version_rol_ref,version,revision::bigint,huella_rol,huella_control FROM vec_autorizacion.publicar_rol_candidato_externo_v1($1,$2,$3,$4,$5::numeric,$6,$7,$8)`, p.rol.RolDocumento, p.rol.RolHuellaSHA256, p.rol.ControlDocumento, p.rol.ControlHuellaSHA256, p.rol.RevisionEsperada, p.rol.ControlHuellaEsperada, p.semilla.ControlVigenciaVersionRol.ActualizadoPor, "acto:externo:rol:"+p.resumen.HuellaSHA256).Scan(&ref, &version, &revision, &hr, &hc)
	if err != nil || ref != p.semilla.VersionRol.Referencia() || version != int64(p.semilla.VersionRol.Version) || revision != p.rol.RevisionEsperada+1 || hr != p.rol.RolHuellaSHA256 || hc != p.rol.ControlHuellaSHA256 {
		return ErrProvisionCandidatoExterno
	}
	var asignacionRef, huella string
	var publicada int64
	err = tx.QueryRow(ctx, `SELECT asignacion_ref,version,huella_sha256 FROM vec_autorizacion.publicar_asignacion_candidato_externo_v1($1,$2,$3,$4,$5,$6)`, p.asignacion.Documento, p.asignacion.HuellaSHA256, p.asignacion.VersionEsperada, p.asignacion.HuellaEsperada, p.semilla.AsignacionPerfil.EmitidaPor, "acto:externo:asignacion:"+p.resumen.HuellaSHA256).Scan(&asignacionRef, &publicada, &huella)
	esperada := "asignacion:" + p.semilla.AsignacionPerfil.AsignacionID + ":v" + strconv.FormatInt(p.asignacion.VersionEsperada+1, 10)
	if err != nil || asignacionRef != esperada || publicada != p.asignacion.VersionEsperada+1 || huella != p.asignacion.HuellaSHA256 {
		return ErrProvisionCandidatoExterno
	}
	return nil
}

// El publicador de motivos ya verifica checkpoint, evento, canon y replay.
// La secuencia aprobada se consume una vez por catálogo en esta transacción.
func publicarMotivosCandidatoExterno(ctx context.Context, tx pgx.Tx, p PlanProvisionCandidatoExterno) error {
	for i, m := range p.motivos {
		contenido, err := contenidoCatalogoMotivosPostgreSQLContratacionTemporalDesarrollo([]core.ReferenciaEntradaCatalogo{m}, p.desde)
		if err != nil {
			return ErrProvisionCandidatoExterno
		}
		var publicada bool
		err = tx.QueryRow(ctx, `SELECT vec_autorizacion.publicar_motivos_autorizacion_v2($1,$2,$3,$4,$5,$6,$7,$8::jsonb)`, referenciaAltaContratacionTemporalDesarrollo("evento_", "catalogo-motivos\x00"+m.CatalogoID), p.preimagen.SecuenciaMotivos+int64(i)+1, huellaAltaContratacionTemporalDesarrollo("catalogo-motivos\x00"+m.CatalogoID), m.CatalogoID, m.CatalogoVersion, m.CatalogoHuellaSHA256, p.desde, contenido).Scan(&publicada)
		if err != nil || !publicada {
			return ErrProvisionCandidatoExterno
		}
	}
	return nil
}
