-- Evidencia estructural ejecutada como propietario y revertida. La prueba
-- positiva de guardar_propuesta_v1 se mantiene prohibida hasta disponer de
-- atestaciones COSE autenticas; no se fabrica una firma para hacerla pasar.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;

DO $inventario$
DECLARE
    restricciones_necesidad integer;
    restricciones_decision integer;
    restricciones_instantanea integer;
BEGIN
    SELECT count(*) INTO restricciones_necesidad
      FROM pg_catalog.pg_constraint AS c
      JOIN pg_catalog.pg_class AS t ON t.oid = c.conrelid
      JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
     WHERE n.nspname = 'vec_bolsa_llamamientos'
       AND t.relname = 'propuesta' AND c.contype = 'u'
       AND pg_get_constraintdef(c.oid) LIKE
          '%(necesidad_ref, version_necesidad, huella_necesidad_sha256)%';
    SELECT count(*) INTO restricciones_decision
      FROM pg_catalog.pg_constraint AS c
      JOIN pg_catalog.pg_class AS t ON t.oid = c.conrelid
      JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
     WHERE n.nspname = 'vec_bolsa_llamamientos'
       AND t.relname = 'uso_decision' AND c.contype IN ('p', 'u')
       AND pg_get_constraintdef(c.oid) LIKE '%(decision_ref)%';
    SELECT count(*) INTO restricciones_instantanea
      FROM pg_catalog.pg_constraint AS c
      JOIN pg_catalog.pg_class AS t ON t.oid = c.conrelid
      JOIN pg_catalog.pg_namespace AS n ON n.oid = t.relnamespace
     WHERE n.nspname = 'vec_bolsa_llamamientos'
       AND t.relname = 'propuesta' AND c.contype = 'u'
       AND pg_get_constraintdef(c.oid) LIKE
          '%(instantanea_ref, version_instantanea, huella_instantanea_sha256)%';
    IF restricciones_necesidad <> 1 OR restricciones_decision < 1 OR
       restricciones_instantanea <> 1 THEN
        RAISE EXCEPTION 'faltan claves atomicas de unicidad/idempotencia';
    END IF;
    IF has_function_privilege(
        'vec_bolsa_llamamientos_ejecutor',
        'vec_bolsa_llamamientos.guardar_propuesta_v1(jsonb,jsonb,bytea,bytea)',
        'EXECUTE'
    ) THEN
        RAISE EXCEPTION 'la prueba cerrada abrio EXECUTE';
    END IF;
END
$inventario$;
ROLLBACK;
