\set ON_ERROR_STOP on
-- AD185: consumo nominal para lista y ficha administrativas AUT43.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000185',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE f oid:=to_regprocedure('vec_autorizacion.validar_administrador_usuarios_v1(jsonb,jsonb)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD185: PARO clave=operador actual=incompatible esperado=superusuario_PG18' USING ERRCODE='55000';END IF;
 IF f IS NULL OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_propietario'::regrole AND p.prosecdef AND p.prorettype='boolean'::regtype)
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario',f,'EXECUTE') IS NOT TRUE
 OR to_regprocedure('vec_autorizacion.recurso_lectura_usuarios_admin_v1(text)') IS NULL
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_autorizacion.recurso_lectura_usuarios_admin_v1(text)','EXECUTE') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD185: PARO clave=AUT43 actual=ausente_o_no_acreditado esperado=gate_y_recurso_propietario_AUT' USING ERRCODE='55000';END IF;
 IF to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_denominacion_persona_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)') IS NULL
 OR to_regrole('vec_admin_usuarios_lector') IS NULL
 THEN RAISE EXCEPTION 'AD185: PARO clave=preimagen actual=incompatible esperado=AD173_AD184_AUT43_sin_AD185' USING ERRCODE='55000';END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.login_usuarios_admin_valido_v1()
RETURNS boolean LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
 SELECT current_setting('role')='none' AND EXISTS(
 SELECT 1 FROM pg_roles l JOIN pg_auth_members m ON m.member=l.oid JOIN pg_roles g ON g.oid=m.roleid
 WHERE l.rolname=session_user AND l.rolcanlogin AND l.rolinherit AND NOT(l.rolsuper OR l.rolcreatedb OR l.rolcreaterole OR l.rolreplication OR l.rolbypassrls) AND l.rolconfig IS NULL
 AND g.oid=to_regrole('vec_admin_usuarios_lector') AND NOT(g.rolcanlogin OR g.rolinherit OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls) AND g.rolconfig IS NULL
 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
 AND (SELECT count(*) FROM pg_auth_members x WHERE x.member=l.oid)=1
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members x WHERE x.member=g.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_db_role_setting x WHERE x.setrole IN(l.oid,g.oid)))
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.login_usuarios_admin_valido_v1() FROM PUBLIC;

