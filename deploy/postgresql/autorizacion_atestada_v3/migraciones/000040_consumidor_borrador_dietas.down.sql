\set ON_ERROR_STOP on
-- Retirar antes de Dietas000001, sólo sin historia. Las funciones del módulo
-- quedan cerradas por falta de consumidor hasta su retirada vacía posterior.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:mutacion:perfiles',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000040',0));
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $retirar$
DECLARE
    v_def text; v_acl aclitem[]; v_owner oid;
    v_metadata jsonb; v_dependencias jsonb; v_esperada text;
    v_runtime_inicio text := $inicio$       OR session_user = current_user
       OR NOT (
$inicio$;
    v_runtime_fin text := $fin$       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo VEC-AD-3 rechazado';$fin$;
    v_dietas_inicio text := $dietas_inicio$           /* AD3-40 runtime Dietas inicio */
           (p_perfil_mutacion IS DISTINCT FROM 'borrador_dietas' AND (
$dietas_inicio$;
    v_dietas_fin text := $dietas_fin$           /* AD3-40 runtime Dietas cierre */
           )) OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'borrador_dietas'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
                   AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
                   AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls)
               AND (SELECT count(*) FROM pg_auth_members a WHERE a.member=session_user::regrole)=1
               AND EXISTS (SELECT 1 FROM pg_auth_members a
                   WHERE a.member=session_user::regrole AND a.roleid='vec_dietas_v1_ejecutor'::regrole
                     AND a.inherit_option AND NOT a.set_option AND NOT a.admin_option)
               AND NOT EXISTS (SELECT 1 FROM pg_roles r
                   WHERE r.oid<>session_user::regrole AND r.oid<>'vec_dietas_v1_ejecutor'::regrole
                     AND pg_has_role(session_user,r.oid,'MEMBER'))
           )
$dietas_fin$;
    v_extension text:=$perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'borrador_dietas'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas_v1.borrador_propio.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND c ->> 'operacion' = ANY (ARRAY[
                   'dietas.borrador.crear_propio',
                   'dietas.borrador.recuperar_propio',
                   'dietas.borrador.listar_propios'
               ])
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'dietas'
               AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'borrador_comision'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'gestionar_borrador_propio'
               AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$perfil$;
BEGIN
    IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
        WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_dietas_v1.borrador_propio.v1'
           OR convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion' IN
              ('dietas.borrador.crear_propio','dietas.borrador.recuperar_propio','dietas.borrador.listar_propios')) THEN
        RAISE EXCEPTION 'reversión denegada: historia de Dietas' USING ERRCODE='55000';
    END IF;
    SELECT pg_get_functiondef(p.oid),p.proacl,p.proowner,to_jsonb(p)-'prosrc'
      INTO STRICT v_def,v_acl,v_owner,v_metadata
    FROM pg_proc p
    WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
      AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
      AND p.provolatile='v' AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
    SELECT coalesce(jsonb_agg(to_jsonb(z) ORDER BY z.classid,z.objid,z.objsubid,z.refclassid,z.refobjid,z.refobjsubid,z.deptype),'[]'::jsonb)
      INTO v_dependencias FROM pg_depend z
     WHERE z.classid='pg_proc'::regclass
       AND z.objid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    IF length(v_def)-length(replace(v_def,v_extension,''))<>length(v_extension) THEN
        RAISE EXCEPTION 'núcleo incompatible; no retirar otros perfiles' USING ERRCODE='55000';
    END IF;
    IF length(v_def)-length(replace(v_def,v_dietas_inicio,''))<>length(v_dietas_inicio)
       OR length(v_def)-length(replace(v_def,v_dietas_fin,''))<>length(v_dietas_fin) THEN
        RAISE EXCEPTION 'guardia runtime incompatible AD3-40' USING ERRCODE='55000';
    END IF;
    v_def:=replace(replace(v_def,v_dietas_inicio,''),v_dietas_fin,'');
    v_esperada:=replace(v_def,v_extension,'');
    EXECUTE v_esperada;
    IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p
         WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
          IS DISTINCT FROM v_metadata
       OR pg_get_functiondef('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
          IS DISTINCT FROM v_esperada
       OR (SELECT coalesce(jsonb_agg(to_jsonb(z) ORDER BY z.classid,z.objid,z.objsubid,z.refclassid,z.refobjid,z.refobjsubid,z.deptype),'[]'::jsonb)
           FROM pg_depend z WHERE z.classid='pg_proc'::regclass
             AND z.objid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure)
          IS DISTINCT FROM v_dependencias THEN
        RAISE EXCEPTION 'Dietas alteró metadata o dependencias AD3' USING ERRCODE='55000';
    END IF;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl
       OR (SELECT proowner FROM pg_proc WHERE oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_owner THEN
        RAISE EXCEPTION 'retirada de Dietas alteró autoridad del núcleo' USING ERRCODE='55000';
    END IF;
END
$retirar$;
DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_borrador_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_dietas_v1_propietario;

-- Contrato exacto de audiencias: conserva la variante base/corporativa AD3-39.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencia_dietas$
DECLARE
    definicion text; reconstruida text; audiencias text[]; esperada text;
    base_esperada text[]:=ARRAY[
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
        'vec_contratacion_temporal.despacho_correo_llamamiento.v1',
        'vec_contratacion_temporal.resultado_correo_llamamiento.v1',
        'vec_dietas_rutas_v1.acceso.v1',
        'vec_dietas_v1.borrador_propio.v1'];
    corporativa_esperada text[]:=ARRAY[
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
        'vec_contratacion_temporal.despacho_correo_llamamiento.v1',
        'vec_contratacion_temporal.resultado_correo_llamamiento.v1',
        'vec_dietas_rutas_v1.acceso.v1',
        'vec_dietas_v1.borrador_propio.v1'];
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
    IF definicion IS DISTINCT FROM reconstruida
       OR (audiencias IS DISTINCT FROM base_esperada AND audiencias IS DISTINCT FROM corporativa_esperada) THEN
        RAISE EXCEPTION 'gobierno de audiencias incompatible AD3-40' USING ERRCODE='55000';
    END IF;
    IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
               WHERE audiencia_consumo='vec_dietas_v1.borrador_propio.v1') THEN
        RAISE EXCEPTION 'audiencia Dietas conserva gobierno' USING ERRCODE='55000';
    END IF;
    audiencias:=array_remove(audiencias,'vec_dietas_v1.borrador_propio.v1');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version '
        ||'ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check CHECK (audiencia_consumo IN ('
        ||array_to_string(ARRAY(SELECT quote_literal(a)
            FROM unnest(audiencias) WITH ORDINALITY u(a,n) ORDER BY n),', ')||'))';
    esperada:='CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(audiencias) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))';
    IF (SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
          FROM pg_constraint c
         WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
           AND c.conname='clave_capacidad_version_audiencia_consumo_check')
       IS DISTINCT FROM esperada THEN
        RAISE EXCEPTION 'postimagen de audiencias divergente AD3-40' USING ERRCODE='55000';
    END IF;
END
$audiencia_dietas$;

COMMIT;
