\set ON_ERROR_STOP on
-- AD183: mantenimiento técnico del perfil fijo, sin acción de bootstrap.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000183',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_bootstrap_central_admin_v1(jsonb)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
 OR pg_catalog.to_regrole('vec_autorizacion_propietario') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(jsonb)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_attribute a WHERE a.attrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass AND NOT a.attisdropped
  AND a.attname IN('mantenimiento_detalle','mantenimiento_solicitud_sha256'))
 OR (SELECT count(*) FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
  AND c.conname IN('auditoria_tipo_disjunto_v2','auditoria_tipo_disjunto_v4') AND c.contype='c' AND c.convalidated)<>1
 THEN RAISE EXCEPTION 'AD183: PARO clave=preimagen actual=incompatible esperado=POST179_sin_AD183' USING ERRCODE='55000'; END IF;
END $pre$;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN mantenimiento_detalle jsonb, ADD COLUMN mantenimiento_solicitud_sha256 text;
DO $familias$
DECLARE v_def text;v_nombre name;v_sha text;v_nuevo text;v_version text:='';
BEGIN
 SELECT pg_get_constraintdef(c.oid,false),c.conname INTO STRICT v_def,v_nombre FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname IN('auditoria_tipo_disjunto_v2','auditoria_tipo_disjunto_v4') AND c.contype='c' AND c.convalidated;
 v_sha:=encode(sha256(convert_to(v_def,'UTF8')),'hex');
 IF v_nombre='auditoria_tipo_disjunto_v2' AND v_sha='0996f678e1bec083fcd4a09f07424de842e1ee5ee83f6026040924db38118cfa' THEN
  v_version:='';
 ELSIF v_nombre='auditoria_tipo_disjunto_v4' AND v_sha='e2ce386bbc77b80f237a4923966f98619cfb9f9823a6bea9099c9b3835ed6a93' THEN
  v_version:=' AND version_consumo IS NULL';
 ELSE RAISE EXCEPTION 'AD183: PARO clave=CHECK_nombre_SHA256 actual=%:% esperado=v2:0996f678e1bec083fcd4a09f07424de842e1ee5ee83f6026040924db38118cfa_o_v4:e2ce386bbc77b80f237a4923966f98619cfb9f9823a6bea9099c9b3835ed6a93',v_nombre,v_sha USING ERRCODE='55000'; END IF;
 v_nuevo:='CHECK ((mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NULL AND ('||substr(v_def,8,length(v_def)-8)||')) OR ('||$tipado$
 tipo_registro IN('mantenimiento_perfil_fijo_admin','intento_mantenimiento_perfil_fijo_admin')
 AND decision_ref IS NULL AND efecto_ref IS NULL AND huella_efecto_sha256 IS NULL
 AND intento_ref IS NULL AND intento_material_sha256 IS NULL AND actor_ref IS NULL AND perfil_activo_ref IS NULL
 AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL AND procedencia_sha256 IS NULL
 AND autenticacion_ref IS NULL AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL AND vinculo_sha256 IS NULL
 AND fuente_ref IS NULL AND fuente_sha256 IS NULL AND aprobacion_ref IS NULL
 AND fuentes_plan_ref IS NULL AND fuentes_preimagen_sha256 IS NULL AND fuentes_configuracion_sha256 IS NULL AND fuentes_alcance IS NULL AND fuentes_solicitud_sha256 IS NULL
 AND unidad_plan_ref IS NULL AND unidad_preimagen_sha256 IS NULL AND unidad_configuracion_sha256 IS NULL AND unidad_alcance IS NULL
 AND unidad_recibo_ref IS NULL AND unidad_recibo_sha256 IS NULL AND unidad_solicitud_sha256 IS NULL AND bootstrap_solicitud_sha256 IS NULL
 AND evento_ref IS NOT NULL AND evento_material_sha256 IS NOT NULL AND operador_login IS NOT NULL
 AND accion IS NOT DISTINCT FROM 'mantener_version_perfil_fijo_admin_v1' AND modulo_id IS NOT DISTINCT FROM 'administracion'
 AND proceso IS NOT DISTINCT FROM 'postgresql' AND canal IS NOT DISTINCT FROM 'operacion_tecnica_privada'
 AND finalidad_ref IS NOT DISTINCT FROM 'mantenimiento_perfil_fijo_admin' AND recurso_ref IS NOT NULL AND correlacion_ref IS NOT NULL
 AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
 AND ((tipo_registro='mantenimiento_perfil_fijo_admin' AND plan_sha256 IS NOT NULL AND mantenimiento_detalle IS NOT NULL
  AND mantenimiento_solicitud_sha256 IS NULL AND resultado IS NOT DISTINCT FROM 'permitido' AND motivo_ref IS NOT DISTINCT FROM 'mantenimiento_registrado'
  AND recurso_ref IS NOT DISTINCT FROM 'mantenimiento_admin:'||substr(plan_sha256,1,32))
 OR (tipo_registro='intento_mantenimiento_perfil_fijo_admin' AND plan_sha256 IS NULL AND mantenimiento_detalle IS NULL AND mantenimiento_solicitud_sha256 IS NOT NULL))
 $tipado$||v_version||'))';
 EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT %I',v_nombre);
 EXECUTE format('ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT %I %s',v_nombre,v_nuevo);
