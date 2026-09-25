\set ON_ERROR_STOP on
-- AD3-69: sonda de material y lectura de configuración V3 para la superficie
-- interna con un discriminador CERRADO de consumidor. Generaliza, sin
-- modificarlas, comprobar_material_emision_interna_v1 (AD3-50a) y
-- leer_configuracion_interna_v1 (AD3-53a), que siguen instaladas e intactas.
--
-- El llamante solo elige el consumidor ('ct' o 'personal_b2'); la lista de
-- audiencias, su cardinalidad y su orden los fija esta función. La raíz y la
-- audiencia de atestación son las compartidas del gobierno V3 vigente. Se
-- conservan login nominal, membresía exacta, punteros, revocaciones,
-- checkpoint, huellas y validación conjunta de configuración y claves.
-- Ambas funciones son STABLE: todas sus lecturas usan una sola instantánea,
-- de modo que una renovación concurrente no produce una vista mezclada. El
-- reloj es clock_timestamp(), como en la sonda v1: una sentencia larga del
-- llamante no puede validar una clave que caducó durante ella.
--
-- TRANSICIÓN DE ACL: el binario actual fija en funcionesEsperadasPerfil
-- (fabrica.go) y aprovisionar.py exactamente las dos funciones v1. Instalar
-- 000069 con ese binario impide abrir conexiones de preflight y deja caída la
-- composición interna, CT incluida; el binario con el manifiesto v1+v2 no
-- arranca sin 000069. Aplicar ambos en la misma ventana y solo tras ensayo en
-- clon.
--
-- Numeración: el bloque 000060–000069 quedó libre para B2 y Documentos
-- (comentario de AD3-70; Documentos usa 60/62, main usa 61). Se toma el
-- último del bloque para no chocar con correcciones secuenciales de
-- Documentos. No reescribe el núcleo: no toma el consultivo común.
-- Requiere AD3-50a y AD3-53a.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000069', 0));

