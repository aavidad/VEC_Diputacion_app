\set ON_ERROR_STOP on
-- AD161: añade sólo los contratos pendientes AD141/142/144 al POST149+155 real.
-- AD149 de #470 se instala antes; CRN11 y AD155 se conservan íntegros.
-- Los archivos históricos no se modifican ni se registran como reaplicados.
-- Requiere los roles originales de Convocatorias, Méritos y Baremo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog, pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000161',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE rol text; nombre text; f155 oid; f149 oid;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
 OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'PARO clave=AD161.POST155, observado=propietario_%/fachada_%, esperado=propietario_true/fachada_true', current_user='vec_autorizacion_atestada_v3_propietario', to_regprocedure('vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY[
  'consumir_consulta_version_convocatoria_v3_atestada',
  'consumir_operacion_meritos_v3_atestada',
  'registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada'
 ] LOOP
  IF to_regprocedure('vec_autorizacion_atestada_v3.'||nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
  THEN RAISE EXCEPTION 'PARO clave=AD161.consumidor_%, observado=presente, esperado=ausente', nombre USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH rol IN ARRAY ARRAY[
  'vec_bolsa_convocatorias_propietario','vec_bolsa_convocatorias_ejecutor_consulta',
  'vec_meritos_propietario','vec_meritos_migrador','vec_meritos_ejecutor','vec_meritos_interno','vec_meritos_externo',
  'vec_bolsa_reglas_baremo_propietario','vec_bolsa_reglas_baremo_ejecutor_gobierno',
  'vec_personal_propietario','vec_personal_migrador','vec_personal_ejecutor'
 ] LOOP
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=rol AND NOT rolcanlogin
   AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls)
  THEN RAISE EXCEPTION 'PARO clave=AD161.rol_%, observado=ausente_o_atributos_incompatibles, esperado=NOLOGIN_sin_privilegios_elevados', rol USING ERRCODE='55000'; END IF;
 END LOOP;
 IF EXISTS (SELECT 1 FROM pg_auth_members WHERE member IN (
  'vec_bolsa_convocatorias_ejecutor_consulta'::regrole,'vec_meritos_ejecutor'::regrole,
  'vec_meritos_interno'::regrole,'vec_meritos_externo'::regrole,
  'vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole,'vec_personal_ejecutor'::regrole))
 OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_personal' AND p.proname='resolver_empleado_canonico_persona_v1'
   AND p.pronargs=2 AND p.proargtypes[0]='text'::regtype AND p.proargtypes[1]='timestamptz'::regtype
   AND p.proowner='vec_personal_propietario'::regrole)
 OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_personal' AND p.proname='bloquear_generacion_proyeccion_empleado_persona_v1'
   AND p.pronargs=1 AND p.proargtypes[0]='text'::regtype AND p.proowner='vec_personal_propietario'::regrole)
 OR NOT EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_contexto_actor_v1' AND p.proname='proyeccion_empleado_personal_v2'
   AND p.pronargs=2 AND p.proargtypes[0]='text'::regtype AND p.proargtypes[1]='timestamptz'::regtype
   AND p.proowner='vec_contexto_actor_v1_propietario'::regrole)
 THEN RAISE EXCEPTION 'PARO clave=AD161.dependencias_herencia, observado=incompatible, esperado=tres_objetos_Personal_CA_y_grupos_sin_herencia' USING ERRCODE='55000'; END IF;
 f149:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_vinculo_propio_crn11_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF f149 IS NULL THEN RAISE EXCEPTION 'PARO clave=AD161.POST149, observado=ausente, esperado=CRN11_instalado_por_AD149' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f149 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f149)<>2
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f149 AND (a.grantee NOT IN (p.proowner,'vec_personal_propietario'::regrole) OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'PARO clave=AD161.POST149_ACL, observado=incompatible, esperado=propietario_autorizacion_y_Personal_solo_EXECUTE' USING ERRCODE='55000'; END IF;
 f155:='vec_autorizacion_atestada_v3.consumir_solicitud_documental_bolsa_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 IF (SELECT encode(sha256(convert_to(pg_get_functiondef(f155),'UTF8')),'hex'))
   IS DISTINCT FROM '6cbc471ddab55e56b97d4c7c4b75149ffc09992fc735ccca461d1372000d1197'
 OR NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f155
  AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef
  AND p.proconfig=ARRAY['search_path=pg_catalog','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
  aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f155)<>2
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
  aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f155
  AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole)
   OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'PARO clave=AD161.fachada_POST155, observado_def=%/ACL_count=%/ACL_incompatible=%, esperado_def=6cbc471ddab55e56b97d4c7c4b75149ffc09992fc735ccca461d1372000d1197/ACL_count=2/ACL_incompatible=false', encode(sha256(convert_to(pg_get_functiondef(f155),'UTF8')),'hex'), (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f155), EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f155 AND (a.grantee NOT IN (p.proowner,'vec_bolsa_llamamientos_propietario'::regrole) OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) USING ERRCODE='55000'; END IF;
