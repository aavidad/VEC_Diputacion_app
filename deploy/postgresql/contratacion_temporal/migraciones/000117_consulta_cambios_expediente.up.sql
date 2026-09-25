\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_contratacion_temporal:migracion:000117', 0)
);

-- Petición RRHH p.4: en el histórico del expediente, cada cambio de datos con
-- su valor anterior y el nuevo. expediente_version_integral ya guarda, de
-- solo adición, la instantánea completa de cada versión: la anterior es la
-- preimagen y la siguiente la postimagen. Esta migración no escribe historia;
-- añade una lectura que compara versiones consecutivas campo a campo.
--
-- Mismo permiso que el detalle completo (campos_permitidos vacío, garantía
-- alta, sin el rol de seguimiento) y mismo consumo AD3 'detalle' que la
-- consulta mínima de 000109: no abre otro consumidor de autorización.
--
-- Minimización por lista cerrada de campos (la hoja de la ruta), no por la
-- forma del valor: solo salen en claro los booleanos, las fechas de campos de
-- fecha ('inicio', 'fin', 'desde', 'hasta', 'fecha*' salvo la de nacimiento,
-- '*_en'), los números de gestión (solo estas hojas exactas: 'centimos',
-- 'porcentaje_jornada', 'orden', 'plazas', 'dias', 'meses', 'horas',
-- 'numero' de la retención de crédito, 'numero_visible' del expediente,
-- 'ginpix_numero' y 'numero_resolucion'; ningún otro campo con «numero»), los
-- códigos de catálogo ('*_clave', 'estado', 'fase', 'resultado', 'moneda',
-- 'tipo', 'modalidad', 'origen', 'grupo_subgrupo') y las referencias opacas
-- con espacio de nombres ('*_ref', 'referencia') que no identifican a una
-- persona (dni:, nif:, nie:, tel:, correo:, iban:, nss: y similares nunca, ni
-- ninguna referencia que contenga un fragmento con forma de DNI o NIE).
-- Además el valor debe tener la forma de su clase. Todo lo demás (texto
-- libre, observaciones, nombres, teléfonos, documentos, un número fuera de la
-- lista) sale como la marca «*protegido», sin huella: una huella sin sal de un
-- dato de pocas posibilidades sería reversible. Vacío y fecha cero cuentan
-- como sin valor. Es una invariante de protección de datos, no una regla de
-- negocio: solo restringe. La IP o el equipo no se registran: pendientes de
-- la política de seguridad.
--
-- Auditoría: la consulta consume la misma decisión 'detalle' que el detalle
-- completo (no hay consumidor propio de «cambios» y no se abre otro), así que
-- en la auditoría de AD3 ambas lecturas figuran como consulta del detalle del
-- expediente; la respuesta de cambios nunca contiene más que el detalle.
DO $prevalidacion$
BEGIN
    IF pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NOT NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.valor_traza_cambio_v1(text,jsonb)') IS NOT NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consultar_resumen_seguimiento_rrhh_atestado_v1(vec_contratacion_temporal.alcance_consulta_rrhh_v1,vec_contratacion_temporal.consulta_detalle_rrhh_v1,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ) IS NULL
    OR pg_catalog.to_regprocedure(
        'vec_contratacion_temporal.consumir_autorizacion_motor_consultas_rrhh_v1(text,vec_contratacion_temporal.material_autorizacion_consulta_rrhh_v3)'
    ) IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_version_integral') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE = '55000',
            MESSAGE = 'estado incompatible para la consulta de cambios del expediente';
    END IF;
END
$prevalidacion$;

CREATE FUNCTION vec_contratacion_temporal.valor_traza_cambio_v1(p_ruta text, p_valor jsonb)
RETURNS text
LANGUAGE plpgsql
STABLE
PARALLEL SAFE
SET search_path = pg_catalog
SET timezone = 'UTC'
AS $funcion$
DECLARE
    v_texto text;
    v_instante timestamptz;
    -- La hoja: último nombre de la ruta, sin índices.
    v_campo text := pg_catalog.lower(pg_catalog.substring(COALESCE(p_ruta, ''), '([A-Za-z_][A-Za-z0-9_]*)(\[[0-9]+\])*$'));
