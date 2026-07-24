\set ON_ERROR_STOP on

SET search_path = pg_catalog;

DO $inventario_global$
DECLARE
    base_oid oid;
    roles_vec constant text[] := ARRAY[
        'vec_autorizacion_fuente',
        'vec_autorizacion_migrador',
        'vec_autorizacion_propietario',
        'vec_autorizacion_registro',
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion_lector_outbox',
        'vec_bolsa_baremacion_migrador',
        'vec_bolsa_baremacion_propietario',
        'vec_bolsa_baremacion_registrador_atestacion'
    ];
    rol text;
BEGIN
    SELECT oid
      INTO STRICT base_oid
      FROM pg_database
     WHERE datname = current_database();

    IF (SELECT count(*) FROM pg_roles WHERE rolname = ANY (roles_vec)) <> 9 THEN
        RAISE EXCEPTION
            'la restauracion no contiene exactamente los nueve roles VEC';
    END IF;

    IF EXISTS (
            SELECT 1
              FROM pg_roles
             WHERE rolname = ANY (roles_vec)
               AND (
                   rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole
                   OR rolreplication OR rolbypassrls
                   OR rolvaliduntil IS NOT NULL OR rolconfig IS NOT NULL
               )
       ) THEN
        RAISE EXCEPTION USING
            MESSAGE =
                'un rol restaurado contiene capacidad, credencial o ajuste inesperado',
            DETAIL = (
                SELECT jsonb_agg(jsonb_build_object(
                    'rol', rolname,
                    'login', rolcanlogin,
                    'superusuario', rolsuper,
                    'crea_bases', rolcreatedb,
                    'crea_roles', rolcreaterole,
                    'replicacion', rolreplication,
                    'omite_rls', rolbypassrls,
                    'caducidad', rolvaliduntil IS NOT NULL,
                    'ajustes', rolconfig IS NOT NULL
                ) ORDER BY rolname)::text
                  FROM pg_roles
                 WHERE rolname = ANY (roles_vec)
                   AND (
                       rolcanlogin OR rolsuper OR rolcreatedb OR rolcreaterole
                       OR rolreplication OR rolbypassrls
                       OR rolvaliduntil IS NOT NULL OR rolconfig IS NOT NULL
                   )
            );
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_authid
         WHERE rolname = ANY (roles_vec)
           AND rolpassword IS NOT NULL
    ) THEN
        RAISE EXCEPTION
            'un rol NOLOGIN restaurado contiene material de autenticacion';
    END IF;

    IF EXISTS (
            SELECT 1
              FROM pg_roles
             WHERE rolname IN (
                 'vec_autorizacion_propietario',
                 'vec_autorizacion_migrador',
                 'vec_bolsa_baremacion_propietario',
                 'vec_bolsa_baremacion_migrador'
             )
               AND rolinherit
       ) THEN
        RAISE EXCEPTION
            'un rol propietario o migrador recupero INHERIT';
    END IF;

    IF EXISTS (
            SELECT 1
              FROM pg_roles
             WHERE rolname = ANY (roles_vec)
               AND rolname NOT IN (
                   'vec_autorizacion_propietario',
                   'vec_autorizacion_migrador',
                   'vec_bolsa_baremacion_propietario',
                 'vec_bolsa_baremacion_migrador'
             )
               AND NOT rolinherit
       ) THEN
        RAISE EXCEPTION
            'un rol de ejecucion restaurado perdio INHERIT';
    END IF;

    IF (SELECT count(*)
          FROM pg_auth_members AS membresia
          JOIN pg_roles AS grupo ON grupo.oid = membresia.roleid
          JOIN pg_roles AS miembro ON miembro.oid = membresia.member
         WHERE grupo.rolname LIKE 'vec\_%' ESCAPE '\'
            OR miembro.rolname LIKE 'vec\_%' ESCAPE '\') <> 2
       OR NOT EXISTS (
            SELECT 1
              FROM pg_auth_members AS membresia
              JOIN pg_roles AS grupo ON grupo.oid = membresia.roleid
              JOIN pg_roles AS miembro ON miembro.oid = membresia.member
             WHERE grupo.rolname = 'vec_autorizacion_propietario'
               AND miembro.rolname = 'vec_autorizacion_migrador'
               AND NOT membresia.admin_option
               AND NOT membresia.inherit_option
               AND membresia.set_option
       )
       OR NOT EXISTS (
            SELECT 1
              FROM pg_auth_members AS membresia
              JOIN pg_roles AS grupo ON grupo.oid = membresia.roleid
              JOIN pg_roles AS miembro ON miembro.oid = membresia.member
             WHERE grupo.rolname = 'vec_bolsa_baremacion_propietario'
               AND miembro.rolname = 'vec_bolsa_baremacion_migrador'
               AND NOT membresia.admin_option
               AND NOT membresia.inherit_option
               AND membresia.set_option
       ) THEN
        RAISE EXCEPTION
            'las membresias restauradas no son las dos aristas esperadas';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_db_role_setting AS ajuste
          JOIN pg_roles AS identidad ON identidad.oid = ajuste.setrole
         WHERE identidad.rolname = ANY (roles_vec)
    ) OR EXISTS (
        SELECT 1
          FROM aclexplode(COALESCE(
              (SELECT datacl FROM pg_database WHERE oid = base_oid),
              acldefault(
                  'd',
                  (SELECT datdba FROM pg_database WHERE oid = base_oid)
              )
          )) AS privilegio
         WHERE privilegio.grantee = 0
    ) THEN
        RAISE EXCEPTION
            'la restauracion introdujo ajustes o privilegios PUBLIC';
    END IF;

    FOREACH rol IN ARRAY ARRAY[
        'vec_autorizacion_propietario',
        'vec_bolsa_baremacion_propietario'
    ] LOOP
        IF NOT has_database_privilege(rol, base_oid, 'CONNECT')
           OR NOT has_database_privilege(rol, base_oid, 'CREATE') THEN
            RAISE EXCEPTION
                'el propietario % no recupero CONNECT y CREATE', rol;
        END IF;
    END LOOP;

    FOREACH rol IN ARRAY ARRAY[
        'vec_autorizacion_migrador',
        'vec_autorizacion_fuente',
        'vec_autorizacion_registro',
        'vec_bolsa_baremacion_migrador',
        'vec_bolsa_baremacion_ejecutor',
        'vec_bolsa_baremacion_lector_outbox'
    ] LOOP
        IF NOT has_database_privilege(rol, base_oid, 'CONNECT')
           OR has_database_privilege(rol, base_oid, 'CREATE')
           OR has_database_privilege(rol, base_oid, 'TEMPORARY') THEN
            RAISE EXCEPTION
                'el rol % no recupero su ACL minima de base', rol;
        END IF;
    END LOOP;

    IF has_database_privilege(
           'vec_bolsa_baremacion_registrador_atestacion',
           base_oid,
           'CONNECT'
       ) THEN
        RAISE EXCEPTION
            'el registrador reservado recupero CONNECT indebidamente';
    END IF;
END
$inventario_global$;
