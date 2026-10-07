\set ON_ERROR_STOP on
-- AD3-136 sobre la postimagen exacta de AD3-133/135 c6b29fd4 tras #222.
-- AD3-136 añade únicamente el caso de custodia a la ligadura documental.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000136',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE
 f regprocedure:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 w regprocedure;
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb;
 acl aclitem[]; propietario oid; f_inicial oid;
 -- Inventario PG18 post-B SHA256 abe50dc52be1a4257da496b3763baa6f04789747b471f1f78c4fe965f1663d39.
 esperada_def_sha256 text:='5ddcf72a8cd5c1a414673a0c843e64fc18f8d6ad7a1d304e941cafae7c94e42c';
 esperada_fuente_sha256 text:='26edea3c1765d60f39628805d5fb1c5956004cd832251705b36874e212c50793';
 esperada_wrapper_sha256 text:='3e4412dbac79bcce6d0425f4c0fddfbb32602ed2169b9d694c2292fbb83a5d4f';
 esperada_doc9_sha256 text:='8a7b601db5bfb8965eb11bd20a0b4c1ef51e92ffebac9d9dc48daa563b783ff1';
 doc9_total integer; doc9_exacto integer; doc9_huella text;
 antiguo text:=$x$   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
 ))$x$;
 ampliado text:=$x$   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.firmado.custodiar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_firmado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_documento_firmado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento_firmado.custodia","evidencia_custodia"]'::jsonb)
 ))$x$;
BEGIN
 IF f IS NULL THEN
  RAISE EXCEPTION 'AD3-136: núcleo V3 ausente' USING ERRCODE='55000';
 END IF;
 w:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 -- El propietario AD3 carece de USAGE en vec_documentos. Resolver el nombre
 -- mediante regprocedure exigiría ese permiso; el catálogo permite comprobar
 -- la firma y descartar cualquier sobrecarga sin conceder acceso al esquema.
 SELECT count(*),count(*) FILTER (WHERE p.prokind='f' AND p.pronargs=13
   AND p.proowner='vec_documentos_propietario'::regrole
   AND p.prosecdef
   AND p.proargtypes=ARRAY[
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.jsonb'::regtype::oid,'pg_catalog.jsonb'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.numeric'::regtype::oid,'pg_catalog.numeric'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,'pg_catalog.bytea'::regtype::oid,
    'pg_catalog.bytea'::regtype::oid]::oidvector),
   max(encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex'))
     FILTER (WHERE p.prokind='f' AND p.pronargs=13)
   INTO doc9_total,doc9_exacto,doc9_huella FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_documentos' AND p.proname='custodiar_firmado_v1';
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR w IS NULL
    OR encode(sha256(convert_to(pg_get_functiondef(w),'UTF8')),'hex') IS DISTINCT FROM esperada_wrapper_sha256
    OR doc9_total<>1 OR doc9_exacto<>1
    OR doc9_huella IS DISTINCT FROM esperada_doc9_sha256
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=w
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
         AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
         AND EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                     WHERE a.grantee='vec_documentos_propietario'::regrole
                       AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
         AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                         WHERE a.grantee NOT IN (p.proowner,'vec_documentos_propietario'::regrole)
                            OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
         AND (SELECT count(*) FROM pg_depend d
              WHERE d.classid='pg_proc'::regclass AND d.objid=p.oid)=2
         AND EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass
              AND d.objid=p.oid AND d.objsubid=0 AND d.refclassid='pg_language'::regclass
              AND d.refobjid=(SELECT oid FROM pg_language WHERE lanname='plpgsql')
              AND d.refobjsubid=0 AND d.deptype='n')
         AND EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass
              AND d.objid=p.oid AND d.objsubid=0 AND d.refclassid='pg_namespace'::regclass
              AND d.refobjid='vec_autorizacion_atestada_v3'::regnamespace
              AND d.refobjsubid=0 AND d.deptype='n')
         AND (SELECT count(*) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
              AND d.classid='pg_proc'::regclass AND d.objid=p.oid)=2
         AND EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
              AND d.classid='pg_proc'::regclass AND d.objid=p.oid AND d.objsubid=0
              AND d.refclassid='pg_authid'::regclass AND d.refobjid=p.proowner AND d.deptype='o')
         AND EXISTS (SELECT 1 FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
              AND d.classid='pg_proc'::regclass AND d.objid=p.oid AND d.objsubid=0
              AND d.refclassid='pg_authid'::regclass
              AND d.refobjid='vec_documentos_propietario'::regrole AND d.deptype='a'))
    OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
         WHERE n.nspname='vec_documentos' AND p.proname='custodiar_firmado_v1'
           AND p.proowner='vec_documentos_propietario'::regrole AND p.prosecdef
           AND p.proconfig=ARRAY['search_path=pg_catalog','row_security=on','TimeZone=UTC','lock_timeout=2s']
           AND EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                       WHERE a.grantee='vec_documentos_ejecutor'::regrole
                         AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
           AND NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
                           WHERE a.grantee NOT IN (p.proowner,'vec_documentos_ejecutor'::regrole)
                              OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD3-136: AD3-113 o Documentos9 incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.oid
   INTO STRICT original,fuente,meta,acl,propietario,f_inicial
   FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
   INTO deps_compartidas FROM pg_shdepend d
   WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
     AND d.classid='pg_proc'::regclass AND d.objid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM esperada_def_sha256
    OR encode(sha256(convert_to(fuente,'UTF8')),'hex') IS DISTINCT FROM esperada_fuente_sha256
    OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f
         AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
         AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u'
         AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
    OR EXISTS (SELECT 1 FROM pg_database db
         CROSS JOIN LATERAL aclexplode(coalesce(db.datacl,acldefault('d',db.datdba))) a
         WHERE db.datname=current_database() AND a.grantee=0 AND a.privilege_type='TEMPORARY')
    OR NOT EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
                   WHERE a.grantee=propietario AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
    OR EXISTS (SELECT 1 FROM aclexplode(coalesce(acl,acldefault('f',propietario))) a
               WHERE a.grantee<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)
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
 THEN RAISE EXCEPTION 'AD3-136: definición o ACL de núcleo post-B incompatible' USING ERRCODE='55000'; END IF;
 IF length(original)-length(replace(original,antiguo,''))<>length(antiguo)
    OR strpos(original,'operacion_documentos_comunes')=0
    OR strpos(original,'documentos.firmado.custodiar')<>0
 THEN RAISE EXCEPTION 'AD3-136: preimagen del núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,antiguo,ampliado);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo OR replace(actual,ampliado,antiguo) IS DISTINCT FROM original
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')::oid IS DISTINCT FROM f_inicial
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
        FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
          AND d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps_compartidas
 THEN RAISE EXCEPTION 'AD3-136: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
COMMIT;
