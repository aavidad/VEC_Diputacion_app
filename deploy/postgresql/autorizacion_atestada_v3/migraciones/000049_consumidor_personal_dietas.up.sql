\set ON_ERROR_STOP on
-- AD3-000049: Personal y Dietas sobre la estructura instalada hasta AD3-48.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000049',0));
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; inverso text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'emision_llamamiento_bolsa'$x$;
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_relacion_propia_dietas_personal''\n               AND p_perfil_mutacion IS DISTINCT FROM ''crear_borrador_propio_dietas''\n               AND p_perfil_mutacion IS DISTINCT FROM ''consultar_borrador_propio_dietas''';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IN (''consulta_relacion_propia_dietas_personal'',''crear_borrador_propio_dietas'',''consultar_borrador_propio_dietas'')\n               AND pg_catalog.pg_has_role(session_user,''vec_dietas_ejecutor'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_dietas_ejecutor''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_dietas_ejecutor''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_dietas_ejecutor'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_relacion_propia_dietas_personal'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.relacion_propia.consultar_dietas.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.relacion.propia.consultar_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM 'personal.relacion.propia.consultar_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'relacion_empleado_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_borrador_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'crear_borrador_propio_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.crear.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.crear'
 AND d->>'accion' IS NOT DISTINCT FROM 'dietas.borrador.propio.crear'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'crear_borrador_propio')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consultar_borrador_propio_dietas'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.consultar.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.consultar'
 AND d->>'accion' IS NOT DISTINCT FROM 'dietas.borrador.propio.consultar'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_borrador_propio')
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_emision_llamamiento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls) THEN RAISE EXCEPTION 'AD3-49: precondición incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR strpos(original,'emision_llamamiento_bolsa')=0 OR strpos(original,'consulta_relacion_propia_dietas_personal')<>0 THEN RAISE EXCEPTION 'AD3-49: núcleo AD3-48 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo); nuevo:=replace(nuevo,guarda,guarda_nueva); nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 inverso:=replace(actual,extension||marca,marca); inverso:=replace(inverso,guarda_nueva,guarda); inverso:=replace(inverso,excl_nuevo,excl);
 IF actual IS DISTINCT FROM nuevo OR inverso IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps THEN RAISE EXCEPTION 'AD3-49: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 -- AD44 preserva el conjunto anterior completo; no se presupone su lista ni
 -- se acepta una forma distinta, un valor repetido o una extensión parcial.
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_bolsa_llamamientos.borrador_llamamiento_interno.crear.v1')=0
    OR strpos(d,'vec_bolsa_llamamientos.borrador_llamamiento_interno.consultar.v1')=0
    OR strpos(d,'vec_personal.relacion_propia.consultar_dietas.v1')<>0
    OR strpos(d,'vec_dietas.borrador_propio.crear.v1')<>0
    OR strpos(d,'vec_dietas.borrador_propio.consultar.v1')<>0 THEN
   RAISE EXCEPTION 'AD3-49: audiencia post-AD48 incompatible' USING ERRCODE='55000';
 END IF;
 nueva:=left(d,length(d)-3)||', ''vec_personal.relacion_propia.consultar_dietas.v1''::text, ''vec_dietas.borrador_propio.crear.v1''::text, ''vec_dietas.borrador_propio.consultar.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;

-- Nombre físico <=63 bytes: el contrato previo de 64 bytes se descartó antes
-- de instalar por truncarse silenciosamente en PostgreSQL.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; campos jsonb:='["desde","empleado_ref","estado","fuente_ref","fuente_version","hasta","persona_ref","procedencia_acto_ref","relacion_ref","unidad_ref","version"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-49: material Personal inválido' USING ERRCODE='22023'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.relacion_propia.consultar_dietas.v1' OR c->>'operacion' IS DISTINCT FROM 'personal.relacion.propia.consultar_dietas' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'personal' OR d->>'tipo_recurso' IS DISTINCT FROM 'relacion_empleado_dietas' OR d->>'finalidad' IS DISTINCT FROM 'preparar_borrador_dietas' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'AD3-49: sello Personal Dietas rechazado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna('consulta_relacion_propia_dietas_personal',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-49: consulta Personal requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text; campos jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb; EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-49: material Dietas inválido' USING ERRCODE='22023'; END;
 IF d->>'modulo_id' IS DISTINCT FROM 'dietas' OR d->>'tipo_recurso' IS DISTINCT FROM 'comision_borrador' OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256' THEN RAISE EXCEPTION 'AD3-49: sello Dietas inválido' USING ERRCODE='42501'; END IF;
 CASE c->>'operacion'
  WHEN 'dietas.borrador.propio.crear' THEN campos:='["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb; IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.borrador_propio.crear.v1' OR d->>'finalidad' IS DISTINCT FROM 'crear_borrador_propio' THEN RAISE EXCEPTION 'AD3-49: crear Dietas rechazado' USING ERRCODE='42501'; END IF; perfil:='crear_borrador_propio_dietas';
  WHEN 'dietas.borrador.propio.consultar' THEN campos:='["comision.calculo","comision.codigos_ruta","comision.estado","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.referencia","comision.relacion_ref","items.comision.calculo","items.comision.codigos_ruta","items.comision.estado","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.referencia","items.comision.relacion_ref","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb; IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.borrador_propio.consultar.v1' OR d->>'finalidad' IS DISTINCT FROM 'consultar_borrador_propio' THEN RAISE EXCEPTION 'AD3-49: consulta Dietas rechazada' USING ERRCODE='42501'; END IF; perfil:='consultar_borrador_propio_dietas';
  ELSE RAISE EXCEPTION 'AD3-49: operación Dietas no admitida' USING ERRCODE='42501';
 END CASE;
 IF d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN RAISE EXCEPTION 'AD3-49: campos u obligaciones Dietas rechazados' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-49: Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
ALTER FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea),vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_personal_propietario,vec_dietas_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_dietas_propietario;

DO $acl$
DECLARE f regprocedure; permitido regrole; x record;
BEGIN
 FOR f,permitido IN SELECT 'vec_autorizacion_atestada_v3.registrar_consumir_consulta_relacion_propia_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_personal_propietario'::regrole UNION ALL SELECT 'vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_borrador_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,'vec_dietas_propietario'::regrole LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner AND a.grantee<>permitido LOOP EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END); END LOOP;
  IF EXISTS (SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND (a.grantee NOT IN (p.proowner,permitido) OR a.privilege_type<>'EXECUTE' OR (a.grantee=permitido AND a.is_grantable))) THEN RAISE EXCEPTION 'AD3-49: ACL de fachada abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
