\set ON_ERROR_STOP on
-- Vincula sólo una participación de una Bolsa ya constituida por su autoridad.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT set_config('vec.fixture.apertura', :'apertura', true);
SELECT set_config('vec.fixture.llamamiento', :'llamamiento', true);
SELECT set_config('vec.fixture.candidato', :'candidato', true);
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
DO $anclaje$
DECLARE
 apertura jsonb;
 acta text;
 participacion text;
 candidato text:=current_setting('vec.fixture.candidato');
 coincidencias integer;
BEGIN
 SELECT convert_from(registro_canonico,'UTF8')::jsonb INTO STRICT apertura
 FROM vec_bolsa_llamamientos.integracion_desarrollo
 WHERE operacion_ref=current_setting('vec.fixture.apertura') AND tipo='propuesta';
 IF apertura->'llamamiento'->>'LlamamientoRef' IS DISTINCT FROM current_setting('vec.fixture.llamamiento') THEN
   RAISE EXCEPTION 'FALTA: apertura original del llamamiento esperado';
 END IF;
 participacion:=apertura->'propuesta'->>'participacion_seleccionada_ref';
 SELECT count(*),min(c.acta_ref) INTO coincidencias,acta
 FROM vec_bolsa_llamamientos.constitucion c
 JOIN vec_bolsa_llamamientos.bolsa_constituida b
   ON b.bolsa_ref=c.bolsa_ref AND b.version=c.version_bolsa
  AND b.huella_bolsa_sha256=c.huella_bolsa_sha256
 JOIN vec_bolsa_llamamientos.constitucion_entrada e
   ON e.instantanea_ref=c.instantanea_ref AND e.version_instantanea=c.version_instantanea
 WHERE c.bolsa_ref=apertura->'instantanea'->>'bolsa_ref'
   AND c.version_bolsa=(apertura->'instantanea'->>'version_bolsa')::bigint
   AND c.huella_bolsa_sha256=apertura->'instantanea'->>'huella_bolsa_sha256'
   AND e.participacion_ref=participacion;
 IF coincidencias<>1 OR NOT EXISTS (
   SELECT 1 FROM vec_bolsa_llamamientos.situacion_participacion
   WHERE participacion_ref=participacion) THEN
   RAISE EXCEPTION 'FALTA: participación seleccionada en una única Bolsa constituida y con situación';
 END IF;
 PERFORM vec_bolsa_llamamientos.registrar_vinculos_candidato_v1(
   acta,jsonb_build_array(jsonb_build_object('candidato_ref',candidato,'participacion_ref',participacion)),clock_timestamp());
 IF (SELECT count(*) FROM vec_bolsa_llamamientos.vinculo_candidato v
     WHERE v.participacion_ref=participacion AND v.candidato_ref=candidato AND v.acta_ref=acta)<>1 THEN
   RAISE EXCEPTION 'FALTA: vínculo candidato/participación/acta no confirmado';
 END IF;
END $anclaje$;
COMMIT;
