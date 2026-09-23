\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
DO $guarda$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.reposicion_orden_bolsa) THEN
  RAISE EXCEPTION '000018 conserva historia de reposiciones; DOWN denegado' USING ERRCODE='55000';
 END IF;
END $guarda$;
DO $restaura_b7$
DECLARE
 v_def text;
 v_validas_ant text := $sql$SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref;$sql$;
 v_orden_ant text := $sql$SELECT count(*) INTO v_ordenadas FROM (SELECT e.orden,lag(e.orden) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.constitucion c ON c.bolsa_ref=p_bolsa JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea AND e.participacion_ref=x.ref) q WHERE anterior IS NULL OR orden>anterior;$sql$;
 v_validas_nueva text := $sql$SELECT count(*) INTO v_validas FROM jsonb_array_elements_text(p_participaciones) x(ref) JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o ON o.participacion_ref=x.ref AND o.orden_vigente IS NOT NULL;$sql$;
 v_orden_nueva text := $sql$SELECT count(*) INTO v_ordenadas FROM (SELECT o.orden_vigente AS orden,lag(o.orden_vigente) OVER(ORDER BY x.n) anterior FROM jsonb_array_elements_text(p_participaciones) WITH ORDINALITY x(ref,n) JOIN vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(p_bolsa,p_emitido) o ON o.participacion_ref=x.ref AND o.orden_vigente IS NOT NULL) q WHERE anterior IS NULL OR orden>anterior;$sql$;
BEGIN
 SELECT pg_get_functiondef('vec_bolsa_llamamientos.reservar_llamamiento_v1(text,text,text,text,text,jsonb,jsonb,timestamptz,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) INTO STRICT v_def;
 IF strpos(v_def,v_validas_nueva)=0 OR strpos(v_def,v_orden_nueva)=0 THEN
  RAISE EXCEPTION 'reserva B7 no contiene ajuste B6' USING ERRCODE='55000';
 END IF;
 EXECUTE replace(replace(v_def,v_validas_nueva,v_validas_ant),v_orden_nueva,v_orden_ant);
END $restaura_b7$;
DROP FUNCTION vec_bolsa_llamamientos.leer_orden_vigente_bolsa_v1(text,timestamptz);
DROP TRIGGER situacion_participacion_registra_reposicion ON vec_bolsa_llamamientos.situacion_participacion;
DROP FUNCTION vec_bolsa_llamamientos.registrar_reposicion_orden_v1();
DROP TRIGGER bolsa_constituida_crea_politica_orden ON vec_bolsa_llamamientos.bolsa_constituida;
DROP FUNCTION vec_bolsa_llamamientos.crear_politica_orden_provisional_v1();
DROP TABLE vec_bolsa_llamamientos.reposicion_orden_bolsa;
DROP TABLE vec_bolsa_llamamientos.politica_orden_bolsa;
COMMIT;
