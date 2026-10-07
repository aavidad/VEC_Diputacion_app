\set ON_ERROR_STOP on
-- AD176: familias propias Personal33, con variantes POST174 H9 y POST173.
-- Ambas preimágenes son parejas nombre/SHA observadas en el clon PG18.4.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000176',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_provision_fuentes_iniciales_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_fuentes_iniciales_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.control_cadena_auditoria') IS NULL
 OR pg_catalog.to_regrole('vec_personal_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.motivo_intento_unidad_valido_v1(text,text)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a
   WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND NOT a.attisdropped
    AND a.attname IN ('unidad_plan_ref','unidad_preimagen_sha256','unidad_configuracion_sha256','unidad_alcance',
     'unidad_recibo_ref','unidad_recibo_sha256','unidad_solicitud_sha256'))
 OR (SELECT count(*) FROM pg_catalog.pg_constraint c
   WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
    AND c.conname IN ('auditoria_tipo_disjunto_v2','auditoria_tipo_disjunto_v4') AND c.contype='c' AND c.convalidated)<>1
 THEN RAISE EXCEPTION 'AD176: PARO clave=preimagen actual=incompatible esperado=AD174_sin_AD176' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN unidad_plan_ref text, ADD COLUMN unidad_preimagen_sha256 text,
 ADD COLUMN unidad_configuracion_sha256 text, ADD COLUMN unidad_alcance text,
 ADD COLUMN unidad_recibo_ref text, ADD COLUMN unidad_recibo_sha256 text, ADD COLUMN unidad_solicitud_sha256 text;
DO $familias$
DECLARE v_predicado text;v_nombre name;v_nuevo text;v_actual_sha text;v_version_nula text:='';
 v_h9_sha constant text:='c7dc8abc0c0ea178cadb22960976af57a7f711e158a076c7068227d5718efbb8';
BEGIN
 SELECT pg_catalog.pg_get_constraintdef(c.oid,false),c.conname INTO STRICT v_predicado,v_nombre FROM pg_catalog.pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname IN ('auditoria_tipo_disjunto_v2','auditoria_tipo_disjunto_v4') AND c.contype='c' AND c.convalidated;
 v_actual_sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(v_predicado,'UTF8')),'hex');
 IF v_nombre='auditoria_tipo_disjunto_v2' AND v_actual_sha=v_h9_sha THEN
  v_version_nula:='';
 ELSIF v_nombre='auditoria_tipo_disjunto_v4'
  AND v_actual_sha='3a2b7514294cd022102440e37b784916e48c340e63ce7195defdea118bad34ba' THEN
  v_version_nula:=' AND version_consumo IS NULL';
 ELSE
  RAISE EXCEPTION 'AD176: PARO clave=CHECK_nombre_SHA256 actual=%:% esperado=v2:%_o_v4:3a2b7514294cd022102440e37b784916e48c340e63ce7195defdea118bad34ba',
   v_nombre,v_actual_sha,v_h9_sha USING ERRCODE='55000';
 END IF;
 v_nuevo:='CHECK ((unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL'
  ||' AND unidad_alcance IS NULL AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND ('
  ||pg_catalog.substr(v_predicado,8,pg_catalog.length(v_predicado)-8)||')) OR ('||$confirmado$tipo_registro='unidad_inicial_personal'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL
 AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_solicitud_sha256 IS NULL AND unidad_plan_ref IS NOT NULL AND unidad_preimagen_sha256 IS NOT NULL
 AND unidad_configuracion_sha256 IS NOT NULL AND unidad_alcance IS NOT DISTINCT FROM 'sintetico_declarado'
 AND unidad_recibo_ref IS NOT NULL AND unidad_recibo_sha256 IS NOT NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND plan_sha256 IS NOT NULL AND aprobacion_ref IS NOT NULL AND fuente_ref IS NOT NULL AND fuente_sha256 IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'inicializar_unidad_sintetica_admin_v1' AND modulo_id IS NOT DISTINCT FROM 'personal'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'inicializar_unidad_sintetica_admin' AND resultado IS NOT DISTINCT FROM 'permitido'
 AND motivo_ref IS NOT DISTINCT FROM 'unidad_registrada' AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL$confirmado$
 ||v_version_nula||') OR ('||$intento$tipo_registro='intento_unidad_inicial_personal'
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL
 AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL
 AND unidad_alcance IS NULL AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL
 AND plan_sha256 IS NULL AND aprobacion_ref IS NULL AND fuente_ref IS NULL AND fuente_sha256 IS NULL
 AND unidad_solicitud_sha256 IS NOT NULL AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'inicializar_unidad_sintetica_admin_v1' AND modulo_id IS NOT DISTINCT FROM 'personal'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'inicializar_unidad_sintetica_admin'
 AND resultado IS NOT NULL AND motivo_ref IS NOT NULL AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL$intento$||v_version_nula||'))';
 EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT %I',v_nombre);
 EXECUTE pg_catalog.format('ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT %I %s',v_nombre,v_nuevo);
