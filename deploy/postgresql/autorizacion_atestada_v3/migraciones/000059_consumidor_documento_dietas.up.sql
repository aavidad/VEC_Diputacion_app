\set ON_ERROR_STOP on
-- AD3-59: consumo nominal del documento de Dietas (D2-D5), su circuito
-- (D6/D8) y la asignación Personal D7. Se instala una sola vez tras AD3-52;
-- no depende de AD3-53..58 y no modifica migraciones con historia.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000059',0));

DO $núcleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'importacion_organizacion_historica_personal'$x$;
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''documento_dietas_mutacion''\n               AND p_perfil_mutacion IS DISTINCT FROM ''documento_dietas_consulta''\n               AND p_perfil_mutacion IS DISTINCT FROM ''consulta_asignacion_dietas_personal''\n               AND p_perfil_mutacion IS DISTINCT FROM ''correccion_asignacion_dietas_personal''\n               AND p_perfil_mutacion IS DISTINCT FROM ''alta_inicial_asignacion_dietas_personal''\n               AND p_perfil_mutacion IS DISTINCT FROM ''prelectura_dietas''\n               AND p_perfil_mutacion IS DISTINCT FROM ''circuito_dietas''\n               AND p_perfil_mutacion IS DISTINCT FROM ''bandeja_dietas''';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IN (''documento_dietas_mutacion'',''documento_dietas_consulta'',''prelectura_dietas'',''circuito_dietas'',''bandeja_dietas'')\n               AND pg_catalog.pg_has_role(session_user,''vec_dietas_ejecutor'',''MEMBER'')\n               AND EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=session_user::regrole AND m.roleid=''vec_dietas_ejecutor''::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option)\n               AND NOT EXISTS (SELECT 1 FROM pg_auth_members m WHERE m.member=''vec_dietas_ejecutor''::regrole)\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_dietas_ejecutor'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n           OR (\n               p_perfil_mutacion IN (''consulta_asignacion_dietas_personal'',''correccion_asignacion_dietas_personal'',''alta_inicial_asignacion_dietas_personal'')\n               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname=''vec_personal_d7_ejecutor'' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,''MEMBER''))\n               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname=''vec_personal_d7_ejecutor'')\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_personal_d7_ejecutor'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documento_dietas_mutacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.editar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.editar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'editar_borrador_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.borrar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.borrar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'borrar_borrador_propio')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.borrador.propio.enviar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.borrador_propio.enviar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'enviar_borrador_propio')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'documento_dietas_consulta'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.propio.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento_propio.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'comision_borrador'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_documento_propio_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'consulta_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.consultar.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preparar_borrador_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'correccion_asignacion_dietas_personal'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.corregir' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.corregir.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'corregir_asignacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.grupo_corregir' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.grupo_corregir.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'corregir_grupo_dieta')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'alta_inicial_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.registrar_inicial'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.registrar_inicial.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'registrar_asignacion_dietas_inicial')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'prelectura_dietas'
 AND c->>'operacion' IS NOT DISTINCT FROM 'dietas.circuito.preleer'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.circuito.preleer.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'preleer_competencia_circuito_dietas')
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'circuito_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'documento_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.revisar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.revisar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'revisar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.autorizar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.autorizar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'autorizar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.liquidar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.liquidar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'liquidar_documento_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.documento.fiscalizar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.documento.fiscalizar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'fiscalizar_documento_dietas')))
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'bandeja_dietas'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'dietas'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'bandeja_dietas'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.revision.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.revision.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_revision_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.autorizacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.autorizacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_autorizacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.liquidacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.liquidacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_liquidacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'dietas.bandeja.fiscalizacion.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_dietas.bandeja.fiscalizacion.consultar.v1' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_bandeja_fiscalizacion_dietas')))
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_acceso_rutas_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_importacion_organizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_dietas_ejecutor' AND NOT rolcanlogin AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-59: preimagen incompatible' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR strpos(original,'importacion_organizacion_historica_personal')=0 OR strpos(original,'documento_dietas_mutacion')<>0
 THEN RAISE EXCEPTION 'AD3-59: núcleo incompatible' USING ERRCODE='55000'; END IF;
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
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-59: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $núcleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; audiencia text;
 lista text[]:=ARRAY[
  'vec_dietas.borrador_propio.editar.v1','vec_dietas.borrador_propio.borrar.v1',
  'vec_dietas.borrador_propio.enviar.v1','vec_dietas.documento_propio.consultar.v1',
  'vec_personal.asignacion_dietas.consultar.v1','vec_personal.asignacion_dietas.corregir.v1',
  'vec_personal.asignacion_dietas.grupo_corregir.v1',
  'vec_personal.asignacion_dietas.registrar_inicial.v1',
  'vec_dietas.documento.revisar.v1','vec_dietas.documento.autorizar.v1',
  'vec_dietas.circuito.preleer.v1',
  'vec_dietas.documento.liquidar.v1','vec_dietas.documento.fiscalizar.v1',
  'vec_dietas.bandeja.revision.consultar.v1','vec_dietas.bandeja.autorizacion.consultar.v1',
  'vec_dietas.bandeja.liquidacion.consultar.v1','vec_dietas.bandeja.fiscalizacion.consultar.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'vec_dietas_rutas_v1.acceso.v1')=0
 THEN RAISE EXCEPTION 'AD3-59: audiencias previas incompatibles' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH audiencia IN ARRAY lista LOOP
  IF strpos(d,quote_literal(audiencia))<>0 THEN
   RAISE EXCEPTION 'AD3-59: audiencia ya registrada' USING ERRCODE='55000';
  END IF;
  nueva:=nueva||', '||quote_literal(audiencia)||'::text';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; verbo text;
 campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: material Dietas inválido' USING ERRCODE='22023'; END;
 verbo:=split_part(c->>'operacion','.',4);
 IF verbo IS NULL OR verbo NOT IN ('editar','borrar','enviar')
    OR c->>'operacion' IS DISTINCT FROM 'dietas.borrador.propio.'||verbo
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.borrador_propio.'||verbo||'.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'comision_borrador'
    OR d->>'finalidad' IS DISTINCT FROM verbo||'_borrador_propio'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: operación documento Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'documento_dietas_mutacion',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: documento Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;



CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_asignacion_dietas_v3_interna(
 p_variante text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; perfil text; operacion text; audiencia text; finalidad text;
 campos jsonb:='["administrativo_persona_ref","asignacion_ref","auditoria_ad3_ref","centro_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado_local","grupo_dieta","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","unidad_ref","version","vigente_desde"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: asignación Personal inválida' USING ERRCODE='22023'; END;
 CASE p_variante
  WHEN 'consultar' THEN perfil:='consulta_asignacion_dietas_personal'; finalidad:='preparar_borrador_dietas';
  WHEN 'corregir' THEN perfil:='correccion_asignacion_dietas_personal'; finalidad:='corregir_asignacion_dietas';
  WHEN 'grupo_corregir' THEN perfil:='correccion_asignacion_dietas_personal'; finalidad:='corregir_grupo_dieta';
  WHEN 'registrar_inicial' THEN perfil:='alta_inicial_asignacion_dietas_personal'; finalidad:='registrar_asignacion_dietas_inicial';
  ELSE RAISE EXCEPTION 'AD3-59: variante Personal denegada' USING ERRCODE='42501';
 END CASE;
 operacion:='personal.asignacion_dietas.'||p_variante;
 audiencia:='vec_personal.asignacion_dietas.'||p_variante||'.v1';
 IF c->>'operacion' IS DISTINCT FROM operacion
    OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR d->>'accion' IS DISTINCT FROM operacion
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'asignacion_dietas'
    OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: asignación Personal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: Personal requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_asignacion_dietas_v3_interna('consultar',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_asignacion_dietas_v3_interna('corregir',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_grupo_dieta_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_asignacion_dietas_v3_interna('grupo_corregir',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
$f$;
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE sql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
 SELECT * FROM vec_autorizacion_atestada_v3.consumir_asignacion_dietas_v3_interna('registrar_inicial',$1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
$f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_prelectura_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
 campos jsonb:='["comision.administrativo_persona_ref","comision.asignacion_ref","comision.asignacion_version","comision.centro_ref","comision.estado","comision.grupo_dieta","comision.referencia","comision.relacion_ref","comision.responsable_persona_ref","comision.unidad_ref","comision.version","resultado"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: prelectura Dietas inválida' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'dietas.circuito.preleer'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.circuito.preleer.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'documento_dietas'
    OR d->>'finalidad' IS DISTINCT FROM 'preleer_competencia_circuito_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: prelectura Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'prelectura_dietas',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: prelectura Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_circuito_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; verbo text;
 campos jsonb:='["comision.estado","comision.referencia","comision.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: circuito Dietas inválido' USING ERRCODE='22023'; END;
 verbo:=split_part(c->>'operacion','.',3);
 IF verbo IS NULL OR verbo NOT IN ('revisar','autorizar','liquidar','fiscalizar')
    OR c->>'operacion' IS DISTINCT FROM 'dietas.documento.'||verbo
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.documento.'||verbo||'.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'documento_dietas'
    OR d->>'finalidad' IS DISTINCT FROM verbo||'_documento_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: etapa Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'circuito_dietas',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: circuito Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_bandeja_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; etapa text;
 campos jsonb:='["items.estado","items.fecha_fin","items.fecha_inicio","items.referencia","items.version","siguiente_cursor"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: bandeja Dietas inválida' USING ERRCODE='22023'; END;
 etapa:=split_part(c->>'operacion','.',3);
 IF etapa IS NULL OR etapa NOT IN ('revision','autorizacion','liquidacion','fiscalizacion')
    OR c->>'operacion' IS DISTINCT FROM 'dietas.bandeja.'||etapa||'.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.bandeja.'||etapa||'.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'bandeja_dietas'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_bandeja_'||etapa||'_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: bandeja Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'bandeja_dietas',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: bandeja Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_dietas_documento_consulta_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
 campos jsonb:='["comision.calculo","comision.codigos_ruta","comision.documento","comision.estado","comision.fecha_apertura","comision.fecha_fin","comision.fecha_inicio","comision.motivo","comision.numero_documento","comision.referencia","comision.relacion_ref","comision.rutas","comision.vehiculo_propio","comision.version","items.comision.calculo","items.comision.codigos_ruta","items.comision.documento","items.comision.estado","items.comision.fecha_apertura","items.comision.fecha_fin","items.comision.fecha_inicio","items.comision.motivo","items.comision.numero_documento","items.comision.referencia","items.comision.relacion_ref","items.comision.rutas","items.comision.vehiculo_propio","items.comision.version","items.recibo.referencia","items.recibo.registrado_en","items.recibo.repeticion","items.recibo.version","recibo.referencia","recibo.registrado_en","recibo.repeticion","recibo.version","siguiente_cursor"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-59: consulta Dietas inválida' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'dietas.documento.propio.consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_dietas.documento_propio.consultar.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'dietas'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'comision_borrador'
    OR d->>'finalidad' IS DISTINCT FROM 'consultar_documento_propio_dietas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-59: consulta documento Dietas denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'documento_dietas_consulta',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-59: consulta Dietas requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

-- La fachada auxiliar no se concede a otros esquemas. Las fachadas públicas
-- tienen un único consumidor propietario; el login terminal se coteja en núcleo.
DO $acl$
DECLARE nombre text; f regprocedure; permitido oid; x record;
 nombres text[]:=ARRAY[
  'consumir_asignacion_dietas_v3_interna',
  'registrar_y_consumir_consulta_asignacion_dietas_v3_atestada',
  'registrar_y_consumir_correccion_asignacion_dietas_v3_atestada',
  'registrar_y_consumir_correccion_grupo_dieta_v3_atestada',
  'registrar_y_consumir_alta_inicial_asignacion_dietas_v3_atestada',
  'registrar_y_consumir_dietas_documento_v3_atestada',
  'registrar_y_consumir_dietas_documento_consulta_v3_atestada',
  'registrar_y_consumir_dietas_prelectura_v3_atestada',
  'registrar_y_consumir_dietas_circuito_v3_atestada',
  'registrar_y_consumir_dietas_bandeja_v3_atestada'];
 firma text;
BEGIN
 FOREACH nombre IN ARRAY nombres LOOP
  firma:=CASE WHEN nombre='consumir_asignacion_dietas_v3_interna'
    THEN '(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'
    ELSE '(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)' END;
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre||firma);
  IF f IS NULL THEN RAISE EXCEPTION 'AD3-59: fachada ausente %',nombre USING ERRCODE='55000'; END IF;
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
  permitido:=CASE WHEN nombre='consumir_asignacion_dietas_v3_interna' THEN NULL
    WHEN nombre LIKE '%asignacion_dietas%' OR nombre='registrar_y_consumir_correccion_grupo_dieta_v3_atestada'
      THEN 'vec_personal_propietario'::regrole::oid
    ELSE 'vec_dietas_propietario'::regrole::oid END;
  IF permitido IS NOT NULL THEN
   EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO %I',f::text,pg_get_userbyid(permitido));
  END IF;
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
  THEN RAISE EXCEPTION 'AD3-59: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
  FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
   FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
   IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
     OR (x.grantee=permitido AND x.is_grantable)
   THEN RAISE EXCEPTION 'AD3-59: ACL de fachada abierta %',nombre USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
