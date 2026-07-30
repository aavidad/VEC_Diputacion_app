package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const consultaAcreditacionPoolResolucionMotivosRRHH = `
WITH
bloqueo_rol AS MATERIALIZED (
  SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_autorizacion:rol-motivos-rrhh-resolutor:v1',0))
),
bloqueo_000008 AS MATERIALIZED (
  SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000008',0))
  FROM bloqueo_rol
),
bloqueo_000009 AS MATERIALIZED (
  SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000009',0))
  FROM bloqueo_000008
),
bloqueo_000010 AS MATERIALIZED (
  SELECT pg_catalog.pg_advisory_xact_lock_shared(pg_catalog.hashtextextended(
    'vec_autorizacion:migracion:vinculaciones-motivo-rrhh:000010',0))
  FROM bloqueo_000009
),
login AS MATERIALIZED (
  SELECT * FROM pg_catalog.pg_roles WHERE rolname=$1
),
grupo AS MATERIALIZED (
  SELECT * FROM pg_catalog.pg_roles
  WHERE rolname='vec_autorizacion_motivos_rrhh_resolutor'
),
base_actual AS MATERIALIZED (
  SELECT oid,datname,datacl,datdba FROM pg_catalog.pg_database
  WHERE datname=pg_catalog.current_database()
),
esquema AS MATERIALIZED (
  SELECT oid,nspname,nspacl,nspowner FROM pg_catalog.pg_namespace
  WHERE nspname='vec_autorizacion'
),
esquemas_aplicativos AS MATERIALIZED (
  SELECT oid,nspname,nspacl,nspowner FROM pg_catalog.pg_namespace
  WHERE nspname<>'information_schema' AND nspname!~'^pg_'
),
esperado(nombre,objeto,identidad,cuerpo,longitud,comentario) AS (VALUES
 ('resolver_motivo_cuadro_rrhh_v1',
  'vec_autorizacion.resolver_motivo_cuadro_rrhh_v1(timestamp with time zone)',
  'd92704658d0af8acea83cd765e02976561c787a95906b9a10ee8a43ac0be16ef',
  'a3d784e0a266885ca98b01e355a36cfd3acb0be1c3da6eea53a09376bc680264',
  6699,
  'vec_autorizacion:vinculacion-motivo-consulta-rrhh:resolver-cuadro-v1:000010'),
 ('resolver_motivo_detalle_rrhh_v1',
  'vec_autorizacion.resolver_motivo_detalle_rrhh_v1(timestamp with time zone)',
  'ec662cc7118eb25eb2ebe79107c1ad1f16e5f5197fab8ff5e2051b3ddbc9fc7a',
  'a6c3617ccb33da4bf69e2deff447bb1f93966472409fca518723919e2ce80d60',
  6702,
  'vec_autorizacion:vinculacion-motivo-consulta-rrhh:resolver-detalle-v1:000010')
),
funciones AS MATERIALIZED (
  SELECT p.*,l.lanname,p.oid::pg_catalog.regprocedure::text AS objeto,
    pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
      pg_catalog.jsonb_build_array(
        l.lanname,p.proowner::pg_catalog.regrole::text,p.prokind::text,
        p.provolatile::text,p.proparallel::text,p.prosecdef,p.proleakproof,
        p.proisstrict,p.proretset,p.pronargs,p.pronargdefaults,
        p.prorettype::pg_catalog.regtype::text,p.proargtypes::text,
        p.proallargtypes,p.proargmodes,p.proargnames,p.protrftypes,
        p.provariadic::pg_catalog.regtype::text,p.proconfig,p.prosrc,p.probin,
        p.prosqlbody::text,p.procost,p.prorows,
        p.prosupport::pg_catalog.regprocedure::text,
        pg_catalog.obj_description(p.oid,'pg_proc'))::text,'UTF8')),'hex')
      AS identidad,
    pg_catalog.encode(pg_catalog.sha256(
      pg_catalog.convert_to(p.prosrc,'UTF8')),'hex') AS cuerpo,
    pg_catalog.octet_length(p.prosrc) AS longitud
  FROM pg_catalog.pg_proc p
  JOIN pg_catalog.pg_language l ON l.oid=p.prolang
  WHERE p.pronamespace='vec_autorizacion'::pg_catalog.regnamespace
    AND p.proname=ANY(ARRAY[
      'resolver_motivo_cuadro_rrhh_v1',
      'resolver_motivo_detalle_rrhh_v1']::name[])
),
funciones_resumen AS MATERIALIZED (
  SELECT
    COALESCE(pg_catalog.max(oid) FILTER (
      WHERE proname='resolver_motivo_cuadro_rrhh_v1'),0::oid) AS oid_cuadro,
    COALESCE(pg_catalog.max(oid) FILTER (
      WHERE proname='resolver_motivo_detalle_rrhh_v1'),0::oid) AS oid_detalle
  FROM funciones
),
acl_esperada(clase,objeto,grantor,grantee,privilegio,delegable) AS (
  SELECT 'base'::text,b.datname::text,b.datdba,g.oid,'CONNECT'::text,false
  FROM base_actual b CROSS JOIN grupo g
  UNION ALL
  SELECT 'esquema',e.nspname::text,e.nspowner,g.oid,'USAGE',false
  FROM esquema e CROSS JOIN grupo g
  UNION ALL
  SELECT 'funcion',x.objeto::text,e.nspowner,e.nspowner,'EXECUTE',false
  FROM esperado x CROSS JOIN esquema e
  UNION ALL
  SELECT 'funcion',x.objeto::text,e.nspowner,g.oid,'EXECUTE',false
  FROM esperado x CROSS JOIN esquema e CROSS JOIN grupo g
),
acl_actual AS MATERIALIZED (
  SELECT 'base'::text,b.datname::text,a.grantor,a.grantee,
    a.privilege_type,a.is_grantable
  FROM pg_catalog.pg_database b
  CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
    b.datacl,pg_catalog.acldefault('d',b.datdba))) a
  CROSS JOIN grupo g
  WHERE a.grantee=g.oid OR a.grantor=g.oid
  UNION ALL
  SELECT 'esquema',n.nspname,a.grantor,a.grantee,
    a.privilege_type,a.is_grantable
  FROM pg_catalog.pg_namespace n
  CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
    n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a
  CROSS JOIN grupo g
  WHERE a.grantee=g.oid OR a.grantor=g.oid
  UNION ALL
  SELECT 'funcion',p.oid::pg_catalog.regprocedure::text,a.grantor,a.grantee,
    a.privilege_type,a.is_grantable
  FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
    p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid IN (SELECT oid FROM funciones)
  UNION ALL
  SELECT 'funcion',p.oid::pg_catalog.regprocedure::text,a.grantor,a.grantee,
    a.privilege_type,a.is_grantable
  FROM pg_catalog.pg_proc p
  CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
    p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  CROSS JOIN grupo g
  WHERE p.oid NOT IN (SELECT oid FROM funciones)
    AND (a.grantee=g.oid OR a.grantor=g.oid)
),
dependencias_esperadas(dbid,classid,objid,objsubid,deptype) AS (
  SELECT 0::oid,'pg_catalog.pg_database'::pg_catalog.regclass,
    b.oid,0,'a'::"char"
  FROM base_actual b
  UNION ALL
  SELECT b.oid,'pg_catalog.pg_namespace'::pg_catalog.regclass,
    e.oid,0,'a'::"char"
  FROM base_actual b CROSS JOIN esquema e
  UNION ALL
  SELECT b.oid,'pg_catalog.pg_proc'::pg_catalog.regclass,
    f.oid,0,'a'::"char"
  FROM base_actual b CROSS JOIN funciones f
),
dependencias_actuales AS MATERIALIZED (
  SELECT d.dbid,d.classid,d.objid,d.objsubid,d.deptype
  FROM pg_catalog.pg_shdepend d CROSS JOIN grupo g
  WHERE d.refclassid='pg_catalog.pg_authid'::pg_catalog.regclass
    AND d.refobjid=g.oid
)
SELECT
  session_user::text,
  current_user::text,
  (SELECT oid_cuadro FROM funciones_resumen),
  (SELECT oid_detalle FROM funciones_resumen),
  COALESCE((
    SELECT CASE WHEN $2::boolean THEN
      ssl AND (
        (version='TLSv1.2' AND cipher IN (
          'ECDHE-ECDSA-AES128-GCM-SHA256',
          'ECDHE-ECDSA-AES256-GCM-SHA384',
          'ECDHE-RSA-AES128-GCM-SHA256',
          'ECDHE-RSA-AES256-GCM-SHA384',
          'ECDHE-ECDSA-CHACHA20-POLY1305',
          'ECDHE-RSA-CHACHA20-POLY1305'))
        OR
        (version='TLSv1.3' AND cipher IN (
          'TLS_AES_128_GCM_SHA256',
          'TLS_AES_256_GCM_SHA384',
          'TLS_CHACHA20_POLY1305_SHA256')))
    ELSE NOT ssl END
    FROM pg_catalog.pg_stat_ssl
    WHERE pid=pg_catalog.pg_backend_pid()),false),
  NOT pg_catalog.pg_is_in_recovery(),
  COALESCE((
    SELECT r.rolcanlogin AND r.rolinherit AND NOT r.rolsuper
      AND NOT r.rolcreatedb AND NOT r.rolcreaterole
      AND NOT r.rolreplication AND NOT r.rolbypassrls
      AND r.rolconnlimit=-1 AND r.rolvaliduntil IS NULL
      AND r.rolconfig IS NULL
    FROM login r),false),
  COALESCE((
    SELECT NOT r.rolcanlogin AND NOT r.rolinherit AND NOT r.rolsuper
      AND NOT r.rolcreatedb AND NOT r.rolcreaterole
      AND NOT r.rolreplication AND NOT r.rolbypassrls
      AND r.rolconnlimit=-1 AND r.rolvaliduntil IS NULL
      AND r.rolconfig IS NULL
      AND pg_catalog.shobj_description(r.oid,'pg_authid')=
        'vec_autorizacion:rol-motivos-rrhh-resolutor:v1'
    FROM grupo r),false),
  COALESCE((
    SELECT pg_catalog.count(*)=1
      AND pg_catalog.bool_and(m.roleid=g.oid)
      AND pg_catalog.bool_and(NOT m.admin_option)
      AND pg_catalog.bool_and(m.inherit_option)
      AND pg_catalog.bool_and(NOT m.set_option)
      AND pg_catalog.bool_and(m.grantor=10)
    FROM pg_catalog.pg_auth_members m
    CROSS JOIN login l CROSS JOIN grupo g
    WHERE m.member=l.oid),false)
  AND COALESCE((
    SELECT pg_catalog.count(*)=1 AND
      pg_catalog.bool_and(r.oid=g.oid)
    FROM pg_catalog.pg_roles r CROSS JOIN login l CROSS JOIN grupo g
    WHERE r.oid<>l.oid AND pg_catalog.pg_has_role(l.oid,r.oid,'MEMBER')
  ),false)
  AND NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_auth_members m
    CROSS JOIN login l CROSS JOIN grupo g
    WHERE (m.roleid=l.oid OR m.grantor=l.oid OR m.member=g.oid
      OR m.grantor=g.oid)
  )
  AND EXISTS (
    SELECT 1 FROM pg_catalog.pg_roles WHERE oid=10 AND rolsuper
  ),
  NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_db_role_setting s CROSS JOIN login l
    WHERE s.setrole=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_default_acl d
    CROSS JOIN LATERAL pg_catalog.aclexplode(NULLIF(
      d.defaclacl,'{}'::aclitem[])) a CROSS JOIN login l
    WHERE d.defaclrole=l.oid OR a.grantee=l.oid OR a.grantor=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_policy p CROSS JOIN login l
    WHERE l.oid=ANY(p.polroles)
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_shdepend d CROSS JOIN login l
    WHERE d.refclassid='pg_catalog.pg_authid'::pg_catalog.regclass
      AND d.refobjid=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_database d
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      d.datacl,pg_catalog.acldefault('d',d.datdba))) a CROSS JOIN login l
    WHERE a.grantee=l.oid OR a.grantor=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_namespace n
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      n.nspacl,pg_catalog.acldefault('n',n.nspowner))) a CROSS JOIN login l
    WHERE a.grantee=l.oid OR a.grantor=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      c.relacl,pg_catalog.acldefault(
        CASE WHEN c.relkind='S' THEN 's'::"char" ELSE 'r'::"char" END,
        c.relowner))) a CROSS JOIN login l
    WHERE a.grantee=l.oid OR a.grantor=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_attribute at
    CROSS JOIN LATERAL pg_catalog.aclexplode(NULLIF(
      at.attacl,'{}'::aclitem[])) a CROSS JOIN login l
    WHERE a.grantee=l.oid OR a.grantor=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_type t
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      t.typacl,pg_catalog.acldefault('T',t.typowner))) a CROSS JOIN login l
    WHERE a.grantee=l.oid OR a.grantor=l.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_proc p
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      p.proacl,pg_catalog.acldefault('f',p.proowner))) a CROSS JOIN login l
    WHERE a.grantee=l.oid OR a.grantor=l.oid
  ),
  NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_db_role_setting s CROSS JOIN grupo g
    WHERE s.setrole=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_default_acl d
    CROSS JOIN LATERAL pg_catalog.aclexplode(NULLIF(
      d.defaclacl,'{}'::aclitem[])) a CROSS JOIN grupo g
    WHERE d.defaclrole=g.oid OR a.grantee=g.oid OR a.grantor=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_policy p
    CROSS JOIN pg_catalog.pg_class c
    CROSS JOIN esquema e CROSS JOIN grupo g
    WHERE p.polrelid=c.oid AND c.relnamespace=e.oid
      AND (0=ANY(p.polroles) OR g.oid=ANY(p.polroles))
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_database d CROSS JOIN grupo g
    WHERE d.datdba=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_namespace n CROSS JOIN grupo g
    WHERE n.nspowner=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_class c CROSS JOIN grupo g
    WHERE c.relowner=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN grupo g
    WHERE p.proowner=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_type t CROSS JOIN grupo g
    WHERE t.typowner=g.oid
  )
  AND NOT EXISTS (
    (SELECT * FROM dependencias_esperadas
     EXCEPT ALL SELECT * FROM dependencias_actuales)
    UNION ALL
    (SELECT * FROM dependencias_actuales
     EXCEPT ALL SELECT * FROM dependencias_esperadas)
  ),
  NOT EXISTS (
    (SELECT * FROM acl_esperada EXCEPT ALL SELECT * FROM acl_actual)
    UNION ALL
    (SELECT * FROM acl_actual EXCEPT ALL SELECT * FROM acl_esperada)
  )
  AND NOT EXISTS (
    SELECT 1 FROM base_actual b
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      b.datacl,pg_catalog.acldefault('d',b.datdba))) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM esquemas_aplicativos e
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      e.nspacl,pg_catalog.acldefault('n',e.nspowner))) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_class c
    JOIN esquemas_aplicativos e ON e.oid=c.relnamespace
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      c.relacl,pg_catalog.acldefault(
        CASE WHEN c.relkind='S' THEN 's'::"char" ELSE 'r'::"char" END,
        c.relowner))) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_attribute at
    JOIN pg_catalog.pg_class c ON c.oid=at.attrelid
    JOIN esquemas_aplicativos e ON e.oid=c.relnamespace
    CROSS JOIN LATERAL pg_catalog.aclexplode(NULLIF(
      at.attacl,'{}'::aclitem[])) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_proc p
    JOIN esquemas_aplicativos e ON e.oid=p.pronamespace
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_type t
    JOIN esquemas_aplicativos e ON e.oid=t.typnamespace
    CROSS JOIN LATERAL pg_catalog.aclexplode(CASE
      WHEN t.typelem=0 THEN COALESCE(
        t.typacl,pg_catalog.acldefault('T',t.typowner))
      ELSE NULLIF(t.typacl,'{}'::aclitem[])
    END) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_default_acl d
    CROSS JOIN LATERAL pg_catalog.aclexplode(NULLIF(
      d.defaclacl,'{}'::aclitem[])) a
    WHERE a.grantee=0
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_policy p
    JOIN pg_catalog.pg_class c ON c.oid=p.polrelid
    JOIN esquemas_aplicativos e ON e.oid=c.relnamespace
    WHERE 0=ANY(p.polroles)
  )
  AND NOT EXISTS (
    SELECT 1 FROM pg_catalog.pg_class c
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      c.relacl,pg_catalog.acldefault(
        CASE WHEN c.relkind='S' THEN 's'::"char" ELSE 'r'::"char" END,
        c.relowner))) a CROSS JOIN grupo g
    WHERE a.grantee=g.oid OR a.grantor=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_attribute at
    CROSS JOIN LATERAL pg_catalog.aclexplode(NULLIF(
      at.attacl,'{}'::aclitem[])) a CROSS JOIN grupo g
    WHERE a.grantee=g.oid OR a.grantor=g.oid
    UNION ALL
    SELECT 1 FROM pg_catalog.pg_type t
    CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(
      t.typacl,pg_catalog.acldefault('T',t.typowner))) a CROSS JOIN grupo g
    WHERE a.grantee=g.oid OR a.grantor=g.oid
  ),
  COALESCE((
    SELECT
      pg_catalog.has_database_privilege(l.oid,b.oid,'CONNECT')
      AND NOT pg_catalog.has_database_privilege(l.oid,b.oid,'CREATE')
      AND NOT pg_catalog.has_database_privilege(l.oid,b.oid,'TEMP')
      AND pg_catalog.has_schema_privilege(l.oid,e.oid,'USAGE')
      AND NOT pg_catalog.has_schema_privilege(l.oid,e.oid,'CREATE')
      AND NOT EXISTS (
        SELECT 1 FROM esquemas_aplicativos ea
        WHERE ea.oid<>e.oid AND (
          pg_catalog.has_schema_privilege(l.oid,ea.oid,'USAGE')
          OR pg_catalog.has_schema_privilege(l.oid,ea.oid,'CREATE'))
      )
      AND (
        SELECT pg_catalog.count(*)=2
        FROM pg_catalog.pg_proc p
        JOIN esquemas_aplicativos ea ON ea.oid=p.pronamespace
        WHERE pg_catalog.has_function_privilege(l.oid,p.oid,'EXECUTE')
      )
      AND (
        SELECT pg_catalog.count(*)=2
        FROM funciones f
        WHERE pg_catalog.has_function_privilege(l.oid,f.oid,'EXECUTE')
      )
      AND NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_class c
        JOIN esquemas_aplicativos ea ON ea.oid=c.relnamespace
        WHERE CASE
          WHEN c.relkind IN ('r','p','v','m','f')
          THEN
            pg_catalog.has_table_privilege(l.oid,c.oid,
              'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN')
            OR pg_catalog.has_any_column_privilege(l.oid,c.oid,
              'SELECT,INSERT,UPDATE,REFERENCES')
          ELSE false
        END
      )
      AND NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_class c
        JOIN esquemas_aplicativos ea ON ea.oid=c.relnamespace
        WHERE CASE
          WHEN c.relkind='S'
          THEN pg_catalog.has_sequence_privilege(
            l.oid,c.oid,'USAGE,SELECT,UPDATE')
          ELSE false
        END
      )
      AND NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_type t
        JOIN esquemas_aplicativos ea ON ea.oid=t.typnamespace
        WHERE (t.typelem=0 OR t.typacl IS NOT NULL)
          AND pg_catalog.has_type_privilege(l.oid,t.oid,'USAGE')
      )
    FROM login l CROSS JOIN base_actual b CROSS JOIN esquema e
  ),false),
  COALESCE((
    SELECT pg_catalog.count(*)=2 FROM funciones
  ),false)
  AND NOT EXISTS (
    SELECT 1 FROM esperado x
    LEFT JOIN funciones f ON f.objeto=x.objeto
    WHERE f.oid IS NULL OR f.identidad IS DISTINCT FROM x.identidad
      OR f.cuerpo IS DISTINCT FROM x.cuerpo
      OR f.longitud IS DISTINCT FROM x.longitud
      OR pg_catalog.obj_description(f.oid,'pg_proc')
        IS DISTINCT FROM x.comentario
  )
  AND NOT EXISTS (
    SELECT 1 FROM funciones f
    LEFT JOIN esperado x ON x.objeto=f.objeto
    WHERE x.objeto IS NULL
  ),
  COALESCE((SELECT pg_catalog.count(*)=1 FROM bloqueo_000010),false)
`

