\set ON_ERROR_STOP on
-- AD3-33 CANDIDATA: numeración pendiente de confirmación del integrador.
-- Fuente sobre AD3-32; no acredita instalación. ADMIN1 consume esta fachada
-- en la transacción de configuración, sobre cifrado y auditoría. Sin SMTP.
-- Requiere el rol administrativo ya provisionado; no lo crea ni lo amplía.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_administracion.dependencias.configuracion_correo.v1', 0));
SELECT pg_advisory_xact_lock(hashtextextended(
    'vec_autorizacion_atestada_v3:migracion:000033', 0));

DO $precondicion$
DECLARE
    propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    ct oid := 'vec_contratacion_temporal_propietario'::regrole;
    admin oid := 'vec_administracion_propietario'::regrole;
    rol_admin text;
    anterior oid := to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_despacho_correo_llamamiento_ct_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
    FOREACH rol_admin IN ARRAY ARRAY[
        'vec_administracion_ejecutor','vec_administracion_migrador'
    ] LOOP
        IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=rol_admin
                       AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
                       AND NOT rolcreatedb AND NOT rolcreaterole
                       AND NOT rolreplication AND NOT rolbypassrls) THEN
            RAISE EXCEPTION 'AD3-33: rol técnico administrativo ausente o incompatible'
                USING ERRCODE='55000';
        END IF;
    END LOOP;
    IF current_user <> 'vec_autorizacion_atestada_v3_propietario'
       OR getdatabaseencoding() <> 'UTF8'
       OR NOT EXISTS (SELECT 1 FROM pg_namespace
                      WHERE nspname='vec_autorizacion_atestada_v3' AND nspowner=propietario)
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario
                      AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
                      AND NOT rolcreatedb AND NOT rolcreaterole
                      AND NOT rolreplication AND NOT rolbypassrls)
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=ct
                      AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
                      AND NOT rolcreatedb AND NOT rolcreaterole
                      AND NOT rolreplication AND NOT rolbypassrls)
       OR NOT has_schema_privilege(ct, 'vec_autorizacion_atestada_v3', 'USAGE')
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=admin
                      AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
                      AND NOT rolcreatedb AND NOT rolcreaterole
                      AND NOT rolreplication AND NOT rolbypassrls)
       OR NOT has_schema_privilege(admin, 'vec_autorizacion_atestada_v3', 'USAGE')
       OR EXISTS (SELECT 1 FROM pg_proc
                  WHERE pronamespace='vec_autorizacion_atestada_v3'::regnamespace
                    AND proname='registrar_y_consumir_configuracion_correo_admin_v3_atestada')
       OR anterior IS NULL
       OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=anterior
                      AND proowner=propietario AND prosecdef AND provolatile='v'
                      AND pronargdefaults=0
                      AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s']
                      AND encode(sha256(convert_to(prosrc,'UTF8')),'hex')=
                          'd4da65fd5a27a6f22fa610e0465b099e98739d0c5eb338095b54efba8f3108f3')
       OR NOT COALESCE((
           SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
                  AND bool_and(a.grantee IN (propietario,ct) AND a.grantor=propietario
                               AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_proc p CROSS JOIN LATERAL aclexplode(
                 coalesce(p.proacl,acldefault('f',p.proowner))) a
            WHERE p.oid=anterior
       ),false) THEN
        RAISE EXCEPTION 'AD3-33: propietarios o fachada AD3-32 incompatibles'
            USING ERRCODE='55000';
    END IF;
END $precondicion$;

-- Una única ampliación nominal del núcleo vigente. No se copia un núcleo
-- antiguo: se conserva su definición, metadata, ACL y dependencias restantes.
DO $ampliar$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    definicion text; esperada text; metadata jsonb; dependencias jsonb;
    marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    -- Barrera de sesión exacta heredada de AD3-11 y conservada por AD3-32.
    -- ADMIN tiene una rama propia; nunca necesita membresía de ejecución CT.
    runtime_anterior text := $runtime_anterior$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
           )
       ) THEN$runtime_anterior$;
    runtime_nuevo text := $runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'configuracion_correo_admin'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'bolsa_llamamiento'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'configuracion_correo_admin'
               AND pg_catalog.pg_has_role(
                   session_user, 'vec_administracion_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_administracion_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_administracion_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_bolsa_llamamientos_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_ejecutor', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_personal_migrador', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_propietario', 'MEMBER')
               AND NOT pg_catalog.pg_has_role(
                   session_user, 'vec_autorizacion_atestada_v3_migrador', 'MEMBER')
           )
       ) THEN$runtime_nuevo$;
    perfil32 text := $perfil32$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'despacho_correo_llamamiento_ct'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_contratacion_temporal.despacho_correo_llamamiento.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'contratacion_temporal.llamamiento.correo.despachar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'despacho_correo_llamamiento_contratacion_temporal'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_contratacion_temporal'
           )
