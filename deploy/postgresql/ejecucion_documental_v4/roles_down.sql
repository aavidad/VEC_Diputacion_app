-- Destructivo: ejecutar solo despues de las migraciones descendentes y de
-- retirar todas las cuentas LOGIN miembro de estos grupos.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_ejecucion_documental_v4:roles_down:v2', 0)
);

-- Los catalogos de roles son compartidos por todo el cluster. El orden es
-- deliberado: pg_authid impide que un GRANT concurrente resuelva OID que van
-- a desaparecer y pg_auth_members conserva estable el inventario hasta el
-- COMMIT. Requiere superusuario y una ventana de mantenimiento del cluster.
LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;

-- Se valida antes de mutar ACL o membresias. Se admite exclusivamente el
-- enlace estructural creado por roles_up.sql; cualquier enlace entrante o
-- saliente adicional, incluso si un rol V4 aparece solo como otorgante, aborta
-- todo. La comparacion se hace por OID bajo los bloqueos anteriores.
DO $prevalidacion$
DECLARE
    esperado record;
    oid_dba oid;
    oid_propietario_base oid;
    oid_otorgante_bootstrap oid;
    oid_propietario oid;
    oid_migrador oid;
    oid_emisor oid;
    oid_ejecutor oid;
    oids_v4 oid[];
    oid_esquema_guardia oid;
    oid_funcion_guardia oid;
    oid_esquema_public oid;
    oid_funcion_hmac oid;
    etiquetas_esperadas constant text[] := ARRAY[
        'CREATE TABLE',
        'CREATE TABLE AS',
        'CREATE FOREIGN TABLE',
        'CREATE VIEW',
        'CREATE MATERIALIZED VIEW',
        'CREATE TYPE',
        'CREATE DOMAIN',
        'ALTER TABLE',
        'ALTER VIEW',
        'ALTER MATERIALIZED VIEW',
        'ALTER TYPE',
        'ALTER DOMAIN'
    ];
    numero_membresias bigint;
    inventario_membresias text[];
