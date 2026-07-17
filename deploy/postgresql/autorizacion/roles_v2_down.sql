-- Ejecutar solo despues de 000003_proyeccion_motivos_autorizacion_v2.down.sql
-- y de retirar todas las membresias LOGIN. DROP ROLE falla cerrado ante
-- cualquier dependencia; no se usa CASCADE.
BEGIN;

SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:roles_motivos_v2:down:v1', 0)
);

-- PostgreSQL puede retirar aristas de membresia al eliminar un rol. Esa
-- comodidad no es aceptable en una retirada gobernada: ninguna relacion se
-- adopta como propia ni se borra implicitamente. Primero se bloquea pg_authid
-- para impedir que un GRANT concurrente resuelva y conserve OID obsoletos;
-- despues pg_auth_members serializa la arista hasta que DROP ROLE confirma.
LOCK TABLE pg_catalog.pg_authid IN ACCESS EXCLUSIVE MODE;
LOCK TABLE pg_catalog.pg_auth_members IN ACCESS EXCLUSIVE MODE;

DO $prevalidacion$
DECLARE
    roles_v2 oid[];
    membresias text[];
BEGIN
    SELECT ARRAY[
        'vec_autorizacion_motivos_proyector'::regrole::oid,
        'vec_autorizacion_motivos_evaluador'::regrole::oid
    ] INTO roles_v2;

    SELECT array_agg(
               grupo.rolname || '->' || miembro.rolname
               ORDER BY grupo.rolname, miembro.rolname
           )
      INTO membresias
      FROM pg_catalog.pg_auth_members AS relacion
      JOIN pg_catalog.pg_roles AS grupo ON grupo.oid = relacion.roleid
      JOIN pg_catalog.pg_roles AS miembro ON miembro.oid = relacion.member
     WHERE relacion.roleid = ANY(roles_v2)
        OR relacion.member = ANY(roles_v2);

    IF cardinality(membresias) > 0 THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'retirada V2 rechazada: existen membresias; no se eliminan implicitamente',
            DETAIL = array_to_string(membresias, ',');
    END IF;
END
$prevalidacion$;

DO $privilegios_base$
BEGIN
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_motivos_evaluador',
        current_database()
    );
    EXECUTE format(
        'REVOKE ALL PRIVILEGES ON DATABASE %I FROM vec_autorizacion_motivos_proyector',
        current_database()
    );
END
$privilegios_base$;

DROP ROLE vec_autorizacion_motivos_evaluador;
DROP ROLE vec_autorizacion_motivos_proyector;

COMMIT;
