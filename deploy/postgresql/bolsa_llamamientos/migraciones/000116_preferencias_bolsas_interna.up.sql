\set ON_ERROR_STOP on
-- B116: fachadas propias de Mi Bolsa interna sobre la historia B104.
-- Requiere el consumidor interno de AD279. AD275 instala después la lectura auditada.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='45s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000116',0));
DO $pre$
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regrole('vec_bolsa_llamamientos_preferencias_interna') IS NOT NULL
 OR to_regclass('vec_bolsa_llamamientos.preferencia_bolsas_version') IS NULL
 OR to_regclass('vec_bolsa_llamamientos.preferencia_bolsas_outbox') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR has_function_privilege('vec_bolsa_llamamientos_propietario',
  to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'),'EXECUTE') IS NOT TRUE
 OR to_regprocedure('vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)') IS NOT NULL
 OR to_regprocedure('vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B116: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
CREATE ROLE vec_bolsa_llamamientos_preferencias_interna NOLOGIN NOSUPERUSER
 NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_preferencias_interna;
-- La historia B104 compartida distingue las claves internas sin cambiar
-- filas antiguas. B104 rechaza '@' antes de consultar la clave por su regex.
DO $clave$
DECLARE nombre text; definicion text; total integer;
BEGIN
 SELECT count(*),min(c.conname),min(pg_get_constraintdef(c.oid,false))
 INTO total,nombre,definicion FROM pg_constraint c
 WHERE c.conrelid='vec_bolsa_llamamientos.preferencia_bolsas_version'::regclass
  AND c.contype='c' AND c.convalidated
  AND strpos(pg_get_constraintdef(c.oid,false),'clave_idempotencia')>0;
 IF total<>1 OR strpos(definicion,'^[A-Za-z0-9._:-]{8,128}$')=0
 OR EXISTS(SELECT 1 FROM pg_attribute a
  WHERE a.attrelid='vec_bolsa_llamamientos.preferencia_bolsas_version'::regclass
   AND a.attname IN ('actor_ref_interno','perfil_ref_interno') AND NOT a.attisdropped)
 THEN RAISE EXCEPTION 'B116: preimagen de clave divergente' USING ERRCODE='55000'; END IF;
 EXECUTE format('ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version DROP CONSTRAINT %I',nombre);
END $clave$;
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version
 ADD COLUMN actor_ref_interno text, ADD COLUMN perfil_ref_interno text;
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version
 ADD CONSTRAINT preferencia_bolsas_version_clave_superficie_v1 CHECK (
  clave_idempotencia ~ '^[A-Za-z0-9._:-]{8,128}$'
  OR clave_idempotencia ~ '^@[0-9a-f]{64}$');
ALTER TABLE vec_bolsa_llamamientos.preferencia_bolsas_version
 ADD CONSTRAINT preferencia_bolsas_version_actor_interno_v1 CHECK (
  (clave_idempotencia ~ '^@[0-9a-f]{64}$'
   AND actor_ref_interno IS NOT NULL AND perfil_ref_interno IS NOT NULL
   AND octet_length(actor_ref_interno) BETWEEN 1 AND 256
   AND octet_length(perfil_ref_interno) BETWEEN 1 AND 256)
  OR (clave_idempotencia !~ '^@[0-9a-f]{64}$'
   AND actor_ref_interno IS NULL AND perfil_ref_interno IS NULL));
CREATE FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(
 p_candidato_ref text,p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_estado jsonb; v_auditoria record; v_corte timestamptz(6):=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user <> 'vec_interno_bolsa_preferencias_desarrollo'
    OR current_setting('transaction_isolation') <> 'serializable'
    OR current_setting('transaction_read_only') <> 'off'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
    OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
    OR p_orden->>'recurso_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B116: lectura nominal inválida' USING ERRCODE='42501'; END IF;
 v_estado:=vec_bolsa_llamamientos.preferencias_bolsas_estado_interno_v1(p_candidato_ref,v_corte);
 SELECT * INTO STRICT v_auditoria
 FROM vec_autorizacion_atestada_v3.registrar_lectura_preferencias_bolsas_interna_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden);
 IF v_estado IS NULL OR jsonb_typeof(v_estado) IS DISTINCT FROM 'object'
    OR v_auditoria.auditoria_ref IS NULL OR v_auditoria.huella_sha256 !~ '^[0-9a-f]{64}$'
    OR v_auditoria.correlacion_ref IS DISTINCT FROM p_orden->>'correlacion_ref'
    OR v_auditoria.registrada_en IS NULL
 THEN RAISE EXCEPTION 'B116: lectura no asentada' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('estado',v_estado,'auditoria_ref',v_auditoria.auditoria_ref,
  'auditoria_huella_sha256',v_auditoria.huella_sha256,
  'correlacion_ref',v_auditoria.correlacion_ref,'auditada_en',v_auditoria.registrada_en);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_preferencias_interna;

-- Fallo posterior a la autenticación. La misma sesión LOGIN entra por la
-- fachada Bolsa; AD275 acredita CA/AUT y registra el intento nominal común.
-- No concede al LOGIN acceso directo a AD3 ni consulta la concesión del GET.
CREATE FUNCTION vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(
 p_candidato_ref text,p_contexto_canonico bytea,p_vinculo_canonico bytea,p_orden jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,
 correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='10s' AS $f$
DECLARE v_contexto jsonb;v_candidato text;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR session_user<>'vec_interno_bolsa_preferencias_desarrollo'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
 OR p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR p_orden->>'recurso_ref' IS DISTINCT FROM
  'mi-bolsa:'||encode(sha256(convert_to('mi-bolsa:'||p_candidato_ref,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B116: intento nominal inválido' USING ERRCODE='42501'; END IF;
 BEGIN
  v_contexto:=convert_from(p_contexto_canonico,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'B116: contexto inválido' USING ERRCODE='42501'; END;
 SELECT x->>'referencia' INTO v_candidato FROM jsonb_array_elements(v_contexto->'vinculos') x
 WHERE x->>'tipo'='candidato' AND x->>'estado'='activo';
 IF (SELECT count(*) FROM jsonb_array_elements(v_contexto->'vinculos') x
     WHERE x->>'tipo'='candidato' AND x->>'estado'='activo')<>1
 OR v_candidato IS DISTINCT FROM p_candidato_ref
 THEN RAISE EXCEPTION 'B116: candidato de intento denegado' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en
 FROM vec_autorizacion_atestada_v3.registrar_intento_lectura_preferencias_bolsas_interna_v1(
  p_contexto_canonico,p_vinculo_canonico,p_orden) a;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)
 TO vec_bolsa_llamamientos_preferencias_interna;

-- Fachada de efecto. AD279 verifica la concesión vigente, el contexto y el
-- material atestado y registra su auditoría dentro de esta transacción. Una
-- autorización nueva se exige también para recuperar un recibo por clave.
-- p_registrada_en es solo una marca de frescura de la petición. La fecha
-- conservada de la versión y del outbox procede de clock_timestamp() en PG.
CREATE FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(
 p_candidato_ref text,p_orden text[],p_version_esperada bigint,p_clave text,p_registrada_en timestamptz,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(reutilizada boolean,recibo_ref text,version bigint,orden text[],registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='15s'
AS $f$
DECLARE c jsonb; d jsonb; x jsonb; v_n integer; v_actuales text[]; v_ordenado text[];
 v_previa vec_bolsa_llamamientos.preferencia_bolsas_version%ROWTYPE;
 v_consumo record; v_version bigint; v_huella text; v_conjunto text; v_recibo text; v_clave_interna text;
 v_ahora timestamptz:=clock_timestamp();
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR session_user <> 'vec_interno_bolsa_preferencias_desarrollo'
    OR p_candidato_ref IS NULL OR p_candidato_ref !~ '^can_[A-Za-z0-9_-]{22,128}$'
    OR p_orden IS NULL OR array_ndims(p_orden) IS DISTINCT FROM 1 OR cardinality(p_orden) NOT BETWEEN 1 AND 256
    OR array_position(p_orden,NULL) IS NOT NULL
    OR p_version_esperada IS NULL OR p_version_esperada NOT BETWEEN 0 AND 2147483646
    OR p_clave IS NULL OR p_clave !~ '^[A-Za-z0-9._:-]{8,128}$'
    OR p_registrada_en IS NULL OR NOT isfinite(p_registrada_en)
    OR p_registrada_en < v_ahora-interval '5 minutes' OR p_registrada_en > v_ahora+interval '1 minute'
    OR p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_persona_version IS NULL OR p_perfil_version IS NULL OR p_payload IS NULL
    OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
 THEN RAISE EXCEPTION 'B116: orden inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
  x:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B116: material inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.preferencias_bolsas.ordenar_propias'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec.bolsa.preferencias-bolsas.ordenar.interna.v1'
    OR c->>'efecto_ref' IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'participaciones_candidato'
    OR d->>'finalidad' IS DISTINCT FROM 'gestion_preferencias_bolsas_propias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->>'principal_id' IS DISTINCT FROM x->>'principal_ref'
    OR d->>'perfil_activo_ref' IS DISTINCT FROM x->>'perfil_activo_ref'
    OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR jsonb_typeof(x->'vinculos') IS DISTINCT FROM 'array'
 THEN RAISE EXCEPTION 'B116: orden denegado' USING ERRCODE='42501'; END IF;
 SELECT count(*) INTO v_n FROM jsonb_array_elements(x->'vinculos') e
 WHERE e->>'tipo'='candidato' AND e->>'estado'='activo';
 IF v_n<>1 OR NOT EXISTS (
  SELECT 1 FROM jsonb_array_elements(x->'vinculos') e
  WHERE e->>'tipo'='candidato' AND e->>'estado'='activo' AND e->>'referencia'=p_candidato_ref)
 THEN RAISE EXCEPTION 'B116: candidato denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT v_consumo
 FROM vec_autorizacion_atestada_v3.registrar_y_consumir_preferencias_bolsas_interna_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF v_consumo.consumo_nuevo IS NOT TRUE OR v_consumo.efecto_ref IS DISTINCT FROM 'mi-bolsa:'||p_candidato_ref
 THEN RAISE EXCEPTION 'B116: consumo denegado' USING ERRCODE='42501'; END IF;

 PERFORM pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:preferencias_bolsas:'||p_candidato_ref,0));
 IF EXISTS (SELECT 1 FROM unnest(p_orden) AS u(ref)
  WHERE u.ref IS NULL OR NOT vec_bolsa_llamamientos.constitucion_texto_valido(u.ref,512)) THEN
  RAISE EXCEPTION 'B116: referencia inválida' USING ERRCODE='22023'; END IF;
 -- La huella usa separador de control prohibido en las referencias canónicas.
 v_huella:=encode(sha256(convert_to(p_candidato_ref||chr(31)||p_version_esperada::text||chr(31)||
  array_to_string(p_orden,chr(31)),'UTF8')),'hex');
 v_clave_interna:='@'||encode(sha256(convert_to('interna_corporativa'||chr(31)||p_clave,'UTF8')),'hex');
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref AND p.clave_idempotencia=v_clave_interna;
 IF FOUND THEN
  IF v_previa.actor_ref_interno IS DISTINCT FROM x->>'principal_ref'
   OR v_previa.perfil_ref_interno IS DISTINCT FROM x->>'perfil_activo_ref'
   OR v_previa.solicitud_huella_sha256<>v_huella OR v_previa.version<>p_version_esperada+1
  THEN RAISE EXCEPTION 'B116: clave reutilizada con otro orden' USING ERRCODE='VBP02'; END IF;
  RETURN QUERY SELECT true,v_previa.recibo_ref,v_previa.version,v_previa.orden,v_previa.registrada_en;
  RETURN;
 END IF;
 SELECT coalesce(max(p.version),0) INTO v_version FROM vec_bolsa_llamamientos.preferencia_bolsas_version p
 WHERE p.candidato_ref=p_candidato_ref;
 IF v_version<>p_version_esperada THEN
  RAISE EXCEPTION 'B116: versión de preferencias desactualizada' USING ERRCODE='VBP01';
 END IF;
 v_actuales:=vec_bolsa_llamamientos.preferencias_bolsas_actuales_interna_v1(p_candidato_ref,v_ahora);
 IF cardinality(v_actuales)<>cardinality(p_orden) THEN
  RAISE EXCEPTION 'B116: conjunto de bolsas cambiado' USING ERRCODE='VBP03'; END IF;
 SELECT array_agg(u.ref ORDER BY u.ref) INTO v_ordenado FROM unnest(p_orden) AS u(ref);
 IF v_ordenado IS DISTINCT FROM v_actuales THEN
  RAISE EXCEPTION 'B116: orden incompleto, duplicado o ajeno' USING ERRCODE='VBP03'; END IF;
 v_conjunto:=encode(sha256(convert_to(array_to_string(v_actuales,chr(31)),'UTF8')),'hex');
 v_recibo:='recibo:preferencias-bolsas:'||encode(sha256(convert_to(
  p_candidato_ref||chr(31)||v_clave_interna||chr(31)||v_huella,'UTF8')),'hex');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_version(
  candidato_ref,version,orden,conjunto_huella_sha256,clave_idempotencia,
  solicitud_huella_sha256,recibo_ref,decision_ref,auditoria_ref,registrada_en,
  actor_ref_interno,perfil_ref_interno)
 VALUES(p_candidato_ref,v_version+1,p_orden,v_conjunto,v_clave_interna,v_huella,v_recibo,
  v_consumo.decision_ref,v_consumo.auditoria_ref,v_ahora,x->>'principal_ref',x->>'perfil_activo_ref');
 INSERT INTO vec_bolsa_llamamientos.preferencia_bolsas_outbox(
  evento_ref,candidato_ref,version,recibo_ref,decision_ref,auditoria_ref,esquema,evento,creada_en)
 VALUES('evento:preferencias-bolsas:'||substr(v_recibo,length('recibo:preferencias-bolsas:')+1),
  p_candidato_ref,v_version+1,v_recibo,v_consumo.decision_ref,v_consumo.auditoria_ref,
  'vec.bolsa.preferencias.ordenadas.v1',
  jsonb_build_object('candidato_ref',p_candidato_ref,'version',v_version+1,
   'orden',p_orden,'conjunto_huella_sha256',v_conjunto,'recibo_ref',v_recibo,
   'decision_ref',v_consumo.decision_ref,'auditoria_ref',v_consumo.auditoria_ref),v_ahora);
 RETURN QUERY SELECT false,v_recibo,v_version+1,p_orden,v_ahora;
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(
 text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(
 text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_bolsa_llamamientos_preferencias_interna;
RESET ROLE;
DO $post$
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_preferencias_interna'
  AND NOT(r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid='vec_bolsa_llamamientos.preferencia_bolsas_version'::regclass
  AND c.relrowsecurity AND c.relforcerowsecurity)
 OR (SELECT count(*) FROM pg_constraint c
  WHERE c.conrelid='vec_bolsa_llamamientos.preferencia_bolsas_version'::regclass
   AND c.conname IN ('preferencia_bolsas_version_clave_superficie_v1','preferencia_bolsas_version_actor_interno_v1')
   AND c.contype='c' AND c.convalidated)<>2
 OR has_function_privilege('public','vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE') IS TRUE
 OR has_function_privilege('public','vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE') IS TRUE
 OR has_function_privilege('public','vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_preferencias_interna','vec_bolsa_llamamientos.consultar_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_preferencias_interna','vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_interna_v1(text,bytea,bytea,jsonb)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_bolsa_llamamientos_preferencias_interna','vec_bolsa_llamamientos.ordenar_preferencias_bolsas_interna_v1(text,text[],bigint,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'B116: postimagen o ACL divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
