\set ON_ERROR_STOP on
-- Personal B2: cuatro operaciones nominales.
-- Numeración: 000054 es un hueco reservado para B2; el número no fija el
-- orden. Requiere AD3-59 (ancla bandeja_dietas y su fachada) y se instala
-- después de ella, en serie con 61/70/80 donde existan: toma el consultivo
-- común vec_autorizacion_atestada_v3:nucleo antes de leer la preimagen.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000054',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $nucleo$
DECLARE f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'bandeja_dietas'$x$;
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''registro_empleado_b2''';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IS NOT DISTINCT FROM ''registro_empleado_b2''\n               AND pg_catalog.pg_has_role(session_user,''vec_personal_ejecutor'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_personal_ejecutor''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_personal_ejecutor''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_personal_ejecutor'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'registro_empleado_b2'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.alta.registrar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.alta.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'alta_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.hecho.registrar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.hecho.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'hecho_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_hecho_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.ficha.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.ficha.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'registro_empleado_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_ficha_empleado'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["corte","eficacia_administrativa","empleado_ref","evidencia","firma_oficial","ocupaciones","organismo_ref","persona_ref","relaciones","servicios","situaciones","version"]'::jsonb)
 OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.registro_empleado.vacantes.consultar'
   AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.registro_empleado.vacantes.v1'
   AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'vacantes_rrhh'
   AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_vacantes'
   AND d->'campos_permitidos' IS NOT DISTINCT FROM '["cobertura","corte","cursor","cursor_siguiente","evidencia","limite","organismo_ref","vacantes"]'::jsonb)))
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-54: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) INTO deps
 FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR strpos(original,'registro_empleado_b2')<>0
 THEN RAISE EXCEPTION 'AD3-54: núcleo incompatible' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,excl,excl_nuevo);
 nuevo:=replace(nuevo,guarda,guarda_nueva);
 nuevo:=replace(nuevo,marca,extension||marca);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(replace(actual,extension||marca,marca),guarda_nueva,guarda),excl_nuevo,excl) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
        FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-54: núcleo alterado fuera de contrato' USING ERRCODE='55000'; END IF;
END $nucleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; audiencia text;
 lista text[]:=ARRAY[
  'vec_personal.registro_empleado.alta.v1',
  'vec_personal.registro_empleado.hecho.v1',
  'vec_personal.registro_empleado.ficha.v1',
  'vec_personal.registro_empleado.vacantes.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_dietas.bandeja.fiscalizacion.consultar.v1')=0
 THEN RAISE EXCEPTION 'AD3-54: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH audiencia IN ARRAY lista LOOP
  IF strpos(d,quote_literal(audiencia))<>0 THEN
   RAISE EXCEPTION 'AD3-54: audiencia ya registrada' USING ERRCODE='55000'; END IF;
  nueva:=nueva||', '||quote_literal(audiencia)||'::text';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
   DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text:='registro_empleado_b2';
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-54: material inválido' USING ERRCODE='22023'; END;
 IF d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR NOT ((c->>'operacion'='personal.registro_empleado.alta.registrar'
       AND c->>'audiencia_consumo'='vec_personal.registro_empleado.alta.v1'
       AND d->>'tipo_recurso'='alta_empleado_rrhh'
       AND d->>'finalidad'='registrar_empleado'
       AND d->'campos_permitidos'='["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","persona_ref","proyeccion_ref","recibo","relacion_ref","version"]'::jsonb)
      OR (c->>'operacion'='personal.registro_empleado.hecho.registrar'
       AND c->>'audiencia_consumo'='vec_personal.registro_empleado.hecho.v1'
       AND d->>'tipo_recurso'='hecho_empleado_rrhh'
       AND d->>'finalidad'='registrar_hecho_empleado'
       AND d->'campos_permitidos'='["eficacia_administrativa","empleado_ref","evidencia","firma_oficial","hecho_ref","recibo","relacion_ref","tipo","version"]'::jsonb)
      OR (c->>'operacion'='personal.registro_empleado.ficha.consultar'
       AND c->>'audiencia_consumo'='vec_personal.registro_empleado.ficha.v1'
       AND d->>'tipo_recurso'='registro_empleado_rrhh'
       AND d->>'finalidad'='consultar_ficha_empleado'
       AND d->'campos_permitidos'='["corte","eficacia_administrativa","empleado_ref","evidencia","firma_oficial","ocupaciones","organismo_ref","persona_ref","relaciones","servicios","situaciones","version"]'::jsonb)
      OR (c->>'operacion'='personal.registro_empleado.vacantes.consultar'
       AND c->>'audiencia_consumo'='vec_personal.registro_empleado.vacantes.v1'
       AND d->>'tipo_recurso'='vacantes_rrhh'
       AND d->>'finalidad'='consultar_vacantes'
       AND d->'campos_permitidos'='["cobertura","corte","cursor","cursor_siguiente","evidencia","limite","organismo_ref","vacantes"]'::jsonb))
 THEN RAISE EXCEPTION 'AD3-54: operación denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-54: requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_registro_empleado_b2_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_personal_propietario;
COMMIT;
