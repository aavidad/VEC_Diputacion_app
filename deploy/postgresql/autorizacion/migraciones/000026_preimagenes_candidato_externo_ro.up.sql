\set ON_ERROR_STOP on
-- AUT26 es una extensión separada del paquete H6/SQL62. No modifica su journal.
-- Requiere AUT3 y AUT15/16/19/20 instaladas. No reaplicar ni ejecutar DOWN.
-- Estas lecturas son preimágenes para CAS; no conceden ni consumen autorización.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion:migracion:000026', 0));

DO $preimagen$
DECLARE t pg_catalog.text; f pg_catalog.record; p pg_catalog.pg_proc%ROWTYPE; estructura pg_catalog.text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
        WHERE rolname = current_user AND rolsuper)
       OR pg_catalog.current_setting('transaction_isolation') <> 'read committed'
       OR pg_catalog.to_regprocedure('vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_propietario' AND NOT rolcanlogin
              AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
              AND NOT rolreplication AND NOT rolbypassrls)
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_fuente_externa' AND NOT rolcanlogin
              AND rolinherit AND NOT rolsuper AND NOT rolcreatedb
              AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls
              AND rolconfig IS NULL)
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members
            WHERE member = pg_catalog.to_regrole('vec_autorizacion_fuente_externa'))
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace
            WHERE nspname = 'vec_autorizacion'
              AND nspowner = pg_catalog.to_regrole('vec_autorizacion_propietario'))
       OR NOT pg_catalog.has_schema_privilege('vec_autorizacion_fuente_externa', 'vec_autorizacion', 'USAGE')
       OR pg_catalog.has_schema_privilege('vec_autorizacion_fuente_externa', 'vec_autorizacion', 'CREATE')
       OR pg_catalog.to_regprocedure('vec_autorizacion.publicar_asignacion_candidato_externo_v1(bytea,text,bigint,text,text,text)') IS NULL
    THEN RAISE EXCEPTION 'AUT26: preimagen AUT16/roles incompatible' USING ERRCODE = '55000'; END IF;

    -- Cuerpos exactos del paquete SQL62, tras AUT19/20. No reconstruir helpers.
    FOR f IN SELECT * FROM (VALUES
        ('vec_autorizacion.login_candidato_externo_v1(text,text)',
         '250f5a9f72a02d8f83a205f8844ac0faa596e706ca04d1c5ae39a99df73e5a4d', true, 's'),
        ('vec_autorizacion.rol_candidato_externo_acotado_v1(jsonb)',
         '4212e0388ee883840011ea7874079e24eac5c947b2e8cc3c2f71921e613965da', false, 'i')
    ) AS exactas(firma, huella, definidor, volatilidad) LOOP
        SELECT * INTO p FROM pg_catalog.pg_proc WHERE oid = pg_catalog.to_regprocedure(f.firma);
        IF NOT FOUND OR p.proowner IS DISTINCT FROM pg_catalog.to_regrole('vec_autorizacion_propietario')
           OR p.prosecdef IS DISTINCT FROM f.definidor
           OR p.provolatile::pg_catalog.text IS DISTINCT FROM f.volatilidad
           OR p.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp']::pg_catalog.text[]
           OR pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(p.prosrc, 'UTF8')), 'hex') IS DISTINCT FROM f.huella
           OR (SELECT count(*) FROM pg_catalog.aclexplode(p.proacl)) <> 1
           OR EXISTS (SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
                WHERE a.grantee <> p.proowner OR a.privilege_type <> 'EXECUTE' OR a.is_grantable)
        THEN RAISE EXCEPTION 'AUT26: helper SQL62 divergente %', f.firma USING ERRCODE = '55000'; END IF;
    END LOOP;

    FOREACH t IN ARRAY ARRAY[
        'version_rol', 'control_vigencia_version_rol', 'control_vigencia_version_rol_actual',
        'asignacion_perfil_externa', 'asignacion_perfil_actual_externa',
        'motivo_v2_checkpoint_origen', 'motivo_v2_evento_origen',
        'motivo_v2_catalogo_publicado', 'motivo_v2_retirada', 'motivo_v2_entrada'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c
            WHERE c.oid = pg_catalog.to_regclass('vec_autorizacion.' || t)
              AND c.relkind = 'r' AND c.relrowsecurity AND c.relforcerowsecurity
              AND c.relowner = pg_catalog.to_regrole('vec_autorizacion_propietario'))
           OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_policy q
                WHERE q.polrelid = pg_catalog.to_regclass('vec_autorizacion.' || t)
                  AND q.polname = 'acceso_propietario_exacto' AND q.polcmd = '*'
                  AND q.polroles = ARRAY[pg_catalog.to_regrole('vec_autorizacion_propietario')::pg_catalog.oid]
                  AND pg_catalog.pg_get_expr(q.polqual, q.polrelid) = '(CURRENT_USER = ''vec_autorizacion_propietario''::name)'
                  AND pg_catalog.pg_get_expr(q.polwithcheck, q.polrelid) = '(CURRENT_USER = ''vec_autorizacion_propietario''::name)')
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_policy q
                WHERE q.polrelid = pg_catalog.to_regclass('vec_autorizacion.' || t)
                  AND q.polroles <> ARRAY[pg_catalog.to_regrole('vec_autorizacion_propietario')::pg_catalog.oid])
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_class c,
                LATERAL pg_catalog.aclexplode(coalesce(c.relacl, pg_catalog.acldefault('r', c.relowner))) a
                WHERE c.oid = pg_catalog.to_regclass('vec_autorizacion.' || t) AND a.grantee = 0)
           OR pg_catalog.has_table_privilege('vec_autorizacion_fuente_externa', 'vec_autorizacion.' || t, 'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
           OR pg_catalog.has_any_column_privilege('vec_autorizacion_fuente_externa', 'vec_autorizacion.' || t, 'SELECT,INSERT,UPDATE,REFERENCES')
        THEN RAISE EXCEPTION 'AUT26: tabla/ACL/RLS incompatible %', t USING ERRCODE = '55000'; END IF;
    END LOOP;

    -- Estructuras de H1→SQL62 sintético PG18.4: columnas/defaults, restricciones,
    -- índices y disparadores. Los datos y OID no forman parte de estas huellas.
    FOR f IN SELECT * FROM (VALUES
        ('asignacion_perfil_actual_externa','0093cea9a9d625678119965cfad670e85a50e9b91af768f72199e231d43e6ddd'),
        ('asignacion_perfil_externa','de3e5aeb2ca6e8e762bc31f40d566a417a44a2cdfebba41781bebb7f2655a108'),
        ('control_vigencia_version_rol','0a18b318e03c82de66d6d596dcf22596298de3c62fdb77edb3e7c5ac01ed8895'),
        ('control_vigencia_version_rol_actual','e14124895a1ee3daa8ec40b27c09e94d4195561913d96865609c3a17e2ccc4d8'),
        ('motivo_v2_catalogo_publicado','02579a12ba9269344fc1d220a68627fd9173f0ead3e33e322c519642ec9b682e'),
        ('motivo_v2_checkpoint_origen','e219a2c3f26f626824d2cd1fd9574fef4b5e2a1f4f230539ab2a0a0027b4de16'),
        ('motivo_v2_entrada','2c0f5840e77e729a7094b51dbcb451dfc9bc117197a23fb7c743462d744322d2'),
        ('motivo_v2_evento_origen','fab4b8c27c74affdcc13729350a8f45785cf3b0d530e9dca77c47b7185ecd64a'),
        ('motivo_v2_retirada','87360ffebe5b324534c8f860b20d55b39053c9104a9d451c9525ede21d97eae8'),
        ('version_rol','0d1938a88a833b3f085aa77fb38e6a6e965c0f458762750136d304b45f7aeb0d')
    ) AS exactas(tabla, huella) LOOP
        SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.jsonb_build_object(
            'columns', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(a.attnum, a.attname,
                pg_catalog.format_type(a.atttypid,a.atttypmod), a.attnotnull, a.attidentity, a.attgenerated,
                pg_catalog.pg_get_expr(d.adbin,d.adrelid)) ORDER BY a.attnum)
                FROM pg_catalog.pg_attribute a LEFT JOIN pg_catalog.pg_attrdef d
                  ON d.adrelid=a.attrelid AND d.adnum=a.attnum
                WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped),
            'constraints', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(q.conname,q.contype,
                pg_catalog.pg_get_constraintdef(q.oid,true)) ORDER BY q.conname)
                FROM pg_catalog.pg_constraint q WHERE q.conrelid=c.oid),
            'indexes', (SELECT pg_catalog.jsonb_agg(pg_catalog.pg_get_indexdef(i.indexrelid)
                ORDER BY i.indexrelid::pg_catalog.regclass::pg_catalog.text) FROM pg_catalog.pg_index i WHERE i.indrelid=c.oid),
            'triggers', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_array(tr.tgname,tr.tgenabled,
                pg_catalog.pg_get_triggerdef(tr.oid,true)) ORDER BY tr.tgname)
                FROM pg_catalog.pg_trigger tr WHERE tr.tgrelid=c.oid AND NOT tr.tgisinternal)
        )::pg_catalog.text,'UTF8')),'hex') INTO estructura FROM pg_catalog.pg_class c
         WHERE c.oid=pg_catalog.to_regclass('vec_autorizacion.'||f.tabla);
        IF estructura IS DISTINCT FROM f.huella THEN
            RAISE EXCEPTION 'AUT26: estructura SQL62 divergente %', f.tabla USING ERRCODE='55000';
        END IF;
    END LOOP;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_propietario;