$perfil32$;
    extension text := $perfil33$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'configuracion_correo_admin'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_administracion.actualizar_configuracion_correo.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM 'administracion.configuracion_correo.actualizar'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'vec.module.administracion'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'configuracion_correo_administracion'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'administrar_integraciones'
               AND d ->> 'recurso_ref' IS NOT DISTINCT FROM 'configuracion:smtp:diputacion'
               AND c ->> 'efecto_ref' IS NOT DISTINCT FROM 'configuracion:smtp:diputacion'
           )
$perfil33$;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc'
      INTO STRICT definicion,metadata FROM pg_proc p
     WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF length(definicion)-length(replace(definicion,marca,''))<>length(marca)
       OR length(definicion)-length(replace(definicion,perfil32,''))<>length(perfil32)
       OR length(definicion)-length(replace(definicion,runtime_anterior,''))<>length(runtime_anterior)
       OR strpos(definicion,'configuracion_correo_admin')<>0
       OR strpos(definicion,'cierre_administrativo_sin_cese_ct')=0
       OR strpos(definicion,'anotacion_administrativa_ct')=0
       OR strpos(definicion,'incorporacion_ejercicio_ct')=0
       OR strpos(definicion,'lectura_registro_personal_incorporacion_v2')=0
       OR strpos(definicion,'alta_personal_ejercicio')=0
       OR strpos(definicion,'p_perfil_mutacion IS NOT DISTINCT FROM ''resolucion_formalizacion_ct''')=0 THEN
        RAISE EXCEPTION 'AD3-33: núcleo posterior a AD3-32 incompatible'
            USING ERRCODE='55000';
    END IF;
    SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
                    d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
      INTO dependencias FROM pg_depend d
     WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
        OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f);
    esperada := replace(definicion,runtime_anterior,runtime_nuevo);
    esperada := replace(esperada,marca,extension||marca);
    EXECUTE esperada;
    IF pg_get_functiondef(f) IS DISTINCT FROM esperada
       OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR (SELECT jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
                           d.refclassid,d.refobjid,d.refobjsubid,d.deptype)
             FROM pg_depend d
            WHERE (d.classid='pg_proc'::regclass AND d.objid=f)
               OR (d.refclassid='pg_proc'::regclass AND d.refobjid=f)) IS DISTINCT FROM dependencias THEN
        RAISE EXCEPTION 'AD3-33: modificación ajena a la extensión nominal'
            USING ERRCODE='55000';
    END IF;
END $ampliar$;

-- Las dos postimágenes admitidas por AD3-32 difieren en los cuatro perfiles
-- corporativos previos. Se conserva exactamente la variante encontrada.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia$
DECLARE
    definicion text; esperada text; audiencias text[];
    base32 text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_personal.alta_ejercicio.v1',
        'vec_contratacion_temporal.incorporacion_ejercicio.v2',
        'vec_personal.lectura_incorporacion.v1',
        'vec_personal.lectura_incorporacion.v2',
        'vec_contratacion_temporal.anotacion_administrativa.v1',
        'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1',
        'vec_contratacion_temporal.despacho_correo_llamamiento.v1'];
    corporativa32 text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_contexto_actor.publicar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.revocar_organizacion_corporativa_fuente.v1',
        'vec_contexto_actor.publicar_vinculo_corporativo_fuente.v1',
        'vec_contexto_actor.revocar_vinculo_corporativo_fuente.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'vec_personal.alta_ejercicio.v1',
        'vec_contratacion_temporal.incorporacion_ejercicio.v2',
        'vec_personal.lectura_incorporacion.v1',
        'vec_personal.lectura_incorporacion.v2',
        'vec_contratacion_temporal.anotacion_administrativa.v1',
        'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1',
        'vec_contratacion_temporal.despacho_correo_llamamiento.v1'];
