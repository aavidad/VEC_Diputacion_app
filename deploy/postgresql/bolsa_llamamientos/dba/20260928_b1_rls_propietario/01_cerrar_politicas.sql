\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_llamamientos:dba:b1-rls-propietario:20260928', 0)
);

-- Solo el cambio de TO es admisible. La guarda compara la expresión SQL
-- deparseada por PostgreSQL 18.4 con la preimagen literal creada por B1.
-- El rol propietario no recibe TEMP: no se crean tablas ni privilegios.
DO $cerrar$
DECLARE
    tablas constant text[] := ARRAY[
        'bolsa_autoritativa', 'necesidad_autoritativa', 'necesidad_actual',
        'politica_autoritativa', 'instantanea_autoritativa',
        'evaluacion_autoritativa', 'atestacion_autorizacion_version',
        'atestacion_autorizacion_actual', 'propuesta', 'referencia_consumida',
        'uso_decision', 'auditoria', 'auditoria_actual', 'outbox'
    ];
    predicado constant text :=
        '(CURRENT_USER = ''vec_bolsa_llamamientos_propietario''::name)';
    propietario oid := 'vec_bolsa_llamamientos_propietario'::regrole::oid;
    nombre_tabla text;
    clase record;
    politica record;
    oids oid[] := ARRAY[]::oid[];
    usos text[] := ARRAY[]::text[];
    checks text[] := ARRAY[]::text[];
    acls text[] := ARRAY[]::text[];
    acls_columnas text[] := ARRAY[]::text[];
    acl_columnas text;
    i integer;
