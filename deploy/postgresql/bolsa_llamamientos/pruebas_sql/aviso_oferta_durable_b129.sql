-- B129: ejecutar solo en una copia desechable de la principal, tras la migración.
-- La oferta de este ensayo es un dato ficticio; se revierte al terminar.
\set ON_ERROR_STOP on
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='15s';
SET LOCAL idle_in_transaction_session_timeout='20s';

DO $pre$
BEGIN
 IF to_regprocedure('vec_bolsa_llamamientos.vincular_despacho_aviso_v1(text,text,text,bytea,text)') IS NULL
   OR to_regclass('vec_bolsa_llamamientos.aviso_oferta_destinatario') IS NULL
 THEN RAISE EXCEPTION 'B129 no instalada en la copia'; END IF;
END $pre$;

-- Solo se desactivan las dos comprobaciones de política que dependen de una
-- publicación V3 completa; el disparador B129 y su cierre diferido siguen activos.
ALTER TABLE vec_bolsa_llamamientos.oferta_publicada DISABLE TRIGGER oferta_politica_b47;
ALTER TABLE vec_bolsa_llamamientos.oferta_publicada DISABLE TRIGGER verificar_aceptacion_previa_oferta_b75;
CREATE SCHEMA prueba_aviso_b129;
CREATE FUNCTION prueba_aviso_b129.crear(p_oferta text,p_clave text) RETURNS integer
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
DECLARE v_bolsa text;v_total integer;v_publicada timestamptz:=clock_timestamp();
BEGIN
 SELECT s.bolsa_ref INTO v_bolsa FROM vec_bolsa_llamamientos.situaciones_en_v1(v_publicada) s
 JOIN vec_bolsa_llamamientos.bolsa_constituida b ON b.bolsa_ref=s.bolsa_ref
  AND b.estado='vigente' AND b.vigente_desde<=v_publicada
  AND (b.vigente_hasta IS NULL OR v_publicada<b.vigente_hasta)
 WHERE s.situacion='disponible' GROUP BY s.bolsa_ref ORDER BY count(*) DESC LIMIT 1;
 IF v_bolsa IS NULL THEN RAISE EXCEPTION 'sin bolsa constituida con disponibles'; END IF;
 SELECT count(*) INTO v_total FROM vec_bolsa_llamamientos.situaciones_en_v1(v_publicada) s
 WHERE s.bolsa_ref=v_bolsa AND s.situacion='disponible';
 INSERT INTO vec_bolsa_llamamientos.oferta_publicada
   (oferta_ref,recibo_ref,bolsa_ref,actor_ref,clave_idempotencia,datos,plazo,
    publicada_en,vence_antes_de,huella_comando_sha256,decision_ref)
 VALUES(p_oferta,'recibo:'||p_oferta,v_bolsa,'per_actoractoractoractoractor',p_clave,
   '{}'::jsonb,'{}'::jsonb,v_publicada,v_publicada+interval '1 day',repeat('a',64),
   'decision:prueba-aviso:'||encode(sha256(convert_to(p_clave,'UTF8')),'hex'));
 RETURN v_total;
END $f$;
ALTER FUNCTION prueba_aviso_b129.crear(text,text) OWNER TO vec_bolsa_llamamientos_propietario;
CREATE FUNCTION prueba_aviso_b129.contar(p_oferta text) RETURNS integer
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT count(*)::integer FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
 WHERE oferta_ref=p_oferta
$f$;
ALTER FUNCTION prueba_aviso_b129.contar(text) OWNER TO vec_bolsa_llamamientos_propietario;
CREATE FUNCTION prueba_aviso_b129.estado(p_oferta text,p_participacion text) RETURNS text
LANGUAGE sql STABLE SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $f$
 SELECT estado FROM vec_bolsa_llamamientos.aviso_oferta_destinatario
 WHERE oferta_ref=p_oferta AND participacion_ref=p_participacion
$f$;
ALTER FUNCTION prueba_aviso_b129.estado(text,text) OWNER TO vec_bolsa_llamamientos_propietario;
CREATE FUNCTION prueba_aviso_b129.espera(p_sql text,p_codigo text) RETURNS void
LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE p_sql;
 RAISE EXCEPTION 'la operación prohibida tuvo éxito';
