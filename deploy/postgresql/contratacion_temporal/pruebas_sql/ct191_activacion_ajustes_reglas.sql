\set ON_ERROR_STOP on
-- Prueba sintética en clon PG18 con CT190/CT191. Todo termina en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
DO $acl$
DECLARE f regprocedure:='vec_contratacion_temporal.leer_activacion_regla_base_v1()'::regprocedure;
 o regprocedure:='vec_contratacion_temporal.operar_ajustes_reglas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF NOT has_function_privilege('vec_contratacion_temporal_ejecutor',f,'EXECUTE')
  OR has_function_privilege('vec_contratacion_temporal_consultor_rrhh',f,'EXECUTE')
  OR has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.regla_base_activacion_v1','SELECT')
  OR NOT has_function_privilege('vec_contratacion_temporal_ejecutor',o,'EXECUTE')
  OR (SELECT NOT relforcerowsecurity FROM pg_class WHERE oid='vec_contratacion_temporal.regla_base_activacion_v1'::regclass)
 THEN RAISE EXCEPTION 'CT191: ACL o RLS erróneas'; END IF;
END $acl$;

-- La fachada V3 real se sustituye solo dentro de esta transacción para aislar
-- la guarda CT; CT148 ya prueba la frontera criptográfica por separado.
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_ct_v3_atestada(
 p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT 'decision:prueba:'||gen_random_uuid()::text,'vec.contratacion_temporal.reglas',repeat('a',64),
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
  'auditoria:prueba:'||gen_random_uuid()::text,clock_timestamp(),true
$f$;
CREATE ROLE prueba_ct191 NOLOGIN;
GRANT vec_contratacion_temporal_ejecutor TO prueba_ct191;
CREATE ROLE prueba_ct191_ajeno NOLOGIN;
CREATE FUNCTION pg_temp.operar(p jsonb)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_contratacion_temporal.operar_ajustes_reglas_v1(p,'\x7b7d'::bytea,
  convert_to(jsonb_build_object('principal_id','principal:sintetico:ct191',
   'accion','contratacion_temporal.reglas.ajustar')::text,'UTF8'),
  '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea)
$f$;
CREATE FUNCTION pg_temp.material(p_clave uuid,p_esperada bigint,p_version bigint,
 p_huella text,p_canonico text,p_anterior text,p_nuevo text)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('operacion','ajustar','organizacion_ref','organizacion:desarrollo:dipgra',
  'catalogo_id','vec.contratacion_temporal.reglas.ajustes','clave_idempotencia',p_clave::text,
  'version_esperada',p_esperada,'base_version',p_version,'base_huella_sha256',p_huella,
  'ajustes_canonico',p_canonico,
  'ajustes_huella_sha256',encode(sha256(convert_to(p_canonico,'UTF8')),'hex'),
  'cambios',jsonb_build_array(jsonb_build_object('regla_clave','c03.plazo_fiscalizacion',
   'campo','cantidad','anterior',p_anterior,'nuevo',p_nuevo)),
  'motivo_clave','acuerdo_rrhh')
$f$;
CREATE FUNCTION pg_temp.rechaza(p jsonb,p_estado text,p_caso text)
RETURNS void LANGUAGE plpgsql AS $f$
BEGIN
 PERFORM pg_temp.operar(p);
 RAISE EXCEPTION 'CT191: caso % admitido',p_caso;
EXCEPTION WHEN others THEN
 IF SQLSTATE<>p_estado OR SQLERRM LIKE 'CT191: caso % admitido'
 THEN RAISE EXCEPTION 'CT191: caso % dio % %',p_caso,SQLSTATE,SQLERRM; END IF;
END $f$;

SET SESSION AUTHORIZATION prueba_ct191_ajeno;
DO $ajeno$
BEGIN
 PERFORM vec_contratacion_temporal.leer_activacion_regla_base_v1();
 RAISE EXCEPTION 'CT191: lectura ajena admitida';
EXCEPTION WHEN insufficient_privilege THEN NULL;
END $ajeno$;
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ct191;
DO $prueba$
DECLARE a record; k1 uuid:=gen_random_uuid(); k2 uuid:=gen_random_uuid();
 k3 uuid:=gen_random_uuid(); p1 jsonb; r1 jsonb; r2 jsonb;
 canonico text; huella text; canonico2 text; huella2 text;
 v bigint; s bigint; c1 text:='{"c03.plazo_fiscalizacion":{"cantidad":"7"}}';
 c2 text:='{"c03.plazo_fiscalizacion":{"cantidad":"6"}}';
BEGIN
 SELECT * INTO STRICT a FROM vec_contratacion_temporal.leer_activacion_regla_base_v1();
 IF a.estado<>'sin_publicar' OR a.secuencia IS NOT NULL THEN
  RAISE EXCEPTION 'CT191: ausencia incorrecta'; END IF;
 p1:=pg_temp.material(k1,0,1,repeat('b',64),c1,'10','7');
 PERFORM pg_temp.rechaza(p1,'55000','sin base');
 PERFORM pg_temp.rechaza(p1-'base_version','22023','sin versión');
 PERFORM pg_temp.rechaza(p1-'base_huella_sha256','22023','sin huella');
END $prueba$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $base$
DECLARE v bigint; s bigint; canonico text; huella text;
BEGIN
 SELECT coalesce(max(version),0)+1 INTO v FROM vec_contratacion_temporal.regla_base_publicada_v1;
 SELECT coalesce(max(secuencia),0) INTO s FROM vec_contratacion_temporal.regla_base_activacion_v1;
 canonico:=jsonb_build_object('id','vec.contratacion_temporal.reglas','version',v,
  'estado','publicado','aprobacion_ref','aprobacion:sintetica:ct191','entradas','[]'::jsonb)::text;
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.regla_base_publicada_v1
  (catalogo_id,version,huella_sha256,canonico,fuente_sha256,aprobacion_ref)
 VALUES ('vec.contratacion_temporal.reglas',v,huella,canonico,repeat('a',64),'aprobacion:sintetica:ct191');
 INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
  (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
 VALUES (s+1,s,true,'vec.contratacion_temporal.reglas',v,huella,'aprobacion:sintetica:ct191');
 PERFORM set_config('ct191.base_version',v::text,true);
 PERFORM set_config('ct191.base_huella',huella,true);
END $base$;
RESET ROLE;
SET SESSION AUTHORIZATION prueba_ct191;
DO $positivo$
DECLARE a record; k uuid:=gen_random_uuid(); p jsonb; r jsonb; v bigint;
BEGIN
 SELECT * INTO STRICT a FROM vec_contratacion_temporal.leer_activacion_regla_base_v1();
 IF a.estado<>'activa' OR a.version::text<>current_setting('ct191.base_version')
  OR a.huella_sha256<>current_setting('ct191.base_huella')
  OR a.aprobacion_ref<>'aprobacion:sintetica:ct191' THEN
  RAISE EXCEPTION 'CT191: lectura activa incorrecta'; END IF;
 v:=a.version;
 PERFORM pg_temp.rechaza(pg_temp.material(k,0,v+1,a.huella_sha256,
  '{"c03.plazo_fiscalizacion":{"cantidad":"7"}}','10','7'),'40001','versión base ajena');
 PERFORM pg_temp.rechaza(pg_temp.material(k,0,v,repeat('b',64),
  '{"c03.plazo_fiscalizacion":{"cantidad":"7"}}','10','7'),'40001','huella ajena');
 p:=pg_temp.material(k,0,v,a.huella_sha256,
  '{"c03.plazo_fiscalizacion":{"cantidad":"7"}}','10','7');
 r:=pg_temp.operar(p);
 IF (r->>'replay')::boolean OR r#>>'{recibo,version}'<>'1' THEN
  RAISE EXCEPTION 'CT191: primer ajuste inválido'; END IF;
 PERFORM set_config('ct191.replay_material',p::text,true);
 PERFORM set_config('ct191.replay_recibo',r#>>'{recibo,recibo_ref}',true);
END $positivo$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $rotar$
DECLARE v bigint; s bigint; canonico text; huella text;
BEGIN
 SELECT coalesce(max(version),0)+1 INTO v FROM vec_contratacion_temporal.regla_base_publicada_v1;
 SELECT max(secuencia) INTO s FROM vec_contratacion_temporal.regla_base_activacion_v1;
 canonico:=jsonb_build_object('id','vec.contratacion_temporal.reglas','version',v,
  'estado','publicado','aprobacion_ref','aprobacion:sintetica:ct191-v2','entradas','[]'::jsonb)::text;
 huella:=encode(sha256(convert_to(canonico,'UTF8')),'hex');
 INSERT INTO vec_contratacion_temporal.regla_base_publicada_v1
  (catalogo_id,version,huella_sha256,canonico,fuente_sha256,aprobacion_ref)
 VALUES ('vec.contratacion_temporal.reglas',v,huella,canonico,repeat('c',64),'aprobacion:sintetica:ct191-v2');
 INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
  (secuencia,secuencia_esperada,activa,catalogo_id,version,huella_sha256,aprobacion_ref)
 VALUES (s+1,s,true,'vec.contratacion_temporal.reglas',v,huella,'aprobacion:sintetica:ct191-v2');
 PERFORM set_config('ct191.base2_version',v::text,true);
 PERFORM set_config('ct191.base2_huella',huella,true);
END $rotar$;
RESET ROLE;
SET SESSION AUTHORIZATION prueba_ct191;
DO $replay$
DECLARE p jsonb; r jsonb; p2 jsonb; r2 jsonb;
BEGIN
 p:=current_setting('ct191.replay_material')::jsonb;
 r:=pg_temp.operar(p);
 IF NOT (r->>'replay')::boolean OR r#>>'{recibo,recibo_ref}'<>current_setting('ct191.replay_recibo')
 THEN RAISE EXCEPTION 'CT191: replay histórico perdido'; END IF;
 PERFORM pg_temp.rechaza(pg_temp.material(gen_random_uuid(),1,
  current_setting('ct191.base_version')::bigint,current_setting('ct191.base_huella'),
  '{"c03.plazo_fiscalizacion":{"cantidad":"6"}}','7','6'),'40001','clave nueva base antigua');
 p2:=pg_temp.material(gen_random_uuid(),1,current_setting('ct191.base2_version')::bigint,
  current_setting('ct191.base2_huella'),
  '{"c03.plazo_fiscalizacion":{"cantidad":"6"}}','7','6');
 r2:=pg_temp.operar(p2);
 IF (r2->>'replay')::boolean OR r2#>>'{recibo,version}'<>'2' THEN
  RAISE EXCEPTION 'CT191: base nueva no publicó ajuste'; END IF;
END $replay$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
INSERT INTO vec_contratacion_temporal.regla_base_activacion_v1
 (secuencia,secuencia_esperada,activa,aprobacion_ref)
SELECT max(secuencia)+1,max(secuencia),false,'aprobacion:sintetica:ct191-inactiva'
FROM vec_contratacion_temporal.regla_base_activacion_v1;
RESET ROLE;
SET SESSION AUTHORIZATION prueba_ct191;
DO $inactiva$
DECLARE a record;
BEGIN
 SELECT * INTO STRICT a FROM vec_contratacion_temporal.leer_activacion_regla_base_v1();
 IF a.estado<>'inactiva' OR a.version IS NOT NULL OR a.huella_sha256 IS NOT NULL
 THEN RAISE EXCEPTION 'CT191: desactivación incorrecta'; END IF;
 PERFORM pg_temp.rechaza(pg_temp.material(gen_random_uuid(),1,
  current_setting('ct191.base2_version')::bigint,current_setting('ct191.base2_huella'),
  '{"c03.plazo_fiscalizacion":{"cantidad":"6"}}','7','6'),'55000','base inactiva');
END $inactiva$;
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $historia$
BEGIN
 IF (SELECT count(*) FROM vec_contratacion_temporal.regla_ajuste_version_v1)<>2
  OR (SELECT count(*) FROM vec_contratacion_temporal.regla_ajuste_cambio_v1)<>2
  OR (SELECT count(*) FROM vec_contratacion_temporal.regla_ajuste_outbox_v1)<>2
 THEN RAISE EXCEPTION 'CT191: historia o outbox alterados'; END IF;
END $historia$;
RESET ROLE;
SELECT 'CT191-PRUEBAS-OK' AS resultado;
ROLLBACK;
