\set ON_ERROR_STOP on
-- Fachada nominal de ContextoActor para los avisos de llamamiento. Usuarios
-- necesita saber qué persona corresponde a la referencia de candidato que
-- conoce Bolsa para elegir su correo activo de «Mis correos». Devuelve la
-- persona sólo si hay exactamente un vínculo de candidato activo y vigente, y
-- la persona también lo está; en cualquier otro caso devuelve NULL.
-- Sólo el propietario de Usuarios la ejecuta, dentro de su lectura
-- autorizada por V3 (AD3-109). No expone ningún otro dato de identidad.
-- DOWN prohibido tras historia: la concesión y las lecturas consumidoras perduran.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contexto_actor_v1:migracion:persona_candidato_avisos:v1', 0));

DO $preimagen$
DECLARE
    dueno oid := pg_catalog.to_regrole('vec_contexto_actor_v1_propietario');
    usuarios oid := pg_catalog.to_regrole('vec_usuarios_propietario');
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('transaction_isolation') <> 'read committed' THEN
        RAISE EXCEPTION 'ContextoActor 000010 requiere superusuario y transaccion ordinaria'
            USING ERRCODE = '42501';
    END IF;
    IF dueno IS NULL OR usuarios IS NULL
       OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.acreditar_persona_tercero_v1(text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.persona_candidato_avisos_v1(text)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE oid = usuarios AND (rolcanlogin OR rolbypassrls OR rolsuper))
       OR EXISTS (
           SELECT 1 FROM (VALUES ('vinculo_referencia_actual'),('vinculo_referencia_versiones'),
                                 ('persona_actual'),('persona_versiones')) t(nombre)
            WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
                               WHERE c.oid = pg_catalog.to_regclass('vec_contexto_actor_v1.'||t.nombre)
                                 AND c.relowner = dueno AND c.relkind = 'r')) THEN
        RAISE EXCEPTION 'ContextoActor 000010: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END
$preimagen$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

CREATE FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(p_candidato_ref text)
RETURNS text
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
DECLARE
    personas text[];
    ahora timestamptz := pg_catalog.statement_timestamp();
BEGIN
    IF vec_contexto_actor_v1.referencia_valida(p_candidato_ref, 'can_') IS NOT TRUE THEN
        RETURN NULL;
    END IF;
    SELECT pg_catalog.array_agg(DISTINCT v.persona_ref)
      INTO personas
      FROM vec_contexto_actor_v1.vinculo_referencia_actual AS a
      JOIN vec_contexto_actor_v1.vinculo_referencia_versiones AS v
        USING (vinculo_ref, version)
      JOIN vec_contexto_actor_v1.persona_actual AS pa
        ON pa.persona_ref = v.persona_ref
      JOIN vec_contexto_actor_v1.persona_versiones AS pv
        ON pv.persona_ref = pa.persona_ref AND pv.version = pa.version
     WHERE v.tipo = 'candidato'
       AND v.referencia = p_candidato_ref
       AND v.estado = 'activo'
       AND ahora >= v.vigente_desde AND ahora < v.vigente_hasta
       AND pv.estado = 'activo'
       AND ahora >= pv.vigente_desde AND ahora < pv.vigente_hasta;
    IF pg_catalog.cardinality(personas) IS DISTINCT FROM 1 THEN
        RETURN NULL;
    END IF;
    RETURN personas[1];
END
$funcion$;

REVOKE ALL ON FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(text)
    FROM PUBLIC, vec_contexto_actor_v1_runtime;
GRANT USAGE ON SCHEMA vec_contexto_actor_v1 TO vec_usuarios_propietario;
GRANT EXECUTE ON FUNCTION vec_contexto_actor_v1.persona_candidato_avisos_v1(text)
    TO vec_usuarios_propietario;

DO $postimagen$
DECLARE f regprocedure := 'vec_contexto_actor_v1.persona_candidato_avisos_v1(text)'::regprocedure;
BEGIN
    IF (SELECT proowner FROM pg_catalog.pg_proc WHERE oid = f) IS DISTINCT FROM 'vec_contexto_actor_v1_propietario'::regrole
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid = f) IS NOT TRUE
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid = f) IS DISTINCT FROM ARRAY['search_path=pg_catalog']
       OR NOT pg_catalog.has_function_privilege('vec_usuarios_propietario', f, 'EXECUTE')
       OR pg_catalog.has_function_privilege('vec_contexto_actor_v1_runtime', f, 'EXECUTE')
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                  CROSS JOIN LATERAL pg_catalog.aclexplode(coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
                   WHERE p.oid = f AND (a.grantee = 0 OR a.grantee NOT IN (p.proowner, 'vec_usuarios_propietario'::regrole))) THEN
        RAISE EXCEPTION 'ContextoActor 000010: ACL incompatible' USING ERRCODE = '55000';
    END IF;
END
$postimagen$;
COMMIT;
