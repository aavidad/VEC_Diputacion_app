-- Prueba autoritativa del consumo y su revalidación final para consultas RRHH.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';

SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:migracion:000006', 0
    )
);

LOCK TABLE
    vec_autorizacion_atestada_v3.revocacion_clave_capacidad,
    vec_autorizacion_atestada_v3.revocacion_configuracion,
    vec_autorizacion_atestada_v3.revocacion_raiz
IN ACCESS EXCLUSIVE MODE;

ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_rrhh_v3_interna(
        text, bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
OWNER TO vec_autorizacion_atestada_v3_propietario;

DO $prevalidacion$
DECLARE
    v_propietario oid;
    v_propietario_ct oid;
    v_consumo_cuadro oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_consumo_detalle oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_revalidar_cuadro oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_revalidar_detalle oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_revalidar_interna oid := pg_catalog.to_regprocedure(
        'vec_autorizacion_atestada_v3.revalidar_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    );
    v_checkpoint oid := pg_catalog.to_regclass(
        'vec_autorizacion_atestada_v3.checkpoint_gobierno'
    );
    v_resultado_revalidacion text :=
        'TABLE(decision_ref text, consumo_huella_sha256 text, '
        || 'revalidada_en timestamp with time zone)';
BEGIN
    SELECT oid INTO v_propietario
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_autorizacion_atestada_v3_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;
    SELECT oid INTO v_propietario_ct
      FROM pg_catalog.pg_roles
     WHERE rolname = 'vec_contratacion_temporal_propietario'
       AND NOT rolcanlogin
       AND NOT rolsuper
       AND NOT rolcreatedb
       AND NOT rolcreaterole
       AND NOT rolinherit
       AND NOT rolreplication
       AND NOT rolbypassrls;

    PERFORM 1
      FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id
       AND revision BETWEEN 0 AND 9007199254740990::numeric
     FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'checkpoint incompatible para prueba de consumo RRHH';
    END IF;

    IF v_propietario IS NULL
       OR v_propietario_ct IS NULL
       OR v_checkpoint IS NULL
       OR v_consumo_cuadro IS NULL
       OR v_consumo_detalle IS NULL
       OR v_revalidar_cuadro IS NULL
       OR v_revalidar_detalle IS NULL
       OR v_revalidar_interna IS NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_class c
            WHERE c.oid = v_checkpoint
              AND (
                  c.relowner <> v_propietario
                  OR c.relkind <> 'r'
                  OR c.relpersistence <> 'p'
                  OR NOT c.relrowsecurity
                  OR NOT c.relforcerowsecurity
              )
       )
       OR pg_catalog.obj_description(
              v_checkpoint, 'pg_catalog.pg_class'
          ) IS NOT NULL
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_seclabels s
            WHERE s.classoid = 'pg_catalog.pg_class'::regclass
              AND s.objoid = v_checkpoint
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_trigger g
            WHERE g.tgrelid = v_checkpoint
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_rewrite r
            WHERE r.ev_class = v_checkpoint
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_policy p
            WHERE p.polrelid = v_checkpoint
              AND p.polname = 'propietario_exacto'
              AND p.polcmd = '*'
              AND p.polpermissive
              AND p.polroles = ARRAY[v_propietario]::oid[]
              AND pg_catalog.pg_get_expr(p.polqual, p.polrelid) =
                  '(CURRENT_USER = '
                  || '''vec_autorizacion_atestada_v3_propietario''::name)'
              AND pg_catalog.pg_get_expr(
                      p.polwithcheck, p.polrelid
                  ) =
                  '(CURRENT_USER = '
                  || '''vec_autorizacion_atestada_v3_propietario''::name)'
       ) <> 1
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_policy p
            WHERE p.polrelid = v_checkpoint
       ) <> 1
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_class c
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    c.relacl, pg_catalog.acldefault('r', c.relowner)
                )
            ) a
            WHERE c.oid = v_checkpoint
              AND a.grantee = v_propietario
              AND a.grantor = v_propietario
              AND a.privilege_type = ANY(ARRAY[
                  'SELECT', 'INSERT', 'UPDATE', 'DELETE', 'TRUNCATE',
                  'REFERENCES', 'TRIGGER', 'MAINTAIN'
              ])
              AND NOT a.is_grantable
       ) <> 8
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_class c
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    c.relacl, pg_catalog.acldefault('r', c.relowner)
                )
            ) a
            WHERE c.oid = v_checkpoint
       ) <> 8
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_rrhh_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
       ) IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()'
       ) IS NOT NULL
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_trigger g
             JOIN pg_catalog.pg_class c ON c.oid = g.tgrelid
             JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND c.relname = ANY(ARRAY[
                  'revocacion_clave_capacidad',
                  'revocacion_configuracion',
                  'revocacion_raiz'
              ])
              AND g.tgname = 'serializar_revalidacion_rrhh_antes'
              AND NOT g.tgisinternal
       ) <> 0
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
             JOIN pg_catalog.pg_language l ON l.oid = f.prolang
            WHERE f.oid = ANY(ARRAY[
                      v_consumo_cuadro, v_consumo_detalle,
                      v_revalidar_interna,
                      v_revalidar_cuadro, v_revalidar_detalle
                  ])
              AND (
                  f.proowner <> v_propietario
                  OR l.lanname <> 'plpgsql'
                  OR NOT f.prosecdef
                  OR f.provolatile <> 'v'
                  OR f.proisstrict
                  OR f.proleakproof
                  OR f.proparallel <> 'u'
                  OR f.prokind <> 'f'
                  OR (
                      f.oid = ANY(ARRAY[
                          v_revalidar_interna,
                          v_revalidar_cuadro, v_revalidar_detalle
                      ])
                      AND (
                          NOT f.proretset
                          OR f.prorettype <> 'pg_catalog.record'::regtype
                          OR pg_catalog.pg_get_function_result(f.oid) <>
                             v_resultado_revalidacion
                          OR pg_catalog.obj_description(
                                 f.oid, 'pg_catalog.pg_proc'
                             ) IS NOT NULL
                      )
                  )
                  OR (
                      f.oid = ANY(ARRAY[
                          v_consumo_cuadro, v_consumo_detalle
                      ])
                      AND f.proconfig IS DISTINCT FROM
                          ARRAY[
                              'search_path=pg_catalog',
                              'lock_timeout=2s'
                          ]::text[]
                  )
                  OR (
                      f.oid = ANY(ARRAY[
                          v_revalidar_interna,
                          v_revalidar_cuadro, v_revalidar_detalle
                      ])
                      AND f.proconfig IS DISTINCT FROM
                          ARRAY[
                              'search_path=pg_catalog',
                              'lock_timeout=1s'
                          ]::text[]
                  )
              )
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[
                      v_revalidar_interna,
                      v_revalidar_cuadro, v_revalidar_detalle
                  ])
              AND pg_catalog.encode(
                      pg_catalog.sha256(
                          pg_catalog.convert_to(f.prosrc, 'UTF8')
                      ),
                      'hex'
                  ) IS DISTINCT FROM CASE f.oid
                  WHEN v_revalidar_interna THEN
                      'd3f72e15374a572dd6004193fc136369d3d5a42ec05e38e8afcd1868d8d8c553'
                  WHEN v_revalidar_cuadro THEN
                      '6f697bddff143180a6562cdacaec3a965e47720f5563731fc5c5011d9798a3e3'
                  WHEN v_revalidar_detalle THEN
                      'c7d3f11fbbb08ffb5a75ce347f19124888af4022a74d4a3c79e5076420bd7c09'
              END
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_seclabels s
            WHERE s.classoid = 'pg_catalog.pg_proc'::regclass
              AND s.objoid = ANY(ARRAY[
                  v_revalidar_interna,
                  v_revalidar_cuadro, v_revalidar_detalle
              ])
       )
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = v_revalidar_interna
              AND a.grantee = v_propietario
              AND a.grantor = v_propietario
              AND a.privilege_type = 'EXECUTE'
              AND NOT a.is_grantable
       ) <> 1
       OR (
           SELECT pg_catalog.count(*)
             FROM pg_catalog.pg_proc f
            CROSS JOIN LATERAL pg_catalog.aclexplode(
                COALESCE(
                    f.proacl,
                    pg_catalog.acldefault('f', f.proowner)
                )
            ) a
            WHERE f.oid = v_revalidar_interna
       ) <> 1
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[
                      v_revalidar_cuadro, v_revalidar_detalle
                  ])
              AND (
                  SELECT pg_catalog.count(*)
                    FROM pg_catalog.aclexplode(
                        COALESCE(
                            f.proacl,
                            pg_catalog.acldefault('f', f.proowner)
                        )
                    ) a
                   WHERE a.grantee = ANY(ARRAY[
                             v_propietario, v_propietario_ct
                         ])
                     AND a.grantor = v_propietario
                     AND a.privilege_type = 'EXECUTE'
                     AND NOT a.is_grantable
              ) <> 2
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[
                      v_revalidar_cuadro, v_revalidar_detalle
                  ])
              AND (
                  SELECT pg_catalog.count(*)
                    FROM pg_catalog.aclexplode(
                        COALESCE(
                            f.proacl,
                            pg_catalog.acldefault('f', f.proowner)
                        )
                    )
              ) <> 2
       )
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.pg_proc f
            WHERE f.oid = ANY(ARRAY[
                      v_consumo_cuadro, v_consumo_detalle,
                      v_revalidar_cuadro, v_revalidar_detalle
                  ])
              AND NOT pg_catalog.has_function_privilege(
                  v_propietario_ct, f.oid, 'EXECUTE'
              )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'estado incompatible para prueba de consumo RRHH';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3()