type conexionAcreditacionResolucionMotivosRRHH interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type transaccionAcreditacionResolucionMotivosRRHH interface {
	conexionAcreditacionResolucionMotivosRRHH
	Sello() *selloPoolResolucionMotivosRRHH
}

func (p *PoolResolucionMotivosRRHHPostgreSQL) acreditarInicial(
	ctx context.Context,
) (
	oidCuadro uint32,
	oidDetalle uint32,
	errResultado error,
) {
	conexion, err := p.adquirir(ctx, false)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if liberarConexionResolucionMotivosRRHH(conexion) {
			oidCuadro, oidDetalle = 0, 0
			errResultado = errorPoolResolucionMotivosRRHH(ctx)
		}
	}()
	return acreditarConexionResolucionMotivosRRHH(
		ctx, conexion, p.login, p.modoTLS, 0, 0,
	)
}

func (p *PoolResolucionMotivosRRHHPostgreSQL) reacreditar(
	ctx context.Context,
	transaccion transaccionAcreditacionResolucionMotivosRRHH,
) error {
	if p == nil ||
		!selloPoolResolucionMotivosRRHHValido(p.sello, p, true) ||
		dependenciaNula(transaccion) || transaccion.Sello() != p.sello {
		return errorPoolResolucionMotivosRRHH(ctx)
	}
	_, _, err := acreditarConexionResolucionMotivosRRHH(
		ctx, transaccion, p.login, p.modoTLS, p.oidCuadro, p.oidDetalle,
	)
	return err
}

