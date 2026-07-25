\set ON_ERROR_STOP 1

BEGIN;
SET TRANSACTION ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $prueba$
DECLARE
    v_propietario oid := 'vec_autorizacion_propietario'::regrole;
    v_tabla record;
    v_funcion record;
BEGIN
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_class c
          JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_autorizacion'
           AND c.relkind = 'r'
           AND c.relname LIKE 'motivo_cobertura_v1_%'
    ) <> 5 THEN
        RAISE EXCEPTION 'inventario de tablas de cobertura inesperado';
    END IF;

    FOR v_tabla IN
        SELECT c.oid, c.relname, c.relowner, c.relrowsecurity,
               c.relforcerowsecurity
          FROM pg_catalog.pg_class c
          JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_autorizacion'
           AND c.relkind = 'r'
           AND c.relname LIKE 'motivo_cobertura_v1_%'
    LOOP
        IF v_tabla.relowner <> v_propietario
           OR NOT v_tabla.relrowsecurity
           OR NOT v_tabla.relforcerowsecurity THEN
            RAISE EXCEPTION 'propietario o RLS inesperados en %',
                v_tabla.relname;
        END IF;
        IF NOT EXISTS (
            SELECT 1 FROM pg_catalog.pg_policy p
             WHERE p.polrelid = v_tabla.oid
               AND p.polname = 'acceso_propietario_exacto'
               AND p.polcmd = '*'
               AND p.polroles = ARRAY[v_propietario]
        ) THEN
            RAISE EXCEPTION 'política cerrada ausente en %', v_tabla.relname;
        END IF;
        IF pg_catalog.has_table_privilege(
               'vec_autorizacion_motivos_proyector',
               v_tabla.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           )
           OR pg_catalog.has_table_privilege(
               'vec_autorizacion_motivos_evaluador',
               v_tabla.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           )
           OR pg_catalog.has_table_privilege(
               'vec_contratacion_temporal_propietario',
               v_tabla.oid, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE'
           ) THEN
            RAISE EXCEPTION 'ACL directa indebida en %', v_tabla.relname;
        END IF;
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.aclexplode(COALESCE(
                  (SELECT relacl FROM pg_catalog.pg_class
                    WHERE oid = v_tabla.oid),
                  pg_catalog.acldefault('r', v_propietario)
              )) a
             WHERE a.grantee = 0
        ) THEN
            RAISE EXCEPTION 'PUBLIC conserva privilegios en %',
                v_tabla.relname;
        END IF;
    END LOOP;

    FOR v_funcion IN
        SELECT p.oid, p.proname, p.proowner, p.prosecdef, p.proconfig
          FROM pg_catalog.pg_proc p
          JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
         WHERE n.nspname = 'vec_autorizacion'
           AND (
               p.proname LIKE 'motivo_cobertura_v1_%'
               OR p.proname IN (
                   'publicar_motivos_cobertura_v1',
                   'retirar_motivos_cobertura_v1',
                   'resolver_motivo_cobertura_historico_v1',
                   'resolver_motivo_cobertura_actual_v1'
               )
           )
    LOOP
        IF v_funcion.proowner <> v_propietario THEN
            RAISE EXCEPTION 'propietario de función inesperado: %',
                v_funcion.proname;
        END IF;
        IF v_funcion.proname IN (
               'publicar_motivos_cobertura_v1',
               'retirar_motivos_cobertura_v1',
               'resolver_motivo_cobertura_historico_v1',
               'resolver_motivo_cobertura_actual_v1'
           )
           AND (
               NOT v_funcion.prosecdef
               OR NOT (
                   v_funcion.proconfig @>
                   ARRAY['search_path=pg_catalog, pg_temp']::text[]
               )
           ) THEN
            RAISE EXCEPTION 'frontera sin cierre SECURITY DEFINER: %',
                v_funcion.proname;
        END IF;
        IF EXISTS (
            SELECT 1
              FROM pg_catalog.aclexplode(COALESCE(
                  (SELECT proacl FROM pg_catalog.pg_proc
                    WHERE oid = v_funcion.oid),
                  pg_catalog.acldefault('f', v_propietario)
              )) a
             WHERE a.grantee = 0 AND a.privilege_type = 'EXECUTE'
        ) THEN
            RAISE EXCEPTION 'PUBLIC puede ejecutar %', v_funcion.proname;
        END IF;
    END LOOP;

    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_proc p
          JOIN pg_catalog.pg_namespace n ON n.oid = p.pronamespace
         WHERE n.nspname = 'vec_autorizacion'
           AND (
               p.proname LIKE 'motivo_cobertura_v1_%'
               OR p.proname IN (
                   'publicar_motivos_cobertura_v1',
                   'retirar_motivos_cobertura_v1',
                   'resolver_motivo_cobertura_historico_v1',
                   'resolver_motivo_cobertura_actual_v1'
               )
           )
    ) <> 8 THEN
        RAISE EXCEPTION 'inventario de funciones de cobertura inesperado';
    END IF;

    IF NOT pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_proyector',
           'vec_autorizacion.publicar_motivos_cobertura_v1(text,bigint,text,text,integer,text,text,timestamp with time zone,jsonb)',
           'EXECUTE'
       )
       OR NOT pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_proyector',
           'vec_autorizacion.retirar_motivos_cobertura_v1(text,bigint,text,text,integer,text,text,text,timestamp with time zone)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_proyector',
           'vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamp with time zone)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_proyector',
           'vec_autorizacion.resolver_motivo_cobertura_actual_v1(text,integer,text,text,text)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'superficie del proyector inesperada';
    END IF;
    IF NOT pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_evaluador',
           'vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamp with time zone)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_evaluador',
           'vec_autorizacion.resolver_motivo_cobertura_actual_v1(text,integer,text,text,text)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_autorizacion_motivos_evaluador',
           'vec_autorizacion.publicar_motivos_cobertura_v1(text,bigint,text,text,integer,text,text,timestamp with time zone,jsonb)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'superficie del evaluador inesperada';
    END IF;
    IF NOT pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           'vec_autorizacion.resolver_motivo_cobertura_actual_v1(text,integer,text,text,text)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           'vec_autorizacion.resolver_motivo_cobertura_historico_v1(text,integer,text,text,text,timestamp with time zone)',
           'EXECUTE'
       )
       OR pg_catalog.has_function_privilege(
           'vec_contratacion_temporal_propietario',
           'vec_autorizacion.publicar_motivos_cobertura_v1(text,bigint,text,text,integer,text,text,timestamp with time zone,jsonb)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'superficie del propietario CT inesperada';
    END IF;
END
$prueba$;

-- Readiness inicial: instalar la estructura no inventa un catálogo maestro.
SET LOCAL ROLE vec_contratacion_temporal_propietario;
DO $readiness$
BEGIN
    IF vec_autorizacion.resolver_motivo_cobertura_actual_v1(
        'motivos_cobertura', 1, pg_catalog.repeat('a', 64),
        'rectificacion_decision', 'cobertura.motivo.rectificacion'
    ) IS NOT FALSE THEN
        RAISE EXCEPTION 'la proyección vacía se presentó como preparada';
    END IF;
END
$readiness$;
RESET ROLE;

ROLLBACK;
