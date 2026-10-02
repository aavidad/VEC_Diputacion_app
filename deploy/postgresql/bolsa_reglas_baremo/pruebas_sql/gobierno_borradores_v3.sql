\set ON_ERROR_STOP on
-- Comprobaciones de estructura y denegación de BR4. No sustituyen el ensayo
-- positivo con una concesión nominal nueva emitida por la autoridad V3 real.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL statement_timeout = '30s';
SET LOCAL lock_timeout = '2s';

DO $estructura$
DECLARE
    funcion oid := to_regprocedure(
        'vec_bolsa_reglas_baremo.operar_borrador_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    objeto oid;
    nombre text;
    rol text;
BEGIN
    IF funcion IS NULL OR NOT EXISTS (
        SELECT 1 FROM pg_proc
         WHERE oid = funcion AND prosecdef AND provolatile = 'v'
           AND proowner = 'vec_bolsa_reglas_baremo_propietario'::regrole
           AND proconfig @> ARRAY['search_path=pg_catalog', 'row_security=on']
    ) THEN
        RAISE EXCEPTION 'BR4: fachada ausente o autoridad incompatible';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_roles
         WHERE rolname = 'vec_bolsa_reglas_baremo_propietario'
           AND (rolcanlogin OR rolsuper OR rolbypassrls OR rolcreaterole OR rolcreatedb)
    ) THEN
        RAISE EXCEPTION 'BR4: propietario con privilegios incompatibles';
    END IF;
    FOREACH nombre IN ARRAY ARRAY[
        'acceso_borrador_v3', 'outbox_borrador_v3', 'recibo_borrador_v3'
    ] LOOP
        objeto := to_regclass('vec_bolsa_reglas_baremo.' || nombre);
        IF objeto IS NULL OR NOT EXISTS (
            SELECT 1 FROM pg_class
             WHERE oid = objeto AND relrowsecurity AND relforcerowsecurity
               AND relowner = 'vec_bolsa_reglas_baremo_propietario'::regrole
        ) OR (SELECT count(*) FROM pg_policy WHERE polrelid = objeto) <> 1
          OR NOT EXISTS (
              SELECT 1 FROM pg_policy
               WHERE polrelid = objeto
                 AND polroles = ARRAY['vec_bolsa_reglas_baremo_propietario'::regrole::oid]
          ) OR (SELECT count(*) FROM pg_trigger
                 WHERE tgrelid = objeto AND NOT tgisinternal
                   AND tgname IN ('inmutable', 'no_truncar')) <> 2 THEN
            RAISE EXCEPTION 'BR4: RLS, política o inmutabilidad incompatibles';
        END IF;
        IF EXISTS (
            SELECT 1 FROM pg_type t
            CROSS JOIN LATERAL aclexplode(coalesce(t.typacl, acldefault('T', t.typowner))) a
             WHERE t.typrelid = objeto AND a.grantee = 0
        ) THEN
            RAISE EXCEPTION 'BR4: tipo de fila accesible para PUBLIC';
        END IF;
        FOREACH rol IN ARRAY ARRAY[
            'public', 'vec_bolsa_reglas_baremo_ejecutor_gobierno',
            'vec_bolsa_reglas_baremo_ejecutor_consulta',
            'vec_bolsa_reglas_baremo_publicador_outbox'
        ] LOOP
            IF has_table_privilege(rol, objeto, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
               OR has_function_privilege(rol, funcion, 'EXECUTE') THEN
                RAISE EXCEPTION 'BR4: superficie cerrada concedida a runtime';
            END IF;
        END LOOP;
    END LOOP;
    IF EXISTS (
        SELECT 1 FROM pg_proc p
        CROSS JOIN LATERAL aclexplode(coalesce(p.proacl, acldefault('f', p.proowner))) a
         WHERE p.pronamespace = 'vec_bolsa_reglas_baremo'::regnamespace
           AND a.privilege_type = 'EXECUTE' AND a.grantee = 0
    ) THEN
        RAISE EXCEPTION 'BR4: función de Gobierno accesible para PUBLIC';
    END IF;
END
$estructura$;

SET LOCAL ROLE vec_bolsa_reglas_baremo_propietario;
DO $entrada_invalida$
BEGIN
    BEGIN
        PERFORM vec_bolsa_reglas_baremo.validar_material_borrador_v3(NULL, NULL);
        RAISE EXCEPTION 'BR4: entrada nula aceptada';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
    BEGIN
        PERFORM vec_bolsa_reglas_baremo.validar_material_borrador_v3(
            convert_to('{}', 'UTF8'), convert_to('{}', 'UTF8')
        );
        RAISE EXCEPTION 'BR4: proyecciones vacías aceptadas';
    EXCEPTION WHEN SQLSTATE '22023' THEN NULL;
    END;
END
$entrada_invalida$;
ROLLBACK;
