\set ON_ERROR_STOP on
-- Sólo para el clon desechable: fixture transaccional sobre un alta E2 sintética.
BEGIN ISOLATION LEVEL SERIALIZABLE;
CREATE ROLE ct193_prueba_ejecutor LOGIN;
GRANT vec_contratacion_temporal_ejecutor TO ct193_prueba_ejecutor;
CREATE TEMP TABLE ct193_prueba_lectura AS
SELECT alias.alias_hmac, i.organizacion_ref, i.actor_ref, i.perfil_ref,
       m.expediente_ref, NULL::bytea AS raw_esperada
  FROM vec_contratacion_temporal.confirmacion_agregado_alta m
  JOIN vec_contratacion_temporal.alias_ambito_alta alias
    ON alias.ambito_raiz_hmac=m.ambito_hmac
  JOIN vec_contratacion_temporal.identidad_reserva_alta i
    ON i.ambito_hmac=m.ambito_hmac
  JOIN vec_contratacion_temporal.expediente_alta_version v
    ON v.expediente_ref=m.expediente_ref AND v.version=1
 WHERE pg_catalog.convert_from(v.alta_canonica,'UTF8')::jsonb->>'esquema'
       ='vec.contratacion-temporal.efecto-alta.v2'
   AND pg_catalog.convert_from(v.alta_canonica,'UTF8')::jsonb
       #>'{solicitud,periodo,fin}' IS NOT NULL
 ORDER BY m.confirmada_en DESC LIMIT 1;
DO $pre$
BEGIN
 IF (SELECT count(*) FROM pg_temp.ct193_prueba_lectura)<>1
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.alias_ambito_alta
                WHERE alias_hmac='hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||pg_catalog.repeat('f',64)) THEN
   RAISE EXCEPTION 'CT193: fixture E2 o ámbito ausente incompatible';
 END IF;
END $pre$;

CREATE FUNCTION pg_temp.ct193_verificar_lectura(p_fase text)
RETURNS void LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp
AS $funcion$
DECLARE
 p record;
 v_estado text;
 v_raw bytea;
 v_rechazo boolean;
BEGIN
 SELECT * INTO STRICT p FROM pg_temp.ct193_prueba_lectura;
 IF p_fase='antes' THEN
   SELECT l.estado,l.instantanea INTO STRICT v_estado,v_raw
     FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
       'hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:'||pg_catalog.repeat('f',64),
       p.organizacion_ref,p.actor_ref,p.perfil_ref) l;
   IF v_estado IS DISTINCT FROM 'ausente' OR v_raw IS NOT NULL THEN
     RAISE EXCEPTION 'CT193: ausencia confundida';
   END IF;
   SELECT l.estado,l.instantanea INTO STRICT v_estado,v_raw
     FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
       p.alias_hmac,p.organizacion_ref,p.actor_ref,p.perfil_ref) l;
   IF v_estado IS DISTINCT FROM 'legado_v2' OR v_raw IS NOT NULL THEN
     RAISE EXCEPTION 'CT193: legado E2 confundido';
   END IF;
   v_rechazo:=false;
   BEGIN
     PERFORM * FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
       p.alias_hmac,p.organizacion_ref,'actor:ajeno',p.perfil_ref);
   EXCEPTION WHEN SQLSTATE '42501' THEN v_rechazo:=true;
   END;
   IF NOT v_rechazo THEN RAISE EXCEPTION 'CT193: identidad ajena permitida'; END IF;
 ELSIF p_fase='confirmada_v3' THEN
   SELECT l.estado,l.instantanea INTO STRICT v_estado,v_raw
     FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
       p.alias_hmac,p.organizacion_ref,p.actor_ref,p.perfil_ref) l;
   IF v_estado IS DISTINCT FROM 'confirmada_v3'
      OR v_raw IS DISTINCT FROM p.raw_esperada THEN
     RAISE EXCEPTION 'CT193: snapshot E3 no recuperada';
   END IF;
 ELSIF p_fase='corrupta' THEN
   v_rechazo:=false;
   BEGIN
     PERFORM * FROM vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
       p.alias_hmac,p.organizacion_ref,p.actor_ref,p.perfil_ref);
   EXCEPTION WHEN SQLSTATE '55000' THEN v_rechazo:=true;
   END;
   IF NOT v_rechazo THEN RAISE EXCEPTION 'CT193: E3 dañada confundida con ausencia'; END IF;
 ELSE
   RAISE EXCEPTION 'CT193: fase de prueba inválida';
 END IF;
END $funcion$;
GRANT EXECUTE ON FUNCTION pg_temp.ct193_verificar_lectura(text)
 TO ct193_prueba_ejecutor;

SET SESSION AUTHORIZATION ct193_prueba_ejecutor;
SELECT pg_temp.ct193_verificar_lectura('antes');
RESET SESSION AUTHORIZATION;

-- La prueba altera sólo el clon y hace ROLLBACK. Bypassa triggers de historia
-- exclusivamente para cubrir las ramas de lectura; no acredita una alta E3.
SET LOCAL session_replication_role=replica;
DO $fixture_e3$
DECLARE
 v_ref text;
 v_efecto jsonb;
 v_solicitud jsonb;
 v_necesidad jsonb;
 v_raw bytea;
 v_canon bytea;
 v_huella text;
 v_huella_solicitud text;
