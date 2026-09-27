\set ON_ERROR_STOP on
-- B49: feed privado de ceses B45 confirmados hacia la instantánea pública B10.
-- El publicador calcula y confirma el manifiesto V3 en su propia base antes de
-- acusar aquí el evento. La pista bolsa_ref no limita la reconstrucción: un
-- candidato puede participar en varias bolsas.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000049',0));

DO $rol$
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper) THEN
  RAISE EXCEPTION 'Bolsa 000049: instalación DBA requerida' USING ERRCODE='55000';
 END IF;
 IF to_regclass('vec_bolsa_llamamientos.publicacion_cese_b10') IS NOT NULL
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_bolsa_llamamientos_publicador_cese') THEN
  RAISE EXCEPTION 'Bolsa 000049 ya instalada' USING ERRCODE='55000';
 END IF;
 CREATE ROLE vec_bolsa_llamamientos_publicador_cese NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_publicador_cese',current_database());
END $rol$;

SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $pre$
BEGIN
 IF current_user <> 'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.restriccion_cese_bolsa') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()') IS NULL THEN
  RAISE EXCEPTION 'Bolsa 000049: dependencias incompatibles' USING ERRCODE='55000';
 END IF;
END $pre$;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_publicador_cese;

-- Una fila por fase publicada. El vencimiento local de disponible_desde
-- necesita otra instantánea aunque no haya un nuevo cese. Se conserva la
-- posición y la huella pública, sin candidato, relación, DNI ni situación.
CREATE TABLE vec_bolsa_llamamientos.publicacion_cese_b10 (
 evento_ref text NOT NULL REFERENCES vec_bolsa_llamamientos.restriccion_cese_bolsa(evento_ref),
 fase text NOT NULL CHECK (fase IN ('cese','vencimiento')),
 origen_ref text NOT NULL CHECK (octet_length(origen_ref) BETWEEN 1 AND 512),
 origen_posicion bigint NOT NULL CHECK (origen_posicion >= 0),
 ancla_manifiesto_sha256 text NOT NULL CHECK (
   ancla_manifiesto_sha256 ~ '^[a-f0-9]{64}$' AND ancla_manifiesto_sha256 <> repeat('0',64)),
 confirmada_en timestamptz(6) NOT NULL,
 confirmada_por text NOT NULL DEFAULT session_user CHECK (octet_length(confirmada_por) BETWEEN 1 AND 128),
 PRIMARY KEY (evento_ref,fase),
 UNIQUE (origen_ref,fase)
);
CREATE INDEX restriccion_cese_b10_feed
 ON vec_bolsa_llamamientos.restriccion_cese_bolsa(origen_posicion,origen_ref);
ALTER TABLE vec_bolsa_llamamientos.publicacion_cese_b10 ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_bolsa_llamamientos.publicacion_cese_b10 FORCE ROW LEVEL SECURITY;
CREATE POLICY publicacion_cese_b10_solo_propietario
 ON vec_bolsa_llamamientos.publicacion_cese_b10
 TO vec_bolsa_llamamientos_propietario
 USING (current_user='vec_bolsa_llamamientos_propietario')
 WITH CHECK (current_user='vec_bolsa_llamamientos_propietario');
REVOKE ALL ON TABLE vec_bolsa_llamamientos.publicacion_cese_b10 FROM PUBLIC;
CREATE TRIGGER publicacion_cese_b10_inmutable
 BEFORE UPDATE OR DELETE ON vec_bolsa_llamamientos.publicacion_cese_b10
 FOR EACH ROW EXECUTE FUNCTION vec_bolsa_llamamientos.constitucion_rechazar_mutacion();

-- El feed no revela datos de persona. Cada llamada devuelve la primera fase
-- pendiente, incluso si se insertó después un cese de posición menor. La
-- fase de vencimiento se hace exigible en el día local Europe/Madrid exacto;
-- no atribuye por sí sola disponibilidad ni cambia el estado de Bolsa.
CREATE FUNCTION vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1()
RETURNS TABLE(origen_posicion bigint,origen_ref text,evento_ref text,bolsa_ref text,fase text)
LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE v record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_publicador_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'feed B10 de ceses denegado' USING ERRCODE='42501';
 END IF;
 SELECT x.origen_posicion,x.origen_ref,x.evento_ref,l.bolsa_ref,x.fase INTO v
 FROM (
  SELECT r.origen_posicion,r.origen_ref,r.evento_ref,r.llamamiento_ref,'cese'::text AS fase
  FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  LEFT JOIN vec_bolsa_llamamientos.publicacion_cese_b10 p
    ON p.evento_ref=r.evento_ref AND p.fase='cese'
  WHERE p.evento_ref IS NULL
  UNION ALL
  SELECT r.origen_posicion,r.origen_ref,r.evento_ref,r.llamamiento_ref,'vencimiento'::text
  FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  JOIN vec_bolsa_llamamientos.publicacion_cese_b10 c
    ON c.evento_ref=r.evento_ref AND c.fase='cese'
  LEFT JOIN vec_bolsa_llamamientos.publicacion_cese_b10 pv
    ON pv.evento_ref=r.evento_ref AND pv.fase='vencimiento'
  WHERE pv.evento_ref IS NULL
    AND r.disponible_desde <= (transaction_timestamp() AT TIME ZONE 'Europe/Madrid')::date
 ) x
 LEFT JOIN vec_bolsa_llamamientos.llamamiento_integracion_desarrollo l
   ON l.llamamiento_ref=x.llamamiento_ref
 ORDER BY x.origen_posicion,x.origen_ref,x.fase LIMIT 1;
 IF NOT FOUND THEN RETURN; END IF;
 IF v.bolsa_ref IS NULL THEN
  RAISE EXCEPTION 'feed B10 de ceses sin bolsa de origen' USING ERRCODE='23503';
 END IF;
 RETURN QUERY SELECT v.origen_posicion,v.origen_ref,v.evento_ref,v.bolsa_ref,v.fase;
