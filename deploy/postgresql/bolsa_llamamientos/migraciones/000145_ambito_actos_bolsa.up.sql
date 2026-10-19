\set ON_ERROR_STOP on
-- B145 (F1-60, segundo corte): los actos de Bolsa cotejan el ámbito de la bolsa.
--
-- * leer_ambito_bolsa_v1: única lectura del ejecutor sobre ambito_bolsa (B144).
--   El resolutor de contexto de Bolsa toma de aquí la unidad gestora y el
--   ámbito de la bolsa y los pone en el recurso que autoriza la decisión V3;
--   una bolsa sin fila se deniega.
-- * confirmar_aceptacion_directa_v1 (B128) coteja además en SQL el ámbito del
--   recurso autorizado con el de la bolsa y deniega con 42501 antes de consumir.
-- * confirmar_carga_convoca_v2 (B144) deniega también la recuperación de una
--   carga cuya bolsa existe sin ámbito declarado (antes devolvía el recibo).
-- * El rol constituidor (B80) pierde constituir_bolsa_v1,
--   registrar_vinculos_candidato_v1 y el uso del esquema: la constitución por
--   línea de órdenes, que no pasaba por la decisión V3 ni guardaba ámbito, se
--   retira y nada más los usaba. La carga gobernada (B95/B144) sigue
--   constituyendo como propietario. El rol queda sin miembros ni permisos en el
--   esquema; su CONNECT a la base solo lo puede retirar el administrador.
-- Definiciones finales explícitas, con huellas fijas de lo instalado y de lo
-- nuevo; firma, propietario, configuración y permisos se conservan.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000145',0));

DO $pre$
DECLARE f128 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        f95 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        f7 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)');
        fv oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)');
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR f128 IS NULL OR f95 IS NULL OR f7 IS NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.ambito_bolsa') IS NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.bolsa_constituida') IS NULL
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_ambito_bolsa_v1(text)') IS NOT NULL
 OR pg_catalog.to_regrole('vec_bolsa_llamamientos_constituidor') IS NULL
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f128
   AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')='0fc823991678b4dc04c1fc76383c4d571a3c844c2649312c403b7bd97d595931')
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f128) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f95
   AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')='03cf6fa9b2e2b46f6e50fb05be177ae2b7d5748fbae8b08bdd7e068733fb9fc0')
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f95) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f7) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_constituidor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR fv IS NULL
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=fv) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_constituidor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR NOT pg_catalog.has_schema_privilege('vec_bolsa_llamamientos_constituidor','vec_bolsa_llamamientos','USAGE')
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid='vec_bolsa_llamamientos_constituidor'::pg_catalog.regrole)
 THEN RAISE EXCEPTION 'B145: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

-- Lectura del ámbito gobernado de una bolsa para el resolutor de contexto.
CREATE FUNCTION vec_bolsa_llamamientos.leer_ambito_bolsa_v1(p_bolsa_ref text)
RETURNS TABLE(ambito_ref text, unidad_ref text) LANGUAGE sql STABLE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET row_security=on AS $f$
 SELECT a.ambito_ref,a.unidad_ref FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref
$f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.leer_ambito_bolsa_v1(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.leer_ambito_bolsa_v1(text) TO vec_bolsa_llamamientos_ejecutor;

-- Definición final explícita de B128 con el cotejo del ámbito de la bolsa.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(p_canon text, p_contexto_recurso bytea, p_capacidad bytea, p_decision bytea, p_motivo_autorizacion bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
 RETURNS TABLE(recibo jsonb, reutilizada boolean)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET lock_timeout TO '2s'
AS $function$
DECLARE q jsonb; x jsonb; d jsonb; c jsonb; consumo record; previo record; r record; l record; s record; politica record;
 h text; hc text; ha text; refhash text; recibo_ref text; resolucion_ref text; evento_ref text;
 esperado timestamptz; ahora timestamptz:=date_trunc('microseconds',clock_timestamp()); evento bytea;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER') IS NOT TRUE
 OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER') IS TRUE
 OR current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR (SELECT setting::bigint FROM pg_settings WHERE name='statement_timeout') NOT BETWEEN 1 AND 15000
 OR (SELECT setting::bigint FROM pg_settings WHERE name='idle_in_transaction_session_timeout') NOT BETWEEN 1 AND 20000
 OR p_canon IS NULL OR octet_length(p_canon) NOT BETWEEN 2 AND 16384
 OR p_contexto_recurso IS NULL OR octet_length(p_contexto_recurso) NOT BETWEEN 2 AND 2048 THEN
  RAISE EXCEPTION 'B128: acto denegado' USING ERRCODE='42501'; END IF;
 BEGIN q:=p_canon::jsonb; x:=convert_from(p_contexto_recurso,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb;
  esperado:=(q->>'situacion_desde_esperada')::timestamptz;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B128: material inválido' USING ERRCODE='22023'; END;
 IF q->>'esquema' IS DISTINCT FROM 'vec.bolsa.respuesta-revision.confirmar-aceptacion.v1'
 OR q->>'bolsa_ref' IS NULL OR octet_length(q->>'bolsa_ref') NOT BETWEEN 3 AND 256
 OR q->>'participacion_ref' IS NULL OR octet_length(q->>'participacion_ref') NOT BETWEEN 3 AND 512
 OR q->>'llamamiento_ref' IS NULL OR octet_length(q->>'llamamiento_ref') NOT BETWEEN 3 AND 512
 OR q->>'actor_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR q->>'destino' IS DISTINCT FROM 'pendiente_incorporacion'
 OR q->>'motivo' IS NULL OR octet_length(q->>'motivo') NOT BETWEEN 1 AND 1000 OR q->>'motivo' IS DISTINCT FROM btrim(q->>'motivo')
 OR octet_length(q->>'clave_idempotencia') NOT BETWEEN 8 AND 256
 OR q->>'clave_idempotencia' !~ '^[A-Za-z0-9:_-]+$'
 OR esperado IS NULL OR NOT isfinite(esperado)
 OR q->>'situacion_desde_esperada' IS DISTINCT FROM to_char(esperado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
 OR (SELECT count(*) FROM jsonb_object_keys(q))<>9 THEN
  RAISE EXCEPTION 'B128: comando inválido' USING ERRCODE='22023'; END IF;
 h:=encode(sha256(convert_to(p_canon,'UTF8')),'hex');
 IF jsonb_typeof(x#>'{ambitos}') IS DISTINCT FROM 'object'
 OR jsonb_typeof(x#>'{atributos}') IS DISTINCT FROM 'object'
 OR coalesce(x#>>'{ambitos,unidad_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$'
 OR coalesce(x#>>'{ambitos,ambito_ref}','') !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,255}$'
 OR x IS DISTINCT FROM jsonb_build_object('ambitos',jsonb_build_object(
  'ambito_ref',x#>>'{ambitos,ambito_ref}','unidad_ref',x#>>'{ambitos,unidad_ref}'),
  'atributos',jsonb_build_object('confirmacion_aceptacion_sha256',h))
 OR convert_from(p_contexto_recurso,'UTF8') IS DISTINCT FROM format(
  '{"ambitos":{"ambito_ref":%s,"unidad_ref":%s},"atributos":{"confirmacion_aceptacion_sha256":%s}}',
  to_json(x#>>'{ambitos,ambito_ref}'),to_json(x#>>'{ambitos,unidad_ref}'),to_json(h)) THEN
  RAISE EXCEPTION 'B128: ámbito no canónico' USING ERRCODE='42501'; END IF;
 hc:=encode(sha256(p_contexto_recurso),'hex');
 ha:=encode(sha256(convert_to(format('{"ambito_ref":%s,"unidad_ref":%s}',
  to_json(x#>>'{ambitos,ambito_ref}'),to_json(x#>>'{ambitos,unidad_ref}')),'UTF8')),'hex');
 -- B145 (F1-60): el ámbito autorizado debe ser el de la bolsa (B144). Una bolsa
 -- sin ámbito declarado o de otra unidad se deniega antes de consumir.
 IF NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a
  WHERE a.bolsa_ref=q->>'bolsa_ref' AND a.ambito_ref=x#>>'{ambitos,ambito_ref}'
   AND a.unidad_ref=x#>>'{ambitos,unidad_ref}') THEN
  RAISE EXCEPTION 'B145: bolsa sin ámbito o de otro ámbito' USING ERRCODE='42501'; END IF;
 IF d->>'principal_id' IS DISTINCT FROM q->>'actor_ref' OR d->>'accion' IS DISTINCT FROM 'bolsa.respuesta_revision.resolver'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa' OR d->>'tipo_recurso' IS DISTINCT FROM 'respuesta_revision_bolsa'
 OR d->>'finalidad' IS DISTINCT FROM 'resolver_respuestas_bolsa_rrhh'
 OR d->>'recurso_ref' IS DISTINCT FROM q->>'participacion_ref'
 OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM hc
 OR c->>'efecto_ref' IS DISTINCT FROM q->>'participacion_ref' THEN
  RAISE EXCEPTION 'B128: decisión no ligada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_resolucion_respuesta_revision_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo_autorizacion,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM q->>'participacion_ref'
 OR consumo.huella_efecto_sha256 IS DISTINCT FROM hc THEN
  RAISE EXCEPTION 'B128: consumo divergente' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:confirmacion:'||(q->>'participacion_ref'),0));
 SELECT * INTO previo FROM vec_bolsa_llamamientos.confirmacion_aceptacion_directa
  WHERE clave_idempotencia=q->>'clave_idempotencia';
 IF FOUND THEN
  IF previo.participacion_ref IS DISTINCT FROM q->>'participacion_ref' OR previo.bolsa_ref IS DISTINCT FROM q->>'bolsa_ref'
  OR previo.llamamiento_ref IS DISTINCT FROM q->>'llamamiento_ref' OR previo.actor_ref IS DISTINCT FROM q->>'actor_ref'
  OR previo.canon_sha256 IS DISTINCT FROM h OR previo.canon_original IS DISTINCT FROM p_canon
  OR previo.ambito_sha256 IS DISTINCT FROM ha OR previo.situacion_desde_esperada IS DISTINCT FROM esperado
  OR previo.evento_sha256 IS DISTINCT FROM encode(sha256(previo.evento_canonico),'hex')
  OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion z
   WHERE z.participacion_ref=previo.participacion_ref AND z.desde=previo.situacion_desde_nueva
    AND z.recibo_ref=previo.recibo_ref AND z.situacion='pendiente_incorporacion') THEN
   RAISE EXCEPTION 'B128: clave reutilizada' USING ERRCODE='VBS01'; END IF;
  RETURN QUERY SELECT jsonb_build_object('bolsa_ref',previo.bolsa_ref,'participacion_ref',previo.participacion_ref,
   'llamamiento_ref',previo.llamamiento_ref,'situacion_desde_esperada',to_char(previo.situacion_desde_esperada AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'motivo',previo.motivo,'recibo_ref',previo.recibo_ref,'situacion','pendiente_incorporacion',
   'desde',to_char(previo.situacion_desde_nueva AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'fecha_disponible',NULL,'resuelta_en',to_char(previo.resuelta_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'respuesta_ref',previo.respuesta_ref,'resolucion_ref',previo.resolucion_ref,'canon_sha256',previo.canon_sha256),true;
  RETURN;
 END IF;
 SELECT rp.* INTO r FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento rp
  WHERE rp.llamamiento_ref=q->>'llamamiento_ref' AND rp.participacion_ref=q->>'participacion_ref' FOR SHARE;
 IF NOT FOUND OR r.bolsa_ref IS DISTINCT FROM q->>'bolsa_ref' OR r.respuesta IS DISTINCT FROM 'acepta'
 OR r.respondida_en>ahora
 OR r.causa IS NOT NULL OR r.justificante_ref IS NOT NULL OR r.justificante_sha256 IS NOT NULL
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.confirmacion_aceptacion_directa y WHERE y.respuesta_ref=r.respuesta_ref) THEN
  RAISE EXCEPTION 'B128: respuesta incompatible o resuelta' USING ERRCODE='VBS01'; END IF;
 SELECT z.* INTO l FROM vec_bolsa_llamamientos.llamamiento_emitido z WHERE z.llamamiento_ref=r.llamamiento_ref FOR SHARE;
 IF NOT FOUND OR l.bolsa_ref IS DISTINCT FROM r.bolsa_ref OR NOT(l.participaciones ? r.participacion_ref)
 OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.constitucion c0
  JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
  WHERE c0.bolsa_ref=r.bolsa_ref AND e.participacion_ref=r.participacion_ref) THEN
  RAISE EXCEPTION 'B128: llamamiento o bolsa ajenos' USING ERRCODE='42501'; END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:situacion:'||r.participacion_ref,0));
 SELECT z.* INTO s FROM vec_bolsa_llamamientos.situacion_participacion z
  WHERE z.participacion_ref=r.participacion_ref ORDER BY z.desde DESC LIMIT 1 FOR UPDATE;
 IF NOT FOUND OR s.desde IS DISTINCT FROM esperado OR s.desde>ahora
 OR EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion z
  WHERE z.participacion_ref=r.participacion_ref AND z.registrada_en>=r.respondida_en AND z.desde<=ahora) THEN
  RAISE EXCEPTION 'B128: situación cambiada' USING ERRCODE='VBS02'; END IF;
 PERFORM pg_advisory_xact_lock_shared(hashtextextended('vec_bolsa_llamamientos:politica_transiciones_situacion',0));
 SELECT p.*,g.ambito_sha256 INTO politica FROM
  (SELECT * FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1) p
 JOIN vec_bolsa_llamamientos.politica_transiciones_gobierno g USING(version);
 IF NOT FOUND OR politica.ambito_sha256 IS DISTINCT FROM ha
 OR NOT((s.situacion||'>pendiente_incorporacion')=ANY(politica.transiciones)) THEN
  RAISE EXCEPTION 'B128: política cerrada o ámbito ajeno' USING ERRCODE='55000'; END IF;
 IF ahora<=s.desde THEN ahora:=s.desde+interval '1 microsecond'; END IF;
 refhash:=encode(sha256(convert_to(r.respuesta_ref||chr(31)||(q->>'clave_idempotencia'),'UTF8')),'hex');
 recibo_ref:='recibo:respuesta-revision:'||encode(sha256(convert_to(r.participacion_ref||chr(31)||(q->>'clave_idempotencia'),'UTF8')),'hex');
 resolucion_ref:='resolucion:respuesta-revision:'||refhash; evento_ref:='evento:respuesta-revision:'||refhash;
 INSERT INTO vec_bolsa_llamamientos.situacion_participacion
  (participacion_ref,situacion,desde,hasta,fecha_disponible,motivo,actor,registrada_en,clave_idempotencia,recibo_ref,politica_transiciones_version)
 VALUES(r.participacion_ref,'pendiente_incorporacion',ahora,NULL,NULL,q->>'motivo',q->>'actor_ref',ahora,
  'confirmacion:'||refhash,recibo_ref,politica.version);
 evento:=convert_to(jsonb_build_object('esquema','vec.bolsa.respuesta-revision.confirmada.v1',
  'evento_ref',evento_ref,'respuesta_ref',r.respuesta_ref,'resolucion_ref',resolucion_ref,
  'participacion_ref',r.participacion_ref,'situacion','pendiente_incorporacion','recibo_ref',recibo_ref)::text,'UTF8');
 INSERT INTO vec_bolsa_llamamientos.confirmacion_aceptacion_directa
  (respuesta_ref,resolucion_ref,bolsa_ref,participacion_ref,llamamiento_ref,actor_ref,motivo,clave_idempotencia,
   canon_original,canon_sha256,ambito_sha256,situacion_desde_esperada,situacion_desde_nueva,politica_version,
   recibo_ref,decision_ref,auditoria_ref,evento_ref,evento_canonico,evento_sha256,resuelta_en)
 VALUES(r.respuesta_ref,resolucion_ref,r.bolsa_ref,r.participacion_ref,r.llamamiento_ref,q->>'actor_ref',q->>'motivo',
  q->>'clave_idempotencia',p_canon,h,ha,esperado,ahora,politica.version,recibo_ref,consumo.decision_ref,
  consumo.auditoria_ref,evento_ref,evento,encode(sha256(evento),'hex'),ahora);
 RETURN QUERY SELECT jsonb_build_object('bolsa_ref',r.bolsa_ref,'participacion_ref',r.participacion_ref,
  'llamamiento_ref',r.llamamiento_ref,'situacion_desde_esperada',to_char(esperado AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'motivo',q->>'motivo','recibo_ref',recibo_ref,'situacion','pendiente_incorporacion',
  'desde',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'fecha_disponible',NULL,'resuelta_en',to_char(ahora AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'respuesta_ref',r.respuesta_ref,'resolucion_ref',resolucion_ref,'canon_sha256',h),false;
END $function$;

-- Definición final explícita de B95/B144 con la denegación de la bolsa sin ámbito.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.confirmar_carga_convoca_v2(p_acta jsonb, p_filas_cifradas jsonb, p_original_cifrado jsonb, p_acta_ref text, p_actor_ref text, p_categoria_ref text, p_bolsa_ref text, p_version_bolsa bigint, p_bolsa_canonica bytea, p_vigente_desde timestamp with time zone, p_instantanea_ref text, p_version_instantanea bigint, p_instantanea_canonica bytea, p_referida_en timestamp with time zone, p_generada_en timestamp with time zone, p_entradas jsonb, p_confirmada_en timestamp with time zone, p_vinculos jsonb, p_contexto_recurso bytea, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
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
 -- B144: el ámbito de la bolsa se coteja antes de consumir la decisión y de
 -- escribir, con el mismo cerrojo que declarar_ambito_bolsa_v1.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_bolsa_llamamientos:ambito_bolsa:'||p_bolsa_ref,0));
 -- B145: una bolsa que ya existe sin ámbito no se confirma ni se recupera; su
 -- ámbito solo llega por declaración de Administración (declarar_ambito_bolsa_v1).
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b WHERE b.bolsa_ref=p_bolsa_ref)
 AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref)
 THEN RAISE EXCEPTION 'B145: bolsa sin ámbito declarado' USING ERRCODE='42501'; END IF;
 -- Una bolsa con ámbito solo se confirma o recupera desde ese mismo ámbito.
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref
   AND (a.ambito_ref IS DISTINCT FROM ambito OR a.unidad_ref IS DISTINCT FROM unidad))
 -- Una constitución nueva no da ámbito a una bolsa que ya existía sin él, ni
 -- sustituye (B14, por categoría) una bolsa vigente de otro ámbito o sin él.
 OR (NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.recibo_carga_convoca r WHERE r.acta_ref=p_acta_ref)
  AND (
   (EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.bolsa_constituida b WHERE b.bolsa_ref=p_bolsa_ref)
    AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref))
   OR EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.constitucion c
     WHERE c.categoria_ref=p_categoria_ref AND c.bolsa_ref<>p_bolsa_ref
       AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.sustitucion_bolsa s
         WHERE s.bolsa_ref_sustituida=c.bolsa_ref AND s.version_sustituida=c.version_bolsa
           AND s.huella_sustituida_sha256=c.huella_bolsa_sha256)
       AND NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a
         WHERE a.bolsa_ref=c.bolsa_ref AND a.ambito_ref=ambito AND a.unidad_ref=unidad))))
 THEN RAISE EXCEPTION 'B144: bolsa de otro ámbito' USING ERRCODE='42501'; END IF;
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
 -- B144: el ámbito autorizado de la constitución queda como dato gobernado de
 -- la bolsa en la misma transacción. Si ya existía, debe coincidir.
 INSERT INTO vec_bolsa_llamamientos.ambito_bolsa(bolsa_ref,ambito_ref,unidad_ref,origen,fuente_ref)
 VALUES(p_bolsa_ref,ambito,unidad,'constitucion',p_acta_ref) ON CONFLICT (bolsa_ref) DO NOTHING;
 IF NOT EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.ambito_bolsa a WHERE a.bolsa_ref=p_bolsa_ref
   AND a.ambito_ref=ambito AND a.unidad_ref=unidad)
 THEN RAISE EXCEPTION 'B144: bolsa de otro ámbito' USING ERRCODE='42501'; END IF;
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

-- Se retira la constitución por línea de órdenes (B80).
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz) FROM vec_bolsa_llamamientos_constituidor;
REVOKE EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz) FROM vec_bolsa_llamamientos_constituidor;
REVOKE USAGE ON SCHEMA vec_bolsa_llamamientos FROM vec_bolsa_llamamientos_constituidor;

DO $post$
DECLARE f128 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_aceptacion_directa_v1(text,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        f95 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v2(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
        f7 oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)');
        fv oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)');
        fl oid:=pg_catalog.to_regprocedure('vec_bolsa_llamamientos.leer_ambito_bolsa_v1(text)');
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f128
   AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on','TimeZone=UTC','lock_timeout=2s']
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')='594dcf59e5935ee0cbd92a4f1d36462407c48f678888be490a6ae592841391a4')
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f128) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=f95
   AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','TimeZone=UTC','lock_timeout=2s','statement_timeout=15s','idle_in_transaction_session_timeout=20s']
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')='428448a0c047a078e6c0d0e96ea957554b86c8641e91952410b7ff04c56ff10a')
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f95) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=f7) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_propietario:EXECUTE']
 OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE p.oid=fl
   AND p.proowner='vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
   AND p.prosecdef AND p.provolatile='s'
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','row_security=on']
   AND pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(p.oid),'UTF8')),'hex')='8d2bfd66e0b117eaa36a78533d1cb18e13445d6b497591c4f71b340aa50bda5d')
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=fl) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_ejecutor:EXECUTE','vec_bolsa_llamamientos_propietario:EXECUTE']
 OR (SELECT pg_catalog.array_agg(a.grantee::pg_catalog.regrole::text||':'||a.privilege_type
      ORDER BY a.grantee::pg_catalog.regrole::text||':'||a.privilege_type)
     FROM pg_catalog.pg_proc p CROSS JOIN LATERAL
     pg_catalog.aclexplode(coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a WHERE p.oid=fv) IS DISTINCT FROM ARRAY['vec_bolsa_llamamientos_propietario:EXECUTE']
 OR pg_catalog.has_schema_privilege('vec_bolsa_llamamientos_constituidor','vec_bolsa_llamamientos','USAGE')
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.roleid='vec_bolsa_llamamientos_constituidor'::pg_catalog.regrole)
 THEN RAISE EXCEPTION 'B145: posimagen divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
