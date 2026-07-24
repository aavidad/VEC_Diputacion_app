\if :{?confirmar_retirada_acreditacion_contexto_actor_v2}
\else
\echo 'falta confirmacion de retirada de acreditacion ContextoActor V2'
\quit 3
\endif
SELECT :'confirmar_retirada_acreditacion_contexto_actor_v2' =
       'RETIRAR_ACREDITACION_CONTEXTO_ACTOR_V2' AS confirmacion_valida \gset
\if :confirmacion_valida
\else
\echo 'confirmacion de retirada de acreditacion ContextoActor V2 incorrecta'
\quit 3
\endif

BEGIN;
SET LOCAL search_path = pg_catalog;

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:migracion:acreditacion_uso:v2', 0
    )
);

DO $prevalidacion$
DECLARE
    funcion_oid oid;
    serializar_oid oid;
    avanzar_oid oid;
    control_oid oid;
    control_tipo_oid oid;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'retirada de acreditacion ContextoActor V2 requiere superusuario';
    END IF;
    funcion_oid := pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(text,text,text,text,text,text,numeric,text,numeric,text,numeric,text,numeric,text,text,timestamptz,timestamptz)'
    );
    serializar_oid := pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2()'
    );
    avanzar_oid := pg_catalog.to_regprocedure(
        'vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2()'
    );
    control_oid := pg_catalog.to_regclass(
        'vec_contexto_actor_v1.control_generacion_punteros_actuales_v2'
    );
    IF funcion_oid IS NULL OR serializar_oid IS NULL OR avanzar_oid IS NULL
       OR control_oid IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'faltan objetos de acreditacion ContextoActor V2';
    END IF;
    SELECT c.reltype
      INTO STRICT control_tipo_oid
      FROM pg_catalog.pg_class AS c
     WHERE c.oid = control_oid;
    -- Una composicion futura debe retirar antes su GRANT nominal. No se
    -- destruyen silenciosamente ACL cruzados aunque DROP pudiera hacerlo.
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_proc AS p
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  p.proacl, pg_catalog.acldefault('f', p.proowner)
              )
          ) AS a
         WHERE p.oid = ANY (ARRAY[funcion_oid, serializar_oid, avanzar_oid])
           AND a.grantee <> p.proowner
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_class AS c
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  c.relacl, pg_catalog.acldefault('r', c.relowner)
              )
          ) AS a
         WHERE c.oid = control_oid
           AND a.grantee <> c.relowner
    ) OR EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS t
          CROSS JOIN LATERAL pg_catalog.aclexplode(
              coalesce(
                  t.typacl, pg_catalog.acldefault('T', t.typowner)
              )
          ) AS a
         WHERE t.oid = control_tipo_oid
           AND a.grantee <> t.typowner
    ) THEN
        RAISE EXCEPTION USING ERRCODE = '2BP01',
            MESSAGE = 'la acreditacion ContextoActor V2 conserva concesiones externas';
    END IF;
    IF (
        SELECT pg_catalog.count(*)
          FROM pg_catalog.pg_trigger AS t
         WHERE t.tgrelid = ANY (ARRAY[
             'vec_contexto_actor_v1.proyeccion_cuenta_actual'::regclass,
             'vec_contexto_actor_v1.persona_actual'::regclass,
             'vec_contexto_actor_v1.perfil_actual'::regclass,
             'vec_contexto_actor_v1.vinculo_contexto_actual'::regclass,
             'vec_contexto_actor_v1.vinculo_referencia_actual'::regclass
         ])
           AND t.tgname IN (
             'puntero_actual_no_truncable_v2',
             'serializar_mutacion_punteros_actuales_v2',
             'avanzar_generacion_punteros_actuales_v2'
           )
           AND NOT t.tgisinternal
    ) <> 15 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'inventario de triggers ContextoActor V2 incompleto';
    END IF;
END
$prevalidacion$;

SET LOCAL ROLE vec_contexto_actor_v1_propietario;
SET LOCAL search_path = pg_catalog;

DROP FUNCTION
    vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(
        text, text, text, text, text, text, numeric, text, numeric,
        text, numeric, text, numeric, text, text, timestamptz, timestamptz
    );

DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.proyeccion_cuenta_actual;
DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.proyeccion_cuenta_actual;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.proyeccion_cuenta_actual;
DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.persona_actual;
DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.persona_actual;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.persona_actual;
DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.perfil_actual;
DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.perfil_actual;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.perfil_actual;
DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.vinculo_contexto_actual;
DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.vinculo_contexto_actual;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.vinculo_contexto_actual;
DROP TRIGGER serializar_mutacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.vinculo_referencia_actual;
DROP TRIGGER avanzar_generacion_punteros_actuales_v2
    ON vec_contexto_actor_v1.vinculo_referencia_actual;
DROP TRIGGER puntero_actual_no_truncable_v2
    ON vec_contexto_actor_v1.vinculo_referencia_actual;

DROP FUNCTION
    vec_contexto_actor_v1.avanzar_generacion_punteros_actuales_v2();
DROP FUNCTION
    vec_contexto_actor_v1.serializar_mutacion_punteros_actuales_v2();
DROP TABLE vec_contexto_actor_v1.control_generacion_punteros_actuales_v2;

COMMIT;
