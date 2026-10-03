\set ON_ERROR_STOP on
-- Ensayo focal B73 sobre clon sintético. El consumidor AD3 siguiente es un
-- DOBLE transaccional: este fichero comprueba la operación Bolsa, no acredita
-- autorización criptográfica real. Todo se revierte al terminar.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_situacion_participacion_v3_atestada(
 p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric,
 p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea)
RETURNS TABLE(decision_ref text, efecto_ref text, huella_efecto_sha256 text, consumo_huella_sha256 text, auditoria_ref text, consumida_en timestamptz, consumo_nuevo boolean)
LANGUAGE sql SECURITY DEFINER SET search_path = pg_catalog AS $f$
 SELECT 'decision:doble-b73', convert_from(p_payload, 'UTF8'),
        convert_from(p_decision, 'UTF8')::jsonb->>'contexto_recurso_huella_sha256',
        repeat('0', 64), 'auditoria:doble-b73', clock_timestamp(), true
$f$;

CREATE FUNCTION pg_temp.operar_b73(p_bolsa text, p_participacion text, p_operacion text,
 p_desde timestamptz, p_clave text, p_justificante_ref text DEFAULT 'justificante:b73',
 p_justificante_sha text DEFAULT NULL, p_validador text DEFAULT 'persona:validador-b73',
 p_actor text DEFAULT 'persona:rrhh-b73', p_replay_registrada_en timestamptz DEFAULT NULL,
 p_decision_actor text DEFAULT NULL, p_payload_ref text DEFAULT NULL)
