\set ON_ERROR_STOP on
-- BC17 / F1-95: una clave de publicación solo se recupera con el perfil activo
-- que quedó ligado a la declaración original. El consumo V3 sigue precediendo
-- al acceso al recibo y puede ser nuevo si la decisión fresca es válida.
-- Reconstrucción prospectiva de la función BC15 instalada, sin cambiar historia,
-- firma, dueño, configuración ni concesiones. Sin DOWN ni reaplicación.
BEGIN;
SET LOCAL ROLE vec_bolsa_convocatorias_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_convocatorias:migracion:000017',0));
DO $pre$
DECLARE f regprocedure; p record;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion') IS NULL
 THEN RAISE EXCEPTION 'BC17: pre incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'BC17: función ausente' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p
 FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario'
 OR p.prosecdef IS DISTINCT FROM true
 OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proparallel IS DISTINCT FROM 'u'
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_convocatorias_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'59e961bc0582a9c34893ed9e584191d2b696c57bc762cf1abf1afb65ebfaf092'
 THEN RAISE EXCEPTION 'BC17: pre de declaración divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE OR REPLACE FUNCTION vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(p jsonb, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
AS $function$
DECLARE ctx jsonb; huella text; contexto_sha text; consumo record; d jsonb; m jsonb;
 r jsonb; previa record; vinculo record; es_replay boolean;
 consumidor regprocedure;
