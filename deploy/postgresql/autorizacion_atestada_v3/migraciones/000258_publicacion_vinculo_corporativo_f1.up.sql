\set ON_ERROR_STOP on
-- AD258: familia común y limitada del acto técnico de publicación F1.
-- Requiere AD251 (PR 1083); conserva la familia existente y su semántica.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE h text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_lectura_auditoria_rrhh_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD258: PARO clave=dependencia actual=incompatible esperado=AD251' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,false),'UTF8')),'hex') INTO h
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF h IS DISTINCT FROM '6cd7353335b9815259d600304c89440b96d7123284700668f58d71a099cecd8f'
 THEN RAISE EXCEPTION 'AD258: PARO clave=familia actual=% esperado=post_AD251',coalesce(h,'ausente') USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
LOCK TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 IN ACCESS EXCLUSIVE MODE;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
 ADD COLUMN publicacion_vinculo_f1_detalle jsonb,
 ADD COLUMN intento_vinculo_f1_sha256 text;

CREATE FUNCTION vec_autorizacion_atestada_v3.auditoria_publicacion_vinculo_f1_valida(j jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(
  j->>'tipo_registro' IN ('publicacion_vinculo_corporativo_f1','comprobacion_vinculo_corporativo_f1')
  AND (pg_catalog.jsonb_strip_nulls(j) - ARRAY[
   'auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en',
   'tipo_registro','evento_ref','evento_material_sha256','operador_login',
   'plan_sha256','aprobacion_ref','publicacion_vinculo_f1_detalle',
   'accion','modulo_id','recurso_ref','finalidad_ref','resultado','motivo_ref',
   'proceso','canal','correlacion_ref'])='{}'::jsonb
  AND j->>'evento_ref' ~ '^evento_[0-9a-f]{32}$'
  AND j->>'evento_material_sha256' ~ '^[0-9a-f]{64}$'
  AND j->>'plan_sha256' ~ '^[0-9a-f]{64}$'
  AND j->>'aprobacion_ref' ~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
  AND ((j->>'tipo_registro'='publicacion_vinculo_corporativo_f1'
    AND j->>'accion'='publicar_vinculo_corporativo_desde_fuente_v1'
    AND j->>'motivo_ref'='vinculo_publicado'
    AND j#>>'{publicacion_vinculo_f1_detalle,estado}'='publicado')
   OR (j->>'tipo_registro'='comprobacion_vinculo_corporativo_f1'
    AND j->>'accion'='comprobar_publicacion_vinculo_corporativo_f1'
    AND j->>'motivo_ref'='preimagen_comprobada'
    AND j#>>'{publicacion_vinculo_f1_detalle,estado}' IN ('publicable','replay')))
  AND j->>'modulo_id'='contexto_actor'
  AND j->>'recurso_ref' ~ '^vinculo_corporativo:[0-9a-f]{32}$'
  AND j->>'finalidad_ref'='publicacion_corporativa_f1'
  AND j->>'resultado'='permitido'
  AND j->>'proceso'='postgresql' AND j->>'canal'='operacion_tecnica_privada'
  AND j->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$'
  AND pg_catalog.jsonb_typeof(j->'publicacion_vinculo_f1_detalle')='object'
  AND (j->'publicacion_vinculo_f1_detalle') ?& ARRAY['aprobacion_sha256','procedencia_sha256','version','estado']
  AND j->'publicacion_vinculo_f1_detalle' = pg_catalog.jsonb_build_object(
   'aprobacion_sha256',j#>>'{publicacion_vinculo_f1_detalle,aprobacion_sha256}',
   'procedencia_sha256',j#>>'{publicacion_vinculo_f1_detalle,procedencia_sha256}',
   'version',j#>>'{publicacion_vinculo_f1_detalle,version}',
   'estado',j#>>'{publicacion_vinculo_f1_detalle,estado}')
  AND j#>>'{publicacion_vinculo_f1_detalle,aprobacion_sha256}' ~ '^[0-9a-f]{64}$'
  AND j#>>'{publicacion_vinculo_f1_detalle,procedencia_sha256}' ~ '^[0-9a-f]{64}$'
  AND j#>>'{publicacion_vinculo_f1_detalle,version}' ~ '^[1-9][0-9]{0,19}$',false)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.auditoria_publicacion_vinculo_f1_valida(jsonb) FROM PUBLIC;
CREATE FUNCTION vec_autorizacion_atestada_v3.auditoria_intento_vinculo_f1_valida(j jsonb)
RETURNS boolean LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog,pg_temp AS $f$
 SELECT coalesce(j->>'tipo_registro'='intento_vinculo_corporativo_f1'
  AND (jsonb_strip_nulls(j)-ARRAY[
   'auditoria_ref','secuencia','anterior_sha256','huella_sha256','registrada_en',
   'tipo_registro','evento_ref','evento_material_sha256','operador_login',
   'intento_vinculo_f1_sha256','accion','modulo_id','recurso_ref','finalidad_ref',
   'resultado','motivo_ref','proceso','canal','correlacion_ref'])='{}'::jsonb
  AND j->>'evento_ref' ~ '^evento_[0-9a-f]{32}$'
  AND j->>'evento_material_sha256' ~ '^[0-9a-f]{64}$'
  AND j->>'intento_vinculo_f1_sha256' ~ '^[0-9a-f]{64}$'
  AND j->>'accion' IN ('publicar_vinculo_corporativo_desde_fuente_v1','comprobar_publicacion_vinculo_corporativo_f1')
  AND j->>'modulo_id'='contexto_actor'
  AND j->>'recurso_ref' ~ '^solicitud_vinculo_corporativo:[0-9a-f]{32}$'
  AND j->>'finalidad_ref'='publicacion_corporativa_f1'
  AND j->>'resultado'='denegado' AND j->>'motivo_ref'='vinculo_denegado'
  AND j->>'proceso'='postgresql' AND j->>'canal'='operacion_tecnica_privada'
  AND j->>'correlacion_ref' ~ '^correlacion_[0-9a-f]{32}$',false)
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.auditoria_intento_vinculo_f1_valida(jsonb) FROM PUBLIC;
DO $familia$
DECLARE anterior text;nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM
 '6cd7353335b9815259d600304c89440b96d7123284700668f58d71a099cecd8f'
 THEN RAISE EXCEPTION 'AD258: PARO clave=familia actual=deriva esperado=post_AD251' USING ERRCODE='55000'; END IF;
 nueva:=left(anterior,length(anterior)-1)||
  ' OR (tipo_registro IN (''publicacion_vinculo_corporativo_f1'',''comprobacion_vinculo_corporativo_f1'') AND vec_autorizacion_atestada_v3.auditoria_publicacion_vinculo_f1_valida(to_jsonb(auditoria_consumo_v3)))'||
  ' OR (tipo_registro = ''intento_vinculo_corporativo_f1'' AND vec_autorizacion_atestada_v3.auditoria_intento_vinculo_f1_valida(to_jsonb(auditoria_consumo_v3))))';
 ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 DROP CONSTRAINT auditoria_tipo_disjunto_v4;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_tipo_disjunto_v4 '||nueva;
END $familia$;
ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3 ADD CONSTRAINT auditoria_publicacion_vinculo_f1_disjunta
 CHECK ((tipo_registro IN ('publicacion_vinculo_corporativo_f1','comprobacion_vinculo_corporativo_f1') OR publicacion_vinculo_f1_detalle IS NULL)
  AND (tipo_registro='intento_vinculo_corporativo_f1' OR intento_vinculo_f1_sha256 IS NULL));

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE material text; msha text; previo record; seq numeric; anterior text; instante timestamptz(6); ref text; huella text;
 tipo text;accion text;motivo text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off' OR current_setting('role')<>'none'
 OR current_setting('TimeZone')<>'UTC'
 OR NOT pg_has_role(session_user,'vec_contexto_actor_corporativo_rrhh_publicador','member')
 OR pg_has_role(session_user,'vec_contexto_actor_v1_runtime','member')
 THEN RAISE EXCEPTION 'AD258: PARO clave=operador actual=incompatible esperado=login_privilegiado' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(e) IS DISTINCT FROM 'object' OR octet_length(e::text)>4096
 OR e <> jsonb_build_object('evento_ref',e->>'evento_ref','operador_login',e->>'operador_login',
  'plan_sha256',e->>'plan_sha256','aprobacion_ref',e->>'aprobacion_ref',
  'aprobacion_sha256',e->>'aprobacion_sha256','procedencia_sha256',e->>'procedencia_sha256',
  'version',e->>'version','correlacion_ref',e->>'correlacion_ref',
  'tipo',e->>'tipo','estado',e->>'estado')
 OR e->>'operador_login' IS DISTINCT FROM session_user::text
 OR e->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR e->>'plan_sha256' !~ '^[0-9a-f]{64}$'
 OR e->>'aprobacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$'
 OR e->>'aprobacion_sha256' !~ '^[0-9a-f]{64}$'
 OR e->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
 OR e->>'version' !~ '^[1-9][0-9]{0,19}$'
 OR e->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR NOT ((e->>'tipo'='publicacion' AND e->>'estado'='publicado')
  OR (e->>'tipo'='comprobacion' AND e->>'estado' IN ('publicable','replay')))
 THEN RAISE EXCEPTION 'AD258: PARO clave=evento actual=incompatible esperado=exacto' USING ERRCODE='22023'; END IF;
 tipo:=CASE WHEN e->>'tipo'='publicacion' THEN 'publicacion_vinculo_corporativo_f1' ELSE 'comprobacion_vinculo_corporativo_f1' END;
 accion:=CASE WHEN e->>'tipo'='publicacion' THEN 'publicar_vinculo_corporativo_desde_fuente_v1' ELSE 'comprobar_publicacion_vinculo_corporativo_f1' END;
 motivo:=CASE WHEN e->>'tipo'='publicacion' THEN 'vinculo_publicado' ELSE 'preimagen_comprobada' END;
 material:=e::text;msha:=encode(sha256(convert_to('vec.auditoria.acto-vinculo-f1.v1:'||material,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:evento-admin:'||(e->>'evento_ref'),0));
 SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.registrada_en,a.evento_material_sha256,a.tipo_registro
 INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=e->>'evento_ref';
 IF FOUND THEN
  IF previo.tipo_registro IS DISTINCT FROM tipo
   OR previo.evento_material_sha256 IS DISTINCT FROM msha
  THEN RAISE EXCEPTION 'AD258: PARO clave=evento actual=divergente esperado=original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT previo.auditoria_ref,previo.secuencia,previo.huella_sha256,previo.registrada_en; RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT seq,anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF seq>=9007199254740991::numeric THEN RAISE EXCEPTION 'AD258: secuencia agotada' USING ERRCODE='22003'; END IF;
 seq:=seq+1;instante:=clock_timestamp();ref:='aud_v3_f1_'||substr(e->>'evento_ref',8,32);
 huella:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.acto-vinculo-f1.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(seq::text)||vec_autorizacion_atestada_v3.encuadrar_mac(anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(ref)||vec_autorizacion_atestada_v3.encuadrar_mac(msha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,plan_sha256,aprobacion_ref,
  publicacion_vinculo_f1_detalle,accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(ref,seq,anterior,huella,instante,tipo,
  e->>'evento_ref',msha,(e->>'operador_login')::name,e->>'plan_sha256',e->>'aprobacion_ref',
  jsonb_build_object('aprobacion_sha256',e->>'aprobacion_sha256','procedencia_sha256',e->>'procedencia_sha256','version',e->>'version','estado',e->>'estado'),
  accion,'contexto_actor',
  'vinculo_corporativo:'||substr(e->>'plan_sha256',1,32),'publicacion_corporativa_f1','permitido',motivo,
  'postgresql','operacion_tecnica_privada',e->>'correlacion_ref');
 RETURN QUERY SELECT ref,seq,huella,instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(jsonb)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contexto_actor_v1_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_acto_vinculo_corporativo_f1(jsonb)
 TO vec_contexto_actor_v1_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_vinculo_corporativo_f1(e jsonb)
RETURNS TABLE(auditoria_ref text,secuencia numeric,huella_sha256 text,registrada_en timestamptz)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET lock_timeout='2s' SET statement_timeout='10s'
AS $f$
DECLARE material text;msha text;previo record;seq numeric;anterior text;instante timestamptz(6);ref text;huella text;accion text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
 OR current_setting('role')<>'none' OR current_setting('TimeZone')<>'UTC'
 OR NOT pg_has_role(session_user,'vec_contexto_actor_corporativo_rrhh_publicador','member')
 OR pg_has_role(session_user,'vec_contexto_actor_v1_runtime','member')
 THEN RAISE EXCEPTION 'AD258: PARO clave=operador_intento actual=incompatible esperado=login_publicador' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(e) IS DISTINCT FROM 'object' OR octet_length(e::text)>2048
 OR e<>jsonb_build_object('evento_ref',e->>'evento_ref','operador_login',e->>'operador_login',
  'solicitud_sha256',e->>'solicitud_sha256','modo',e->>'modo','correlacion_ref',e->>'correlacion_ref')
 OR e->>'operador_login' IS DISTINCT FROM session_user::text
 OR e->>'evento_ref' !~ '^evento_[0-9a-f]{32}$'
 OR e->>'solicitud_sha256' !~ '^[0-9a-f]{64}$'
 OR e->>'modo' NOT IN ('comprobar','publicar')
 OR e->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 THEN RAISE EXCEPTION 'AD258: PARO clave=intento actual=incompatible esperado=exacto' USING ERRCODE='22023'; END IF;
 accion:=CASE WHEN e->>'modo'='publicar' THEN 'publicar_vinculo_corporativo_desde_fuente_v1' ELSE 'comprobar_publicacion_vinculo_corporativo_f1' END;
 material:=e::text;msha:=encode(sha256(convert_to('vec.auditoria.intento-vinculo-f1.v1:'||material,'UTF8')),'hex');
 PERFORM pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:evento-admin:'||(e->>'evento_ref'),0));
 SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.registrada_en,a.evento_material_sha256,a.tipo_registro
 INTO previo FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a WHERE a.evento_ref=e->>'evento_ref';
 IF FOUND THEN
  IF previo.tipo_registro IS DISTINCT FROM 'intento_vinculo_corporativo_f1'
   OR previo.evento_material_sha256 IS DISTINCT FROM msha
  THEN RAISE EXCEPTION 'AD258: PARO clave=intento actual=divergente esperado=original' USING ERRCODE='23505'; END IF;
  RETURN QUERY SELECT previo.auditoria_ref,previo.secuencia,previo.huella_sha256,previo.registrada_en; RETURN;
 END IF;
 SELECT r.secuencia_previa,r.anterior_sha256 INTO STRICT seq,anterior
 FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() r;
 IF seq>=9007199254740991::numeric THEN RAISE EXCEPTION 'AD258: secuencia agotada' USING ERRCODE='22003'; END IF;
 seq:=seq+1;instante:=clock_timestamp();ref:='aud_v3_f1_'||substr(e->>'evento_ref',8,32);
 huella:=encode(sha256(vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento-vinculo-f1.v1')||
  vec_autorizacion_atestada_v3.encuadrar_mac(seq::text)||vec_autorizacion_atestada_v3.encuadrar_mac(anterior)||
  vec_autorizacion_atestada_v3.encuadrar_mac(ref)||vec_autorizacion_atestada_v3.encuadrar_mac(msha)||
  vec_autorizacion_atestada_v3.encuadrar_mac(to_char(instante AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(
  auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,tipo_registro,
  evento_ref,evento_material_sha256,operador_login,intento_vinculo_f1_sha256,
  accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,proceso,canal,correlacion_ref)
 VALUES(ref,seq,anterior,huella,instante,'intento_vinculo_corporativo_f1',e->>'evento_ref',msha,
  (e->>'operador_login')::name,e->>'solicitud_sha256',accion,'contexto_actor',
  'solicitud_vinculo_corporativo:'||substr(e->>'solicitud_sha256',1,32),
  'publicacion_corporativa_f1','denegado','vinculo_denegado','postgresql','operacion_tecnica_privada',e->>'correlacion_ref');
 RETURN QUERY SELECT ref,seq,huella,instante;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_vinculo_corporativo_f1(jsonb)
 FROM PUBLIC,vec_contexto_actor_v1_runtime;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_vinculo_corporativo_f1(jsonb)
 TO vec_contexto_actor_v1_propietario;
RESET ROLE;
COMMIT;
