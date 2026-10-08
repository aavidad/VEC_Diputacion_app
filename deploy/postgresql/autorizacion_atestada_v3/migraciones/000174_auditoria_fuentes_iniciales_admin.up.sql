\set ON_ERROR_STOP on
-- AD174 admite exclusivamente H9/AD171 o la preimagen exacta POST-AD173.
-- Una variante no ensayada ni una huella distinta no autoriza instalación.
-- AD174: acto técnico de fuentes iniciales, sin una autorización V3 inventada.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000174',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_evento_admin_preperfil_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.control_cadena_auditoria') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.motivo_intento_fuentes_valido_v1(text,text)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
   WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
     AND a.attname IN ('fuentes_plan_ref','fuentes_preimagen_sha256','fuentes_configuracion_sha256','fuentes_alcance','fuentes_solicitud_sha256'))
 OR (SELECT count(*) FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
     AND c.conname IN ('auditoria_tipo_disjunto_v2','auditoria_tipo_disjunto_v4') AND c.contype='c' AND c.convalidated)<>1
 THEN RAISE EXCEPTION 'AD174: PARO clave=preimagen actual=incompatible esperado=AD171_sin_AD174' USING ERRCODE='55000'; END IF;
END $pre$;

LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN fuentes_plan_ref text,
 ADD COLUMN fuentes_preimagen_sha256 text,
 ADD COLUMN fuentes_configuracion_sha256 text,
 ADD COLUMN fuentes_alcance text,
 ADD COLUMN fuentes_solicitud_sha256 text;

-- Conserva literalmente la pareja de CHECK admitida y su nombre (v2 o v4).
-- POST-AD173 reserva version_consumo exclusivamente a sus consumos reales.
DO $familia$
DECLARE v_predicado text;v_nombre name;v_nuevo text;v_actual_sha text;v_version_nula text:='';
 v_h9_sha constant text:='f31b31dc0ec40bdd2d7a6930346210e919ab4aed225352235723525a27e7dc29';
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false),c.conname INTO STRICT v_predicado,v_nombre
 FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
   AND c.conname IN ('auditoria_tipo_disjunto_v2','auditoria_tipo_disjunto_v4') AND c.contype='c' AND c.convalidated;
 IF pg_catalog.left(v_predicado,7)<>'CHECK (' OR pg_catalog.right(v_predicado,1)<>')'
 THEN RAISE EXCEPTION 'AD174: PARO clave=CHECK actual=incompatible esperado=CHECK_validado' USING ERRCODE='55000'; END IF;
 v_actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_predicado,'UTF8')),'hex');
 IF v_nombre='auditoria_tipo_disjunto_v2' AND v_actual_sha=v_h9_sha THEN
  v_version_nula:='';
 ELSIF v_nombre='auditoria_tipo_disjunto_v4'
  AND v_actual_sha='4ea0f7f797122ddb81c59c4a601c87c213f10206d619d3c031f947f8efd2a1ea' THEN
  v_version_nula:=' AND version_consumo IS NULL';
 ELSE
  RAISE EXCEPTION 'AD174: PARO clave=CHECK_nombre_SHA256 actual=%:% esperado=v2:%_o_v4:4ea0f7f797122ddb81c59c4a601c87c213f10206d619d3c031f947f8efd2a1ea',
   v_nombre,v_actual_sha,v_h9_sha USING ERRCODE='55000';
 END IF;
 v_nuevo:='CHECK ((fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL'
   ||' AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL AND ('
   ||pg_catalog.substr(v_predicado,8,pg_catalog.length(v_predicado)-8)||')) OR ('
   ||$tipado$tipo_registro='provision_fuentes_iniciales_admin'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL
 AND actor_ref IS NULL AND perfil_activo_ref IS NULL AND registro_contexto_ref IS NULL
 AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL
 AND vinculo_sha256 IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL
 AND fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL
 AND operador_login IS NOT NULL AND plan_sha256 IS NOT NULL AND aprobacion_ref IS NOT NULL
 AND fuentes_solicitud_sha256 IS NULL
 AND fuentes_plan_ref IS NOT NULL AND fuentes_preimagen_sha256 IS NOT NULL
 AND fuentes_configuracion_sha256 IS NOT NULL AND fuentes_alcance IS NOT DISTINCT FROM 'sintetico_declarado'
 AND accion IS NOT DISTINCT FROM 'provisionar_fuentes_iniciales_admin_v1'
 AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'provision_fuentes_iniciales_admin'
 AND resultado IS NOT DISTINCT FROM 'permitido'
 AND recurso_ref IS NOT NULL AND motivo_ref IS NOT NULL AND proceso IS NOT NULL AND correlacion_ref IS NOT NULL$tipado$
   ||v_version_nula||') OR ('||$intento$tipo_registro='intento_fuentes_iniciales_admin'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL
 AND actor_ref IS NULL AND perfil_activo_ref IS NULL AND registro_contexto_ref IS NULL
 AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL AND autenticacion_ref IS NULL
 AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND plan_sha256 IS NULL AND aprobacion_ref IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL
 AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL
 AND operador_login IS NOT NULL AND fuentes_solicitud_sha256 IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'provisionar_fuentes_iniciales_admin_v1'
 AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'provision_fuentes_iniciales_admin'
 AND proceso IS NOT DISTINCT FROM 'postgresql'
 AND resultado IS NOT NULL AND recurso_ref IS NOT NULL AND motivo_ref IS NOT NULL AND correlacion_ref IS NOT NULL$intento$||v_version_nula||'))';
 EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT %I',v_nombre);
 EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT %I %s',v_nombre,v_nuevo);
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_fuentes_iniciales_formato_v1 CHECK (
 tipo_registro<>'provision_fuentes_iniciales_admin' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND fuente_sha256 ~ '^[0-9a-f]{64}$'
 AND fuentes_plan_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND plan_sha256 ~ '^[0-9a-f]{64}$'
 AND fuentes_preimagen_sha256 ~ '^[0-9a-f]{64}$' AND fuentes_configuracion_sha256 ~ '^[0-9a-f]{64}$'
 AND aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 AND recurso_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$'
 AND motivo_ref ~ '^[a-z][a-z0-9._:-]{0,159}$'
 AND proceso ~ '^[a-z][a-z0-9._-]{1,79}$' AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 ));