BEGIN
    IF pg_catalog.to_regnamespace('vec_ejecucion_documental_v4') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el esquema V4 sigue instalado';
    END IF;

    SELECT oid
      INTO oid_dba
      FROM pg_catalog.pg_authid
     WHERE rolname = current_user
       AND rolsuper IS TRUE;
    SELECT datdba
      INTO oid_propietario_base
      FROM pg_catalog.pg_database
     WHERE datname = current_database();
    SELECT oid
      INTO oid_otorgante_bootstrap
      FROM pg_catalog.pg_authid
     WHERE oid = 10
       AND rolsuper IS TRUE;
    SELECT oid
      INTO oid_esquema_guardia
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_ejecucion_documental_v4_guardia';
    SELECT oid
      INTO oid_funcion_guardia
      FROM pg_catalog.pg_proc
     WHERE oid = pg_catalog.to_regprocedure(
         'vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos()'
     );
    SELECT oid
      INTO oid_esquema_public
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'public';
    SELECT oid
      INTO oid_funcion_hmac
      FROM pg_catalog.pg_proc
     WHERE oid = pg_catalog.to_regprocedure(
         'public.hmac(bytea,bytea,text)'
     );
    IF oid_dba IS NULL OR oid_propietario_base IS NULL
       OR oid_otorgante_bootstrap IS NULL
       OR oid_esquema_guardia IS NULL OR oid_funcion_guardia IS NULL
       OR oid_esquema_public IS NULL OR oid_funcion_hmac IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta autoridad u objeto gobernado V4';
    END IF;

    -- La retirada corresponde al mismo principal DBA que creo la guarda. Los
    -- nombres y firmas no bastan: una adopcion o cambio de propietario aborta.
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = oid_funcion_guardia
         WHERE espacio.oid = oid_esquema_guardia
           AND espacio.nspowner = oid_dba
           AND funcion.pronamespace = espacio.oid
           AND funcion.proowner = oid_dba
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambio el propietario de la guarda DDL V4';
    END IF;

    -- Al eliminar la guarda desaparecerian tambien sus ACL. Solo se admite la
    -- ACL nativa exacta del DBA sobre esquema y funcion; nada se borra por
    -- sorpresa como efecto lateral de DROP.
    IF (SELECT count(*)
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.oid = oid_esquema_guardia) <> 2
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.oid = oid_esquema_guardia
           AND NOT (
               privilegio.grantor = oid_dba
               AND privilegio.grantee = oid_dba
               AND privilegio.privilege_type IN ('USAGE', 'CREATE')
               AND privilegio.is_grantable IS FALSE
           )
       ) OR (SELECT count(*)
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE funcion.oid = oid_funcion_guardia) <> 1
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE funcion.oid = oid_funcion_guardia
           AND NOT (
               privilegio.grantor = oid_dba
               AND privilegio.grantee = oid_dba
               AND privilegio.privilege_type = 'EXECUTE'
               AND privilegio.is_grantable IS FALSE
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron las ACL de la guarda DDL V4';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger AS disparador
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = disparador.evtfoid
          JOIN pg_catalog.pg_language AS lenguaje
            ON lenguaje.oid = funcion.prolang
         WHERE disparador.evtname =
                   'vec_ejecucion_documental_v4_cerrar_acl_tipos'
           AND disparador.evtevent = 'ddl_command_end'
           AND disparador.evtenabled = 'O'
           AND disparador.evttags @> etiquetas_esperadas
           AND disparador.evttags <@ etiquetas_esperadas
           AND cardinality(disparador.evttags) =
               cardinality(etiquetas_esperadas)
           AND funcion.oid = oid_funcion_guardia
           AND funcion.prosecdef
           AND funcion.prorettype = 'pg_catalog.event_trigger'::regtype
           AND funcion.pronargs = 0
           AND funcion.pronargdefaults = 0
           AND funcion.prokind = 'f'
           AND funcion.provolatile = 'v'
           AND funcion.proparallel = 'u'
           AND funcion.proleakproof IS FALSE
           AND funcion.proisstrict IS FALSE
           AND lenguaje.lanname = 'plpgsql'
           AND funcion.proconfig = ARRAY[
               'search_path=pg_catalog, pg_temp'
           ]::text[]
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL V4 no es la esperada';
    END IF;

    -- Inventario cerrado: una sola funcion contenida en el esquema y un solo
    -- uso como event trigger. La dependencia de espacio por OID evita que un
    -- tipo u otra clase de objeto no enumerada se borre tras una prevalidacion
    -- nominal incompleta.
    IF (SELECT count(*)
          FROM pg_catalog.pg_event_trigger
         WHERE evtfoid = oid_funcion_guardia) <> 1
       OR (SELECT count(*)
             FROM pg_catalog.pg_proc
            WHERE pronamespace = oid_esquema_guardia) <> 1
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class
         WHERE relnamespace = oid_esquema_guardia
       ) OR (SELECT count(*)
          FROM pg_catalog.pg_depend
         WHERE refclassid = 'pg_catalog.pg_namespace'::regclass
           AND refobjid = oid_esquema_guardia
           AND deptype = 'n') <> 1
       OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_depend
         WHERE refclassid = 'pg_catalog.pg_namespace'::regclass
           AND refobjid = oid_esquema_guardia
           AND classid = 'pg_catalog.pg_proc'::regclass
           AND objid = oid_funcion_guardia
           AND objsubid = 0
           AND deptype = 'n'
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL V4 contiene objetos o usos inesperados';
    END IF;

    FOR esperado IN
        SELECT *
          FROM (VALUES
              ('vec_ejecucion_documental_v4_ejecutor_atestado'::text, true),
              ('vec_ejecucion_documental_v4_emisor_capacidad'::text, true),
              ('vec_ejecucion_documental_v4_migrador'::text, false),
              ('vec_ejecucion_documental_v4_propietario'::text, false)
          ) AS opciones(rol, hereda)
    LOOP
        IF NOT EXISTS (
            SELECT 1
              FROM pg_catalog.pg_authid
             WHERE rolname = esperado.rol
               AND rolcanlogin IS FALSE
               AND rolsuper IS FALSE
               AND rolcreatedb IS FALSE
               AND rolcreaterole IS FALSE
               AND rolinherit IS NOT DISTINCT FROM esperado.hereda
               AND rolreplication IS FALSE
               AND rolbypassrls IS FALSE
               AND rolconnlimit = -1
               AND rolpassword IS NULL
               AND rolvaliduntil IS NULL
               AND NOT EXISTS (
                   SELECT 1
                     FROM pg_catalog.pg_db_role_setting AS ajuste
                    WHERE ajuste.setrole = pg_authid.oid
               )
        ) THEN
            RAISE EXCEPTION USING ERRCODE = '55000',
                MESSAGE = 'down rechazado: falta un rol V4 o cambiaron sus opciones',
                DETAIL = esperado.rol;
        END IF;
    END LOOP;

    SELECT oid INTO STRICT oid_propietario
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_ejecucion_documental_v4_propietario';
    SELECT oid INTO STRICT oid_migrador
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_ejecucion_documental_v4_migrador';
    SELECT oid INTO STRICT oid_emisor
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_ejecucion_documental_v4_emisor_capacidad';
    SELECT oid INTO STRICT oid_ejecutor
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_ejecucion_documental_v4_ejecutor_atestado';
    oids_v4 := ARRAY[
        oid_propietario,
        oid_migrador,
        oid_emisor,
        oid_ejecutor
    ];

    -- REVOKE ALL sobre la base solo puede retirar las cinco concesiones que
    -- creo roles_up. Se inventarian todas las bases del cluster para no
    -- aceptar una concesion V4 adicional que el comando local no gobernaria.
    IF (SELECT count(*)
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)) <> 5
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS privilegio
         WHERE (
             privilegio.grantee = ANY(oids_v4)
             OR privilegio.grantor = ANY(oids_v4)
         )
           AND NOT (
               base.datname = current_database()
               AND privilegio.grantor = oid_propietario_base
               AND privilegio.is_grantable IS FALSE
               AND (
                   privilegio.grantee = oid_propietario
                   AND privilegio.privilege_type IN ('CONNECT', 'CREATE')
                   OR privilegio.grantee IN (
                       oid_migrador, oid_emisor, oid_ejecutor
                   ) AND privilegio.privilege_type = 'CONNECT'
               )
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron las ACL de base V4';
    END IF;

    -- Los dos REVOKE de objetos externos tambien son exactos. No se acepta
    -- otra ACL de esquema o funcion para ningun rol V4: DROP ROLE no debe
    -- limpiar una autoridad que este fichero no aprovisiono.
    IF (SELECT count(*)
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)) <> 1
       OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_namespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  espacio.nspacl,
                  pg_catalog.acldefault('n', espacio.nspowner)
              )
          ) AS privilegio
         WHERE espacio.oid = oid_esquema_public
           AND privilegio.grantor = espacio.nspowner
           AND privilegio.grantee = oid_propietario
           AND privilegio.privilege_type = 'USAGE'
           AND privilegio.is_grantable IS FALSE
       ) OR (SELECT count(*)
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)) <> 1
       OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS funcion
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS privilegio
         WHERE funcion.oid = oid_funcion_hmac
           AND privilegio.grantor = funcion.proowner
           AND privilegio.grantee = oid_propietario
           AND privilegio.privilege_type = 'EXECUTE'
           AND privilegio.is_grantable IS FALSE
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron los privilegios externos V4';
    END IF;

    -- No se toleran ACL V4 residuales de tablas, columnas, tipos, lenguajes,
    -- objetos grandes, FDW, servidores, parametros o tablespaces. Son clases
    -- que DROP ROLE podria intentar limpiar o rechazar fuera del contrato.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS clase
          CROSS JOIN LATERAL pg_catalog.aclexplode(clase.relacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_attribute AS atributo
          CROSS JOIN LATERAL pg_catalog.aclexplode(atributo.attacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS tipo
          CROSS JOIN LATERAL pg_catalog.aclexplode(tipo.typacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_language AS lenguaje
          CROSS JOIN LATERAL pg_catalog.aclexplode(lenguaje.lanacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_largeobject_metadata AS objeto
          CROSS JOIN LATERAL pg_catalog.aclexplode(objeto.lomacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_data_wrapper AS envoltorio
          CROSS JOIN LATERAL pg_catalog.aclexplode(envoltorio.fdwacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_foreign_server AS servidor
          CROSS JOIN LATERAL pg_catalog.aclexplode(servidor.srvacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_parameter_acl AS parametro
          CROSS JOIN LATERAL pg_catalog.aclexplode(parametro.paracl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_tablespace AS espacio
          CROSS JOIN LATERAL pg_catalog.aclexplode(espacio.spcacl) AS privilegio
         WHERE privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe una ACL V4 fuera del contrato';
    END IF;

    -- El down altera solo los defaults globales f/T del propietario. Antes de
    -- la primera migracion aun no existe ninguna fila; despues deben existir
    -- exactamente esas dos. Cada fila contiene solo el privilegio nativo de
    -- su dueño: una fila o beneficiario adicional se perderia al borrar el rol.
    IF (SELECT count(*)
          FROM pg_catalog.pg_default_acl
         WHERE defaclrole = ANY(oids_v4)) NOT IN (0, 2)
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl
         WHERE defaclrole = ANY(oids_v4)
           AND NOT (
               defaclrole = oid_propietario
               AND defaclnamespace = 0
               AND defaclobjtype IN ('f', 'T')
           )
       ) OR (SELECT count(*)
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl)
              AS privilegio
         WHERE defecto.defaclrole = ANY(oids_v4)
            OR privilegio.grantee = ANY(oids_v4)
            OR privilegio.grantor = ANY(oids_v4)) <> (
        SELECT count(*)
          FROM pg_catalog.pg_default_acl
         WHERE defaclrole = ANY(oids_v4)
       )
       OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl)
              AS privilegio
         WHERE (
             defecto.defaclrole = ANY(oids_v4)
             OR privilegio.grantee = ANY(oids_v4)
             OR privilegio.grantor = ANY(oids_v4)
         )
           AND NOT (
               defecto.defaclrole = oid_propietario
               AND defecto.defaclnamespace = 0
               AND privilegio.grantor = oid_propietario
               AND privilegio.grantee = oid_propietario
               AND privilegio.is_grantable IS FALSE
               AND (
                   defecto.defaclobjtype = 'f'
                   AND privilegio.privilege_type = 'EXECUTE'
                   OR defecto.defaclobjtype = 'T'
                   AND privilegio.privilege_type = 'USAGE'
               )
           )
       ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron los privilegios por defecto V4';
    END IF;

    -- Tambien se inventaria grantor: una arista cuyos extremos fueran ajenos
    -- seguiria dependiendo de un rol V4 y DROP ROLE podria retirarla de forma
    -- implicita. Los nombres solo ayudan al diagnostico; decide cada OID.
    SELECT count(*),
           array_agg(
               pg_catalog.format(
                   'roleid=%s(%s),member=%s(%s),grantor=%s(%s),admin=%s,inherit=%s,set=%s',
                   membresia.roleid,
                   COALESCE(grupo.rolname::text, '?'),
                   membresia.member,
                   COALESCE(miembro.rolname::text, '?'),
                   membresia.grantor,
                   COALESCE(otorgante.rolname::text, '?'),
                   membresia.admin_option,
                   membresia.inherit_option,
                   membresia.set_option
               ) ORDER BY membresia.roleid,
                          membresia.member,
                          membresia.grantor
           )
      INTO numero_membresias, inventario_membresias
      FROM pg_catalog.pg_auth_members AS membresia
      LEFT JOIN pg_catalog.pg_authid AS grupo
        ON grupo.oid = membresia.roleid
      LEFT JOIN pg_catalog.pg_authid AS miembro
        ON miembro.oid = membresia.member
      LEFT JOIN pg_catalog.pg_authid AS otorgante
        ON otorgante.oid = membresia.grantor
     WHERE membresia.roleid = ANY(oids_v4)
        OR membresia.member = ANY(oids_v4)
        OR membresia.grantor = ANY(oids_v4);

    IF numero_membresias <> 1 OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_auth_members AS membresia
         WHERE membresia.roleid = oid_propietario
           AND membresia.member = oid_migrador
           AND membresia.grantor = oid_otorgante_bootstrap
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la membresia estructural V4 no es unica y exacta',
            DETAIL = COALESCE(
                array_to_string(inventario_membresias, E'\n'),
                'sin membresias V4'
            );
    END IF;
END
$prevalidacion$;

DROP EVENT TRIGGER vec_ejecucion_documental_v4_cerrar_acl_tipos;
DROP FUNCTION vec_ejecucion_documental_v4_guardia.cerrar_acl_tipos() RESTRICT;
DROP SCHEMA vec_ejecucion_documental_v4_guardia RESTRICT;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_emisor_capacidad',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_ejecutor_atestado',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_migrador',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_ejecucion_documental_v4_propietario',
        current_database()
    );
END
$privilegios_base$;

REVOKE EXECUTE ON FUNCTION public.hmac(bytea, bytea, text)
    FROM vec_ejecucion_documental_v4_propietario;
REVOKE USAGE ON SCHEMA public
    FROM vec_ejecucion_documental_v4_propietario;

REVOKE vec_ejecucion_documental_v4_propietario
    FROM vec_ejecucion_documental_v4_migrador;

-- La migracion ascendente cierra globalmente los defaults que PostgreSQL abre
-- para funciones y tipos. Al desaparecer el rol ya no puede crear objetos;
-- se revierten solo esas dos entradas para poder eliminar su pg_default_acl.
-- Esto no concede nada sobre objetos existentes ni reabre pgcrypto.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_ejecucion_documental_v4_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_ejecucion_documental_v4_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP ROLE vec_ejecucion_documental_v4_ejecutor_atestado;
DROP ROLE vec_ejecucion_documental_v4_emisor_capacidad;
DROP ROLE vec_ejecucion_documental_v4_migrador;
DROP ROLE vec_ejecucion_documental_v4_propietario;
COMMIT;
