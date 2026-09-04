BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';

DO $contrato$
DECLARE
    funcion record;
    propietario oid;
    consumidor oid;
    emisor oid;
BEGIN
    SELECT oid INTO propietario FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_propietario';
    SELECT oid INTO consumidor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_consumidor';
    SELECT oid INTO emisor FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v2_emisor_capacidad';
    SELECT funcion_catalogo.* INTO funcion
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
       OR funcion.pronargs <> 2
       OR funcion.proargtypes <> '25 25'::oidvector
       OR funcion.proconfig IS DISTINCT FROM
          ARRAY['search_path=pg_catalog']::text[]
       OR funcion.proargnames IS DISTINCT FROM ARRAY[
           'p_efecto_ref', 'p_huella_efecto_sha256', 'estado',
           'registro_ref', 'consumo_ref', 'auditoria_ref', 'consumida_en',
           'huella_auditoria_sha256'
       ]::text[]
       OR funcion.proargmodes IS DISTINCT FROM
          ARRAY['i', 'i', 't', 't', 't', 't', 't', 't']::"char"[]
       OR funcion.proallargtypes IS DISTINCT FROM ARRAY[
           'text'::regtype::oid, 'text'::regtype::oid,
           'text'::regtype::oid, 'text'::regtype::oid,
           'text'::regtype::oid, 'text'::regtype::oid,
           'timestamptz'::regtype::oid, 'text'::regtype::oid
       ]::oid[] THEN
        RAISE EXCEPTION 'contrato mecanico de reconciliacion V2 incorrecto';
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
        RAISE EXCEPTION 'ACL de reconciliacion V2 abierta';
    END IF;
END
$contrato$;

-- Las inserciones sinteticas solo preparan un ledger revertido para probar la
-- clasificacion. Las FK siguen probadas por la migracion base.
SET LOCAL session_replication_role = replica;
SET LOCAL ROLE vec_autorizacion_atestada_v2_propietario;

INSERT INTO vec_autorizacion_atestada_v2.consumo_capacidad_v2(
    clave_id, clave_version, nonce, registro_ref,
    huella_capacidad_sha256, capacidad, emitida_en, expira_en, consumida_en
) VALUES (
    'clave:d4:prueba', 1, repeat('a', 64), 'registro:d4:exacto',
    repeat('b', 64), '{}'::jsonb,
    '2026-09-03 09:59:58+00'::timestamptz,
    '2026-09-03 10:00:02+00'::timestamptz,
    '2026-09-03 10:00:00+00'::timestamptz
);
INSERT INTO vec_autorizacion_atestada_v2.consumo_decision_v2(
    consumo_ref, registro_ref, decision_ref, huella_decision_sha256,
    efecto_ref, huella_efecto_sha256, principal_id, accion, finalidad,
    sujeto_ref, recurso_ref, contexto_recurso_huella_sha256,
    correlacion_ref, consumida_en
) VALUES (
    'consumo:d4:exacto', 'registro:d4:exacto', 'decision:d4:exacta',
    repeat('c', 64), 'efecto:d4:exacto', repeat('d', 64),
    'principal:d4:prueba', 'bolsa.llamamiento.abierto.confirmar',
    'confirmacion_alta_llamamiento_abierto',
    'hmac-sha256:personas:' || repeat('1', 64),
    'llamamiento-abierto:' || repeat('2', 64), repeat('3', 64),
    'correlacion_44444444444444444444444444444444',
    '2026-09-03 10:00:00+00'::timestamptz
);
INSERT INTO vec_autorizacion_atestada_v2.auditoria_consumo_v2(
    auditoria_ref, secuencia, consumo_ref, registro_ref, decision_ref,
    efecto_ref, accion, finalidad, correlacion_ref, ocurrida_en,
    huella_anterior_sha256, huella_registro_sha256
) VALUES (
    'auditoria:d4:exacta', 18446744073709551615,
    'consumo:d4:exacto', 'registro:d4:exacto', 'decision:d4:exacta',
    'efecto:d4:exacto', 'bolsa.llamamiento.abierto.confirmar',
    'confirmacion_alta_llamamiento_abierto',
    'correlacion_44444444444444444444444444444444',
    '2026-09-03 10:00:00+00'::timestamptz,
    repeat('0', 64), repeat('4', 64)
);