-- Una llamada usa una única instantánea MVCC (STABLE), también bajo READ COMMITTED.
-- La identidad técnica es la fuente AUT15, nunca el publicador ni el portal.
-- No se consulta ContextoActor: la ausencia anterior al alta debe ser observable.
CREATE FUNCTION vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(
    p_rol_id pg_catalog.text, p_version_rol_ref pg_catalog.text, p_principal_id pg_catalog.text, p_perfil_activo_ref pg_catalog.text)
RETURNS pg_catalog.jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog
AS $f$
DECLARE r pg_catalog.record; c pg_catalog.record; a pg_catalog.record; ahora pg_catalog.timestamptz := pg_catalog.statement_timestamp();
        n_roles pg_catalog.int8; max_rol pg_catalog.int8; n_controles pg_catalog.int8; max_control pg_catalog.numeric;
        n_asignaciones pg_catalog.int8; max_asignacion pg_catalog.int8; n_principales pg_catalog.int8;
        estado_rol pg_catalog.text; estado_control pg_catalog.text; estado_asignacion pg_catalog.text;
BEGIN
    IF pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR vec_autorizacion.login_candidato_externo_v1(
            'vec_externo_v3_fuente_autorizacion_desarrollo', 'vec_autorizacion_fuente_externa') IS NOT TRUE
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = session_user AND rolconfig IS NOT NULL)
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_fuente_externa' AND NOT rolcanlogin AND rolinherit
              AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
              AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL)
       OR p_rol_id IS NULL OR p_rol_id NOT IN
            ('candidato_bolsa_historial_propio_desarrollo', 'candidato_bolsa_portal_historial_propio_desarrollo')
       OR vec_autorizacion.texto_positivo_valido(p_version_rol_ref, 512) IS NOT TRUE
       OR p_version_rol_ref !~ ('^rol:' || p_rol_id || ':v[1-9][0-9]{0,18}$')
       OR pg_catalog.substring(p_version_rol_ref, ':v([0-9]+)$')::pg_catalog.numeric > 9223372036854775807
       OR vec_autorizacion.texto_positivo_valido(p_principal_id, 512) IS NOT TRUE
       OR vec_autorizacion.texto_positivo_valido(p_perfil_activo_ref, 512) IS NOT TRUE
    THEN RAISE EXCEPTION 'AUT26: lectura externa rechazada' USING ERRCODE = '42501'; END IF;

    SELECT count(*), max(version) INTO n_roles, max_rol
      FROM vec_autorizacion.version_rol WHERE rol_id = p_rol_id;
    SELECT version_rol_ref, rol_id, version, huella_sha256, documento ->> 'estado' AS estado,
           vec_autorizacion.rol_candidato_externo_acotado_v1(documento) AS acotado
      INTO r FROM vec_autorizacion.version_rol WHERE version_rol_ref = p_version_rol_ref;
    SELECT count(*), max(revision) INTO n_controles, max_control
      FROM vec_autorizacion.control_vigencia_version_rol WHERE version_rol_ref = p_version_rol_ref;
    SELECT ca.revision AS revision_puntero, cv.revision, cv.estado, cv.huella_sha256
      INTO c FROM vec_autorizacion.control_vigencia_version_rol_actual ca
      LEFT JOIN vec_autorizacion.control_vigencia_version_rol cv
        ON cv.version_rol_ref = ca.version_rol_ref AND cv.revision = ca.revision
     WHERE ca.version_rol_ref = p_version_rol_ref;
    estado_rol := CASE WHEN r.version_rol_ref IS NULL AND n_controles = 0 AND c.revision_puntero IS NULL THEN 'ausente'
        WHEN r.version_rol_ref IS NULL OR r.rol_id IS DISTINCT FROM p_rol_id THEN 'inconsistente'
        WHEN r.estado = 'retirada' THEN 'retirada'
        WHEN r.acotado IS NOT TRUE THEN 'inconsistente' ELSE 'publicada' END;
    estado_control := CASE WHEN c.revision_puntero IS NULL AND n_controles = 0 THEN 'ausente'
        WHEN c.revision IS NULL OR c.revision IS DISTINCT FROM max_control THEN 'inconsistente'
        ELSE c.estado END;

    -- No filtrar por principal: una colisión de perfil o historia sin puntero
    -- no puede confundirse con la ausencia auténtica. No se devuelve el otro principal.
    SELECT count(*), max(version), count(*) FILTER (WHERE principal_id <> p_principal_id)
      INTO n_asignaciones, max_asignacion, n_principales
      FROM vec_autorizacion.asignacion_perfil_externa WHERE perfil_activo_ref = p_perfil_activo_ref;
    SELECT aa.asignacion_ref AS ref_puntero, ap.asignacion_ref, ap.principal_id, ap.version,
           ap.huella_sha256, ap.version_rol_ref, ap.documento ->> 'estado' AS estado,
           (ap.documento ->> 'vigente_desde')::pg_catalog.timestamptz AS desde,
           (ap.documento ->> 'vigente_hasta')::pg_catalog.timestamptz AS hasta
      INTO a FROM vec_autorizacion.asignacion_perfil_actual_externa aa
      LEFT JOIN vec_autorizacion.asignacion_perfil_externa ap
        ON ap.perfil_activo_ref = aa.perfil_activo_ref AND ap.asignacion_ref = aa.asignacion_ref
     WHERE aa.perfil_activo_ref = p_perfil_activo_ref;
    estado_asignacion := CASE WHEN a.ref_puntero IS NULL AND n_asignaciones = 0 THEN 'ausente'
        WHEN a.asignacion_ref IS NULL OR n_principales <> 0
          OR a.principal_id IS DISTINCT FROM p_principal_id OR a.version IS DISTINCT FROM max_asignacion
          OR a.version_rol_ref IS DISTINCT FROM p_version_rol_ref THEN 'inconsistente'
        WHEN a.estado = 'revocada' THEN 'revocada'
        WHEN a.estado IS DISTINCT FROM 'activa' OR a.desde IS NULL OR a.hasta IS NULL
          OR NOT pg_catalog.isfinite(a.desde) OR NOT pg_catalog.isfinite(a.hasta)
          OR a.hasta <= a.desde THEN 'inconsistente'
        WHEN ahora < a.desde OR ahora >= a.hasta THEN 'no_vigente' ELSE 'activa' END;
    RETURN pg_catalog.jsonb_build_object(
        'contrato', 'preimagen_candidato_externo_ro_v1', 'observada_en', ahora,
        'rol_id', p_rol_id, 'version_rol_ref', p_version_rol_ref,
        'principal_id', p_principal_id, 'perfil_activo_ref', p_perfil_activo_ref,
        'rol', pg_catalog.jsonb_build_object('estado_observacion', estado_rol,
            'version', r.version, 'huella_sha256', r.huella_sha256, 'estado', r.estado,
            'versiones', n_roles, 'version_maxima', max_rol),
        'control_rol', pg_catalog.jsonb_build_object('estado_observacion', estado_control,
            'revision', c.revision, 'huella_sha256', c.huella_sha256, 'estado', c.estado,
            'revisiones', n_controles, 'revision_maxima', max_control),
        'asignacion', pg_catalog.jsonb_build_object('estado_observacion', estado_asignacion,
            'versiones', n_asignaciones, 'version_maxima', max_asignacion,
            'asignacion_ref', CASE WHEN a.principal_id = p_principal_id THEN a.asignacion_ref END,
            'version', CASE WHEN a.principal_id = p_principal_id THEN a.version END,
            'huella_sha256', CASE WHEN a.principal_id = p_principal_id THEN a.huella_sha256 END,
            'estado', CASE WHEN a.principal_id = p_principal_id THEN a.estado END,
            'vigente_desde', CASE WHEN a.principal_id = p_principal_id THEN a.desde END,
            'vigente_hasta', CASE WHEN a.principal_id = p_principal_id THEN a.hasta END));
