\set ON_ERROR_STOP on
-- Usuarios 000012: la clave propia del proceso externo solo puede usarse
-- cuando la población está vacía o todas sus referencias son ya exclusivas.
-- La función devuelve un booleano, sin recuentos ni referencias. No reclavea.
-- Requiere Usuarios 000010 (#145) y AD3-112 (#151); instalación con la app
-- externa parada. Nunca ejecutar sobre una base con correos externos previos
-- esperando continuidad: el arranque debe denegarse hasta su reclaveado.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_usuarios:migracion:000012',0));
DO $pre$ DECLARE t text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
    OR to_regnamespace('vec_usuarios_correos_externo') IS NULL
    OR to_regnamespace('vec_usuarios_correos_interno') IS NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_correos_externo_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_preflight_externo' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_usuarios_correos_preflight_externo')
    OR to_regprocedure('vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()') IS NOT NULL
 THEN RAISE EXCEPTION 'Usuarios 000012: preimagen incompatible' USING ERRCODE='55000'; END IF;
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio','correos_contexto'] LOOP
  IF to_regclass(format('vec_usuarios_correos_externo.%I',t)) IS NULL
     OR NOT EXISTS (SELECT 1 FROM pg_class c WHERE c.oid=to_regclass(format('vec_usuarios_correos_externo.%I',t))
       AND c.relkind='r' AND c.relowner='vec_usuarios_correos_externo_propietario'::regrole
       AND c.relrowsecurity AND c.relforcerowsecurity)
  THEN RAISE EXCEPTION 'Usuarios 000012: tabla externa incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

-- La función no pertenece al propietario de correos: FORCE RLS filtraría
-- todas las filas sin el contexto de una operación. Este rol NOLOGIN recibe
-- solo SELECT externo y una política propia. El login preflight no es miembro.
CREATE ROLE vec_usuarios_correos_preflight_externo NOLOGIN NOSUPERUSER NOCREATEDB
 NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT USAGE ON SCHEMA vec_usuarios_correos_externo TO vec_usuarios_correos_preflight_externo,
 vec_autorizacion_atestada_v3_preflight_externo;
GRANT CREATE ON SCHEMA vec_usuarios_correos_externo TO vec_usuarios_correos_preflight_externo;
DO $permisos$ DECLARE t text;
BEGIN
 FOREACH t IN ARRAY ARRAY['correos_conjunto','correos_direccion','correos_desafio','correos_intento_fallido',
  'correos_historia','correos_recibo','correos_envio','correos_contexto'] LOOP
  EXECUTE format('GRANT SELECT ON vec_usuarios_correos_externo.%I TO vec_usuarios_correos_preflight_externo',t);
  EXECUTE format('CREATE POLICY preflight_externo_vacio ON vec_usuarios_correos_externo.%I FOR SELECT TO vec_usuarios_correos_preflight_externo USING (true)',t);
 END LOOP;
END $permisos$;

-- Las filas anteriores se conservan intactas y hacen fallar el preflight.
-- NOT VALID permite instalarlas sin reescribir historia, pero PostgreSQL
-- aplica estos CHECK a toda escritura nueva; el proceso antiguo no puede
-- introducir otra referencia compartida después de esta migración.
ALTER TABLE vec_usuarios_correos_externo.correos_conjunto
 ADD CONSTRAINT clave_igualdad_externa_check CHECK
 (clave_igualdad_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')) NOT VALID;
ALTER TABLE vec_usuarios_correos_externo.correos_direccion
 ADD CONSTRAINT claves_direccion_externas_check CHECK
 (clave_sobre_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1')
  AND clave_igualdad_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')) NOT VALID;
ALTER TABLE vec_usuarios_correos_externo.correos_desafio
 ADD CONSTRAINT clave_codigo_externa_check CHECK
 (clave_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-codigo:v1')) NOT VALID;
ALTER TABLE vec_usuarios_correos_externo.correos_intento_fallido
 ADD CONSTRAINT clave_semantica_externa_check CHECK
 (huella_clave_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-semantica:v1')) NOT VALID;
ALTER TABLE vec_usuarios_correos_externo.correos_recibo
 ADD CONSTRAINT clave_semantica_externa_check CHECK
 (huella_clave_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-semantica:v1')) NOT VALID;
ALTER TABLE vec_usuarios_correos_externo.correos_contexto
 ADD CONSTRAINT clave_semantica_externa_check CHECK
 (huella_clave_ref IS NULL OR huella_clave_ref IN ('clave:kms:desarrollo:usuarios-correos-externo-semantica:v1')) NOT VALID;

CREATE FUNCTION vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()
RETURNS boolean LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path=pg_catalog SET row_security=on AS $f$
BEGIN
 IF session_user::text<>'vec_externo_preflight_v3_desarrollo' THEN RETURN false; END IF;
 RETURN NOT EXISTS (SELECT 1 FROM vec_usuarios_correos_externo.correos_conjunto
      WHERE clave_igualdad_ref<>'clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')
    AND NOT EXISTS (SELECT 1 FROM vec_usuarios_correos_externo.correos_direccion
      WHERE clave_sobre_ref<>'clave:kms:desarrollo:usuarios-correos-externo-cifrado:v1'
         OR clave_igualdad_ref<>'clave:kms:desarrollo:usuarios-correos-externo-igualdad:v1')
    AND NOT EXISTS (SELECT 1 FROM vec_usuarios_correos_externo.correos_desafio
      WHERE clave_ref<>'clave:kms:desarrollo:usuarios-correos-externo-codigo:v1')
    AND NOT EXISTS (SELECT 1 FROM vec_usuarios_correos_externo.correos_intento_fallido
      WHERE huella_clave_ref<>'clave:kms:desarrollo:usuarios-correos-externo-semantica:v1')
    AND NOT EXISTS (SELECT 1 FROM vec_usuarios_correos_externo.correos_recibo
      WHERE huella_clave_ref<>'clave:kms:desarrollo:usuarios-correos-externo-semantica:v1')
    AND NOT EXISTS (SELECT 1 FROM vec_usuarios_correos_externo.correos_contexto);
END $f$;
REVOKE ALL ON FUNCTION vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1() FROM PUBLIC;
ALTER FUNCTION vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()
 OWNER TO vec_usuarios_correos_preflight_externo;
REVOKE CREATE ON SCHEMA vec_usuarios_correos_externo FROM vec_usuarios_correos_preflight_externo;
GRANT EXECUTE ON FUNCTION vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()
 TO vec_autorizacion_atestada_v3_preflight_externo;

DO $acl$ DECLARE f regprocedure:=to_regprocedure('vec_usuarios_correos_externo.poblacion_sin_claves_ajenas_v1()');
BEGIN
 IF f IS NULL
    OR (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_usuarios_correos_preflight_externo'::regrole
    OR has_schema_privilege('vec_usuarios_correos_preflight_externo','vec_usuarios_correos_externo','CREATE')
    OR has_table_privilege('vec_autorizacion_atestada_v3_preflight_externo','vec_usuarios_correos_externo.correos_direccion','SELECT')
    OR has_function_privilege('public',f,'EXECUTE')
    OR NOT has_function_privilege('vec_autorizacion_atestada_v3_preflight_externo',f,'EXECUTE')
    OR EXISTS (SELECT 1 FROM pg_auth_members WHERE roleid='vec_usuarios_correos_preflight_externo'::regrole)
 THEN RAISE EXCEPTION 'Usuarios 000012: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
