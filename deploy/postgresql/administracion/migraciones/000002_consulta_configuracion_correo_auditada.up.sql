\set ON_ERROR_STOP on
-- ADMIN2 candidata. Requiere la nueva concesión/consumidor nominal de lectura;
-- AD3-33 de actualización no satisface esta dependencia. No habilitar aislada
-- del adaptador GET V3 con COMMIT antes de entregar datos al cliente.
BEGIN;
SET LOCAL ROLE vec_administracion_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion.dependencias.configuracion_correo.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion:configuracion_correo:000002',0));
DO $dependencias$
DECLARE
    lector oid := to_regprocedure('vec_administracion.leer_configuracion_correo_v1()');
    t13 oid := to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(jsonb,text,text,bigint)');
    consumidor oid := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
    IF to_regprocedure('vec_administracion.consultar_configuracion_correo_v2(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=lector
            AND proowner='vec_administracion_propietario'::regrole
            AND prosecdef AND provolatile='s'
            AND proconfig=ARRAY['search_path=pg_catalog']
            AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')=
                'dfb54d46e473d2007573b0feced1b6926cfe64c2ec126e3772e71ed06e6a16f1')
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=t13
            AND proowner='vec_bolsa_accesos_propietario'::regrole AND prosecdef)
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=consumidor
            AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND prosecdef)
       OR NOT COALESCE(has_function_privilege(current_user,t13,'EXECUTE'),false)
       OR NOT COALESCE(has_function_privilege(current_user,consumidor,'EXECUTE'),false) THEN
        RAISE EXCEPTION 'ADMIN2: faltan lector privado, T13/3 o consumidor nominal de consulta'
            USING ERRCODE='55000';
    END IF;
END $dependencias$;

