\set ON_ERROR_STOP on
-- Sonda de solo lectura para el material V3 de la superficie interna de desarrollo.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000050a', 0));

DO $preimagen$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)') IS NOT NULL
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                   WHERE rolname = 'vec_autorizacion_atestada_v3_preflight_interno')
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.puntero_clave_emision') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.revocacion_clave_capacidad') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_confianza_version') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.puntero_configuracion_actual') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.revocacion_configuracion') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.raiz_confianza_version') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_raiz') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.revocacion_raiz') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.checkpoint_gobierno') IS NULL
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace AS n
            WHERE n.oid = 'vec_autorizacion_atestada_v3'::regnamespace
              AND n.nspowner = 'vec_autorizacion_atestada_v3_propietario'::regrole)
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS c
            WHERE c.oid = ANY (ARRAY[
                'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass,
                'vec_autorizacion_atestada_v3.puntero_clave_emision'::regclass,
                'vec_autorizacion_atestada_v3.revocacion_clave_capacidad'::regclass,
                'vec_autorizacion_atestada_v3.configuracion_confianza_version'::regclass,
                'vec_autorizacion_atestada_v3.puntero_configuracion_actual'::regclass,
                'vec_autorizacion_atestada_v3.revocacion_configuracion'::regclass,
                'vec_autorizacion_atestada_v3.raiz_confianza_version'::regclass,
                'vec_autorizacion_atestada_v3.configuracion_raiz'::regclass,
                'vec_autorizacion_atestada_v3.revocacion_raiz'::regclass,
                'vec_autorizacion_atestada_v3.checkpoint_gobierno'::regclass])
              AND c.relowner <> 'vec_autorizacion_atestada_v3_propietario'::regrole)
       OR EXISTS (
           SELECT 1 FROM (VALUES
               ('clave_capacidad_version','clave_id','text'),
               ('clave_capacidad_version','version','numeric'),
               ('clave_capacidad_version','revision_gobierno','numeric'),
               ('clave_capacidad_version','huella_gobierno_sha256','text'),
               ('clave_capacidad_version','huella_secreto_sha256','text'),
               ('clave_capacidad_version','emisor_id','text'),
               ('clave_capacidad_version','audiencia_consumo','text'),
               ('clave_capacidad_version','valida_desde','timestamp with time zone'),
               ('clave_capacidad_version','valida_hasta','timestamp with time zone'),
               ('puntero_clave_emision','clave_id','text'),
               ('puntero_clave_emision','version','numeric'),
               ('puntero_clave_emision','orden','numeric'),
               ('puntero_clave_emision','establecida_en','timestamp with time zone'),
               ('revocacion_clave_capacidad','clave_id','text'),
               ('revocacion_clave_capacidad','version','numeric'),
               ('revocacion_clave_capacidad','revocada_en','timestamp with time zone'),
               ('configuracion_confianza_version','revision','text'),
               ('configuracion_confianza_version','secuencia','numeric'),
               ('configuracion_confianza_version','huella_configuracion_sha256','text'),
               ('configuracion_confianza_version','publicada_en','timestamp with time zone'),
               ('configuracion_confianza_version','expira_en','timestamp with time zone'),
               ('puntero_configuracion_actual','configuracion_revision','text'),
               ('puntero_configuracion_actual','orden','numeric'),
               ('puntero_configuracion_actual','establecida_en','timestamp with time zone'),
               ('revocacion_configuracion','configuracion_revision','text'),
               ('revocacion_configuracion','revocada_en','timestamp with time zone'),
               ('raiz_confianza_version','clave_id','text'),
               ('raiz_confianza_version','version','numeric'),
               ('raiz_confianza_version','huella_spki_sha256','text'),
               ('raiz_confianza_version','audiencia_despliegue','text'),
               ('raiz_confianza_version','suite','text'),
               ('raiz_confianza_version','valida_desde','timestamp with time zone'),
               ('raiz_confianza_version','valida_hasta','timestamp with time zone'),
               ('configuracion_raiz','configuracion_revision','text'),
               ('configuracion_raiz','raiz_clave_id','text'),
               ('configuracion_raiz','raiz_version','numeric'),
               ('revocacion_raiz','raiz_clave_id','text'),
               ('revocacion_raiz','raiz_version','numeric'),
               ('revocacion_raiz','revocada_en','timestamp with time zone'),
               ('checkpoint_gobierno','control_id','boolean'),
               ('checkpoint_gobierno','revision','numeric'),
               ('checkpoint_gobierno','configuracion_secuencia_minima','numeric'),
               ('checkpoint_gobierno','raiz_version_minima','numeric')
           ) AS e(tabla,columna,tipo)
            WHERE NOT EXISTS (
                SELECT 1 FROM pg_catalog.pg_attribute AS a
                 WHERE a.attrelid = pg_catalog.to_regclass(
                     'vec_autorizacion_atestada_v3.' || e.tabla)
                   AND a.attname = e.columna
                   AND a.atttypid = pg_catalog.to_regtype(e.tipo)
                   AND a.attnum > 0 AND NOT a.attisdropped))
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS c
             CROSS JOIN LATERAL pg_catalog.aclexplode(
                 coalesce(c.relacl, pg_catalog.acldefault('r', c.relowner))) AS acl
            WHERE c.oid = ANY (ARRAY[
                'vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass,
                'vec_autorizacion_atestada_v3.raiz_confianza_version'::regclass])
              AND acl.grantee = 0 AND acl.privilege_type = 'SELECT')
    THEN
        RAISE EXCEPTION 'AD3-50a: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $preimagen$;