BEGIN
    IF p_valor IS NULL OR pg_catalog.jsonb_typeof(p_valor) = 'null'
       OR p_valor IN ('[]'::jsonb, '{}'::jsonb, '""'::jsonb) THEN
        RETURN NULL;
    END IF;
    IF pg_catalog.jsonb_typeof(p_valor) = 'boolean' THEN
        RETURN p_valor #>> '{}';
    END IF;
    IF pg_catalog.jsonb_typeof(p_valor) NOT IN ('number', 'string') OR v_campo IS NULL THEN
        RETURN '*protegido';
    END IF;
    v_texto := p_valor #>> '{}';
    -- El instante cero de Go es «sin valor» en cualquier campo.
    IF v_texto ~ '^0001-01-01(T00:00:00(\.0+)?Z)?$' THEN
        RETURN NULL;
    END IF;
    -- Fechas de campos de fecha: una sola forma canónica, para que un cambio
    -- de formato no parezca un cambio de dato; el instante cero de Go es
    -- «sin valor».
    IF (v_campo IN ('inicio', 'fin', 'desde', 'hasta') OR v_campo LIKE '%\_en'
        OR (v_campo LIKE 'fecha%' AND v_campo NOT LIKE '%nacimiento%') OR v_campo LIKE '%\_fecha') THEN
        IF pg_catalog.jsonb_typeof(p_valor) = 'string'
           AND v_texto ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}(T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2}))?$' THEN
            BEGIN
                v_instante := v_texto::timestamptz;
            EXCEPTION WHEN OTHERS THEN
                v_instante := NULL;
            END;
            IF v_instante IS NOT NULL THEN
                IF pg_catalog.date_part('year', v_instante) <= 1 THEN
                    RETURN NULL;
                END IF;
                RETURN pg_catalog.to_char(v_instante, 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
            END IF;
        END IF;
        RETURN '*protegido';
    END IF;
    -- Números de gestión: importes en céntimos, porcentajes, contadores y
    -- números de registro, nunca un número de persona. Lista cerrada de hojas
    -- exactas: un campo nuevo con «numero» en el nombre sale protegido hasta
    -- que se añada aquí expresamente.
    IF v_campo IN ('centimos', 'porcentaje_jornada', 'orden', 'plazas', 'dias', 'meses', 'horas',
                   'numero', 'numero_visible', 'ginpix_numero', 'numero_resolucion') THEN
        IF v_texto ~ '^-?[0-9]{1,20}(\.[0-9]{1,6})?$' THEN
            RETURN v_texto;
        END IF;
        RETURN '*protegido';
    END IF;
    IF pg_catalog.jsonb_typeof(p_valor) <> 'string' THEN
        RETURN '*protegido';
    END IF;
    -- Códigos de catálogo.
    IF v_campo LIKE '%\_clave' OR v_campo IN ('estado', 'fase', 'resultado', 'moneda', 'tipo', 'modalidad', 'origen', 'grupo_subgrupo') THEN
        IF v_texto ~ '^[A-Za-z][A-Za-z0-9_.-]{0,79}$' AND v_texto !~ '[0-9]{5}' THEN
            RETURN v_texto;
        END IF;
        RETURN '*protegido';
    END IF;
    -- Referencias opacas con espacio de nombres que no identifican a nadie.
    IF v_campo LIKE '%\_ref' OR v_campo = 'referencia' THEN
        IF pg_catalog.octet_length(v_texto) <= 160
           AND v_texto ~ '^[a-z][a-z0-9_.-]*(:[A-Za-z0-9_.#/-]+)+$'
           AND pg_catalog.split_part(v_texto, ':', 1) NOT IN ('dni', 'nif', 'nie', 'cif', 'pasaporte', 'tel', 'telefono',
               'movil', 'correo', 'email', 'mail', 'iban', 'cuenta', 'nss', 'naf', 'nombre', 'apellidos', 'domicilio', 'direccion')
           -- Ningún fragmento alfanumérico con forma de DNI (8 cifras y letra)
           -- o NIE (X/Y/Z, 7 cifras y letra), cualquiera que sea el prefijo.
           AND v_texto !~* '(^|[^a-z0-9])([0-9]{8}|[xyz][0-9]{7})[a-z]($|[^a-z0-9])' THEN
            RETURN v_texto;
        END IF;
        RETURN '*protegido';
    END IF;
    RETURN '*protegido';
END
$funcion$;
ALTER FUNCTION vec_contratacion_temporal.valor_traza_cambio_v1(text, jsonb)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.valor_traza_cambio_v1(text, jsonb) FROM PUBLIC;

-- Hojas de una instantánea: ruta con puntos e índices, valor escalar.
-- Se omite la contabilidad de la propia historia, ya visible como hitos o sin
-- significado para RRHH: 'version', 'actuaciones', 'creado_en' y
-- 'actualizado_en' de la raíz; y en cualquier nivel 'actuacion', 'canon',
-- 'actuacion_registro' y las huellas de integridad ('*_sha256').
CREATE FUNCTION vec_contratacion_temporal.hojas_instantanea_expediente_v1(p_agregado jsonb)
RETURNS TABLE (ruta text, valor jsonb)
LANGUAGE sql
IMMUTABLE
PARALLEL SAFE
SET search_path = pg_catalog
AS $funcion$
    WITH RECURSIVE nodo(ruta, valor) AS (
        SELECT e.key, e.value
          FROM pg_catalog.jsonb_each(p_agregado) e
         WHERE e.key NOT IN ('version', 'actuaciones', 'creado_en', 'actualizado_en')
           AND e.key !~ '_sha256$'
        UNION ALL
        SELECT n.ruta || CASE WHEN pg_catalog.jsonb_typeof(n.valor) = 'object'
                              THEN '.' || h.clave ELSE '[' || h.clave || ']' END,
               h.valor
          FROM nodo n
          CROSS JOIN LATERAL (
              SELECT o.key AS clave, o.value AS valor
                FROM pg_catalog.jsonb_each(CASE WHEN pg_catalog.jsonb_typeof(n.valor) = 'object' THEN n.valor END) o
               WHERE o.key NOT IN ('actuacion', 'actuacion_registro', 'canon')
                 AND o.key !~ '_sha256$'
              UNION ALL
              SELECT (a.ordinal - 1)::text, a.value
                FROM pg_catalog.jsonb_array_elements(CASE WHEN pg_catalog.jsonb_typeof(n.valor) = 'array' THEN n.valor END)
                     WITH ORDINALITY a(value, ordinal)
          ) h
         WHERE pg_catalog.length(n.ruta) < 400
    )
    SELECT n.ruta, n.valor FROM nodo n
     WHERE pg_catalog.jsonb_typeof(n.valor) NOT IN ('object', 'array')
        OR n.valor IN ('{}'::jsonb, '[]'::jsonb)
$funcion$;
ALTER FUNCTION vec_contratacion_temporal.hojas_instantanea_expediente_v1(jsonb)
    OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.hojas_instantanea_expediente_v1(jsonb) FROM PUBLIC;

CREATE FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
    p_alcance vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    p_consulta vec_contratacion_temporal.consulta_detalle_rrhh_v1,
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
    cambios jsonb,
    recortado boolean,
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
    v_version numeric(20, 0);
    v_cambios jsonb;
    v_recortado boolean;
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
       OR pg_catalog.octet_length(COALESCE(p_consulta.expediente_ref, '')) > 160 THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
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
            MESSAGE = 'consulta de cambios RRHH rechazada';
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

    -- El detalle completo, no una proyección limitada: la historia de
    -- cambios alcanza todos los campos del agregado.
    IF v_decision ->> 'garantia_minima' IS DISTINCT FROM 'alto'
       OR v_decision -> 'campos_permitidos' IS DISTINCT FROM '[]'::jsonb
       OR pg_catalog.left(COALESCE(v_decision ->> 'version_rol_ref', ''), 44) =
          'rol:rrhh_interno_certificado_seguimiento_ct_' THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
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
            MESSAGE = 'consulta de cambios RRHH rechazada';
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
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;
    SELECT ultimo_corte INTO STRICT v_corte_global
      FROM vec_contratacion_temporal.control_publicacion_rrhh
     WHERE control;
    IF v_corte_global IS NULL OR v_corte_global NOT BETWEEN
       1 AND 9007199254740991::numeric
       OR v_corte_global <> pg_catalog.trunc(v_corte_global) THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
    END IF;
    SELECT p.version INTO STRICT v_version
      FROM (
        SELECT p.expediente_ref, p.version, p.organizacion_ref,
               p.centro_ref, p.unidad_ref
          FROM vec_contratacion_temporal.publicacion_version_rrhh p
         WHERE p.expediente_ref = p_consulta.expediente_ref
           AND p.corte_global <= v_corte_global
         ORDER BY p.corte_global DESC
         LIMIT 1
      ) p
     WHERE p.organizacion_ref = p_alcance.organizacion_ref
       AND (p_consulta.version_observada = 0
            OR p_consulta.version_observada = p.version)
       AND CASE p_alcance.clase_ambito
           WHEN 'organizacion' THEN p.organizacion_ref = p_alcance.ambito_ref
           WHEN 'centro' THEN p.centro_ref = p_alcance.ambito_ref
           WHEN 'unidad_gestion' THEN p.unidad_ref = p_alcance.ambito_ref
           ELSE false END;

    -- Versiones consecutivas hasta la publicada. Se entregan como mucho 500
    -- cambios y como mucho 192 KiB de JSON acumulado (la respuesta HTTP
    -- admite 256 KiB); si hay más, «recortado» lo indica.
    WITH candidatos AS (
        SELECT n.version AS version_expediente,
               pg_catalog.to_char(n.registrada_en AT TIME ZONE 'UTC',
                   'YYYY-MM-DD"T"HH24:MI:SS.US"Z"') AS registrada_en,
               n.origen_version, n.operacion_ref,
               d.ruta,
               vec_contratacion_temporal.valor_traza_cambio_v1(d.ruta, d.anterior) AS valor_anterior,
               vec_contratacion_temporal.valor_traza_cambio_v1(d.ruta, d.nuevo) AS valor_nuevo
          FROM vec_contratacion_temporal.expediente_version_integral n
          JOIN vec_contratacion_temporal.expediente_version_integral a
            ON a.expediente_ref = n.expediente_ref AND a.version = n.version - 1
          CROSS JOIN LATERAL (
              SELECT COALESCE(x.ruta, y.ruta) AS ruta, x.valor AS anterior, y.valor AS nuevo
                FROM vec_contratacion_temporal.hojas_instantanea_expediente_v1(a.agregado_json) x
                FULL JOIN vec_contratacion_temporal.hojas_instantanea_expediente_v1(n.agregado_json) y
                  ON y.ruta = x.ruta
               WHERE x.valor IS DISTINCT FROM y.valor
          ) d
         WHERE n.expediente_ref = p_consulta.expediente_ref
           AND n.version <= v_version
         ORDER BY n.version, d.ruta
    ), visibles AS (
        -- Un null explícito y un campo ausente, o dos formas de la misma
        -- fecha, no son un cambio visible; dos valores protegidos distintos sí.
        SELECT c.*, pg_catalog.row_number() OVER w AS n,
               pg_catalog.sum(pg_catalog.octet_length(pg_catalog.to_jsonb(c)::text)) OVER w AS acumulado
          FROM candidatos c
         WHERE c.valor_anterior IS DISTINCT FROM c.valor_nuevo
            OR c.valor_anterior = '*protegido'
        WINDOW w AS (ORDER BY c.version_expediente, c.ruta)
         ORDER BY c.version_expediente, c.ruta
         LIMIT 501
    )
    SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) - 'n' - 'acumulado'
                        ORDER BY v.version_expediente, v.ruta)
                        FILTER (WHERE v.n <= 500 AND v.acumulado <= 196608), '[]'::jsonb),
           COALESCE(pg_catalog.bool_or(v.n > 500 OR v.acumulado > 196608), false)
      INTO v_cambios, v_recortado
      FROM visibles v;

    RETURN QUERY SELECT p_consulta.expediente_ref, v_version::numeric, v_cambios, v_recortado,
        v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref, v_consumo.auditoria_huella_sha256,
        v_consumo.consumida_en::timestamptz;
EXCEPTION
    WHEN SQLSTATE '40001' OR SQLSTATE '40P01'
      OR SQLSTATE '55P03' OR SQLSTATE '57014' THEN RAISE;
    WHEN OTHERS THEN
        RAISE EXCEPTION USING ERRCODE = '42501',
            MESSAGE = 'consulta de cambios RRHH rechazada';
END
$funcion$;

ALTER FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) OWNER TO vec_contratacion_temporal_propietario;
REVOKE ALL ON FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) TO vec_contratacion_temporal_consultor_rrhh;
COMMENT ON FUNCTION vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
    vec_contratacion_temporal.alcance_consulta_rrhh_v1,
    vec_contratacion_temporal.consulta_detalle_rrhh_v1,
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) IS 'Cambios del expediente CT entre versiones consecutivas con valor anterior y nuevo minimizados; permiso y consumo AD3 del detalle completo.';
COMMIT;
