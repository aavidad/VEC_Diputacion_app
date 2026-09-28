\set ON_ERROR_STOP on
-- Solo permite revertir una instalación sin ceses ni publicaciones posteriores.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000045',0));
DO $pre$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_relevo_cese')
    OR to_regprocedure('vec_contratacion_temporal.verificar_cese_publicado_bolsa_v1(text,text,bigint)') IS NOT NULL THEN
  RAISE EXCEPTION 'Bolsa 000045 DOWN: sesión o instalación incompatible' USING ERRCODE='55000';
 END IF;
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.restriccion_cese_bolsa)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.auditoria_cese_bolsa)
    OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.cese_ajeno_bolsa)
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.politica_cese_bolsa)<>1
    OR NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.politica_cese_bolsa
       WHERE version=1 AND catalogo_ref='catalogo:bolsa:cese:ejemplo-sintetico:v1')
    OR EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid='vec_bolsa_llamamientos_relevo_cese'::regrole) THEN
  RAISE EXCEPTION 'Bolsa 000045 DOWN: historia o membresía conservada' USING ERRCODE='55000';
 END IF;
END $pre$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $mi_bolsa$
DECLARE v_oid oid:='vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p WHERE p.oid=v_oid;
 FOR v_cambio IN SELECT * FROM (VALUES
  ($a$'estado',situacion.situacion,$a$,
   $b$'estado',CASE WHEN cese.en_restriccion
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN 'disponible_desde' WHEN cese.trabajo_cesado THEN 'disponible'
       ELSE situacion.situacion END,$b$),
  ($a$'desde',to_char(situacion.desde AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),$a$,
   $b$'desde',to_char((CASE WHEN (cese.en_restriccion OR cese.trabajo_cesado)
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN cese.fecha_efecto::timestamp AT TIME ZONE 'Europe/Madrid' ELSE situacion.desde END)
       AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),$b$),
  ($a$'fecha_disponible',CASE WHEN situacion.fecha_disponible IS NULL THEN NULL ELSE to_char(situacion.fecha_disponible AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END$a$,
   $b$'fecha_disponible',CASE WHEN cese.en_restriccion
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN to_char((cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid')
         AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       WHEN situacion.fecha_disponible IS NULL THEN NULL
       ELSE to_char(situacion.fecha_disponible AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END$b$),
  ($a$) situacion ON true
 LEFT JOIN LATERAL ($a$,
   $b$) situacion ON true
 LEFT JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(
   participacion.participacion_ref,p_consultada_en) cese ON true
 LEFT JOIN LATERAL ($b$)
 ) AS x(antes,despues) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.despues,''))<>length(v_cambio.despues) THEN
   RAISE EXCEPTION 'Bolsa 000045 DOWN: Mi Bolsa incompatible' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.despues,v_cambio.antes);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000045 DOWN: Mi Bolsa alterada' USING ERRCODE='55000';
 END IF;
END $mi_bolsa$;
DO $orden$
DECLARE v_oid oid:='vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz)'::regprocedure;
 v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p WHERE p.oid=v_oid;
 FOR v_cambio IN SELECT * FROM (VALUES
  ($a$SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,$a$,
   $b$SELECT e.participacion_ref,e.orden AS orden_acta,s.situacion,s.fecha_disponible,
         coalesce(rc.en_restriccion,false) AS cese_restringido,
         coalesce(rc.trabajo_cesado,false) AS trabajo_cesado,$b$),
  ($a$(s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)) AS ocupa_turno,$a$,
   $b$((s.situacion='disponible' OR (s.situacion='disponible_desde' AND s.fecha_disponible<=p_en)
           OR coalesce(rc.trabajo_cesado,false)) AND NOT coalesce(rc.en_restriccion,false)) AS ocupa_turno,$b$),
  ($a$s ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en$a$,
   $b$s ON true
    LEFT JOIN LATERAL vec_bolsa_llamamientos.estado_cese_bolsa_v1(e.participacion_ref,p_en) rc ON true
    LEFT JOIN LATERAL (SELECT ro.aplicada_en$b$),
  ($a$b.participacion_ref,b.orden_acta,e.orden_vigente,b.situacion,$a$,
   $b$b.participacion_ref,b.orden_acta,e.orden_vigente,
        CASE WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde')
             THEN 'disponible_desde' WHEN b.trabajo_cesado THEN 'disponible'
             ELSE b.situacion END AS situacion,$b$),
  ($a$CASE WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'$a$,
   $b$CASE WHEN b.cese_restringido AND b.situacion IN ('disponible','trabajando','disponible_desde') THEN 'restriccion_cese'
             WHEN b.trabajo_cesado AND b.ocupa_turno THEN 'retorno_tras_cese'
             WHEN NOT b.ocupa_turno AND b.situacion IN ('no_disponible','disponible_desde') THEN 'pausa'$b$)
 ) AS x(antes,despues) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.despues,''))<>length(v_cambio.despues) THEN
   RAISE EXCEPTION 'Bolsa 000045 DOWN: lector incompatible' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.despues,v_cambio.antes);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000045 DOWN: lector alterado' USING ERRCODE='55000';
 END IF;
END $orden$;
DROP FUNCTION vec_bolsa_llamamientos.registrar_restriccion_cese_bolsa_v1(text,text,bigint);
DROP FUNCTION vec_bolsa_llamamientos.confirmar_cese_ajeno_bolsa_v1(text,text,bigint);
DROP FUNCTION vec_bolsa_llamamientos.cursor_restriccion_cese_bolsa_v1();
DROP FUNCTION vec_bolsa_llamamientos.publicar_politica_cese_bolsa_v1(text,jsonb,integer,integer,text);
DROP FUNCTION vec_bolsa_llamamientos.consultar_politica_cese_bolsa_v1();
DROP FUNCTION vec_bolsa_llamamientos.consultar_restriccion_cese_bolsa_v1(text,timestamptz);
DROP FUNCTION vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz);
DROP TABLE vec_bolsa_llamamientos.auditoria_cese_bolsa;
DROP TABLE vec_bolsa_llamamientos.cese_ajeno_bolsa;
DROP TABLE vec_bolsa_llamamientos.restriccion_cese_bolsa;
DROP TABLE vec_bolsa_llamamientos.politica_cese_bolsa;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_relevo_cese;
RESET ROLE;
DO $fin$ BEGIN EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM vec_bolsa_llamamientos_relevo_cese',current_database()); END $fin$;
DROP ROLE vec_bolsa_llamamientos_relevo_cese;
COMMIT;