func (p *PoolResolucionMotivosRRHHPostgreSQL) adquirirOperacion(
	ctx context.Context,
) (conexionPoolResolucionMotivosRRHH, error) {
	return p.adquirir(ctx, true)
}

func (p *PoolResolucionMotivosRRHHPostgreSQL) adquirir(
	ctx context.Context,
	exigirOID bool,
) (
	resultado conexionPoolResolucionMotivosRRHH,
	errResultado error,
) {
	var conexion conexionPoolResolucionMotivosRRHH
	transferida := false
	defer func() {
		panico := recover()
		if !transferida && !dependenciaNula(conexion) {
			liberarConexionResolucionMotivosRRHH(conexion)
		}
		if panico != nil {
			resultado, errResultado = nil, errorPoolResolucionMotivosRRHH(ctx)
		}
	}()
	if dependenciaNula(ctx) || p == nil || dependenciaNula(p.origen) ||
		!selloPoolResolucionMotivosRRHHValido(p.sello, p, exigirOID) ||
		p.origen.Sello() != p.sello ||
		!configuracionPoolResolucionMotivosRRHHValida(
			p.origen.Configuracion(), p.login, p.modoTLS,
		) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	var err error
	conexion, err = p.origen.Adquirir(ctx)
	if err != nil || dependenciaNula(conexion) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	if conexion.Sello() != p.sello ||
		!configuracionConexionResolucionMotivosRRHHValida(
			conexion.Configuracion(), p.login, p.modoTLS,
		) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	transferida = true
	return conexion, nil
}

