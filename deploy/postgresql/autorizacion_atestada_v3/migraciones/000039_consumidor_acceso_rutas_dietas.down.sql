\set ON_ERROR_STOP on
-- AD3-39: acceso nominal al catálogo y cálculo de rutas Dietas.
-- Candidata: instalar sólo tras revisión y prueba PostgreSQL autorizadas.
BEGIN;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000039',0));
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
-- DOWN sólo es válido para una instalación sin gobierno ni consumos de rutas,
-- sin miembros del rol nuevo y después de retirar 041/040. No elimina historia.
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version,
           vec_autorizacion_atestada_v3.atestacion_decision_v3,
           vec_autorizacion_atestada_v3.consumo_decision_v3 IN ACCESS EXCLUSIVE MODE;
DO $sin_historia$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
                WHERE audiencia_consumo='vec_dietas_rutas_v1.acceso.v1')
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
                   WHERE convert_from(capacidad_canonica,'UTF8')::jsonb ->> 'audiencia_consumo'
                         = 'vec_dietas_rutas_v1.acceso.v1')
       OR EXISTS (SELECT 1 FROM pg_auth_members
                   WHERE roleid='vec_dietas_rutas_v1_ejecutor'::regrole
                      OR member='vec_dietas_rutas_v1_ejecutor'::regrole)
       OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_rutas_v1_ejecutor'
                       AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
                       AND NOT rolcreatedb AND NOT rolcreaterole
                       AND NOT rolreplication AND NOT rolbypassrls) THEN
        RAISE EXCEPTION 'AD3-39: conservar gobierno, historia o rol provisionado' USING ERRCODE='55000';
    END IF;
