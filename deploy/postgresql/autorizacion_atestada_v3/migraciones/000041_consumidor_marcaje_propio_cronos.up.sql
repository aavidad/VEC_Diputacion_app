\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:mutacion:perfiles',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000041',0));

-- Añade un perfil consumidor; no duplica validación criptográfica de AD3.
DO $ampliar$
DECLARE
    v_def text; v_acl aclitem[]; v_owner oid;
    v_runtime_inicio text := $inicio$       OR session_user = current_user
       OR NOT (
$inicio$;
    v_runtime_fin text := $fin$       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo VEC-AD-3 rechazado';$fin$;
    v_cronos_inicio text := $cronos_inicio$           /* AD3-41 runtime Cronos inicio */
           (p_perfil_mutacion IS DISTINCT FROM 'cronos_marcaje_propio' AND (
$cronos_inicio$;
    v_cronos_fin text := $cronos_fin$           /* AD3-41 runtime Cronos cierre */
           )) OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'cronos_marcaje_propio'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
                   AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
                   AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
               AND (SELECT count(*) FROM pg_auth_members a WHERE a.member=session_user::regrole)=1
               AND EXISTS (SELECT 1 FROM pg_auth_members a
                   WHERE a.member=session_user::regrole AND a.roleid='vec_cronos_v1_ejecutor'::regrole
                     AND a.inherit_option AND NOT a.set_option AND NOT a.admin_option)
               AND NOT EXISTS (SELECT 1 FROM pg_roles r
                   WHERE r.oid<>session_user::regrole AND r.oid<>'vec_cronos_v1_ejecutor'::regrole
                     AND pg_has_role(session_user,r.oid,'MEMBER'))
           )
$cronos_fin$;
    v_marca text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    v_extension text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'cronos_marcaje_propio'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM
                   'vec_cronos_v1.marcaje_propio.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'cronos.marcaje.propio.registrar'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'cronos'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'marcaje_propio'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'registrar_marcaje_propio'
               AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$perfil$;
BEGIN
    IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL THEN
        RAISE EXCEPTION 'marcaje propio Cronos ya instalada' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner
      INTO STRICT v_def,v_acl,v_owner
      FROM pg_proc p
     WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
       AND p.prosecdef;
    IF length(v_def)-length(replace(v_def,v_marca,''))<>length(v_marca)
       OR strpos(v_def,'cronos_marcaje_propio')<>0
       OR strpos(v_def,'p_perfil_mutacion IS NOT DISTINCT FROM ''peticion_centro''')=0
       OR strpos(v_def,'(d ->> ''valida_hasta'')::timestamptz')=0 THEN
        RAISE EXCEPTION 'núcleo incompatible para marcaje propio Cronos' USING ERRCODE='55000';
    END IF;
    IF length(v_def)-length(replace(v_def,v_runtime_inicio,''))<>length(v_runtime_inicio)
       OR length(v_def)-length(replace(v_def,v_runtime_fin,''))<>length(v_runtime_fin) THEN
        RAISE EXCEPTION 'guardia runtime incompatible AD3-41' USING ERRCODE='55000';
    END IF;
    v_def:=replace(v_def,v_runtime_inicio,v_runtime_inicio||v_cronos_inicio);
    v_def:=replace(v_def,v_runtime_fin,v_cronos_fin||v_runtime_fin);
    EXECUTE replace(v_def,v_marca,v_extension||v_marca);
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl
       OR (SELECT proowner FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_owner THEN
        RAISE EXCEPTION 'marcaje propio Cronos alteró autoridad del núcleo' USING ERRCODE='55000';
    END IF;
END
$ampliar$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(
    p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
    p_persona_version numeric,p_perfil_version numeric,
    p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS TABLE (
    decision_ref text,efecto_ref text,huella_efecto_sha256 text,
    consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean
)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path = pg_catalog SET lock_timeout = '2s'
AS $funcion$
DECLARE v_consumo record;
BEGIN
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
        'cronos_marcaje_propio',p_capacidad,p_decision,p_motivo,p_contexto,
        p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
    IF v_consumo.consumo_nuevo IS NOT TRUE THEN
        RAISE EXCEPTION 'marcaje propio Cronos requiere consumo nuevo' USING ERRCODE='PC003';
    END IF;
    RETURN QUERY SELECT v_consumo.decision_ref,v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,v_consumo.consumo_huella_sha256,
        v_consumo.auditoria_ref,v_consumo.consumida_en,true;
END
$funcion$;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    FROM PUBLIC,vec_autorizacion_atestada_v3_consumidor,vec_autorizacion_atestada_v3_emisor,
    vec_cronos_v1_ejecutor,vec_cronos_v1_migrador;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_cronos_marcaje_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
    TO vec_cronos_v1_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_cronos_v1_propietario;

-- Sólo se añade/retira esta audiencia; las variantes anteriores se conservan.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia_cronos$
DECLARE
    definicion text; reconstruida text; audiencias text[];
    admitidas text[]:=ARRAY['vec_contratacion_temporal.confirmar_alta_atestada.v1',
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
        'vec_contratacion_temporal.despacho_correo_llamamiento.v1',
        'vec_contratacion_temporal.resultado_correo_llamamiento.v1',
        'vec_dietas_rutas_v1.acceso.v1',
        'vec_dietas_v1.borrador_propio.v1',
        'vec_cronos_v1.marcaje_propio.v1'];
BEGIN
    SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT definicion FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'
       AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
    SELECT array_agg(captura[1] ORDER BY orden) INTO audiencias
      FROM regexp_matches(definicion,'''([^'']+)''::text','g') WITH ORDINALITY AS r(captura,orden);
    reconstruida:='CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(audiencias) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))';
    IF definicion IS DISTINCT FROM reconstruida OR cardinality(audiencias)<12
       OR NOT audiencias<@admitidas
       OR NOT ARRAY[
           'vec_contratacion_temporal.confirmar_alta_atestada.v1',
           'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
           'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
           'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
           'vec_personal.alta_ejercicio.v1',
           'vec_contratacion_temporal.incorporacion_ejercicio.v2',
           'vec_personal.lectura_incorporacion.v1',
           'vec_personal.lectura_incorporacion.v2',
           'vec_contratacion_temporal.anotacion_administrativa.v1',
           'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1',
           'vec_contratacion_temporal.despacho_correo_llamamiento.v1',
           'vec_contratacion_temporal.resultado_correo_llamamiento.v1']::text[]<@audiencias
       OR (SELECT count(*) FROM unnest(audiencias) a WHERE a LIKE 'vec_contexto_actor.%') NOT IN (0,4)
       OR cardinality(audiencias)<>(SELECT count(DISTINCT a) FROM unnest(audiencias) a)
       OR NOT 'vec_contratacion_temporal.despacho_correo_llamamiento.v1'=ANY(audiencias)
       OR 'vec_cronos_v1.marcaje_propio.v1'=ANY(audiencias) THEN
        RAISE EXCEPTION 'gobierno de audiencias incompatible AD3-41' USING ERRCODE='55000';
    END IF;
    audiencias:=array_append(audiencias,'vec_cronos_v1.marcaje_propio.v1');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version '
        ||'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('
        ||array_to_string(ARRAY(SELECT quote_literal(a)
            FROM unnest(audiencias) WITH ORDINALITY u(a,n) ORDER BY n),', ')||'))';
END
$audiencia_cronos$;
COMMIT;
