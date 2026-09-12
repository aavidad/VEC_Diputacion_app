BEGIN;
SET LOCAL search_path = pg_catalog;
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal:migracion:000001:canon:v1', 0));
-- Las referencias PL/pgSQL por nombre no siempre crean dependencia pg_depend.
-- La retirada no puede dejar el registrador o su historia sin codec verificable.
DO $dependencias$
BEGIN
 IF EXISTS (
   SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_personal' AND p.proname='registrar_alta_ejercicio_v1'
 ) OR EXISTS (
   SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname='vec_personal' AND c.relkind IN ('r','p','v','m','f','S')
 ) THEN
   RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='retirada codec Personal rechazada: negocio dependiente';
 END IF;
END $dependencias$;
SET LOCAL ROLE vec_personal_propietario;
DROP FUNCTION vec_personal.contexto_alta_ejercicio_canonico_v1(jsonb);
DROP FUNCTION vec_personal.material_alta_ejercicio_canonico_v1(jsonb);
DROP FUNCTION vec_personal.solicitud_alta_ejercicio_canonica_v1(jsonb);
DROP FUNCTION vec_personal.fecha_civil_go_valida_v1(text);
DROP FUNCTION vec_personal.referencia_alta_ejercicio_valida_v1(text);
DROP FUNCTION vec_personal.campo_canonico_alta_v1(text,text);
DROP FUNCTION vec_personal.texto_json_go_v1(text);
COMMIT;