CREATE ROLE vec_autorizacion_atestada_v3_preflight_interno
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT
    NOREPLICATION NOBYPASSRLS;
DO $base$
BEGIN
    EXECUTE pg_catalog.format(
        'GRANT CONNECT ON DATABASE %I TO vec_autorizacion_atestada_v3_preflight_interno',
        pg_catalog.current_database());
END $base$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(
    p_material jsonb
) RETURNS boolean
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_audiencias CONSTANT text[] := ARRAY[
        'vec_personal.alta_ejercicio.v1',
        'vec_personal.lectura_incorporacion.v2',
        'vec_contratacion_temporal.incorporacion_ejercicio.v2',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'
    ];
    v_ahora timestamptz;
    v_claves jsonb;
    v_config jsonb;
    v_raiz jsonb;
    v_item jsonb;
    v_clave vec_autorizacion_atestada_v3.clave_capacidad_version%ROWTYPE;
    v_configuracion vec_autorizacion_atestada_v3.configuracion_confianza_version%ROWTYPE;
    v_raiz_fila vec_autorizacion_atestada_v3.raiz_confianza_version%ROWTYPE;
    v_checkpoint vec_autorizacion_atestada_v3.checkpoint_gobierno%ROWTYPE;
    v_indice integer;
BEGIN
    -- El LOGIN de arranque es distinto de los emisores y consumidores. El canal
    -- PostgreSQL usa TLS; la sonda de canal y la validación de CA/hostname
    -- pertenecen al pool privado antes de llamar a esta función.
    IF session_user <> 'vec_interno_preflight_v3_desarrollo'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles AS r
                       WHERE r.rolname = session_user
                         AND r.rolcanlogin AND NOT r.rolsuper
                         AND NOT r.rolcreatedb AND NOT r.rolcreaterole
                         AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR NOT pg_catalog.pg_has_role(session_user,
              'vec_autorizacion_atestada_v3_preflight_interno', 'MEMBER')
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = session_user::regrole) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = session_user::regrole
              AND m.roleid = 'vec_autorizacion_atestada_v3_preflight_interno'::regrole
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;

    IF pg_catalog.jsonb_typeof(p_material) <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <> 3
       OR NOT (p_material ?& ARRAY['claves', 'configuracion', 'raiz'])
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;
    v_claves := p_material -> 'claves';
    v_config := p_material -> 'configuracion';
    v_raiz := p_material -> 'raiz';
    IF pg_catalog.jsonb_typeof(v_claves) <> 'array'
       OR pg_catalog.jsonb_array_length(v_claves) <> 5
       OR pg_catalog.jsonb_typeof(v_config) <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_config)) <> 3
       OR NOT (v_config ?& ARRAY['revision', 'secuencia', 'huella_configuracion_sha256'])
       OR pg_catalog.jsonb_typeof(v_raiz) <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_raiz)) <> 5
       OR NOT (v_raiz ?& ARRAY['clave_id', 'version', 'huella_spki_sha256',
                                  'audiencia_despliegue', 'suite'])
       OR v_raiz ->> 'audiencia_despliegue' <> 'vec:desarrollo:contratacion-temporal:atestacion:v3'
       OR v_raiz ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(
                  v_config ->> 'huella_configuracion_sha256')
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(
                  v_raiz ->> 'huella_spki_sha256')
       OR (v_config ->> 'secuencia') !~ '^[1-9][0-9]{0,15}$'
       OR (v_raiz ->> 'version') !~ '^[1-9][0-9]{0,15}$'
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;

    v_ahora := pg_catalog.clock_timestamp();
    SELECT * INTO v_checkpoint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
     WHERE control_id = true;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;
    SELECT * INTO v_configuracion
      FROM vec_autorizacion_atestada_v3.configuracion_confianza_version
     WHERE revision = v_config ->> 'revision';
    IF NOT FOUND OR v_configuracion.secuencia IS DISTINCT FROM (v_config ->> 'secuencia')::numeric
       OR v_configuracion.secuencia < v_checkpoint.configuracion_secuencia_minima
       OR v_configuracion.huella_configuracion_sha256 IS DISTINCT FROM
          v_config ->> 'huella_configuracion_sha256'
       OR v_ahora < v_configuracion.publicada_en
       OR v_ahora >= v_configuracion.expira_en
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion AS r
                   WHERE r.configuracion_revision = v_configuracion.revision
                     AND r.revocada_en <= v_ahora)
       OR (SELECT p.configuracion_revision
             FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual AS p
            WHERE p.establecida_en <= v_ahora ORDER BY p.orden DESC LIMIT 1)
          IS DISTINCT FROM v_configuracion.revision
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;
    SELECT * INTO v_raiz_fila
      FROM vec_autorizacion_atestada_v3.raiz_confianza_version
     WHERE clave_id = v_raiz ->> 'clave_id'
       AND version = (v_raiz ->> 'version')::numeric;
    IF NOT FOUND OR v_raiz_fila.version < v_checkpoint.raiz_version_minima
       OR v_raiz_fila.huella_spki_sha256 IS DISTINCT FROM v_raiz ->> 'huella_spki_sha256'
       OR v_raiz_fila.audiencia_despliegue IS DISTINCT FROM v_raiz ->> 'audiencia_despliegue'
       OR v_raiz_fila.suite IS DISTINCT FROM v_raiz ->> 'suite'
       OR v_ahora < v_raiz_fila.valida_desde OR v_ahora >= v_raiz_fila.valida_hasta
       OR NOT EXISTS (
           SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz AS cr
            WHERE cr.configuracion_revision = v_configuracion.revision
              AND cr.raiz_clave_id = v_raiz_fila.clave_id
              AND cr.raiz_version = v_raiz_fila.version)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz AS r
                   WHERE r.raiz_clave_id = v_raiz_fila.clave_id
                     AND r.raiz_version = v_raiz_fila.version
                     AND r.revocada_en <= v_ahora)
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;

    FOR v_indice IN 0..4 LOOP
        v_item := v_claves -> v_indice;
        IF pg_catalog.jsonb_typeof(v_item) <> 'object'
           OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_item)) <> 7
           OR NOT (v_item ?& ARRAY['audiencia_consumo', 'clave_id', 'version',
                'revision_gobierno', 'huella_gobierno_sha256',
                'huella_secreto_sha256', 'emisor_id'])
           OR v_item ->> 'audiencia_consumo' IS DISTINCT FROM v_audiencias[v_indice + 1]
           OR (v_item ->> 'version') !~ '^[1-9][0-9]{0,15}$'
           OR (v_item ->> 'revision_gobierno') !~ '^[1-9][0-9]{0,15}$'
           OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(
                      v_item ->> 'huella_gobierno_sha256')
           OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(
                      v_item ->> 'huella_secreto_sha256')
        THEN
            RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
        END IF;
        SELECT * INTO v_clave
          FROM vec_autorizacion_atestada_v3.clave_capacidad_version
         WHERE clave_id = v_item ->> 'clave_id'
           AND version = (v_item ->> 'version')::numeric;
        IF NOT FOUND
           OR v_clave.revision_gobierno IS DISTINCT FROM
              (v_item ->> 'revision_gobierno')::numeric
           OR v_clave.revision_gobierno > v_checkpoint.revision
           OR v_clave.huella_gobierno_sha256 IS DISTINCT FROM
              v_item ->> 'huella_gobierno_sha256'
           OR v_clave.huella_secreto_sha256 IS DISTINCT FROM
              v_item ->> 'huella_secreto_sha256'
           OR v_clave.emisor_id IS DISTINCT FROM v_item ->> 'emisor_id'
           OR v_clave.audiencia_consumo IS DISTINCT FROM v_audiencias[v_indice + 1]
           OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
           OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad AS r
                       WHERE r.clave_id = v_clave.clave_id
                         AND r.version = v_clave.version
                         AND r.revocada_en <= v_ahora)
           OR (SELECT p.clave_id FROM vec_autorizacion_atestada_v3.puntero_clave_emision AS p
                 JOIN vec_autorizacion_atestada_v3.clave_capacidad_version AS k
                   ON k.clave_id = p.clave_id AND k.version = p.version
                WHERE k.audiencia_consumo = v_audiencias[v_indice + 1]
                  AND p.establecida_en <= v_ahora
                ORDER BY p.orden DESC LIMIT 1) IS DISTINCT FROM v_clave.clave_id
           OR (SELECT p.version FROM vec_autorizacion_atestada_v3.puntero_clave_emision AS p
                 JOIN vec_autorizacion_atestada_v3.clave_capacidad_version AS k
                   ON k.clave_id = p.clave_id AND k.version = p.version
                WHERE k.audiencia_consumo = v_audiencias[v_indice + 1]
                  AND p.establecida_en <= v_ahora
                ORDER BY p.orden DESC LIMIT 1) IS DISTINCT FROM v_clave.version
        THEN
            RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
END $funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)
    FROM PUBLIC, vec_autorizacion_atestada_v3_emisor, vec_autorizacion_atestada_v3_consumidor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_autorizacion_atestada_v3_preflight_interno;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)
    TO vec_autorizacion_atestada_v3_preflight_interno;
COMMIT;