END $pre$;
DO $nucleo$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; actual text; fuente text; meta jsonb; deps jsonb; deps_compartidas jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 -- Preimagen POST149+155 real PG18.4, medida por #470 y cotejada en el ensayo.
 esperada_def_sha256 text:=$esperada_def_sha256$d912064d905e4349ffb2e1e8f1aab1aebef71e1fd842d5603373c4e6236fcd15$esperada_def_sha256$;
 esperada_fuente_sha256 text:=$esperada_fuente_sha256$1bb33d97bf8bbaa7f97dec4e1af1aa41577be38cea17f4b18375346cf7d62dcd$esperada_fuente_sha256$;
 marca text:=$marca$       )
       OR c ->> 'suite' <> 'VEC-AD-3-COSE-EDDSA-1'$marca$;
 excl text:=$excl$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
$excl$;
 excl_nuevo text:=$excl_nuevo$               p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento'
               AND p_perfil_mutacion IS DISTINCT FROM 'gobierno_borrador_reglas_baremo'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_interno'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_propio_externo'
               AND p_perfil_mutacion IS DISTINCT FROM 'meritos_hecho_rechazar'
               AND p_perfil_mutacion IS DISTINCT FROM 'consulta_version_convocatoria_bolsa'
$excl_nuevo$;
 runtime text:=$runtime$       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_propio_historico_crn11'
