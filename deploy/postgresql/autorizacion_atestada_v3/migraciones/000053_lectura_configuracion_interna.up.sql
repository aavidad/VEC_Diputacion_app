\set ON_ERROR_STOP on
-- Lector de gobierno publicado para vec-interno. No publica ni altera gobierno.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000053', 0));

DO $preimagen$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
                    WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.to_regprocedure(
          'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure(
          'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)') IS NULL
       OR pg_catalog.to_regrole('vec_autorizacion_atestada_v3_preflight_interno') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.checkpoint_gobierno') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.configuracion_confianza_version') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.puntero_configuracion_actual') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.raiz_confianza_version') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.clave_capacidad_version') IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n
                       WHERE n.oid = 'vec_autorizacion_atestada_v3'::regnamespace
                         AND n.nspowner = 'vec_autorizacion_atestada_v3_propietario'::regrole)
    THEN
        RAISE EXCEPTION 'AD3-53: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(p_material jsonb)
RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER
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
    v_ahora timestamptz := pg_catalog.statement_timestamp();
    v_previa jsonb;
    v_raiz jsonb;
    v_claves jsonb;
    v_item jsonb;
    v_anterior vec_autorizacion_atestada_v3.configuracion_confianza_version%ROWTYPE;
    v_actual vec_autorizacion_atestada_v3.configuracion_confianza_version%ROWTYPE;
    v_raiz_fila vec_autorizacion_atestada_v3.raiz_confianza_version%ROWTYPE;
    v_clave vec_autorizacion_atestada_v3.clave_capacidad_version%ROWTYPE;
    v_checkpoint vec_autorizacion_atestada_v3.checkpoint_gobierno%ROWTYPE;
    v_indice integer;