END $f$;

-- Confirmar solo la primera fase pendiente evita saltos. El replay exacto
-- conserva ancla y fecha; una ancla distinta es divergencia. El vencimiento
-- siempre exige nueva publicación/ACK: el snapshot de cese pudo prepararse
-- antes de medianoche aunque su ACK ocurriera después. El publicador invoca
-- esto DESPUÉS del COMMIT de V3 en su propia base.
CREATE FUNCTION vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(
 p_origen_posicion bigint,p_origen_ref text,p_fase text,p_ancla_manifiesto_sha256 text)
RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE v_origen vec_bolsa_llamamientos.restriccion_cese_bolsa;
 v_previa vec_bolsa_llamamientos.publicacion_cese_b10; v_siguiente record;
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_publicador_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_relevo_cese','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER') THEN
  RAISE EXCEPTION 'confirmación B10 de cese denegada' USING ERRCODE='42501';
 END IF;
 IF p_origen_posicion IS NULL OR p_origen_posicion<0
    OR p_origen_ref IS NULL OR octet_length(p_origen_ref) NOT BETWEEN 1 AND 512
    OR p_origen_ref<>btrim(p_origen_ref)
    OR p_fase IS NULL OR p_fase NOT IN ('cese','vencimiento')
    OR p_ancla_manifiesto_sha256 IS NULL
    OR p_ancla_manifiesto_sha256 !~ '^[a-f0-9]{64}$'
    OR p_ancla_manifiesto_sha256=repeat('0',64) THEN
  RAISE EXCEPTION 'confirmación B10 de cese inválida' USING ERRCODE='22023';
 END IF;
 PERFORM pg_advisory_xact_lock(hashtextextended('bolsa:publicacion-cese-b10',0));
 SELECT * INTO v_origen FROM vec_bolsa_llamamientos.restriccion_cese_bolsa r
  WHERE r.origen_ref=p_origen_ref;
 IF NOT FOUND OR v_origen.origen_posicion<>p_origen_posicion THEN
  RAISE EXCEPTION 'cese B10 no confirmado en Bolsa' USING ERRCODE='23503';
 END IF;
 SELECT * INTO v_previa FROM vec_bolsa_llamamientos.publicacion_cese_b10 p
  WHERE p.evento_ref=v_origen.evento_ref AND p.fase=p_fase;
 IF FOUND THEN
  IF v_previa.origen_ref<>p_origen_ref OR v_previa.origen_posicion<>p_origen_posicion
     OR v_previa.ancla_manifiesto_sha256<>p_ancla_manifiesto_sha256 THEN
   RAISE EXCEPTION 'publicación B10 de cese divergente' USING ERRCODE='VBP01';
  END IF;
  RETURN true;
 END IF;
 SELECT * INTO v_siguiente FROM vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1();
 IF NOT FOUND OR v_siguiente.origen_posicion<>p_origen_posicion
    OR v_siguiente.origen_ref<>p_origen_ref OR v_siguiente.fase<>p_fase THEN
  RAISE EXCEPTION 'publicación B10 de cese fuera de orden' USING ERRCODE='55000';
 END IF;
 INSERT INTO vec_bolsa_llamamientos.publicacion_cese_b10(
  evento_ref,fase,origen_ref,origen_posicion,ancla_manifiesto_sha256,confirmada_en)
 VALUES(v_origen.evento_ref,p_fase,p_origen_ref,p_origen_posicion,p_ancla_manifiesto_sha256,
        date_trunc('microseconds',clock_timestamp()));
 RETURN false;
END $f$;

REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1() FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(bigint,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.siguiente_publicacion_cese_b10_v1() TO vec_bolsa_llamamientos_publicador_cese;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.confirmar_publicacion_cese_b10_v1(bigint,text,text,text) TO vec_bolsa_llamamientos_publicador_cese;
COMMIT;
