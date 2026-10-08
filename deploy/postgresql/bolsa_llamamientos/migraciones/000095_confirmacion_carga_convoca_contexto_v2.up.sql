\set ON_ERROR_STOP on
-- B95: sustituye sólo la entrada web de B79 por confirmación ligada al
-- contexto exacto V3. La definición v1 instalada y toda su historia siguen
-- intactas; B93 puede consumir una vista previa, pero no confirmar una bolsa.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000095',0));
DO $pre$
DECLARE antigua oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        nueva oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        definicion text; fuente text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR antigua IS NULL OR nueva IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',antigua,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',antigua,'EXECUTE')
 THEN RAISE EXCEPTION 'B95: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.pg_get_functiondef(p.oid),p.prosrc INTO definicion,fuente
 FROM pg_catalog.pg_proc p WHERE p.oid=antigua
  AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
  AND p.prosecdef AND p.provolatile='v'
  AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=60s'];
 IF definicion IS NULL OR fuente IS NULL
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(definicion,'UTF8')),'hex')
    IS DISTINCT FROM '2fec62b7524293ee7e8c83fb79145a0aa8dac60978e95df1eb9d9365cde25bf5'
 OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(fuente,'UTF8')),'hex')
    IS DISTINCT FROM '9946d416eebc796f1aa200508ccafd5f738a29d697f9cabcbca15c7d99739d33'
 OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
     WHERE p.oid=antigua AND a.grantee NOT IN
       (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole))<>0
 THEN RAISE EXCEPTION 'B95: definición o ACL B79 instalada divergente' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.confirmar_carga_convoca_v2(p_acta jsonb, p_filas_cifradas jsonb, p_original_cifrado jsonb, p_acta_ref text, p_actor_ref text, p_categoria_ref text, p_bolsa_ref text, p_version_bolsa bigint, p_bolsa_canonica bytea, p_vigente_desde timestamp with time zone, p_instantanea_ref text, p_version_instantanea bigint, p_instantanea_canonica bytea, p_referida_en timestamp with time zone, p_generada_en timestamp with time zone, p_entradas jsonb, p_confirmada_en timestamp with time zone, p_vinculos jsonb, p_contexto_recurso bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS jsonb
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
 SET statement_timeout TO '15s'
 SET idle_in_transaction_session_timeout TO '20s'
AS $function$
DECLARE decision jsonb; bolsa jsonb; recurso jsonb; capacidad jsonb; consumo record; estado jsonb; acta_durable jsonb;
        carga record; recibo jsonb; vinculos_recibo jsonb; previo vec_bolsa_llamamientos.recibo_carga_convoca%ROWTYPE;
        huella_entradas text; huella_vinculos text; nueva boolean;
        ambito text; unidad text; canon text; huella_contexto text;
        registrada_db timestamptz; registrada_texto text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR pg_catalog.jsonb_typeof(p_acta) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_filas_cifradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(p_original_cifrado) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_entradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(p_vinculos) IS DISTINCT FROM 'array'
 OR p_acta_ref IS NULL OR p_acta_ref !~ '^acta:importacion-convoca:[0-9a-f]{64}$'
 OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 1 AND 256
 OR p_actor_ref<>pg_catalog.btrim(p_actor_ref)
 OR p_acta->>'acta_ref' IS DISTINCT FROM p_acta_ref
 OR p_acta->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR p_acta->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR p_acta->>'actor_ref' IS DISTINCT FROM 'actor:rrhh:'||pg_catalog.substr(
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||p_actor_ref,'UTF8')),'hex'),1,32)
 OR p_acta->>'huella_fichero_sha256' IS NULL
 OR p_acta->>'huella_fichero_sha256' !~ '^[0-9a-f]{64}$'
 OR p_acta_ref IS DISTINCT FROM 'acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to((p_acta->>'huella_fichero_sha256')||pg_catalog.chr(31)||p_categoria_ref,'UTF8')),'hex')
 OR p_contexto_recurso IS NULL OR pg_catalog.octet_length(p_contexto_recurso) NOT BETWEEN 1 AND 1024
 THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN
  decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  bolsa:=pg_catalog.convert_from(p_bolsa_canonica,'UTF8')::jsonb;
  recurso:=pg_catalog.convert_from(p_contexto_recurso,'UTF8')::jsonb;
  capacidad:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B79: material inválido' USING ERRCODE='42501'; END;
 IF pg_catalog.jsonb_typeof(recurso) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(recurso->'ambitos') IS DISTINCT FROM 'object'
 OR recurso->'atributos' IS DISTINCT FROM '{}'::jsonb
 OR pg_catalog.jsonb_typeof(capacidad) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B95: contexto de confirmación inválido' USING ERRCODE='42501'; END IF;
 ambito:=recurso#>>'{ambitos,ambito_ref}';
 unidad:=recurso#>>'{ambitos,unidad_ref}';
 IF ambito IS NULL OR pg_catalog.octet_length(ambito) NOT BETWEEN 1 AND 256
 OR ambito !~ '^[A-Za-z0-9:._/-]+$'
 OR unidad IS NULL OR pg_catalog.octet_length(unidad) NOT BETWEEN 1 AND 256
 OR unidad !~ '^[A-Za-z0-9:._/-]+$'
 THEN RAISE EXCEPTION 'B95: ámbito de confirmación inválido' USING ERRCODE='42501'; END IF;
 -- La preimagen V3 usa json.Marshal Go sobre dos mapas. La igualdad exacta
 -- de bytes impide que un contexto de vista previa (fase/filtro/página) autorice
 -- la confirmación y rechaza claves duplicadas o codificaciones alternativas.
 canon:='{"ambitos":{"ambito_ref":'||pg_catalog.to_json(ambito)::text||
  ',"unidad_ref":'||pg_catalog.to_json(unidad)::text||'},"atributos":{}}';
 IF p_contexto_recurso IS DISTINCT FROM pg_catalog.convert_to(canon,'UTF8')
 THEN RAISE EXCEPTION 'B95: contexto de confirmación no canónico' USING ERRCODE='42501'; END IF;
 huella_contexto:=pg_catalog.encode(pg_catalog.sha256(p_contexto_recurso),'hex');
 IF decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella_contexto
 OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM huella_contexto
 OR capacidad->>'efecto_ref' IS DISTINCT FROM p_acta_ref
 THEN RAISE EXCEPTION 'B95: contexto y decisión divergentes' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_typeof(decision) IS DISTINCT FROM 'object'
 OR decision->>'principal_id' IS DISTINCT FROM p_actor_ref
 OR decision->>'recurso_ref' IS DISTINCT FROM p_acta_ref
 OR pg_catalog.jsonb_typeof(bolsa) IS DISTINCT FROM 'object'
 OR bolsa->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR bolsa->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR bolsa->>'huella_listado_sha256' IS DISTINCT FROM p_acta->>'huella_fichero_sha256'
 THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_array_length(p_entradas)<1
 OR pg_catalog.jsonb_array_length(p_vinculos)>pg_catalog.jsonb_array_length(p_entradas)
 OR pg_catalog.jsonb_array_length(p_entradas)<>pg_catalog.jsonb_array_length(p_filas_cifradas)
 OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_entradas) e
     WHERE pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'object'
        OR e.value->>'participacion_ref' IS NULL OR e.value->>'fila_numero' IS NULL)
 OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_vinculos) v
     WHERE pg_catalog.jsonb_typeof(v.value) IS DISTINCT FROM 'object'
        OR v.value->>'participacion_ref' IS NULL OR v.value->>'candidato_ref' IS NULL)
 OR (SELECT pg_catalog.count(DISTINCT e.value->>'participacion_ref')
      FROM pg_catalog.jsonb_array_elements(p_entradas) e)<>pg_catalog.jsonb_array_length(p_entradas)
 OR (SELECT pg_catalog.count(DISTINCT e.value->>'fila_numero')
      FROM pg_catalog.jsonb_array_elements(p_entradas) e)<>pg_catalog.jsonb_array_length(p_entradas)
 OR (SELECT pg_catalog.count(DISTINCT v.value->>'participacion_ref')
      FROM pg_catalog.jsonb_array_elements(p_vinculos) v)<>pg_catalog.jsonb_array_length(p_vinculos)
 OR EXISTS (
   SELECT v.value->>'participacion_ref' FROM pg_catalog.jsonb_array_elements(p_vinculos) v
   EXCEPT SELECT e.value->>'participacion_ref' FROM pg_catalog.jsonb_array_elements(p_entradas) e)
 OR EXISTS (
   (SELECT e.value->>'fila_numero' FROM pg_catalog.jsonb_array_elements(p_entradas) e
    EXCEPT SELECT f.value->>'numero' FROM pg_catalog.jsonb_array_elements(p_filas_cifradas) f)
   UNION ALL
   (SELECT f.value->>'numero' FROM pg_catalog.jsonb_array_elements(p_filas_cifradas) f
    EXCEPT SELECT e.value->>'fila_numero' FROM pg_catalog.jsonb_array_elements(p_entradas) e))
 THEN RAISE EXCEPTION 'B79: filas y vínculos incompatibles' USING ERRCODE='22023'; END IF;
 huella_entradas:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_entradas::text,'UTF8')),'hex');
 huella_vinculos:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_vinculos::text,'UTF8')),'hex');
 -- La fachada AD218 exige consumo_nuevo=true; aquí no existe acceso por mera
 -- posesión de una referencia histórica. El reintento recibe otra decisión.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_acta_ref
 OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL OR consumo.consumida_en IS NULL
 THEN RAISE EXCEPTION 'B79: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Los locks internos de guardar_lote y B7 siguen el mismo orden en todos los
 -- intentos. En el replay se pasa el acta histórica para no cambiar custodia.
 estado:=vec_bolsa_importacion_convoca.consultar_estado_v1(
  p_acta->>'huella_fichero_sha256',p_categoria_ref);
 acta_durable:=COALESCE(estado->'acta',p_acta);
 IF acta_durable->>'acta_ref' IS DISTINCT FROM p_acta_ref
 OR acta_durable->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR acta_durable->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR acta_durable->>'actor_ref' IS DISTINCT FROM p_acta->>'actor_ref'
 OR acta_durable->>'nombre_fichero' IS DISTINCT FROM p_acta->>'nombre_fichero'
 OR acta_durable->>'fichero_custodiado_ref' IS DISTINCT FROM p_acta->>'fichero_custodiado_ref'
 OR acta_durable->>'huella_fichero_sha256' IS DISTINCT FROM p_acta->>'huella_fichero_sha256'
 THEN RAISE EXCEPTION 'B79: acta histórica incompatible' USING ERRCODE='23505'; END IF;
 SELECT * INTO STRICT carga FROM vec_bolsa_importacion_convoca.guardar_lote_v1(acta_durable,p_filas_cifradas);
 -- BIC3 fija registrada_en con el reloj PostgreSQL al crear el lote; Go
 -- aporta una fecha técnica previa. Todas las demás claves del acta deben
 -- permanecer idénticas, también actor, categoría, custodia y demás fechas.
 IF pg_catalog.jsonb_typeof(carga.acta_canonica) IS DISTINCT FROM 'object'
 OR carga.acta_canonica - 'registrada_en' IS DISTINCT FROM acta_durable - 'registrada_en'
 OR pg_catalog.jsonb_typeof(carga.acta_canonica->'registrada_en') IS DISTINCT FROM 'string'
 THEN
  RAISE EXCEPTION 'B79: acta durable divergente' USING ERRCODE='55000'; END IF;
 registrada_texto:=carga.acta_canonica->>'registrada_en';
 IF registrada_texto !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
 THEN RAISE EXCEPTION 'B95: fecha de registro durable inválida' USING ERRCODE='55000'; END IF;
 BEGIN
  registrada_db:=registrada_texto::timestamptz;
 EXCEPTION WHEN invalid_datetime_format OR datetime_field_overflow OR invalid_text_representation THEN
  RAISE EXCEPTION 'B95: fecha de registro durable inválida' USING ERRCODE='55000';
 END;
 IF NOT pg_catalog.isfinite(registrada_db)
 OR pg_catalog.to_char(registrada_db AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"') IS DISTINCT FROM registrada_texto
 THEN RAISE EXCEPTION 'B95: fecha de registro durable inválida' USING ERRCODE='55000'; END IF;
 PERFORM vec_bolsa_importacion_convoca.guardar_original_v1(carga.acta_canonica,p_original_cifrado);
 SELECT * INTO previo FROM vec_bolsa_llamamientos.recibo_carga_convoca WHERE acta_ref=p_acta_ref FOR SHARE;
 IF FOUND THEN
  IF previo.actor_ref<>p_actor_ref OR previo.categoria_ref<>p_categoria_ref OR previo.bolsa_ref<>p_bolsa_ref
  OR previo.huella_fichero_sha256<>p_acta->>'huella_fichero_sha256'
  OR previo.recibo_constitucion->>'version_bolsa' IS DISTINCT FROM p_version_bolsa::text
  OR previo.recibo_constitucion->>'instantanea_ref' IS DISTINCT FROM p_instantanea_ref
  OR previo.recibo_constitucion->>'version_instantanea' IS DISTINCT FROM p_version_instantanea::text
  OR previo.huella_entradas_sha256<>huella_entradas OR previo.huella_vinculos_sha256<>huella_vinculos
  THEN RAISE EXCEPTION 'B79: reintento incompatible' USING ERRCODE='23505'; END IF;
  RETURN previo.recibo_constitucion||pg_catalog.jsonb_build_object(
   'reutilizada',true,
   'decision_ref',previo.decision_ref,'auditoria_ref',previo.auditoria_ref,
   'consumida_en',pg_catalog.to_char(previo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'auditoria_acceso_ref',consumo.auditoria_ref,
   'decision_acceso_ref',consumo.decision_ref,
   'acta_reutilizada',true,'vinculos',previo.recibo_vinculos);
 END IF;
 recibo:=vec_bolsa_llamamientos.constituir_bolsa_v1(
  p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_version_bolsa,p_bolsa_canonica,p_vigente_desde,
  p_instantanea_ref,p_version_instantanea,p_instantanea_canonica,p_referida_en,p_generada_en,p_entradas,p_confirmada_en);
 IF pg_catalog.jsonb_typeof(recibo) IS DISTINCT FROM 'object'
 OR recibo->>'acta_ref' IS DISTINCT FROM p_acta_ref OR recibo->>'reutilizada' IS DISTINCT FROM 'false'
 THEN RAISE EXCEPTION 'B79: constitución previa sin recibo B1' USING ERRCODE='23505'; END IF;
 IF pg_catalog.jsonb_array_length(p_vinculos)>0 THEN
  vinculos_recibo:=vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(p_acta_ref,p_vinculos,p_confirmada_en);
 ELSE
  vinculos_recibo:=pg_catalog.jsonb_build_object('nuevos',0,'existentes',0);
 END IF;
 IF pg_catalog.jsonb_typeof(vinculos_recibo) IS DISTINCT FROM 'object'
 OR (vinculos_recibo->>'nuevos')::integer IS DISTINCT FROM pg_catalog.jsonb_array_length(p_vinculos)
 OR (vinculos_recibo->>'existentes')::integer IS DISTINCT FROM 0
 THEN RAISE EXCEPTION 'B79: vínculos divergentes' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.recibo_carga_convoca(
  acta_ref,actor_ref,categoria_ref,bolsa_ref,huella_fichero_sha256,
  huella_entradas_sha256,huella_vinculos_sha256,recibo_constitucion,recibo_vinculos,
  decision_ref,auditoria_ref,consumida_en)
 VALUES(p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_acta->>'huella_fichero_sha256',
  huella_entradas,huella_vinculos,recibo,vinculos_recibo,
  consumo.decision_ref,consumo.auditoria_ref,consumo.consumida_en);
 RETURN recibo||pg_catalog.jsonb_build_object(
  'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,
  'consumida_en',pg_catalog.to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'auditoria_acceso_ref',consumo.auditoria_ref,'decision_acceso_ref',consumo.decision_ref,
  'acta_reutilizada',carga.reutilizada,'vinculos',vinculos_recibo);
END $function$;

-- Se retira sólo la entrada web antigua. El propietario conserva v1 para
-- historia y el grupo CLI mantiene exclusivamente B7/B8.
DO $acl$
DECLARE antigua pg_catalog.regprocedure:='vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
        nueva pg_catalog.regprocedure:='vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
BEGIN
 EXECUTE pg_catalog.format('REVOKE EXECUTE ON FUNCTION %s FROM vec_bolsa_llamamientos_ejecutor',antigua);
 EXECUTE pg_catalog.format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',nueva);
 EXECUTE pg_catalog.format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_llamamientos_ejecutor',nueva);
 IF pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',antigua,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_desarrollo',antigua,'EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_ejecutor',nueva,'EXECUTE')
 OR pg_catalog.has_function_privilege('vec_bolsa_llamamientos_constituidor',nueva,'EXECUTE')
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=nueva
    AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
    AND p.prosecdef AND p.provolatile='v'
    AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s',
      'statement_timeout=15s','idle_in_transaction_session_timeout=20s'])
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
    pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE p.oid=nueva AND a.grantee NOT IN
     (p.proowner,'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole))
 THEN RAISE EXCEPTION 'B95: ACL o configuración divergente' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