END $sin_historia$;
DO $acl_exacta$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    propietario oid := 'vec_autorizacion_atestada_v3_propietario'::regrole;
    ejecutor oid := 'vec_dietas_rutas_v1_ejecutor'::regrole;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=propietario
                    AND p.prosecdef AND p.provolatile='v' AND p.pronargdefaults=0
                    AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
       OR NOT coalesce((SELECT count(*)=2 AND count(DISTINCT a.grantee)=2
                        AND bool_and(a.grantee IN (propietario,ejecutor) AND a.grantor=propietario
                            AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
                          FROM pg_proc p CROSS JOIN LATERAL aclexplode(
                              coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f),false)
       OR EXISTS (SELECT 1 FROM pg_auth_members WHERE member=ejecutor)
       OR has_schema_privilege(ejecutor,'vec_autorizacion_atestada_v3','CREATE')
       OR NOT has_schema_privilege(ejecutor,'vec_autorizacion_atestada_v3','USAGE')
       OR EXISTS (SELECT 1 FROM pg_proc p
                   WHERE p.pronamespace='vec_autorizacion_atestada_v3'::regnamespace
                     AND p.oid<>f AND has_function_privilege(ejecutor,p.oid,'EXECUTE'))
       OR EXISTS (SELECT 1 FROM pg_class t
                   WHERE t.relnamespace='vec_autorizacion_atestada_v3'::regnamespace
                     AND t.relkind IN ('r','p','v','m','f')
                     AND (has_table_privilege(ejecutor,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
                          OR has_any_column_privilege(ejecutor,t.oid,'SELECT,INSERT,UPDATE,REFERENCES'))) THEN
        RAISE EXCEPTION 'AD3-39: ACL incompatible' USING ERRCODE='55000';
    END IF;
END $acl_exacta$;
-- Sólo cambia tres fragmentos nominales; la revalidación viva, HMAC/COSE,
-- gobierno, consumo y auditoría permanecen en el núcleo existente.
DO $nucleo$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    definicion text; esperada text; metadata jsonb; dependencias jsonb; fragmento text;
    inicio text := $inicio$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'$inicio$;
    inicio_nuevo text := $inicio_nuevo$       OR NOT (
           (p_perfil_mutacion IS DISTINCT FROM 'acceso_rutas_dietas' AND (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'$inicio_nuevo$;
    cierre text := $cierre$       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo VEC-AD-3 rechazado';$cierre$;
    cierre_nuevo text := $cierre_nuevo$           ))
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'acceso_rutas_dietas'
               AND EXISTS (
                   SELECT 1 FROM pg_roles r WHERE r.rolname=session_user
                     AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb
                     AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls
               )
               AND EXISTS (
                   SELECT 1 FROM pg_roles r WHERE r.rolname='vec_dietas_rutas_v1_ejecutor'
                     AND NOT r.rolcanlogin AND NOT r.rolinherit AND NOT r.rolsuper
                     AND NOT r.rolcreatedb AND NOT r.rolcreaterole
                     AND NOT r.rolreplication AND NOT r.rolbypassrls
               )
               AND EXISTS (
                   SELECT 1 FROM pg_auth_members m
                    WHERE m.member=session_user::regrole
                      AND m.roleid='vec_dietas_rutas_v1_ejecutor'::regrole
                      AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
               )
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (
                   SELECT 1 FROM pg_auth_members m
                    WHERE m.member='vec_dietas_rutas_v1_ejecutor'::regrole
               )
           )
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo VEC-AD-3 rechazado';$cierre_nuevo$;
    marca text := $marca$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$marca$;
    perfil text := $perfil$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'acceso_rutas_dietas'
               AND c ->> 'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas_rutas_v1.acceso.v1'
               AND c ->> 'operacion' IS NOT DISTINCT FROM d ->> 'accion'
               AND d ->> 'modulo_id' IS NOT DISTINCT FROM 'dietas'
               AND d ->> 'finalidad' IS NOT DISTINCT FROM 'consultar_itinerario_dietas'
               AND (
                   (d ->> 'accion' IS NOT DISTINCT FROM 'dietas.ruta.catalogo.consultar'
                    AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'catalogo_rutas_dietas')
                   OR
                   (d ->> 'accion' IS NOT DISTINCT FROM 'dietas.ruta.calculo.solicitar'
                    AND d ->> 'tipo_recurso' IS NOT DISTINCT FROM 'calculo_rutas_dietas')
               )
               AND d #>> '{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
               AND d -> 'campos_permitidos' IS NOT DISTINCT FROM '[]'::jsonb
               AND d -> 'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
           )
$perfil$;
BEGIN
    SELECT pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc'
      INTO STRICT definicion,metadata FROM pg_proc p
     WHERE p.oid=f AND p.proowner=current_user::regrole
       AND p.prosecdef AND p.provolatile='v'
       AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'];
    IF strpos(definicion,'subsanacion_reparos_ct')=0
       OR strpos(definicion,'vec_autorizacion.revalidar_decision_contexto_actor_v3_viva')=0
       OR strpos(definicion,'borrador_dietas')<>0
       OR strpos(definicion,'cronos_marcaje_propio')<>0
        THEN
        RAISE EXCEPTION 'AD3-39: núcleo incompatible o migraciones posteriores' USING ERRCODE='55000';
    END IF;
    FOREACH fragmento IN ARRAY ARRAY[inicio_nuevo,cierre_nuevo,perfil||marca] LOOP
        IF length(definicion)-length(replace(definicion,fragmento,''))<>length(fragmento) THEN
            RAISE EXCEPTION 'AD3-39: fragmento de núcleo divergente' USING ERRCODE='55000';
        END IF;
    END LOOP;
    SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
                             d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      INTO dependencias FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
    esperada := replace(replace(replace(definicion,inicio_nuevo,inicio),cierre_nuevo,cierre),perfil||marca,marca);
    EXECUTE esperada;
    IF (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM metadata
       OR pg_get_functiondef(f) IS DISTINCT FROM esperada
       OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,
                                    d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f)
          IS DISTINCT FROM dependencias THEN
        RAISE EXCEPTION 'AD3-39: alteración de metadata/ACL/dependencias' USING ERRCODE='55000';
    END IF;
END $nucleo$;
DO $audiencia$
DECLARE
    definicion text; esperada text; audiencias text[];
    base31 text[] := ARRAY[
        'vec_contratacion_temporal.confirmar_alta_atestada.v1',
        'vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1',
        'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1',
        'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1',
        'vec_personal.alta_ejercicio.v1',
        'vec_contratacion_temporal.incorporacion_ejercicio.v2',
        'vec_personal.lectura_incorporacion.v1',
        'vec_personal.lectura_incorporacion.v2',
        'vec_contratacion_temporal.anotacion_administrativa.v1',
        'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1'];
    corporativa31 text[] := ARRAY[
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
        'vec_contratacion_temporal.cierre_administrativo_sin_cese.v1'];
BEGIN
    base31 := base31 || ARRAY['vec_contratacion_temporal.despacho_correo_llamamiento.v1','vec_contratacion_temporal.resultado_correo_llamamiento.v1'];
    corporativa31 := corporativa31 || ARRAY['vec_contratacion_temporal.despacho_correo_llamamiento.v1','vec_contratacion_temporal.resultado_correo_llamamiento.v1'];
    base31 := array_append(base31,'vec_dietas_rutas_v1.acceso.v1');
    corporativa31 := array_append(corporativa31,'vec_dietas_rutas_v1.acceso.v1');
    SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT definicion FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check'
       AND c.contype='c' AND c.convalidated AND c.conkey=ARRAY[8]::smallint[];
    IF definicion='CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(base31) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN
        audiencias := base31;
    ELSIF definicion='CHECK (audiencia_consumo = ANY (ARRAY['||
        array_to_string(ARRAY(SELECT quote_literal(a)||'::text'
            FROM unnest(corporativa31) WITH ORDINALITY u(a,n) ORDER BY n),', ')||']))' THEN
        audiencias := corporativa31;
    ELSE
        RAISE EXCEPTION 'AD3-39: CHECK de audiencias no corresponde a AD3-39'
            USING ERRCODE='55000';
    END IF;
    audiencias := audiencias[1:cardinality(audiencias)-1];
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
        RAISE EXCEPTION 'AD3-39: postimagen de audiencias divergente'
            USING ERRCODE='55000';
    END IF;
END $audiencia$;

DROP FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) RESTRICT;
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_dietas_rutas_v1_ejecutor;
RESET ROLE;
DROP ROLE vec_dietas_rutas_v1_ejecutor;
COMMIT;
