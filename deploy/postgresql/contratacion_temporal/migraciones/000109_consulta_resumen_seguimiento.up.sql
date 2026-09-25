\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000109', 0)
);

-- El lector de seguimiento usa la audiencia y el contexto atestados de
-- detalle, pero una proyección propia: nunca materializa el detalle completo.
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(text,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'
    ) IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para consulta mínima de seguimiento';
    END IF;
END
$prevalidacion$;

-- La fachada CT45 devuelve el expediente completo. Una decisión con campos
-- expresamente limitados no puede pasar por ella, aunque VEC-AD-3 sea válida.
-- Se conserva su definición instalada y se añade una única guarda antes del
-- motor, tras el cotejo de capacidad/decisión que ya hacía CT45.
DO $cerrar_detalle_amplio$
DECLARE
    v_funcion oid :=
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::pg_catalog.regprocedure;
    v_definicion text;
    v_cuerpo text;
    v_ancla text := '    v_material := ROW(';
BEGIN
    SELECT pg_catalog.pg_get_functiondef(p.oid), p.prosrc
      INTO STRICT v_definicion, v_cuerpo
      FROM pg_catalog.pg_proc p
     WHERE p.oid = v_funcion
       AND p.proowner = 'vec_contratacion_temporal_propietario'::regrole
       AND p.prokind = 'f' AND p.prorettype = 'pg_catalog.record'::regtype
       AND p.proretset AND p.prosecdef AND p.provolatile = 'v'
       AND p.proparallel = 'u'
       AND p.proconfig = ARRAY[
           'search_path=pg_catalog', 'row_security=on', 'TimeZone=UTC',
           'lock_timeout=1s', 'statement_timeout=4s',
           'idle_in_transaction_session_timeout=6s'
       ]::text[]
       AND (SELECT pg_catalog.count(*) FROM pg_catalog.aclexplode(p.proacl)) = 2
       AND NOT EXISTS (
           SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
            WHERE a.privilege_type <> 'EXECUTE' OR a.is_grantable
               OR a.grantee NOT IN (
                   'vec_contratacion_temporal_propietario'::regrole::oid,
                   'vec_contratacion_temporal_consultor_rrhh'::regrole::oid
               )
       )
       AND EXISTS (
           SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
            WHERE a.grantee =
                  'vec_contratacion_temporal_propietario'::regrole::oid
       )
       AND EXISTS (
           SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
            WHERE a.grantee =
                  'vec_contratacion_temporal_consultor_rrhh'::regrole::oid
       );
    IF pg_catalog.octet_length(v_cuerpo) <> 11450
       OR pg_catalog.encode(pg_catalog.sha256(
           pg_catalog.convert_to(v_cuerpo, 'UTF8')), 'hex') <>
          '7b7d6c4a419262d54ddb7f2a096e5a4d6e1c962bc58e3027546e221717ed1814'
       OR pg_catalog.length(v_definicion) -
       pg_catalog.length(pg_catalog.replace(v_definicion, v_ancla, ''))
       <> pg_catalog.length(v_ancla)
       OR pg_catalog.strpos(v_definicion,
           'v_decision -> ''campos_permitidos'' IS DISTINCT FROM') <> 0 THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'preimagen inesperada de consulta de detalle RRHH';
    END IF;
    EXECUTE pg_catalog.replace(
        v_definicion, v_ancla,
        '    IF v_decision ->> ''garantia_minima'' IS DISTINCT FROM ''alto'''
        || E'\n' || '       OR v_decision -> ''campos_permitidos'' IS DISTINCT FROM ''[]''::jsonb'
        || E'\n' || '       OR pg_catalog.left(COALESCE(v_decision ->> ''version_rol_ref'', ''''), 44) ='
        || E'\n' || '          ''rol:rrhh_interno_certificado_seguimiento_ct_'' THEN'
        || E'\n' || '        RAISE EXCEPTION USING ERRCODE = ''42501'','
        || E'\n' || '            MESSAGE = ''consulta RRHH rechazada'';'
        || E'\n' || '    END IF;'
        || E'\n\n' || v_ancla
    );
END
$cerrar_detalle_amplio$;

CREATE FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    p_unidad_ref text,
    p_capacidad_canonica bytea,
    p_decision_canonica bytea,
    p_motivo_canonico bytea,
    p_contexto_actor_canonico bytea,
    p_persona_version numeric,
    p_perfil_version numeric,
    p_payload_vec_ad_3 bytea,
    p_sobre_cose_sign_1 bytea,
    p_evidencia_verificacion bytea,
    p_raiz_publica_spki bytea
)
RETURNS TABLE (
    expediente_ref text,
    version_expediente numeric,
    organizacion_ref text,
    unidad_ref text,
    consumo_vec_huella_sha256 text,
    auditoria_vec_ref text,
    auditoria_vec_huella_sha256 text,
    consumida_en timestamptz
)
LANGUAGE plpgsql
VOLATILE
SECURITY DEFINER
PARALLEL UNSAFE
SET search_path = pg_catalog
SET row_security = 'on'
SET timezone = 'UTC'
SET lock_timeout = '1s'
SET statement_timeout = '4s'
SET idle_in_transaction_session_timeout = '6s'
AS $funcion$
DECLARE
    v_login pg_catalog.pg_roles%ROWTYPE;
    v_capacidad jsonb;
    v_decision jsonb;
    v_contexto_actor jsonb;
    v_consulta_canonica bytea;
    v_contexto_recurso bytea;
    v_contexto_huella text;
    v_material vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    v_consumo vec_contratacion_temporal.evidencia_consumo_nuevo_rrhh_v3;
    v_corte_global numeric(20, 0);
    v_fila record;
BEGIN
    SELECT * INTO v_login FROM pg_catalog.pg_roles
     WHERE rolname = SESSION_USER;
    IF CURRENT_USER <> 'vec_contratacion_temporal_propietario'
       OR SESSION_USER = CURRENT_USER
       OR v_login.oid IS NULL OR NOT v_login.rolcanlogin
       OR NOT v_login.rolinherit OR v_login.rolsuper
       OR v_login.rolcreatedb OR v_login.rolcreaterole
       OR v_login.rolreplication OR v_login.rolbypassrls
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = v_login.oid) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
           JOIN pg_catalog.pg_roles r ON r.oid = m.roleid
           WHERE m.member = v_login.oid
             AND r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
             AND NOT m.admin_option AND m.inherit_option
             AND NOT m.set_option
       )
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
                   WHERE m.roleid = v_login.oid)
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_roles r
            WHERE r.rolname = 'vec_contratacion_temporal_consultor_rrhh'
              AND NOT r.rolcanlogin AND r.rolinherit
              AND NOT r.rolsuper AND NOT r.rolcreatedb
              AND NOT r.rolcreaterole AND NOT r.rolreplication
              AND NOT r.rolbypassrls
       )
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member =
                  'vec_contratacion_temporal_consultor_rrhh'::regrole
       )
       OR pg_catalog.pg_is_in_recovery()
       OR pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('lock_timeout') = '0'
       OR pg_catalog.current_setting('lock_timeout')::interval > interval '1 second'
       OR pg_catalog.current_setting('statement_timeout') = '0'
       OR pg_catalog.current_setting('statement_timeout')::interval > interval '4 seconds'
       OR pg_catalog.current_setting('idle_in_transaction_session_timeout') = '0'
       OR pg_catalog.current_setting(
           'idle_in_transaction_session_timeout')::interval > interval '6 seconds'
       OR p_alcance IS NULL OR p_consulta IS NULL
       OR pg_catalog.octet_length(COALESCE(p_alcance.organizacion_ref, '')) > 160
       OR pg_catalog.octet_length(COALESCE(p_alcance.clase_ambito, '')) > 16
       OR pg_catalog.octet_length(COALESCE(p_alcance.ambito_ref, '')) > 160
       OR pg_catalog.octet_length(COALESCE(p_consulta.expediente_ref, '')) > 160
       OR p_unidad_ref IS NULL
       OR pg_catalog.octet_length(p_unidad_ref) > 160
       OR p_unidad_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;

    -- Límites O(1) antes de construir el material privado y antes de
    -- decodificar JSON o ejecutar los cánones CT40, iguales a CT45.
    IF p_capacidad_canonica IS NULL
       OR pg_catalog.octet_length(p_capacidad_canonica)
          NOT BETWEEN 512 AND 32768
       OR p_decision_canonica IS NULL
       OR pg_catalog.octet_length(p_decision_canonica)
          NOT BETWEEN 1 AND 524288
       OR p_motivo_canonico IS NULL
       OR pg_catalog.octet_length(p_motivo_canonico)
          NOT BETWEEN 1 AND 65536
       OR p_contexto_actor_canonico IS NULL
       OR pg_catalog.octet_length(p_contexto_actor_canonico)
          NOT BETWEEN 1 AND 262144
       OR p_persona_version IS NULL
       OR p_persona_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_persona_version <> pg_catalog.trunc(p_persona_version)
       OR p_perfil_version IS NULL
       OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991::numeric
       OR p_perfil_version <> pg_catalog.trunc(p_perfil_version)
       OR p_payload_vec_ad_3 IS NULL
       OR pg_catalog.octet_length(p_payload_vec_ad_3)
          NOT BETWEEN 1 AND 1048576
       OR p_sobre_cose_sign_1 IS NULL
       OR pg_catalog.octet_length(p_sobre_cose_sign_1)
          NOT BETWEEN 1 AND 1048576
       OR p_evidencia_verificacion IS NULL
       OR pg_catalog.octet_length(p_evidencia_verificacion)
          NOT BETWEEN 1 AND 262144
       OR p_raiz_publica_spki IS NULL
       OR pg_catalog.octet_length(p_raiz_publica_spki) <> 44 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;

    v_material := ROW(
        p_capacidad_canonica, p_decision_canonica,
        p_motivo_canonico, p_contexto_actor_canonico,
        p_persona_version, p_perfil_version, p_payload_vec_ad_3,
        p_sobre_cose_sign_1, p_evidencia_verificacion, p_raiz_publica_spki
    )::vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3;
    PERFORM vec_contratacion_temporal.acreditar_contexto_motor_consultas_rrhh_v1(
        p_alcance, v_material
    );
    v_consulta_canonica :=
        vec_contratacion_temporal.canon_consulta_detalle_rrhh_v1(p_consulta);
    v_capacidad := pg_catalog.convert_from(p_capacidad_canonica, 'UTF8')::jsonb;
    v_decision := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    v_contexto_actor := pg_catalog.convert_from(
        p_contexto_actor_canonico, 'UTF8')::jsonb;

    -- El orden es el canónico V3. Ni un campo más ni uno menos, incluidos
    -- duplicados y permutaciones, dan acceso a esta proyección nominal.
    IF v_decision -> 'campos_permitidos' IS DISTINCT FROM
       '["actuaciones","alcance","eficacia_administrativa",'
       '"ejercicio_sintetico","esquema","estado_clave",'
       '"expediente_ref","firma_oficial","periodo",'
       '"recibo_incorporacion_ref","registrado_en","seguimiento_ref",'
       '"version_expediente","version_seguimiento"]'::jsonb THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"ambito_ref":"' || p_alcance.ambito_ref
        || '","clase_ambito":"' || p_alcance.clase_ambito
        || '","organizacion_ref":"' || p_alcance.organizacion_ref
        || '"},"atributos":{"consulta_dominio":"'
        || 'vec.contratacion_temporal.consulta_rrhh.detalle.v1'
        || '","consulta_huella_sha256":"'
        || pg_catalog.encode(pg_catalog.sha256(v_consulta_canonica), 'hex')
        || '"}}', 'UTF8'
    );
    v_contexto_huella := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex');
    IF v_capacidad ->> 'audiencia_consumo' IS DISTINCT FROM
           'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
       OR v_capacidad ->> 'operacion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_capacidad ->> 'efecto_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_capacidad ->> 'huella_efecto_sha256' IS DISTINCT FROM v_contexto_huella
       OR v_capacidad ->> 'huella_decision_sha256' IS DISTINCT FROM
          pg_catalog.encode(pg_catalog.sha256(p_decision_canonica), 'hex')
       OR v_decision ->> 'accion' IS DISTINCT FROM
          'contratacion_temporal.expediente.consultar'
       OR v_decision ->> 'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
       OR v_decision ->> 'tipo_recurso' IS DISTINCT FROM
          'expediente_contratacion_temporal'
       OR v_decision ->> 'finalidad' IS DISTINCT FROM
          'tramitacion_expediente_contratacion_temporal'
       OR v_decision ->> 'recurso_ref' IS DISTINCT FROM
          p_consulta.expediente_ref
       OR v_decision ->> 'contexto_recurso_huella_sha256' IS DISTINCT FROM
          v_contexto_huella
       OR v_decision ->> 'principal_id' IS DISTINCT FROM
          v_contexto_actor ->> 'principal_ref'
       OR v_decision ->> 'perfil_activo_ref' IS DISTINCT FROM
          v_contexto_actor ->> 'perfil_activo_ref'
       OR v_contexto_actor ->> 'persona_version' IS DISTINCT FROM
          p_persona_version::text
       OR v_contexto_actor ->> 'perfil_version' IS DISTINCT FROM
          p_perfil_version::text THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;

    -- AD3 registra y consume la decisión y su auditoría en esta transacción.
    -- Solo después se lee la publicación mínima, sin tocar agregado_json.
    v_consumo :=
        vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(
            'detalle', v_material);
    IF v_consumo.decision_ref IS DISTINCT FROM v_decision ->> 'decision_ref'
       OR v_consumo.efecto_ref IS DISTINCT FROM p_consulta.expediente_ref
       OR v_consumo.huella_efecto_sha256 IS DISTINCT FROM v_contexto_huella
       OR v_consumo.auditoria_ref IS NULL
       OR v_consumo.auditoria_huella_sha256 IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;
    SELECT ultimo_corte INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       1 AND 9007199254740991::numeric
       OR v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
    END IF;
    WITH elegida AS MATERIALIZED (
        SELECT p.expediente_ref, p.version, p.organizacion_ref,
               p.centro_ref, p.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh p
         WHERE p.expediente_ref = p_consulta.expediente_ref
           AND p.corte_global <= v_corte_global
         ORDER BY p.corte_global DESC
         LIMIT 1
    )
    SELECT e.expediente_ref, e.version, e.organizacion_ref, e.unidad_ref
      INTO STRICT v_fila
      FROM elegida e
     WHERE e.organizacion_ref = p_alcance.organizacion_ref
       AND e.unidad_ref = p_unidad_ref
       AND (p_consulta.version_observada = 0
            OR p_consulta.version_observada = e.version)
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN e.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN e.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN e.unidad_ref = p_alcance.ambito_ref
           ELSE false END;
    RETURN QUERY SELECT v_fila.expediente_ref::text, v_fila.version::numeric,
        v_fila.organizacion_ref::text, v_fila.unidad_ref::text,
        v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref, v_consumo.auditoria_huella_sha256,
        v_consumo.consumida_en::timestamptz;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de seguimiento RRHH rechazada';
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    text, bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    text, bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    text, bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMENT ON FUNCTION vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    text, bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) IS 'Consulta nominal y mínima de seguimiento CT: campos V3 exactos, consumo y auditoría AD3 antes de leer publicación.';
COMMIT;
