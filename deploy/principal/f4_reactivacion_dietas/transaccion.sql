\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('p6:retirada-dietas-r1d:20260923',0));
SELECT pg_advisory_xact_lock(hashtextextended('f4:reactivacion-dietas-r1d:20260924',0));

CREATE TEMP TABLE f4_plan (dato jsonb NOT NULL) ON COMMIT DROP;
INSERT INTO f4_plan VALUES (convert_from(decode(:'f4_plan_b64','base64'),'UTF8')::jsonb);
GRANT SELECT ON f4_plan TO vec_autorizacion_propietario;

DO $pre$
DECLARE v jsonb; n integer;
BEGIN
 IF current_setting('server_version_num')::integer < 180000
    OR current_setting('server_version_num')::integer >= 190000
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR session_user<>current_user OR inet_server_addr() IS NOT NULL
    OR current_database()<>'postgres' THEN
   RAISE EXCEPTION 'F4 requiere PostgreSQL 18, DBA y socket local de postgres' USING ERRCODE='42501';
 END IF;
 SELECT dato INTO STRICT v FROM f4_plan;
 IF v->>'actor'<>'administracion:f4:reactivacion-dietas-r1d'
    OR v->>'acto'<>'acto:f4:reactivacion-dietas-r1d:20260924'
    OR v->>'original_ref' !~ '^asignacion:dietas_r1d_[a-z0-9]+:v1$'
    OR v->>'anterior_ref'<>'asignacion:'||(v->'documento'->>'asignacion_id')||':v2'
    OR v->>'nueva_ref'<>'asignacion:'||(v->'documento'->>'asignacion_id')||':v3'
    OR v->>'original_huella' !~ '^[0-9a-f]{64}$'
    OR v->>'anterior_huella' !~ '^[0-9a-f]{64}$'
    OR v->>'nueva_huella' !~ '^[0-9a-f]{64}$'
    OR v->'documento'->>'estado'<>'activa'
    OR v->'documento'->>'version_rol_ref'<>'rol:dietas_r1d_provisional:v1'
    OR v->'documento'->>'emitida_por' IS DISTINCT FROM v->>'actor'
    OR v->'documento'->>'emitida_en' IS DISTINCT FROM v->'documento'->>'vigente_desde'
    OR (v->'documento' ? 'revocada_por') OR (v->'documento' ? 'revocacion_ref') THEN
   RAISE EXCEPTION 'plan F4 invalido' USING ERRCODE='55000';
 END IF;
 SELECT count(*) INTO n FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$';
 IF n<>8 THEN RAISE EXCEPTION 'inventario F4 requiere exactamente ocho cuentas; revisar novena auditoria' USING ERRCODE='55000'; END IF;
 IF EXISTS (
   WITH esperados(nombre,grupo) AS (VALUES
    ('vec_dietas_r1d_registro_identidad_desarrollo','vec_identidad_sesiones_v1_registrador'),
    ('vec_dietas_r1d_revalidacion_identidad_desarrollo','vec_identidad_sesiones_v1_revalidador'),
    ('vec_dietas_r1d_contexto_desarrollo','vec_contexto_actor_v1_runtime'),
    ('vec_dietas_r1d_fuente_autorizacion_desarrollo','vec_autorizacion_fuente'),
    ('vec_dietas_r1d_registro_autorizacion_desarrollo','vec_autorizacion_registro'),
    ('vec_dietas_r1d_motivos_desarrollo','vec_autorizacion_motivos_evaluador'),
    ('vec_dietas_r1d_dietas_desarrollo','vec_dietas_ejecutor'),
    ('vec_dietas_r1d_personal_desarrollo','vec_dietas_ejecutor')
   )
   SELECT 1 FROM esperados e
   LEFT JOIN pg_roles r ON r.rolname=e.nombre
   LEFT JOIN pg_roles g ON g.rolname=e.grupo
   WHERE r.oid IS NULL OR g.oid IS NULL OR r.rolcanlogin OR r.rolsuper OR r.rolcreatedb
      OR r.rolcreaterole OR NOT r.rolinherit OR r.rolreplication OR r.rolbypassrls
      OR r.rolconnlimit<>-1 OR r.rolvaliduntil IS NOT NULL
      OR NOT has_database_privilege(r.oid,current_database(),'CONNECT')
      OR has_database_privilege(r.oid,current_database(),'CREATE')
      OR has_database_privilege(r.oid,current_database(),'TEMP')
      OR g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole
      OR g.rolreplication OR g.rolbypassrls
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid)<>1
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.roleid=r.oid)<>0
      -- Los grupos técnicos terminales no pueden heredar pg_read_all_data,
      -- un propietario u otro rol superior, ni permitir SET ROLE hacia él.
      OR EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=g.oid)
      OR EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass
                   AND d.refobjid=r.oid AND d.deptype='a')
      OR EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.refclassid='pg_authid'::regclass
                   AND d.refobjid IN (r.oid,g.oid) AND d.deptype='o')
      OR NOT EXISTS (SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
         WHERE m.member=r.oid AND g.rolname=e.grupo AND NOT m.admin_option
           AND m.inherit_option AND NOT m.set_option)
 ) THEN RAISE EXCEPTION 'cuentas F4 con atributos, ACL o membresias inesperadas' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_roles l JOIN pg_roles g
   ON g.rolname IN ('vec_dietas_ejecutor','vec_dietas_registrador_frontera')
   WHERE l.rolcanlogin AND NOT l.rolsuper
     AND l.rolname NOT IN (
       'vec_dietas_r1d_registro_identidad_desarrollo',
       'vec_dietas_r1d_revalidacion_identidad_desarrollo',
       'vec_dietas_r1d_contexto_desarrollo',
       'vec_dietas_r1d_fuente_autorizacion_desarrollo',
       'vec_dietas_r1d_registro_autorizacion_desarrollo',
       'vec_dietas_r1d_motivos_desarrollo',
       'vec_dietas_r1d_dietas_desarrollo',
       'vec_dietas_r1d_personal_desarrollo')
     AND pg_has_role(l.oid,g.oid,'MEMBER')) THEN
   RAISE EXCEPTION 'LOGIN ajeno conserva ruta a grupo Dietas' USING ERRCODE='55000';
 END IF;
 -- La preimagen permitida no concede derechos a PUBLIC en la base ni en
 -- esquemas/objetos VEC. Un GRANT PUBLIC reabriría acceso de cualquiera de
 -- estos LOGIN aunque su membresía nominal permanezca intacta.
 IF EXISTS (SELECT 1 FROM pg_database d,
      LATERAL aclexplode(coalesce(d.datacl,acldefault('d',d.datdba))) acl
      WHERE d.datname=current_database() AND acl.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_namespace n,
      LATERAL aclexplode(coalesce(n.nspacl,acldefault('n',n.nspowner))) acl
      WHERE left(n.nspname,4)='vec_' AND acl.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace,
      LATERAL aclexplode(coalesce(c.relacl,acldefault((CASE WHEN c.relkind='S' THEN 'S' ELSE 'r' END)::"char",c.relowner))) acl
      WHERE left(n.nspname,4)='vec_' AND c.relkind IN ('r','p','v','m','S','f') AND acl.grantee=0)
    OR EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace,
      LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) acl
      WHERE left(n.nspname,4)='vec_' AND acl.grantee=0) THEN
   RAISE EXCEPTION 'ACL PUBLIC fuera de preimagen F4 permitida' USING ERRCODE='55000';
 END IF;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$') THEN
   RAISE EXCEPTION 'F4 requiere cero sesiones Dietas' USING ERRCODE='55000';
 END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Bloquea la creación/avance concurrente de otro puntero Dietas antes de
