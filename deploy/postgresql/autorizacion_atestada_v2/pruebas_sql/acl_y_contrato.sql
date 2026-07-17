BEGIN;
SET LOCAL search_path = pg_catalog;

DO $comprobar$
DECLARE
    oid_esquema oid;
    oid_esquema_checkpoint_confianza oid;
    oid_propietario oid;
    oid_migrador oid;
    oid_emisor oid;
    oid_consumidor oid;
    oid_propietario_confianza oid;
    oid_registro oid;
    oid_reconciliacion oid;
    oid_material oid;
    oid_identidad oid;
    oid_checkpoint_clave oid;
    oid_cotejo oid;
    oid_cotejo_exacto oid;
    oid_instante_nominal oid;
    oid_checkpoint_confianza oid;
BEGIN
    SELECT oid INTO STRICT oid_esquema FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_autorizacion_atestada_v2';
    SELECT oid INTO STRICT oid_propietario FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_propietario'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
       AND NOT rolinherit;
    SELECT oid INTO STRICT oid_migrador FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_migrador'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
       AND NOT rolinherit;
    SELECT oid INTO STRICT oid_emisor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_emisor_capacidad'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
       AND rolinherit;
    SELECT oid INTO STRICT oid_consumidor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_consumidor'
       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreatedb
       AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
       AND rolinherit;
    SELECT oid INTO STRICT oid_propietario_confianza
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_confianza_atestacion_v2_propietario';
    SELECT oid INTO STRICT oid_esquema_checkpoint_confianza
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_confianza_atestacion_v2_consumo_atestado';
    oid_registro := to_regprocedure(
        'vec_autorizacion_atestada_v2.registrar_y_consumir_decision_v2_atestada(bytea,bytea,bytea,bytea,bytea,bytea,jsonb)'
    );
    oid_reconciliacion := to_regprocedure(
        'vec_autorizacion_atestada_v2.reconciliar_consumo_decision_v2(text,text,text,text,text)'
    );
    oid_material := to_regprocedure(
        'vec_autorizacion_atestada_v2.obtener_material_emisor_capacidad()'
    );
    oid_identidad := to_regprocedure(
        'vec_autorizacion_atestada_v2.identidad_runtime_valida(text,boolean)'
    );
    oid_checkpoint_clave := to_regprocedure(
        'vec_autorizacion_atestada_v2.avanzar_checkpoint_gobierno_clave()'
    );
    oid_cotejo := to_regprocedure(
        'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone)'
    );
    oid_cotejo_exacto := to_regprocedure(
        'vec_confianza_atestacion_v2.cotejar_confianza_consumo_atestado_en_v1(text,text,timestamp with time zone,timestamp with time zone,text,numeric,text,timestamp with time zone,timestamp with time zone,text,text,timestamp with time zone,timestamp with time zone)'
    );
    oid_instante_nominal := to_regprocedure(
        'vec_autorizacion.obtener_instante_decision_atestada_v2(text,text)'
    );
    oid_checkpoint_confianza := to_regprocedure(
        'vec_confianza_atestacion_v2_consumo_atestado.avanzar_checkpoint_gobierno()'
    );
    IF oid_registro IS NULL OR oid_reconciliacion IS NULL
       OR oid_material IS NULL OR oid_identidad IS NULL
       OR oid_checkpoint_clave IS NULL OR oid_cotejo IS NULL
       OR oid_cotejo_exacto IS NULL OR oid_instante_nominal IS NULL
       OR oid_checkpoint_confianza IS NULL THEN
        RAISE EXCEPTION 'faltan funciones del contrato atestado';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.pg_auth_members
         WHERE member = oid_migrador) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = oid_migrador AND roleid = oid_propietario
              AND admin_option IS FALSE AND inherit_option IS FALSE
              AND set_option IS TRUE
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member IN (oid_propietario, oid_emisor, oid_consumidor)
       ) THEN
        RAISE EXCEPTION 'topologia de roles base incorrecta';
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
    IF has_schema_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad',
           oid_esquema_checkpoint_confianza, 'USAGE'
       ) OR has_schema_privilege(
           'vec_autorizacion_atestada_v2_consumidor',
           oid_esquema_checkpoint_confianza, 'USAGE'
       ) OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace AS espacio
           CROSS JOIN LATERAL pg_catalog.aclexplode(
               COALESCE(
                   espacio.nspacl,
                   pg_catalog.acldefault('n', espacio.nspowner)
               )
           ) AS privilegio
          WHERE espacio.oid = oid_esquema_checkpoint_confianza
            AND privilegio.grantee = 0
            AND privilegio.privilege_type IN ('USAGE', 'CREATE')
       ) THEN
        RAISE EXCEPTION 'ACL del esquema checkpoint de confianza abierta';
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
         WHERE clase.relnamespace = oid_esquema_checkpoint_confianza
           AND clase.relkind = 'r'
           AND (has_table_privilege(
                   'vec_autorizacion_atestada_v2_emisor_capacidad',
                   clase.oid,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               ) OR has_table_privilege(
                   'vec_autorizacion_atestada_v2_consumidor', clase.oid,
                   'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER'
               ) OR EXISTS (
                   SELECT 1 FROM pg_catalog.aclexplode(
                       COALESCE(
                           clase.relacl,
                           pg_catalog.acldefault('r', clase.relowner)
                       )
                   ) AS privilegio
                    WHERE privilegio.grantee = 0
               ))
    ) THEN
        RAISE EXCEPTION 'checkpoint de confianza expuesto';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_class AS clase
         WHERE clase.relnamespace = oid_esquema AND clase.relkind = 'r'
           AND (NOT clase.relrowsecurity OR NOT clase.relforcerowsecurity)
    ) OR (SELECT count(*)
            FROM pg_catalog.pg_policy AS politica
            JOIN pg_catalog.pg_class AS clase ON clase.oid = politica.polrelid
           WHERE clase.relnamespace = oid_esquema) <> 9 THEN
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
    IF (SELECT count(*) FROM pg_catalog.pg_class
         WHERE relnamespace = oid_esquema_checkpoint_confianza
           AND relkind = 'r' AND relrowsecurity AND relforcerowsecurity) <> 1
       OR (SELECT count(*)
             FROM pg_catalog.pg_policy AS politica
             JOIN pg_catalog.pg_class AS clase
               ON clase.oid = politica.polrelid
            WHERE clase.relnamespace = oid_esquema_checkpoint_confianza
              AND politica.polname = 'propietario_exacto'
              AND politica.polroles = ARRAY[oid_propietario_confianza]
              AND politica.polcmd = '*'
              AND politica.polqual IS NOT NULL
              AND politica.polwithcheck IS NOT NULL) <> 1
       OR (SELECT count(*) FROM
              vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno
           ) <> 1 THEN
        RAISE EXCEPTION 'RLS o cardinalidad del checkpoint de confianza invalida';
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
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_cotejo_exacto,
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad',
           oid_cotejo_exacto, 'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_consumidor', oid_instante_nominal,
           'EXECUTE'
       ) OR has_function_privilege(
           'vec_autorizacion_atestada_v2_emisor_capacidad',
           oid_instante_nominal, 'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'runtime alcanza cotejo de confianza o puerta nominal';
    END IF;
    IF NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_propietario', oid_cotejo, 'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_propietario', oid_cotejo_exacto,
           'EXECUTE'
       ) OR NOT has_function_privilege(
           'vec_autorizacion_atestada_v2_propietario', oid_instante_nominal,
           'EXECUTE'
       ) THEN
        RAISE EXCEPTION 'el propietario no puede hacer el cotejo estrecho';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS funcion
         WHERE funcion.oid = ANY (ARRAY[
             oid_registro, oid_reconciliacion, oid_material, oid_cotejo
             , oid_cotejo_exacto, oid_instante_nominal, oid_identidad,
             oid_checkpoint_clave, oid_checkpoint_confianza
         ])
           AND (NOT funcion.prosecdef OR funcion.proconfig IS DISTINCT FROM
                ARRAY['search_path=pg_catalog']::text[])
    ) THEN
        RAISE EXCEPTION 'SECURITY DEFINER o search_path no cerrado';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = oid_instante_nominal AND provolatile = 'v'
    ) THEN
        RAISE EXCEPTION 'el lector del INSERT nominal no usa snapshot fresco';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = oid_registro
           AND prosrc LIKE '%registrar_decision_solicitud_ligada_v2_si_vigente%'
           AND prosrc LIKE '%cotejar_confianza_consumo_atestado_v1%'
           AND prosrc LIKE '%cotejar_confianza_consumo_atestado_en_v1%'
           AND prosrc LIKE '%obtener_instante_decision_atestada_v2%'
           AND prosrc LIKE '%pg_advisory_xact_lock_shared%'
    ) THEN
        RAISE EXCEPTION 'la puerta no liga autoridad y confianza';
    END IF;
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc AS funcion
        CROSS JOIN LATERAL pg_catalog.aclexplode(
            COALESCE(funcion.proacl, pg_catalog.acldefault('f', funcion.proowner))
        ) AS privilegio
        WHERE (funcion.pronamespace = oid_esquema OR funcion.oid = ANY (
                   ARRAY[
                       oid_cotejo, oid_cotejo_exacto,
                       oid_instante_nominal, oid_checkpoint_confianza
                   ]
               ))
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
    IF EXISTS (
        SELECT 1 FROM pg_catalog.pg_type AS tipo
         WHERE tipo.typnamespace = oid_esquema_checkpoint_confianza
           AND tipo.typelem = 0 AND tipo.typisdefined
           AND (has_type_privilege(
                   'vec_autorizacion_atestada_v2_emisor_capacidad',
                   tipo.oid, 'USAGE'
               ) OR has_type_privilege(
                   'vec_autorizacion_atestada_v2_consumidor',
                   tipo.oid, 'USAGE'
               ) OR EXISTS (
                   SELECT 1 FROM pg_catalog.aclexplode(
                       COALESCE(
                           tipo.typacl,
                           pg_catalog.acldefault('T', tipo.typowner)
                       )
                   ) AS privilegio
                    WHERE privilegio.grantee = 0
                      AND privilegio.privilege_type = 'USAGE'
               ))
    ) THEN
        RAISE EXCEPTION 'tipo del checkpoint de confianza expuesto';
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_proc
         WHERE oid = oid_identidad
           AND prosrc NOT LIKE '%pg_has_role%'
           AND prosrc LIKE '%admin_option%'
           AND prosrc LIKE '%inherit_option%'
           AND prosrc LIKE '%set_option%'
    ) THEN
        RAISE EXCEPTION 'identidad runtime no exige membresia directa exacta';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.pg_trigger
         WHERE tgname = 'z90_avanzar_checkpoint_gobierno_clave'
           AND NOT tgisinternal) <> 3
       OR (SELECT count(*) FROM pg_catalog.pg_trigger
            WHERE tgname = 'a00_sellar_conocimiento_gobierno_clave'
              AND NOT tgisinternal) <> 3
       OR (SELECT count(*) FROM pg_catalog.pg_trigger
            WHERE tgname = 'z90_avanzar_checkpoint_consumo_atestado'
              AND NOT tgisinternal) <> 8
       OR (SELECT count(*) FROM pg_catalog.pg_trigger
            WHERE tgname = 'a05_sellar_conocimiento_consumo_atestado'
              AND NOT tgisinternal) <> 8
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_trigger
            WHERE tgrelid =
                  'vec_autorizacion_atestada_v2.control_cadena_auditoria'::regclass
              AND tgname = 'control_cadena_auditoria_no_eliminar'
       ) THEN
        RAISE EXCEPTION 'triggers de checkpoint o control incompletos';
    END IF;
    IF (SELECT count(*) FROM pg_catalog.pg_constraint
         WHERE conrelid =
               'vec_autorizacion_atestada_v2.atestacion_decision_v2'::regclass
           AND contype = 'f') < 4
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_autorizacion_atestada_v2.consumo_decision_v2'::regclass
              AND contype = 'f' AND cardinality(conkey) = 3
       ) OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_autorizacion_atestada_v2.auditoria_consumo_v2'::regclass
              AND contype = 'f' AND cardinality(conkey) = 4
       ) OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_constraint
            WHERE conrelid =
                  'vec_autorizacion_atestada_v2.clave_capacidad_version'::regclass
              AND contype = 'u'
              AND conname =
                  'clave_capacidad_version_huella_secreto_sha256_key'
       ) THEN
        RAISE EXCEPTION 'vinculos compuestos o unicidad de secreto incompletos';
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

SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;
DO $controles_no_retirables$
DECLARE
    rechazo_borrado_control boolean := false;
    rechazo_truncado_control boolean := false;
    rechazo_borrado_checkpoint boolean := false;
BEGIN
    BEGIN
        DELETE FROM vec_autorizacion_atestada_v2.control_cadena_auditoria;
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazo_borrado_control := true;
    END;
    BEGIN
        EXECUTE 'TRUNCATE vec_autorizacion_atestada_v2.control_cadena_auditoria';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazo_truncado_control := true;
    END;
    BEGIN
        DELETE FROM vec_autorizacion_atestada_v2.checkpoint_gobierno_clave;
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazo_borrado_checkpoint := true;
    END;
    IF NOT rechazo_borrado_control OR NOT rechazo_truncado_control
       OR NOT rechazo_borrado_checkpoint THEN
        RAISE EXCEPTION 'un control mutable pudo retirarse';
    END IF;
END
$controles_no_retirables$;
RESET ROLE;

SET LOCAL ROLE vec_confianza_atestacion_v2_propietario;
DO $checkpoint_confianza_no_retirar$
DECLARE
    rechazo_borrado boolean := false;
    rechazo_truncado boolean := false;
BEGIN
    BEGIN
        DELETE FROM
            vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno;
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazo_borrado := true;
    END;
    BEGIN
        EXECUTE 'TRUNCATE vec_confianza_atestacion_v2_consumo_atestado.checkpoint_gobierno';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        rechazo_truncado := true;
    END;
    IF NOT rechazo_borrado OR NOT rechazo_truncado THEN
        RAISE EXCEPTION 'el checkpoint de confianza pudo retirarse';
    END IF;
END
$checkpoint_confianza_no_retirar$;
RESET ROLE;
ROLLBACK;
