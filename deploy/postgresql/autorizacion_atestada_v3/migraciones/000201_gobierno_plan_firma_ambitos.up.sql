\set ON_ERROR_STOP on
-- AD201: el recurso de gobierno del plan nominal de firma lleva los ámbitos de
-- la asignación del administrador (aprobado por dirección el 5/10/2026). AD177
-- calculaba la huella de contexto con "ambitos":{} y el PDP común exige que el
-- recurso tenga las dimensiones de la asignación (siempre al menos una), así
-- que toda decisión se denegaba con ambito_no_autorizado. La fachada nueva
-- registrar_y_confirmar_gobierno_plan_firma_v2 recibe organizacion_ref y
-- unidad_ref, calcula con ellos la huella, coteja organización y unidad con la
-- asignación actual (AUT52) y confirma con CC9, que calcula la misma huella. La
-- v1 deja de ser ejecutable por el grupo dedicado. El material del kit (13
-- claves), el núcleo y el CHECK de audiencias no cambian. Requiere AD200, AUT52
-- y CC9. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000201',0));
DO $pre$
DECLARE v1 oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD201: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF v1 IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2(bytea,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.login_gobierno_plan_firma_valido_v1()') IS NULL
 OR to_regrole('vec_plan_firma_gobierno_ejecutor') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)') IS NULL
 OR to_regprocedure('vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(bytea,jsonb)') IS NULL
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(text,text,text,text,text)','EXECUTE') IS NOT TRUE
 OR has_function_privilege('vec_autorizacion_atestada_v3_propietario','vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(bytea,text,text,jsonb)','EXECUTE') IS NOT TRUE
 -- La v1 tiene el ACL que deja AD200: propietario y grupo dedicado.
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=v1)<>2
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=v1 AND a.grantee IN(p.proowner,'vec_plan_firma_gobierno_ejecutor'::regrole)
   AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)<>2
 THEN RAISE EXCEPTION 'AD201: PARO clave=preimagen actual=incompatible esperado=AD177_AD200_AUT52_CC9_sin_AD201' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;

CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2(
 p_material_exacto bytea,p_organizacion_ref text,p_unidad_ref text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='15s' SET TimeZone='UTC'
AS $f$
DECLARE m jsonb; original json; c jsonb; d jsonb; catalogo jsonb;
 accion text; estado text; revision text; recurso text; material_sha text; contexto_sha text;
 consumo record; resultado record; comprobado jsonb;
BEGIN
 IF p_material_exacto IS NULL OR octet_length(p_material_exacto) NOT BETWEEN 2 AND 4194304
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 32768
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
  -- Ámbitos del recurso: los mismos formatos que admiten AUT y el dominio, sin
  -- caracteres que alteren la representación JSON de la huella.
  OR (p_organizacion_ref ~ '^org_[a-z0-9]{16,80}$') IS NOT TRUE
  OR (p_unidad_ref ~ '^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD201 material de gobierno inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  original:=convert_from(p_material_exacto,'UTF8')::json; m:=original::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD201 material de gobierno inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(m) IS DISTINCT FROM 'object' OR (SELECT count(*) FROM json_each(original))<>13
  OR (SELECT count(*) FROM jsonb_object_keys(m))<>13
  OR (m ?& ARRAY['esquema','operacion','catalogo_id','version','revision_esperada','huella_esperada',
    'clave_operacion','catalogo_canonico_base64','catalogo_sha256','traza_canonica_base64','traza_sha256',
    'evento_canonico_base64','evento_sha256']) IS NOT TRUE
  OR m->>'esquema' IS DISTINCT FROM 'vec.catalogos.plan-firma.gobierno.v1'
  OR jsonb_typeof(m->'operacion') IS DISTINCT FROM 'string'
  OR (m->>'operacion' IN('crear','actualizar','publicar','retirar')) IS NOT TRUE
  OR jsonb_typeof(m->'catalogo_id') IS DISTINCT FROM 'string'
  OR (m->>'catalogo_id' ~ '^[a-z][a-z0-9._-]{2,127}$') IS NOT TRUE
  OR jsonb_typeof(m->'version') IS DISTINCT FROM 'number'
  OR (m->>'version' ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
  OR (m->>'catalogo_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION 'AD201 material de gobierno inválido' USING ERRCODE='22023'; END IF;
 BEGIN catalogo:=convert_from(decode(m->>'catalogo_canonico_base64','base64'),'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD201 catálogo de gobierno inválido' USING ERRCODE='22023';
 END;
 accion:='vec.catalogos.'||(m->>'operacion');
 estado:=catalogo->>'estado'; revision:=catalogo->>'revision';
 recurso:=(m->>'catalogo_id')||':'||(m->>'version');
 material_sha:=encode(sha256(p_material_exacto),'hex');
 IF jsonb_typeof(catalogo) IS DISTINCT FROM 'object'
  OR catalogo->>'id' IS DISTINCT FROM m->>'catalogo_id' OR catalogo->'version' IS DISTINCT FROM m->'version'
  OR catalogo->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR jsonb_typeof(catalogo->'revision') IS DISTINCT FROM 'number'
  OR (revision ~ '^[1-9][0-9]{0,9}$') IS NOT TRUE
  OR estado IS DISTINCT FROM (CASE m->>'operacion' WHEN 'crear' THEN 'borrador'
    WHEN 'actualizar' THEN 'borrador' WHEN 'publicar' THEN 'publicado' WHEN 'retirar' THEN 'retirado' END)
  OR encode(sha256(decode(m->>'catalogo_canonico_base64','base64')),'hex') IS DISTINCT FROM m->>'catalogo_sha256' THEN
  RAISE EXCEPTION 'AD201 catálogo de gobierno divergente' USING ERRCODE='22023'; END IF;
 -- Claves ordenadas como en la huella canónica de RecursoAutorizable (Go).
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||p_organizacion_ref||
  '","unidad_ref":"'||p_unidad_ref||'"},"atributos":{"estado":"'||estado||
  '","material_sha256":"'||material_sha||'","revision":"'||revision||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM accion OR c->>'operacion' IS DISTINCT FROM accion
  OR d->>'recurso_ref' IS DISTINCT FROM recurso OR c->>'efecto_ref' IS DISTINCT FROM recurso
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
  OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_configurable'
  OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'
  OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD201 gobierno de plan denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'gobierno_plan_nominal_firma_ct',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 comprobado:=vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(to_jsonb(consumo));
 -- La decisión consumida es la recibida (el núcleo la cotejó con su atestación)
 -- y la organización y la unidad del recurso siguen en la asignación actual,
 -- activa y vigente del actor (AUT52).
 IF d->>'decision_ref' IS DISTINCT FROM comprobado->>'decision_ref'
  OR d->>'principal_id' IS DISTINCT FROM comprobado->>'registrador_principal_ref'
  OR vec_autorizacion.acreditar_ambitos_gobierno_plan_firma_v1(
   d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',p_organizacion_ref,p_unidad_ref) IS NOT TRUE THEN
  RAISE EXCEPTION 'AD201 ámbito de gobierno no acreditado' USING ERRCODE='42501'; END IF;
 IF (comprobado->>'decision_valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'AD201 gobierno de plan caducado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT resultado FROM vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v2(
  p_material_exacto,p_organizacion_ref,p_unidad_ref,jsonb_build_object('consumo',to_jsonb(consumo),'actor_ref',comprobado->>'registrador_principal_ref',
   'perfil_ref',comprobado->>'registrador_perfil_ref','accion',comprobado->>'operacion',
   'finalidad',comprobado->>'finalidad','proceso',comprobado->>'proceso','canal',comprobado->>'canal'));
 IF (comprobado->>'decision_valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'AD201 gobierno de plan caducado' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recibo',to_jsonb(resultado),'consumo',to_jsonb(consumo));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2(bytea,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2(bytea,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_plan_firma_gobierno_ejecutor;
-- La v1 sin ámbitos queda sólo para su propietario: nunca podía autorizarse.
REVOKE EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM vec_plan_firma_gobierno_ejecutor;
RESET ROLE;

DO $post$
DECLARE v1 oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 v2 oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v2(bytea,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF v2 IS NULL
 OR NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=v2 AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole AND p.prosecdef)
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=v2)<>2
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=v2 AND a.grantee IN(p.proowner,'vec_plan_firma_gobierno_ejecutor'::regrole)
   AND a.grantor=p.proowner AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)<>2
 OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=v1)<>1
 OR has_function_privilege('vec_plan_firma_gobierno_ejecutor',v1,'EXECUTE')
 OR has_function_privilege('vec_contratacion_temporal_ejecutor',v2,'EXECUTE')
 THEN RAISE EXCEPTION 'AD201: PARO clave=acl_post esperado=v2_grupo_v1_solo_propietario actual=divergente' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