-- comprobar la unicidad global y antes de habilitar los LOGIN.
LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE ROW EXCLUSIVE MODE;
DO $activar$
DECLARE v jsonb; anterior vec_autorizacion.asignacion_perfil%ROWTYPE;
 original vec_autorizacion.asignacion_perfil%ROWTYPE;
 puntero vec_autorizacion.asignacion_perfil_actual%ROWTYPE;
 nueva jsonb; filas integer; punteros integer;
BEGIN
 SELECT dato INTO STRICT v FROM f4_plan;
 SELECT count(*) INTO punteros FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
 JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=a.version_rol_ref
 WHERE r.rol_id='dietas_r1d_provisional';
 IF punteros<>1 THEN
   RAISE EXCEPTION 'F4 requiere un unico puntero Dietas global antes de LOGIN' USING ERRCODE='55000';
 END IF;
 SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual
  WHERE perfil_activo_ref=v->'documento'->>'perfil_activo_ref' FOR UPDATE;
 SELECT * INTO STRICT anterior FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=puntero.asignacion_ref;
 SELECT * INTO STRICT original FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=v->>'original_ref';
 nueva:=v->'documento';
 IF puntero.asignacion_ref IS DISTINCT FROM v->>'anterior_ref'
    OR puntero.actualizada_por<>'administracion:p6:retirada-dietas-r1d'
    OR puntero.acto_ref<>'acto:p6:revocacion-dietas-r1d:20260923'
    OR original.version<>1 OR original.documento->>'estado'<>'activa'
    OR original.huella_sha256 IS DISTINCT FROM v->>'original_huella'
    OR anterior.version<>2 OR anterior.documento->>'estado'<>'revocada'
    OR anterior.documento->>'revocada_por'<>'administracion:p6:retirada-dietas-r1d'
    OR anterior.documento->>'revocacion_ref'<>'acto:p6:revocacion-dietas-r1d:20260923'
    OR anterior.huella_sha256 IS DISTINCT FROM v->>'anterior_huella'
    OR anterior.version_rol_ref<>'rol:dietas_r1d_provisional:v1'
    OR anterior.asignacion_id IS DISTINCT FROM original.asignacion_id
    OR anterior.principal_id IS DISTINCT FROM original.principal_id
    OR anterior.perfil_activo_ref IS DISTINCT FROM original.perfil_activo_ref
    OR anterior.documento->'ambitos' IS DISTINCT FROM original.documento->'ambitos'
    OR anterior.documento->>'vigente_desde' IS DISTINCT FROM original.documento->>'vigente_desde'
    OR anterior.documento->>'vigente_hasta' IS DISTINCT FROM original.documento->>'vigente_hasta'
    OR anterior.documento->>'emitida_por' IS DISTINCT FROM original.documento->>'emitida_por'
    OR anterior.documento->>'emitida_en' IS DISTINCT FROM original.documento->>'emitida_en'
    OR nueva->>'asignacion_id' IS DISTINCT FROM anterior.asignacion_id
    OR nueva->>'principal_id' IS DISTINCT FROM anterior.principal_id
    OR nueva->>'perfil_activo_ref' IS DISTINCT FROM anterior.perfil_activo_ref
    OR nueva->>'version_rol_ref' IS DISTINCT FROM anterior.version_rol_ref
    OR nueva->'ambitos' IS DISTINCT FROM anterior.documento->'ambitos'
    OR nueva->>'vigente_hasta' IS DISTINCT FROM anterior.documento->>'vigente_hasta'
    OR nueva->>'vigente_desde' IS DISTINCT FROM nueva->>'emitida_en'
    OR (nueva->>'vigente_desde')::timestamptz <= (anterior.documento->>'revocada_en')::timestamptz
    OR (nueva->>'vigente_desde')::timestamptz >= (nueva->>'vigente_hasta')::timestamptz
    OR clock_timestamp() >= (nueva->>'vigente_hasta')::timestamptz
    OR nueva->>'version'<>'3'
    OR (nueva-'version'-'estado'-'vigente_desde'-'emitida_por'-'emitida_en'-'revocada_por'-'revocada_en'-'revocacion_ref')
       IS DISTINCT FROM (anterior.documento-'version'-'estado'-'vigente_desde'-'emitida_por'-'emitida_en'-'revocada_por'-'revocada_en'-'revocacion_ref')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.version_rol r
      JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
      JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
      WHERE r.version_rol_ref='rol:dietas_r1d_provisional:v1' AND r.documento->>'estado'='publicada'
        AND c.documento->>'estado'='habilitada') THEN
   RAISE EXCEPTION 'preimagen, vigencia o control V3 F4 incompatibles' USING ERRCODE='55000';
 END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
   version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES (v->>'nueva_ref',anterior.asignacion_id,3,anterior.perfil_activo_ref,
  anterior.principal_id,anterior.version_rol_ref,v->>'nueva_huella',
  (nueva->>'emitida_en')::timestamptz,nueva);
 UPDATE vec_autorizacion.asignacion_perfil_actual
 SET asignacion_ref=v->>'nueva_ref',actualizada_en=clock_timestamp(),
     actualizada_por=v->>'actor',acto_ref=v->>'acto'
 WHERE perfil_activo_ref=puntero.perfil_activo_ref AND asignacion_ref=puntero.asignacion_ref;
 GET DIAGNOSTICS filas=ROW_COUNT;
 IF filas<>1 THEN RAISE EXCEPTION 'puntero F4 no avanzo' USING ERRCODE='55000'; END IF;
END $activar$;
RESET ROLE;

ALTER ROLE vec_dietas_r1d_registro_identidad_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_revalidacion_identidad_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_contexto_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_fuente_autorizacion_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_registro_autorizacion_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_motivos_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_dietas_desarrollo LOGIN;
ALTER ROLE vec_dietas_r1d_personal_desarrollo LOGIN;

DO $post$
DECLARE v jsonb;
BEGIN
 SELECT dato INTO STRICT v FROM f4_plan;
 PERFORM pg_stat_clear_snapshot();
 IF (SELECT count(*) FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$' AND rolcanlogin)<>8
    OR EXISTS (SELECT 1 FROM pg_stat_activity WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$')
    OR NOT EXISTS (SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual p
       JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
       WHERE p.asignacion_ref=v->>'nueva_ref' AND a.huella_sha256=v->>'nueva_huella'
         AND a.documento=v->'documento' AND a.documento->>'estado'='activa') THEN
   RAISE EXCEPTION 'postcondicion F4 fallida' USING ERRCODE='55000';
 END IF;
END $post$;

:f4_finalizar;