END $familias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.detalle_mantenimiento_valido_v1(p jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog AS $f$
DECLARE claves constant text[]:=ARRAY['preimagen_sha256','catalogo_sha256','rol_origen_ref','rol_origen_sha256','rol_destino_ref','rol_destino_sha256',
 'asignacion_1_origen_ref','asignacion_1_origen_sha256','asignacion_1_destino_ref','asignacion_1_destino_sha256',
 'asignacion_2_origen_ref','asignacion_2_origen_sha256','asignacion_2_destino_ref','asignacion_2_destino_sha256'];k text;
BEGIN
 IF jsonb_typeof(p) IS DISTINCT FROM 'object' OR NOT p ?& claves OR (SELECT count(*) FROM jsonb_object_keys(p))<>cardinality(claves) THEN RETURN false; END IF;
 FOREACH k IN ARRAY claves LOOP
  IF jsonb_typeof(p->k) IS DISTINCT FROM 'string' THEN RETURN false; END IF;
  IF right(k,7)='_sha256' THEN
   IF p->>k !~ '^[0-9a-f]{64}$' THEN RETURN false; END IF;
  ELSIF p->>k !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$' THEN RETURN false; END IF;
 END LOOP;
 RETURN true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.detalle_mantenimiento_valido_v1(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.motivo_intento_mantenimiento_valido_v1(p_resultado text,p_motivo text)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE SET search_path=pg_catalog AS $f$
 SELECT EXISTS(SELECT 1 FROM (VALUES ('permitido','mantenimiento_registrado'),('permitido','mantenimiento_replay'),
 ('denegado','mantenimiento_denegado'),('error','mantenimiento_error')) AS m(resultado,motivo_ref)
 WHERE m.resultado=p_resultado AND m.motivo_ref=p_motivo)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.motivo_intento_mantenimiento_valido_v1(text,text) FROM PUBLIC;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_mantenimiento_fijo_formato_v1 CHECK (
 tipo_registro NOT IN('mantenimiento_perfil_fijo_admin','intento_mantenimiento_perfil_fijo_admin') OR (
 evento_ref ~ '^evento_[0-9a-f]{32}$' AND evento_material_sha256 ~ '^[0-9a-f]{64}$' AND correlacion_ref ~ '^correlacion_[0-9a-f]{32}$'
 AND ((tipo_registro='mantenimiento_perfil_fijo_admin' AND plan_sha256 ~ '^[0-9a-f]{64}$' AND vec_autorizacion_atestada_v3.detalle_mantenimiento_valido_v1(mantenimiento_detalle))
 OR (tipo_registro='intento_mantenimiento_perfil_fijo_admin' AND mantenimiento_solicitud_sha256 ~ '^[0-9a-f]{64}$'
  AND recurso_ref ~ '^solicitud_mantenimiento:[0-9a-f]{32}$' AND vec_autorizacion_atestada_v3.motivo_intento_mantenimiento_valido_v1(resultado,motivo_ref)))));
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(p_evento jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,correlacion_ref text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $funcion$
DECLARE
 v_orden constant text[]:=ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','preimagen_sha256','catalogo_sha256','rol_origen_ref','rol_origen_sha256','rol_destino_ref','rol_destino_sha256','asignacion_1_origen_ref','asignacion_1_origen_sha256','asignacion_1_destino_ref','asignacion_1_destino_sha256','asignacion_2_origen_ref','asignacion_2_origen_sha256','asignacion_2_destino_ref','asignacion_2_destino_sha256','proceso','canal','finalidad_ref','correlacion_ref'];
 v_claves text[];v_clave text;v_material bytea;v_material_sha text;v_anterior text;
 v_secuencia numeric;v_instante timestamptz(6);v_ref text;v_huella text;v_existente record;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off' OR pg_catalog.current_setting('TimeZone')<>'UTC'
 THEN RAISE EXCEPTION 'AD183: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD183: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD183: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD183: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'mantenimiento_perfil_fijo_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR vec_autorizacion_atestada_v3.detalle_mantenimiento_valido_v1(p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','proceso','canal','finalidad_ref','correlacion_ref']) IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'mantenimiento_perfil_fijo_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD183: PARO clave=semantica actual=incompatible esperado=mantenimiento_fijo_privado' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.mantenimiento-perfil-fijo-admin.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'mantenimiento_perfil_fijo_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD183: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD183: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_mf_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.mantenimiento-perfil-fijo-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,plan_sha256,mantenimiento_detalle,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'mantenimiento_perfil_fijo_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'plan_sha256',
  p_evento-ARRAY['tipo_registro','evento_ref','operador_login','plan_sha256','proceso','canal','finalidad_ref','correlacion_ref'],
  'mantener_version_perfil_fijo_admin_v1','administracion','mantenimiento_admin:'||pg_catalog.substr(p_evento->>'plan_sha256',1,32),
  'mantenimiento_perfil_fijo_admin','permitido','mantenimiento_registrado',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(p_evento jsonb)
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
 THEN RAISE EXCEPTION 'AD183: PARO clave=transaccion actual=incompatible esperado=serializable_rw_UTC' USING ERRCODE='25000'; END IF;
 IF pg_catalog.jsonb_typeof(p_evento) IS DISTINCT FROM 'object' OR pg_catalog.octet_length(p_evento::text)>8192
 THEN RAISE EXCEPTION 'AD183: PARO clave=evento actual=incompatible esperado=objeto_acotado' USING ERRCODE='22023'; END IF;
 SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") INTO v_claves FROM pg_catalog.jsonb_object_keys(p_evento) k;
 IF v_claves IS DISTINCT FROM (SELECT pg_catalog.array_agg(k ORDER BY k COLLATE "C") FROM pg_catalog.unnest(v_orden) k)
 THEN RAISE EXCEPTION 'AD183: PARO clave=campos actual=incompatible esperado=ABI_exacto' USING ERRCODE='22023'; END IF;
 FOREACH v_clave IN ARRAY v_orden LOOP
  IF pg_catalog.jsonb_typeof(p_evento->v_clave) IS DISTINCT FROM 'string'
  OR pg_catalog.octet_length(p_evento->>v_clave) NOT BETWEEN 1 AND 200
  THEN RAISE EXCEPTION 'AD183: PARO clave=valor actual=incompatible esperado=string_acotado' USING ERRCODE='22023'; END IF;
 END LOOP;
 IF p_evento->>'tipo_registro' IS DISTINCT FROM 'intento_mantenimiento_perfil_fijo_admin'
 OR p_evento->>'operador_login' IS DISTINCT FROM session_user::text
 OR pg_catalog.octet_length(p_evento->>'operador_login')>63
 OR p_evento->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR p_evento->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR p_evento->>'accion' IS DISTINCT FROM 'mantener_version_perfil_fijo_admin_v1'
 OR p_evento->>'recurso_ref' !~ '^solicitud_mantenimiento:[0-9a-f]{32}$'
 OR vec_autorizacion_atestada_v3.motivo_intento_mantenimiento_valido_v1(p_evento->>'resultado',p_evento->>'motivo_ref') IS DISTINCT FROM true
 OR p_evento->>'proceso' IS DISTINCT FROM 'postgresql'
 OR p_evento->>'canal' IS DISTINCT FROM 'operacion_tecnica_privada'
 OR p_evento->>'finalidad_ref' IS DISTINCT FROM 'mantenimiento_perfil_fijo_admin'
 OR p_evento->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD183: PARO clave=semantica actual=incompatible esperado=mantenimiento_fijo_privado' USING ERRCODE='22023'; END IF;
 v_material:=vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.intento-mantenimiento-perfil-fijo-admin.v1');
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
  IF v_existente.tipo_registro IS DISTINCT FROM 'intento_mantenimiento_perfil_fijo_admin'
  OR v_existente.evento_material_sha256 IS DISTINCT FROM v_material_sha
  THEN RAISE EXCEPTION 'AD183: PARO clave=replay actual=material_distinto esperado=material_original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,v_existente.huella_sha256,
   v_existente.correlacion_ref,v_existente.registrada_en;
  RETURN;
 END IF;
 SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c WHERE c.control_id FOR UPDATE;
 IF v_secuencia>=9007199254740991::numeric
 THEN RAISE EXCEPTION 'AD183: PARO clave=secuencia actual=limite esperado=entero_seguro' USING ERRCODE='22003'; END IF;
 v_secuencia:=v_secuencia+1;v_instante:=pg_catalog.clock_timestamp();
 v_ref:='aud_v3_mfi_'||pg_catalog.substr(p_evento->>'evento_ref',8,32);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(
  vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-mantenimiento-perfil-fijo-admin.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_ref)||
  vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(pg_catalog.to_char(v_instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,mantenimiento_solicitud_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(v_ref,v_secuencia,v_anterior,v_huella,v_instante,'intento_mantenimiento_perfil_fijo_admin',
  p_evento->>'evento_ref',v_material_sha,(p_evento->>'operador_login')::name,p_evento->>'solicitud_sha256',
  p_evento->>'accion','administracion',p_evento->>'recurso_ref',p_evento->>'finalidad_ref',
  p_evento->>'resultado',p_evento->>'motivo_ref',p_evento->>'proceso',p_evento->>'canal',p_evento->>'correlacion_ref');
 UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
 SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante WHERE control_id;
 RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_evento->>'correlacion_ref',v_instante;
END $funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(jsonb) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(jsonb),vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f record;roles oid[]:=ARRAY['vec_autorizacion_atestada_v3_propietario'::regrole,'vec_autorizacion_propietario'::regrole];
BEGIN
 FOR f IN SELECT oid FROM pg_proc WHERE oid IN(to_regprocedure('vec_autorizacion_atestada_v3.registrar_mantenimiento_perfil_fijo_admin_v1(jsonb)'),to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_mantenimiento_perfil_fijo_admin_v1(jsonb)')) LOOP
  IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(COALESCE(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f.oid
   AND (a.grantee<>ALL(roles) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  THEN RAISE EXCEPTION 'AD183: PARO clave=ACL actual=ampliada esperado=AD_AUT_owners' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
