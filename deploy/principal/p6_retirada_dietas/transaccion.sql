\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('p6:retirada-dietas-r1d:20260923',0));

-- El plan llega de Validar/HuellaSHA256 de Go, en base64 para que psql no
-- interprete ni el JSON ni las referencias como sintaxis SQL.
CREATE TEMP TABLE p6_plan (dato jsonb NOT NULL) ON COMMIT DROP;
INSERT INTO p6_plan VALUES
 (convert_from(decode(:'p6_plan_b64','base64'),'UTF8')::jsonb);
GRANT SELECT ON p6_plan TO vec_autorizacion_propietario;

DO $pre$
DECLARE v jsonb; n integer;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR session_user<>current_user OR inet_server_addr() IS NOT NULL
    OR current_database()<>'postgres' THEN
   RAISE EXCEPTION 'P6 requiere DBA en socket local de postgres' USING ERRCODE='42501';
 END IF;
 SELECT dato INTO STRICT v FROM p6_plan;
 IF v->>'actor'<>'administracion:p6:retirada-dietas-r1d'
    OR v->>'acto'<>'acto:p6:revocacion-dietas-r1d:20260923'
    OR v->>'nueva_huella' !~ '^[0-9a-f]{64}$'
    OR v->>'anterior_huella' !~ '^[0-9a-f]{64}$'
    OR v->'documento'->>'estado'<>'revocada'
    OR v->'documento'->>'revocada_por' IS DISTINCT FROM v->>'actor'
    OR v->'documento'->>'revocacion_ref' IS DISTINCT FROM v->>'acto'
    OR v->'documento'->>'version_rol_ref'<>'rol:dietas_r1d_provisional:v1' THEN
   RAISE EXCEPTION 'plan Go P6 invalido' USING ERRCODE='55000';
 END IF;
 SELECT count(*) INTO n FROM pg_roles
 WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$';
 IF n<>8 THEN
   RAISE EXCEPTION 'inventario P6 no contiene exactamente ocho LOGIN nominales' USING ERRCODE='55000';
 END IF;
 -- Un LOGIN ajeno con pertenencia directa o transitiva al ejecutor Dietas
 -- conservaria una ruta de entrada. Los grupos compartidos de autorizacion
 -- se inventarian, sin alterar a los consumidores CT/Bolsa.
 IF EXISTS (
   SELECT 1 FROM pg_roles l JOIN pg_roles g
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
     AND pg_has_role(l.oid,g.oid,'MEMBER')
 ) THEN
   RAISE EXCEPTION 'LOGIN ajeno conserva ruta a grupo Dietas' USING ERRCODE='55000';
 END IF;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_stat_activity
            WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$') THEN
   RAISE EXCEPTION 'P6 requiere cero sesiones Dietas antes de la transaccion' USING ERRCODE='55000';
 END IF;
 IF EXISTS (
   WITH esperados(nombre, grupo) AS (VALUES
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
   WHERE r.oid IS NULL OR NOT r.rolcanlogin OR r.rolsuper OR r.rolcreatedb
      OR r.rolcreaterole OR NOT r.rolinherit OR r.rolreplication OR r.rolbypassrls
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.member=r.oid)<>1
      OR (SELECT count(*) FROM pg_auth_members m WHERE m.roleid=r.oid)<>0
      OR NOT EXISTS (
        SELECT 1 FROM pg_auth_members m JOIN pg_roles g ON g.oid=m.roleid
        WHERE m.member=r.oid AND g.rolname=e.grupo
          AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
 ) THEN
   RAISE EXCEPTION 'atributos o membresias P6 incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
DO $revocar$
DECLARE v jsonb; anterior vec_autorizacion.asignacion_perfil%ROWTYPE;
        puntero vec_autorizacion.asignacion_perfil_actual%ROWTYPE;
        nueva jsonb; filas integer;
BEGIN
 SELECT dato INTO STRICT v FROM p6_plan;
 SELECT * INTO STRICT puntero FROM vec_autorizacion.asignacion_perfil_actual
  WHERE perfil_activo_ref=v->'documento'->>'perfil_activo_ref' FOR UPDATE;
 SELECT * INTO STRICT anterior FROM vec_autorizacion.asignacion_perfil
  WHERE asignacion_ref=puntero.asignacion_ref;
 nueva:=v->'documento';
 IF puntero.asignacion_ref IS DISTINCT FROM v->>'anterior_ref'
    OR anterior.huella_sha256 IS DISTINCT FROM v->>'anterior_huella'
    OR anterior.documento->>'estado'<>'activa'
    OR anterior.version_rol_ref<>'rol:dietas_r1d_provisional:v1'
    OR nueva->>'asignacion_id' IS DISTINCT FROM anterior.asignacion_id
    OR nueva->>'principal_id' IS DISTINCT FROM anterior.principal_id
    OR nueva->>'perfil_activo_ref' IS DISTINCT FROM anterior.perfil_activo_ref
    OR nueva->>'version_rol_ref' IS DISTINCT FROM anterior.version_rol_ref
    OR nueva->'ambitos' IS DISTINCT FROM anterior.documento->'ambitos'
    OR nueva->>'vigente_desde' IS DISTINCT FROM anterior.documento->>'vigente_desde'
    OR nueva->>'vigente_hasta' IS DISTINCT FROM anterior.documento->>'vigente_hasta'
    OR nueva->>'emitida_por' IS DISTINCT FROM anterior.documento->>'emitida_por'
    OR nueva->>'emitida_en' IS DISTINCT FROM anterior.documento->>'emitida_en'
    OR (nueva->>'version')::bigint<>anterior.version+1
    OR v->>'nueva_ref'<>'asignacion:'||anterior.asignacion_id||':v'||(anterior.version+1)::text
    OR (nueva-'version'-'estado'-'revocada_por'-'revocada_en'-'revocacion_ref')
       IS DISTINCT FROM (anterior.documento-'version'-'estado'-'revocada_por'-'revocada_en'-'revocacion_ref') THEN
   RAISE EXCEPTION 'preimagen P6 cambio o documento alterado' USING ERRCODE='55000';
 END IF;
 INSERT INTO vec_autorizacion.asignacion_perfil
  (asignacion_ref,asignacion_id,version,perfil_activo_ref,principal_id,
   version_rol_ref,huella_sha256,emitida_en,documento)
 VALUES (v->>'nueva_ref',anterior.asignacion_id,anterior.version+1,
   anterior.perfil_activo_ref,anterior.principal_id,anterior.version_rol_ref,
   v->>'nueva_huella',anterior.emitida_en,nueva);
 UPDATE vec_autorizacion.asignacion_perfil_actual
 SET asignacion_ref=v->>'nueva_ref',actualizada_en=clock_timestamp(),
     actualizada_por=v->>'actor',acto_ref=v->>'acto'
 WHERE perfil_activo_ref=puntero.perfil_activo_ref
   AND asignacion_ref=puntero.asignacion_ref;
 GET DIAGNOSTICS filas=ROW_COUNT;
 IF filas<>1 THEN RAISE EXCEPTION 'puntero P6 no avanzo' USING ERRCODE='55000'; END IF;
END $revocar$;
RESET ROLE;

ALTER ROLE vec_dietas_r1d_registro_identidad_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_revalidacion_identidad_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_contexto_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_fuente_autorizacion_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_registro_autorizacion_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_motivos_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_dietas_desarrollo NOLOGIN;
ALTER ROLE vec_dietas_r1d_personal_desarrollo NOLOGIN;

DO $post$
DECLARE v jsonb;
BEGIN
 SELECT dato INTO STRICT v FROM p6_plan;
 PERFORM pg_stat_clear_snapshot();
 IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname ~ '^vec_dietas_r1d_.*_desarrollo$'
                                 AND rolcanlogin)
    OR EXISTS (
      SELECT 1 FROM pg_roles l JOIN pg_roles g
        ON g.rolname IN ('vec_dietas_ejecutor','vec_dietas_registrador_frontera')
      WHERE l.rolcanlogin AND NOT l.rolsuper
        AND pg_has_role(l.oid,g.oid,'MEMBER'))
    OR EXISTS (SELECT 1 FROM pg_stat_activity
               WHERE usename ~ '^vec_dietas_r1d_.*_desarrollo$')
    OR NOT EXISTS (
      SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual p
      JOIN vec_autorizacion.asignacion_perfil a ON a.asignacion_ref=p.asignacion_ref
      WHERE p.asignacion_ref=v->>'nueva_ref' AND a.huella_sha256=v->>'nueva_huella'
        AND a.documento=v->'documento' AND a.documento->>'estado'='revocada') THEN
   RAISE EXCEPTION 'postcondicion P6 fallida' USING ERRCODE='55000';
 END IF;
END $post$;

:p6_finalizar;