DO $preimagen$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.huella_sha256_valida(text)') IS NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles AS r
                       WHERE r.rolname = 'vec_autorizacion_atestada_v3_preflight_interno'
                         AND NOT r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolbypassrls)
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles AS r
                       WHERE r.rolname = 'vec_autorizacion_atestada_v3_propietario'
                         AND NOT r.rolcanlogin)
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_namespace AS n
             JOIN pg_catalog.pg_roles AS r ON r.oid = n.nspowner
            WHERE n.nspname = 'vec_autorizacion_atestada_v3'
              AND r.rolname = 'vec_autorizacion_atestada_v3_propietario')
       OR EXISTS (
           SELECT 1 FROM (VALUES
               ('clave_capacidad_version'), ('puntero_clave_emision'),
               ('revocacion_clave_capacidad'), ('configuracion_confianza_version'),
               ('puntero_configuracion_actual'), ('revocacion_configuracion'),
               ('raiz_confianza_version'), ('configuracion_raiz'),
               ('revocacion_raiz'), ('checkpoint_gobierno')) AS t(tabla)
            WHERE NOT EXISTS (
                SELECT 1 FROM pg_catalog.pg_class AS c
                  JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
                  JOIN pg_catalog.pg_roles AS r ON r.oid = c.relowner
                 WHERE n.nspname = 'vec_autorizacion_atestada_v3'
                   AND c.relname = t.tabla AND c.relkind = 'r'
                   AND r.rolname = 'vec_autorizacion_atestada_v3_propietario'))
       -- El rol de preflight no puede leer directamente ninguna tabla de gobierno.
       OR EXISTS (
           SELECT 1 FROM pg_catalog.pg_class AS c
             JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
            WHERE n.nspname = 'vec_autorizacion_atestada_v3' AND c.relkind = 'r'
              AND pg_catalog.has_table_privilege(
                  'vec_autorizacion_atestada_v3_preflight_interno', c.oid,
                  'SELECT,INSERT,UPDATE,DELETE'))
    THEN
        RAISE EXCEPTION 'AD3-69: preimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $preimagen$;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(
    p_consumidor text,
    p_material jsonb
) RETURNS boolean
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_audiencias text[];
    v_ahora timestamptz := pg_catalog.clock_timestamp();
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
    -- Lista cerrada por consumidor: el llamante no aporta audiencias.
    v_audiencias := CASE p_consumidor
        WHEN 'ct' THEN ARRAY[
            'vec_personal.alta_ejercicio.v1',
            'vec_personal.lectura_incorporacion.v2',
            'vec_contratacion_temporal.incorporacion_ejercicio.v2',
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1']
        WHEN 'personal_b2' THEN ARRAY[
            'vec_personal.registro_empleado.ficha.v1',
            'vec_personal.registro_empleado.vacantes.v1',
            'vec_personal.registro_empleado.alta.v1',
            'vec_personal.registro_empleado.hecho.v1',
            'vec_personal.registro_empleado.catalogo.consultar.v1',
            'vec_personal.registro_empleado.catalogo.publicar.v1',
            'vec_personal.registro_empleado.catalogo.retirar.v1',
            'vec_personal.registro_empleado.empleados.v1']
        END;

    -- El LOGIN de arranque es nominal y distinto de emisores y consumidores.
    IF v_audiencias IS NULL
       OR session_user <> 'vec_interno_preflight_v3_desarrollo'
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
             JOIN pg_catalog.pg_roles AS g ON g.oid = m.roleid
            WHERE m.member = session_user::regrole
              AND g.rolname = 'vec_autorizacion_atestada_v3_preflight_interno'
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;

    IF pg_catalog.jsonb_typeof(p_material) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <> 3
       OR NOT (p_material ?& ARRAY['claves', 'configuracion', 'raiz'])
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;
    v_claves := p_material -> 'claves';
    v_config := p_material -> 'configuracion';
    v_raiz := p_material -> 'raiz';
    IF pg_catalog.jsonb_typeof(v_claves) IS DISTINCT FROM 'array'
       OR pg_catalog.jsonb_array_length(v_claves) <> pg_catalog.cardinality(v_audiencias)
       OR pg_catalog.jsonb_typeof(v_config) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_config)) <> 3
       OR NOT (v_config ?& ARRAY['revision', 'secuencia', 'huella_configuracion_sha256'])
       OR pg_catalog.jsonb_typeof(v_raiz) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_raiz)) <> 5
       OR NOT (v_raiz ?& ARRAY['clave_id', 'version', 'huella_spki_sha256',
                                  'audiencia_despliegue', 'suite'])
       OR (v_raiz ->> 'audiencia_despliegue') IS DISTINCT FROM
          'vec:desarrollo:contratacion-temporal:atestacion:v3'
       OR (v_raiz ->> 'suite') IS DISTINCT FROM 'VEC-AD-3-COSE-EDDSA-1'
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(
                  v_config ->> 'huella_configuracion_sha256')
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(
                  v_raiz ->> 'huella_spki_sha256')
       OR (v_config ->> 'secuencia') !~ '^[1-9][0-9]{0,15}$'
       OR (v_raiz ->> 'version') !~ '^[1-9][0-9]{0,15}$'
    THEN
        RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
    END IF;

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

    FOR v_indice IN 1..pg_catalog.cardinality(v_audiencias) LOOP
        v_item := v_claves -> (v_indice - 1);
        IF pg_catalog.jsonb_typeof(v_item) IS DISTINCT FROM 'object'
           OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_item)) <> 7
           OR NOT (v_item ?& ARRAY['audiencia_consumo', 'clave_id', 'version',
                'revision_gobierno', 'huella_gobierno_sha256',
                'huella_secreto_sha256', 'emisor_id'])
           OR v_item ->> 'audiencia_consumo' IS DISTINCT FROM v_audiencias[v_indice]
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
           OR v_clave.audiencia_consumo IS DISTINCT FROM v_audiencias[v_indice]
           OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
           OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad AS r
                       WHERE r.clave_id = v_clave.clave_id
                         AND r.version = v_clave.version
                         AND r.revocada_en <= v_ahora)
           OR (SELECT (p.clave_id, p.version)
                 FROM vec_autorizacion_atestada_v3.puntero_clave_emision AS p
                 JOIN vec_autorizacion_atestada_v3.clave_capacidad_version AS k
                   ON k.clave_id = p.clave_id AND k.version = p.version
                WHERE k.audiencia_consumo = v_audiencias[v_indice]
                  AND p.establecida_en <= v_ahora
                ORDER BY p.orden DESC LIMIT 1)
              IS DISTINCT FROM (v_clave.clave_id, v_clave.version)
        THEN
            RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION 'material de emisión interna rechazado' USING ERRCODE = '42501';
