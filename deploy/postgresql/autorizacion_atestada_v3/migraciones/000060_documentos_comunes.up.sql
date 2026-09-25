\set ON_ERROR_STOP on
-- B5: consumidor nominal del servicio común de documentos; sin firma ni entrega.
-- Incluye el registro de documentos con custodia externa (referencia y huella,
-- sin contenido). Nunca instalada: se amplió en su sitio el 25/09/2026.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000060',0));
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=E'               AND p_perfil_mutacion IS DISTINCT FROM ''importacion_organizacion_historica_personal''';
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''operacion_documentos_comunes''';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IS NOT DISTINCT FROM ''operacion_documentos_comunes''\n               AND pg_catalog.pg_has_role(session_user,''vec_documentos_ejecutor'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_documentos_ejecutor''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_documentos_ejecutor''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_documentos_ejecutor'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND (
   (d->>'accion' IS NOT DISTINCT FROM 'documentos.generado.alta'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_generado'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'alta_documento_generado'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.expediente.listar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'expediente_documental'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'listar_documentos_expediente'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["items","siguiente_cursor"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.original.descargar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_original'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'descargar_documento_original'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["contenido","documento"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.notificacion.preparar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'notificacion_preparada'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_notificacion'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["preparacion","recibo"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.externo.registrar'
    AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_externo'
    AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_documento_externo'
    AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)
 ))
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_documentos_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-60: precondición incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR strpos(original,'importacion_organizacion_historica_personal')=0 OR strpos(original,'operacion_documentos_comunes')<>0
 THEN RAISE EXCEPTION 'AD3-60: núcleo AD3-52 incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo); nuevo:=replace(nuevo,guarda,guarda_nueva); nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),guarda_nueva,guarda),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-60: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;
LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text;
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_personal.organizacion_historica.importar.v1')=0
    OR strpos(d,'vec_documentos.operacion.v1')<>0
 THEN RAISE EXCEPTION 'AD3-60: audiencia post-AD51 incompatible' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3)||', ''vec_documentos.operacion.v1''::text]))';
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva;
END $audiencias$;
CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; a text;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-60: material inválido' USING ERRCODE='22023'; END;
 a:=d->>'accion';
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.operacion.v1'
    OR c->>'operacion' IS DISTINCT FROM a OR d->>'modulo_id' IS DISTINCT FROM 'documentos'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR NOT ((a='documentos.generado.alta' AND d->>'tipo_recurso'='documento_generado'
       AND d->>'finalidad'='alta_documento_generado' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (a='documentos.expediente.listar' AND d->>'tipo_recurso'='expediente_documental'
       AND d->>'finalidad'='listar_documentos_expediente' AND d->'campos_permitidos'='["items","siguiente_cursor"]'::jsonb)
     OR (a='documentos.original.descargar' AND d->>'tipo_recurso'='documento_original'
       AND d->>'finalidad'='descargar_documento_original' AND d->'campos_permitidos'='["contenido","documento"]'::jsonb)
     OR (a='documentos.notificacion.preparar' AND d->>'tipo_recurso'='notificacion_preparada'
       AND d->>'finalidad'='preparar_notificacion' AND d->'campos_permitidos'='["preparacion","recibo"]'::jsonb)
     OR (a='documentos.externo.registrar' AND d->>'tipo_recurso'='documento_externo'
       AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb))
 THEN RAISE EXCEPTION 'AD3-60: operación documental denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'operacion_documentos_comunes',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-60: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
ALTER FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) OWNER TO vec_autorizacion_atestada_v3_propietario;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_documentos_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_propietario;
COMMIT;
