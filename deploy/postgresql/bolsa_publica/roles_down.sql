-- Reversion DBA conservadora. La cuenta LOGIN de aplicacion debe retirarse o
-- perder su membresia antes de desmontar los grupos tecnicos.
BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_bolsa_publica:roles_down:v1', 0)
);

DO $prevalidacion$
DECLARE
    miembros text[];
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper IS TRUE
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'reversion de roles de bolsa publica rechazada: requiere superusuario';
    END IF;
    SELECT array_agg(miembro.rolname::text ORDER BY miembro.rolname)
      INTO miembros
      FROM pg_catalog.pg_auth_members AS membresia
      JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = membresia.roleid
      JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = membresia.member
     WHERE grupo.rolname IN ('vec_bolsa_publica_consulta', 'vec_bolsa_publica_publicador');
    IF cardinality(miembros) > 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion rechazada: quedan identidades de consulta',
            DETAIL = array_to_string(miembros, ',');
    END IF;
    IF pg_catalog.to_regnamespace('vec_bolsa_publica_datos') IS NOT NULL
       OR pg_catalog.to_regnamespace('vec_bolsa_publica_lectura') IS NOT NULL
       OR pg_catalog.to_regnamespace('vec_bolsa_publica_publicacion') IS NOT NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reversion rechazada: quedan esquemas de bolsa publica';
    END IF;
END
$prevalidacion$;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL ON DATABASE %I FROM vec_bolsa_publica_consulta, vec_bolsa_publica_publicador, vec_bolsa_publica_migrador, vec_bolsa_publica_publicacion_propietario, vec_bolsa_publica_propietario',
        current_database()
    );
END
$privilegios_base$;

REVOKE vec_bolsa_publica_propietario,
       vec_bolsa_publica_publicacion_propietario
  FROM vec_bolsa_publica_migrador;
DROP ROLE vec_bolsa_publica_consulta;
DROP ROLE vec_bolsa_publica_publicador;
DROP ROLE vec_bolsa_publica_migrador;
DROP ROLE vec_bolsa_publica_publicacion_propietario;
DROP ROLE vec_bolsa_publica_propietario;

COMMIT;