BEGIN
    IF current_user <> 'vec_bolsa_llamamientos_propietario'
       OR cardinality(tablas) <> 14
       OR (SELECT count(DISTINCT nombre) FROM unnest(tablas) nombre) <> 14
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles
            WHERE oid = propietario AND NOT rolcanlogin AND NOT rolinherit
              AND NOT rolsuper AND NOT rolbypassrls
       ) THEN
        RAISE EXCEPTION 'B1 RLS: rol o inventario incompatible' USING ERRCODE = '55000';
    END IF;

    -- Bloqueos en orden fijo: ningún DDL puede cambiar la preimagen mientras
    -- la transacción comprueba y ajusta las catorce políticas.
    FOREACH nombre_tabla IN ARRAY tablas LOOP
        EXECUTE format(
            'LOCK TABLE vec_bolsa_llamamientos.%I IN ACCESS EXCLUSIVE MODE',
            nombre_tabla
        );
    END LOOP;

    FOR i IN 1..14 LOOP
        nombre_tabla := tablas[i];
        SELECT c.oid, c.relowner, c.relkind, c.relrowsecurity,
               c.relforcerowsecurity, coalesce(c.relacl::text, '<NULL>') AS acl
          INTO STRICT clase
          FROM pg_catalog.pg_class c
          JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_bolsa_llamamientos' AND c.relname = nombre_tabla;
        IF clase.relowner <> propietario OR clase.relkind <> 'r'
           OR NOT clase.relrowsecurity OR NOT clase.relforcerowsecurity
           OR EXISTS (
               SELECT 1
                 FROM pg_catalog.aclexplode(
                     coalesce(
                         (SELECT relacl FROM pg_catalog.pg_class WHERE oid = clase.oid),
                         pg_catalog.acldefault('r', propietario)
                     )
                 ) a
                WHERE a.grantee <> propietario
           )
           OR EXISTS (
               SELECT 1
                 FROM pg_catalog.pg_attribute a
                 CROSS JOIN LATERAL pg_catalog.aclexplode(a.attacl) permiso
                WHERE a.attrelid = clase.oid AND a.attnum > 0
                  AND NOT a.attisdropped AND permiso.grantee <> propietario
           ) THEN
            RAISE EXCEPTION 'B1 RLS: tabla o ACL incompatible: %', nombre_tabla
                USING ERRCODE = '55000';
        END IF;
        IF (SELECT count(*) FROM pg_catalog.pg_policy WHERE polrelid = clase.oid) <> 1 THEN
            RAISE EXCEPTION 'B1 RLS: políticas adicionales o ausentes: %', nombre_tabla
                USING ERRCODE = '55000';
        END IF;
        SELECT p.oid, p.polname, p.polcmd, p.polpermissive, p.polroles,
               pg_catalog.pg_get_expr(p.polqual, p.polrelid) AS uso,
               pg_catalog.pg_get_expr(p.polwithcheck, p.polrelid) AS check_expr
          INTO STRICT politica
          FROM pg_catalog.pg_policy p
         WHERE p.polrelid = clase.oid;
        IF politica.polname <> 'solo_propietario' OR politica.polcmd <> '*'
           OR NOT politica.polpermissive
           OR politica.polroles <> ARRAY[0]::oid[]
           OR politica.uso IS DISTINCT FROM predicado
           OR politica.check_expr IS DISTINCT FROM predicado THEN
            RAISE EXCEPTION 'B1 RLS: política preimagen incompatible: %', nombre_tabla
                USING ERRCODE = '55000';
        END IF;
        oids := array_append(oids, politica.oid);
        usos := array_append(usos, politica.uso);
        checks := array_append(checks, politica.check_expr);
        acls := array_append(acls, clase.acl);
        SELECT coalesce(string_agg(
                   a.attnum::text || ':' || coalesce(a.attacl::text, '<NULL>'),
                   '|' ORDER BY a.attnum
               ), '') INTO acl_columnas
          FROM pg_catalog.pg_attribute a
         WHERE a.attrelid = clase.oid AND a.attnum > 0 AND NOT a.attisdropped;
        acls_columnas := array_append(acls_columnas, acl_columnas);
    END LOOP;

    FOREACH nombre_tabla IN ARRAY tablas LOOP
        EXECUTE format(
            'ALTER POLICY solo_propietario ON vec_bolsa_llamamientos.%I TO vec_bolsa_llamamientos_propietario',
            nombre_tabla
        );
    END LOOP;

    FOR i IN 1..14 LOOP
        nombre_tabla := tablas[i];
        SELECT c.oid, c.relowner, c.relrowsecurity, c.relforcerowsecurity,
               coalesce(c.relacl::text, '<NULL>') AS acl
          INTO STRICT clase
          FROM pg_catalog.pg_class c
          JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'vec_bolsa_llamamientos' AND c.relname = nombre_tabla;
        SELECT p.oid, p.polroles,
               pg_catalog.pg_get_expr(p.polqual, p.polrelid) AS uso,
               pg_catalog.pg_get_expr(p.polwithcheck, p.polrelid) AS check_expr
          INTO STRICT politica
          FROM pg_catalog.pg_policy p
         WHERE p.polrelid = clase.oid AND p.polname = 'solo_propietario';
        SELECT coalesce(string_agg(
                   a.attnum::text || ':' || coalesce(a.attacl::text, '<NULL>'),
                   '|' ORDER BY a.attnum
               ), '') INTO acl_columnas
          FROM pg_catalog.pg_attribute a
         WHERE a.attrelid = clase.oid AND a.attnum > 0 AND NOT a.attisdropped;
        IF clase.relowner <> propietario OR NOT clase.relrowsecurity
           OR NOT clase.relforcerowsecurity OR clase.acl <> acls[i]
           OR acl_columnas <> acls_columnas[i]
           OR politica.oid <> oids[i]
           OR politica.polroles <> ARRAY[propietario]::oid[]
           OR politica.uso <> usos[i]
           OR politica.check_expr <> checks[i]
           OR (SELECT count(*) FROM pg_catalog.pg_policy WHERE polrelid = clase.oid) <> 1 THEN
            RAISE EXCEPTION 'B1 RLS: postimagen incompatible: %', nombre_tabla
                USING ERRCODE = '55000';
        END IF;
    END LOOP;
END
$cerrar$;
COMMIT;
