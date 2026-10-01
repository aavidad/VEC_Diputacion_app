\set ON_ERROR_STOP on
-- Comprobación de solo lectura tras AD139 y AD140 en la copia física H6.
BEGIN READ ONLY;
DO $comprobar$
DECLARE
  n_roles pg_catalog.int8;
  n_membresias pg_catalog.int8;
  n_tipos pg_catalog.int8;
  n_public pg_catalog.int8;
  n_externas pg_catalog.int8;
BEGIN
  SELECT pg_catalog.count(*) INTO n_roles
  FROM pg_catalog.pg_roles WHERE pg_catalog.left(rolname, 4) = 'vec_';
  SELECT pg_catalog.count(*) INTO n_membresias
  FROM pg_catalog.pg_auth_members a
  JOIN pg_catalog.pg_roles g ON g.oid = a.roleid
  JOIN pg_catalog.pg_roles m ON m.oid = a.member
  WHERE pg_catalog.left(g.rolname, 4) = 'vec_'
     OR pg_catalog.left(m.rolname, 4) = 'vec_';
  IF n_roles <> 171 OR n_membresias <> 90 THEN
    RAISE EXCEPTION 'AD139/140: roles o membresías perdidos (%/%).', n_roles, n_membresias;
  END IF;

  SELECT pg_catalog.count(*) INTO n_tipos FROM pg_catalog.pg_type
  WHERE typnamespace = pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
    AND typtype = 'c';
  SELECT pg_catalog.count(*) INTO n_public
  FROM pg_catalog.pg_type t
  CROSS JOIN LATERAL pg_catalog.aclexplode(
    coalesce(t.typacl, pg_catalog.acldefault('T', t.typowner))) a
  WHERE t.typnamespace = pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
    AND t.typtype = 'c' AND a.grantee = 0;
  IF n_tipos <> 21 OR n_public <> 0 THEN
    RAISE EXCEPTION 'AD139/140: tipos o USAGE PUBLIC incompatibles (%/%).', n_tipos, n_public;
  END IF;

  IF EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
             CROSS JOIN LATERAL pg_catalog.aclexplode(
               coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) a
             WHERE p.pronamespace = pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
               AND a.grantee = 0)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p
                WHERE p.pronamespace = pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
                  AND p.proconfig @> ARRAY['search_path=pg_catalog']) THEN
    RAISE EXCEPTION 'AD139/140: función pública o search_path pendiente.';
  END IF;

  SELECT pg_catalog.count(*) INTO n_externas FROM pg_catalog.pg_proc p
  WHERE p.pronamespace = pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
    AND p.proname = ANY(ARRAY[
      'avanzar_checkpoint_externo', 'leer_estado_publicacion_externa_v1',
      'material_publico_externo_v1', 'preparar_publicacion_externa_v1',
      'publicar_confianza_externa_v1']);
  IF n_externas <> 5 OR
     (SELECT pg_catalog.count(*) FROM pg_catalog.pg_class c
      WHERE c.relnamespace = pg_catalog.to_regnamespace('vec_autorizacion_atestada_v3')
        AND c.relname = ANY(ARRAY[
          'puntero_configuracion_externa', 'puntero_clave_emision_externa',
          'checkpoint_gobierno_externo'])
        AND c.relrowsecurity AND c.relforcerowsecurity) <> 3 THEN
    RAISE EXCEPTION 'AD139/140: raíz externa o RLS incompletos.';
  END IF;
END $comprobar$;
ROLLBACK;
