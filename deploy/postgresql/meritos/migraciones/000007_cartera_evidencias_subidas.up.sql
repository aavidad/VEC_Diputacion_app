\set ON_ERROR_STOP on
-- RUM7: la cartera propia proyecta las referencias de evidencias ya guardadas
-- en la versión del hecho. No añade escritura ni altera la historia.
BEGIN;
SET LOCAL ROLE vec_meritos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000007',0));
DO $pre$
BEGIN
 IF encode(sha256(convert_to(pg_get_functiondef(
  'vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb)'::regprocedure),'UTF8')),'hex') IS DISTINCT FROM
  'fc93ab0cb8527a368d4e953a9280952af39f20c92d8aeba3a4976645c168be4b'
 THEN RAISE EXCEPTION 'RUM7: cuerpo instalado divergente' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE OR REPLACE FUNCTION vec_meritos.listar_cartera_propia_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE a record; persona text;limite integer;filtro text;cursor text;corte timestamptz(6);
 total bigint;datos jsonb;siguiente text;recibo text;ahora timestamptz(6);
 motivos_catalogo jsonb;
BEGIN
 IF current_user<>'vec_meritos_propietario' OR session_user=current_user
 OR session_user<>'vec_externo_meritos_cartera'
 OR NOT EXISTS(SELECT 1 FROM pg_auth_members m
  WHERE m.member=session_user::regrole AND m.roleid='vec_meritos_cartera_externo'::regrole
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
 OR (SELECT count(*) FROM pg_auth_members WHERE member=session_user::regrole)<>1
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (p_orden ?& ARRAY['esquema','persona_ref','limite','filtro','cursor_hecho_ref',
  'corte_en','correlacion_ref','idioma']) IS NOT TRUE
 OR p_orden->>'esquema' IS DISTINCT FROM 'vec.meritos.cartera.propia.v1'
 OR p_orden->>'filtro' IS NULL
 OR p_orden->>'filtro' NOT IN ('todos','disponibles','retirados')
 OR jsonb_typeof(p_orden->'limite') IS DISTINCT FROM 'number'
 OR coalesce(p_orden->>'limite' ~ '^([1-9]|[1-4][0-9]|50)$',false) IS NOT TRUE
 OR vec_meritos.referencia_valida_v1(p_orden->>'persona_ref') IS NOT TRUE
 OR jsonb_typeof(p_orden->'cursor_hecho_ref') IS DISTINCT FROM 'string'
 OR p_orden->>'idioma' NOT IN ('es','en') OR p_orden->>'idioma' IS NULL
 THEN RAISE EXCEPTION 'M4: consulta inválida o sin ejecutor' USING ERRCODE='42501'; END IF;
 persona:=p_orden->>'persona_ref';limite:=(p_orden->>'limite')::integer;
 filtro:=p_orden->>'filtro';cursor:=coalesce(p_orden->>'cursor_hecho_ref','');
 BEGIN corte:=(p_orden->>'corte_en')::timestamptz(6);
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'M4: corte inválido' USING ERRCODE='22023'; END;
 IF corte IS NULL OR corte>clock_timestamp() OR corte<clock_timestamp()-interval '30 days'
 OR (cursor<>'' AND vec_meritos.referencia_valida_v1(cursor) IS NOT TRUE)
 THEN RAISE EXCEPTION 'M4: corte o cursor inválido' USING ERRCODE='22023'; END IF;
 -- El registrador común verifica contexto, sesión, permiso vivo y titularidad,
 -- y asienta la lectura positiva antes de proyectar datos personales.
 SELECT * INTO STRICT a FROM vec_autorizacion_atestada_v3.registrar_lectura_cartera_meritos_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF a.auditoria_ref IS NULL OR a.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
 OR a.secuencia IS NULL OR a.huella_sha256 !~ '^[0-9a-f]{64}$'
 OR a.registrada_en IS NULL
 THEN RAISE EXCEPTION 'M4: auditoría divergente' USING ERRCODE='42501'; END IF;
 motivos_catalogo:=vec_autorizacion.listar_motivos_meritos_cartera_actual_v1(p_orden->>'idioma');
 IF motivos_catalogo->>'idioma' IS DISTINCT FROM p_orden->>'idioma'
 OR jsonb_typeof(motivos_catalogo->'motivos') IS DISTINCT FROM 'array'
 OR jsonb_array_length(motivos_catalogo->'motivos') NOT BETWEEN 1 AND 128
 THEN RAISE EXCEPTION 'M4: motivos no disponibles' USING ERRCODE='55000';END IF;
 PERFORM set_config('vec_meritos.persona_ref',persona,true);
 WITH ultimas AS MATERIALIZED (
  SELECT DISTINCT ON (hv.hecho_ref COLLATE "C")
   hv.hecho_ref,hv.version,hv.registro->'hecho' AS hecho
  FROM vec_meritos.hecho_version hv
  WHERE hv.persona_ref=persona AND hv.registrada_en<=corte
  ORDER BY hv.hecho_ref COLLATE "C",hv.version DESC
 ),base AS MATERIALIZED (
  SELECT u.hecho_ref,u.version,u.hecho,d.estado AS retirada_estado,
   d.version AS disponibilidad_version,d.recibo_ref,d.retirada_en
  FROM ultimas u LEFT JOIN vec_meritos.disponibilidad_evento_v1 d
   ON d.hecho_ref=u.hecho_ref AND d.persona_ref=persona AND d.retirada_en<=corte
  WHERE filtro='todos' OR (filtro='retirados')=(d.hecho_ref IS NOT NULL)
 ),pagina AS (
  SELECT * FROM base WHERE hecho_ref COLLATE "C">cursor COLLATE "C"
  ORDER BY hecho_ref COLLATE "C" LIMIT limite+1
 )
 SELECT (SELECT count(*) FROM base),
  (SELECT coalesce(jsonb_agg(jsonb_build_object(
   'hecho_ref',p.hecho_ref,'version_hecho',p.version,
   'denominacion',p.hecho->>'denominacion','tipo',p.hecho->>'tipo',
   'disponibilidad',jsonb_build_object('estado',coalesce(p.retirada_estado,'disponible'),
     'version',coalesce(p.disponibilidad_version,0)),
   'retirada',CASE WHEN p.retirada_estado IS NULL THEN NULL ELSE
    jsonb_build_object('recibo_ref',p.recibo_ref,'fecha',to_char(p.retirada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')) END)
   || CASE WHEN p_orden->'campos_permitidos' ? 'evidencias'
    THEN jsonb_build_object('evidencias',coalesce(nullif(p.hecho->'evidencias','null'::jsonb),'[]'::jsonb))
    ELSE '{}'::jsonb END ORDER BY p.hecho_ref COLLATE "C"),'[]'::jsonb) FROM (SELECT * FROM pagina ORDER BY hecho_ref COLLATE "C" LIMIT limite) p),
  (SELECT CASE WHEN count(*)>limite THEN (SELECT hecho_ref FROM pagina ORDER BY hecho_ref COLLATE "C" OFFSET limite-1 LIMIT 1) ELSE NULL END FROM pagina)
 INTO total,datos,siguiente;
 ahora:=a.registrada_en;recibo:='cartera:'||gen_random_uuid()::text;
 INSERT INTO vec_meritos.cartera_acceso_v1 VALUES(recibo,persona,a.auditoria_ref,a.correlacion_ref,filtro,corte,total,ahora);
 RETURN jsonb_build_object('hechos',datos,'total',total,'cursor_siguiente_hecho_ref',coalesce(siguiente,''),
  'motivos_retirada',motivos_catalogo->'motivos',
  'corte_en',to_char(corte AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'recibo_acceso',jsonb_build_object('auditoria_ref',a.auditoria_ref,
   'secuencia',a.secuencia::text,'huella_sha256',a.huella_sha256,
   'correlacion_ref',a.correlacion_ref,'consultada_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $function$;
COMMIT;
