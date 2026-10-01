\set ON_ERROR_STOP on
-- AD3-141 DOWN solo en ensayo sin historia y después de retirar Bolsa000007.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC'; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000141',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
LOCK TABLE vec_autorizacion_atestada_v3.atestacion_decision_v3 IN SHARE MODE;
DO $pre$
BEGIN
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE n.nspname='vec_bolsa_convocatorias' AND p.proname='obtener_version_exacta_v3')
 OR EXISTS (SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
     WHERE n.nspname='vec_bolsa_convocatorias' AND c.relname='lectura_version_v3')
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.atestacion_decision_v3
     WHERE convert_from(capacidad_canonica,'UTF8')::jsonb->>'operacion'='bolsa.convocatoria.version.consultar')
 OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
     WHERE audiencia_consumo='vec_bolsa_convocatorias.version.consultar.v1')
 THEN RAISE EXCEPTION 'AD3-141 DOWN: consumidor o historia existentes' USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Preimagen real PG18.4 postbaseline main4617 + dependencias Convocatorias.
 esperada_def_sha256 text:=$esperada_def_sha256$d0a2a942a0f6df7f0e975f922073ba5c64652709902f4a1602a56f79da85ccee$esperada_def_sha256$;
 esperada_fuente_sha256 text:=$esperada_fuente_sha256$17a411ce89674b760d01ebc89e3576341607a7759b18ad5034331836d405ec83$esperada_fuente_sha256$;
 marca text:=$marca$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$marca$;
 excl text:=$excl$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$excl$;
 excl_nuevo text:=$excl_nuevo$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_version_convocatoria_bolsa'
$excl_nuevo$;
 runtime text:=$runtime$       OR NOT (
           (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$runtime$;
 runtime_nuevo text:=$runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_version_convocatoria_bolsa'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb AND NOT r.rolbypassrls)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_bolsa_convocatorias_ejecutor_consulta'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_bolsa_convocatorias_ejecutor_consulta'::regrole)
           )
           OR (
               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$runtime_nuevo$;
 extension text:=$extension$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_version_convocatoria_bolsa'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_convocatorias.version.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'bolsa.convocatoria.version.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'version_convocatoria_gobernada'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_interna_convocatorias'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["version_convocatoria"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-141: núcleo ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'AD3-141: metadatos de núcleo ausentes' USING ERRCODE='55000'; END IF;
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
    OR length(original)-length(replace(original,replace(runtime_nuevo,excl,excl_nuevo),''))<>length(replace(runtime_nuevo,excl,excl_nuevo))
    OR length(original)-length(replace(original,extension,''))<>length(extension)
 THEN RAISE EXCEPTION 'AD3-141: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(replace(original,extension,''),excl_nuevo,excl),runtime_nuevo,runtime);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR encode(sha256(convert_to(actual,'UTF8')),'hex') IS DISTINCT FROM 'dd5a4a795f5b03ebaf4a0f437d4416e20cffae5450dae04fc8dad721c67a1db4'
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
 THEN RAISE EXCEPTION 'AD3-141: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

DO $audiencias$
DECLARE d text; nuevo text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM '8f6670d43dc687a7fc51db62d90ea4f48c2ff9b61a52e38c768a0426e4198fcf'
 THEN RAISE EXCEPTION 'AD3-141 DOWN: postimagen de audiencias divergente' USING ERRCODE='55000'; END IF;
 nuevo:=replace(d,', ''vec_bolsa_convocatorias.version.consultar.v1''::text','');
 IF encode(sha256(convert_to(nuevo,'UTF8')),'hex') IS DISTINCT FROM '5ad406c6e428cdc1eb827fa7f5dd86d7406add833cad3deffb25b082789dec0f'
 THEN RAISE EXCEPTION 'AD3-141 DOWN: audiencia no localizable' USING ERRCODE='55000'; END IF;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nuevo;
END $audiencias$;
DROP FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
REVOKE USAGE ON SCHEMA vec_autorizacion_atestada_v3 FROM vec_bolsa_convocatorias_propietario;
COMMIT;
