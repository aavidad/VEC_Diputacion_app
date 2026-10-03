BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL TIME ZONE 'UTC';
SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(
    pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000169', 0));

-- La corriente sigue siendo auditoria_consumo_v3 y conserva su cabeza, sus
-- secuencias y los bytes de las huellas anteriores. Los intentos no consumen
-- una decisión V3: las tres columnas de la FK se mantienen NULAS juntas.
DO $pre$
BEGIN
    IF pg_catalog.to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3') IS NULL
       OR pg_catalog.to_regclass('vec_autorizacion_atestada_v3.control_cadena_auditoria') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.encuadrar_mac(text)') IS NULL
       OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)') IS NOT NULL
       OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(text,text,text,bytea)') IS NULL
       OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(bytea)') IS NULL THEN
        RAISE EXCEPTION USING ERRCODE='55000', MESSAGE='precondiciones AD169 no satisfechas';
    END IF;
END
$pre$;

ALTER TABLE vec_autorizacion_atestada_v3.auditoria_consumo_v3
    ALTER COLUMN decision_ref DROP NOT NULL,
    ALTER COLUMN efecto_ref DROP NOT NULL,
    ALTER COLUMN huella_efecto_sha256 DROP NOT NULL,
    ADD COLUMN tipo_registro text NOT NULL DEFAULT 'consumo_confirmado',
    ADD COLUMN intento_ref text UNIQUE,
    ADD COLUMN intento_material_sha256 text,
    ADD COLUMN actor_ref text,
    ADD COLUMN perfil_activo_ref text,
    ADD COLUMN registro_contexto_ref text,
    ADD COLUMN contexto_sha256 text,
    ADD COLUMN procedencia_sha256 text,
    ADD COLUMN autenticacion_ref text,
    ADD COLUMN sesion_ref text,
    ADD COLUMN autenticacion_sha256 text,
    ADD COLUMN accion text,
    ADD COLUMN modulo_id text,
    ADD COLUMN recurso_ref text,
    ADD COLUMN finalidad_ref text,
    ADD COLUMN resultado text,
    ADD COLUMN motivo_ref text,
    ADD COLUMN proceso text,
    ADD COLUMN canal text,
    ADD COLUMN correlacion_ref text,
    ADD COLUMN vinculo_sha256 text,
    ADD CONSTRAINT auditoria_tipo_disjunto_v1 CHECK (
       (tipo_registro='consumo_confirmado'
        AND decision_ref IS NOT NULL AND efecto_ref IS NOT NULL
        AND huella_efecto_sha256 IS NOT NULL
        AND intento_ref IS NULL AND intento_material_sha256 IS NULL
        AND actor_ref IS NULL AND perfil_activo_ref IS NULL
        AND registro_contexto_ref IS NULL AND contexto_sha256 IS NULL
        AND procedencia_sha256 IS NULL AND autenticacion_ref IS NULL
        AND sesion_ref IS NULL AND autenticacion_sha256 IS NULL
        AND accion IS NULL AND modulo_id IS NULL AND recurso_ref IS NULL
        AND finalidad_ref IS NULL AND resultado IS NULL AND motivo_ref IS NULL
        AND proceso IS NULL AND canal IS NULL AND correlacion_ref IS NULL
        AND vinculo_sha256 IS NULL)
       OR
       (tipo_registro='intento_nominal'
        AND decision_ref IS NULL AND efecto_ref IS NULL
        AND huella_efecto_sha256 IS NULL
        AND intento_ref IS NOT NULL AND intento_material_sha256 IS NOT NULL
        AND actor_ref IS NOT NULL AND perfil_activo_ref IS NOT NULL
        AND registro_contexto_ref IS NOT NULL AND contexto_sha256 IS NOT NULL
        AND procedencia_sha256 IS NOT NULL AND autenticacion_ref IS NOT NULL
        AND sesion_ref IS NOT NULL AND autenticacion_sha256 IS NOT NULL
        AND accion IS NOT NULL AND modulo_id IS NOT NULL AND recurso_ref IS NOT NULL
        AND finalidad_ref IS NOT NULL AND resultado IS NOT NULL AND motivo_ref IS NOT NULL
        AND proceso IS NOT NULL AND canal IS NOT NULL AND correlacion_ref IS NOT NULL
        AND vinculo_sha256 IS NOT NULL)
    ),
    ADD CONSTRAINT auditoria_intento_formato_v1 CHECK (
        tipo_registro <> 'intento_nominal' OR (
          intento_ref ~ '^intento_[0-9a-f]{32}$'
          AND intento_material_sha256 ~ '^[0-9a-f]{64}$'
          AND contexto_sha256 ~ '^[0-9a-f]{64}$'
          AND procedencia_sha256 ~ '^[0-9a-f]{64}$'
          AND autenticacion_sha256 ~ '^[0-9a-f]{64}$'
          AND vinculo_sha256 ~ '^[0-9a-f]{64}$'
          AND resultado IN ('denegado','error')
          AND pg_catalog.octet_length(actor_ref) BETWEEN 1 AND 128
          AND pg_catalog.octet_length(perfil_activo_ref) BETWEEN 1 AND 128
          AND pg_catalog.octet_length(registro_contexto_ref) BETWEEN 1 AND 128
          AND pg_catalog.octet_length(autenticacion_ref) BETWEEN 1 AND 128
          AND pg_catalog.octet_length(sesion_ref) BETWEEN 1 AND 128
          AND pg_catalog.octet_length(accion) BETWEEN 1 AND 160
          AND pg_catalog.octet_length(modulo_id) BETWEEN 1 AND 160
          AND pg_catalog.octet_length(recurso_ref) BETWEEN 1 AND 200
          AND pg_catalog.octet_length(finalidad_ref) BETWEEN 1 AND 160
          AND pg_catalog.octet_length(motivo_ref) BETWEEN 1 AND 160
          AND pg_catalog.octet_length(proceso) BETWEEN 1 AND 80
          AND pg_catalog.octet_length(canal) BETWEEN 1 AND 80
          AND pg_catalog.octet_length(correlacion_ref) BETWEEN 1 AND 128
        )
    );

