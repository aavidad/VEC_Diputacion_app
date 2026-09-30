\set ON_ERROR_STOP on
-- Solo clon desechable sin decisiones ni datos de gobierno. Nunca sobre historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000134',0));
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.consumo_decision_v3 IN ACCESS EXCLUSIVE MODE;
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR pg_catalog.to_regrole('vec_catalogos_configurables_gobierno_ejecutor') IS NULL
 OR EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members WHERE member='vec_catalogos_configurables_gobierno_ejecutor'::regrole OR roleid='vec_catalogos_configurables_gobierno_ejecutor'::regrole)
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version WHERE audiencia_consumo='vec_catalogos_configurables.gobierno_categorias.v1')
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
   WHERE pg_catalog.convert_from(capacidad_canonica,'UTF8')::jsonb->>'audiencia_consumo'='vec_catalogos_configurables.gobierno_categorias.v1')
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD3-134: DOWN con historia o preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
DROP FUNCTION vec_autorizacion_atestada_v3.proponer_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion_atestada_v3.aprobar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),
 vec_autorizacion_atestada_v3.confirmar_gobierno_categoria_rpt_v3_atestada(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.ejecutar_gobierno_categoria_rpt_v3_interna(text,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_autorizacion_atestada_v3.acreditar_aprobacion_historica_gobierno_categoria_rpt_v3_interna(jsonb,text,text,text,text);
DROP FUNCTION vec_autorizacion_atestada_v3.autorizar_gobierno_categoria_rpt_v3_interna(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_catalogos_configurables_gobierno_ejecutor;

DO $nucleo$
DECLARE
    f oid := 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
    original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
    propietario oid; config text[]; definidora boolean;
    exclusor text := E'               AND p_perfil_mutacion IS DISTINCT FROM ''alta_personal_ejercicio'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_registro_personal_incorporacion_v2'' AND p_perfil_mutacion IS DISTINCT FROM ''lectura_categorias'' AND p_perfil_mutacion IS DISTINCT FROM ''usos_categorias''';
    cierre_roles text := E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',\n            MESSAGE = ''consumo VEC-AD-3 rechazado'';';
    rol_nuevo text := $x$           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_categorias'
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole
                    AND m.roleid='vec_catalogos_configurables_gobierno_ejecutor'::pg_catalog.regrole
                    AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option
                    AND pg_catalog.pg_has_role(session_user,m.roleid,'MEMBER'))=1
               AND (SELECT count(*) FROM pg_catalog.pg_auth_members m WHERE m.member=session_user::pg_catalog.regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_auth_members m WHERE m.member='vec_catalogos_configurables_gobierno_ejecutor'::pg_catalog.regrole)
               AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_roles r WHERE pg_catalog.left(r.rolname,4)='vec_'
                    AND r.rolname<>session_user AND r.rolname<>'vec_catalogos_configurables_gobierno_ejecutor'
                    AND pg_catalog.pg_has_role(session_user,r.oid,'MEMBER'))
           )
$x$;
    marca_capacidad text := E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
    capacidad_nueva text := $x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_categorias'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_catalogos_configurables.gobierno_categorias.v1'
 AND c->>'operacion' = ANY (ARRAY['vec.catalogos.categorias.gobierno.proponer',
                                   'vec.catalogos.categorias.gobierno.aprobar',
                                   'vec.catalogos.categorias.gobierno.confirmar'])
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'propuesta_categoria'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gobernar_categorias_rpt'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["gobierno","recibo"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
    SELECT pg_catalog.pg_get_functiondef(f),pg_catalog.to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
      INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_catalog.pg_proc AS p WHERE p.oid=f;
    SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
      INTO deps FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f;
    IF propietario <> 'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
       OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
       OR pg_catalog.strpos(original,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''gobierno_categorias''')=0
       OR pg_catalog.strpos(original,rol_nuevo||cierre_roles)=0
       OR pg_catalog.strpos(original,capacidad_nueva||marca_capacidad)=0 THEN
        RAISE EXCEPTION 'AD3-134: nucleo cambiado, DOWN denegado' USING ERRCODE='55000';
    END IF;
    nuevo := pg_catalog.replace(original,capacidad_nueva||marca_capacidad,marca_capacidad);
    nuevo := pg_catalog.replace(nuevo,rol_nuevo||cierre_roles,cierre_roles);
    nuevo := pg_catalog.replace(nuevo,exclusor||E' AND p_perfil_mutacion IS DISTINCT FROM ''gobierno_categorias''',exclusor);
    EXECUTE nuevo;
    SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT actual;
    IF actual IS DISTINCT FROM nuevo
       OR (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc AS p WHERE p.oid=f) IS DISTINCT FROM meta
       OR (SELECT proacl FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM acl
       OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM propietario
       OR (SELECT proconfig FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM config
       OR (SELECT prosecdef FROM pg_catalog.pg_proc WHERE oid=f) IS DISTINCT FROM definidora
       OR (SELECT coalesce(pg_catalog.jsonb_agg(pg_catalog.to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
             FROM pg_catalog.pg_depend AS d WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN
        RAISE EXCEPTION 'AD3-134: nucleo alterado en DOWN' USING ERRCODE='55000';
    END IF;
END $nucleo$;

DO $audiencia$
DECLARE definicion text; vieja text; nueva text := 'vec_catalogos_configurables.gobierno_categorias.v1';
BEGIN
    SELECT pg_catalog.regexp_replace(pg_catalog.pg_get_constraintdef(c.oid,true),'\s+',' ','g')
      INTO STRICT definicion FROM pg_catalog.pg_constraint AS c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
    IF pg_catalog.right(definicion,3)<>']))'
       OR pg_catalog.strpos(definicion,', '||pg_catalog.quote_literal(nueva)||'::text]))')=0 THEN
        RAISE EXCEPTION 'AD3-134: audiencia cambiada, DOWN denegado' USING ERRCODE='55000';
    END IF;
    vieja := pg_catalog.replace(definicion,', '||pg_catalog.quote_literal(nueva)||'::text]))', ']))');
    ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
        DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
    EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||vieja;
END $audiencia$;
SET LOCAL ROLE NONE;
DROP ROLE vec_catalogos_configurables_gobierno_ejecutor;
COMMIT;
