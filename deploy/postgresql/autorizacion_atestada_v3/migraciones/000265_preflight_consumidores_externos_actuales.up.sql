\set ON_ERROR_STOP on
-- AD265: mismos lectores técnicos AD112, listas de consumidores actuales.
-- Después de AD263; literales instalados en la copia principal post-HZ37.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE actual text;
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regprocedure('vec_autorizacion_atestada_v3.gestionar_publicacion_externa_v1(jsonb)') IS NULL
 THEN RAISE EXCEPTION 'AD265: PARO clave=dependencia actual=incompatible esperado=PG18_AD263_superusuario' USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb)')),'UTF8')),'hex') INTO actual;
 IF actual IS DISTINCT FROM 'c64fe835356736afd25f6538dc117cd771d711d654133c1b0ecf8a55d0529358' THEN RAISE EXCEPTION 'AD265: PARO clave=leer_configuracion_externa_v1(text,jsonb) actual=% esperado=c64fe835356736afd25f6538dc117cd771d711d654133c1b0ecf8a55d0529358',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
 SELECT encode(sha256(convert_to(pg_get_functiondef(to_regprocedure('vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb)')),'UTF8')),'hex') INTO actual;
 IF actual IS DISTINCT FROM 'bd0c0c509a62772f9e71dfd4079e9ee9239ebd85e2b65b17d4325ce93b82bf91' THEN RAISE EXCEPTION 'AD265: PARO clave=comprobar_material_emision_externa_v1(text,jsonb) actual=% esperado=bd0c0c509a62772f9e71dfd4079e9ee9239ebd85e2b65b17d4325ce93b82bf91',coalesce(actual,'ausente') USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(p_consumidor text, p_material jsonb)
 RETURNS jsonb
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
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
    v_checkpoint vec_autorizacion_atestada_v3.checkpoint_gobierno_externo%ROWTYPE;
    v_indice integer;