RETURNS text LANGUAGE plpgsql AS $f$
DECLARE d bytea; v record;
BEGIN
 d := convert_to(jsonb_build_object('principal_id', coalesce(p_decision_actor,p_actor),
   'accion', 'bolsa.situacion_participacion.cambiar', 'modulo_id', 'bolsa',
   'tipo_recurso', 'participacion_bolsa', 'finalidad', 'gestion_situacion_participacion',
   'recurso_ref', p_participacion, 'campos_permitidos', '[]'::jsonb,
   'obligaciones', '[]'::jsonb, 'contexto_recurso_huella_sha256', repeat('a',64))::text, 'UTF8');
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.registrar_operacion_situacion_participacion_v1(
   p_bolsa,p_participacion,p_operacion,p_desde,NULL,'Acto expreso RRHH B73',p_actor,
   p_clave,'recibo:' || p_clave,coalesce(p_replay_registrada_en,p_desde),
   'resolucion',p_justificante_ref,coalesce(p_justificante_sha,repeat('b',64)),
   p_validador,p_desde,'\x00'::bytea,d,'\x00'::bytea,'\x00'::bytea,1,1,
   convert_to(coalesce(p_payload_ref,p_participacion),'UTF8'),'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
 RESET ROLE;
 RETURN CASE WHEN v.reutilizada THEN 'replay:' ELSE 'nuevo:' END || v.recibo_ref;
EXCEPTION WHEN OTHERS THEN
 RESET ROLE;
 RETURN 'error:' || SQLSTATE;
END $f$;

CREATE FUNCTION pg_temp.b2_b73(p_bolsa text, p_participacion text, p_desde timestamptz, p_clave text,
 p_actor text DEFAULT 'persona:rrhh-b73') RETURNS text LANGUAGE plpgsql AS $f$
DECLARE d bytea; v record;
BEGIN
 d := convert_to(jsonb_build_object('principal_id', p_actor,
   'accion', 'bolsa.situacion_participacion.cambiar', 'modulo_id', 'bolsa',
   'tipo_recurso', 'participacion_bolsa', 'finalidad', 'gestion_situacion_participacion',
   'recurso_ref', p_participacion, 'campos_permitidos', '[]'::jsonb,
   'obligaciones', '[]'::jsonb, 'contexto_recurso_huella_sha256', repeat('a',64))::text, 'UTF8');
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 SELECT * INTO STRICT v FROM vec_bolsa_llamamientos.registrar_situacion_participacion_v1(
   p_bolsa,p_participacion,'disponible',p_desde,NULL,'B2 no reincorpora',p_actor,
   p_clave,'recibo:' || p_clave,p_desde,'\x00'::bytea,d,'\x00'::bytea,'\x00'::bytea,1,1,
   convert_to(p_participacion,'UTF8'),'\x00'::bytea,'\x00'::bytea,'\x00'::bytea);
 RESET ROLE;
 RETURN CASE WHEN v.reutilizada THEN 'replay:' ELSE 'nuevo:' END || v.recibo_ref;
EXCEPTION WHEN OTHERS THEN
 RESET ROLE;
 RETURN 'error:' || SQLSTATE;
END $f$;

DO $test$
DECLARE b text; p text; t timestamptz; old_policy text[]; ver bigint; resultado text;
 before_rows bigint; before_sanciones bigint; first_desde timestamptz; historia_antes text;
BEGIN
 SELECT c.bolsa_ref,e.participacion_ref INTO STRICT b,p
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.constitucion_entrada e USING(instantanea_ref,version_instantanea)
 JOIN LATERAL (SELECT s.situacion FROM vec_bolsa_llamamientos.situacion_participacion s
               WHERE s.participacion_ref=e.participacion_ref ORDER BY s.desde DESC LIMIT 1) actual ON true
 WHERE actual.situacion='disponible' LIMIT 1;
 SELECT greatest(clock_timestamp(),max(desde)) + interval '1 hour' INTO t
 FROM vec_bolsa_llamamientos.situacion_participacion;
 SELECT count(*) INTO before_rows FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p;
 SELECT count(*) INTO before_sanciones FROM vec_bolsa_llamamientos.sancion_participacion WHERE participacion_ref=p;
 SELECT md5(string_agg(s::text,'|' ORDER BY s.desde)) INTO historia_antes
 FROM vec_bolsa_llamamientos.situacion_participacion s WHERE s.participacion_ref=p;
 SELECT version,transiciones INTO STRICT ver,old_policy
 FROM vec_bolsa_llamamientos.politica_transiciones_situacion ORDER BY version DESC LIMIT 1;
 IF 'excluido>disponible'=ANY(old_policy) THEN RAISE EXCEPTION 'B73: clon ya tiene opt-in'; END IF;
 IF pg_temp.operar_b73(b,p,'excluir',t,'b73:exclusion') <> 'nuevo:recibo:b73:exclusion' THEN
  RAISE EXCEPTION 'B73: exclusión preparatoria falló'; END IF;
 first_desde:=t;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '1 second','b73:reactivar') <> 'error:22023' THEN
  RAISE EXCEPTION 'B73: política sin opt-in permitió reincorporación'; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p) <> before_rows+1 THEN
  RAISE EXCEPTION 'B73: fallo dejó efecto'; END IF;
 IF vec_bolsa_llamamientos.transiciones_situacion_canonicas(old_policy || 'excluido>no_disponible'::text) IS NOT NULL THEN
  RAISE EXCEPTION 'B73: canon abrió otra salida de excluido'; END IF;
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 PERFORM * FROM vec_bolsa_llamamientos.publicar_politica_transiciones_situacion_v1(
   'catalogo:b73:opt-in',repeat('c',64),old_policy || 'excluido>disponible'::text);
 RESET ROLE;
 IF (SELECT max(version) FROM vec_bolsa_llamamientos.politica_transiciones_situacion) <> ver+1 THEN
  RAISE EXCEPTION 'B73: no publicó versión nueva'; END IF;
 IF pg_temp.b2_b73(b,p,t+interval '2 second','b73:b2') <> 'error:22023' THEN
  RAISE EXCEPTION 'B73: B2 salió de excluido'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:invalido','justificante:b73','mala') <> 'error:22023' THEN
  RAISE EXCEPTION 'B73: justificante inválido admitido'; END IF;
 resultado:=pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:actor','justificante:b73',NULL,'persona:validador-b73','');
 IF resultado <> 'error:23514' THEN
  RAISE EXCEPTION 'B73: actor inválido, resultado %', resultado; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:actor-ajeno',
      'justificante:b73',NULL,'persona:validador-b73','persona:rrhh-b73',NULL,
      'persona:otro-actor') <> 'error:42501' THEN
  RAISE EXCEPTION 'B73: decisión de actor ajeno admitida'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:payload-ajeno',
      'justificante:b73',NULL,'persona:validador-b73','persona:rrhh-b73',NULL,
      NULL,'participacion:ajena') <> 'error:42501' THEN
  RAISE EXCEPTION 'B73: material para recurso ajeno admitido'; END IF;
 IF pg_temp.operar_b73(b,'participacion:ajena','reactivar',t+interval '2 second','b73:ajeno') <> 'error:23503' THEN
  RAISE EXCEPTION 'B73: recurso ajeno admitido'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:reactivar') <> 'nuevo:recibo:b73:reactivar' THEN
  RAISE EXCEPTION 'B73: reincorporación expresa falló'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:reactivar',
      'justificante:b73',NULL,'persona:validador-b73','persona:rrhh-b73',t+interval '1 day') <> 'replay:recibo:b73:reactivar' THEN
  RAISE EXCEPTION 'B73: replay idéntico falló'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:reactivar','justificante:alterado') <> 'error:VBS01' THEN
  RAISE EXCEPTION 'B73: replay con justificante alterado'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:reactivar',
      'justificante:b73',repeat('d',64)) <> 'error:VBS01' THEN
  RAISE EXCEPTION 'B73: replay con huella alterada'; END IF;
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '2 second','b73:reactivar',
      'justificante:b73',NULL,'persona:otro-validador') <> 'error:VBS01' THEN
  RAISE EXCEPTION 'B73: replay con validador alterado'; END IF;
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.situacion_participacion WHERE participacion_ref=p) <> before_rows+2
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.sancion_participacion WHERE participacion_ref=p) <> before_sanciones
    OR (SELECT count(*) FROM vec_bolsa_llamamientos.operacion_situacion_participacion
        WHERE participacion_ref=p AND desde=t+interval '2 second') <> 1 THEN
  RAISE EXCEPTION 'B73: replay duplicó efecto o creó sanción'; END IF;
 IF (SELECT politica_transiciones_version FROM vec_bolsa_llamamientos.situacion_participacion
      WHERE participacion_ref=p AND desde=t+interval '2 second') <> ver+1 THEN
  RAISE EXCEPTION 'B73: nueva situación no guardó versión de política'; END IF;
 -- Segunda exclusión: un efecto de sanción vivo de esa fila bloquea B8.
 IF pg_temp.operar_b73(b,p,'excluir',t+interval '3 second','b73:exclusion-sancion') <> 'nuevo:recibo:b73:exclusion-sancion' THEN
  RAISE EXCEPTION 'B73: segunda exclusión falló'; END IF;
 INSERT INTO vec_bolsa_llamamientos.sancion_participacion(
  sancion_ref,participacion_ref,bolsa_ref,consecuencia,consecuencia_etiqueta,efecto,causa,
  fecha_notificacion,resolucion_ref,resolucion_sha256,resuelta_por,regla_ref,regla_huella_sha256,
  suspension_hasta,recurso_vence,recurso_regla_ref,recurso_regla_huella_sha256,
  situacion_desde,recibo_ref,actor,registrada_en,clave_idempotencia)
 VALUES ('sancion:'||encode(sha256(convert_to('b73-prueba-sancion','UTF8')),'hex'),
  p,b,'baja','Baja sintética','excluir','Prueba B73',current_date,'resolucion:b73',
  repeat('d',64),'persona:validador-b73','regla:b73',repeat('e',64),NULL,current_date+10,
  'regla:recurso:b73',repeat('f',64),t+interval '3 second','recibo:b73:sancion',
  'persona:rrhh-b73',t+interval '3 second','b73:sancion');
 IF pg_temp.operar_b73(b,p,'reactivar',t+interval '4 second','b73:bloqueada') <> 'error:22023' THEN
  RAISE EXCEPTION 'B73: sanción viva permitió reincorporación'; END IF;
 IF pg_temp.operar_b73(b,p,'excluir',first_desde,'b73:exclusion') <> 'replay:recibo:b73:exclusion' THEN
  RAISE EXCEPTION 'B73: replay histórico de exclusión falló'; END IF;
 IF (SELECT md5(string_agg(s::text,'|' ORDER BY s.desde))
     FROM vec_bolsa_llamamientos.situacion_participacion s
     WHERE s.participacion_ref=p AND s.desde<t) IS DISTINCT FROM historia_antes THEN
  RAISE EXCEPTION 'B73: historia previa alterada'; END IF;
END $test$;

DO $acl$
DECLARE helper regprocedure := 'vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b73(text,text,text,timestamptz,timestamptz,text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,boolean)'::regprocedure;
BEGIN
 IF has_function_privilege('vec_bolsa_llamamientos_ejecutor',helper,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
               CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
               WHERE p.oid=helper AND a.grantee<>p.proowner AND a.privilege_type='EXECUTE') THEN
  RAISE EXCEPTION 'B73: helper interno ejecutable desde frontera externa';
 END IF;
 SET LOCAL ROLE vec_bolsa_llamamientos_ejecutor;
 BEGIN
  PERFORM * FROM vec_bolsa_llamamientos.registrar_situacion_participacion_interna_b73(
   NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,true);
  RAISE EXCEPTION 'B73: ejecución directa admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 RESET ROLE;
END $acl$;
SELECT 'b73_reincorporacion_b8_justificada: correcto' AS resultado;
ROLLBACK;
