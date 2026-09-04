-- Retirada protegida de la reconciliacion por efecto, sin CASCADE.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';

DO $prevalidacion_operador$
BEGIN
    IF NOT EXISTS (
        SELECT 1
          FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de reconciliacion V2 requiere superusuario';
    END IF;
    IF current_setting(
           'vec.confirmar_retirada_reconciliacion_efecto_v2', true
       ) IS DISTINCT FROM
       'RETIRAR_RECONCILIACION_EFECTO_V2' THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'retirada de reconciliacion V2 no confirmada';
    END IF;
END
$prevalidacion_operador$;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:migracion:000001', 0
    )
);
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v2:migracion:000002', 0
    )
);

DO $prevalidacion_objeto$
DECLARE
    funcion record;
    propietario oid;
    consumidor oid;
    emisor oid;
BEGIN
    SELECT oid INTO propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_propietario';
    SELECT oid INTO consumidor
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_consumidor';
    SELECT oid INTO emisor
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_emisor_capacidad';
    SELECT funcion_catalogo.oid, funcion_catalogo.proowner,
           funcion_catalogo.prosecdef, funcion_catalogo.provolatile,
           funcion_catalogo.proretset, funcion_catalogo.proconfig,
           funcion_catalogo.proacl
      INTO funcion
      FROM pg_catalog.pg_proc AS funcion_catalogo
     WHERE funcion_catalogo.oid = to_regprocedure(
         'vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(text,text)'
     );

    IF funcion.oid IS NULL OR propietario IS NULL
       OR consumidor IS NULL OR emisor IS NULL
       OR funcion.proowner <> propietario
       OR funcion.prosecdef IS NOT TRUE
       OR funcion.provolatile <> 's'
       OR funcion.proretset IS NOT TRUE
       OR funcion.proconfig IS DISTINCT FROM
          ARRAY['search_path=pg_catalog']::text[] THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'reconciliacion V2 alterada; retirada denegada';
    END IF;

    IF EXISTS (
        SELECT 1
          FROM pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS permiso
         WHERE permiso.privilege_type = 'EXECUTE'
           AND permiso.grantee IN (0, emisor)
    ) OR NOT EXISTS (
        SELECT 1
          FROM pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS permiso
         WHERE permiso.privilege_type = 'EXECUTE'
           AND permiso.grantee = consumidor
           AND permiso.is_grantable IS FALSE
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.aclexplode(
              COALESCE(
                  funcion.proacl,
                  pg_catalog.acldefault('f', funcion.proowner)
              )
          ) AS permiso
         WHERE permiso.privilege_type = 'EXECUTE'
           AND permiso.grantee NOT IN (propietario, consumidor)
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'ACL de reconciliacion V2 alterada; retirada denegada';
    END IF;
END
$prevalidacion_objeto$;

DROP FUNCTION
    vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(text, text)
    RESTRICT;
COMMIT;