EXCEPTION WHEN OTHERS THEN
 IF SQLSTATE<>p_codigo THEN RAISE; END IF;
END $f$;
GRANT USAGE ON SCHEMA prueba_aviso_b129 TO vec_bolsa_llamamientos_ejecutor;
GRANT EXECUTE ON FUNCTION prueba_aviso_b129.crear(text,text),
 prueba_aviso_b129.contar(text),prueba_aviso_b129.estado(text,text),
 prueba_aviso_b129.espera(text,text)
 TO vec_bolsa_llamamientos_ejecutor;

SET SESSION AUTHORIZATION vec_bolsa_llamamientos_ejecutor;
SELECT 'oferta:'||encode(sha256(convert_to('b129-aviso-focal','UTF8')),'hex') AS oferta_ref,
       'b129-aviso-focal' AS clave \gset
SELECT prueba_aviso_b129.crear(:'oferta_ref',:'clave') AS disponibles_antes \gset
SELECT CASE WHEN :disponibles_antes::integer>1 THEN true ELSE 1/0=0 END;

-- La autoridad de despacho solo se vincula al efecto creado en esta transacción.
SELECT prueba_aviso_b129.espera(format(
 'SELECT vec_bolsa_llamamientos.vincular_despacho_aviso_v1(%L,''per_actoractoractoractoractor'',''clave-ajena'',sha256(decode(repeat(''11'',32),''hex'')),''b129-v1'')',
 :'oferta_ref'),'42501');
SELECT vec_bolsa_llamamientos.vincular_despacho_aviso_v1(
 :'oferta_ref','per_actoractoractoractoractor',:'clave',
 sha256(decode(repeat('11',32),'hex')),'b129-v1')->>'aviso_version_token' AS version \gset
SELECT CASE WHEN :'version'='b129-v1' THEN true ELSE 1/0=0 END;
SET CONSTRAINTS exigir_token_aviso_oferta_b129 IMMEDIATE;

-- Ningún tercero con el mismo rol puede reclamar solo con la referencia.
SELECT prueba_aviso_b129.espera(format(
 'SELECT * FROM vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(%L,decode(repeat(''22'',32),''hex''),1)',
 :'oferta_ref'),'42501');
SELECT * FROM vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(
 :'oferta_ref',decode(repeat('11',32),'hex'),1) \gset primero_
SELECT CASE WHEN :'primero_participacion_ref'<>'' AND :'primero_message_id' LIKE '<%@vec.dipgra.local>'
  AND :primero_intento::integer=1 THEN true ELSE 1/0=0 END;

-- Reclamar cambia a incierto antes de tocar el relay.
SELECT CASE WHEN prueba_aviso_b129.estado(:'oferta_ref',:'primero_participacion_ref')
 ='enviado_sin_confirmacion' THEN true ELSE 1/0=0 END;
SELECT (vec_bolsa_llamamientos.finalizar_aviso_oferta_v1(
 :'oferta_ref',:'primero_participacion_ref',decode(repeat('11',32),'hex'),:primero_intento,'transitorio')->'aviso'->>'pendientes')::integer>0;
SELECT * FROM vec_bolsa_llamamientos.reclamar_aviso_oferta_v1(
 :'oferta_ref',decode(repeat('11',32),'hex'),1) \gset segundo_
-- Un transitorio no monopoliza las recuperaciones mientras queden personas sin intento.
SELECT CASE WHEN :'segundo_participacion_ref'<>:'primero_participacion_ref'
 AND :segundo_intento::integer=1 THEN true ELSE 1/0=0 END;
SELECT vec_bolsa_llamamientos.finalizar_aviso_oferta_v1(
 :'oferta_ref',:'segundo_participacion_ref',decode(repeat('11',32),'hex'),:segundo_intento,'aceptado')->'aviso'->>'enviados';
SELECT CASE WHEN prueba_aviso_b129.contar(:'oferta_ref')=:disponibles_antes::integer
 THEN true ELSE 1/0=0 END;
ROLLBACK;