$runtime$;
 runtime_nuevo text:=$runtime_nuevo$       OR NOT (
           (
               p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_borrador_reglas_baremo'
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
                  AND NOT r.rolreplication AND NOT r.rolbypassrls)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_bolsa_reglas_baremo_ejecutor_gobierno'::regrole)
           )
           OR (
               p_perfil_mutacion IN ('meritos_hecho_propio_interno','meritos_hecho_rechazar')
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
                  AND NOT r.rolreplication AND NOT r.rolbypassrls)
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
               AND EXISTS (SELECT 1 FROM pg_roles r WHERE r.rolname=session_user AND r.rolcanlogin
                  AND NOT r.rolsuper AND NOT r.rolcreaterole AND NOT r.rolcreatedb
                  AND NOT r.rolreplication AND NOT r.rolbypassrls)
               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole
                  AND m.roleid='vec_bolsa_convocatorias_ejecutor_consulta'::regrole
                  AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option)
               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1
               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member='vec_bolsa_convocatorias_ejecutor_consulta'::regrole)
           )
           OR (
               p_perfil_mutacion IS NOT DISTINCT FROM 'vinculo_propio_historico_crn11'
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
           OR (
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
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'gobierno_borrador_reglas_baremo'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_bolsa_reglas_baremo.gobierno_borrador.v3'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reglas_baremo.borrador.crear'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'intencion_gobierno_reglas_baremo'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'gobierno_reglas_baremo'
       AND coalesce(d->>'recurso_ref','') ~ '^intencion-reglas-baremo:[0-9a-f]{64}$'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb)
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reglas_baremo.version.consultar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'version_reglas_baremo_gobernada'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_gobierno_reglas_baremo'
       AND coalesce(d->>'recurso_ref','') ~ '^reglas-baremo:[0-9a-f]{64}$'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["estado_reglas_baremo"]'::jsonb)
   OR (c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reglas_baremo.recibo.consultar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'version_reglas_baremo_gobernada'
       AND d->>'finalidad' IS NOT DISTINCT FROM 'consulta_gobierno_reglas_baremo'
       AND coalesce(d->>'recurso_ref','') ~ '^reglas-baremo:[0-9a-f]{64}$'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["estado_reglas_baremo","recibo"]'::jsonb))
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'bolsa'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'false'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
$extension$;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'PARO clave=AD161.nucleo_presencia, observado=ausente, esperado=presente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO original,fuente,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 IF NOT FOUND OR original IS NULL OR fuente IS NULL OR meta IS NULL
    OR propietario IS NULL OR config IS NULL OR definidora IS NULL
 THEN RAISE EXCEPTION 'PARO clave=AD161.nucleo_metadata, observado=incompleta, esperado=def_fuente_meta_propietario_config_definidora_presentes' USING ERRCODE='55000'; END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb)
 INTO deps_compartidas FROM pg_shdepend d
 WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())
   AND d.classid='pg_proc'::regclass AND d.objid=f;
 -- Tres contratos nominales: exclusiones del bloque general y selectores
 -- de sesión de Convocatorias, Méritos y Baremo; Personal ya procede de AD149.
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
    OR strpos(original,'consulta_version_convocatoria_bolsa')<>0
    OR strpos(original,'bolsa.convocatoria.version.consultar')<>0
    OR strpos(original,'meritos_hecho_propio_')<>0
    OR strpos(original,'meritos_hecho_rechazar')<>0
    OR strpos(original,'gobierno_borrador_reglas_baremo')<>0
    OR strpos(original,'vinculo_propio_historico_crn11')=0
 THEN RAISE EXCEPTION 'PARO clave=AD161.nucleo_preimagen, observado_def=%/fuente=%/metadata=%, esperado_def=%/fuente=%/metadata=propietario_y_ACL_nominal_config_pg_catalog_pg_temp_lock_timeout_2s_dependencias_y_anclas_unicas', encode(sha256(convert_to(original,'UTF8')),'hex'), encode(sha256(convert_to(fuente,'UTF8')),'hex'), encode(sha256(convert_to(meta::text,'UTF8')),'hex'), esperada_def_sha256, esperada_fuente_sha256 USING ERRCODE='55000'; END IF;
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
 THEN RAISE EXCEPTION 'PARO clave=AD161.nucleo_postimagen, observado_def=%/metadata=%/dependencias_iguales=%, esperado_def=%/metadata=%/dependencias_iguales=true', encode(sha256(convert_to(actual,'UTF8')),'hex'), (SELECT encode(sha256(convert_to((to_jsonb(p)-'prosrc')::text,'UTF8')),'hex') FROM pg_proc p WHERE p.oid=f), ((SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS NOT DISTINCT FROM deps AND (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database()) AND d.classid='pg_proc'::regclass AND d.objid=f) IS NOT DISTINCT FROM deps_compartidas), encode(sha256(convert_to(nuevo,'UTF8')),'hex'), encode(sha256(convert_to(meta::text,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; a text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF encode(sha256(convert_to(d,'UTF8')),'hex') IS DISTINCT FROM '0ba3eabde2f45d27afd278cfc008ddc0c3a24cc6d0e5de790b5c6d65dec906a6'
 OR strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
 THEN RAISE EXCEPTION 'PARO clave=AD161.audiencias_preimagen, observado=%, esperado=0ba3eabde2f45d27afd278cfc008ddc0c3a24cc6d0e5de790b5c6d65dec906a6', encode(sha256(convert_to(d,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
 nueva:=d;
 FOREACH a IN ARRAY ARRAY[
  'vec_bolsa_convocatorias.version.consultar.v1',
  'vec_meritos.hecho.declarar.v1','vec_meritos.hecho.rectificar.v1','vec_meritos.hecho.rechazar.v1',
  'vec_bolsa_reglas_baremo.gobierno_borrador.v3'
 ] LOOP
  IF strpos(d,quote_literal(a))<>0 THEN
   RAISE EXCEPTION 'PARO clave=AD161.audiencia_%, observado=presente, esperado=ausente', a USING ERRCODE='55000'; END IF;
  nueva:=left(nueva,length(nueva)-3)||', '||quote_literal(a)||'::text]))';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
 IF (SELECT pg_get_constraintdef(c.oid,true) FROM pg_constraint c
  WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated) IS DISTINCT FROM nueva
 THEN RAISE EXCEPTION 'PARO clave=AD161.audiencias_postimagen, observado=distinta_o_sin_validar, esperado=%', encode(sha256(convert_to(nueva,'UTF8')),'hex') USING ERRCODE='55000'; END IF;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-141: material de versión de convocatoria inválido' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.convocatoria.version.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_convocatorias.version.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'version_convocatoria_gobernada'
    OR d->>'finalidad' IS DISTINCT FROM 'consulta_interna_convocatorias'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
    OR d->'campos_permitidos' IS DISTINCT FROM '["version_convocatoria"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-141: versión de convocatoria denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'consulta_version_convocatoria_bolsa',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-141: la lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_convocatorias_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_consulta_version_convocatoria_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_bolsa_convocatorias_propietario'::regrole::oid; x record;
BEGIN
 -- También las ACL por defecto: ningún rol conserva acceso por haber sido
 -- destinatario predeterminado del propietario.
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_bolsa_convocatorias_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_bolsa_convocatorias_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD3-141: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-141: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog, pg_temp SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-142: material RUM inválido' USING ERRCODE='22023'; END;
 -- El exterior permanece cerrado hasta disponer de confianza segregada nominal.
 -- Se rechaza antes de llamar al núcleo interno, incluso con material correcto.
 IF d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'externa_personal'
 THEN RAISE EXCEPTION 'AD3-142: exterior no habilitado' USING ERRCODE='42501'; END IF;
 -- No se consulta el agregado ni se publica una concesión; solo el núcleo común
 -- revalida y consume. El módulo aplica titularidad/CAS/historia en la misma TX.
 IF (
 ((c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.declarar'
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
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 ) IS TRUE THEN perfil:='meritos_hecho_propio_interno';
 ELSIF (
 c->>'operacion' IS NOT DISTINCT FROM 'meritos.hecho.rechazar'
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
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 ) IS TRUE THEN perfil:='meritos_hecho_rechazar';
 ELSE RAISE EXCEPTION 'AD3-142: operación RUM denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-142: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_meritos_propietario;
DO $acl$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.consumir_operacion_meritos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 permitido oid:='vec_meritos_propietario'::regrole::oid; x record;
BEGIN
 -- También las ACL por defecto: ningún rol conserva acceso por haber sido
 -- destinatario predeterminado del propietario.
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO vec_meritos_propietario',f::text);
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
    OR NOT has_schema_privilege('vec_meritos_propietario','vec_autorizacion_atestada_v3','USAGE')
 THEN RAISE EXCEPTION 'AD3-142: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
  FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
    OR (x.grantee=permitido AND x.is_grantable)
  THEN RAISE EXCEPTION 'AD3-142: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,
 p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,
 consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,
 consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog, pg_temp SET lock_timeout='2s'
AS $f$
DECLARE c jsonb; d jsonb; x record; escritura boolean;
BEGIN
 IF p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD3-144: material V3 fuera de límites' USING ERRCODE='22023'; END IF;
 BEGIN
  c:=convert_from(p_capacidad,'UTF8')::jsonb;
  d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN
  RAISE EXCEPTION 'AD3-144: material V3 inválido' USING ERRCODE='22023';
 END;
 escritura:=c->>'operacion' IS NOT DISTINCT FROM 'bolsa.reglas_baremo.borrador.crear';
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_reglas_baremo.gobierno_borrador.v3'
 OR coalesce(c->>'operacion','') NOT IN ('bolsa.reglas_baremo.borrador.crear','bolsa.reglas_baremo.version.consultar','bolsa.reglas_baremo.recibo.consultar')
 OR d->>'accion' IS DISTINCT FROM c->>'operacion'
 OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
 OR d->>'tipo_recurso' IS DISTINCT FROM (CASE WHEN escritura THEN 'intencion_gobierno_reglas_baremo' ELSE 'version_reglas_baremo_gobernada' END)
 OR d->>'finalidad' IS DISTINCT FROM (CASE WHEN escritura THEN 'gobierno_reglas_baremo' ELSE 'consulta_gobierno_reglas_baremo' END)
 OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
 OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'interna_corporativa'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'false'
 OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 OR (escritura AND (coalesce(d->>'recurso_ref','') !~ '^intencion-reglas-baremo:[0-9a-f]{64}$'
   OR d->'campos_permitidos' IS DISTINCT FROM '["auditoria","estado_reglas_baremo","salida_eventos"]'::jsonb))
 OR (NOT escritura AND coalesce(d->>'recurso_ref','') !~ '^reglas-baremo:[0-9a-f]{64}$')
 OR (c->>'operacion'='bolsa.reglas_baremo.version.consultar'
   AND d->'campos_permitidos' IS DISTINCT FROM '["estado_reglas_baremo"]'::jsonb)
 OR (c->>'operacion'='bolsa.reglas_baremo.recibo.consultar'
   AND d->'campos_permitidos' IS DISTINCT FROM '["estado_reglas_baremo","recibo"]'::jsonb)
 THEN RAISE EXCEPTION 'AD3-144: capacidad GobiernoG denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'gobierno_borrador_reglas_baremo',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN
  RAISE EXCEPTION 'AD3-144: requiere consumo fresco' USING ERRCODE='42501';
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_bolsa_reglas_baremo_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_reglas_baremo_propietario;
DO $acl$
DECLARE f oid:='vec_autorizacion_atestada_v3.registrar_y_consumir_gobierno_borrador_reglas_baremo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
 OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
 OR EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.privilege_type<>'EXECUTE' OR a.is_grantable OR a.grantor<>p.proowner
    OR a.grantee NOT IN (p.proowner,'vec_bolsa_reglas_baremo_propietario'::regrole)))
 OR NOT has_function_privilege('vec_bolsa_reglas_baremo_propietario',f,'EXECUTE')
 THEN RAISE EXCEPTION 'AD3-144: ACL de consumidor incompatible' USING ERRCODE='55000'; END IF;
END $acl$;

-- La fachada CRN11 de AD149 no se redefine ni cambia sus ACL.
COMMIT;
