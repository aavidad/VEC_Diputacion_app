\set ON_ERROR_STOP on
-- Focal para clúster aislado con AD3-34 instalada. Ejecutar como DBA.
-- Sólo inspecciona ACL y rechazos anteriores al consumo: no inventa material
-- firmado, no usa dobles del núcleo V3 y no acredita el caso positivo real.
BEGIN;
SET LOCAL search_path = pg_catalog;
DO $acl$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    admin oid := 'vec_administracion_propietario'::regrole;
    rol text;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario
                    AND prosecdef AND provolatile='v' AND pronargdefaults=0
                    AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
       OR NOT COALESCE((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
                AND bool_and(a.grantee IN (propietario,admin) AND a.grantor=propietario
                             AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
            FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
            WHERE p.oid=f),false) THEN
        RAISE EXCEPTION 'AD3-34: fachada o ACL incorrectas';
    END IF;
    FOREACH rol IN ARRAY ARRAY[
        'vec_administracion_ejecutor','vec_administracion_migrador',
        'vec_contratacion_temporal_propietario','vec_contratacion_temporal_ejecutor',
        'vec_bolsa_llamamientos_propietario','vec_bolsa_llamamientos_ejecutor',
        'vec_personal_propietario','vec_personal_ejecutor'
    ] LOOP
        IF has_function_privilege(rol,f,'EXECUTE') THEN
            RAISE EXCEPTION 'AD3-34: consumidor expuesto a otra autoridad o runtime';
        END IF;
    END LOOP;
    IF (SELECT encode(sha256(convert_to(prosrc,'UTF8')),'hex') FROM pg_proc
         WHERE oid='vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
       IS DISTINCT FROM '6d5717bdbed38b088810722f0d0fce3278fa53dc2adb4e0b296a779d775e8f07' THEN
        RAISE EXCEPTION 'AD3-34: fachada previa de actualizacion alterada';
    END IF;
END $acl$;

SET SESSION AUTHORIZATION vec_administracion_ejecutor;
DO $runtime$
BEGIN
    BEGIN
        PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(
            NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
        RAISE EXCEPTION 'AD3-34: ejecutor llamó directamente al consumidor';
    EXCEPTION WHEN insufficient_privilege THEN NULL;
    END;
END $runtime$;
RESET SESSION AUTHORIZATION;

-- El propietario ADMIN tiene EXECUTE, pero las entradas malformadas deben
-- rechazarse por la fachada antes del núcleo. Se comprueba también el mensaje
-- exacto para no confundir el rechazo con otra frontera posterior.
SET SESSION AUTHORIZATION vec_administracion_propietario;
DO $negativos$
DECLARE
    base jsonb := '{
        "accion":"administracion.configuracion_correo.consultar",
        "modulo_id":"vec.module.administracion",
        "tipo_recurso":"configuracion_correo_administracion",
        "recurso_ref":"configuracion:smtp:diputacion",
        "finalidad":"administrar_integraciones",
        "campos_permitidos":["configurada","host","modo_autenticacion","modo_tls","puerto","referencia_ca","remitente_fijo","secreto_configurado","server_name","tiempo_maximo_ms","usuario","version"],
        "obligaciones":[]
    }'::jsonb;
    dummy bytea := convert_to('{}','UTF8');
    entrada bytea;
    variante jsonb;
    clave text;
    mensaje text;
    n integer;
BEGIN
    FOR n IN 1..10 LOOP
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(
                CASE WHEN n=1 THEN NULL ELSE dummy END,
                CASE WHEN n=2 THEN NULL ELSE convert_to(base::text,'UTF8') END,
                CASE WHEN n=3 THEN NULL ELSE dummy END,
                CASE WHEN n=4 THEN NULL ELSE dummy END,
                CASE WHEN n=5 THEN NULL ELSE 1::numeric END,
                CASE WHEN n=6 THEN NULL ELSE 1::numeric END,
                CASE WHEN n=7 THEN NULL ELSE dummy END,
                CASE WHEN n=8 THEN NULL ELSE dummy END,
                CASE WHEN n=9 THEN NULL ELSE dummy END,
                CASE WHEN n=10 THEN NULL ELSE dummy END);
            RAISE EXCEPTION 'AD3-34: parametro SQL NULL aceptado: %',n;
        EXCEPTION WHEN insufficient_privilege THEN
            GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
            IF mensaje<>'consulta SMTP: material de decision no admitido' THEN
                RAISE EXCEPTION 'AD3-34: NULL rechazado por frontera equivocada';
            END IF;
        END;
    END LOOP;
    FOR clave IN SELECT jsonb_object_keys(base) LOOP
        FOR variante IN SELECT base-clave UNION ALL
                        SELECT jsonb_set(base,ARRAY[clave],'null'::jsonb) LOOP
            BEGIN
                PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(
                    dummy,convert_to(variante::text,'UTF8'),dummy,dummy,1,1,dummy,dummy,dummy,dummy);
                RAISE EXCEPTION 'AD3-34: campo nominal ausente/nulo aceptado';
            EXCEPTION WHEN insufficient_privilege THEN
                GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
                IF mensaje<>'consulta SMTP: material de decision no admitido' THEN
                    RAISE EXCEPTION 'AD3-34: campo rechazado por frontera equivocada';
                END IF;
            END;
        END LOOP;
    END LOOP;
    FOR entrada IN SELECT v FROM (VALUES
        (''::bytea),(decode('ff','hex')),(convert_to('{','UTF8')),
        (convert_to('null','UTF8')),(convert_to('[]','UTF8')),
        (convert_to(repeat(' ',524289),'UTF8')),
        (convert_to((base||'{"accion":"administracion.configuracion_correo.actualizar"}'::jsonb)::text,'UTF8')),
        (convert_to((base||'{"recurso_ref":"configuracion:smtp:otra"}'::jsonb)::text,'UTF8')),
        (convert_to((base||'{"obligaciones":["doble_control"]}'::jsonb)::text,'UTF8')),
        (convert_to((base||'{"obligaciones":{}}'::jsonb)::text,'UTF8')),
        (convert_to((base||'{"campos_permitidos":[]}'::jsonb)::text,'UTF8')),
        (convert_to(jsonb_set(base,'{campos_permitidos}',base->'campos_permitidos'||'"secreto_cifrado"'::jsonb)::text,'UTF8'))
    ) casos(v) LOOP
        BEGIN
            PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_correo_admin_v3_atestada(
                dummy,entrada,dummy,dummy,1,1,dummy,dummy,dummy,dummy);
            RAISE EXCEPTION 'AD3-34: decision malformada o fuera de contrato aceptada';
        EXCEPTION WHEN insufficient_privilege THEN
            GET STACKED DIAGNOSTICS mensaje=MESSAGE_TEXT;
            IF mensaje<>'consulta SMTP: material de decision no admitido' THEN
                RAISE EXCEPTION 'AD3-34: material rechazado por frontera equivocada';
            END IF;
        END;
    END LOOP;
END $negativos$;
RESET SESSION AUTHORIZATION;
ROLLBACK;
-- Pendiente en el arnés real: material V3 sintético firmado/gobernado GET,
-- consumo nuevo y rechazo del replay, caducidad/revocación, audiencia PUT,
-- combinación de perfiles técnicos, bytes de material divergentes, y rollback
-- conjunto ADMIN2/T13/3. Este focal negativo no acredita esos recorridos.
