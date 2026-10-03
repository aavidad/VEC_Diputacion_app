\set ON_ERROR_STOP on
-- AD149. Consumo nominal del vínculo propio histórico CRN11.
-- Reanclaje 03/10/2026 a copia fría principal + H7/H8 + AD155/B77/Convoca5.
-- Dependencias funcionales: Personal16, CA7 y núcleo V3 con preimagen exacta.
-- CRN11 no usa AD141/142/144 ni Baremo; conserva todos los consumidores previos.
-- Añade un perfil, su selección runtime y una sola audiencia. Nunca presta
-- AD54/74, publica permisos por petición ni amplía el proceso exterior.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000149',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE rol text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_personal' AND p.proname='resolver_empleado_canonico_persona_v1'
         AND p.pronargs=2 AND p.proargtypes[0]='text'::regtype AND p.proargtypes[1]='timestamptz'::regtype
         AND p.proowner='vec_personal_propietario'::regrole)
    OR NOT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_personal' AND p.proname='bloquear_generacion_proyeccion_empleado_persona_v1'
         AND p.pronargs=1 AND p.proargtypes[0]='text'::regtype AND p.proowner='vec_personal_propietario'::regrole)
    OR NOT EXISTS(SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
       WHERE n.nspname='vec_contexto_actor_v1' AND p.proname='proyeccion_empleado_personal_v2'
         AND p.pronargs=2 AND p.proargtypes[0]='text'::regtype AND p.proargtypes[1]='timestamptz'::regtype
         AND p.proowner='vec_contexto_actor_v1_propietario'::regrole) THEN
  RAISE EXCEPTION 'AD149: dependencias o preimagen incompatibles' USING ERRCODE='55000';
 END IF;
 FOREACH rol IN ARRAY ARRAY['vec_personal_propietario','vec_personal_migrador','vec_personal_ejecutor'] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=rol AND NOT rolcanlogin
     AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls) THEN
   RAISE EXCEPTION 'AD149: rol incompatible' USING ERRCODE='55000';
  END IF;
 END LOOP;
 IF EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_personal_ejecutor'::regrole) THEN
  RAISE EXCEPTION 'AD149: herencia del ejecutor incompatible' USING ERRCODE='55000';
 END IF;
END $pre$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Preimagen completa capturada en PG18.4 con search_path=pg_catalog,pg_temp.
 esperada_def_sha256 text:=$esperada_def_sha256$5dbdac03a2a4e52ca4cb45c57b3bf18e621091da9e818091e30a3f90e68e7330$esperada_def_sha256$;
 esperada_fuente_sha256 text:=$esperada_fuente_sha256$cdc8cb87f27360741a2d0d52e8b9d58d0a1423be8abd8ff9063f389b2c8ea75e$esperada_fuente_sha256$;
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
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,runtime,''))<>length(runtime)
    OR strpos(original,'vinculo_propio_historico_crn11')<>0
    OR strpos(original,'personal.vinculo_propio.crn11.consultar')<>0
 THEN
  RAISE EXCEPTION 'PARO clave=AD149.nucleo_preimagen, actual=def:%/src:%/runtime:%/config:%, esperado=def:%/src:%/runtime:1/config:search_path=pg_catalog, pg_temp;lock_timeout=2s',
   encode(sha256(convert_to(original,'UTF8')),'hex'),
   encode(sha256(convert_to(fuente,'UTF8')),'hex'),
   (length(original)-length(replace(original,runtime,'')))/length(runtime),
   config,esperada_def_sha256,esperada_fuente_sha256 USING ERRCODE='55000';
 END IF;
 nuevo:=replace(original,runtime,runtime_nuevo);
 nuevo:=replace(nuevo,excl,excl_nuevo);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),excl_nuevo,excl),runtime_nuevo,runtime) IS DISTINCT FROM original
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

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM 'c220a791d3bf62f5a87ca192373900178c7c3344f10384b1080cfef9c2f81626'
    OR strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_personal.vinculo_propio.crn11.v1')<>0 THEN
  RAISE EXCEPTION 'PARO clave=AD149.audiencias_preimagen, actual=sha256:%/forma:%/crn11_ausente:%, esperado=sha256:c220a791d3bf62f5a87ca192373900178c7c3344f10384b1080cfef9c2f81626/forma:true/crn11_ausente:true',
   encode(sha256(convert_to(d,'UTF8')),'hex'),
   strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')=1 AND right(d,3)=']))',
   strpos(d,'vec_personal.vinculo_propio.crn11.v1')=0 USING ERRCODE='55000';
 END IF;
 nueva:=left(d,length(d)-3)||', ''vec_personal.vinculo_propio.crn11.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF (SELECT pg_get_constraintdef(c.oid,true) FROM pg_constraint c
     WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
       AND c.conname='clave_capacidad_version_audiencia_consumo_check') IS DISTINCT FROM nueva THEN
  RAISE EXCEPTION 'AD149: audiencia cambió fuera del contrato' USING ERRCODE='55000';
 END IF;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off' THEN
  RAISE EXCEPTION 'consumo CRN11 denegado' USING ERRCODE='42501';
 END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'material CRN11 inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
    OR d->>'concedida' IS DISTINCT FROM 'true'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM 'personal.vinculo_propio.crn11.consultar'
    OR c->>'operacion' IS DISTINCT FROM d->>'accion'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.vinculo_propio.crn11.v1'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'vinculo_historico_propio_crn11'
    OR d->>'finalidad' IS DISTINCT FROM 'acreditar_vinculo_historico_propio_crn11'
    OR d->'campos_permitidos' IS DISTINCT FROM '["empleado_ref","fuente_ref","persona_ref","version","vinculo_ref"]'::jsonb
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR d->>'recurso_ref' IS NULL OR d->>'recurso_ref' !~ '^emp_[A-Za-z0-9_-]{22,128}$'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS NULL
    OR d->>'contexto_recurso_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
  RAISE EXCEPTION 'consumo CRN11 denegado' USING ERRCODE='42501';
 END IF;
 SELECT * INTO STRICT x
 FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'vinculo_propio_historico_crn11',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256' THEN
  RAISE EXCEPTION 'consumo CRN11 divergente' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(
 bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;

GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_personal_propietario'::regrole::oid; x record;
BEGIN
 -- También las ACL por defecto: ningún rol conserva acceso por haber sido
 -- destinatario predeterminado del propietario.
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_personal_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_personal_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD149: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD149: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
