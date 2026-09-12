\set ON_ERROR_STOP on
-- Sólo para clúster aislado con T13/2 y roles ADMIN instalados, como DBA.
-- Ensayo mecánico del wrapper/ACL/cadena/rollback. La fachada de prueba NO
-- acredita consumo V3 ni el recorrido ADMIN real; desaparece con ROLLBACK.
BEGIN TRANSACTION ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path = pg_catalog;
DO $acl$
DECLARE rol text;
BEGIN
    FOREACH rol IN ARRAY ARRAY[
        'vec_administracion_ejecutor','vec_administracion_migrador',
        'vec_bolsa_accesos_registrador','vec_bolsa_accesos_consultor'
    ] LOOP
        IF has_function_privilege(rol,
            'vec_bolsa_registro_accesos.registrar_cambio_configuracion_correo_admin_v1(jsonb,text,text,bigint)','EXECUTE') THEN
            RAISE EXCEPTION 'wrapper T13 accesible directamente por runtime';
        END IF;
    END LOOP;
    FOREACH rol IN ARRAY ARRAY[
        'vec_administracion_propietario','vec_administracion_ejecutor',
        'vec_administracion_migrador'
    ] LOOP
        IF has_table_privilege(rol,'vec_bolsa_registro_accesos.registro_acceso',
                              'SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
           OR has_function_privilege(rol,
               'vec_bolsa_registro_accesos.registrar_interno_v1(jsonb)','EXECUTE')
           OR has_function_privilege(rol,
               'vec_bolsa_registro_accesos.registrar_acceso_v1(jsonb)','EXECUTE') THEN
            RAISE EXCEPTION 'ADMIN tiene acceso T13 fuera de la fachada nominal';
        END IF;
    END LOOP;
END $acl$;
SELECT count(*) AS registros_antes FROM vec_bolsa_registro_accesos.registro_acceso \gset
CREATE SCHEMA vec_prueba_t13_admin AUTHORIZATION vec_administracion_propietario;
CREATE FUNCTION vec_prueba_t13_admin.puente(a jsonb,d text,c text,v bigint)
RETURNS jsonb LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog
AS $puente$
    SELECT vec_bolsa_registro_accesos.registrar_cambio_configuracion_correo_admin_v1(a,d,c,v)
$puente$;
ALTER FUNCTION vec_prueba_t13_admin.puente(jsonb,text,text,bigint)
    OWNER TO vec_administracion_propietario;
REVOKE ALL ON SCHEMA vec_prueba_t13_admin FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_prueba_t13_admin.puente(jsonb,text,text,bigint) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_prueba_t13_admin TO vec_administracion_ejecutor;
GRANT EXECUTE ON FUNCTION vec_prueba_t13_admin.puente(jsonb,text,text,bigint)
    TO vec_administracion_ejecutor;

SET SESSION AUTHORIZATION vec_administracion_ejecutor;
DO $mecanica$
DECLARE
    entrada jsonb := jsonb_build_object(
        'id','','seq',0,'signature','',
        'actor_id','hmac-sha256:bolsa_accesos_v1:'||repeat('a',64),
        'actor_profile','perfil_admin_sintetico','actor_roles','["admin_sintetico"]'::jsonb,
        'auth_method','certificado','auth_assurance','alto',
        'purpose','administrar_integraciones',
        'action','administracion.configuracion_correo.actualizar',
        'module_id','vec.module.administracion',
        'subject_ref','configuracion:smtp:diputacion','object_version',1,
        'result','accepted','correlation_ref','correlacion_'||repeat('b',32),
        'occurred_at',to_char(transaction_timestamp() AT TIME ZONE 'UTC',
                              'YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
    primera jsonb; repetida jsonb; invalida jsonb; clave text;
BEGIN
    BEGIN
        PERFORM vec_bolsa_registro_accesos.registrar_cambio_configuracion_correo_admin_v1(
            entrada,'decision_sintetica','consumo_sintetico',1);
        RAISE EXCEPTION 'ejecutor llamó T13 directamente';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
    primera := vec_prueba_t13_admin.puente(entrada,'decision_sintetica','consumo_sintetico',1);
    repetida := vec_prueba_t13_admin.puente(entrada,'decision_sintetica','consumo_sintetico',1);
    IF primera IS DISTINCT FROM repetida
       OR primera->>'result' IS DISTINCT FROM 'permitido'
       OR primera->>'authorization_ref' IS DISTINCT FROM 'decision_sintetica'
       OR primera->'metadata' IS DISTINCT FROM '{"consumo_ref":"consumo_sintetico"}'::jsonb
       OR primera->>'object_version' IS DISTINCT FROM '1'
       OR primera->>'actor_id' IS DISTINCT FROM entrada->>'actor_id'
       OR primera->>'id' !~ '^acc_[0-9a-f]{40}$'
       OR primera->>'signature' !~ '^[0-9a-f]{64}$' THEN
        RAISE EXCEPTION 'T13 no conservó entrada/recibo/replay/normalizacion';
    END IF;
    FOR clave IN SELECT jsonb_object_keys(entrada) LOOP
        BEGIN
            PERFORM vec_prueba_t13_admin.puente(entrada-clave,'decision_sintetica','consumo_sintetico',1);
            RAISE EXCEPTION 'campo obligatorio ausente aceptado: %',clave;
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
        BEGIN
            PERFORM vec_prueba_t13_admin.puente(jsonb_set(entrada,ARRAY[clave],'null'::jsonb),
                                               'decision_sintetica','consumo_sintetico',1);
            RAISE EXCEPTION 'campo nulo aceptado: %',clave;
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;
    FOR invalida IN SELECT v FROM (VALUES
        (NULL::jsonb),
        (entrada||'{"secreto":"dato_sintetico_prohibido"}'::jsonb),
        (entrada||'{"actor_id":"persona:referencia_sin_seudonimizar"}'::jsonb),
        (entrada||jsonb_build_object('actor_id','hmac-sha256:bolsa_accesos_v1:'||repeat('0',64))),
        (entrada||'{"actor_profile":""}'::jsonb),
        (entrada||'{"actor_roles":[]}'::jsonb),
        (entrada||'{"actor_roles":["z","a"]}'::jsonb),
        (entrada||'{"actor_roles":["a","a"]}'::jsonb),
        (entrada||'{"actor_roles":[7]}'::jsonb),
        (entrada||'{"auth_method":"sso"}'::jsonb),
        (entrada||'{"auth_assurance":"bajo"}'::jsonb),
        (entrada||'{"purpose":"otra_finalidad"}'::jsonb),
        (entrada||'{"result":"permitido"}'::jsonb),
        (entrada||'{"object_version":2}'::jsonb),
        (entrada||'{"signature":"firma_cliente"}'::jsonb),
        (entrada||'{"correlation_ref":"configuracion_correo"}'::jsonb),
        (entrada||'{"occurred_at":"2026-09-12T00:00:00+00:00"}'::jsonb)
    ) candidatos(v) LOOP
        BEGIN
            PERFORM vec_prueba_t13_admin.puente(invalida,'decision_sintetica','consumo_sintetico',1);
            RAISE EXCEPTION 'auditoria invalida aceptada';
        EXCEPTION WHEN invalid_parameter_value THEN NULL;
        END;
    END LOOP;
    BEGIN
        PERFORM vec_prueba_t13_admin.puente(entrada,NULL,'consumo_sintetico',1);
        RAISE EXCEPTION 'decision ausente aceptada';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
    BEGIN
        PERFORM vec_prueba_t13_admin.puente(entrada,'decision_sintetica',NULL,1);
        RAISE EXCEPTION 'consumo ausente aceptado';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
    BEGIN
        PERFORM vec_prueba_t13_admin.puente(entrada,'decision_sintetica','consumo_sintetico',NULL);
        RAISE EXCEPTION 'version ausente aceptada';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
    BEGIN
        PERFORM vec_prueba_t13_admin.puente(entrada,'otra_decision','consumo_sintetico',1);
        RAISE EXCEPTION 'replay divergente aceptado';
    EXCEPTION WHEN invalid_parameter_value THEN NULL;
    END;
END $mecanica$;
RESET SESSION AUTHORIZATION;
DO $cadena$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_registro_accesos.registro_acceso
        WHERE firma<>encode(sha256(decode(firma_anterior,'hex')||registro_canonico),'hex')) THEN
        RAISE EXCEPTION 'cadena T13 divergente';
    END IF;
END $cadena$;
ROLLBACK;
SELECT count(*) = :'registros_antes'::bigint AS rollback_intacto
  FROM vec_bolsa_registro_accesos.registro_acceso \gset
\if :rollback_intacto
\else
    \echo 'T13/2: ROLLBACK dejó efectos'
    SELECT 1/0;
\endif
-- Pendiente E2E real: emitir material V3 ADMIN, ejecutar guardar_configuracion
-- con la llamada nominal y verificar consumo+CAS+outbox+T13; forzar fallo T13
-- y confirmar que ninguno de los cuatro efectos persiste. Este archivo no lo simula.
