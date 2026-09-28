\set ON_ERROR_STOP on
-- Puerta de cadena: toda DDL posterior en estos dos esquemas debe cerrar su
-- tipo fila antes de que se considere apta para el selector corporativo RRHH.
DO $comprobar$
DECLARE abiertos text;
BEGIN
    SELECT string_agg(pg_catalog.format('%I.%I', n.nspname, t.typname), ', '
                      ORDER BY n.nspname, t.typname)
      INTO abiertos
      FROM pg_catalog.pg_type AS t
      JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
     WHERE n.nspname IN ('vec_bolsa_llamamientos', 'vec_autorizacion')
       AND t.typtype IN ('c', 'd', 'e', 'm', 'r')
       AND EXISTS (
           SELECT 1 FROM pg_catalog.aclexplode(coalesce(
               t.typacl, pg_catalog.acldefault('T', t.typowner))) AS a
            WHERE a.grantee = 0 AND a.privilege_type = 'USAGE'
       );
    IF abiertos IS NOT NULL THEN
        RAISE EXCEPTION 'C3 ACL tipos: USAGE PUBLIC recuperado: %', abiertos
            USING ERRCODE = '55000';
    END IF;
END $comprobar$;
