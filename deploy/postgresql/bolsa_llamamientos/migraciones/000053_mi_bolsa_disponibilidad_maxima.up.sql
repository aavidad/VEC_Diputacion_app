\set ON_ERROR_STOP on
-- Bolsa 000053: Mi Bolsa proyecta la fecha máxima entre la pausa propia B2 y
-- la restricción global B45. Son historias distintas; no se altera ninguna.
-- B45 ya está instalada y puede tener recibos: no reaplicar ni modificarla.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000053',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.situacion_participacion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.estado_cese_bolsa_v1(text,timestamptz)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL THEN
  RAISE EXCEPTION 'Bolsa 000053: dependencias B2/B45 incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;

DO $proyeccion$
DECLARE v_oid oid:='vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
 SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
 WHERE p.oid=v_oid AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
 FOR v_cambio IN SELECT * FROM (VALUES
  ($antes$'estado',CASE WHEN cese.en_restriccion
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN 'disponible_desde' WHEN cese.trabajo_cesado THEN 'disponible'
       ELSE situacion.situacion END,$antes$,
   $nuevo$'estado',CASE WHEN situacion.situacion IN ('disponible','trabajando','disponible_desde')
       AND plazo.disponible_en>p_consultada_en THEN 'disponible_desde'
       WHEN cese.trabajo_cesado OR situacion.situacion='disponible_desde' THEN 'disponible'
       ELSE situacion.situacion END,$nuevo$),
  ($antes$'desde',to_char((CASE WHEN (cese.en_restriccion OR cese.trabajo_cesado)
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN cese.fecha_efecto::timestamp AT TIME ZONE 'Europe/Madrid' ELSE situacion.desde END)
       AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),$antes$,
   $nuevo$'desde',to_char((CASE
       WHEN cese.en_restriccion AND plazo.disponible_en>p_consultada_en
         AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
         AND (situacion.fecha_disponible IS NULL OR plazo.cese_disponible_en>situacion.fecha_disponible)
         THEN greatest(situacion.desde,cese.fecha_efecto::timestamp AT TIME ZONE 'Europe/Madrid')
       WHEN plazo.disponible_en<=p_consultada_en AND plazo.disponible_en IS NOT NULL
         AND (cese.trabajo_cesado OR situacion.situacion='disponible_desde')
         THEN greatest(situacion.desde,plazo.disponible_en)
       ELSE situacion.desde END)
       AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),$nuevo$),
  ($antes$'fecha_disponible',CASE WHEN cese.en_restriccion
       AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN to_char((cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid')
         AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       WHEN situacion.fecha_disponible IS NULL THEN NULL
       ELSE to_char(situacion.fecha_disponible AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') END$antes$,
   $nuevo$'fecha_disponible',CASE
       WHEN situacion.situacion IN ('disponible','trabajando','disponible_desde')
         AND plazo.disponible_en>p_consultada_en
         THEN to_char(plazo.disponible_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
       ELSE NULL END$nuevo$),
  ($antes$) cese ON true
 LEFT JOIN LATERAL ($antes$,
   $nuevo$) cese ON true
 LEFT JOIN LATERAL (
   SELECT greatest(
     CASE WHEN situacion.situacion='disponible_desde' THEN situacion.fecha_disponible END,
     CASE WHEN (cese.en_restriccion OR cese.trabajo_cesado)
            AND situacion.situacion IN ('disponible','trabajando','disponible_desde')
       THEN cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid' END
   ) AS disponible_en,
   cese.disponible_desde::timestamp AT TIME ZONE 'Europe/Madrid' AS cese_disponible_en
 ) plazo ON true
 LEFT JOIN LATERAL ($nuevo$)
 ) AS x(antes,nuevo) LOOP
  IF length(v_def)-length(replace(v_def,v_cambio.antes,''))<>length(v_cambio.antes) THEN
   RAISE EXCEPTION 'Bolsa 000053: proyección Mi Bolsa incompatible' USING ERRCODE='55000';
  END IF;
  v_def:=replace(v_def,v_cambio.antes,v_cambio.nuevo);
 END LOOP;
 EXECUTE v_def;
 IF pg_get_functiondef(v_oid) IS DISTINCT FROM v_def
    OR (SELECT proacl FROM pg_proc WHERE oid=v_oid) IS DISTINCT FROM v_acl THEN
  RAISE EXCEPTION 'Bolsa 000053: Mi Bolsa alterada fuera de contrato' USING ERRCODE='55000';
 END IF;
END $proyeccion$;
COMMIT;