BEGIN
 SELECT p.expediente_ref,pg_catalog.convert_from(v.alta_canonica,'UTF8')::jsonb
   INTO STRICT v_ref,v_efecto
   FROM pg_temp.ct193_prueba_lectura p
   JOIN vec_contratacion_temporal.expediente_alta_version v
     ON v.expediente_ref=p.expediente_ref AND v.version=1;
 v_raw:=pg_catalog.convert_to(
   '{"esquema":"vec.ct.necesidades_alta.v1","referencia":"catalogo:ct:prueba","version":1,"jornada_referencia_minutos":2250,"causas":[{"clave":"sustitucion","fecha_fin":"opcional","causa_fin":"reincorporacion_titular","maximo_meses":120,"campos_permitidos":[],"campos_obligatorios":[]}]}',
   'UTF8');
 v_necesidad:=pg_catalog.jsonb_build_object(
   'esquema','vec.ct.necesidad_alta.v1',
   'catalogo_ref','catalogo:ct:prueba', 'catalogo_version',1,
   'catalogo_huella_sha256',pg_catalog.encode(pg_catalog.sha256(v_raw),'hex'),
   'causa_clave','sustitucion','periodo',v_efecto#>'{solicitud,periodo}',
   'jornada_minutos',2250,'campos','{}'::jsonb,
   'catalogo_instantanea',pg_catalog.replace(pg_catalog.encode(v_raw,'base64'),E'\n',''));
 v_solicitud:=pg_catalog.jsonb_set(v_efecto->'solicitud','{motivo_clave}',
    '"sustitucion"'::jsonb);
 v_solicitud:=pg_catalog.jsonb_set(v_solicitud,'{necesidad}',v_necesidad);
 v_efecto:=pg_catalog.jsonb_set(v_efecto,'{esquema}',
    '"vec.contratacion-temporal.efecto-alta.v3"'::jsonb);
 v_efecto:=pg_catalog.jsonb_set(v_efecto,'{solicitud}',v_solicitud);
 IF vec_contratacion_temporal.necesidad_alta_valida_v3(v_solicitud) IS NOT TRUE THEN
   RAISE EXCEPTION 'CT193: fixture E3 inválida';
 END IF;
 v_canon:=vec_contratacion_temporal.reconstruir_efecto_alta_v3(v_efecto);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(v_canon),'hex');
 v_huella_solicitud:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(v_solicitud),
   'UTF8')),'hex');
 UPDATE vec_contratacion_temporal.expediente_alta_version
    SET alta_canonica=v_canon,huella_alta_sha256=v_huella,
        solicitud_huella_sha256=v_huella_solicitud
  WHERE expediente_ref=v_ref AND version=1;
 UPDATE vec_contratacion_temporal.confirmacion_agregado_alta
    SET huella_alta_sha256=v_huella WHERE expediente_ref=v_ref;
 UPDATE pg_temp.ct193_prueba_lectura SET raw_esperada=v_raw;
END $fixture_e3$;
SET LOCAL session_replication_role=origin;
SET SESSION AUTHORIZATION ct193_prueba_ejecutor;
SELECT pg_temp.ct193_verificar_lectura('confirmada_v3');
RESET SESSION AUTHORIZATION;

SET LOCAL session_replication_role=replica;
DO $fixture_corrupta$
DECLARE
 v_ref text;
 v_efecto jsonb;
 v_canon bytea;
 v_huella text;
 v_huella_solicitud text;
BEGIN
 SELECT p.expediente_ref,pg_catalog.convert_from(v.alta_canonica,'UTF8')::jsonb
   INTO STRICT v_ref,v_efecto
   FROM pg_temp.ct193_prueba_lectura p
   JOIN vec_contratacion_temporal.expediente_alta_version v
     ON v.expediente_ref=p.expediente_ref AND v.version=1;
 v_efecto:=pg_catalog.jsonb_set(v_efecto,
   '{solicitud,necesidad,catalogo_huella_sha256}',
   pg_catalog.to_jsonb(pg_catalog.repeat('0',64)));
 v_canon:=vec_contratacion_temporal.reconstruir_efecto_alta_v3(v_efecto);
 v_huella:=pg_catalog.encode(pg_catalog.sha256(v_canon),'hex');
 v_huella_solicitud:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(
      v_efecto->'solicitud'),'UTF8')),'hex');
 UPDATE vec_contratacion_temporal.expediente_alta_version
    SET alta_canonica=v_canon,huella_alta_sha256=v_huella,
        solicitud_huella_sha256=v_huella_solicitud
  WHERE expediente_ref=v_ref AND version=1;
 UPDATE vec_contratacion_temporal.confirmacion_agregado_alta
    SET huella_alta_sha256=v_huella WHERE expediente_ref=v_ref;
END $fixture_corrupta$;
SET LOCAL session_replication_role=origin;
SET SESSION AUTHORIZATION ct193_prueba_ejecutor;
SELECT pg_temp.ct193_verificar_lectura('corrupta');
RESET SESSION AUTHORIZATION;
ROLLBACK;
