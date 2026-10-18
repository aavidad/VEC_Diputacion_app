\set ON_ERROR_STOP on
-- Tras B125: el LOGIN técnico, con referencias reales o ajenas, no recibe
-- metadatos ni acuse sin una capacidad documental. Pasar dos refs ficticias
-- privadas con -v b125_bolsa_ref=... -v b125_participacion_ref=...
BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
CREATE TEMP TABLE b125_fixture(bolsa_ref text,participacion_ref text) ON COMMIT DROP;
INSERT INTO b125_fixture VALUES(:'b125_bolsa_ref',:'b125_participacion_ref');
GRANT SELECT ON b125_fixture TO vec_bolsa_llamamientos_desarrollo;
SET SESSION AUTHORIZATION vec_bolsa_llamamientos_desarrollo;
DO $negativa$
DECLARE f record; bolsa text; participacion text; fallo text; funcion text;
BEGIN
 SELECT * INTO STRICT f FROM pg_temp.b125_fixture;
 FOREACH bolsa IN ARRAY ARRAY[f.bolsa_ref,'bolsa:ajena'] LOOP
  participacion:=CASE WHEN bolsa=f.bolsa_ref THEN f.participacion_ref ELSE 'participacion:ajena' END;
  FOREACH funcion IN ARRAY ARRAY['metadatos','auditado'] LOOP
   fallo:='sin_error';
   BEGIN
    IF funcion='metadatos' THEN
     PERFORM vec_bolsa_llamamientos.resolver_metadatos_justificante_rrhh_v1(
      bolsa,participacion);
    ELSE
     PERFORM vec_bolsa_llamamientos.resolver_ambito_justificante_rrhh_auditado_v1(
      bolsa,participacion,NULL,NULL,'{}'::jsonb);
    END IF;
   EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS fallo=RETURNED_SQLSTATE;
   END;
   IF fallo IS DISTINCT FROM '42501'
   THEN RAISE EXCEPTION 'B125 prueba: acceso técnico no cerrado % %',funcion,fallo; END IF;
  END LOOP;
 END LOOP;
END $negativa$;
ROLLBACK;
\echo B125_PRUEBA_SIN_METADATA_OK
