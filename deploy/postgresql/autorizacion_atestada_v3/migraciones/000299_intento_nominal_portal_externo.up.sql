\set ON_ERROR_STOP on
-- AD299 (F1-85): el intento nominal del aspirante en el portal externo coteja
-- su contexto (CA47) y su sesión (IS22) en los almacenes del portal externo.
-- registrar_intento_nominal_v1 (AD169, AD252, AD259) admite el canal
-- externa_personal pero cotejaba siempre con CA26/CA27 e IS20, así que un
-- intento denegado del portal externo separado fallaba con P0002 (503) y no
-- quedaba asentado. Reconstruye solo esa función desde su cuerpo instalado
-- (AD259); los demás canales conservan exactamente su cotejo. Mismo
-- propietario, ACL, configuración y firma; no toca el núcleo, otras
-- funciones, tablas ni datos. Mismo patrón que AD289 y AD298.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000299',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
-- Solo metadatos de esta transacción; el operador DBA conserva su propiedad.
CREATE TEMP TABLE ad299_meta ON COMMIT DROP AS
 SELECT p.oid,to_jsonb(p)-'prosrc' AS meta,
  encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex') AS huella
 FROM pg_proc p WHERE p.oid=to_regprocedure(
  'vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)');
DO $pre$
DECLARE
 c regprocedure:=to_regprocedure('vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(text,text,text,bytea)');
 s regprocedure:=to_regprocedure('vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(bytea)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR (SELECT count(*) FROM pg_temp.ad299_meta)<>1
 -- Posimagen exacta de AD259.
 OR NOT EXISTS(SELECT 1 FROM pg_temp.ad299_meta
  WHERE huella='2375c84d4076a85c22150ef1a00916a337e70225a7924c55c4bc8b2d698d5e0d'
   AND meta->>'proowner'=('vec_autorizacion_atestada_v3_propietario'::regrole::oid)::text)
 OR (SELECT p.proacl FROM pg_proc p WHERE p.oid=(SELECT oid FROM pg_temp.ad299_meta)) IS DISTINCT FROM
    ARRAY['vec_autorizacion_atestada_v3_propietario=X/vec_autorizacion_atestada_v3_propietario',
     'vec_autorizacion_atestada_v3_registrador_intentos=X/vec_autorizacion_atestada_v3_propietario']::aclitem[]
 -- CA47 e IS22 instaladas con su posimagen exacta y solo ejecutables por su
 -- propietario y el de AD3 (ni el registrador de intentos ni Bolsa).
 OR c IS NULL OR s IS NULL
 OR encode(sha256(convert_to(pg_get_functiondef(c),'UTF8')),'hex')
    IS DISTINCT FROM '2890e6afe71d2e9e12cd3de00702959bd3145856cf178207c4d4c9b515636a05'
 OR encode(sha256(convert_to(pg_get_functiondef(s),'UTF8')),'hex')
    IS DISTINCT FROM 'ca870f4ee3ba01c22a5ab916d94954ebfac19a48f24dc702db5862898508d142'
 OR (SELECT p.proacl FROM pg_proc p WHERE p.oid=c) IS DISTINCT FROM
    ARRAY['vec_contexto_actor_v1_propietario=X/vec_contexto_actor_v1_propietario',
     'vec_autorizacion_atestada_v3_propietario=X/vec_contexto_actor_v1_propietario']::aclitem[]
 OR (SELECT p.proacl FROM pg_proc p WHERE p.oid=s) IS DISTINCT FROM
    ARRAY['vec_identidad_sesiones_v1_propietario=X/vec_identidad_sesiones_v1_propietario',
     'vec_autorizacion_atestada_v3_propietario=X/vec_identidad_sesiones_v1_propietario']::aclitem[]
 OR has_function_privilege('vec_autorizacion_atestada_v3_registrador_intentos',c,'EXECUTE')
 OR has_function_privilege('vec_autorizacion_atestada_v3_registrador_intentos',s,'EXECUTE')
 THEN RAISE EXCEPTION 'AD299: preimagen, ACL, CA47 o IS22 divergente; no reaplicar' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE OR REPLACE FUNCTION vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(p_contexto_canonico bytea, p_vinculo_canonico bytea, p_orden jsonb)
 RETURNS TABLE(auditoria_ref text, secuencia numeric, huella_sha256 text, correlacion_ref text, registrada_en timestamp with time zone)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET statement_timeout TO '10s'
 SET lock_timeout TO '2s'
AS $function$
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
       OR (p_orden ->> 'resultado') NOT IN ('denegado','error','exito')
       OR (p_orden ->> 'accion') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'modulo_id') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'recurso_ref') !~ '^[a-z0-9][a-z0-9._:-]{0,199}$'
       OR (p_orden ->> 'finalidad_ref') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'motivo_ref') !~ '^[a-z][a-z0-9._:-]{0,159}$'
       OR (p_orden ->> 'correlacion_ref') !~ '^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,127}$' THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='identificadores de auditoría inválidos';
    END IF;
    -- Éxito solo para las cuatro parejas acción/finalidad exactas. Para
    -- configuración, la referencia contiene la huella del perfil propio.
    -- El registrador constata el resultado; no concede la lectura.
    IF (p_orden ->> 'resultado') = 'exito' AND (
        (p_orden ->> 'modulo_id') IS DISTINCT FROM 'bolsa'
        OR (p_orden ->> 'proceso') IS DISTINCT FROM 'vec-rrhh'
        OR (p_orden ->> 'canal') IS DISTINCT FROM 'interna_corporativa'
        OR (p_orden ->> 'recurso_ref') !~ '^bolsa:[a-z0-9][a-z0-9._:-]{0,193}$'
        OR NOT (
            ((p_orden ->> 'accion') IN
                ('lectura_bolsas_rrhh_consultar','lectura_bolsas_rrhh_descargar_recibo')
             AND (p_orden ->> 'finalidad_ref') = 'gestion_bolsa_rrhh')
            OR ((p_orden ->> 'accion') = 'bolsa.configuracion.consultar'
                AND (p_orden ->> 'finalidad_ref') = 'consulta_configuracion_bolsa_rrhh'
                AND (p_orden ->> 'recurso_ref') ~ '^bolsa:perfil:[0-9a-f]{64}$')
            OR ((p_orden ->> 'accion') = 'bolsa.origen_ct.consultar'
                AND (p_orden ->> 'finalidad_ref') = 'gestionar_contratacion_temporal')
        )
    ) THEN
        RAISE EXCEPTION USING ERRCODE='22023', MESSAGE='éxito nominal no admitido';
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
    -- AD299: el portal externo coteja contexto y sesión en sus almacenes
    -- (CA47 e IS22); los demás canales siguen en CA26/CA27 e IS20.
    IF (CASE p_orden ->> 'canal'
         WHEN 'externa_personal' THEN
          vec_contexto_actor_v1.cotejar_contexto_historico_portal_externo_v1(
           p_orden ->> 'registro_contexto_ref',
           p_orden ->> 'contexto_sha256',
           p_orden ->> 'procedencia_sha256',p_contexto_canonico)
         ELSE vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
           p_orden ->> 'registro_contexto_ref',
           p_orden ->> 'contexto_sha256',
           p_orden ->> 'procedencia_sha256',p_contexto_canonico) END) IS NOT TRUE
       OR (CASE p_orden ->> 'canal'
         WHEN 'externa_personal' THEN
          vec_identidad_sesiones_v1.cotejar_autenticacion_historica_portal_externo_v1(
           p_vinculo_canonico)
         ELSE vec_identidad_sesiones_v1.cotejar_autenticacion_historica_auditoria_v1(
           p_vinculo_canonico) END) IS NOT TRUE THEN
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
    SELECT asiento_ad207.secuencia_previa, asiento_ad207.anterior_sha256 INTO STRICT v_secuencia, v_anterior FROM vec_autorizacion_atestada_v3.reservar_asiento_auditoria_v5() asiento_ad207;
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
    NULL; /* AD207: la cabeza la avanza el sellado diferido */
    RETURN QUERY SELECT v_ref,v_secuencia,v_huella,p_orden ->> 'correlacion_ref',
        v_instante;
END
$function$;

RESET ROLE;
DO $post$
DECLARE
 firma constant regprocedure:='vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(bytea,bytea,jsonb)'::regprocedure;
 anterior jsonb; actual jsonb;
BEGIN
 SELECT meta INTO STRICT anterior FROM pg_temp.ad299_meta;
 SELECT to_jsonb(p)-'prosrc' INTO STRICT actual FROM pg_proc p WHERE p.oid=firma;
 IF actual IS DISTINCT FROM anterior
 OR encode(sha256(convert_to(pg_get_functiondef(firma),'UTF8')),'hex')
  IS DISTINCT FROM '93f9c7eaa298d878e3c3a2f637a67003582e25cdc24f91d80100791dfed718c0' THEN
  RAISE EXCEPTION 'AD299: posimagen o metadatos divergentes' USING ERRCODE='55000';
 END IF;
END $post$;
COMMIT;
