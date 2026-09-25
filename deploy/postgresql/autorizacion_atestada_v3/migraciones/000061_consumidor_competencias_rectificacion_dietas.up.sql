\set ON_ERROR_STOP on
-- AD3-61: consumo nominal de las dos lecturas/acciones Personal que Dietas
-- necesita después de D7: competencias actuales del actor (D7b, Personal
-- 000014) y solicitud, consulta, resolución y lista competente de
-- rectificaciones (D7c, Personal 000015). Crea las dos fachadas que esas
-- migraciones exigen en su precondición y que hasta ahora solo existían como
-- stubs TEST-ONLY en deploy/postgresql/personal/pruebas_sql/.
--
-- Orden de instalación, una migración tras otra y confirmada cada una:
--   AD3-52 → AD3-53 (Cronos) y AD3-59 (Dietas/D7), en serie y en cualquier
--   orden entre sí → Personal 000012 y 000013 → AD3-61 → Personal 000014 y
--   000015.
-- AD3-61 exige AD3-59 ya instalada (su núcleo y sus fachadas D7) y el rol de
-- grupo vec_personal_d7_ejecutor que crea Personal 000012. No depende de
-- AD3-53: su reescritura del núcleo usa las mismas anclas únicas y conserva
-- las de AD3-53, de modo que esta puede instalarse antes o después. Toma el
-- consultivo común vec_autorizacion_atestada_v3:nucleo antes de leer el
-- núcleo, igual que AD3-53 y AD3-59. Se instala una sola vez; no modifica
-- migraciones con historia y no tiene DOWN destructivo.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000061',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));