END $familias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_unidad_valido_v1(p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog
AS $catalogo$
 SELECT EXISTS(SELECT 1 FROM (VALUES ('permitido','unidad_registrada'),('permitido','unidad_replay'),
 ('denegado','unidad_denegada'),('error','unidad_error')) AS m(resultado,motivo_ref)
 WHERE m.resultado=p_resultado AND m.motivo_ref=p_motivo)
$catalogo$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_unidad_valido_v1(text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD CONSTRAINT auditoria_unidad_inicial_formato_v1 CHECK (tipo_registro<>'unidad_inicial_personal' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND unidad_plan_ref ~ '^pui_[A-Za-z0-9_-]{22,124}$' AND plan_sha256 ~ '^[0-9a-f]{64}$'
 AND unidad_preimagen_sha256 ~ '^[0-9a-f]{64}$' AND unidad_configuracion_sha256 ~ '^[0-9a-f]{64}$'
 AND unidad_recibo_ref ~ '^recibo_unidad:[0-9a-f]{32}$' AND unidad_recibo_sha256 ~ '^[0-9a-f]{64}$'
 AND fuente_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND fuente_sha256 ~ '^[0-9a-f]{64}$'
 AND aprobacion_ref ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$' AND recurso_ref ~ '^unidad:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$')),
 ADD CONSTRAINT auditoria_intento_unidad_formato_v1 CHECK (tipo_registro<>'intento_unidad_inicial_personal' OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$'
 AND unidad_solicitud_sha256 ~ '^[0-9a-f]{64}$' AND recurso_ref ~ '^solicitud_unidad:[0-9a-f]{32}$'
 AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND vec_autorizacion_atestada_v3.motivo_intento_unidad_valido_v1(resultado,motivo_ref)));
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','plan_ref','plan_sha256',
  'preimagen_sha256','configuracion_sha256','aprobacion_ref','alcance_fuente','accion','recurso_ref',
  'resultado','motivo_ref','proceso','canal','finalidad_ref','correlacion_ref','fuente_ref','fuente_sha256','recibo_ref','recibo_sha256'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD176: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD176: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD176: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD176: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'unidad_inicial_personal'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'plan_ref' !~ '^pui_[A-Za-z0-9_-]{22,124}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'preimagen_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'configuracion_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'alcance_fuente' IS DISTINCT FROM 'sintetico_declarado'
 OR p_evento->>'accion' IS DISTINCT FROM 'inicializar_unidad_sintetica_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^unidad:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
 OR p_evento->>'resultado' IS DISTINCT FROM 'permitido'
 OR p_evento->>'motivo_ref' IS DISTINCT FROM 'unidad_registrada'
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'inicializar_unidad_sintetica_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR p_evento->>'fuente_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR p_evento->>'fuente_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'recibo_ref' !~ '^recibo_unidad:[0-9a-f]{32}$'
 OR p_evento->>'recibo_sha256' !~ '^[0-9a-f]{64}$'
 THEN RAISE EXCEPTION 'AD176: PARO clave=semantica actual=incompatible esperado=unidad_inicial_privada' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.unidad-inicial-personal.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'unidad_inicial_personal'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD176: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD176: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_u_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.unidad-inicial-personal.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,fuente_ref,fuente_sha256,operador_login,plan_sha256,aprobacion_ref,
  unidad_plan_ref,unidad_preimagen_sha256,unidad_configuracion_sha256,unidad_alcance,unidad_recibo_ref,unidad_recibo_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'unidad_inicial_personal',
  p_evento->>'evento_ref',v_material_sha,p_evento->>'fuente_ref',p_evento->>'fuente_sha256',
  (p_evento->>'operador_login')::name,p_evento->>'plan_sha256',p_evento->>'aprobacion_ref',
  p_evento->>'plan_ref',p_evento->>'preimagen_sha256',p_evento->>'configuracion_sha256',p_evento->>'alcance_fuente',p_evento->>'recibo_ref',p_evento->>'recibo_sha256',
  p_evento->>'accion','personal',p_evento->>'recurso_ref',p_evento->>'finalidad_ref','permitido',
  p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(jsonb) TO vec_personal_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_unidad_inicial_personal_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_personal_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD176: PARO clave=ACL actual=ampliada esperado=owners_orquestador' USING ERRCODE='55000'; END IF;
END $acl$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(p_evento jsonb)
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
 THEN RAISE EXCEPTION 'AD176: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD176: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD176: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD176: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_unidad_inicial_personal'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' IS DISTINCT FROM 'inicializar_unidad_sintetica_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^solicitud_unidad:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_unidad_valido_v1(p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'inicializar_unidad_sintetica_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD176: PARO clave=semantica actual=incompatible esperado=unidad_inicial_privada' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-unidad-inicial-personal.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_unidad_inicial_personal'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD176: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD176: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_ui_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-unidad-inicial-personal.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,unidad_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_unidad_inicial_personal',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','personal',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(jsonb) TO vec_personal_propietario;
DO $acl$
DECLARE f oid:=pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_unidad_inicial_personal_v1(jsonb)');
BEGIN
 IF EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
  WHERE p.oid=f AND (a.grantee NOT IN ('vec_autorizacion_atestada_v3_propietario'::regrole,'vec_personal_propietario'::regrole)
   OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD176: PARO clave=ACL actual=ampliada esperado=owners_orquestador' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
