\set ON_ERROR_STOP on
-- Delta DBA aditivo: 28 tipos fila CT1..16 y 14 tipos fila AD3-1/2.
-- Ejecutar una sola vez antes del alta del selector corporativo RRHH.
-- No hay DOWN: restaurar USAGE a PUBLIC abriría una frontera cerrada.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_contexto_actor_v1:acl-tipos-ct-ad3-preselector:v1', 0
    )
);
-- GRANT/REVOKE ON TYPE modifica pg_type sin bloquear la tabla propietaria.
LOCK TABLE pg_catalog.pg_type IN SHARE ROW EXCLUSIVE MODE;

DO $delta$
DECLARE
    ct constant text[] := ARRAY[
        'identidad_reserva_alta',
        'reserva_alta_version',
        'reserva_alta_actual',
        'politica_generaciones_hmac_alta',
        'alias_ambito_alta',
        'alias_huella_alta',
        'expediente_alta',
        'expediente_alta_version',
        'actuacion_alta',
        'control_cadenas_alta',
        'auditoria_alta',
        'outbox_alta',
        'confirmacion_agregado_alta',
        'expediente_version_integral',
        'expediente_integral_actual',
        'reserva_operacion_analisis',
        'alias_operacion_analisis',
        'reserva_operacion_analisis_version',
        'reserva_operacion_analisis_actual',
        'confirmacion_operacion_analisis',
        'alias_consulta_operacion_analisis',
        'actuacion_expediente_integral',
        'consumo_fuentes_analisis',
        'consumo_decision_analisis',
        'control_cadenas_expediente_integral',
        'auditoria_expediente_integral',
        'outbox_expediente_integral',
        'vinculo_replay_operacion_analisis_v2'
    ];
    ad3 constant text[] := ARRAY[
        'clave_capacidad_version',
        'puntero_clave_emision',
        'revocacion_clave_capacidad',
        'configuracion_confianza_version',
        'raiz_confianza_version',
        'configuracion_raiz',
        'puntero_configuracion_actual',
        'revocacion_configuracion',
        'revocacion_raiz',
        'checkpoint_gobierno',
        'atestacion_decision_v3',
        'consumo_decision_v3',
        'control_cadena_auditoria',
        'auditoria_consumo_v3'
    ];
    esquema text;
    dueno name;
    nombre text;
    fila pg_catalog.pg_type%ROWTYPE;
    tabla pg_catalog.pg_class%ROWTYPE;
    total integer := 0;
    abiertos integer := 0;
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_catalog.pg_roles
         WHERE rolname = current_user AND rolsuper
    ) THEN
        RAISE EXCEPTION 'ACL CT/AD3: requiere DBA superusuario'
            USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.current_setting('server_version_num')::integer < 180000 THEN
        RAISE EXCEPTION 'ACL CT/AD3: requiere PostgreSQL 18'
            USING ERRCODE = '55000';
    END IF;
    IF pg_catalog.cardinality(ct) <> 28 OR pg_catalog.cardinality(ad3) <> 14
       OR (SELECT count(DISTINCT x) FROM pg_catalog.unnest(ct) AS x) <> 28
       OR (SELECT count(DISTINCT x) FROM pg_catalog.unnest(ad3) AS x) <> 14 THEN
        RAISE EXCEPTION 'ACL CT/AD3: inventario interno incompleto'
            USING ERRCODE = '55000';
    END IF;
    IF (SELECT count(*)
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE ((n.nspname = 'vec_contratacion_temporal'
                 AND t.typtype = 'c' AND t.typrelid <> 0)
             OR (n.nspname = 'vec_autorizacion_atestada_v3'
                 AND t.typtype = 'c' AND t.typrelid <> 0))) <> 42 THEN
        RAISE EXCEPTION 'ACL CT/AD3: conjunto de tipos fila incompatible'
            USING ERRCODE = '55000';
    END IF;

    FOR esquema, dueno, nombre IN
        SELECT 'vec_contratacion_temporal'::text,
               'vec_contratacion_temporal_propietario'::name, x
          FROM pg_catalog.unnest(ct) AS x
        UNION ALL
        SELECT 'vec_autorizacion_atestada_v3'::text,
               'vec_autorizacion_atestada_v3_propietario'::name, x
          FROM pg_catalog.unnest(ad3) AS x
    LOOP
        total := total + 1;
        SELECT t.* INTO fila
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE n.nspname = esquema AND t.typname = nombre;
        IF NOT FOUND OR pg_catalog.to_regrole(dueno) IS NULL
           OR fila.typowner <> pg_catalog.to_regrole(dueno)::oid
           OR fila.typtype <> 'c' OR fila.typrelid = 0
           OR fila.typisdefined IS NOT TRUE
           OR (SELECT n.nspowner FROM pg_catalog.pg_namespace AS n
                WHERE n.oid = fila.typnamespace)
              <> pg_catalog.to_regrole(dueno)::oid THEN
            RAISE EXCEPTION 'ACL CT/AD3: tipo o propietario incompatible: %.%',
                esquema, nombre USING ERRCODE = '55000';
        END IF;
        SELECT c.* INTO tabla FROM pg_catalog.pg_class AS c
         WHERE c.oid = fila.typrelid;
        IF NOT FOUND OR tabla.relkind <> 'r' OR tabla.reltype <> fila.oid
           OR tabla.relowner <> fila.typowner
           OR tabla.relnamespace <> fila.typnamespace
           OR tabla.relname <> fila.typname THEN
            RAISE EXCEPTION 'ACL CT/AD3: tipo fila no ligado a tabla: %.%',
                esquema, nombre USING ERRCODE = '55000';
        END IF;
        -- NULL es exactamente la preimagen implícita owner+PUBLIC de PG18.
        -- La única postimagen aceptada conserva USAGE exclusivamente al owner.
        IF (SELECT count(*) FROM pg_catalog.aclexplode(
                coalesce(fila.typacl,
                         pg_catalog.acldefault('T', fila.typowner)))) <>
                (CASE WHEN fila.typacl IS NULL THEN 2 ELSE 1 END)
           OR NOT EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(coalesce(
                    fila.typacl,
                    pg_catalog.acldefault('T', fila.typowner))) AS a
                 WHERE a.grantee = fila.typowner
                   AND a.grantor = fila.typowner
                   AND a.privilege_type = 'USAGE'
                   AND NOT a.is_grantable)
           OR EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(coalesce(
                    fila.typacl,
                    pg_catalog.acldefault('T', fila.typowner))) AS a
                 WHERE NOT (a.grantee = fila.typowner
                            AND a.grantor = fila.typowner
                            AND a.privilege_type = 'USAGE'
                            AND NOT a.is_grantable)
                   AND NOT (fila.typacl IS NULL
                            AND a.grantee = 0
                            AND a.grantor = fila.typowner
                            AND a.privilege_type = 'USAGE'
                            AND NOT a.is_grantable))
           OR (fila.typacl IS NULL AND NOT EXISTS (
                SELECT 1 FROM pg_catalog.aclexplode(
                    pg_catalog.acldefault('T', fila.typowner)) AS a
                 WHERE a.grantee = 0
                   AND a.grantor = fila.typowner
                   AND a.privilege_type = 'USAGE'
                   AND NOT a.is_grantable)) THEN
            RAISE EXCEPTION 'ACL CT/AD3: ACL previa incompatible: %.%',
                esquema, nombre USING ERRCODE = '55000';
        END IF;
        IF fila.typacl IS NULL THEN
            abiertos := abiertos + 1;
        END IF;
    END LOOP;
    IF total <> 42 OR abiertos NOT IN (0, 42) THEN
        RAISE EXCEPTION 'ACL CT/AD3: estado parcial: %/42 abiertos',
            abiertos USING ERRCODE = '55000';
    END IF;
    IF abiertos = 0 THEN
        RAISE NOTICE 'NO_APLICA ACL CT/AD3: 42 tipos fila ya cerrados';
        RETURN;
    END IF;
    IF pg_catalog.to_regrole('vec_contexto_actor_corporativo_rrhh_selector')
       IS NOT NULL THEN
        RAISE EXCEPTION 'ACL CT/AD3: selector ya existe con tipos abiertos'
            USING ERRCODE = '55000';
    END IF;

    -- El conjunto completo quedó validado bajo el bloqueo del catálogo.
    FOR esquema, nombre IN
        SELECT 'vec_contratacion_temporal'::text, x
          FROM pg_catalog.unnest(ct) AS x
        UNION ALL
        SELECT 'vec_autorizacion_atestada_v3'::text, x
          FROM pg_catalog.unnest(ad3) AS x
    LOOP
        EXECUTE pg_catalog.format(
            'REVOKE USAGE ON TYPE %I.%I FROM PUBLIC', esquema, nombre
        );
    END LOOP;
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.pg_type AS t
          JOIN pg_catalog.pg_namespace AS n ON n.oid = t.typnamespace
         WHERE ((n.nspname = 'vec_contratacion_temporal'
                 AND t.typname = ANY(ct))
             OR (n.nspname = 'vec_autorizacion_atestada_v3'
                 AND t.typname = ANY(ad3)))
           AND (
                t.typacl IS NULL
                OR (SELECT count(*)
                      FROM pg_catalog.aclexplode(t.typacl)) <> 1
                OR NOT EXISTS (
                    SELECT 1 FROM pg_catalog.aclexplode(t.typacl) AS a
                     WHERE a.grantee = t.typowner
                       AND a.grantor = t.typowner
                       AND a.privilege_type = 'USAGE'
                       AND NOT a.is_grantable))
    ) THEN
        RAISE EXCEPTION 'ACL CT/AD3: postimagen incompatible'
            USING ERRCODE = '55000';
    END IF;
    RAISE NOTICE 'ACL CT/AD3: 42 tipos fila cerrados';
END $delta$;
COMMIT;
