\set ON_ERROR_STOP on
-- CT121 sobre la estructura real restaurada. El volcado conserva un aviso de
-- sucesor (CT62) que cuelga de la continuación de una renuncia (expediente B).
-- Dentro de una transacción que se revierte, la resolución de esa renuncia se
-- convierte en una expiración confirmada con la forma de CT111 (sin
-- justificante, con contacto y plazo expirado) y se lee el antecedente del
-- aviso del sucesor con la cuenta de ejecución. Con CT121 se admite; sin
-- CT121, no. Uso: psql -v esperado=si|no
\set exp_b 'expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7'
\set res 'resolucion:0c5fdea4-be11-4bdd-bd0b-5bc035dd9ae0'
\set sucesor 'llamamiento:eipmgkkfjncbalgebihinpeeajhllfmkoikjhiodlbcdlhaohmgpojefdfilcmoe'
DO $r$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_ct121_runtime') THEN
  CREATE ROLE vec_ct121_runtime LOGIN INHERIT IN ROLE vec_contratacion_temporal_ejecutor;
 END IF;
END $r$;
GRANT CONNECT ON DATABASE postgres TO vec_ct121_runtime;

CREATE FUNCTION pg_temp.exigir(c boolean, msg text) RETURNS text LANGUAGE plpgsql AS $$
BEGIN IF c IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %', msg; END IF; RETURN 'ok'; END $$;
CREATE FUNCTION pg_temp.leer(p_org text, p_exp text, p_llam text) RETURNS text LANGUAGE plpgsql AS $$
DECLARE v bigint;
BEGIN
 SELECT version_actual INTO STRICT v FROM vec_contratacion_temporal.leer_expediente_aviso_confirmado_v1(p_org,p_exp,p_llam);
 RETURN 'admitido';
EXCEPTION WHEN others THEN RETURN SQLSTATE;
END $$;
DO $$ BEGIN EXECUTE format('GRANT USAGE ON SCHEMA %I TO vec_ct121_runtime',pg_my_temp_schema()::regnamespace); END $$;

SELECT organizacion_ref AS org FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE resolucion_ref=:'res' \gset

BEGIN;
-- La renuncia admitida hoy sigue admitida con o sin CT121.
SET SESSION AUTHORIZATION vec_ct121_runtime;
SELECT pg_temp.exigir(pg_temp.leer(:'org',:'exp_b',:'sucesor')='admitido','antecedente de renuncia');
RESET SESSION AUTHORIZATION;
-- La misma continuación, ahora tras una expiración confirmada (forma CT111).
SET LOCAL session_replication_role = replica;
UPDATE vec_contratacion_temporal.resolucion_manual_respuesta_rrhh
   SET solicitud_json=jsonb_set(solicitud_json,'{Respuesta}','"expiracion_gobernada"'),
       material_json=jsonb_set(material_json,'{Solicitud,Respuesta}','"expiracion_gobernada"'),
       material=jsonb_set(material::jsonb,'{Solicitud,Respuesta}','"expiracion_gobernada"')::text,
       material_huella_sha256=encode(sha256(convert_to(jsonb_set(material::jsonb,'{Solicitud,Respuesta}','"expiracion_gobernada"')::text,'UTF8')),'hex'),
       recibo_json=jsonb_set(jsonb_set(recibo_json,'{Solicitud,Respuesta}','"expiracion_gobernada"'),'{IntencionSiguiente,Solicitud,Respuesta}','"expiracion_gobernada"'),
       comando_siguiente_json=jsonb_set(comando_siguiente_json,'{justificante_ref}','null'),
       estado_plazo='expirado', justificante_ref=NULL, contacto_ref='contacto:ct121:efectivo'
 WHERE resolucion_ref=:'res';
SET LOCAL session_replication_role = origin;
SELECT pg_temp.exigir((SELECT solicitud_json->>'Respuesta' FROM vec_contratacion_temporal.resolucion_manual_respuesta_rrhh WHERE resolucion_ref=:'res')='expiracion_gobernada','expiración simulada');
SET SESSION AUTHORIZATION vec_ct121_runtime;
SELECT pg_temp.leer(:'org',:'exp_b',:'sucesor') AS tras_expiracion \gset
RESET SESSION AUTHORIZATION;
\if :{?esperado}
\else
\set esperado si
\endif
SELECT pg_temp.exigir(CASE WHEN :'esperado'='si' THEN :'tras_expiracion'='admitido' ELSE :'tras_expiracion'<>'admitido' END,
  'antecedente tras expiración ('||:'esperado'||'): '||:'tras_expiracion');
-- Otro llamamiento distinto del abierto por la continuación no se admite.
SET SESSION AUTHORIZATION vec_ct121_runtime;
SELECT pg_temp.exigir(pg_temp.leer(:'org',:'exp_b','llamamiento:ajeno:ct121')<>'admitido','llamamiento ajeno a la continuación');
RESET SESSION AUTHORIZATION;
ROLLBACK;
SELECT 'CT121 OK' AS resultado;