BEGIN
    -- Misma lista cerrada que comprobar_material_emision_externa_v1.
    v_audiencias := CASE p_consumidor
        WHEN 'usuarios_preferencias' THEN ARRAY[
            'vec_usuarios.preferencias.consultar.externa_personal.v1',
            'vec_usuarios.preferencias.actualizar.externa_personal.v1']
        WHEN 'usuarios_correos' THEN ARRAY[
            'vec_usuarios.correos.consultar.externa_personal.v1',
            'vec_usuarios.correos.anadir.externa_personal.v1',
            'vec_usuarios.correos.reenviar.externa_personal.v1',
            'vec_usuarios.correos.verificar.externa_personal.v1',
            'vec_usuarios.correos.activar.externa_personal.v1',
            'vec_usuarios.correos.retirar.externa_personal.v1']
        WHEN 'usuarios_imagen' THEN ARRAY[
            'vec_usuarios.imagen.consultar.externa_personal.v1',
            'vec_usuarios.imagen.actualizar.externa_personal.v1']
        WHEN 'mi_bolsa' THEN ARRAY[
            'vec.bolsa.mi-bolsa.v1',
            'vec.bolsa.mi-bolsa.historial.v1']
        WHEN 'portal_candidato' THEN ARRAY[
            'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1',
            'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1',
            'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1',
            'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1',
            'vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1',
            'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1']
        WHEN 'preferencias_bolsas' THEN ARRAY['vec.bolsa.preferencias-bolsas.ordenar.v1']
        WHEN 'inscripcion_bolsa' THEN ARRAY[
            'vec_bolsa_llamamientos.inscripcion.presentar.v1',
            'vec_bolsa_llamamientos.alegacion.presentar.v1']
        WHEN 'meritos_cartera' THEN ARRAY[
            'vec_meritos.cartera.retirar_propio.v1',
            'vec_meritos.hecho.declarar.v1']
        END;

    -- El LOGIN es nominal y no puede heredar SELECT sobre las tablas de gobierno.
    IF v_audiencias IS NULL
       OR session_user <> 'vec_externo_preflight_v3_desarrollo'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE r.rolname = session_user
            AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
            AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR NOT pg_catalog.pg_has_role(session_user,
            'vec_autorizacion_atestada_v3_preflight_externo', 'MEMBER')
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m
            WHERE m.member = session_user::regrole) <> 1
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m
              JOIN pg_catalog.pg_roles g ON g.oid = m.roleid
            WHERE m.member = session_user::regrole
              AND g.rolname = 'vec_autorizacion_atestada_v3_preflight_externo'
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
       OR pg_catalog.jsonb_typeof(p_material) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <> 3
       OR NOT (p_material ?& ARRAY['claves','configuracion','raiz'])
    THEN
        RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
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
        RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
    END IF;

    SELECT * INTO v_checkpoint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo
      WHERE control_id = true;
    SELECT * INTO v_anterior FROM vec_autorizacion_atestada_v3.configuracion_confianza_version
      WHERE revision = v_previa ->> 'revision';
    IF v_checkpoint.control_id IS DISTINCT FROM true
       OR v_anterior.revision IS NULL
       OR v_anterior.secuencia IS DISTINCT FROM (v_previa ->> 'secuencia')::numeric
       OR v_anterior.huella_configuracion_sha256 IS DISTINCT FROM v_previa ->> 'huella_configuracion_sha256'
       OR NOT EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa p
                       WHERE p.configuracion_revision = v_anterior.revision
                         AND p.orden = v_anterior.secuencia
                         AND p.establecida_en <= v_ahora)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
                   WHERE r.configuracion_revision = v_anterior.revision AND r.revocada_en <= v_ahora)
    THEN
        RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
    END IF;

    SELECT c.* INTO v_actual
      FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa p
      JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version c
        ON c.revision = p.configuracion_revision
     WHERE p.establecida_en <= v_ahora
     ORDER BY p.orden DESC LIMIT 1;
    IF v_actual.revision IS NULL OR v_actual.secuencia < v_anterior.secuencia
       OR v_actual.secuencia < v_checkpoint.configuracion_secuencia_minima
       OR v_ahora < v_actual.publicada_en OR v_ahora >= v_actual.expira_en
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
                   WHERE r.configuracion_revision = v_actual.revision AND r.revocada_en <= v_ahora)
       OR (SELECT p.orden FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa p
            WHERE p.configuracion_revision = v_actual.revision) IS DISTINCT FROM v_actual.secuencia
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
            WHERE cr.configuracion_revision = v_actual.revision) <> 1
       OR (SELECT pg_catalog.count(*) FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
            WHERE cr.configuracion_revision = v_anterior.revision) <> 1
    THEN
        RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
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
        RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
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
            RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
        END IF;
        SELECT * INTO v_clave FROM vec_autorizacion_atestada_v3.clave_capacidad_version
          WHERE clave_id = v_item ->> 'clave_id'
            AND version = (v_item ->> 'version')::numeric;
        IF v_clave.clave_id IS NULL
           OR v_clave.revision_gobierno IS DISTINCT FROM (v_item ->> 'revision_gobierno')::numeric
           OR v_clave.huella_gobierno_sha256 IS DISTINCT FROM v_item ->> 'huella_gobierno_sha256'
           OR v_clave.huella_secreto_sha256 IS DISTINCT FROM v_item ->> 'huella_secreto_sha256'
           OR v_clave.emisor_id IS DISTINCT FROM v_item ->> 'emisor_id'
           OR v_clave.audiencia_consumo IS DISTINCT FROM v_audiencias[v_indice]
           OR v_ahora < v_clave.valida_desde OR v_ahora >= v_clave.valida_hasta
           OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
                       WHERE (r.clave_id,r.version)=(v_clave.clave_id,v_clave.version)
                         AND r.revocada_en <= v_ahora)
           OR (SELECT (p.clave_id,p.version) FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa p
                JOIN vec_autorizacion_atestada_v3.clave_capacidad_version k
                  ON (k.clave_id,k.version)=(p.clave_id,p.version)
                WHERE k.audiencia_consumo=v_audiencias[v_indice] AND p.establecida_en<=v_ahora
                ORDER BY p.orden DESC LIMIT 1) IS DISTINCT FROM (v_clave.clave_id,v_clave.version)
        THEN
            RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
        END IF;
    END LOOP;
    RETURN pg_catalog.jsonb_build_object(
       'revision',v_actual.revision,'secuencia',v_actual.secuencia,
       'huella_configuracion_sha256',v_actual.huella_configuracion_sha256,
       'publicada_en',v_actual.publicada_en,'expira_en',v_actual.expira_en);
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION 'gobierno externo rechazado' USING ERRCODE = '42501';
END $function$;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(p_consumidor text, p_material jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
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
    v_checkpoint vec_autorizacion_atestada_v3.checkpoint_gobierno_externo%ROWTYPE;
    v_indice integer;
BEGIN
    -- Lista cerrada por consumidor: el llamante no aporta audiencias.
    v_audiencias := CASE p_consumidor
        WHEN 'usuarios_preferencias' THEN ARRAY[
            'vec_usuarios.preferencias.consultar.externa_personal.v1',
            'vec_usuarios.preferencias.actualizar.externa_personal.v1']
        WHEN 'usuarios_correos' THEN ARRAY[
            'vec_usuarios.correos.consultar.externa_personal.v1',
            'vec_usuarios.correos.anadir.externa_personal.v1',
            'vec_usuarios.correos.reenviar.externa_personal.v1',
            'vec_usuarios.correos.verificar.externa_personal.v1',
            'vec_usuarios.correos.activar.externa_personal.v1',
            'vec_usuarios.correos.retirar.externa_personal.v1']
        WHEN 'usuarios_imagen' THEN ARRAY[
            'vec_usuarios.imagen.consultar.externa_personal.v1',
            'vec_usuarios.imagen.actualizar.externa_personal.v1']
        WHEN 'mi_bolsa' THEN ARRAY[
            'vec.bolsa.mi-bolsa.v1',
            'vec.bolsa.mi-bolsa.historial.v1']
        WHEN 'portal_candidato' THEN ARRAY[
            'vec_bolsa_llamamientos.participaciones_propias.solicitar_pausa.v1',
            'vec_bolsa_llamamientos.participaciones_propias.solicitar_reactivacion.v1',
            'vec_bolsa_llamamientos.participaciones_propias.responder_llamamiento.v1',
            'vec_bolsa_llamamientos.participaciones_propias.manifestar_disposicion.v1',
            'vec_bolsa_llamamientos.participaciones_propias.presentar_solicitud_documental.v1',
            'vec_bolsa_llamamientos.participaciones_propias.confirmar_contacto.v1']
        WHEN 'preferencias_bolsas' THEN ARRAY['vec.bolsa.preferencias-bolsas.ordenar.v1']
        WHEN 'inscripcion_bolsa' THEN ARRAY[
            'vec_bolsa_llamamientos.inscripcion.presentar.v1',
            'vec_bolsa_llamamientos.alegacion.presentar.v1']
        WHEN 'meritos_cartera' THEN ARRAY[
            'vec_meritos.cartera.retirar_propio.v1',
            'vec_meritos.hecho.declarar.v1']
        END;

    -- El LOGIN de arranque es nominal y distinto de emisores y consumidores.
    IF v_audiencias IS NULL
       OR session_user <> 'vec_externo_preflight_v3_desarrollo'
       OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles AS r
                       WHERE r.rolname = session_user
                         AND r.rolcanlogin AND NOT r.rolsuper
                         AND NOT r.rolcreatedb AND NOT r.rolcreaterole
                         AND NOT r.rolreplication AND NOT r.rolbypassrls)
       OR NOT pg_catalog.pg_has_role(session_user,
              'vec_autorizacion_atestada_v3_preflight_externo', 'MEMBER')
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members AS m
            WHERE m.member = session_user::regrole) <> 1
       OR NOT EXISTS (
           SELECT 1 FROM pg_catalog.pg_auth_members AS m
             JOIN pg_catalog.pg_roles AS g ON g.oid = m.roleid
            WHERE m.member = session_user::regrole
              AND g.rolname = 'vec_autorizacion_atestada_v3_preflight_externo'
              AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)
    THEN
        RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
    END IF;

    IF pg_catalog.jsonb_typeof(p_material) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_material)) <> 3
       OR NOT (p_material ?& ARRAY['claves', 'configuracion', 'raiz'])
    THEN
        RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
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
        RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
    END IF;

    SELECT * INTO v_checkpoint FROM vec_autorizacion_atestada_v3.checkpoint_gobierno_externo
     WHERE control_id = true;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
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
             FROM vec_autorizacion_atestada_v3.puntero_configuracion_externa AS p
            WHERE p.establecida_en <= v_ahora ORDER BY p.orden DESC LIMIT 1)
          IS DISTINCT FROM v_configuracion.revision
    THEN
        RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
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
        RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
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
            RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
        END IF;
        SELECT * INTO v_clave
          FROM vec_autorizacion_atestada_v3.clave_capacidad_version
         WHERE clave_id = v_item ->> 'clave_id'
           AND version = (v_item ->> 'version')::numeric;
        IF NOT FOUND
           OR v_clave.revision_gobierno IS DISTINCT FROM
              (v_item ->> 'revision_gobierno')::numeric
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
                 FROM vec_autorizacion_atestada_v3.puntero_clave_emision_externa AS p
                 JOIN vec_autorizacion_atestada_v3.clave_capacidad_version AS k
                   ON k.clave_id = p.clave_id AND k.version = p.version
                WHERE k.audiencia_consumo = v_audiencias[v_indice]
                  AND p.establecida_en <= v_ahora
                ORDER BY p.orden DESC LIMIT 1)
              IS DISTINCT FROM (v_clave.clave_id, v_clave.version)
        THEN
            RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
        END IF;
    END LOOP;
    RETURN true;
EXCEPTION WHEN OTHERS THEN
    RAISE EXCEPTION 'material de emisión externa rechazado' USING ERRCODE = '42501';
END $function$;
-- CREATE OR REPLACE conserva propietario y ACL; PUBLIC sigue denegado.
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.leer_configuracion_externa_v1(text,jsonb),
 vec_autorizacion_atestada_v3.comprobar_material_emision_externa_v1(text,jsonb) FROM PUBLIC;
COMMIT;
