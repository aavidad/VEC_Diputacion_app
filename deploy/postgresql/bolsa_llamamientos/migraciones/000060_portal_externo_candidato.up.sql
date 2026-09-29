\set ON_ERROR_STOP on
-- Bolsa 000060: el proceso externo usa una identidad SQL propia. Solo puede
-- invocar las fachadas personales existentes; cada una consume material V3
-- del candidato o una marca ligada a ese consumo en la misma transacción.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000060', 0));

DO $pre$
DECLARE f regprocedure; nombre text;
        fachadas text[] := ARRAY[
 'vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz)',
 'vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[])',
 'vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
 'vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz)'];
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=session_user AND rolsuper)
    OR to_regrole('vec_bolsa_llamamientos_portal_externo') IS NOT NULL
    OR to_regrole('vec_bolsa_llamamientos_propietario') IS NULL
    OR to_regrole('vec_bolsa_llamamientos_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'Bolsa 000060: instalación DBA o preimagen incompatibles' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY fachadas LOOP
  f := to_regprocedure(nombre);
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef)
     OR NOT has_function_privilege('vec_bolsa_llamamientos_ejecutor',f,'EXECUTE')
     OR EXISTS (SELECT 1 FROM pg_proc p,
                LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                WHERE p.oid=f AND a.grantee=0 AND a.privilege_type='EXECUTE')
  THEN RAISE EXCEPTION 'Bolsa 000060: fachada personal ausente o abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $pre$;

CREATE ROLE vec_bolsa_llamamientos_portal_externo NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
DO $conexion$ BEGIN
 EXECUTE format('GRANT CONNECT ON DATABASE %I TO vec_bolsa_llamamientos_portal_externo',current_database());
END $conexion$;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
GRANT USAGE ON SCHEMA vec_bolsa_llamamientos TO vec_bolsa_llamamientos_portal_externo;
GRANT EXECUTE ON FUNCTION
 vec_bolsa_llamamientos.consultar_mi_bolsa_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.consultar_mi_bolsa_portal_v1(text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.consultar_historial_mi_bolsa_v1(text,timestamptz,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.manifestar_disposicion_oferta_v1(text,text,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.listar_ofertas_candidato_v1(text,timestamptz),
 vec_bolsa_llamamientos.solicitar_portal_candidato_v1(text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.responder_llamamiento_portal_v1(text,text,text,text,text,text,text,text,text,timestamptz,timestamptz,text[],text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.preparar_respuesta_portal_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.leer_portal_candidato_v1(text,timestamptz,text[]),
 vec_bolsa_llamamientos.confirmar_contacto_propio_v1(text,text,bigint,text,text,timestamptz,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_bolsa_llamamientos.leer_contacto_candidato_v1(text,timestamptz)
 TO vec_bolsa_llamamientos_portal_externo;
COMMIT;