END $f$;

-- Sin parámetros: se observan siempre los tres catálogos nominales completos.
-- Las versiones retiradas siguen presentes, con su secuencia y huella propias.
CREATE FUNCTION vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()
RETURNS pg_catalog.jsonb LANGUAGE plpgsql STABLE SECURITY DEFINER SET search_path = pg_catalog
AS $f$
DECLARE resultado pg_catalog.jsonb;
BEGIN
    IF pg_catalog.current_setting('role') <> 'none'
       OR pg_catalog.current_setting('transaction_read_only') <> 'on'
       OR vec_autorizacion.login_candidato_externo_v1(
            'vec_externo_v3_fuente_autorizacion_desarrollo', 'vec_autorizacion_fuente_externa') IS NOT TRUE
       OR EXISTS (SELECT 1 FROM pg_catalog.pg_roles WHERE rolname = session_user AND rolconfig IS NOT NULL)
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles
            WHERE rolname = 'vec_autorizacion_fuente_externa' AND NOT rolcanlogin AND rolinherit
              AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole
              AND NOT rolreplication AND NOT rolbypassrls AND rolconfig IS NULL)
    THEN RAISE EXCEPTION 'AUT26: lectura externa rechazada' USING ERRCODE = '42501'; END IF;
    IF (SELECT count(*) FROM vec_autorizacion.motivo_v2_catalogo_publicado WHERE catalogo_id IN
        ('motivos_mi_bolsa_desarrollo', 'motivos_historial_mi_bolsa_desarrollo', 'motivos_portal_mi_bolsa_desarrollo')) > 4096
       OR (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada WHERE catalogo_id IN
        ('motivos_mi_bolsa_desarrollo', 'motivos_historial_mi_bolsa_desarrollo', 'motivos_portal_mi_bolsa_desarrollo')) > 65536
    THEN RAISE EXCEPTION 'AUT26: observación excede cardinalidad admitida' USING ERRCODE = '54000'; END IF;

    WITH checkpoint AS (
        SELECT cp.*, (SELECT count(*) FROM vec_autorizacion.motivo_v2_evento_origen) AS eventos,
            (SELECT max(secuencia_origen) FROM vec_autorizacion.motivo_v2_evento_origen) AS maxima,
            (SELECT count(*) FROM vec_autorizacion.motivo_v2_checkpoint_origen) AS controles
          FROM vec_autorizacion.motivo_v2_checkpoint_origen cp WHERE cp.control_id
    ), permitidos(id) AS (VALUES ('motivos_mi_bolsa_desarrollo'),
        ('motivos_historial_mi_bolsa_desarrollo'), ('motivos_portal_mi_bolsa_desarrollo')),
    catalogos AS (
        SELECT ids.id, count(pub.catalogo_version) AS versiones,
            coalesce(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
                'version', pub.catalogo_version, 'huella_publicada_sha256', pub.catalogo_huella_publicada_sha256,
                'publicado_en', pub.publicado_en, 'evento_publicacion_ref', pub.evento_origen_ref,
                'secuencia_publicacion', pub.secuencia_origen,
                'huella_evento_publicacion_sha256', ep.huella_evento_sha256,
                'entradas', (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada e
                    WHERE e.catalogo_id = pub.catalogo_id AND e.catalogo_version = pub.catalogo_version),
                'entradas_vigentes', (SELECT count(*) FROM vec_autorizacion.motivo_v2_entrada e
                    WHERE e.catalogo_id = pub.catalogo_id AND e.catalogo_version = pub.catalogo_version
                      AND e.vigente_desde <= pg_catalog.statement_timestamp()
                      AND (e.vigente_hasta IS NULL OR e.vigente_hasta > pg_catalog.statement_timestamp())),
                'retirada', ret.catalogo_id IS NOT NULL,
                'huella_retirada_sha256', ret.catalogo_huella_retirada_sha256,
                'retirado_en', ret.retirado_en, 'evento_retirada_ref', ret.evento_origen_ref,
                'secuencia_retirada', ret.secuencia_origen, 'huella_evento_retirada_sha256', er.huella_evento_sha256,
                'coherente', ep.tipo_evento = 'publicacion' AND ep.catalogo_id = pub.catalogo_id
                    AND ep.catalogo_version = pub.catalogo_version
                    AND (ret.catalogo_id IS NULL OR (er.tipo_evento = 'retirada'
                        AND er.catalogo_id = ret.catalogo_id AND er.catalogo_version = ret.catalogo_version))
            ) ORDER BY pub.catalogo_version) FILTER (WHERE pub.catalogo_version IS NOT NULL), '[]'::pg_catalog.jsonb) AS observaciones
          FROM permitidos ids LEFT JOIN vec_autorizacion.motivo_v2_catalogo_publicado pub ON pub.catalogo_id = ids.id
          LEFT JOIN vec_autorizacion.motivo_v2_evento_origen ep
            ON ep.secuencia_origen = pub.secuencia_origen AND ep.evento_origen_ref = pub.evento_origen_ref
          LEFT JOIN vec_autorizacion.motivo_v2_retirada ret
            ON ret.catalogo_id = pub.catalogo_id AND ret.catalogo_version = pub.catalogo_version
          LEFT JOIN vec_autorizacion.motivo_v2_evento_origen er
            ON er.secuencia_origen = ret.secuencia_origen AND er.evento_origen_ref = ret.evento_origen_ref
         GROUP BY ids.id
    )
    SELECT pg_catalog.jsonb_build_object('contrato', 'checkpoint_motivos_candidato_externo_ro_v1',
        'observada_en', pg_catalog.statement_timestamp(),
        'checkpoint', pg_catalog.jsonb_build_object('controles', cp.controles,
            'ultima_secuencia', cp.ultima_secuencia, 'ultimo_evento_ref', cp.ultimo_evento_ref,
            'ultima_huella_evento_sha256', cp.ultima_huella_evento_sha256,
            'actualizado_en', cp.actualizado_en, 'eventos', cp.eventos,
            'secuencia_maxima', coalesce(cp.maxima, 0),
            'coherente', cp.controles = 1 AND cp.eventos = cp.ultima_secuencia
                AND coalesce(cp.maxima, 0) = cp.ultima_secuencia
                AND (cp.ultima_secuencia = 0 OR EXISTS (
                    SELECT 1 FROM vec_autorizacion.motivo_v2_evento_origen e
                    WHERE e.secuencia_origen = cp.ultima_secuencia AND e.evento_origen_ref = cp.ultimo_evento_ref
                      AND e.huella_evento_sha256 = cp.ultima_huella_evento_sha256))),
        'catalogos', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
            'catalogo_id', cat.id, 'versiones', cat.versiones, 'observaciones', cat.observaciones)
            ORDER BY cat.id) FROM catalogos cat)) INTO resultado FROM checkpoint cp;
    IF resultado IS NULL THEN
        RAISE EXCEPTION 'AUT26: checkpoint ausente o inconsistente' USING ERRCODE = '55000';
    END IF;
    RETURN resultado;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text),
    vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text),
    vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1() TO vec_autorizacion_fuente_externa;

DO $postimagen$
DECLARE f pg_catalog.oid;
BEGIN
    FOREACH f IN ARRAY ARRAY[
        'vec_autorizacion.obtener_preimagen_candidato_externo_ro_v1(text,text,text,text)'::pg_catalog.regprocedure::pg_catalog.oid,
        'vec_autorizacion.obtener_checkpoint_motivos_candidato_externo_ro_v1()'::pg_catalog.regprocedure::pg_catalog.oid
    ] LOOP
        IF (SELECT count(*) FROM pg_catalog.aclexplode((SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f))) <> 2
           OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p,
                LATERAL pg_catalog.aclexplode(p.proacl) a WHERE p.oid=f
                AND (a.grantee NOT IN ('vec_autorizacion_propietario'::pg_catalog.regrole,'vec_autorizacion_fuente_externa'::pg_catalog.regrole)
                    OR a.privilege_type <> 'EXECUTE' OR a.is_grantable))
        THEN RAISE EXCEPTION 'AUT26: postimagen ACL divergente' USING ERRCODE='55000'; END IF;
    END LOOP;
END $postimagen$;
COMMIT;
