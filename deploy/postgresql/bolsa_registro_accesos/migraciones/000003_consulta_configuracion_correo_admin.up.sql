\set ON_ERROR_STOP on
-- Fuente candidata: T13/3. No instala ni habilita el append genérico.
BEGIN;
SET LOCAL ROLE vec_bolsa_accesos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion.dependencias.configuracion_correo.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_bolsa_registro_accesos:migracion:000003', 0));

DO $dependencias$
DECLARE
    propietario oid := 'vec_bolsa_accesos_propietario'::regrole;
    admin oid := 'vec_administracion_propietario'::regrole;
    rol text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_namespace
                    WHERE nspname='vec_bolsa_registro_accesos'
                      AND nspowner=propietario)
       OR to_regprocedure('vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(jsonb,text,text,bigint)') IS NOT NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc
                       WHERE oid=to_regprocedure('vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)')
                         AND proowner=propietario AND NOT prosecdef
                         AND proconfig=ARRAY['search_path=pg_catalog'])
       OR NOT EXISTS (SELECT 1 FROM pg_proc
                       WHERE oid=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')
                         AND proowner='vec_autorizacion_atestada_v3_propietario'::regrole
                         AND prosecdef)
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=admin
                       AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls
                       AND NOT rolcreatedb AND NOT rolcreaterole
                       AND NOT rolreplication AND NOT rolinherit) THEN
        RAISE EXCEPTION 'T13/3: dependencias incompatibles' USING ERRCODE='55000';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_administracion_propietario','vec_administracion_ejecutor',
        'vec_administracion_migrador'
    ] LOOP
        IF has_function_privilege(rol,
               'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
           OR has_function_privilege(rol,
               'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)','EXECUTE')
           OR has_table_privilege(rol,
               'vec_bolsa_registro_accesos.registro_acceso','INSERT,UPDATE,DELETE,TRUNCATE') THEN
            RAISE EXCEPTION 'T13/3: acceso administrativo directo incompatible'
                USING ERRCODE='55000';
        END IF;
    END LOOP;
END $dependencias$;

-- Frontera de autoridad:
-- 1. Sólo el propietario NOLOGIN de ADMIN recibe EXECUTE. El ejecutor ADMIN
--    no puede llamar este wrapper ni escribir/consultar tablas T13.
-- 2. consultar_configuracion_correo_v2 debe invocarlo en su propia transacción
--    serializable, DESPUÉS del consumidor V3 nominal de consulta (consumo_nuevo
--    verdadero), del cotejo exacto de huella_efecto_sha256 con el hash del
--    material y de la lectura privada de la vista. No se reutiliza AD3-33.
-- 3. p_decision_ref y p_consumo_ref proceden del record devuelto por ese
--    consumidor; p_entrada es p_negocio.auditoria. p_version es la versión
--    realmente observada: cero sólo cuando no existe configuración.
--    No son una capacidad independiente: aceptar parámetros desde runtime o
--    añadir otra fachada ADMIN que los admita rompe esta frontera y exige revisión.
-- 4. Un error T13 aborta el consumo V3 y no entrega la vista. Este wrapper
--    no abre otra transacción, no reconsume V3 y no lee tablas de otro módulo.
CREATE FUNCTION vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(
    p_entrada jsonb, p_decision_ref text, p_consumo_ref text, p_version bigint
) RETURNS jsonb
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s' SET row_security = on
AS $funcion$
DECLARE
    entrada_t13 jsonb;
BEGIN
    IF NOT pg_has_role(session_user,'vec_administracion_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_administracion_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_administracion_migrador','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_accesos_propietario','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_accesos_migrador','MEMBER')
       OR current_setting('transaction_isolation') <> 'serializable'
       OR current_setting('transaction_read_only') <> 'off' THEN
        RAISE EXCEPTION 'T13/3: consulta administrativa no autorizada'
            USING ERRCODE='42501';
    END IF;
    IF p_entrada IS NULL OR octet_length(p_entrada::text)>16384
       OR vec_bolsa_registro_accesos.objeto_tipos_exactos_v1(p_entrada,'{
           "id":"string","seq":"number","signature":"string",
           "actor_id":"string","actor_profile":"string","actor_roles":"array",
           "auth_method":"string","auth_assurance":"string","purpose":"string",
           "action":"string","module_id":"string","subject_ref":"string",
           "result":"string",
           "correlation_ref":"string","occurred_at":"string"
       }'::jsonb) IS NOT TRUE THEN
        RAISE EXCEPTION 'T13/3: auditoria administrativa incompleta'
            USING ERRCODE='22023';
    END IF;
    IF p_decision_ref IS NULL OR p_decision_ref !~ '^[^*?[:space:][:cntrl:]]{1,160}$'
       OR p_consumo_ref IS NULL OR p_consumo_ref !~ '^[^*?[:space:][:cntrl:]]{1,160}$'
       OR p_version IS NULL OR p_version NOT BETWEEN 0 AND 9007199254740991
       OR p_entrada->>'id' <> '' OR p_entrada->>'seq' <> '0'
       OR p_entrada->>'signature' <> ''
       OR p_entrada->>'actor_id' !~ '^hmac-sha256:[a-z][a-z0-9._-]{0,63}:[0-9a-f]{64}$'
       OR split_part(p_entrada->>'actor_id',':',3)=repeat('0',64)
       OR p_entrada->>'actor_profile' !~ '^[^*?[:space:][:cntrl:]]{1,160}$'
       OR jsonb_array_length(p_entrada->'actor_roles') NOT BETWEEN 1 AND 16
       OR EXISTS (SELECT 1 FROM jsonb_array_elements(p_entrada->'actor_roles') r
                   WHERE jsonb_typeof(r)<>'string'
                      OR r#>>'{}' !~ '^[^*?[:space:][:cntrl:]]{1,128}$')
       OR p_entrada->>'auth_method' NOT IN ('certificado','dnie')
       OR p_entrada->>'auth_assurance' <> 'alto'
       OR p_entrada->>'purpose' <> 'administrar_integraciones'
       OR p_entrada->>'action' <> 'administracion.configuracion_correo.consultar'
       OR p_entrada->>'module_id' <> 'vec.module.administracion'
       OR p_entrada->>'subject_ref' <> 'configuracion:smtp:diputacion'
       OR p_entrada->>'result' <> 'permitido'
       OR p_entrada->>'correlation_ref' !~ '^correlacion_[0-9a-f]{32}$'
       OR p_entrada->>'correlation_ref' = 'correlacion_'||repeat('0',32)
       OR p_entrada->>'occurred_at' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}([.][0-9]{1,9})?Z$' THEN
        RAISE EXCEPTION 'T13/3: auditoria administrativa invalida'
            USING ERRCODE='22023';
    END IF;
    -- Incorporación explícita de la versión observada al AuditEntry atestado;
    -- no se presupone versión uno en ausencia de configuración. Entrada
    -- canónica T13; integridad, recibo y retención los produce la autoridad T13.
    -- No se rellenan actor, perfil, autenticación o finalidad ausentes.
    entrada_t13 := jsonb_build_object(
        'actor_id',p_entrada->>'actor_id',
        'actor_profile',p_entrada->>'actor_profile',
        'actor_roles',p_entrada->'actor_roles',
        'represented_subject_id','',
        'auth_method',p_entrada->>'auth_method',
        'auth_assurance',p_entrada->>'auth_assurance',
        'authorization_ref',p_decision_ref,
        'purpose',p_entrada->>'purpose','action',p_entrada->>'action',
        'module_id',p_entrada->>'module_id','subject_ref',p_entrada->>'subject_ref',
        'object_version',p_version,'expediente_ref','','document_ref','',
        'rule_ref','','reason','','result',p_entrada->>'result','before_hash','','after_hash','',
        'correlation_ref',p_entrada->>'correlation_ref',
        'metadata',jsonb_build_object('consumo_ref',p_consumo_ref),
        'occurred_at',p_entrada->>'occurred_at');
    RETURN vec_bolsa_registro_accesos.registrar_interno_v1(entrada_t13);
END $funcion$;
REVOKE ALL ON FUNCTION vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(jsonb,text,text,bigint) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_bolsa_registro_accesos TO vec_administracion_propietario;
GRANT EXECUTE ON FUNCTION vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(jsonb,text,text,bigint) TO vec_administracion_propietario;

DO $acl_final$
DECLARE
    f oid := 'vec_bolsa_registro_accesos.registrar_consulta_configuracion_correo_admin_v1(jsonb,text,text,bigint)'::regprocedure;
    propietario oid := 'vec_bolsa_accesos_propietario'::regrole;
    admin oid := 'vec_administracion_propietario'::regrole;
BEGIN
    IF NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
            AND bool_and(a.grantee IN (propietario,admin) AND a.grantor=propietario
                AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
        FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
        WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'T13/3: ACL final incompatible' USING ERRCODE='55000';
    END IF;
END $acl_final$;
COMMIT;