BEGIN
    SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT definicion FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'
       AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
    IF definicion='CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(base32) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN
        audiencias := base32;
    ELSIF definicion='CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(corporativa32) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN
        audiencias := corporativa32;
    ELSE
        RAISE EXCEPTION 'AD3-33: CHECK de audiencias no corresponde a AD3-32'
            USING ERRCODE='55000';
    END IF;
    audiencias := array_append(audiencias,'vec_administracion.actualizar_configuracion_correo.v1');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version '
        ||'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
        ||'CHECK (audiencia_consumo IN ('
        ||array_to_string(ARRAY(SELECT quote_literal(a)
            FROM unnest(audiencias) WITH ORDINALITY u(a,n) ORDER BY n),', ')||'))';
    esperada := 'CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(audiencias) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))';
    IF (SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
          FROM pg_constraint c
         WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
           AND c.conname='clave_capacidad_version_audiencia_consumo_check'
           AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[])
       IS DISTINCT FROM esperada THEN
        RAISE EXCEPTION 'AD3-33: postimagen de audiencias divergente'
            USING ERRCODE='55000';
    END IF;
END $audiencia$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(
    p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea,
    p_persona_version numeric, p_perfil_version numeric,
    p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea
) RETURNS TABLE (
    decision_ref text, efecto_ref text, huella_efecto_sha256 text,
    consumo_huella_sha256 text, auditoria_ref text,
    consumida_en timestamptz, consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET lock_timeout='2s'
AS $funcion$
DECLARE consumo record;
BEGIN
    SELECT * INTO STRICT consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'configuracion_correo_admin',
        p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
        p_payload,p_sobre,p_evidencia,p_raiz);
    IF consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'configuración de correo requiere concesión nueva'
            USING ERRCODE='P1102';
    END IF;
    RETURN QUERY SELECT consumo.decision_ref,consumo.efecto_ref,
        consumo.huella_efecto_sha256,consumo.consumo_huella_sha256,
        consumo.auditoria_ref,consumo.consumida_en,true;
END $funcion$;

-- Cerrar las ACL por defecto sólo en esta fachada nueva. No se concede
-- acceso directo al ejecutor administrativo ni se alteran permisos de otros consumidores.
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) FROM PUBLIC;
DO $acl_nueva$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    a record;
BEGIN
    FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p
        CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
        WHERE p.oid=f AND x.grantee<>0 AND x.grantee<>p.proowner LOOP
        EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %I',f::regprocedure,pg_get_userbyid(a.grantee));
    END LOOP;
END $acl_nueva$;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(
    bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea
) TO vec_administracion_propietario;

DO $postcondicion$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_configuracion_correo_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    admin oid := 'vec_administracion_propietario'::regrole;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=propietario
                   AND prosecdef AND provolatile='v' AND pronargdefaults=0
                   AND proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
       OR NOT COALESCE((
           SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
                  AND bool_and(a.grantee IN (propietario,admin) AND a.grantor=propietario
                               AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
             FROM pg_proc p CROSS JOIN LATERAL aclexplode(
                 coalesce(p.proacl,acldefault('f',p.proowner))) a
            WHERE p.oid=f
       ),false) THEN
        RAISE EXCEPTION 'AD3-33: fachada o ACL final divergentes'
            USING ERRCODE='55000';
    END IF;
END $postcondicion$;
COMMIT;