END $funcion$;

CREATE FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(
    p_consumidor text,
    p_material jsonb
) RETURNS jsonb
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path = pg_catalog
SET lock_timeout = '2s'
AS $funcion$
DECLARE
    v_audiencias text[];
    v_ahora timestamptz := pg_catalog.clock_timestamp();
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
    -- Misma lista cerrada que comprobar_material_emision_interna_v2.
    v_audiencias := CASE p_consumidor
        WHEN 'ct' THEN ARRAY[
            'vec_personal.alta_ejercicio.v1',
            'vec_personal.lectura_incorporacion.v2',
            'vec_contratacion_temporal.incorporacion_ejercicio.v2',
            'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
            'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1']
        WHEN 'personal_b2' THEN ARRAY[
            'vec_personal.registro_empleado.ficha.v1',
            'vec_personal.registro_empleado.vacantes.v1',
            'vec_personal.registro_empleado.alta.v1',
            'vec_personal.registro_empleado.hecho.v1',
            'vec_personal.registro_empleado.catalogo.consultar.v1',
            'vec_personal.registro_empleado.catalogo.publicar.v1',
            'vec_personal.registro_empleado.catalogo.retirar.v1',
            'vec_personal.registro_empleado.empleados.v1']
        END;

    -- El LOGIN es nominal y no puede heredar SELECT sobre las tablas de gobierno.
    IF v_audiencias IS NULL
       OR session_user <> 'vec_interno_preflight_v3_desarrollo'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname = session_user
            AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
            AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR NOT pg_catalog.pg_has_role(session_user,
            'vec_autorizacion_atestada_v3_preflight_interno', 'MEMBER')
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = session_user::regrole) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
              JOIN pg_catalog.pg_roles g ON g.oid = m.roleid
            WHERE m.member = session_user::regrole
              AND g.rolname = 'vec_autorizacion_atestada_v3_preflight_interno'
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR pg_catalog.jsonb_typeof(p_material) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <> 3
       OR NOT (p_material ?& ARRAY['claves','configuracion','raiz'])
    THEN
        RAISE EXCEPTION 'gobierno interno rechazado' USING ERRCODE = '42501';
    END IF;
    v_claves := p_material -> 'claves';
    v_previa := p_material -> 'configuracion';
    v_raiz := p_material -> 'raiz';
    IF pg_catalog.jsonb_typeof(v_claves) IS DISTINCT FROM 'array'
       OR pg_catalog.jsonb_array_length(v_claves) <> pg_catalog.cardinality(v_audiencias)
       OR pg_catalog.jsonb_typeof(v_previa) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_previa)) <> 3
       OR NOT (v_previa ?& ARRAY['revision','secuencia','huella_configuracion_sha256'])
       OR pg_catalog.jsonb_typeof(v_raiz) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_raiz)) <> 5
       OR NOT (v_raiz ?& ARRAY['clave_id','version','huella_spki_sha256','audiencia_despliegue','suite'])
       OR (v_previa ->> 'secuencia') !~ '^[1-9][0-9]{0,15}$'
       OR (v_raiz ->> 'version') !~ '^[1-9][0-9]{0,15}$'
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(v_previa ->> 'huella_configuracion_sha256')
       OR NOT vec_autorizacion_atestada_v3.huella_sha256_valida(v_raiz ->> 'huella_spki_sha256')
       OR (v_raiz ->> 'audiencia_despliegue') IS DISTINCT FROM
          'vec:desarrollo:contratacion-temporal:atestacion:v3'
       OR (v_raiz ->> 'suite') IS DISTINCT FROM 'VEC-AD-3-COSE-EDDSA-1'
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

    FOR v_indice IN 1..pg_catalog.cardinality(v_audiencias) LOOP
        v_item := v_claves -> (v_indice - 1);
        IF pg_catalog.jsonb_typeof(v_item) IS DISTINCT FROM 'object'
           OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(v_item)) <> 7
           OR NOT (v_item ?& ARRAY['audiencia_consumo','clave_id','version',
                 'revision_gobierno','huella_gobierno_sha256','huella_secreto_sha256','emisor_id'])
           OR v_item ->> 'audiencia_consumo' IS DISTINCT FROM v_audiencias[v_indice]
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
           OR v_clave.audiencia_consumo IS DISTINCT FROM v_audiencias[v_indice]
           OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
           OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
                       WHERE (r.clave_id,r.version)=(v_clave.clave_id,v_clave.version)
                         AND r.revocada_en <= v_ahora)
           OR (SELECT (p.clave_id,p.version) FROM vec_autorizacion_atestada_v3.puntero_clave_emision p
                JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k
                  ON (k.clave_id,k.version)=(p.clave_id,p.version)
                WHERE k.audiencia_consumo=v_audiencias[v_indice] AND p.establecida_en<=v_ahora
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

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)
    FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)
    FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)
    TO vec_autorizacion_atestada_v3_preflight_interno;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)
    TO vec_autorizacion_atestada_v3_preflight_interno;