-- Configuración técnica: el DBA registra por fuera de Git el LOGIN de cada
-- proceso autorizado. No se siembra ningún LOGIN ni valor de proceso supuesto.
-- La ausencia de fila cierra el puerto. Revocar la membresía del rol desactiva
-- la escritura; cambiar proceso o canal requiere un LOGIN nuevo gobernado.
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_runtime_intentos (
    login_nombre name PRIMARY KEY,
    proceso text NOT NULL CHECK (
        proceso ~ '^[a-z][a-z0-9._-]{1,79}$'),
    canal text NOT NULL CHECK (
        canal IN ('administracion_privilegiada',
                  'interna_corporativa', 'externa_personal')),
    configurada_en timestamptz(6) NOT NULL DEFAULT pg_catalog.clock_timestamp()
);
CREATE TRIGGER inmutable BEFORE UPDATE OR DELETE ON
    vec_autorizacion_atestada_v3.configuracion_runtime_intentos
    FOR EACH ROW EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_mutacion();
CREATE TRIGGER no_truncar BEFORE TRUNCATE ON
    vec_autorizacion_atestada_v3.configuracion_runtime_intentos
    FOR EACH STATEMENT EXECUTE FUNCTION vec_autorizacion_atestada_v3.rechazar_truncado();
ALTER TABLE vec_autorizacion_atestada_v3.configuracion_runtime_intentos
    ENABLE ROW LEVEL SECURITY;
ALTER TABLE vec_autorizacion_atestada_v3.configuracion_runtime_intentos
    FORCE ROW LEVEL SECURITY;
CREATE POLICY propietario_exacto ON
    vec_autorizacion_atestada_v3.configuracion_runtime_intentos
    FOR ALL TO vec_autorizacion_atestada_v3_propietario
    USING (current_user='vec_autorizacion_atestada_v3_propietario')
    WITH CHECK (current_user='vec_autorizacion_atestada_v3_propietario');
REVOKE ALL ON vec_autorizacion_atestada_v3.configuracion_runtime_intentos
    FROM PUBLIC, vec_autorizacion_atestada_v3_registrador_intentos,
         vec_autorizacion_atestada_v3_consumidor,
         vec_autorizacion_atestada_v3_emisor;