func configuracionPoolResolucionMotivosRRHHValida(
	configuracion *pgxpool.Config,
	login string,
	modo modoTLSAcreditacionPoolO405,
) bool {
	return configuracion != nil && configuracion.ConnConfig != nil &&
		configuracion.ConnConfig.User == login &&
		configuracionPoolAcreditacionO405Valida(configuracion, modo)
}

func configuracionConexionResolucionMotivosRRHHValida(
	configuracion *pgx.ConnConfig,
	login string,
	modo modoTLSAcreditacionPoolO405,
) bool {
	return configuracion != nil && configuracion.User == login &&
		configuracionConexionAcreditacionO405Valida(configuracion, modo)
}

func acreditarConexionResolucionMotivosRRHH(
	ctx context.Context,
	conexion conexionAcreditacionResolucionMotivosRRHH,
	login string,
	modo modoTLSAcreditacionPoolO405,
	oidCuadroEsperado uint32,
	oidDetalleEsperado uint32,
) (oidCuadro uint32, oidDetalle uint32, errResultado error) {
	defer func() {
		if recover() != nil {
			oidCuadro, oidDetalle = 0, 0
			errResultado = errorPoolResolucionMotivosRRHH(ctx)
		}
	}()
	if dependenciaNula(ctx) || dependenciaNula(conexion) ||
		!loginResolutorMotivosRRHHValido(login) ||
		(modo != modoTLSAcreditacionPoolO405Produccion &&
			modo != modoTLSAcreditacionPoolO405SocketUnixPrueba) {
		return 0, 0, errorPoolResolucionMotivosRRHH(ctx)
	}
	if err := ctx.Err(); err != nil {
		return 0, 0, errorPoolResolucionMotivosRRHH(ctx)
	}
	var sesion, efectivo string
	var transporte, primaria, loginSeguro, grupoSeguro, topologia bool
	var inventarioLogin, inventarioGrupo, acl, privilegios, manifiesto bool
	var bloqueos bool
	err := conexion.QueryRow(
		ctx, consultaAcreditacionPoolResolucionMotivosRRHH, login,
		modo == modoTLSAcreditacionPoolO405Produccion,
	).Scan(
		&sesion, &efectivo, &oidCuadro, &oidDetalle,
		&transporte, &primaria, &loginSeguro, &grupoSeguro, &topologia,
		&inventarioLogin, &inventarioGrupo, &acl, &privilegios,
		&manifiesto, &bloqueos,
	)
	if err != nil || sesion != login || efectivo != login ||
		oidCuadro == 0 || oidDetalle == 0 || oidCuadro == oidDetalle ||
		(oidCuadroEsperado != 0 && oidCuadro != oidCuadroEsperado) ||
		(oidDetalleEsperado != 0 && oidDetalle != oidDetalleEsperado) ||
		!transporte || !primaria || !loginSeguro || !grupoSeguro ||
		!topologia || !inventarioLogin || !inventarioGrupo || !acl ||
		!privilegios || !manifiesto || !bloqueos {
		return 0, 0, errorPoolResolucionMotivosRRHH(ctx)
	}
	return oidCuadro, oidDetalle, nil
}

func liberarConexionResolucionMotivosRRHH(
	conexion conexionPoolResolucionMotivosRRHH,
) (fallo bool) {
	defer func() {
		if recover() != nil {
			fallo = true
		}
	}()
	if !dependenciaNula(conexion) {
		conexion.Liberar()
	}
	return false
}