CREATE FUNCTION vec_administracion.consultar_configuracion_correo_v2(
    p_negocio bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
    p_sobre_cose bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s' SET row_security = on
AS $funcion$
DECLARE
    negocio jsonb;
    vista jsonb;
    version_observada bigint;
    consumo record;
    auditoria jsonb;
    decision jsonb;
BEGIN
    IF NOT pg_has_role(session_user,'vec_administracion_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_administracion_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_administracion_migrador','MEMBER')
       OR current_setting('transaction_isolation')<>'serializable'
       OR current_setting('transaction_read_only')<>'off' THEN
        RAISE EXCEPTION 'ADMIN2: consulta administrativa no autorizada' USING ERRCODE='42501';
    END IF;
    IF p_negocio IS NULL OR octet_length(p_negocio) NOT BETWEEN 1 AND 32768 THEN
        RAISE EXCEPTION 'ADMIN2: material de consulta invalido' USING ERRCODE='22023';
    END IF;
    negocio := convert_from(p_negocio,'UTF8')::jsonb;
    IF jsonb_typeof(negocio) IS DISTINCT FROM 'object' THEN
        RAISE EXCEPTION 'ADMIN2: material de consulta invalido' USING ERRCODE='22023';
    END IF;
    IF (SELECT array_agg(k ORDER BY k COLLATE "C") FROM jsonb_object_keys(negocio) k)
           IS DISTINCT FROM ARRAY['auditoria','esquema']::text[]
       OR negocio->>'esquema' IS DISTINCT FROM 'vec.administracion.configuracion-correo.consulta.v1'
       OR jsonb_typeof(negocio->'auditoria') IS DISTINCT FROM 'object' THEN
        RAISE EXCEPTION 'ADMIN2: material de consulta invalido' USING ERRCODE='22023';
    END IF;
    -- La vista se mantiene privada en esta función. El consumidor se llama
    -- después del SELECT para comprobar la concesión fresca antes de registrar
    -- la lectura. Ni una excepción ni un COMMIT fallido autorizan su entrega.
    vista := vec_administracion.leer_configuracion_correo_v1();
    IF vista->'configurada' IS NOT DISTINCT FROM 'false'::jsonb THEN
        IF vista IS DISTINCT FROM '{"configurada":false}'::jsonb THEN
            RAISE EXCEPTION 'ADMIN2: ausencia de configuracion incoherente' USING ERRCODE='55000';
        END IF;
        version_observada := 0;
        vista := jsonb_build_object('configurada',false,'version',0);
    ELSIF vista->'configurada' IS NOT DISTINCT FROM 'true'::jsonb
          AND jsonb_typeof(vista->'version') IS NOT DISTINCT FROM 'number'
          AND vista->>'version' ~ '^[1-9][0-9]{0,15}$' THEN
        version_observada := (vista->>'version')::bigint;
        IF version_observada>9007199254740991 THEN
            RAISE EXCEPTION 'ADMIN2: version fuera de contrato T13' USING ERRCODE='55000';
        END IF;
    ELSE
        RAISE EXCEPTION 'ADMIN2: version de configuracion incoherente' USING ERRCODE='55000';
    END IF;
    SELECT * INTO STRICT consumo FROM
        vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(
            p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
            p_payload,p_sobre_cose,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE
       OR consumo.efecto_ref IS DISTINCT FROM 'configuracion:smtp:diputacion'
       OR consumo.huella_efecto_sha256 IS DISTINCT FROM encode(sha256(convert_to(
            '{"ambitos":{"organizacion_ref":"organizacion:dipgra"},"atributos":{"material_sha256":"'||
            encode(sha256(p_negocio),'hex')||'"}}','UTF8')),'hex') THEN
        RAISE EXCEPTION 'ADMIN2: concesion no ligada a la consulta exacta' USING ERRCODE='42501';
    END IF;
    -- Los bytes ya han sido verificados por AD3. El contrato de esta vista no
    -- acepta campos extra ni una concesión para modificar o recuperar secretos.
    decision := convert_from(p_decision,'UTF8')::jsonb;
    IF decision->>'accion' IS DISTINCT FROM 'administracion.configuracion_correo.consultar'
       OR decision->>'modulo_id' IS DISTINCT FROM 'vec.module.administracion'
       OR decision->>'tipo_recurso' IS DISTINCT FROM 'configuracion_correo_administracion'
       OR decision->>'recurso_ref' IS DISTINCT FROM 'configuracion:smtp:diputacion'
       OR decision->>'finalidad' IS DISTINCT FROM 'administrar_integraciones'
       OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
       OR decision->'campos_permitidos' IS DISTINCT FROM '[
            "configurada","host","modo_autenticacion","modo_tls","puerto",
            "referencia_ca","remitente_fijo","secreto_configurado","server_name",
            "tiempo_maximo_ms","usuario","version"
          ]'::jsonb THEN
        RAISE EXCEPTION 'ADMIN2: concesion fuera de los campos de consulta' USING ERRCODE='42501';
    END IF;
    auditoria := vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(
        negocio->'auditoria',consumo.decision_ref,consumo.auditoria_ref,version_observada);
    RETURN jsonb_build_object('configuracion',vista,'auditoria',auditoria);
END $funcion$;
REVOKE ALL ON FUNCTION vec_administracion.consultar_configuracion_correo_v2(
    bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_administracion.consultar_configuracion_correo_v2(
    bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_administracion_ejecutor;
-- El lector antiguo queda exclusivamente privado al propietario. No hay
-- excepción por preparación PUT: en la postimagen actual ésta sólo cifra el
-- sobre desde VersionEsperada+1. Cualquier futuro consumidor debe usar V3.
REVOKE ALL ON FUNCTION vec_administracion.leer_configuracion_correo_v1()
    FROM PUBLIC,vec_administracion_ejecutor,vec_administracion_migrador;
DO $acl_final$
DECLARE
    f oid := 'vec_administracion.leer_configuracion_correo_v1()'::regprocedure;
    nueva oid := 'vec_administracion.consultar_configuracion_correo_v2(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    propietario oid := 'vec_administracion_propietario'::regrole;
    ejecutor oid := 'vec_administracion_ejecutor'::regrole;
BEGIN
    IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
            aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.oid=f AND a.grantee<>p.proowner)
       OR has_function_privilege('vec_administracion_ejecutor',f,'EXECUTE') THEN
        RAISE EXCEPTION 'ADMIN2: lector sin auditoria sigue abierto' USING ERRCODE='55000';
    END IF;
    IF NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
            AND bool_and(a.grantee IN (propietario,ejecutor) AND a.grantor=propietario
                         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
        FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.oid=nueva),false) THEN
        RAISE EXCEPTION 'ADMIN2: ACL de consulta divergente' USING ERRCODE='55000';
    END IF;
END $acl_final$;
COMMIT;