CREATE FUNCTION vec_autorizacion_atestada_v3.acreditar_registrador_intentos_v1(
    p_proceso text,p_canal text
) RETURNS boolean
LANGUAGE sql STABLE SECURITY INVOKER
SET search_path=pg_catalog
AS $funcion$
    SELECT EXISTS (
      SELECT 1 FROM pg_catalog.pg_roles login
      JOIN pg_catalog.pg_auth_members m ON m.member=login.oid
      JOIN pg_catalog.pg_roles grupo ON grupo.oid=m.roleid
      JOIN vec_autorizacion_atestada_v3.configuracion_runtime_intentos c
        ON c.login_nombre=login.rolname
      WHERE login.rolname=session_user
        AND login.rolcanlogin AND login.rolinherit
        AND NOT login.rolsuper AND NOT login.rolcreaterole
        AND NOT login.rolcreatedb AND NOT login.rolreplication
        AND NOT login.rolbypassrls AND login.rolconfig IS NULL
        AND grupo.rolname='vec_autorizacion_atestada_v3_registrador_intentos'
        AND NOT grupo.rolcanlogin AND NOT grupo.rolsuper
        AND NOT grupo.rolcreaterole AND NOT grupo.rolcreatedb
        AND NOT grupo.rolreplication AND NOT grupo.rolbypassrls
        AND m.admin_option IS FALSE AND m.inherit_option IS TRUE
        AND m.set_option IS FALSE
        AND (SELECT pg_catalog.count(*) FROM pg_catalog.pg_auth_members m2
              WHERE m2.member=login.oid)=1
        AND c.proceso=p_proceso AND c.canal=p_canal
        AND pg_catalog.current_setting('role')='none')
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.acreditar_registrador_intentos_v1(text,text)
    FROM PUBLIC,vec_autorizacion_atestada_v3_registrador_intentos,
         vec_autorizacion_atestada_v3_consumidor,
         vec_autorizacion_atestada_v3_emisor;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
    p_contexto_canonico bytea,
    p_vinculo_canonico bytea,
    p_orden jsonb
) RETURNS TABLE(
    auditoria_ref text, secuencia numeric, huella_sha256 text,
    correlacion_ref text, registrada_en timestamptz
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog
SET row_security=on
SET statement_timeout='10s'
SET lock_timeout='2s'
AS $funcion$
DECLARE
    v_contexto jsonb;
    v_vinculo jsonb;
    v_esperadas constant text[] := ARRAY[
        'intento_ref','registro_contexto_ref','contexto_sha256',
        'procedencia_sha256','autenticacion_ref','sesion_ref',
        'autenticacion_sha256','accion','modulo_id','recurso_ref',
        'finalidad_ref','resultado','motivo_ref','proceso','canal',
        'correlacion_ref'];
    v_clave text;
    v_material bytea := ''::bytea;
    v_material_sha text;
    v_anterior text;
    v_secuencia numeric;
    v_instante timestamptz(6);
    v_huella text;
    v_ref text;
    v_existente record;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR pg_catalog.current_setting('role') <> 'none' THEN
        RAISE EXCEPTION USING ERRCODE='25000', MESSAGE='transacción de auditoría inválida';
    END IF;
    IF vec_autorizacion_atestada_v3.acreditar_registrador_intentos_v1(
           p_orden ->> 'proceso', p_orden ->> 'canal') IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='registrador técnico no acreditado';
    END IF;
    IF p_contexto_canonico IS NULL OR p_vinculo_canonico IS NULL
       OR pg_catalog.octet_length(p_contexto_canonico) NOT BETWEEN 1 AND 65536
       OR pg_catalog.octet_length(p_vinculo_canonico) NOT BETWEEN 1 AND 16384
       OR pg_catalog.jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(p_orden)) <> 16
       OR (p_orden ?& v_esperadas) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='orden de auditoría inválida';
    END IF;
    FOREACH v_clave IN ARRAY v_esperadas LOOP
        IF pg_catalog.jsonb_typeof(p_orden -> v_clave) IS DISTINCT FROM 'string'
           OR pg_catalog.octet_length(p_orden ->> v_clave) NOT BETWEEN 1 AND 200 THEN
            RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='campo de auditoría inválido';
        END IF;
    END LOOP;
    IF (p_orden ->> 'intento_ref') !~ '^intento_[0-9a-f]{32}$'
       OR (p_orden ->> 'contexto_sha256') !~ '^[0-9a-f]{64}$'
       OR (p_orden ->> 'procedencia_sha256') !~ '^[0-9a-f]{64}$'
       OR (p_orden ->> 'autenticacion_sha256') !~ '^[0-9a-f]{64}$'
       OR (p_orden ->> 'resultado') NOT IN ('denegado','error')
       OR (p_orden ->> 'accion') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'modulo_id') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'recurso_ref') !~ '^[a-z0-9][a-z0-9._:-]{0,199}$'
       OR (p_orden ->> 'finalidad_ref') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'motivo_ref') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'correlacion_ref') !~ '^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}$' THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='identificadores de auditoría inválidos';
    END IF;
    v_contexto := pg_catalog.convert_from(p_contexto_canonico,'UTF8')::jsonb;
    v_vinculo := pg_catalog.convert_from(p_vinculo_canonico,'UTF8')::jsonb;
    IF pg_catalog.jsonb_typeof(v_contexto) IS DISTINCT FROM 'object'
       OR pg_catalog.jsonb_typeof(v_vinculo) IS DISTINCT FROM 'object'
       OR v_contexto ->> 'esquema' IS DISTINCT FROM 'vec.contexto-actor.vinculado.v2'
       OR v_vinculo ->> 'esquema' IS DISTINCT FROM 'vec.autenticacion-actor.vinculo.v2.contexto-registrado'
       OR v_vinculo ->> 'bloque_version' IS DISTINCT FROM '2'
       OR v_vinculo ->> 'principal_id' IS DISTINCT FROM v_contexto ->> 'principal_ref'
       OR v_vinculo ->> 'perfil_activo_ref' IS DISTINCT FROM v_contexto ->> 'perfil_activo_ref'
       OR v_vinculo ->> 'cuenta_ref' IS DISTINCT FROM v_contexto ->> 'cuenta_ref'
       OR v_vinculo ->> 'metodo_observado' IS DISTINCT FROM v_contexto ->> 'metodo'
       OR v_vinculo ->> 'garantia_observada' IS DISTINCT FROM v_contexto ->> 'garantia'
       OR v_vinculo ->> 'contexto_actor_esquema' IS DISTINCT FROM v_contexto ->> 'esquema'
       OR v_vinculo ->> 'contexto_actor_ref' IS DISTINCT FROM v_contexto ->> 'contexto_actor_ref'
       OR v_vinculo ->> 'contexto_actor_version' IS DISTINCT FROM v_contexto ->> 'contexto_version'
       OR v_vinculo ->> 'contexto_actor_cuenta_version' IS DISTINCT FROM v_contexto ->> 'cuenta_version'
       OR v_vinculo ->> 'registro_contexto_ref' IS DISTINCT FROM p_orden ->> 'registro_contexto_ref'
       OR v_vinculo ->> 'contexto_actor_huella_sha256' IS DISTINCT FROM p_orden ->> 'contexto_sha256'
       OR v_vinculo ->> 'manifiesto_procedencia_huella_sha256' IS DISTINCT FROM p_orden ->> 'procedencia_sha256'
       OR v_vinculo ->> 'autenticacion_ref' IS DISTINCT FROM p_orden ->> 'autenticacion_ref'
       OR v_vinculo ->> 'sesion_ref' IS DISTINCT FROM p_orden ->> 'sesion_ref'
       OR v_vinculo ->> 'autenticacion_huella_sha256' IS DISTINCT FROM p_orden ->> 'autenticacion_sha256'
       OR v_vinculo ->> 'autoridad_efectiva' IS DISTINCT FROM 'autoridad_maestra_acreditada'
       OR v_vinculo ->> 'superficie' IS DISTINCT FROM p_orden ->> 'canal'
       OR (((v_contexto ->> 'resuelto_en')::timestamptz >=
            (v_vinculo ->> 'sesion_revalidada_en')::timestamptz)
           AND ((v_contexto ->> 'resuelto_en')::timestamptz <
            (v_vinculo ->> 'sesion_valida_hasta')::timestamptz)) IS NOT TRUE
       OR pg_catalog.encode(pg_catalog.sha256(p_contexto_canonico),'hex')
          IS DISTINCT FROM p_orden ->> 'contexto_sha256' THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='vínculo nominal incoherente';
    END IF;
    -- Estas fachadas pertenecen a CA e Identidad. Contrastan los recibos
    -- históricos en tablas propias incluso cuando ya no hay vigencia actual.
    IF vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
           p_orden ->> 'registro_contexto_ref',
           p_orden ->> 'contexto_sha256',
           p_orden ->> 'procedencia_sha256',p_contexto_canonico) IS NOT TRUE
       OR vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
           p_vinculo_canonico) IS NOT TRUE THEN
        RAISE EXCEPTION USING ERRCODE='42501', MESSAGE='evidencia histórica no acreditada';
    END IF;
    -- Preimagen estable del intento; el instante y la posición se comprometen
    -- después en el eslabón de cadena. Una repetición exacta devuelve su acuse.
    v_material := vec_autorizacion_atestada_v3.encuadrar_mac(
        'vec.auditoria.intento.v1') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(pg_catalog.sha256(p_contexto_canonico),'hex')) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.encode(pg_catalog.sha256(p_vinculo_canonico),'hex'));
    FOREACH v_clave IN ARRAY v_esperadas LOOP
        v_material := v_material ||
            vec_autorizacion_atestada_v3.encuadrar_mac(p_orden ->> v_clave);
    END LOOP;
    v_material_sha := pg_catalog.encode(pg_catalog.sha256(v_material),'hex');
    PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
        'vec_autorizacion_atestada_v3:intento:' || (p_orden ->> 'intento_ref'),0));
    SELECT a.auditoria_ref,a.secuencia,a.huella_sha256,a.correlacion_ref,a.registrada_en,
           a.intento_material_sha256
      INTO v_existente FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a
     WHERE a.intento_ref=(p_orden ->> 'intento_ref');
    IF FOUND THEN
        IF v_existente.intento_material_sha256 IS DISTINCT FROM v_material_sha THEN
            RAISE EXCEPTION USING ERRCODE='23505', MESSAGE='intento incompatible con recibo previo';
        END IF;
        RETURN QUERY SELECT v_existente.auditoria_ref,v_existente.secuencia,
            v_existente.huella_sha256,v_existente.correlacion_ref,
            v_existente.registrada_en;
        RETURN;
    END IF;
    SELECT c.secuencia,c.cabeza_sha256 INTO STRICT v_secuencia,v_anterior
      FROM vec_autorizacion_atestada_v3.control_cadena_auditoria c
     WHERE c.control_id FOR UPDATE;
    IF v_secuencia >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING ERRCODE='22003', MESSAGE='límite de cadena de auditoría alcanzado';
    END IF;
    v_secuencia := v_secuencia+1;
    v_instante := pg_catalog.clock_timestamp();
    v_ref := 'aud_v3_i_' || pg_catalog.substr(p_orden ->> 'intento_ref',9,32);
    v_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_autorizacion_atestada_v3.encuadrar_mac('vec.auditoria.eslabon.intento.v1') ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_secuencia::text) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_anterior) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_ref) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(v_material_sha) ||
        vec_autorizacion_atestada_v3.encuadrar_mac(
            pg_catalog.to_char(v_instante AT TIME ZONE 'UTC',
                'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'))),'hex');
    INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3 (
        auditoria_ref,secuencia,anterior_sha256,huella_sha256,registrada_en,
        tipo_registro,intento_ref,intento_material_sha256,actor_ref,
        perfil_activo_ref,registro_contexto_ref,contexto_sha256,
        procedencia_sha256,autenticacion_ref,sesion_ref,autenticacion_sha256,
        accion,modulo_id,recurso_ref,finalidad_ref,resultado,motivo_ref,
        proceso,canal,correlacion_ref,vinculo_sha256
    ) VALUES (
        v_ref,v_secuencia,v_anterior,v_huella,v_instante,
        'intento_nominal',p_orden ->> 'intento_ref',v_material_sha,
        v_contexto ->> 'principal_ref',v_contexto ->> 'perfil_activo_ref',
        p_orden ->> 'registro_contexto_ref',p_orden ->> 'contexto_sha256',
        p_orden ->> 'procedencia_sha256',p_orden ->> 'autenticacion_ref',
        p_orden ->> 'sesion_ref',p_orden ->> 'autenticacion_sha256',
        p_orden ->> 'accion',p_orden ->> 'modulo_id',p_orden ->> 'recurso_ref',
        p_orden ->> 'finalidad_ref',p_orden ->> 'resultado',p_orden ->> 'motivo_ref',
        p_orden ->> 'proceso',p_orden ->> 'canal',p_orden ->> 'correlacion_ref',
        pg_catalog.encode(pg_catalog.sha256(p_vinculo_canonico),'hex'));
    UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
       SET secuencia=v_secuencia,cabeza_sha256=v_huella,actualizada_en=v_instante
     WHERE control_id;
    RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden ->> 'correlacion_ref',
        v_instante;
END
$funcion$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
    bytea,bytea,jsonb) FROM PUBLIC,vec_autorizacion_atestada_v3_emisor,
    vec_autorizacion_atestada_v3_consumidor;
CREATE FUNCTION vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(
    p_proceso text,p_canal text
) RETURNS boolean
LANGUAGE sql STABLE SECURITY DEFINER
SET search_path=pg_catalog
SET row_security=on
AS $funcion$
    SELECT vec_autorizacion_atestada_v3.acreditar_registrador_intentos_v1(
               p_proceso,p_canal)
       AND pg_catalog.has_function_privilege(
               session_user,
               'vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)',
               'EXECUTE')
$funcion$;
REVOKE ALL ON FUNCTION
    vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(text,text)
    FROM PUBLIC,vec_autorizacion_atestada_v3_emisor,
         vec_autorizacion_atestada_v3_consumidor;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3
    TO vec_autorizacion_atestada_v3_registrador_intentos;
GRANT EXECUTE ON FUNCTION
    vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb),
    vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(text,text)
    TO vec_autorizacion_atestada_v3_registrador_intentos;

COMMIT;
