\set ON_ERROR_STOP on
-- B93: una vista previa valida el recurso y la pagina antes de consumir AD218.
-- El consumo y su auditoria son el unico efecto durable; la consulta no crea
-- acta, lote, original, bolsa ni vinculos de candidatos.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000093',0));

DO $pre$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR f IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',f,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(text,text,text,text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B93: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
 p_acta_ref text,p_actor_ref text,p_categoria_ref text,p_huella_fichero_sha256 text,
 p_contexto_recurso bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,
 p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET timezone='UTC'
 SET lock_timeout='2s' SET statement_timeout='15s'
 SET idle_in_transaction_session_timeout='20s' AS $f$
DECLARE recurso jsonb; decision jsonb; capacidad jsonb; consumo record;
        ambito text; unidad text; filtro text; limite text; desplazamiento text;
        canon text; huella text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 1 AND 256
 OR p_actor_ref<>pg_catalog.btrim(p_actor_ref)
 OR p_categoria_ref IS NULL OR pg_catalog.octet_length(p_categoria_ref) NOT BETWEEN 1 AND 256
 OR p_categoria_ref !~ '^[A-Za-z0-9:._/-]+$'
 OR p_huella_fichero_sha256 IS NULL OR p_huella_fichero_sha256 !~ '^[0-9a-f]{64}$'
 OR p_acta_ref IS DISTINCT FROM 'acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
    pg_catalog.convert_to(p_huella_fichero_sha256||pg_catalog.chr(31)||p_categoria_ref,'UTF8')),'hex')
 OR p_contexto_recurso IS NULL OR pg_catalog.octet_length(p_contexto_recurso) NOT BETWEEN 1 AND 2048
 THEN RAISE EXCEPTION 'B93: vista previa no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN
  recurso:=pg_catalog.convert_from(p_contexto_recurso,'UTF8')::jsonb;
  decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  capacidad:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'B93: material invalido' USING ERRCODE='42501';
 END;
 IF pg_catalog.jsonb_typeof(recurso) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(recurso->'atributos') IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(decision) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(capacidad) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B93: contexto invalido' USING ERRCODE='42501'; END IF;
 ambito:=recurso#>>'{ambitos,ambito_ref}';
 unidad:=recurso#>>'{ambitos,unidad_ref}';
 filtro:=recurso#>>'{atributos,filtro}';
 limite:=recurso#>>'{atributos,limite}';
 desplazamiento:=recurso#>>'{atributos,desplazamiento}';
 IF ambito IS NULL OR pg_catalog.octet_length(ambito) NOT BETWEEN 1 AND 256
 OR ambito !~ '^[A-Za-z0-9:._/-]+$'
 OR unidad IS NULL OR pg_catalog.octet_length(unidad) NOT BETWEEN 1 AND 256
 OR unidad !~ '^[A-Za-z0-9:._/-]+$'
 OR filtro IS NULL OR filtro NOT IN ('todas','aceptadas','rechazadas','con_avisos')
 OR limite IS NULL OR limite !~ '^[1-9][0-9]{0,2}$'
 OR desplazamiento IS NULL OR desplazamiento !~ '^(0|[1-9][0-9]{0,4})$'
 OR recurso#>>'{atributos,esquema}' IS DISTINCT FROM 'vec.bolsa.rrhh.carga_convoca.vista_previa.v1'
 OR recurso#>>'{atributos,fase}' IS DISTINCT FROM 'vista_previa'
 THEN RAISE EXCEPTION 'B93: pagina o ambito invalido' USING ERRCODE='42501'; END IF;
 IF limite::integer>100 OR desplazamiento::integer>20000
 THEN RAISE EXCEPTION 'B93: pagina fuera de limite' USING ERRCODE='42501'; END IF;
 -- json.Marshal de la estructura Go y de sus mapas ordena estas claves. La
 -- igualdad de bytes descarta espacios, otras claves, duplicados y escapes
 -- alternativos; nunca se calcula la huella sobre jsonb::text.
 canon:='{"ambitos":{"ambito_ref":'||pg_catalog.to_json(ambito)::text||
  ',"unidad_ref":'||pg_catalog.to_json(unidad)::text||
  '},"atributos":{"desplazamiento":'||pg_catalog.to_json(desplazamiento)::text||
  ',"esquema":"vec.bolsa.rrhh.carga_convoca.vista_previa.v1","fase":"vista_previa","filtro":'||
  pg_catalog.to_json(filtro)::text||',"limite":'||pg_catalog.to_json(limite)::text||'}}';
 IF p_contexto_recurso IS DISTINCT FROM pg_catalog.convert_to(canon,'UTF8')
 THEN RAISE EXCEPTION 'B93: contexto no canonico' USING ERRCODE='42501'; END IF;
 huella:=pg_catalog.encode(pg_catalog.sha256(p_contexto_recurso),'hex');
 IF decision->>'principal_id' IS DISTINCT FROM p_actor_ref
 OR decision->>'recurso_ref' IS DISTINCT FROM p_acta_ref
 OR decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
 OR capacidad->>'efecto_ref' IS DISTINCT FROM p_acta_ref
 OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM huella
 OR decision->>'accion' IS DISTINCT FROM 'bolsa.carga_convoca.confirmar'
 OR decision->>'finalidad' IS DISTINCT FROM 'carga_bolsa_convoca'
 OR capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.carga_convoca.confirmar.v1'
 OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
 OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B93: decision divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE
 OR consumo.decision_ref IS DISTINCT FROM decision->>'decision_ref'
 OR consumo.efecto_ref IS DISTINCT FROM p_acta_ref
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM huella
 OR consumo.auditoria_ref IS NULL OR consumo.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$'
 OR consumo.consumida_en IS NULL OR NOT pg_catalog.isfinite(consumo.consumida_en)
 THEN RAISE EXCEPTION 'B93: consumo divergente' USING ERRCODE='42501'; END IF;
 RETURN pg_catalog.jsonb_build_object(
  'decision_ref',consumo.decision_ref,'acta_ref',p_acta_ref,'huella_contexto',huella,
  'auditoria_ref',consumo.auditoria_ref,
  'consumida_en',pg_catalog.to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
 text,text,text,text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.autorizar_vista_previa_carga_convoca_v1(
 text,text,text,text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
