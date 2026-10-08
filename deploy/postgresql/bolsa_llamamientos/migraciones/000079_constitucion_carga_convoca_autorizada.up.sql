\set ON_ERROR_STOP on
-- B79: AD218 consume la autorización en el mismo TopXID SERIALIZABLE que
-- acta/staging/original cifrado BIC7, constitución B7, vínculos B8 y recibo.
-- Un reintento exige material V3 nuevo; recupera el recibo de negocio original
-- y expone por separado la auditoría de acceso actual.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_bolsa_llamamientos:migracion:000079',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_bolsa_importacion_convoca.guardar_lote_v1(jsonb,jsonb)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_bolsa_importacion_convoca.guardar_original_v1(jsonb,jsonb)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',
   'vec_bolsa_importacion_convoca.consultar_estado_v1(text,text)','EXECUTE')
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constituir_bolsa_v1(text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz)') IS NULL
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(text,jsonb,timestamptz)') IS NULL
 OR pg_catalog.to_regprocedure('vec_bolsa_llamamientos.confirmar_carga_convoca_v1(jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regclass('vec_bolsa_llamamientos.recibo_carga_convoca') IS NOT NULL
 THEN RAISE EXCEPTION 'B79: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE TABLE vec_bolsa_llamamientos.recibo_carga_convoca (
 acta_ref text PRIMARY KEY REFERENCES vec_bolsa_llamamientos.constitucion(acta_ref),
 actor_ref text NOT NULL,
 categoria_ref text NOT NULL,
 bolsa_ref text NOT NULL,
 huella_fichero_sha256 text NOT NULL CHECK (huella_fichero_sha256 ~ '^[0-9a-f]{64}$'),
 huella_entradas_sha256 text NOT NULL CHECK (huella_entradas_sha256 ~ '^[0-9a-f]{64}$'),
 huella_vinculos_sha256 text NOT NULL CHECK (huella_vinculos_sha256 ~ '^[0-9a-f]{64}$'),
 recibo_constitucion jsonb NOT NULL CHECK (pg_catalog.jsonb_typeof(recibo_constitucion)='object'),
 recibo_vinculos jsonb NOT NULL CHECK (pg_catalog.jsonb_typeof(recibo_vinculos)='object'),
 decision_ref text NOT NULL UNIQUE,
 auditoria_ref text NOT NULL UNIQUE CHECK (auditoria_ref ~ '^aud_v3_[0-9a-f]{32}$'),
 consumida_en timestamptz(6) NOT NULL,
 registrada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
REVOKE ALL ON vec_bolsa_llamamientos.recibo_carga_convoca FROM PUBLIC;
CREATE FUNCTION vec_bolsa_llamamientos.negar_mutacion_recibo_carga_v1()
RETURNS trigger LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
BEGIN
 RAISE EXCEPTION 'B79: recibo inmutable' USING ERRCODE='55000';
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.negar_mutacion_recibo_carga_v1() FROM PUBLIC;
CREATE TRIGGER negar_cambio_recibo_carga BEFORE UPDATE OR DELETE OR TRUNCATE
 ON vec_bolsa_llamamientos.recibo_carga_convoca
 FOR EACH STATEMENT EXECUTE FUNCTION vec_bolsa_llamamientos.negar_mutacion_recibo_carga_v1();

CREATE FUNCTION vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
 p_acta jsonb,p_filas_cifradas jsonb,p_original_cifrado jsonb,
 p_acta_ref text,p_actor_ref text,p_categoria_ref text,p_bolsa_ref text,p_version_bolsa bigint,
 p_bolsa_canonica bytea,p_vigente_desde timestamptz,p_instantanea_ref text,p_version_instantanea bigint,
 p_instantanea_canonica bytea,p_referida_en timestamptz,p_generada_en timestamptz,p_entradas jsonb,
 p_confirmada_en timestamptz,p_vinculos jsonb,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,
 p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
 SET search_path=pg_catalog,pg_temp SET timezone='UTC' SET lock_timeout='2s' SET statement_timeout='60s' AS $f$
DECLARE decision jsonb; bolsa jsonb; consumo record; estado jsonb; acta_durable jsonb;
        carga record; recibo jsonb; vinculos_recibo jsonb; previo vec_bolsa_llamamientos.recibo_carga_convoca%ROWTYPE;
        huella_entradas text; huella_vinculos text; nueva boolean;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
 OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
 OR pg_catalog.current_setting('transaction_read_only')<>'off'
 OR pg_catalog.current_setting('TimeZone')<>'UTC'
 OR pg_catalog.jsonb_typeof(p_acta) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_filas_cifradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(p_original_cifrado) IS DISTINCT FROM 'object'
 OR pg_catalog.jsonb_typeof(p_entradas) IS DISTINCT FROM 'array'
 OR pg_catalog.jsonb_typeof(p_vinculos) IS DISTINCT FROM 'array'
 OR p_acta_ref IS NULL OR p_acta_ref !~ '^acta:importacion-convoca:[0-9a-f]{64}$'
 OR p_actor_ref IS NULL OR pg_catalog.octet_length(p_actor_ref) NOT BETWEEN 1 AND 256
 OR p_actor_ref<>pg_catalog.btrim(p_actor_ref)
 OR p_acta->>'acta_ref' IS DISTINCT FROM p_acta_ref
 OR p_acta->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR p_acta->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR p_acta->>'actor_ref' IS DISTINCT FROM 'actor:rrhh:'||pg_catalog.substr(
  pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to('vec.bolsa.carga_convoca.actor'||pg_catalog.chr(31)||p_actor_ref,'UTF8')),'hex'),1,32)
 OR p_acta->>'huella_fichero_sha256' IS NULL
 OR p_acta->>'huella_fichero_sha256' !~ '^[0-9a-f]{64}$'
 OR p_acta_ref IS DISTINCT FROM 'acta:importacion-convoca:'||pg_catalog.encode(pg_catalog.sha256(
  pg_catalog.convert_to((p_acta->>'huella_fichero_sha256')||pg_catalog.chr(31)||p_categoria_ref,'UTF8')),'hex')
 THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 BEGIN
  decision:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
  bolsa:=pg_catalog.convert_from(p_bolsa_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B79: material inválido' USING ERRCODE='42501'; END;
 IF pg_catalog.jsonb_typeof(decision) IS DISTINCT FROM 'object'
 OR decision->>'principal_id' IS DISTINCT FROM p_actor_ref
 OR decision->>'recurso_ref' IS DISTINCT FROM p_acta_ref
 OR pg_catalog.jsonb_typeof(bolsa) IS DISTINCT FROM 'object'
 OR bolsa->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR bolsa->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR bolsa->>'huella_listado_sha256' IS DISTINCT FROM p_acta->>'huella_fichero_sha256'
 THEN RAISE EXCEPTION 'B79: carga de bolsa no autorizada' USING ERRCODE='42501'; END IF;
 IF pg_catalog.jsonb_array_length(p_entradas)<1
 OR pg_catalog.jsonb_array_length(p_vinculos)>pg_catalog.jsonb_array_length(p_entradas)
 OR pg_catalog.jsonb_array_length(p_entradas)<>pg_catalog.jsonb_array_length(p_filas_cifradas)
 OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_entradas) e
     WHERE pg_catalog.jsonb_typeof(e.value) IS DISTINCT FROM 'object'
        OR e.value->>'participacion_ref' IS NULL OR e.value->>'fila_numero' IS NULL)
 OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(p_vinculos) v
     WHERE pg_catalog.jsonb_typeof(v.value) IS DISTINCT FROM 'object'
        OR v.value->>'participacion_ref' IS NULL OR v.value->>'candidato_ref' IS NULL)
 OR (SELECT pg_catalog.count(DISTINCT e.value->>'participacion_ref')
      FROM pg_catalog.jsonb_array_elements(p_entradas) e)<>pg_catalog.jsonb_array_length(p_entradas)
 OR (SELECT pg_catalog.count(DISTINCT e.value->>'fila_numero')
      FROM pg_catalog.jsonb_array_elements(p_entradas) e)<>pg_catalog.jsonb_array_length(p_entradas)
 OR (SELECT pg_catalog.count(DISTINCT v.value->>'participacion_ref')
      FROM pg_catalog.jsonb_array_elements(p_vinculos) v)<>pg_catalog.jsonb_array_length(p_vinculos)
 OR EXISTS (
   SELECT v.value->>'participacion_ref' FROM pg_catalog.jsonb_array_elements(p_vinculos) v
   EXCEPT SELECT e.value->>'participacion_ref' FROM pg_catalog.jsonb_array_elements(p_entradas) e)
 OR EXISTS (
   (SELECT e.value->>'fila_numero' FROM pg_catalog.jsonb_array_elements(p_entradas) e
    EXCEPT SELECT f.value->>'numero' FROM pg_catalog.jsonb_array_elements(p_filas_cifradas) f)
   UNION ALL
   (SELECT f.value->>'numero' FROM pg_catalog.jsonb_array_elements(p_filas_cifradas) f
    EXCEPT SELECT e.value->>'fila_numero' FROM pg_catalog.jsonb_array_elements(p_entradas) e))
 THEN RAISE EXCEPTION 'B79: filas y vínculos incompatibles' USING ERRCODE='22023'; END IF;
 huella_entradas:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_entradas::text,'UTF8')),'hex');
 huella_vinculos:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p_vinculos::text,'UTF8')),'hex');
 -- La fachada AD218 exige consumo_nuevo=true; aquí no existe acceso por mera
 -- posesión de una referencia histórica. El reintento recibe otra decisión.
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_carga_convoca_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM p_acta_ref
 OR consumo.decision_ref IS NULL OR consumo.auditoria_ref IS NULL OR consumo.consumida_en IS NULL
 THEN RAISE EXCEPTION 'B79: consumo divergente' USING ERRCODE='42501'; END IF;
 -- Los locks internos de guardar_lote y B7 siguen el mismo orden en todos los
 -- intentos. En el replay se pasa el acta histórica para no cambiar custodia.
 estado:=vec_bolsa_importacion_convoca.consultar_estado_v1(
  p_acta->>'huella_fichero_sha256',p_categoria_ref);
 acta_durable:=COALESCE(estado->'acta',p_acta);
 IF acta_durable->>'acta_ref' IS DISTINCT FROM p_acta_ref
 OR acta_durable->>'categoria_ref' IS DISTINCT FROM p_categoria_ref
 OR acta_durable->>'bolsa_ref' IS DISTINCT FROM p_bolsa_ref
 OR acta_durable->>'huella_fichero_sha256' IS DISTINCT FROM p_acta->>'huella_fichero_sha256'
 THEN RAISE EXCEPTION 'B79: acta histórica incompatible' USING ERRCODE='23505'; END IF;
 SELECT * INTO STRICT carga FROM vec_bolsa_importacion_convoca.guardar_lote_v1(acta_durable,p_filas_cifradas);
 IF carga.acta_canonica IS DISTINCT FROM acta_durable THEN
  RAISE EXCEPTION 'B79: acta durable divergente' USING ERRCODE='55000'; END IF;
 PERFORM vec_bolsa_importacion_convoca.guardar_original_v1(carga.acta_canonica,p_original_cifrado);
 SELECT * INTO previo FROM vec_bolsa_llamamientos.recibo_carga_convoca WHERE acta_ref=p_acta_ref FOR SHARE;
 IF FOUND THEN
  IF previo.categoria_ref<>p_categoria_ref OR previo.bolsa_ref<>p_bolsa_ref
  OR previo.huella_fichero_sha256<>p_acta->>'huella_fichero_sha256'
  OR previo.recibo_constitucion->>'version_bolsa' IS DISTINCT FROM p_version_bolsa::text
  OR previo.recibo_constitucion->>'instantanea_ref' IS DISTINCT FROM p_instantanea_ref
  OR previo.recibo_constitucion->>'version_instantanea' IS DISTINCT FROM p_version_instantanea::text
  OR previo.huella_entradas_sha256<>huella_entradas OR previo.huella_vinculos_sha256<>huella_vinculos
  THEN RAISE EXCEPTION 'B79: reintento incompatible' USING ERRCODE='23505'; END IF;
  RETURN previo.recibo_constitucion||pg_catalog.jsonb_build_object(
   'decision_ref',previo.decision_ref,'auditoria_ref',previo.auditoria_ref,
   'consumida_en',pg_catalog.to_char(previo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
   'auditoria_acceso_ref',consumo.auditoria_ref,
   'decision_acceso_ref',consumo.decision_ref,
   'acta_reutilizada',true,'vinculos',previo.recibo_vinculos);
 END IF;
 recibo:=vec_bolsa_llamamientos.constituir_bolsa_v1(
  p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_version_bolsa,p_bolsa_canonica,p_vigente_desde,
  p_instantanea_ref,p_version_instantanea,p_instantanea_canonica,p_referida_en,p_generada_en,p_entradas,p_confirmada_en);
 IF pg_catalog.jsonb_typeof(recibo) IS DISTINCT FROM 'object'
 OR recibo->>'acta_ref' IS DISTINCT FROM p_acta_ref OR recibo->>'reutilizada' IS DISTINCT FROM 'false'
 THEN RAISE EXCEPTION 'B79: constitución previa sin recibo B1' USING ERRCODE='23505'; END IF;
 IF pg_catalog.jsonb_array_length(p_vinculos)>0 THEN
  vinculos_recibo:=vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(p_acta_ref,p_vinculos,p_confirmada_en);
 ELSE
  vinculos_recibo:=pg_catalog.jsonb_build_object('nuevos',0,'existentes',0);
 END IF;
 IF pg_catalog.jsonb_typeof(vinculos_recibo) IS DISTINCT FROM 'object'
 OR (vinculos_recibo->>'nuevos')::integer IS DISTINCT FROM pg_catalog.jsonb_array_length(p_vinculos)
 OR (vinculos_recibo->>'existentes')::integer IS DISTINCT FROM 0
 THEN RAISE EXCEPTION 'B79: vínculos divergentes' USING ERRCODE='55000'; END IF;
 INSERT INTO vec_bolsa_llamamientos.recibo_carga_convoca(
  acta_ref,actor_ref,categoria_ref,bolsa_ref,huella_fichero_sha256,
  huella_entradas_sha256,huella_vinculos_sha256,recibo_constitucion,recibo_vinculos,
  decision_ref,auditoria_ref,consumida_en)
 VALUES(p_acta_ref,p_actor_ref,p_categoria_ref,p_bolsa_ref,p_acta->>'huella_fichero_sha256',
  huella_entradas,huella_vinculos,recibo,vinculos_recibo,
  consumo.decision_ref,consumo.auditoria_ref,consumo.consumida_en);
 RETURN recibo||pg_catalog.jsonb_build_object(
  'decision_ref',consumo.decision_ref,'auditoria_ref',consumo.auditoria_ref,
  'consumida_en',pg_catalog.to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'),
  'auditoria_acceso_ref',consumo.auditoria_ref,'decision_acceso_ref',consumo.decision_ref,
  'acta_reutilizada',carga.reutilizada,'vinculos',vinculos_recibo);
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
 jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.confirmar_carga_convoca_v1(
 jsonb,jsonb,jsonb,text,text,text,text,bigint,bytea,timestamptz,text,bigint,bytea,timestamptz,timestamptz,jsonb,timestamptz,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