BEGIN
    -- El LOGIN es nominal y no puede heredar SELECT sobre las tablas de gobierno.
    IF session_user <> 'vec_interno_preflight_v3_desarrollo'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname = session_user
            AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
            AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR NOT pg_catalog.pg_has_role(session_user,
            'vec_autorizacion_atestada_v3_preflight_interno', 'MEMBER')
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = session_user::regrole) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
            WHERE m.member = session_user::regrole
              AND m.roleid = 'vec_autorizacion_atestada_v3_preflight_interno'::regrole
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR pg_catalog.jsonb_typeof(p_material) <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <> 3
       OR NOT (p_material ?& ARRAY['claves','configuracion','raiz'])
    THEN
        RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
    END IF;
    v_claves := p_material -> 'claves';
    v_previa := p_material -> 'configuracion';
    v_raiz := p_material -> 'raiz';
    IF pg_catalog.jsonb_typeof(v_claves) <> 'array'
       OR pg_catalog.jsonb_array_length(v_claves) <> 5
       OR pg_catalog.jsonb_typeof(v_previa) <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_previa)) <> 3
       OR NOT (v_previa ?& ARRAY['revision','secuencia','huella_configuracion_sha256'])
       OR pg_catalog.jsonb_typeof(v_raiz) <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_raiz)) <> 5
       OR NOT (v_raiz ?& ARRAY['clave_id','version','huella_spki_sha256','audiencia_despliegue','suite'])
       OR (v_previa ->> 'secuencia') !~ '^[1-9][0-9]{0,15}$'
       OR (v_raiz ->> 'version') !~ '^[1-9][0-9]{0,15}$'
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(v_previa ->> 'huella_configuracion_sha256')
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(v_raiz ->> 'huella_spki_sha256')
       OR v_raiz ->> 'audiencia_despliegue' <> 'vec:desarrollo:contratacion-temporal:atestacion:v3'
       OR v_raiz ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'
    THEN
        RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
    END IF;

    SELECT * INTO v_checkpoint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno
      WHERE control_id = true;
    SELECT * INTO v_anterior FROM vec_autorizacion_atestada_v3.configuracion_confianza_version
      WHERE revision = v_previa ->> 'revision';
    IF v_checkpoint.control_id IS DISTINCT FROM true
       OR v_anterior.revision IS NULL
       OR v_anterior.secuencia IS DISTINCT FROM (v_previa ->> 'secuencia')::numeric
       OR v_anterior.huella_configuracion_sha256 IS DISTINCT FROM v_previa ->> 'huella_configuracion_sha256'
       OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
                       WHERE p.configuracion_revision = v_anterior.revision
                         AND p.orden = v_anterior.secuencia
                         AND p.establecida_en <= v_ahora)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
                   WHERE r.configuracion_revision = v_anterior.revision AND r.revocada_en <= v_ahora)
    THEN
        RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
    END IF;

    SELECT c.* INTO v_actual
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
      JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c
        ON c.revision = p.configuracion_revision
     WHERE p.establecida_en <= v_ahora
     ORDER BY p.orden DESC LIMIT 1;
    IF v_actual.revision IS NULL OR v_actual.secuencia < v_anterior.secuencia
       OR v_actual.secuencia < v_checkpoint.configuracion_secuencia_minima
       OR v_ahora < v_actual.publicada_en OR v_ahora >= v_actual.expira_en
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
                   WHERE r.configuracion_revision = v_actual.revision AND r.revocada_en <= v_ahora)
       OR (SELECT p.orden FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
            WHERE p.configuracion_revision = v_actual.revision) IS DISTINCT FROM v_actual.secuencia
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
            WHERE cr.configuracion_revision = v_actual.revision) <> 1
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
            WHERE cr.configuracion_revision = v_anterior.revision) <> 1
    THEN
        RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
    END IF;
    SELECT r.* INTO v_raiz_fila
      FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
      JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
        ON (r.clave_id,r.version)=(cr.raiz_clave_id,cr.raiz_version)
     WHERE cr.configuracion_revision = v_actual.revision;
    IF v_raiz_fila.clave_id IS NULL
       OR v_raiz_fila.clave_id IS DISTINCT FROM v_raiz ->> 'clave_id'
       OR v_raiz_fila.version IS DISTINCT FROM (v_raiz ->> 'version')::numeric
       OR v_raiz_fila.version < v_checkpoint.raiz_version_minima
       OR v_raiz_fila.huella_spki_sha256 IS DISTINCT FROM v_raiz ->> 'huella_spki_sha256'
       OR v_raiz_fila.audiencia_despliegue IS DISTINCT FROM v_raiz ->> 'audiencia_despliegue'
       OR v_raiz_fila.suite IS DISTINCT FROM v_raiz ->> 'suite'
       OR pg_catalog.encode(pg_catalog.sha256(v_raiz_fila.clave_publica_spki),'hex')
          IS DISTINCT FROM v_raiz_fila.huella_spki_sha256
       OR v_ahora < v_raiz_fila.valida_desde OR v_ahora >= v_raiz_fila.valida_hasta
       OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
                       WHERE cr.configuracion_revision = v_anterior.revision
                         AND cr.raiz_clave_id = v_raiz_fila.clave_id
                         AND cr.raiz_version = v_raiz_fila.version)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r
                   WHERE (r.raiz_clave_id,r.raiz_version)=(v_raiz_fila.clave_id,v_raiz_fila.version)
                     AND r.revocada_en <= v_ahora)
    THEN
        RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
    END IF;

    FOR v_indice IN 0..4 LOOP
        v_item := v_claves -> v_indice;
        IF pg_catalog.jsonb_typeof(v_item) <> 'object'
           OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_item)) <> 7
           OR NOT (v_item ?& ARRAY['audiencia_consumo','clave_id','version',
                 'revision_gobierno','huella_gobierno_sha256','huella_secreto_sha256','emisor_id'])
           OR v_item ->> 'audiencia_consumo' IS DISTINCT FROM v_audiencias[v_indice+1]
           OR (v_item ->> 'version') !~ '^[1-9][0-9]{0,15}$'
           OR (v_item ->> 'revision_gobierno') !~ '^[1-9][0-9]{0,15}$'
           OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(v_item ->> 'huella_gobierno_sha256')
           OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(v_item ->> 'huella_secreto_sha256')
        THEN
            RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
        END IF;
        SELECT * INTO v_clave FROM vec_autorizacion_atestada_v3.clave_capacidad_version
          WHERE clave_id = v_item ->> 'clave_id'
            AND version = (v_item ->> 'version')::numeric;
        IF v_clave.clave_id IS NULL
           OR v_clave.revision_gobierno IS DISTINCT FROM (v_item ->> 'revision_gobierno')::numeric
           OR v_clave.revision_gobierno > v_checkpoint.revision
           OR v_clave.huella_gobierno_sha256 IS DISTINCT FROM v_item ->> 'huella_gobierno_sha256'
           OR v_clave.huella_secreto_sha256 IS DISTINCT FROM v_item ->> 'huella_secreto_sha256'
           OR v_clave.emisor_id IS DISTINCT FROM v_item ->> 'emisor_id'
           OR v_clave.audiencia_consumo IS DISTINCT FROM v_audiencias[v_indice+1]
           OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
           OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
                       WHERE (r.clave_id,r.version)=(v_clave.clave_id,v_clave.version)
                         AND r.revocada_en <= v_ahora)
           OR (SELECT (p.clave_id,p.version) FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
                JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k
                  ON (k.clave_id,k.version)=(p.clave_id,p.version)
                WHERE k.audiencia_consumo=v_audiencias[v_indice+1] AND p.establecida_en<=v_ahora
                ORDER BY p.orden DESC LIMIT 1) IS DISTINCT FROM (v_clave.clave_id,v_clave.version)
        THEN
            RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
        END IF;
    END LOOP;
    RETURN pg_catalog.jsonb_build_object(
       'revision',v_actual.revision,'secuencia',v_actual.secuencia,
       'huella_configuracion_sha256',v_actual.huella_configuracion_sha256,
       'publicada_en',v_actual.publicada_en,'expira_en',v_actual.expira_en);
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
END $funcion$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)
    FROM PUBLIC, vec_autorizacion_atestada_v3_emisor, vec_autorizacion_atestada_v3_consumidor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)
    TO vec_autorizacion_atestada_v3_preflight_interno;
COMMIT;
