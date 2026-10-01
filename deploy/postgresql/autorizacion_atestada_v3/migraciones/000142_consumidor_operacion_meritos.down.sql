\set ON_ERROR_STOP on
-- AD3-142 DOWN solo en ensayo sin historia y después de retirar Méritos000001.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000142',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $pre$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname='vec_meritos' AND p.proname='operar_hecho_v1')
 OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
      WHERE n.nspname='vec_meritos' AND c.relname IN ('hecho_identidad','hecho_version','operacion','auditoria_operacion','outbox','acceso_operacion'))
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
      WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion' IN ('meritos.hecho.declarar','meritos.hecho.rectificar','meritos.hecho.rechazar'))
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
      WHERE audiencia_consumo IN ('vec_meritos.hecho.declarar.v1','vec_meritos.hecho.rectificar.v1','vec_meritos.hecho.rechazar.v1'))
 THEN RAISE EXCEPTION 'AD3-142 DOWN: consumidor o historia existentes' USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Postimagen AD141 comprobada en PostgreSQL18.4 sobre baseline actual.
 esperada_def_sha256 text:=$esperada_def_sha256$202b1580f00e1618e0fb311dcf0911992e56d9eb5f17bc642d58c73768f360f1$esperada_def_sha256$;
 esperada_fuente_sha256 text:=$esperada_fuente_sha256$4a98b94be7e198a6c1e364949e60f35f39b2e5bf931bc361b1adb52a12a94905$esperada_fuente_sha256$;
 marca text:=$marca$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$marca$;
 excl text:=$excl$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$excl$;
 excl_nuevo text:=$excl_nuevo$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_interno'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_externo'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_rechazar'
$excl_nuevo$;
 runtime text:=$runtime$       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_version_convocatoria_bolsa'
$runtime$;
 runtime_nuevo text:=$runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_rechazar')
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb AND NOT r.rolbypassrls)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_meritos_ejecutor'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_meritos_interno'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=2
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m
                   WHERE m.member IN ('vec_meritos_ejecutor'::regrole,'vec_meritos_interno'::regrole,'vec_meritos_externo'::regrole))
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_version_convocatoria_bolsa'
$runtime_nuevo$;
 extension text:=$extension$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_hecho_propio_interno'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.declarar'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.declarar.v1'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'declaracion_hecho_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rectificar'
       AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.rectificar.v1'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'rectificacion_hecho_propio'))
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND pg_has_role(session_user,'vec_meritos_interno','MEMBER')
 AND NOT pg_has_role(session_user,'vec_meritos_externo','MEMBER')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'meritos_hecho_rechazar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rechazar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_meritos.hecho.rechazar.v1'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'revision_hecho_merito'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND pg_has_role(session_user,'vec_meritos_interno','MEMBER')
 AND NOT pg_has_role(session_user,'vec_meritos_externo','MEMBER')
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["declarante_ref","hecho","recibo","version"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb)
$extension$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-142: núcleo ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'AD3-142: metadatos de núcleo ausentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 -- Perfil nuevo en las dos listas del núcleo: exclusión del bloque general y
 -- selección de la guarda de sesión miembro del ejecutor Bolsa.
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
         AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_database db
         CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
         WHERE db.datname=current_database() AND a.grantee=0 AND a.privilege_type='TEMPORARY')
    OR EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)='vec_' AND r.rolcanlogin
         AND has_database_privilege(r.oid,current_database(),'TEMPORARY'))
    OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee=propietario AND a.grantor=propietario
           AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
         WHERE a.grantee<>propietario OR a.grantor<>propietario
            OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
    OR deps IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_language'::regclass::oid,
           'refobjid',(SELECT oid FROM pg_language WHERE lanname='plpgsql'),
           'refobjsubid',0,'deptype','n'),
         jsonb_build_object('classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_namespace'::regclass::oid,
           'refobjid','vec_autorizacion_atestada_v3'::regnamespace::oid,
           'refobjsubid',0,'deptype','n'))
    OR deps_compartidas IS DISTINCT FROM jsonb_build_array(
         jsonb_build_object('dbid',(SELECT oid FROM pg_database WHERE datname=current_database()),
           'classid','pg_proc'::regclass::oid,'objid',f::oid,'objsubid',0,
           'refclassid','pg_authid'::regclass::oid,'refobjid',propietario,'deptype','o'))
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl_nuevo,''))<>length(excl_nuevo)
    OR length(original)-length(replace(original,runtime_nuevo,''))<>length(runtime_nuevo)
    OR length(original)-length(replace(original,extension,''))<>length(extension)
 THEN RAISE EXCEPTION 'AD3-142: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(replace(original,extension,''),excl_nuevo,excl),runtime_nuevo,runtime);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM 'd0a2a942a0f6df7f0e975f922073ba5c64652709902f4a1602a56f79da85ccee'
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'AD3-142: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

DO $audiencias$
DECLARE d text; nuevo text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM 'd5c8048786b283485016af29fba41ff68b93076ba4f37f2badfa6bb7d5532fd9'
 THEN RAISE EXCEPTION 'AD3-142 DOWN: postimagen de audiencias divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(d,', ''vec_meritos.hecho.declarar.v1''::text, ''vec_meritos.hecho.rectificar.v1''::text, ''vec_meritos.hecho.rechazar.v1''::text','');
 IF encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM '8f6670d43dc687a7fc51db62d90ea4f48c2ff9b61a52e38c768a0426e4198fcf'
 THEN RAISE EXCEPTION 'AD3-142 DOWN: audiencias no localizables' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencias$;
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_meritos_propietario;
COMMIT;