CREATE FUNCTION vec_autorizacion_atestada_v3.validar_contrato_usuarios_admin_v1(p_material text,p_capacidad bytea,p_decision bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE r jsonb;c jsonb;d jsonb;campos jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off' OR current_setting('TimeZone')<>'UTC'
 OR current_setting('role')<>'none' OR session_user=current_user
 OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
 OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
 THEN RAISE EXCEPTION 'AD185: contrato_rechazado' USING ERRCODE='42501';END IF;
 r:=vec_autorizacion.recurso_lectura_usuarios_admin_v1(p_material);
 IF r->>'accion' IS NULL OR r->>'accion' NOT IN('administracion.usuarios.listar','administracion.usuarios.consultar')
 THEN RAISE EXCEPTION 'AD185: accion_rechazada' USING ERRCODE='42501';END IF;
 c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 campos:=CASE WHEN r->>'accion'='administracion.usuarios.listar' THEN '["denominacion_version","perfiles","persona_ref","siguiente_cursor","unidad_ref"]'::jsonb
 ELSE '["denominacion_version","perfiles","persona_ref","unidad_ref"]'::jsonb END;
 IF c->>'operacion' IS DISTINCT FROM r->>'accion' OR c->>'audiencia_consumo' IS DISTINCT FROM r->>'audiencia'
 OR c->>'efecto_ref' IS DISTINCT FROM r->>'recurso_ref' OR c->>'huella_efecto_sha256' IS DISTINCT FROM r->>'contexto_sha256'
 OR d->>'accion' IS DISTINCT FROM r->>'accion' OR d->>'modulo_id' IS DISTINCT FROM 'administracion' OR d->>'tipo_recurso' IS DISTINCT FROM r->>'tipo_recurso'
 OR d->>'recurso_ref' IS DISTINCT FROM r->>'recurso_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM r->>'contexto_sha256'
 OR d->>'finalidad' IS DISTINCT FROM 'gestion_usuarios' OR d->'campos_permitidos' IS DISTINCT FROM campos
 OR d->'obligaciones' IS DISTINCT FROM '["auditar"]'::jsonb OR d->>'garantia_minima' IS DISTINCT FROM 'alto'
 OR c->>'decision_ref' IS DISTINCT FROM d->>'decision_ref' OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
 OR d->>'correlacion_ref' IS NULL OR d->>'correlacion_ref' !~ '^correlacion_[0-9a-f]{32}$'
 OR d#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
 OR d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
 OR vec_autorizacion.validar_administrador_usuarios_v1(d,r->'material') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD185: decision_nominal_rechazada' USING ERRCODE='42501';END IF;
 RETURN r;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.validar_contrato_usuarios_admin_v1(text,bytea,bytea) FROM PUBLIC;

DO $nucleo$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text;fuente text;nueva text;actual text;pre184 text;pre184_fuente text;meta jsonb;deps jsonb;compartidas jsonb;h text;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 runtime_marca text:=E'       OR NOT (\n           (p_perfil_mutacion IS NOT DISTINCT FROM ''capacidades_admin''';
 runtime_ad184 text:=E'       OR NOT (\n           (p_perfil_mutacion IN (''persona_denominacion_publicar'',''persona_denominacion_leer'')\n            AND vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1() IS TRUE)\n           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''capacidades_admin''';
 excl_marca text:=$x$p_perfil_mutacion IS DISTINCT FROM 'bolsa_llamamiento' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_certificado_nominal' AND p_perfil_mutacion IS DISTINCT FROM 'publicacion_cargo_competencial' AND p_perfil_mutacion IS DISTINCT FROM 'capacidades_admin'$x$;
 extension_ad184 text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'persona_denominacion_publicar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.persona.denominacion.publicar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.persona.denominacion.publicar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_denominacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentacion_persona'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'persona_denominacion_leer'
 AND c->>'operacion' IS NOT DISTINCT FROM 'vec.persona.denominacion.leer'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.persona.denominacion.leer.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'vec'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_denominacion'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'presentacion_persona'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["nombre_mostrar"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
$x$;
 excl_ad184 text:=excl_marca||' AND p_perfil_mutacion IS DISTINCT FROM ''persona_denominacion_publicar'' AND p_perfil_mutacion IS DISTINCT FROM ''persona_denominacion_leer''';
 runtime_ad185 text:=E'       OR NOT (\n           (p_perfil_mutacion IN (''usuarios_admin_listar'',''usuarios_admin_consultar'')\n            AND vec_autorizacion_atestada_v3.login_usuarios_admin_valido_v1() IS TRUE)\n           OR (p_perfil_mutacion IN (''persona_denominacion_publicar'',''persona_denominacion_leer'')\n            AND vec_autorizacion_atestada_v3.login_denominacion_persona_valido_v1() IS TRUE)\n           OR (p_perfil_mutacion IS NOT DISTINCT FROM ''capacidades_admin''';
 excl_ad185 text:=excl_ad184||' AND p_perfil_mutacion IS DISTINCT FROM ''usuarios_admin_listar'' AND p_perfil_mutacion IS DISTINCT FROM ''usuarios_admin_consultar''';
 extension_ad185 text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'usuarios_admin_listar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.usuarios.listar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.admin.usuarios.listar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'conjunto_usuarios'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_usuarios'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^conjunto_admin:[0-9a-f]{32}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion_version","perfiles","persona_ref","siguiente_cursor","unidad_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'usuarios_admin_consultar'
 AND c->>'operacion' IS NOT DISTINCT FROM 'administracion.usuarios.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec.admin.usuarios.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'administracion'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'persona_administrable'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'gestion_usuarios'
 AND d->>'garantia_minima' IS NOT DISTINCT FROM 'alto'
 AND d->>'recurso_ref' IS NOT NULL AND d->>'recurso_ref' ~ '^per_[A-Za-z0-9_-]{22,124}$'
 AND c->>'efecto_ref' IS NOT DISTINCT FROM d->>'recurso_ref'
 AND c->>'huella_efecto_sha256' IS NOT DISTINCT FROM d->>'contexto_recurso_huella_sha256'
 AND d->'campos_permitidos' IS NOT DISTINCT FROM '["denominacion_version","perfiles","persona_ref","unidad_ref"]'::jsonb
 AND d->'obligaciones' IS NOT DISTINCT FROM '["auditar"]'::jsonb
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'administracion_privilegiada'
 AND d#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS NOT DISTINCT FROM 'true')
$x$;
BEGIN
 SELECT pg_get_functiondef(f),p.prosrc,to_jsonb(p)-'prosrc' INTO STRICT original,fuente,meta FROM pg_proc p WHERE p.oid=f;
 h:=encode(sha256(convert_to(original,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'ff77db3d6dac93c3ba8f359e6bca22a8a03489954b9c120acef4e44dc90a0bd6'
 THEN RAISE EXCEPTION 'AD185: PARO clave=nucleo_postAD184_def_SHA actual=% esperado=ff77db3d6dac93c3ba8f359e6bca22a8a03489954b9c120acef4e44dc90a0bd6',h USING ERRCODE='55000';END IF;
 h:=encode(sha256(convert_to(fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM '73cb05e1c57c82c13e066a1f3c27cf03d9bdbaed3e028184ee9d040b9b230735'
 THEN RAISE EXCEPTION 'AD185: PARO clave=nucleo_postAD184_src_SHA actual=% esperado=73cb05e1c57c82c13e066a1f3c27cf03d9bdbaed3e028184ee9d040b9b230735',h USING ERRCODE='55000';END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s'])
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>1
 OR NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=p.proowner AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
 THEN RAISE EXCEPTION 'AD185: PARO clave=nucleo_metadatos actual=incompatible esperado=AD184_owner_privado' USING ERRCODE='55000';END IF;
 IF length(original)-length(replace(original,extension_ad184||marca,''))<>length(extension_ad184||marca)
 OR length(original)-length(replace(original,runtime_ad184,''))<>length(runtime_ad184)
 OR length(original)-length(replace(original,excl_ad184,''))<>length(excl_ad184)
 THEN RAISE EXCEPTION 'AD185: PARO clave=marcas_AD184 actual=no_unicas esperado=tres_marcas_unicas' USING ERRCODE='55000';END IF;
 -- Las dos huellas se miden invirtiendo exactamente AD184; no se supone su SHA posterior.
 pre184:=replace(replace(replace(original,extension_ad184||marca,marca),excl_ad184,excl_marca),runtime_ad184,runtime_marca);
 pre184_fuente:=replace(replace(replace(fuente,extension_ad184||marca,marca),excl_ad184,excl_marca),runtime_ad184,runtime_marca);
 h:=encode(sha256(convert_to(pre184,'UTF8')),'hex');
 IF h IS DISTINCT FROM '6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9'
 THEN RAISE EXCEPTION 'AD185: PARO clave=preAD184_def_SHA actual=% esperado=6c22fdbb165a00c4f37cb2f7dbb7add4e939e5b0134c0c599b9519bfe3b86db9',h USING ERRCODE='55000';END IF;
 h:=encode(sha256(convert_to(pre184_fuente,'UTF8')),'hex');
 IF h IS DISTINCT FROM 'bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c'
 THEN RAISE EXCEPTION 'AD185: PARO clave=preAD184_src_SHA actual=% esperado=bbb932ef29375e88645fb524e6aae5fd3059d51cb0dfd9d4952ffd509470534c',h USING ERRCODE='55000';END IF;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) INTO compartidas FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database());
 nueva:=replace(replace(replace(original,runtime_ad184,runtime_ad185),excl_ad184,excl_ad185),extension_ad184||marca,extension_ad184||extension_ad185||marca);
 EXECUTE nueva;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nueva
 OR replace(replace(replace(actual,extension_ad184||extension_ad185||marca,extension_ad184||marca),excl_ad185,excl_ad184),runtime_ad185,runtime_ad184) IS DISTINCT FROM original
 OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.dbid,d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.deptype),'[]'::jsonb) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid=f AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())) IS DISTINCT FROM compartidas
 THEN RAISE EXCEPTION 'AD185: PARO clave=postimagen_nucleo actual=divergente esperado=extension_minima_metadata_intacta' USING ERRCODE='55000';END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE anterior text;nueva text;h text;
BEGIN
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT anterior FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 h:=encode(sha256(convert_to(anterior,'UTF8')),'hex');
 -- Preimagen medida tras AD184 exacta: PostgreSQL deparsa IN como ANY.
 IF h IS DISTINCT FROM '4c57e39c9b725b428149fea490bd38e6c41d608b8c727609d548363008576ebd'
 OR left(anterior,7)<>'CHECK (' OR right(anterior,1)<>')'
 THEN RAISE EXCEPTION 'AD185: PARO clave=CHECK_postAD184_SHA actual=% esperado=4c57e39c9b725b428149fea490bd38e6c41d608b8c727609d548363008576ebd',h USING ERRCODE='55000';END IF;
 nueva:='CHECK (('||substr(anterior,8,length(anterior)-8)||') OR audiencia_consumo IN (''vec.admin.usuarios.listar.v1'',''vec.admin.usuarios.consultar.v1''))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE r jsonb;d jsonb;x record;perfil text;
BEGIN
 r:=vec_autorizacion_atestada_v3.validar_contrato_usuarios_admin_v1(p_material,p_capacidad,p_decision);
 IF vec_autorizacion_atestada_v3.login_usuarios_admin_valido_v1() IS NOT TRUE THEN RAISE EXCEPTION 'AD185: login_rechazado' USING ERRCODE='42501';END IF;
 perfil:=CASE WHEN r->>'accion'='administracion.usuarios.listar' THEN 'usuarios_admin_listar' ELSE 'usuarios_admin_consultar' END;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 d:=convert_from(p_decision,'UTF8')::jsonb;
 IF x.consumo_nuevo IS NOT TRUE OR x.decision_ref IS DISTINCT FROM d->>'decision_ref'
 OR x.efecto_ref IS DISTINCT FROM r->>'recurso_ref' OR x.huella_efecto_sha256 IS DISTINCT FROM r->>'contexto_sha256'
 OR x.auditoria_ref IS NULL OR x.auditoria_ref !~ '^aud_v3_[0-9a-f]{32}$'
 OR NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 a JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c USING(decision_ref) JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t USING(decision_ref)
  WHERE a.auditoria_ref=x.auditoria_ref AND a.tipo_registro='consumo_confirmado_v3' AND a.version_consumo=3
  AND a.efecto_ref=x.efecto_ref AND a.huella_efecto_sha256=x.huella_efecto_sha256
  AND a.actor_ref=d->>'principal_id' AND a.perfil_activo_ref=d->>'perfil_activo_ref' AND a.finalidad_ref='gestion_usuarios'
  AND a.registrada_en=x.consumida_en AND c.consumida_en=x.consumida_en AND c.consumo_huella_sha256=x.consumo_huella_sha256
  AND t.capacidad_canonica=p_capacidad AND t.decision_canonica=p_decision AND t.motivo_canonico=p_motivo AND t.contexto_actor_canonico=p_contexto
  AND t.payload_vec_ad_3=p_payload AND t.sobre_cose_sign1=p_sobre AND t.evidencia_verificacion=p_evidencia AND t.raiz_publica_spki=p_raiz)
 OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
 OR vec_autorizacion.validar_administrador_usuarios_v1(d,r->'material') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD185: acuse_o_autoridad_rechazados' USING ERRCODE='42501';END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_autorizacion_propietario;
DO $acl$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_usuarios_admin_v3_atestada(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
 OR NOT has_function_privilege('vec_autorizacion_propietario',f,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole) OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AD185: PARO clave=ACL actual=ampliada esperado=owner_AD_y_AUT_EXECUTE' USING ERRCODE='55000';END IF;
END $acl$;
COMMIT;
