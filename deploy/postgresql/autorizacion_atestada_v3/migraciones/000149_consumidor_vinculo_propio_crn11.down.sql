\set ON_ERROR_STOP on
-- DOWN AD149 preparado, nunca ejecutado en este ensayo.
-- Personal26 debe retirarse primero y sólo sin recibos; cualquier clave propia
-- CRN11 registrada impide retirar la frontera, aun revocada. No borrar historia.
-- Requiere postimagen propia exacta y restaura la preimagen fría y sus ACL.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000149',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $historia$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL
    OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
       WHERE audiencia_consumo='vec_personal.vinculo_propio.crn11.v1')
    OR EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
       WHERE n.nspname='vec_personal' AND c.relname='recibo_vinculo_propio_crn11')
    OR EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_personal' AND p.proname='consultar_vinculo_propio_historico_crn11_v1') THEN
  RAISE EXCEPTION 'AD149: historia o dependencia impide DOWN' USING ERRCODE='55000';
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
     AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
       WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_personal_propietario'::regrole)
         OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
  RAISE EXCEPTION 'AD149: fachada propia incompatible' USING ERRCODE='55000';
 END IF;
END $historia$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Postimagen149 derivada de la preimagen fría capturada y extensión propia.
 -- Esta fuente sólo se revisa; no ejecutar DOWN sobre la base conservada.
 esperada_def_sha256 text:=$esperada_def_sha256$d912064d905e4349ffb2e1e8f1aab1aebef71e1fd842d5603373c4e6236fcd15$esperada_def_sha256$;
 esperada_fuente_sha256 text:=$esperada_fuente_sha256$1bb33d97bf8bbaa7f97dec4e1af1aa41577be38cea17f4b18375346cf7d62dcd$esperada_fuente_sha256$;
 marca text:=$marca$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$marca$;
 excl text:=$excl$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$excl$;
 excl_nuevo text:=$excl_nuevo$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'vinculo_propio_historico_crn11'
$excl_nuevo$;
 runtime text:=$runtime$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$runtime$;
 runtime_nuevo text:=$runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_propio_historico_crn11'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
                  AND NOT r.rolreplication AND NOT r.rolbypassrls)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_personal_ejecutor'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_personal_ejecutor'::regrole)
           )
           OR (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$runtime_nuevo$;
 extension text:=$extension$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_propio_historico_crn11'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.vinculo_propio.crn11.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vinculo_historico_propio_crn11'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'acreditar_vinculo_historico_propio_crn11'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'recurso_ref' ~ '^emp_[A-Za-z0-9_-]{22,128}$'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-149: núcleo ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'AD3-149: metadatos de núcleo ausentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 -- Perfil nuevo en las dos listas del núcleo: exclusión del bloque general y
 -- selección de LOGIN con único grupo técnico Personal, sin SET/ADMIN.
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
 THEN
  RAISE EXCEPTION 'PARO clave=AD149.nucleo_preimagen, actual=def:%/src:%/runtime:%/config:%, esperado=def:%/src:%/runtime:1/config:search_path=pg_catalog, pg_temp;lock_timeout=2s',
   encode(sha256(convert_to(original,'UTF8')),'hex'),
   encode(sha256(convert_to(fuente,'UTF8')),'hex'),
   (length(original)-length(replace(original,runtime_nuevo,'')))/length(runtime_nuevo),
   config,esperada_def_sha256,esperada_fuente_sha256 USING ERRCODE='55000';
 END IF;
 nuevo:=replace(original,extension||marca,marca);
 nuevo:=replace(nuevo,excl_nuevo,excl);
 nuevo:=replace(nuevo,runtime_nuevo,runtime);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,runtime,runtime_nuevo),excl,excl_nuevo),marca,extension||marca) IS DISTINCT FROM original
    OR encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM '5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330'
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
 THEN RAISE EXCEPTION 'AD3-149: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
DO $audiencias$
DECLARE d text; retirar constant text:=', ''vec_personal.vinculo_propio.crn11.v1''::text'; nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM '0ba3eabde2f45d27afd278cfc008ddc0c3a24cc6d0e5de790b5c6d65dec906a6'
    OR length(d)-length(replace(d,retirar,''))<>length(retirar) THEN
  RAISE EXCEPTION 'PARO clave=AD149.audiencias_postimagen, actual=sha256:%/extension:%, esperado=sha256:0ba3eabde2f45d27afd278cfc008ddc0c3a24cc6d0e5de790b5c6d65dec906a6/extension:1',
   encode(sha256(convert_to(d,'UTF8')),'hex'),
   (length(d)-length(replace(d,retirar,'')))/length(retirar) USING ERRCODE='55000';
 END IF;
 nueva:=replace(d,retirar,'');
 IF encode(sha256(convert_to(nueva,'UTF8')),'hex') IS DISTINCT FROM 'c220a791d3bf62f5a87ca192373900178c7c3344f10384b1080cfef9c2f81626' THEN
  RAISE EXCEPTION 'AD149: reversión de audiencias divergente' USING ERRCODE='55000';
 END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC,vec_personal_propietario;
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
-- USAGE previo del esquema no se revoca: pertenece a otros contratos existentes.
COMMIT;