BEGIN
 IF NOT vec_bolsa_convocatorias.sesion_bolsa_rrhh_valida_v1() THEN
  RAISE EXCEPTION 'BC15: declaración denegada' USING ERRCODE='42501'; END IF;
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'BC15: la declaración exige una transacción serializable' USING ERRCODE='55000'; END IF;
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR octet_length(p::text)>4096
 OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(p) k) IS DISTINCT FROM
    ARRAY['anuncio_ref','boletin_numero','clave_idempotencia','convocatoria_ref','direccion',
     'fecha_publicacion','huella_version_sha256','medio','sustituye_ref']
 OR jsonb_typeof(p->'convocatoria_ref') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p->'huella_version_sha256') IS DISTINCT FROM 'string'
 OR jsonb_typeof(p->'clave_idempotencia') IS DISTINCT FROM 'string'
 OR p->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]{8,128}$'
 THEN RAISE EXCEPTION 'BC15: solicitud inválida' USING ERRCODE='22023'; END IF;
 -- Sin consumidor en el núcleo no hay declaración: falla cerrada, sin efectos.
 IF has_schema_privilege('vec_autorizacion_atestada_v3','USAGE') IS NOT TRUE THEN
  RAISE EXCEPTION 'BC15: autorización de la declaración no disponible' USING ERRCODE='55000'; END IF;
 consumidor:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF consumidor IS NULL OR has_function_privilege(consumidor,'EXECUTE') IS NOT TRUE THEN
  RAISE EXCEPTION 'BC15: autorización de la declaración no disponible' USING ERRCODE='55000'; END IF;
 ctx:=vec_bolsa_convocatorias.contexto_publicacion_oficial_v1(p->>'convocatoria_ref');
 IF ctx->>'huella_version_sha256' IS DISTINCT FROM p->>'huella_version_sha256' THEN
  RAISE EXCEPTION 'BC15: versión no vigente' USING ERRCODE='B9601'; END IF;
 huella:=encode(sha256(convert_to(p::text,'UTF8')),'hex');
 -- Canon del recurso V3, igual que RecursoAutorizable en Go y B133.
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"ambito_ref":'||to_json(ctx->>'ambito_ref')::text||
  ',"unidad_ref":'||to_json(ctx->>'unidad_ref')::text||
  '},"atributos":{"material_sha256":"'||huella||'"}}','UTF8')),'hex');
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_publicacion_oficial_bases_v1(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 m:=convert_from(p_motivo,'UTF8')::jsonb->'referencia';
 IF consumo.efecto_ref IS DISTINCT FROM p->>'convocatoria_ref'
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM contexto_sha
 OR d->>'accion' IS DISTINCT FROM 'bolsa.convocatoria.publicacion_oficial.declarar'
 OR d->>'tipo_recurso' IS DISTINCT FROM 'version_convocatoria'
 OR d->>'recurso_ref' IS DISTINCT FROM p->>'convocatoria_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
 OR jsonb_typeof(m) IS DISTINCT FROM 'object'
 OR coalesce(m->>'catalogo_id','') !~ '^[a-z][a-z0-9._-]{0,127}$'
 OR coalesce(m->>'catalogo_version','') !~ '^[1-9][0-9]{0,9}$'
 OR coalesce(m->>'entrada_clave','') !~ '^[a-z][a-z0-9._-]{0,127}$'
 THEN RAISE EXCEPTION 'BC15: decisión ajena a la declaración' USING ERRCODE='42501'; END IF;
 SELECT x.* INTO previa FROM vec_bolsa_convocatorias.publicacion_oficial_bases x
  WHERE x.clave_idempotencia=p->>'clave_idempotencia';
 es_replay:=FOUND;
 IF NOT es_replay AND consumo.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'BC15: falta efecto del consumo' USING ERRCODE='42501'; END IF;
 r:=vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_v1(jsonb_build_object(
  'convocatoria_id',ctx->>'convocatoria_id','secuencia',(ctx->>'secuencia')::bigint,
  'huella_version_sha256',p->'huella_version_sha256','medio',p->'medio',
  'boletin_numero',p->'boletin_numero','anuncio_ref',p->'anuncio_ref',
  'fecha_publicacion',p->'fecha_publicacion','direccion',p->'direccion',
  'sustituye_ref',p->'sustituye_ref','declarada_por',d->'principal_id',
  'motivo_ref',(m->>'catalogo_id')||':'||(m->>'catalogo_version')||':'||(m->>'entrada_clave'),
  'clave_idempotencia',p->'clave_idempotencia'));
 IF (r->>'reutilizada')::boolean IS DISTINCT FROM es_replay THEN
  RAISE EXCEPTION 'BC15: estado de la declaración incoherente' USING ERRCODE='55000'; END IF;
 IF NOT es_replay THEN
  INSERT INTO vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion(
   publicacion_ref,decision_ref,consumo_huella_sha256,auditoria_ref,perfil_ref,registrada_en)
  VALUES(r->>'publicacion_ref',consumo.decision_ref,consumo.consumo_huella_sha256,
   consumo.auditoria_ref,d->>'perfil_activo_ref',(r->>'registrada_en')::timestamptz);
 END IF;
 SELECT a.* INTO vinculo FROM vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion a
  WHERE a.publicacion_ref=r->>'publicacion_ref';
 -- Una clave declarada fuera de esta vía no se recupera desde la web.
 IF NOT FOUND OR vinculo.perfil_ref IS DISTINCT FROM d->>'perfil_activo_ref' THEN
  RAISE EXCEPTION 'BC15: clave reutilizada con otra declaración' USING ERRCODE='23505'; END IF;
 -- Recibo mínimo: sin identificador interno de la convocatoria ni declarante.
 RETURN jsonb_build_object('publicacion_ref',r->>'publicacion_ref','recibo_ref',r->>'recibo_ref',
  'evento_ref',r->>'evento_ref','convocatoria_ref',p->>'convocatoria_ref',
  'secuencia',(ctx->>'secuencia')::bigint,'medio',r->>'medio','boletin_numero',r->'boletin_numero',
  'anuncio_ref',r->>'anuncio_ref','fecha_publicacion',r->>'fecha_publicacion',
  'direccion',r->'direccion','sustituye_ref',r->'sustituye_ref',
  'auditoria_ref',vinculo.auditoria_ref,'registrada_en',r->'registrada_en',
  'reutilizada',es_replay);
END $function$;

DO $post$
DECLARE f regprocedure; p record;
BEGIN
 IF current_user<>'vec_bolsa_convocatorias_propietario'
 OR current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases') IS NULL
 OR to_regclass('vec_bolsa_convocatorias.publicacion_oficial_bases_autorizacion') IS NULL
 THEN RAISE EXCEPTION 'BC17: post incompatible' USING ERRCODE='55000'; END IF;
 f:=to_regprocedure('vec_bolsa_convocatorias.declarar_publicacion_oficial_bases_rrhh_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f IS NULL THEN RAISE EXCEPTION 'BC17: función ausente' USING ERRCODE='55000'; END IF;
 SELECT proowner,prosecdef,provolatile,proparallel,proacl,proconfig INTO STRICT p
 FROM pg_proc WHERE oid=f;
 IF pg_get_userbyid(p.proowner)<>'vec_bolsa_convocatorias_propietario'
 OR p.prosecdef IS DISTINCT FROM true
 OR p.provolatile IS DISTINCT FROM 'v'
 OR p.proparallel IS DISTINCT FROM 'u'
 OR p.proacl::text IS DISTINCT FROM '{vec_bolsa_convocatorias_propietario=X/vec_bolsa_convocatorias_propietario,vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_convocatorias_propietario}'
 OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s']::text[]
 OR encode(sha256(convert_to(pg_get_functiondef(f),'UTF8')),'hex')<>'6f0fd898510d55b7f0662aa12d0b8990263d508b90216b953e02904ca185fbb0'
 THEN RAISE EXCEPTION 'BC17: post de declaración divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
