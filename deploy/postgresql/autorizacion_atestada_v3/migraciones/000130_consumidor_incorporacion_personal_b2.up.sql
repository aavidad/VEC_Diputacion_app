\set ON_ERROR_STOP on
-- AD3-130: plan prospectivo y origen de incorporación Personal, paso RRHH de incorporación.
-- Tres operaciones nominales; consume V3 actual también en lectura y replay.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='2min';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000130',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_contratacion_temporal_propietario'
                   AND NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-130: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Preimagen post-AD129 del clon causal B -> AD136 -> AD127 -> AD128 -> AD131 -> AD129.
 esperada_def_sha256 text:='70e26e0019bbb135f846b4a26e35850009cda3a8a597e63d35e8eefc5f1bca34';
 esperada_fuente_sha256 text:='43d8c235a4c8f21810b0fd1f0d2773bc0f3c283a71950bd50f3e0056e73aaed4';
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'incorporacion_personal_ct'
 AND c->>'operacion' = ANY (ARRAY['contratacion_temporal.incorporacion_personal.plan.consultar','contratacion_temporal.incorporacion_personal.plan.registrar','contratacion_temporal.incorporacion_personal.origen.confirmar'])
 AND ((c->>'operacion'='contratacion_temporal.incorporacion_personal.plan.consultar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1'
       AND d->'campos_permitidos'='["plan"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.incorporacion_personal.plan.registrar'
       AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1'
       AND d->'campos_permitidos'='["recibo"]'::jsonb)
   OR (c->>'operacion'='contratacion_temporal.incorporacion_personal.origen.confirmar' AND c->>'audiencia_consumo'='vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1' AND d->'campos_permitidos'='["recibo"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'contratacion_temporal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'incorporacion_personal_ct'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'incorporar_personal_desde_ct'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' LIKE 'expediente:%'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$x$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'AD3-130: núcleo ausente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'AD3-130: metadatos de núcleo ausentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
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
    OR strpos(original,'vec_contratacion_temporal_ejecutor')=0
    OR strpos(original,'incorporacion_personal_ct')<>0
 THEN RAISE EXCEPTION 'AD3-130: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,extension||marca,marca) IS DISTINCT FROM original
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
 THEN RAISE EXCEPTION 'AD3-130: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; esperada_audiencia_sha256 text:='4e22728340ea408131117cd753e03ad6450325dbde674e1d5e80c73f0b0e895e'; a text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF NOT FOUND OR d IS NULL
    OR encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM esperada_audiencia_sha256
 THEN RAISE EXCEPTION 'AD3-130: CHECK de audiencias incompatible' USING ERRCODE='55000'; END IF;
 d:=regexp_replace(d,'\s+',' ','g');
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_catalogos_configurables.lectura_categorias.v1')=0
 THEN RAISE EXCEPTION 'AD3-130: audiencia previa incompatible' USING ERRCODE='55000'; END IF;
 FOREACH a IN ARRAY ARRAY['vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1','vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1','vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1'] LOOP
  IF strpos(d,quote_literal(a))<>0 THEN RAISE EXCEPTION 'AD3-130: audiencia repetida' USING ERRCODE='55000'; END IF;
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '
  ||left(d,length(d)-3)||', ''vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1''::text, ''vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1''::text, ''vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1''::text]))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE s jsonb; c jsonb; d jsonb; x record; material_h text; contexto_h text; accion text;unidad text;material jsonb;
BEGIN
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 2 AND 65536
 THEN RAISE EXCEPTION 'AD3-130: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb; c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-130: material inválido' USING ERRCODE='22023'; END;
 material:=s;accion:=c->>'operacion';
 unidad:=CASE accion WHEN 'contratacion_temporal.incorporacion_personal.plan.registrar' THEN material#>>'{material,unidad_ct_ref}' WHEN 'contratacion_temporal.incorporacion_personal.origen.confirmar' THEN s->>'unidad_ct_ref' ELSE s->>'unidad_ref' END;
 IF accion='contratacion_temporal.incorporacion_personal.plan.registrar' THEN s:=s->'solicitud'; END IF;
 material_h:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 contexto_h:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'organizacion_ref')||'","unidad_ref":"'||unidad||'"},"atributos":{"material_sha256":"'||material_h||'"}}','UTF8')),'hex');
 IF unidad IS NULL OR unidad!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' OR jsonb_typeof(s)<>'object'
    OR s->>'organizacion_ref' IS NULL OR s->>'organizacion_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR s->>'expediente_ref' IS NULL OR s->>'expediente_ref' !~ '^expediente:[A-Za-z0-9._:/#-]{2,149}$'
    OR accion NOT IN ('contratacion_temporal.incorporacion_personal.plan.consultar','contratacion_temporal.incorporacion_personal.plan.registrar','contratacion_temporal.incorporacion_personal.origen.confirmar')
    OR c->>'audiencia_consumo' IS DISTINCT FROM (CASE accion
      WHEN 'contratacion_temporal.incorporacion_personal.plan.consultar' THEN 'vec_contratacion_temporal.incorporacion_personal.plan.consultar.v1'
      WHEN 'contratacion_temporal.incorporacion_personal.plan.registrar' THEN 'vec_contratacion_temporal.incorporacion_personal.plan.registrar.v1' ELSE 'vec_contratacion_temporal.incorporacion_personal.origen.confirmar.v1' END)
    OR d->>'accion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'incorporacion_personal_ct'
    OR d->>'finalidad' IS DISTINCT FROM 'incorporar_personal_desde_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'expediente_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->'campos_permitidos' IS DISTINCT FROM (CASE accion
      WHEN 'contratacion_temporal.incorporacion_personal.plan.consultar' THEN '["plan"]'::jsonb
      ELSE '["recibo"]'::jsonb END)
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-130: autorización divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'incorporacion_personal_ct',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-130: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_incorporacion_personal_ct_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 a record;
BEGIN
 FOR a IN SELECT DISTINCT x.grantee FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
 WHERE p.oid=f AND x.grantee<>p.proowner AND x.grantee<>'vec_contratacion_temporal_propietario'::regrole LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(a.grantee)) END);
 END LOOP;
 IF (SELECT proowner FROM pg_proc WHERE oid=f)<>'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
              WHERE p.oid=f AND x.grantee NOT IN (p.proowner,'vec_contratacion_temporal_propietario'::regrole))
 THEN RAISE EXCEPTION 'AD3-130: ACL incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
