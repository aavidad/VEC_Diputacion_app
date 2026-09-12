\set ON_ERROR_STOP on
-- Catálogo y codec PURO. El texto HMAC de esta fixture es sólo formato:
-- no firma, no prueba criptográfica, no capacidad y no registro de auditoría.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
DO $acl$
DECLARE f oid; n text; propietario oid:='vec_bolsa_accesos_propietario'::regrole;
    ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
    f:='vec_bolsa_registro_accesos.registrar_resultado_correo_llamamiento_ct_v1(bytea,text,bytea,bytea,text,text,text,boolean,text)'::regprocedure;
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario AND prosecdef
        AND provolatile='v' AND pronargdefaults=0
        AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s','row_security=on'])
       OR NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
            AND bool_and(a.grantee IN(propietario,ct) AND a.grantor=propietario
                AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
            FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'T13/5: fachada/ACL divergente';
    END IF;
    f:='vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(bytea,text,bytea,bytea,text,text,text,boolean)'::regprocedure;
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario AND NOT prosecdef
        AND provolatile='i' AND proconfig=ARRAY['search_path=pg_catalog'])
       OR EXISTS (SELECT 1 FROM pg_proc p,
            LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
            WHERE p.oid=f AND a.grantee<>propietario) THEN
        RAISE EXCEPTION 'T13/5: codec expuesto';
    END IF;
    FOREACH n IN ARRAY ARRAY['vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor','vec_contratacion_temporal_migrador'] LOOP
        IF has_function_privilege(n,'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
           OR has_function_privilege(n,'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)','EXECUTE')
           OR has_table_privilege(n,'vec_bolsa_registro_accesos.registro_acceso','SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') THEN
            RAISE EXCEPTION 'T13/5: autoridad genérica accesible a CT';
        END IF;
    END LOOP;
END $acl$;

DO $codec$
DECLARE
    solicitud text := $sol${"OrganizacionRef":"organizacion:prueba","ExpedienteRef":"expediente:prueba","LlamamientoRef":"llamamiento:prueba","ComunicacionRef":"comunicacion:prueba","IntencionEnvioRef":"outbox:prueba","IntentoRef":"intento-correo:prueba","SolicitudHuella":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","Estado":"aceptado_por_relay","PlantillaRef":"llamamiento_rrhh_v1","VersionEsperada":1}$sol$;
    auditoria text := $aud${"id":"","seq":0,"signature":"","actor_id":"hmac-sha256:prueba:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","actor_profile":"perfil:rrhh","actor_roles":["rol-version:rrhh:1"],"auth_method":"certificado","auth_assurance":"alto","purpose":"gestionar_contratacion_temporal","action":"contratacion_temporal.llamamiento.correo.registrar_resultado","module_id":"vec.module.contratacion_temporal","subject_ref":"intento-correo:prueba","object_version":2,"expediente_ref":"expediente:prueba","result":"accepted","correlation_ref":"correlacion:resultado:prueba","occurred_at":"2026-09-12T12:00:00.000000Z"}$aud$;
    decision text := $dec${"decision_ref":"decision:resultado:prueba","concedida":true,"codigo":"concedida","principal_id":"persona:prueba","perfil_activo_ref":"perfil:rrhh","version_rol_ref":"rol-version:rrhh:1","accion":"contratacion_temporal.llamamiento.correo.registrar_resultado","modulo_id":"contratacion_temporal","tipo_recurso":"resultado_correo_llamamiento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal","recurso_ref":"intento-correo:prueba","contexto_recurso_huella_sha256":"7fbc2ff1a01236eada45fc8233fbca7cdaccbeb26ace5c9cf6bf13ca0e829178","correlacion_ref":"correlacion:resultado:prueba","vinculo_autenticacion_actor":{"metodo_observado":"certificado","garantia_observada":"alto"}}$dec$;
    contexto text := $ctx${"principal_ref":"persona:prueba","perfil_activo_ref":"perfil:rrhh","metodo":"certificado","garantia":"alto"}$ctx$;
    entrada jsonb; mala text; dc text; denegado boolean; i integer;
    consumo text:=repeat('c',64); ref_consumo text:='aud_v3_'||repeat('c',32);
BEGIN
    IF encode(sha256(convert_to(auditoria,'UTF8')),'hex')<>'54b4cfbaa4a35b8a29edb580f18f9a2cf0702e4b94ca728902a0622306497b3e' THEN
        RAISE EXCEPTION 'T13/5: vector canónico auditoría divergente';
    END IF;
    entrada:=vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
        convert_to(auditoria,'UTF8'),solicitud,convert_to(decision,'UTF8'),convert_to(contexto,'UTF8'),
        'decision:resultado:prueba',ref_consumo,consumo,false);
    IF entrada->>'authorization_ref'<>'decision:resultado:prueba'
       OR entrada->>'after_hash'<>'c5891f8d86ba7fd0382918af240e3f0a87d2fe8d88b4cc0c1b6da98f2c57165a'
       OR entrada#>>'{metadata,estado_smtp}'<>'aceptado_por_relay'
       OR entrada#>>'{metadata,recuperado}'<>'false'
       OR entrada->>'result'<>'permitido'
       OR entrada->>'subject_ref'<>'intento-correo:prueba' THEN
        RAISE EXCEPTION 'T13/5: proyección de auditoría divergente';
    END IF;
    entrada:=vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
        convert_to(auditoria,'UTF8'),solicitud,convert_to(decision,'UTF8'),convert_to(contexto,'UTF8'),
        'decision:resultado:prueba',ref_consumo,consumo,true);
    IF entrada#>>'{metadata,recuperado}'<>'true' THEN RAISE EXCEPTION 'T13/5: replay mal identificado'; END IF;
    FOREACH mala IN ARRAY ARRAY[
        NULL::text,'{}','[]','{',' '||auditoria,auditoria::jsonb::text,
        replace(auditoria,'"seq":0','"seq":0.0'),
        replace(auditoria,'"seq":0','"seq":1'),
        replace(auditoria,'"signature":""','"signature":"anticipada"'),
        replace(auditoria,repeat('b',64),repeat('0',64)),
        replace(auditoria,'hmac-sha256:prueba:','sha256:'),
        replace(auditoria,'"actor_roles":["rol-version:rrhh:1"]','"actor_roles":[]'),
        replace(auditoria,'"actor_roles":["rol-version:rrhh:1"]','"actor_roles":["rol-version:rrhh:1","rol-version:rrhh:1"]'),
        replace(auditoria,'"object_version":2','"object_version":1'),
        replace(auditoria,'"result":"accepted"','"result":"delivered"'),
        replace(auditoria,'.000000Z','.000Z'),
        replace(auditoria,'2026-09-12','2026-02-30'),
        replace(auditoria,'"id":""','"id":"","id":""'),
        replace(auditoria,'"seq":0','"seq":0,"secreto":"invalido"')
    ] LOOP
        denegado:=false;
        BEGIN
            PERFORM vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
                convert_to(mala,'UTF8'),solicitud,convert_to(decision,'UTF8'),convert_to(contexto,'UTF8'),
                'decision:resultado:prueba',ref_consumo,consumo,false);
        EXCEPTION WHEN SQLSTATE '22023' OR SQLSTATE '42501' THEN denegado:=true;
        END;
        IF NOT denegado THEN RAISE EXCEPTION 'T13/5: auditoría inválida admitida'; END IF;
    END LOOP;
    FOREACH dc IN ARRAY ARRAY[
        replace(decision,'"concedida":true','"concedida":false'),
        replace(decision,'perfil:rrhh','perfil:otro'),
        replace(decision,'rol-version:rrhh:1','rol-version:rrhh:2'),
        replace(decision,'correlacion:resultado:prueba','correlacion:otro'),
        replace(decision,'7fbc2ff1a01236eada45fc8233fbca7cdaccbeb26ace5c9cf6bf13ca0e829178',repeat('0',64)),
        replace(decision,'persona:prueba','persona:otra'),
        replace(decision,'"metodo_observado":"certificado"','"metodo_observado":"dnie"')
    ] LOOP
        denegado:=false;
        BEGIN
            PERFORM vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
                convert_to(auditoria,'UTF8'),solicitud,convert_to(dc,'UTF8'),convert_to(contexto,'UTF8'),
                'decision:resultado:prueba',ref_consumo,consumo,false);
        EXCEPTION WHEN SQLSTATE '42501' THEN denegado:=true;
        END;
        IF NOT denegado THEN RAISE EXCEPTION 'T13/5: vínculo divergente admitido'; END IF;
    END LOOP;
    FOR i IN 1..3 LOOP
        denegado:=false;
        BEGIN
            PERFORM vec_bolsa_registro_accesos.resultado_correo_validar_auditoria_v1(
                convert_to(auditoria,'UTF8'),solicitud,convert_to(decision,'UTF8'),convert_to(contexto,'UTF8'),
                CASE WHEN i=1 THEN 'decision:otra' ELSE 'decision:resultado:prueba' END,
                CASE WHEN i=2 THEN 'aud_v3_'||repeat('d',32) ELSE ref_consumo END,
                CASE WHEN i=3 THEN repeat('0',64) ELSE consumo END,false);
        EXCEPTION WHEN SQLSTATE '42501' THEN denegado:=true;
        END;
        IF NOT denegado THEN RAISE EXCEPTION 'T13/5: consumo divergente admitido'; END IF;
    END LOOP;
END $codec$;
ROLLBACK;
-- Pendiente PG real con V3 genuina: fachada nominal sólo CTowner, auditoría
-- nueva/replay y prueba atómica compartida CT88. Este codec no acredita esa puerta.
