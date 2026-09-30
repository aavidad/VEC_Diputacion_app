\set ON_ERROR_STOP on
-- Comprobación focal de estructura/ACL y codec, no un E2E Personal/RPT.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE t text;col text;f record;
BEGIN
 FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','confirmacion_ginpix_v1','reincorporacion_titular_v1'] LOOP
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=('vec_contratacion_temporal.'||t)::regclass AND c.contype='f' AND c.confrelid='vec_contratacion_temporal.incorporacion_registro_v2'::regclass)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=('vec_contratacion_temporal.'||t)::regclass AND c.conname='origen_b2_ct155_fk' AND c.contype='f' AND c.confmatchtype='s' AND c.confrelid='vec_contratacion_temporal.origen_incorporacion_personal_b2_v1'::regclass AND c.convalidated)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=('vec_contratacion_temporal.'||t)::regclass AND c.conname='origen_xor_ct155' AND c.contype='c' AND c.convalidated AND strpos(pg_get_constraintdef(c.oid),'IS NOT NULL')>0)
 THEN RAISE EXCEPTION 'CT155: FK histórica, B2 o XOR ausente: %',t;END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY['plan_incorporacion_personal_b2_v1','origen_incorporacion_personal_b2_v1'] LOOP
 IF has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.'||t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=('vec_contratacion_temporal.'||t)::regclass AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'CT155: ACL o RLS incompleta: %',t;END IF;
 END LOOP;
 FOR f IN SELECT p.oid,p.proname,p.prosecdef,p.proconfig FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE '%ct155' OR p.proname IN ('registrar_plan_nominal_b2_v1','leer_plan_nominal_b2_v1','confirmar_origen_incorporacion_b2_v1','leer_origen_incorporacion_b2_v1','leer_antecedentes_plan_b2_v1')) LOOP
 IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f.oid AND x.grantee=0) THEN RAISE EXCEPTION 'CT155: EXECUTE PUBLIC: %',f.proname;END IF;
 IF f.proname LIKE '%ct155' AND has_function_privilege('vec_contratacion_temporal_ejecutor',f.oid,'EXECUTE') THEN RAISE EXCEPTION 'CT155: auxiliar alcanzable: %',f.proname;END IF;
 END LOOP;
 IF vec_contratacion_temporal.canon_plan_personal_ct155('{"z":9007199254740991,"a":{"z":"<&>","a":"fecha"}}'::jsonb) IS DISTINCT FROM '{"a":{"a":"fecha","z":"\u003c\u0026\u003e"},"z":9007199254740991}' THEN RAISE EXCEPTION 'CT155: codec Go/SQL divergente';END IF;
 BEGIN PERFORM vec_contratacion_temporal.canon_plan_personal_ct155('{"z":9007199254740992}'::jsonb);RAISE EXCEPTION 'CT155: entero fuera de rango admitido';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
END $prueba$;
ROLLBACK;