-- Catálogo técnico cerrado de datos: resultado y motivo de la invocación DB.
-- No es un permiso ni una aprobación; no conserva texto libre del motor.
CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_fuentes_valido_v1(p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog
AS $catalogo$
 SELECT EXISTS(SELECT 1 FROM (VALUES
  ('permitido','fuentes_registradas'),('permitido','fuentes_replay'),
  ('denegado','fuentes_denegadas'),('error','fuentes_error')) AS m(resultado,motivo_ref)
  WHERE m.resultado=p_resultado AND m.motivo_ref=p_motivo)
$catalogo$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_fuentes_valido_v1(text,text) FROM PUBLIC;

ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_intento_fuentes_formato_v1 CHECK (
 tipo_registro<>'intento_fuentes_iniciales_admin' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND fuentes_solicitud_sha256 ~ '^[0-9a-f]{64}$'
 AND recurso_ref ~ '^solicitud_fuentes:[0-9a-f]{32}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND vec_autorizacion_atestada_v3.motivo_intento_fuentes_valido_v1(resultado,motivo_ref)
 ));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','plan_ref','plan_sha256',
  'preimagen_sha256','configuracion_sha256','aprobacion_ref','alcance_fuente','accion','recurso_ref',
  'resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD174: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD174: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD174: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD174: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'provision_fuentes_iniciales_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'plan_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'preimagen_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'configuracion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR p_evento->>'accion' IS DISTINCT FROM 'provisionar_fuentes_iniciales_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$'
 OR p_evento->>'resultado' IS DISTINCT FROM 'permitido'
 OR p_evento->>'motivo_ref' !~ '^[a-z][a-z0-9._:-]{0,159}$'
 OR p_evento->>'proceso' !~ '^[a-z][a-z0-9._-]{1,79}$'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'provision_fuentes_iniciales_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR p_evento->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'fuente_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AD174: PARO clave=semantica actual=incompatible esperado=provision_fuentes_privada' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.fuentes-iniciales.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 -- Mismo espacio de cerrojo y misma unicidad que los eventos AD171.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'provision_fuentes_iniciales_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD174: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD174: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_f_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.fuentes-iniciales.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,fuente_ref,fuente_sha256,operador_login,plan_sha256,aprobacion_ref,
  fuentes_plan_ref,fuentes_preimagen_sha256,fuentes_configuracion_sha256,fuentes_alcance,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'provision_fuentes_iniciales_admin',
  p_evento->>'evento_ref',v_material_sha,p_evento->>'fuente_ref',p_evento->>'fuente_sha256',
  (p_evento->>'operador_login')::name,p_evento->>'plan_sha256',p_evento->>'aprobacion_ref',
  p_evento->>'plan_ref',p_evento->>'preimagen_sha256',p_evento->>'configuracion_sha256',p_evento->>'alcance_fuente',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref','permitido',
  p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD174: PARO clave=ACL actual=ampliada esperado=owners_orquestador' USING ERRCODE='55000'; END IF;
END $acl$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','solicitud_sha256',
  'accion','recurso_ref','resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD174: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD174: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD174: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD174: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_fuentes_iniciales_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' IS DISTINCT FROM 'provisionar_fuentes_iniciales_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^solicitud_fuentes:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_fuentes_valido_v1(p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'provision_fuentes_iniciales_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD174: PARO clave=semantica actual=incompatible esperado=provision_fuentes_privada' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-fuentes-iniciales.v1');
 FOREACH v_clave IN ARRAY v_orden LOOP
  v_material:=v_material||vec_autorizacion_atestada_v3.encuadrar_mac(p_evento->>v_clave);
 END LOOP;
 v_material_sha:=pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
 -- Mismo espacio de cerrojo y misma unicidad que los eventos AD171.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
  'vec_autorizacion_atestada_v3:evento-admin:'||(p_evento->>'evento_ref'),0));
 SELECT a.tipo_registro,a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
  a.evento_material_sha256 INTO v_existente
 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=p_evento->>'evento_ref';
 IF FOUND THEN
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_fuentes_iniciales_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD174: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD174: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_fi_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-fuentes-iniciales.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,fuentes_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_fuentes_iniciales_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD174: PARO clave=ACL actual=ampliada esperado=owners_orquestador' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
