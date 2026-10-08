\set ON_ERROR_STOP on
-- B88 sobre AD223+B88 recién instaladas en clon desechable. El doble de AD223
-- sólo aísla el contrato de persistencia; no acredita criptografía V3.
-- Toda la prueba y sus roles/datos sintéticos desaparecen con ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_ajustes_reglas_bolsa_v3_atestada(
 p_material jsonb,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT 'decision:prueba:'||gen_random_uuid()::text,'vec.bolsa.reglas',repeat('a',64),
  encode(sha256(convert_to(gen_random_uuid()::text,'UTF8')),'hex'),
  'auditoria:prueba:'||gen_random_uuid()::text,clock_timestamp(),true
$f$;
CREATE ROLE prueba_b88 NOLOGIN;
GRANT vec_bolsa_llamamientos_ejecutor TO prueba_b88;
CREATE ROLE prueba_b88_ajeno NOLOGIN;

CREATE FUNCTION pg_temp.material(k uuid,esperada bigint,canonico text,cambios jsonb,fecha text DEFAULT NULL)
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('operacion','ajustar','organizacion_ref','organizacion:desarrollo:dipgra',
  'catalogo_id','vec.bolsa.reglas.ajustes','clave_idempotencia',k::text,
  'version_esperada',esperada,'base_version',6,'base_huella_sha256',repeat('b',64),
  'ajustes_canonico',canonico,'ajustes_huella_sha256',encode(sha256(convert_to(canonico,'UTF8')),'hex'),
  'cambios',cambios,'motivo_clave','acuerdo_rrhh') ||
  CASE WHEN fecha IS NULL THEN '{}'::jsonb ELSE jsonb_build_object('vigente_desde',fecha) END
$f$;
CREATE FUNCTION pg_temp.operar(p jsonb,actor text DEFAULT 'per_prueba_rrhh_000000000000000000')
RETURNS jsonb LANGUAGE sql AS $f$
 SELECT vec_bolsa_llamamientos.operar_ajustes_reglas_v1(p,'\x7b7d'::bytea,
  convert_to(jsonb_build_object('principal_id',actor,'accion','bolsa.reglas.ajustar')::text,'UTF8'),
  '\x00'::bytea,'\x00'::bytea,1,1,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea,'\x00'::bytea)
$f$;
CREATE FUNCTION pg_temp.falla(p jsonb,estado text,nombre text)
RETURNS void LANGUAGE plpgsql AS $f$
DECLARE visto text;
BEGIN
 BEGIN
  PERFORM pg_temp.operar(p);
  RAISE EXCEPTION 'admitido' USING ERRCODE='P0001';
 EXCEPTION WHEN others THEN GET STACKED DIAGNOSTICS visto=RETURNED_SQLSTATE;
 END;
 IF visto IS DISTINCT FROM estado THEN
  RAISE EXCEPTION 'B88 fallo %: SQLSTATE %, esperado %',nombre,visto,estado;
 END IF;
END $f$;

SET SESSION AUTHORIZATION prueba_b88_ajeno;
DO $no_lectura$
BEGIN
 BEGIN
  PERFORM vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1('vec.bolsa.reglas.ajustes',now());
  RAISE EXCEPTION 'B88 fallo: lectura ajena admitida';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.leer_historial_ajustes_reglas_v1('vec.bolsa.reglas.ajustes',10,NULL,now());
  RAISE EXCEPTION 'B88 fallo: historial ajeno admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM vec_bolsa_llamamientos.leer_material_original_ajuste_reglas_v1('vec.bolsa.reglas.ajustes',gen_random_uuid());
  RAISE EXCEPTION 'B88 fallo: material original ajeno admitido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $no_lectura$;
RESET SESSION AUTHORIZATION;

SET SESSION AUTHORIZATION prueba_b88;
DO $casos$
DECLARE k1 uuid:=gen_random_uuid(); k2 uuid:=gen_random_uuid(); k3 uuid:=gen_random_uuid();
 fecha text:=to_char((clock_timestamp()+interval '1 day') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 antes text:=to_char((clock_timestamp()+interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
 c1 text:='{"b05.plazo_respuesta":{"cantidad":"2"}}';
 c2 text:='{"b05.plazo_respuesta":{"cantidad":"3"}}';
 d1 jsonb:='[{"regla_clave":"b05.plazo_respuesta","campo":"cantidad","anterior":"1","nuevo":"2"}]';
 d2 jsonb:='[{"regla_clave":"b05.plazo_respuesta","campo":"cantidad","anterior":"2","nuevo":"3"}]';
 r jsonb; r2 jsonb; n bigint;
BEGIN
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1('vec.bolsa.reglas.ajustes',now());
 IF n<>0 THEN RAISE EXCEPTION 'B88 fallo: clon ya tiene ajustes'; END IF;
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,d1,fecha)-'cambios','22023','cambios ausentes');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,d1,fecha)||'{"operacion":"consultar"}','22023','lectura V3 paralela');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,d1,fecha)||'{"vigente_desde":"mañana"}','22023','fecha inválida');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,d1,fecha)||jsonb_build_object('vigente_desde',
   to_char((clock_timestamp()-interval '1 hour') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')),
   '40001','fecha pasada');
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,d1,fecha)||'{"extra":1}','22023','clave extra');
 r:=pg_temp.operar(pg_temp.material(k1,0,c1,d1,fecha));
 IF (r->>'replay')::boolean OR r#>>'{recibo,version}'<>'1'
    OR r#>>'{recibo,vigente_desde}' IS NULL OR r#>>'{recibo,publicada_en}' IS NULL
 THEN RAISE EXCEPTION 'B88 fallo: alta programada %',r; END IF;
 SELECT count(*) INTO n FROM vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1('vec.bolsa.reglas.ajustes',now());
 IF n<>0 THEN RAISE EXCEPTION 'B88 fallo: versión futura aplicada antes'; END IF;
 SELECT version INTO n FROM vec_bolsa_llamamientos.leer_cabeza_ajustes_reglas_v1('vec.bolsa.reglas.ajustes');
 IF n<>1 THEN RAISE EXCEPTION 'B88 fallo: cabeza futura ausente'; END IF;
 SELECT version INTO n FROM vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1(
  'vec.bolsa.reglas.ajustes',(fecha::timestamptz)+interval '1 microsecond');
 IF n<>1 THEN RAISE EXCEPTION 'B88 fallo: versión futura ausente'; END IF;
 r2:=pg_temp.operar(pg_temp.material(k1,0,c1,d1,fecha));
 IF NOT (r2->>'replay')::boolean OR r2#>>'{recibo,recibo_ref}'<>r#>>'{recibo,recibo_ref}'
    OR r2#>>'{recibo,vigente_desde}'<>r#>>'{recibo,vigente_desde}'
 THEN RAISE EXCEPTION 'B88 fallo: replay %',r2; END IF;
 PERFORM pg_temp.falla(pg_temp.material(k1,0,c1,d1,antes),'23505','misma clave otra fecha');
 PERFORM pg_temp.falla(pg_temp.material(k2,0,c2,d2,fecha),'40001','CAS obsoleto');
 PERFORM pg_temp.falla(pg_temp.material(k2,1,c2,d2,antes),'40001','efecto decreciente');
 r2:=pg_temp.operar(pg_temp.material(k2,1,c2,d2,fecha));
 IF r2#>>'{recibo,version}'<>'2' THEN RAISE EXCEPTION 'B88 fallo: corrección igual fecha'; END IF;
 SELECT version INTO n FROM vec_bolsa_llamamientos.leer_ajustes_reglas_en_v1(
  'vec.bolsa.reglas.ajustes',(fecha::timestamptz)+interval '1 microsecond');
 IF n<>2 THEN RAISE EXCEPTION 'B88 fallo: desempate por versión'; END IF;
 PERFORM pg_temp.falla(pg_temp.material(k3,2,c2,d2,fecha),'22023','delta sin cambio');
END $casos$;
RESET SESSION AUTHORIZATION;

DO $inmutable$
BEGIN
 SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
 BEGIN
  UPDATE vec_bolsa_llamamientos.regla_ajuste_version_v1 SET motivo_clave='otro';
  RAISE EXCEPTION 'B88 fallo: UPDATE admitido';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 BEGIN
  TRUNCATE vec_bolsa_llamamientos.regla_ajuste_cambio_v1 CASCADE;
  RAISE EXCEPTION 'B88 fallo: TRUNCATE admitido';
 EXCEPTION WHEN SQLSTATE '55000' THEN NULL; END;
 RESET ROLE;
END $inmutable$;
SELECT 'B88-PRUEBAS-OK' AS resultado;
ROLLBACK;