-- Postimagen: propietario, definidora, entorno fijo y ACL exacta (solo el
-- propietario y el rol de preflight, sin opción de concesión).
DO $postimagen$
DECLARE
    x record;
BEGIN
    FOR x IN
        SELECT p.oid, p.proowner, p.prosecdef, p.proconfig, p.provolatile
          FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND p.proname IN ('comprobar_material_emision_interna_v2',
                             'leer_configuracion_interna_v2')
    LOOP
        IF x.proowner IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_roles
                  WHERE rolname = 'vec_autorizacion_atestada_v3_propietario')
           OR x.prosecdef IS NOT TRUE
           OR x.provolatile IS DISTINCT FROM 's'
           OR x.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
           OR EXISTS (
               SELECT 1 FROM pg_catalog.pg_proc AS p
                 CROSS JOIN LATERAL pg_catalog.aclexplode(
                     coalesce(p.proacl, pg_catalog.acldefault('f', p.proowner))) AS a
                WHERE p.oid = x.oid
                  AND (a.privilege_type <> 'EXECUTE'
                       OR (a.grantee <> p.proowner
                           AND a.grantee IS DISTINCT FROM (SELECT oid FROM pg_catalog.pg_roles
                                WHERE rolname = 'vec_autorizacion_atestada_v3_preflight_interno'))
                       OR (a.grantee <> p.proowner AND a.is_grantable)))
           OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
                 CROSS JOIN LATERAL pg_catalog.aclexplode(p.proacl) AS a
                 JOIN pg_catalog.pg_roles AS g ON g.oid = a.grantee
                WHERE p.oid = x.oid
                  AND g.rolname = 'vec_autorizacion_atestada_v3_preflight_interno') <> 1
        THEN
            RAISE EXCEPTION 'AD3-69: postimagen incompatible' USING ERRCODE = '55000';
        END IF;
    END LOOP;
    IF (SELECT pg_catalog.count(*) FROM pg_catalog.pg_proc AS p
          JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
         WHERE n.nspname = 'vec_autorizacion_atestada_v3'
           AND p.proname IN ('comprobar_material_emision_interna_v2',
                             'leer_configuracion_interna_v2')) <> 2
    THEN
        RAISE EXCEPTION 'AD3-69: postimagen incompatible' USING ERRCODE = '55000';
    END IF;
END $postimagen$;
COMMIT;