DO $núcleo$
DECLARE
 f oid:='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[];
 propietario oid; config text[]; definidora boolean;
 marca text:=E'       )\n       OR c ->> ''suite'' <> ''VEC-AD-3-COSE-EDDSA-1''';
 excl text:=$x$               AND p_perfil_mutacion IS DISTINCT FROM 'importacion_organizacion_historica_personal'$x$;
 excl_nuevo text:=excl||E'\n               AND p_perfil_mutacion IS DISTINCT FROM ''competencias_asignacion_dietas_personal''\n               AND p_perfil_mutacion IS DISTINCT FROM ''rectificacion_dietas_personal''';
 guarda text:=E'           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 -- Un único grupo exacto, cotejado por nombre en pg_roles: nunca un literal
 -- ::regrole que falle al compilar si el rol no existiera.
 guarda_nueva text:=E'           )\n           OR (\n               p_perfil_mutacion IN (''competencias_asignacion_dietas_personal'',''rectificacion_dietas_personal'')\n               AND EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.roleid=g.oid WHERE g.rolname=''vec_personal_d7_ejecutor'' AND m.member=session_user::regrole AND NOT m.admin_option AND m.inherit_option AND NOT m.set_option AND pg_catalog.pg_has_role(session_user,g.oid,''MEMBER''))\n               AND NOT EXISTS (SELECT 1 FROM pg_roles g JOIN pg_auth_members m ON m.member=g.oid WHERE g.rolname=''vec_personal_d7_ejecutor'')\n               AND (SELECT count(*) FROM pg_auth_members m WHERE m.member=session_user::regrole)=1\n               AND NOT EXISTS (SELECT 1 FROM pg_roles r WHERE left(r.rolname,4)=''vec_'' AND r.rolname<>session_user AND r.rolname<>''vec_personal_d7_ejecutor'' AND pg_catalog.pg_has_role(session_user,r.oid,''MEMBER''))\n           )\n       ) THEN\n        RAISE EXCEPTION USING\n            ERRCODE = ''42501'',';
 extension text:=$x$           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'competencias_asignacion_dietas_personal'
 AND c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.competencias_consultar'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.competencias.v1'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'asignacion_dietas_competencias'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'tramitar_dietas_asignadas'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb)
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'rectificacion_dietas_personal'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'personal'
 AND d->>'accion' IS NOT DISTINCT FROM c->>'operacion'
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.solicitar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.solicitar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'solicitar_rectificacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.propia.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_rectificacion_dietas_propia')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.resolver' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.resolver.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificacion_asignacion_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'resolver_rectificacion_dietas')
   OR (c->>'operacion' IS NOT DISTINCT FROM 'personal.asignacion_dietas.rectificacion.competente.consultar' AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1' AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'rectificaciones_competentes_dietas' AND d->>'finalidad' IS NOT DISTINCT FROM 'consultar_rectificaciones_dietas_competentes')))
$x$;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_correccion_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_d7_ejecutor' AND NOT rolcanlogin AND NOT rolsuper AND NOT rolbypassrls)
    OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='vec_personal_propietario' AND NOT rolcanlogin AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'AD3-61: preimagen incompatible (requiere AD3-59 y Personal 000012)' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
 INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
 INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF propietario IS DISTINCT FROM (SELECT oid FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_propietario') OR NOT definidora
    OR config IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
    OR length(original)-length(replace(original,marca,''))<>length(marca)
    OR length(original)-length(replace(original,excl,''))<>length(excl)
    OR length(original)-length(replace(original,guarda,''))<>length(guarda)
    OR strpos(original,'importacion_organizacion_historica_personal')=0
    OR strpos(original,'consulta_asignacion_dietas_personal')=0
    OR strpos(original,'competencias_asignacion_dietas_personal')<>0
    OR strpos(original,'rectificacion_dietas_personal')<>0
 THEN RAISE EXCEPTION 'AD3-61: núcleo incompatible (requiere AD3-59)' USING ERRCODE='55000'; END IF;
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
 THEN RAISE EXCEPTION 'AD3-61: núcleo alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $núcleo$;

LOCK TABLE vec_autorizacion_atestada_v3.clave_capacidad_version IN ACCESS EXCLUSIVE MODE;
DO $audiencias$
DECLARE d text; nueva text; audiencia text;
 lista text[]:=ARRAY[
  'vec_personal.asignacion_dietas.competencias.v1',
  'vec_personal.asignacion_dietas.rectificacion.solicitar.v1',
  'vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1',
  'vec_personal.asignacion_dietas.rectificacion.resolver.v1',
  'vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1'];
BEGIN
 SELECT regexp_replace(pg_get_constraintdef(c.oid,true),'\s+',' ','g') INTO STRICT d
 FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'CHECK (audiencia_consumo = ANY (ARRAY[')<>1 OR right(d,3)<>']))'
    OR strpos(d,'''vec_personal.asignacion_dietas.consultar.v1''')=0
 THEN RAISE EXCEPTION 'AD3-61: audiencias previas incompatibles (requiere AD3-59)' USING ERRCODE='55000'; END IF;
 nueva:=left(d,length(d)-3);
 FOREACH audiencia IN ARRAY lista LOOP
  IF strpos(d,quote_literal(audiencia))<>0 THEN
   RAISE EXCEPTION 'AD3-61: audiencia ya registrada' USING ERRCODE='55000';
  END IF;
  nueva:=nueva||', '||quote_literal(audiencia)||'::text';
 END LOOP;
 ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  DROP CONSTRAINT clave_capacidad_version_audiencia_consumo_check;
 EXECUTE 'ALTER TABLE vec_autorizacion_atestada_v3.clave_capacidad_version ADD CONSTRAINT clave_capacidad_version_audiencia_consumo_check '||nueva||']))';
END $audiencias$;

-- D7b: el recurso es la persona del actor y los campos, los que devuelve
-- Personal 000014. Cada consulta consume una decisión nueva.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_competencias_asignacion_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record;
 campos jsonb:='["asignacion_ref","auditoria_ad3_ref","cardinalidad","competencias","consultada_en","consumo_huella_sha256","decision_ref","efecto_ref","recibo_ref","relacion_ref","rol","unidad_ref","version","vigente_desde"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-61: competencias Personal inválidas' USING ERRCODE='22023'; END;
 IF c->>'operacion' IS DISTINCT FROM 'personal.asignacion_dietas.competencias_consultar'
    OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_personal.asignacion_dietas.competencias.v1'
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'asignacion_dietas_competencias'
    OR d->>'finalidad' IS DISTINCT FROM 'tramitar_dietas_asignadas'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM campos
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-61: competencias Personal denegadas' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'competencias_asignacion_dietas_personal',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-61: competencias requieren consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

-- D7c: una sola fachada para las cuatro acciones de Personal 000015. La
-- operación de la capacidad fija audiencia, recurso, finalidad y campos; la
-- lista competente devuelve más campos que las tres acciones sobre una
-- solicitud concreta.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_consumir_rectificacion_dietas_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; audiencia text; tipo text; finalidad text; esperados jsonb;
 campos_solicitud jsonb:='["asignacion_ref","auditoria_ad3_ref","consumo_huella_sha256","decision_ref","efecto_ref","estado","recibo_ref","registrada_en","solicitud_ref","version_origen"]'::jsonb;
 campos_competentes jsonb:='["administrativo_persona_ref","asignacion_actual","asignacion_ref","auditoria_ad3_ref","campos_a_revisar","cardinalidad","centro_ref","consultada_en","consumo_huella_sha256","decision_ref","detalle_solicitado","efecto_ref","empleado_ref","estado","fecha_referencia","grupo_dieta","motivo_revision","persona_ref","recibo_ref","registrada_en","relacion_ref","responsable_persona_ref","solicitud_ref","solicitudes","unidad_ref","version","version_origen","vigente_desde"]'::jsonb;
BEGIN
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-61: rectificación Personal inválida' USING ERRCODE='22023'; END;
 CASE c->>'operacion'
  WHEN 'personal.asignacion_dietas.rectificacion.solicitar' THEN
   audiencia:='vec_personal.asignacion_dietas.rectificacion.solicitar.v1';
   tipo:='rectificacion_asignacion_dietas'; finalidad:='solicitar_rectificacion_dietas'; esperados:=campos_solicitud;
  WHEN 'personal.asignacion_dietas.rectificacion.propia.consultar' THEN
   audiencia:='vec_personal.asignacion_dietas.rectificacion.propia.consultar.v1';
   tipo:='rectificacion_asignacion_dietas'; finalidad:='consultar_rectificacion_dietas_propia'; esperados:=campos_solicitud;
  WHEN 'personal.asignacion_dietas.rectificacion.resolver' THEN
   audiencia:='vec_personal.asignacion_dietas.rectificacion.resolver.v1';
   tipo:='rectificacion_asignacion_dietas'; finalidad:='resolver_rectificacion_dietas'; esperados:=campos_solicitud;
  WHEN 'personal.asignacion_dietas.rectificacion.competente.consultar' THEN
   audiencia:='vec_personal.asignacion_dietas.rectificacion.competente.consultar.v1';
   tipo:='rectificaciones_competentes_dietas'; finalidad:='consultar_rectificaciones_dietas_competentes'; esperados:=campos_competentes;
  ELSE RAISE EXCEPTION 'AD3-61: operación de rectificación denegada' USING ERRCODE='42501';
 END CASE;
 IF c->>'audiencia_consumo' IS DISTINCT FROM audiencia
    OR d->>'accion' IS DISTINCT FROM c->>'operacion'
    OR d->>'modulo_id' IS DISTINCT FROM 'personal'
    OR d->>'tipo_recurso' IS DISTINCT FROM tipo
    OR d->>'finalidad' IS DISTINCT FROM finalidad
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'campos_permitidos' IS DISTINCT FROM esperados
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'AD3-61: rectificación Personal denegada' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'rectificacion_dietas_personal',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE THEN RAISE EXCEPTION 'AD3-61: rectificación requiere consumo nuevo' USING ERRCODE='42501'; END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,true;
END $f$;

-- Un único consumidor por fachada: el propietario de Personal. El login
-- terminal (grupo vec_personal_d7_ejecutor exacto) se coteja en el núcleo.
DO $acl$
DECLARE nombre text; f regprocedure; permitido oid; dueño oid; x record;
 nombres text[]:=ARRAY[
  'registrar_y_consumir_competencias_asignacion_dietas_v3_atestada',
  'registrar_y_consumir_rectificacion_dietas_v3_atestada'];
BEGIN
 SELECT oid INTO STRICT permitido FROM pg_roles WHERE rolname='vec_personal_propietario';
 SELECT oid INTO STRICT dueño FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_propietario';
 FOREACH nombre IN ARRAY nombres LOOP
  f:=to_regprocedure('vec_autorizacion_atestada_v3.'||nombre||'(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
  IF f IS NULL THEN RAISE EXCEPTION 'AD3-61: fachada ausente %',nombre USING ERRCODE='55000'; END IF;
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC',f::text);
  EXECUTE format('GRANT EXECUTE ON FUNCTION %s TO %I',f::text,pg_get_userbyid(permitido));
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM dueño
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
  THEN RAISE EXCEPTION 'AD3-61: propietario o entorno de fachada incompatible' USING ERRCODE='55000'; END IF;
  FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner
   FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
   IF (x.grantee<>x.proowner AND x.grantee IS DISTINCT FROM permitido) OR x.privilege_type<>'EXECUTE'
     OR (x.grantee=permitido AND x.is_grantable)
   THEN RAISE EXCEPTION 'AD3-61: ACL de fachada abierta %',nombre USING ERRCODE='55000'; END IF;
  END LOOP;
 END LOOP;
END $acl$;
COMMIT;
