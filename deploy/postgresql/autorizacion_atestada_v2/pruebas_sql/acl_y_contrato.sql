BEGIN;
SET LOCAL search_path = pg_catalog;

DO $comprobar$
DECLARE
    oid_esquema oid;
    oid_propietario oid;
    oid_emisor oid;
    oid_consumidor oid;
    oid_registro oid;
    oid_reconciliacion oid;
    oid_material oid;
    oid_cotejo oid;
BEGIN
    SELECT oid INTO STRICT oid_esquema FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_autorizacion_atestada_v2';
    SELECT oid INTO STRICT oid_propietario FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_propietario'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls;
    SELECT oid INTO STRICT oid_emisor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_emisor_capacidad'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls;
    SELECT oid INTO STRICT oid_consumidor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_consumidor'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls;
    oid_registro := to_regprocedure(
        'vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
    );
    oid_reconciliacion := to_regprocedure(
        'vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(text,text,text,text,text)'
    );
    oid_material := to_regprocedure(
        'vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()'
    );
    oid_cotejo := to_regprocedure(
        'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone)'
    );
    IF oid_registro IS NULL OR oid_reconciliacion IS NULL
       OR oid_material IS NULL OR oid_cotejo IS NULL THEN
        RAISE EXCEPTION 'faltan funciones del contrato atestado';
    END IF;
    IF NOT has_schema_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad', oid_esquema,
           'USAGE'
       ) OR has_schema_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad', oid_esquema,
           'CREATE'
       ) OR NOT has_schema_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_esquema, 'USAGE'
       ) OR has_schema_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_esquema, 'CREATE'
       ) THEN
        RAISE EXCEPTION 'ACL de esquema incorrecta';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_class AS clase
         WHERE clase.relnamespace = oid_esquema
           AND clase.relkind IN ('r', 'p', 'v', 'm', 'S')
           AND (has_table_privilege(
                   'vec_autorizacion_atestada_v2_emisor_capacidad',
                   clase.oid,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               ) OR has_table_privilege(
                   'vec_autorizacion_atestada_v2_consumidor',
                   clase.oid,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               ))
    ) THEN
        RAISE EXCEPTION 'runtime conserva DML o lectura directa';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_class AS clase
         WHERE clase.relnamespace = oid_esquema AND clase.relkind = 'r'
           AND (NOT clase.relrowsecurity OR NOT clase.relforcerowsecurity)
    ) OR (SELECT count(*)
            FROM pg_catalog.pg_policy AS politica
            JOIN pg_catalog.pg_class AS clase ON clase.oid = politica.polrelid
           WHERE clase.relnamespace = oid_esquema) <> 8 THEN
        RAISE EXCEPTION 'RLS FORCE incompleto';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_policy AS politica
        JOIN pg_catalog.pg_class AS clase ON clase.oid = politica.polrelid
        WHERE clase.relnamespace = oid_esquema
          AND (politica.polname <> 'propietario_exacto'
               OR politica.polroles <> ARRAY[oid_propietario])
    ) THEN
        RAISE EXCEPTION 'politica RLS no es del propietario exacto';
    END IF;
    IF NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad', oid_material,
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad', oid_registro,
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad',
           oid_reconciliacion, 'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_registro,
           'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_reconciliacion,
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_material,
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'superficie runtime de funciones incorrecta';
    END IF;
    IF has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_cotejo, 'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad', oid_cotejo,
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor',
           'vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)',
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad',
           'vec_autorizacion.registrar_decision_solicitud_ligada_v2_si_vigente(bytea,bytea)',
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'runtime alcanza cotejo de confianza o puerta nominal';
    END IF;
    IF NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_propietario', oid_cotejo, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'el propietario no puede hacer el cotejo estrecho';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS funcion
         WHERE funcion.oid = ANY (ARRAY[
             oid_registro, oid_reconciliacion, oid_material, oid_cotejo
         ])
           AND (NOT funcion.prosecdef OR funcion.proconfig IS DISTINCT FROM
                ARRAY['search_path=pg_catalog']::text[])
    ) THEN
        RAISE EXCEPTION 'SECURITY DEFINER o search_path no cerrado';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = oid_registro
           AND prosrc LIKE '%registrar_decision_solicitud_ligada_v2_si_vigente%'
           AND prosrc LIKE '%cotejar_confianza_consumo_atestado_v1%'
           AND prosrc LIKE '%pg_advisory_xact_lock_shared%'
    ) THEN
        RAISE EXCEPTION 'la puerta no liga autoridad y confianza';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS funcion
        CROSS JOIN LATERAL pg_catalog.aclexplode(
            COALESCE(funcion.proacl, pg_catalog.acldefault('f', funcion.proowner))
        ) AS privilegio
        WHERE (funcion.pronamespace = oid_esquema OR funcion.oid = oid_cotejo)
          AND privilegio.grantee = 0
          AND privilegio.privilege_type = 'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'PUBLIC conserva EXECUTE';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_type AS tipo
         WHERE tipo.typnamespace = oid_esquema AND tipo.typelem = 0
           AND tipo.typisdefined
           AND (has_type_privilege(
                   'vec_autorizacion_atestada_v2_emisor_capacidad',
                   tipo.oid, 'USAGE'
               ) OR has_type_privilege(
                   'vec_autorizacion_atestada_v2_consumidor',
                   tipo.oid, 'USAGE'
               ))
    ) THEN
        RAISE EXCEPTION 'runtime conserva USAGE de tipos fila';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.atestacion_decision_v2'::regclass
           AND contype = 'f'
           AND confrelid =
               'vec_confianza_atestacion_v2.configuracion_raiz'::regclass
    ) THEN
        RAISE EXCEPTION 'falta FK al catalogo historico';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.consumo_capacidad_v2'::regclass
           AND conname = 'consumo_capacidad_v2_pkey' AND contype = 'p'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.atestacion_decision_v2'::regclass
           AND conname = 'atestacion_decision_v2_decision_ref_key'
           AND contype = 'u'
    ) OR NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass
           AND conname = 'consumo_decision_v2_efecto_ref_key'
           AND contype = 'u'
    ) THEN
        RAISE EXCEPTION 'faltan barreras de replay para nonce/decision/efecto';
    END IF;
END
$comprobar$;
ROLLBACK;