-- Un par exacto sin las piezas del recibo debe ser indeterminado, nunca exacto
-- ni ausente.
INSERT INTO vec_autorizacion_atestada_v2.consumo_decision_v2(
    consumo_ref, registro_ref, decision_ref, huella_decision_sha256,
    efecto_ref, huella_efecto_sha256, principal_id, accion, finalidad,
    sujeto_ref, recurso_ref, contexto_recurso_huella_sha256,
    correlacion_ref, consumida_en
) VALUES (
    'consumo:d4:incompleto', 'registro:d4:incompleto',
    'decision:d4:incompleta', repeat('5', 64),
    'efecto:d4:incompleto', repeat('e', 64),
    'principal:d4:prueba', 'bolsa.llamamiento.abierto.confirmar',
    'confirmacion_alta_llamamiento_abierto',
    'hmac-sha256:personas:' || repeat('6', 64),
    'llamamiento-abierto:' || repeat('7', 64), repeat('8', 64),
    'correlacion_99999999999999999999999999999999',
    '2026-09-03 10:00:00+00'::timestamptz
);

RESET ROLE;
SET LOCAL session_replication_role = origin;

SET SESSION AUTHORIZATION vec_ad2_consumidor_prueba;
DO $clasificacion$
DECLARE
    resultado record;
BEGIN
    SELECT * INTO STRICT resultado
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
          'efecto:d4:exacto', repeat('d', 64)
      );
    IF resultado.estado IS DISTINCT FROM 'exacto'
       OR resultado.registro_ref IS DISTINCT FROM 'registro:d4:exacto'
       OR resultado.consumo_ref IS DISTINCT FROM 'consumo:d4:exacto'
       OR resultado.auditoria_ref IS DISTINCT FROM 'auditoria:d4:exacta'
       OR resultado.consumida_en IS DISTINCT FROM
          '2026-09-03 10:00:00+00'::timestamptz
       OR resultado.huella_auditoria_sha256 IS DISTINCT FROM repeat('4', 64)
    THEN
        RAISE EXCEPTION 'el par exacto no devolvio su recibo opaco';
    END IF;

    SELECT * INTO STRICT resultado
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
          'efecto:d4:exacto', repeat('f', 64)
      );
    IF resultado.estado IS DISTINCT FROM 'colision'
       OR resultado.registro_ref IS NOT NULL
       OR resultado.consumo_ref IS NOT NULL
       OR resultado.auditoria_ref IS NOT NULL
       OR resultado.consumida_en IS NOT NULL
       OR resultado.huella_auditoria_sha256 IS NOT NULL THEN
        RAISE EXCEPTION 'la colision por efecto expuso recibo';
    END IF;

    SELECT * INTO STRICT resultado
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
          'efecto:d4:otro', repeat('d', 64)
      );
    IF resultado.estado IS DISTINCT FROM 'colision'
       OR resultado.registro_ref IS NOT NULL
       OR resultado.consumo_ref IS NOT NULL
       OR resultado.auditoria_ref IS NOT NULL
       OR resultado.consumida_en IS NOT NULL
       OR resultado.huella_auditoria_sha256 IS NOT NULL THEN
        RAISE EXCEPTION 'la colision por huella expuso recibo';
    END IF;
    SELECT * INTO STRICT resultado
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
          'efecto:d4:exacto', repeat('e', 64)
      );
    IF resultado.estado IS DISTINCT FROM 'colision'
       OR resultado.registro_ref IS NOT NULL
       OR resultado.consumo_ref IS NOT NULL
       OR resultado.auditoria_ref IS NOT NULL
       OR resultado.consumida_en IS NOT NULL
       OR resultado.huella_auditoria_sha256 IS NOT NULL THEN
        RAISE EXCEPTION 'la colision entre dos filas expuso recibo';
    END IF;


    SELECT * INTO STRICT resultado
      FROM vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
          'efecto:d4:ausente', repeat('9', 64)
      );
    IF resultado.estado IS DISTINCT FROM 'ausente'
       OR resultado.registro_ref IS NOT NULL
       OR resultado.consumo_ref IS NOT NULL
       OR resultado.auditoria_ref IS NOT NULL
       OR resultado.consumida_en IS NOT NULL
       OR resultado.huella_auditoria_sha256 IS NOT NULL THEN
        RAISE EXCEPTION 'la ausencia no quedo cerrada y opaca';
    END IF;

    BEGIN
        PERFORM * FROM
            vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
                'efecto:d4:incompleto', repeat('e', 64)
            );
        RAISE EXCEPTION 'un recibo incompleto se acepto como exacto';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
    BEGIN
        PERFORM * FROM
            vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
                NULL, repeat('a', 64)
            );
        RAISE EXCEPTION 'una entrada nula se acepto';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
    BEGIN
        PERFORM * FROM
            vec_autorizacion_atestada_v2.reconciliar_consumo_efecto_v2(
                'efecto:d4:invalido', repeat('A', 64)
            );
        RAISE EXCEPTION 'una huella no canonica se acepto';
    EXCEPTION WHEN SQLSTATE '55000' THEN
        NULL;
    END;
END
$clasificacion$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
