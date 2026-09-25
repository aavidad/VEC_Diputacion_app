\set ON_ERROR_STOP on
-- Errores en el cliente solo con el mensaje primario: sin LINE, QUERY ni
-- CONTEXT que pudieran reproducir texto de sentencias con verificadores.
\set VERBOSITY terse
\set SHOW_CONTEXT never
-- F4b: inicio y definiciones comunes de activación y retirada. El ejecutor
-- concatena variables, este fichero y transaccion.sql o retirada.sql en una
-- sola transacción. Solo objetos temporales; ningún objeto persistente.
BEGIN;
-- Ninguna sentencia de esta transacción llega al registro del servidor: la
-- activación interpola los verificadores SCRAM en su texto. Se anulan el
-- registro de sentencias, el de sentencias fallidas y los de duración
-- (completo y por muestreo); los errores se siguen registrando sin sentencia.
-- Exige superusuario, como el resto de F4b; SET LOCAL muere con la transacción.
SET LOCAL log_statement='none';
SET LOCAL log_min_error_statement='panic';
SET LOCAL log_min_duration_statement=-1;
SET LOCAL log_min_duration_sample=-1;
SET LOCAL log_transaction_sample_rate=0;
-- Extensiones que registran texto de sentencias anidadas si están cargadas;
-- si no lo están, el ajuste solo crea un marcador inocuo de la transacción.
SET LOCAL pg_stat_statements.track='none';
SET LOCAL auto_explain.log_min_duration=-1;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('p6:retirada-dietas-r1d:20260923',0));
SELECT pg_advisory_xact_lock(hashtextextended('f4:reactivacion-dietas-r1d:20260924',0));
SELECT pg_advisory_xact_lock(hashtextextended('f4b:acceso-dietas:20260925',0));

DO $version$
BEGIN
 IF current_setting('server_version_num')::integer < 180000
    OR current_setting('server_version_num')::integer >= 190000
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR session_user<>current_user OR inet_server_addr() IS NOT NULL
    OR current_database()<>'postgres' THEN
   RAISE EXCEPTION 'F4b requiere PostgreSQL 18, DBA y socket local de postgres' USING ERRCODE='42501';
 END IF;
END $version$;

CREATE TEMP TABLE f4b_cuenta (nombre text PRIMARY KEY, grupo text NOT NULL, nueva boolean NOT NULL)
 ON COMMIT DROP;
INSERT INTO f4b_cuenta VALUES
 ('vec_dietas_r1d_registro_identidad_desarrollo','vec_identidad_sesiones_v1_registrador',false),
 ('vec_dietas_r1d_revalidacion_identidad_desarrollo','vec_identidad_sesiones_v1_revalidador',false),
 ('vec_dietas_r1d_contexto_desarrollo','vec_contexto_actor_v1_runtime',false),
 ('vec_dietas_r1d_fuente_autorizacion_desarrollo','vec_autorizacion_fuente',false),
 ('vec_dietas_r1d_registro_autorizacion_desarrollo','vec_autorizacion_registro',false),
 ('vec_dietas_r1d_motivos_desarrollo','vec_autorizacion_motivos_evaluador',false),
 ('vec_dietas_r1d_dietas_desarrollo','vec_dietas_ejecutor',false),
 ('vec_dietas_r1d_personal_desarrollo','vec_dietas_ejecutor',false),
 -- Tres cuentas que la composición actual exige además de las ocho R1D:
 -- auditoría de frontera de Dietas y las dos de Personal (asignación D7 y su
 -- auditoría). Fuera del prefijo r1d para no alterar el recuento P6/F4.
 ('vec_dietas_f4b_auditoria_frontera_desarrollo','vec_dietas_registrador_frontera',true),
 ('vec_personal_d7_asignacion','vec_personal_d7_ejecutor',true),
 ('vec_personal_d7_auditoria_frontera','vec_personal_registrador_frontera',true);

-- Esquemas donde cada grupo técnico puede tener privilegios. Cualquier ACL de
-- un grupo fuera de ellos (CT, Bolsa, otra base, CREATE/TEMP, escritura directa
-- de tablas, objetos de otra clase) detiene F4b.
CREATE TEMP TABLE f4b_esquema_permitido (grupo text NOT NULL, esquema text NOT NULL) ON COMMIT DROP;
INSERT INTO f4b_esquema_permitido VALUES
 ('vec_identidad_sesiones_v1_registrador','vec_identidad_sesiones_v1'),
 ('vec_identidad_sesiones_v1_revalidador','vec_identidad_sesiones_v1'),
 ('vec_contexto_actor_v1_runtime','vec_contexto_actor_v1'),
 ('vec_autorizacion_fuente','vec_autorizacion'),
 ('vec_autorizacion_registro','vec_autorizacion'),
 ('vec_autorizacion_motivos_evaluador','vec_autorizacion'),
 ('vec_dietas_ejecutor','vec_dietas'),
 ('vec_dietas_ejecutor','vec_personal'),
 ('vec_dietas_ejecutor','vec_autorizacion_atestada_v3'),
 ('vec_dietas_registrador_frontera','vec_dietas'),
 ('vec_personal_d7_ejecutor','vec_personal'),
 ('vec_personal_registrador_frontera','vec_personal');

-- ACL heredable observada de los nueve grupos (base, esquema, relación,
-- columna, función y tipo), con el esquema del objeto.
CREATE TEMP VIEW f4b_acl_grupo AS
WITH grupos AS (SELECT DISTINCT r.oid, r.rolname FROM pg_roles r JOIN f4b_cuenta c ON c.grupo=r.rolname)
SELECT g.rolname::text grupo,'base'::text clase,d.datname::text objeto,NULL::text esquema,
       a.privilege_type::text privilegio,a.is_grantable grantable
  FROM pg_database d,LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'esquema',n.nspname,n.nspname,a.privilege_type,a.is_grantable
  FROM pg_namespace n,LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'relacion',c.oid::regclass::text,n.nspname,a.privilege_type,a.is_grantable
  FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace,
       LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'columna',c.oid::regclass::text||'.'||x.attname,n.nspname,a.privilege_type,a.is_grantable
  FROM pg_attribute x JOIN pg_class c ON c.oid=x.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace,
       LATERAL aclexplode(x.attacl) a
  JOIN grupos g ON g.oid=a.grantee WHERE x.attnum>0
UNION ALL
SELECT g.rolname,'funcion',p.oid::regprocedure::text,n.nspname,a.privilege_type,a.is_grantable
  FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace,
       LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  JOIN grupos g ON g.oid=a.grantee
UNION ALL
SELECT g.rolname,'tipo',t.oid::regtype::text,n.nspname,a.privilege_type,a.is_grantable
  FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace,
       LATERAL aclexplode(t.typacl) a
  JOIN grupos g ON g.oid=a.grantee;