RETURNS trigger
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
AS $funcion$
BEGIN
    IF TG_OP <> 'INSERT'
       OR TG_TABLE_SCHEMA <> 'vec_autorizacion_atestada_v3'
       OR TG_TABLE_NAME NOT IN (
           'revocacion_clave_capacidad',
           'revocacion_configuracion',
           'revocacion_raiz'
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'serialización de revocación RRHH rechazada';
    END IF;

    UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
       SET revision = revision + 1,
           actualizada_en = pg_catalog.clock_timestamp()
     WHERE control_id
       AND revision < 9007199254740991::numeric;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'checkpoint de revocación RRHH no disponible';
    END IF;
    RETURN NEW;
END
$funcion$;

CREATE TRIGGER serializar_revalidacion_rrhh_antes
BEFORE INSERT ON
    vec_autorizacion_atestada_v3.revocacion_clave_capacidad
FOR EACH ROW
EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3();

CREATE TRIGGER serializar_revalidacion_rrhh_antes
BEFORE INSERT ON
    vec_autorizacion_atestada_v3.revocacion_configuracion
FOR EACH ROW
EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3();

CREATE TRIGGER serializar_revalidacion_rrhh_antes
BEFORE INSERT ON vec_autorizacion_atestada_v3.revocacion_raiz
FOR EACH ROW
EXECUTE FUNCTION
    vec_autorizacion_atestada_v3.serializar_revocacion_consultas_rrhh_v3();

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .serializar_revocacion_consultas_rrhh_v3()
FROM PUBLIC,
    vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario;

CREATE FUNCTION
vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
    p_perfil_consulta text,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
DECLARE
    v_revalidacion record;
    v_prueba record;
    v_huella_auditoria text;
BEGIN
    IF p_perfil_consulta = 'cuadro' THEN
        SELECT * INTO STRICT v_revalidacion
          FROM vec_autorizacion_atestada_v3
               .revalidar_consumo_consulta_cuadro_rrhh_v3_atestada(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign1,
              p_evidencia_verificacion, p_raiz_publica_spki
          );
    ELSIF p_perfil_consulta = 'detalle' THEN
        SELECT * INTO STRICT v_revalidacion
          FROM vec_autorizacion_atestada_v3
               .revalidar_consumo_consulta_detalle_rrhh_v3_atestada(
              p_capacidad_canonica, p_decision_canonica,
              p_motivo_canonico, p_contexto_actor_canonico,
              p_persona_version, p_perfil_version,
              p_payload_vec_ad_3, p_sobre_cose_sign1,
              p_evidencia_verificacion, p_raiz_publica_spki
          );
    ELSE
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'perfil de prueba de consumo RRHH inválido';
    END IF;

    SELECT
        t.decision_ref,
        t.huella_decision_sha256,
        t.decision_canonica,
        t.motivo_canonico,
        t.contexto_actor_canonico,
        t.payload_vec_ad_3,
        t.sobre_cose_sign1,
        t.evidencia_verificacion,
        t.raiz_publica_spki,
        t.capacidad_canonica,
        t.huella_capacidad_sha256,
        t.efecto_ref,
        t.huella_efecto_sha256,
        t.registrada_en AS atestada_en,
        c.huella_decision_sha256 AS consumo_decision_huella,
        c.consumo_huella_sha256,
        c.consumida_en,
        a.auditoria_ref,
        a.secuencia,
        a.anterior_sha256,
        a.huella_sha256 AS auditoria_huella_sha256,
        a.registrada_en AS auditada_en
      INTO STRICT v_prueba
      FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 t
      JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
        USING (decision_ref, efecto_ref, huella_efecto_sha256)
      JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
        USING (decision_ref, efecto_ref, huella_efecto_sha256)
     WHERE t.decision_ref = v_revalidacion.decision_ref
     FOR SHARE OF t, c, a;

    v_huella_auditoria := pg_catalog.encode(
        pg_catalog.sha256(
            vec_autorizacion_atestada_v3.encuadrar_mac(
                v_prueba.secuencia::text
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                v_prueba.anterior_sha256
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                v_prueba.decision_ref
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                v_prueba.efecto_ref
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                v_prueba.huella_efecto_sha256
            )
            || vec_autorizacion_atestada_v3.encuadrar_mac(
                v_prueba.consumo_huella_sha256
            )
        ),
        'hex'
    );

    IF v_prueba.decision_ref IS DISTINCT FROM
           v_revalidacion.decision_ref
       OR v_prueba.consumo_huella_sha256 IS DISTINCT FROM
          v_revalidacion.consumo_huella_sha256
       OR v_prueba.capacidad_canonica IS DISTINCT FROM
          p_capacidad_canonica
       OR v_prueba.decision_canonica IS DISTINCT FROM
          p_decision_canonica
       OR v_prueba.motivo_canonico IS DISTINCT FROM p_motivo_canonico
       OR v_prueba.contexto_actor_canonico IS DISTINCT FROM
          p_contexto_actor_canonico
       OR v_prueba.payload_vec_ad_3 IS DISTINCT FROM p_payload_vec_ad_3
       OR v_prueba.sobre_cose_sign1 IS DISTINCT FROM p_sobre_cose_sign1
       OR v_prueba.evidencia_verificacion IS DISTINCT FROM
          p_evidencia_verificacion
       OR v_prueba.raiz_publica_spki IS DISTINCT FROM p_raiz_publica_spki
       OR v_prueba.huella_capacidad_sha256 IS DISTINCT FROM
          pg_catalog.encode(
              pg_catalog.sha256(p_capacidad_canonica), 'hex'
          )
       OR v_prueba.huella_decision_sha256 IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(p_decision_canonica), 'hex')
       OR v_prueba.consumo_decision_huella IS DISTINCT FROM
          v_prueba.huella_decision_sha256
       OR v_prueba.auditoria_ref IS DISTINCT FROM
          'aud_v3_' || pg_catalog.substr(
              v_prueba.consumo_huella_sha256, 1, 32
          )
       OR v_prueba.auditoria_huella_sha256 IS DISTINCT FROM
          v_huella_auditoria
       OR v_prueba.atestada_en IS DISTINCT FROM v_prueba.consumida_en
       OR v_prueba.auditada_en IS DISTINCT FROM v_prueba.consumida_en
       OR v_revalidacion.revalidada_en < v_prueba.consumida_en
       OR (
           v_prueba.secuencia = 1
           AND v_prueba.anterior_sha256 <>
               pg_catalog.repeat('0', 64)
       )
       OR (
           v_prueba.secuencia > 1
           AND NOT EXISTS (
               SELECT 1
                 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 p
                WHERE p.secuencia = v_prueba.secuencia - 1
                  AND p.huella_sha256 = v_prueba.anterior_sha256
           )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'prueba autoritativa de consumo RRHH rechazada';
    END IF;

    RETURN QUERY SELECT
        v_prueba.decision_ref,
        v_prueba.efecto_ref,
        v_prueba.huella_efecto_sha256,
        v_prueba.consumo_huella_sha256,
        v_prueba.auditoria_ref,
        v_prueba.auditoria_huella_sha256,
        v_prueba.consumida_en,
        v_revalidacion.revalidada_en;
EXCEPTION
    WHEN no_data_found OR too_many_rows THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'prueba autoritativa de consumo RRHH rechazada';
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
BEGIN
    RETURN QUERY
    SELECT *
      FROM vec_autorizacion_atestada_v3
           .revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
          'cuadro',
          p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico,
          p_persona_version, p_perfil_version,
          p_payload_vec_ad_3, p_sobre_cose_sign1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
END
$funcion$;

CREATE FUNCTION
vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE (
    decision_ref text,
    efecto_ref text,
    huella_efecto_sha256 text,
    consumo_huella_sha256 text,
    auditoria_ref text,
    auditoria_huella_sha256 text,
    consumida_en timestamptz,
    revalidada_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '1s'
AS $funcion$
BEGIN
    RETURN QUERY
    SELECT *
      FROM vec_autorizacion_atestada_v3
           .revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
          'detalle',
          p_capacidad_canonica, p_decision_canonica,
          p_motivo_canonico, p_contexto_actor_canonico,
          p_persona_version, p_perfil_version,
          p_payload_vec_ad_3, p_sobre_cose_sign1,
          p_evidencia_verificacion, p_raiz_publica_spki
      );
END
$funcion$;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_rrhh_v3_interna(
        text, bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
FROM PUBLIC,
    vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario;

REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ),
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
FROM PUBLIC,
    vec_autorizacion_atestada_v3_migrador,
    vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor,
    vec_contratacion_temporal_propietario;

GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_cuadro_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    ),
    vec_autorizacion_atestada_v3
    .revalidar_evidencia_consumo_consulta_detalle_rrhh_v3_atestada(
        bytea, bytea, bytea, bytea, numeric, numeric,
        bytea, bytea, bytea, bytea
    )
TO vec_contratacion_temporal_propietario;

COMMIT;
