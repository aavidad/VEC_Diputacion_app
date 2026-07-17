-- Reversion DBA. Falla cerrada si quedan objetos, opciones o membresias ajenas.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_baremacion:roles_down:v1', 0)
);

-- Los roles y sus membresias son catalogo compartido por todo el cluster. Un
-- GRANT iniciado desde otra base podria resolver los OID durante el down y
-- escribir la arista despues de DROP ROLE. El orden es deliberado: primero se
-- impide resolver o retirar roles, despues se inmovilizan todas las aristas y
-- finalmente las ACL de base y predeterminadas que el down va a retirar.
-- Requiere superusuario y una ventana de mantenimiento sin administracion
-- concurrente; los locks se conservan hasta el COMMIT.
LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_database IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_default_acl IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    esperado record;
    enlaces_inesperados text[];
    oid_dba oid;
    oid_otorgante_bootstrap oid;
    oid_propietario_base oid;
    oid_propietario oid;
    oid_migrador oid;
    oids_bolsa oid[];
    oid_esquema_guardia oid;
    oid_funcion_guardia oid;
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
BEGIN
    IF pg_catalog.to_regnamespace('vec_bolsa_baremacion') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el esquema Bolsa sigue instalado';
    END IF;
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion.revalidar_decision_bolsa_baremacion_v1(jsonb,bytea,bytea,text,text,text,jsonb,timestamp with time zone)'
       ) IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la frontera de autorizacion Bolsa sigue instalada';
    END IF;

    SELECT oid INTO oid_dba
      FROM pg_catalog.pg_authid
     WHERE rolname = current_user
       AND rolsuper IS TRUE;
    SELECT oid INTO oid_otorgante_bootstrap
      FROM pg_catalog.pg_authid
     WHERE oid = 10
       AND rolsuper IS TRUE;
    SELECT oid INTO oid_esquema_guardia
      FROM pg_catalog.pg_namespace
     WHERE nspname = 'vec_bolsa_baremacion_guardia';
    SELECT oid INTO oid_funcion_guardia
      FROM pg_catalog.pg_proc
     WHERE oid = pg_catalog.to_regprocedure(
         'vec_bolsa_baremacion_guardia.cerrar_acl_tipos()'
     );
    IF oid_dba IS NULL OR oid_otorgante_bootstrap IS NULL
       OR oid_esquema_guardia IS NULL
       OR oid_funcion_guardia IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: falta la guarda DDL Bolsa';
    END IF;

    -- La retirada se ejecuta por el mismo principal DBA que creo la guarda.
    -- Se rechazan adopciones o cambios de propietario, aunque los dos objetos
    -- sigan teniendo nombres y firmas validos.
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
            MESSAGE = 'down rechazado: cambio el propietario de la guarda DDL Bolsa';
    END IF;

    -- ACL exacta: el DBA conserva USAGE/CREATE sobre el esquema y EXECUTE
    -- sobre la funcion; ningun otro principal, incluido PUBLIC, recibe nada.
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
            MESSAGE = 'down rechazado: cambiaron las ACL de la guarda DDL Bolsa';
    END IF;

    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger AS disparador
          JOIN pg_catalog.pg_proc AS funcion
            ON funcion.oid = disparador.evtfoid
          JOIN pg_catalog.pg_language AS lenguaje
            ON lenguaje.oid = funcion.prolang
         WHERE disparador.evtname =
                   'vec_bolsa_baremacion_cerrar_acl_tipos'
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
            MESSAGE = 'down rechazado: la guarda DDL Bolsa no es la esperada';
    END IF;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_event_trigger
         WHERE evtfoid = oid_funcion_guardia
           AND evtname <> 'vec_bolsa_baremacion_cerrar_acl_tipos'
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc
         WHERE pronamespace = oid_esquema_guardia
           AND oid <> oid_funcion_guardia
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class
         WHERE relnamespace = oid_esquema_guardia
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: la guarda DDL Bolsa contiene objetos inesperados';
    END IF;

    FOR esperado IN
        SELECT *
          FROM (VALUES
              ('vec_bolsa_baremacion_propietario'::text, false),
              ('vec_bolsa_baremacion_migrador'::text, false),
              ('vec_bolsa_baremacion_ejecutor'::text, true),
              ('vec_bolsa_baremacion_lector_outbox'::text, true),
              ('vec_bolsa_baremacion_registrador_atestacion'::text, true)
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
                MESSAGE = 'down rechazado: falta un rol Bolsa o sus opciones cambiaron',
                DETAIL = esperado.rol;
        END IF;
    END LOOP;

    SELECT oid
      INTO oid_propietario
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_bolsa_baremacion_propietario';
    SELECT oid
      INTO oid_migrador
      FROM pg_catalog.pg_authid
     WHERE rolname = 'vec_bolsa_baremacion_migrador';
    SELECT pg_catalog.array_agg(oid ORDER BY oid)
      INTO oids_bolsa
      FROM pg_catalog.pg_authid
     WHERE rolname = ANY (ARRAY[
         'vec_bolsa_baremacion_propietario',
         'vec_bolsa_baremacion_migrador',
         'vec_bolsa_baremacion_ejecutor',
         'vec_bolsa_baremacion_lector_outbox',
         'vec_bolsa_baremacion_registrador_atestacion'
     ]);
    IF oid_propietario IS NULL OR oid_migrador IS NULL
       OR cardinality(oids_bolsa) <> 5 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: no se pudo fijar el inventario OID Bolsa';
    END IF;

    SELECT datdba
      INTO oid_propietario_base
      FROM pg_catalog.pg_database
     WHERE datname = current_database();
    IF oid_propietario_base IS NULL
       OR oid_propietario_base = ANY (oids_bolsa) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el propietario de la base Bolsa no es gobernable';
    END IF;

    -- Solo se retiraran las cinco concesiones creadas por roles_up. El
    -- otorgante de una ACL de base es su propietario, incluso cuando el GRANT
    -- lo ejecuta otro superusuario. Cualquier privilegio o grant option extra
    -- se conserva haciendo abortar la retirada antes del primer REVOKE.
    IF (
        SELECT count(*)
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS privilegio
         WHERE base.datname = current_database()
           AND (
               privilegio.grantee = ANY (oids_bolsa)
               OR privilegio.grantor = ANY (oids_bolsa)
           )
    ) <> 5 OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_database AS base
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              COALESCE(
                  base.datacl,
                  pg_catalog.acldefault('d', base.datdba)
              )
          ) AS privilegio
         WHERE base.datname = current_database()
           AND (
               privilegio.grantee = ANY (oids_bolsa)
               OR privilegio.grantor = ANY (oids_bolsa)
           )
           AND NOT (
               privilegio.grantor = oid_propietario_base
               AND privilegio.is_grantable IS FALSE
               AND (
                   (
                       privilegio.grantee = oid_propietario
                       AND privilegio.privilege_type IN ('CONNECT', 'CREATE')
                   ) OR (
                       privilegio.grantee = ANY (ARRAY[
                           oid_migrador,
                           (
                               SELECT oid
                                 FROM pg_catalog.pg_authid
                                WHERE rolname = 'vec_bolsa_baremacion_ejecutor'
                           ),
                           (
                               SELECT oid
                                 FROM pg_catalog.pg_authid
                                WHERE rolname = 'vec_bolsa_baremacion_lector_outbox'
                           )
                       ]::oid[])
                       AND privilegio.privilege_type = 'CONNECT'
                   )
               )
           )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron las ACL de base Bolsa';
    END IF;

    -- Tras retirar el esquema solo deben quedar los dos defaults globales que
    -- 000001 cerro: funciones y tipos. Se inspeccionan tambien grantor y
    -- grantee para detectar dependencias creadas desde otro rol.
    IF (
        SELECT count(*)
          FROM pg_catalog.pg_default_acl
         WHERE defaclrole = ANY (oids_bolsa)
    ) NOT IN (0, 2) OR (
        SELECT count(*)
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl) AS privilegio
         WHERE defecto.defaclrole = ANY (oids_bolsa)
            OR privilegio.grantee = ANY (oids_bolsa)
            OR privilegio.grantor = ANY (oids_bolsa)
    ) <> (
        SELECT count(*)
          FROM pg_catalog.pg_default_acl
         WHERE defaclrole = ANY (oids_bolsa)
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_default_acl AS defecto
          CROSS JOIN LATERAL pg_catalog.aclexplode(defecto.defaclacl) AS privilegio
         WHERE (
             defecto.defaclrole = ANY (oids_bolsa)
             OR privilegio.grantee = ANY (oids_bolsa)
             OR privilegio.grantor = ANY (oids_bolsa)
         ) AND NOT (
             defecto.defaclrole = oid_propietario
             AND defecto.defaclnamespace = 0
             AND defecto.defaclobjtype IN ('f', 'T')
             AND privilegio.grantor = oid_propietario
             AND privilegio.grantee = oid_propietario
             AND privilegio.is_grantable IS FALSE
             AND (
                 (defecto.defaclobjtype = 'f' AND privilegio.privilege_type = 'EXECUTE')
                 OR (defecto.defaclobjtype = 'T' AND privilegio.privilege_type = 'USAGE')
             )
         )
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: cambiaron los privilegios predeterminados Bolsa';
    END IF;

    -- La unica arista gobernada fue creada por roles_up. PostgreSQL atribuye
    -- toda concesion emitida por un superusuario al bootstrap del cluster,
    -- aunque otro DBA sea el propietario de la guarda y ejecute este down.
    IF NOT EXISTS (
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
            MESSAGE = 'down rechazado: falta la membresia estructural Bolsa exacta';
    END IF;

    -- El inventario usa los OID ya inmovilizados e inspecciona las tres
    -- coordenadas. Asi tambien se rechaza que un rol Bolsa aparezca como
    -- otorgante de una relacion cuyos extremos sean ajenos. Los LEFT JOIN no
    -- ocultan una arista previamente corrupta con un OID sin principal.
    SELECT pg_catalog.array_agg(
               pg_catalog.format(
                   '%s->%s; otorgante=%s; admin=%s; inherit=%s; set=%s',
                   COALESCE(
                       miembro.rolname,
                       '<oid:' || membresia.member::text || '>'
                   ),
                   COALESCE(
                       grupo.rolname,
                       '<oid:' || membresia.roleid::text || '>'
                   ),
                   COALESCE(
                       otorgante.rolname,
                       '<oid:' || membresia.grantor::text || '>'
                   ),
                   membresia.admin_option,
                   membresia.inherit_option,
                   membresia.set_option
               )
               ORDER BY membresia.roleid, membresia.member, membresia.grantor
           )
      INTO enlaces_inesperados
      FROM pg_catalog.pg_auth_members AS membresia
      LEFT JOIN pg_catalog.pg_authid AS miembro
        ON miembro.oid = membresia.member
      LEFT JOIN pg_catalog.pg_authid AS grupo
        ON grupo.oid = membresia.roleid
      LEFT JOIN pg_catalog.pg_authid AS otorgante
        ON otorgante.oid = membresia.grantor
     WHERE (
         membresia.roleid = ANY (oids_bolsa)
         OR membresia.member = ANY (oids_bolsa)
         OR membresia.grantor = ANY (oids_bolsa)
     )
       AND NOT (
           membresia.roleid = oid_propietario
           AND membresia.member = oid_migrador
           AND membresia.grantor = oid_otorgante_bootstrap
           AND membresia.admin_option IS FALSE
           AND membresia.inherit_option IS FALSE
           AND membresia.set_option IS TRUE
       );
    IF cardinality(enlaces_inesperados) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: existe una membresia Bolsa inesperada',
            DETAIL = array_to_string(enlaces_inesperados, ',');
    END IF;

    -- Debe existir una sola fila relacionada con los cinco OID: la arista
    -- estructural exacta. Esta cuenta evita aceptar duplicados o variantes que
    -- una futura version de PostgreSQL pudiera representar por separado.
    IF (
        SELECT count(*)
          FROM pg_catalog.pg_auth_members AS membresia
         WHERE membresia.roleid = ANY (oids_bolsa)
            OR membresia.member = ANY (oids_bolsa)
            OR membresia.grantor = ANY (oids_bolsa)
    ) <> 1 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'down rechazado: el inventario de membresias Bolsa no es exacto';
    END IF;
END
$prevalidacion$;

DROP EVENT TRIGGER vec_bolsa_baremacion_cerrar_acl_tipos;
DROP FUNCTION vec_bolsa_baremacion_guardia.cerrar_acl_tipos() RESTRICT;
DROP SCHEMA vec_bolsa_baremacion_guardia RESTRICT;

DO $revocar_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_bolsa_baremacion_propietario',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_bolsa_baremacion_migrador, vec_bolsa_baremacion_ejecutor, vec_bolsa_baremacion_lector_outbox, vec_bolsa_baremacion_registrador_atestacion',
        current_database()
    );
END
$revocar_base$;

REVOKE vec_bolsa_baremacion_propietario
    FROM vec_bolsa_baremacion_migrador;

-- La migracion base cierra los defaults globales que PostgreSQL abre para
-- funciones y tipos. Al no poder crear ya objetos, se restauran solo esas dos
-- entradas para eliminar las dependencias pg_default_acl del rol. Esto no
-- concede privilegios sobre ningun objeto existente.
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_baremacion_propietario
    GRANT EXECUTE ON FUNCTIONS TO PUBLIC;
ALTER DEFAULT PRIVILEGES
    FOR ROLE vec_bolsa_baremacion_propietario
    GRANT USAGE ON TYPES TO PUBLIC;

DROP ROLE vec_bolsa_baremacion_registrador_atestacion;
DROP ROLE vec_bolsa_baremacion_lector_outbox;
DROP ROLE vec_bolsa_baremacion_ejecutor;
DROP ROLE vec_bolsa_baremacion_migrador;
DROP ROLE vec_bolsa_baremacion_propietario;
COMMIT;
