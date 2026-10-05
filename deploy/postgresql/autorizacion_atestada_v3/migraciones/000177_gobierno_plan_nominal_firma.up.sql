\set ON_ERROR_STOP on
-- AD177: fachadas de gobierno del plan nominal de firma CT y consumo ligado
-- del plan para CT176. Requiere AD193 y AD178 instaladas; preimágenes medidas
-- el 05/10/2026 tras AD178 sobre main posterior a AD195/AD196.
-- No modifica AD167/AD172–176 ni publica concesiones o perfiles por petición.
-- Una sola vez; sin DOWN.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000177',0));
-- Mismo bloqueo que toman las migraciones que reconstruyen el núcleo.
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $pre$
DECLARE nucleo regprocedure;
 -- Núcleo y CHECK de audiencias (pg_get_constraintdef(...,true)) después de AD178.
 esperado_nucleo_sha256 text := '2ccd704afe6140d604faa626631e9743edda8f785517d1136054c746cb9b1381';
 esperado_audiencias_sha256 text := '2e687cbf9055a1c1a94035a74bdbf80fcf37c91b0613a7a5325f5d55def56e64';
 observado_nucleo_sha256 text;
 observado_audiencias_sha256 text;
 columnas_sello_xid8 bigint;
BEGIN
 nucleo:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
  OR getdatabaseencoding()<>'UTF8' OR nucleo IS NULL
  OR to_regrole('vec_catalogos_configurables_propietario') IS NULL
  OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
  OR to_regprocedure('vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(text,text,text,text,text,text,text,text,jsonb,jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(jsonb)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)') IS NOT NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb)') IS NOT NULL THEN
  RAISE EXCEPTION 'AD177 clave=objetos observado=incompatible esperado=preimagen_no_instalada' USING ERRCODE='55000';
 END IF;
 SELECT count(*) INTO columnas_sello_xid8 FROM pg_attribute a
 WHERE a.attrelid IN (to_regclass('vec_autorizacion_atestada_v3.consumo_decision_v3'),
                     to_regclass('vec_autorizacion_atestada_v3.auditoria_consumo_v3'))
  AND a.attname='transaccion_origen' AND a.attnum>0 AND NOT a.attisdropped
  AND a.atttypid='xid8'::regtype;
 IF columnas_sello_xid8<>2 THEN
  RAISE EXCEPTION 'AD177 clave=columnas_sello_xid8 observado=% esperado=2',columnas_sello_xid8 USING ERRCODE='55000';
 END IF;
 observado_nucleo_sha256:=encode(sha256(convert_to(pg_get_functiondef(nucleo),'UTF8')),'hex');
 SELECT encode(sha256(convert_to(pg_get_constraintdef(c.oid,true),'UTF8')),'hex')
 INTO observado_audiencias_sha256 FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
  AND c.conname='clave_capacidad_version_audiencia_consumo_check';
 IF esperado_nucleo_sha256 IS NULL OR esperado_audiencias_sha256 IS NULL THEN
  RAISE EXCEPTION 'AD177 clave=preimagen_aprobada observado=pendiente esperado=AD178_medida' USING ERRCODE='55000';
 END IF;
 IF observado_nucleo_sha256 IS DISTINCT FROM esperado_nucleo_sha256 THEN
  RAISE EXCEPTION 'AD177 clave=nucleo_sha256 observado=% esperado=%',observado_nucleo_sha256,esperado_nucleo_sha256 USING ERRCODE='55000';
 END IF;
 IF observado_audiencias_sha256 IS DISTINCT FROM esperado_audiencias_sha256 THEN
  RAISE EXCEPTION 'AD177 clave=audiencias_sha256 observado=% esperado=%',observado_audiencias_sha256,esperado_audiencias_sha256 USING ERRCODE='55000';
 END IF;
END $pre$;

-- Este comprobador usa sólo tablas de su autoridad. CC no las consulta.
-- La decisión y la auditoría deben pertenecer al efecto y a esta transacción.
CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(p_consumo jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='15s' SET TimeZone='UTC'
AS $f$
DECLARE r record; capacidad jsonb; decision jsonb; ahora timestamptz(6);
BEGIN
 IF current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
  OR current_setting('TimeZone')<>'UTC' OR pg_is_in_recovery()
  OR jsonb_typeof(p_consumo) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM jsonb_object_keys(p_consumo))<>7
  OR NOT (p_consumo ?& ARRAY['decision_ref','efecto_ref','huella_efecto_sha256',
    'consumo_huella_sha256','auditoria_ref','consumida_en','consumo_nuevo'])
  OR p_consumo->'consumo_nuevo' IS DISTINCT FROM 'true'::jsonb
  OR p_consumo->>'decision_ref' IS NULL OR p_consumo->>'efecto_ref' IS NULL
  OR (p_consumo->>'huella_efecto_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_consumo->>'consumo_huella_sha256' ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR p_consumo->>'auditoria_ref' IS NULL OR p_consumo->>'consumida_en' IS NULL THEN
  RAISE EXCEPTION 'AD177 consumo de gobierno no disponible' USING ERRCODE='42501';
 END IF;
 SELECT a.decision_ref,a.efecto_ref,a.huella_efecto_sha256,a.consumo_huella_sha256,
  a.consumida_en,u.auditoria_ref,u.registrada_en,u.huella_sha256 AS auditoria_huella_sha256,
  t.huella_decision_sha256,t.decision_canonica,t.capacidad_canonica,
  u.tipo_registro,u.version_consumo,u.proceso,u.canal,u.actor_ref,u.perfil_activo_ref,u.finalidad_ref,
  a.transaccion_origen AS consumo_origen,u.transaccion_origen AS auditoria_origen
 INTO STRICT r
 FROM vec_autorizacion_atestada_v3.consumo_decision_v3 a
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
  ON u.decision_ref=a.decision_ref AND u.efecto_ref=a.efecto_ref AND u.huella_efecto_sha256=a.huella_efecto_sha256
 JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t
  ON t.decision_ref=a.decision_ref AND t.efecto_ref=a.efecto_ref AND t.huella_efecto_sha256=a.huella_efecto_sha256
 WHERE a.decision_ref=p_consumo->>'decision_ref' FOR SHARE OF a,u,t;
 ahora:=clock_timestamp();
 IF r.efecto_ref IS DISTINCT FROM p_consumo->>'efecto_ref'
  OR r.huella_efecto_sha256 IS DISTINCT FROM p_consumo->>'huella_efecto_sha256'
  OR r.consumo_huella_sha256 IS DISTINCT FROM p_consumo->>'consumo_huella_sha256'
  OR r.auditoria_ref IS DISTINCT FROM p_consumo->>'auditoria_ref'
  OR r.consumida_en IS DISTINCT FROM (p_consumo->>'consumida_en')::timestamptz
  OR r.registrada_en IS DISTINCT FROM r.consumida_en
  OR r.consumo_origen IS DISTINCT FROM pg_current_xact_id()
  OR r.auditoria_origen IS DISTINCT FROM pg_current_xact_id()
  OR r.tipo_registro IS DISTINCT FROM 'consumo_confirmado_v4' OR r.version_consumo IS DISTINCT FROM 4
  OR r.consumida_en>ahora
  OR r.huella_decision_sha256 IS DISTINCT FROM encode(sha256(r.decision_canonica),'hex') THEN
  RAISE EXCEPTION 'AD177 consumo de gobierno no disponible' USING ERRCODE='42501';
 END IF;
 decision:=convert_from(r.decision_canonica,'UTF8')::jsonb;
 capacidad:=convert_from(r.capacidad_canonica,'UTF8')::jsonb;
 IF decision->>'decision_ref' IS DISTINCT FROM r.decision_ref
  OR decision->>'concedida' IS DISTINCT FROM 'true' OR decision->>'codigo' IS DISTINCT FROM 'concedida'
  OR decision->>'accion' IS DISTINCT FROM capacidad->>'operacion'
  OR (decision->>'accion' IN ('vec.catalogos.crear','vec.catalogos.actualizar','vec.catalogos.publicar','vec.catalogos.retirar')) IS NOT TRUE
  OR decision->>'recurso_ref' IS DISTINCT FROM r.efecto_ref
  OR decision->>'contexto_recurso_huella_sha256' IS DISTINCT FROM r.huella_efecto_sha256
  OR capacidad->>'efecto_ref' IS DISTINCT FROM r.efecto_ref
  OR capacidad->>'huella_efecto_sha256' IS DISTINCT FROM r.huella_efecto_sha256
  OR capacidad->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'
  OR decision->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal'
  OR decision->>'tipo_recurso' IS DISTINCT FROM 'catalogo_configurable'
  OR decision->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR decision#>>'{vinculo_autenticacion_actor,superficie}' IS DISTINCT FROM 'administracion_privilegiada'
  -- Repite la rama del núcleo: no depende de que sólo una fachada use este perfil.
  OR decision#>>'{vinculo_autenticacion_actor,cuenta_privilegiada}' IS DISTINCT FROM 'true'
  OR (decision->>'recurso_ref' ~ '^[a-z][a-z0-9._-]{2,127}:[1-9][0-9]{0,9}$') IS NOT TRUE
  OR decision->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR decision->'obligaciones' IS DISTINCT FROM '[]'::jsonb
  OR decision->>'principal_id' IS NULL OR decision->>'perfil_activo_ref' IS NULL
  OR r.actor_ref IS DISTINCT FROM decision->>'principal_id'
  OR r.perfil_activo_ref IS DISTINCT FROM decision->>'perfil_activo_ref'
  OR r.finalidad_ref IS DISTINCT FROM decision->>'finalidad'
  OR decision->>'valida_hasta' IS NULL OR (decision->>'valida_hasta')::timestamptz<=ahora THEN
  RAISE EXCEPTION 'AD177 consumo de gobierno no disponible' USING ERRCODE='42501';
 END IF;
 -- AUT acredita la categoría y las concesiones fijas, no el LOGIN.
 -- La fuente actual rechaza acciones no catalogadas; K debe publicar
 -- su extensión aprobada antes de habilitar el gobierno del plan.
 IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   decision->>'version_rol_ref',decision->>'asignacion_ref',decision->>'principal_id',decision->>'perfil_activo_ref',
   decision->>'accion','contratacion_temporal','catalogo_configurable','gestionar_contratacion_temporal',
   '[]'::jsonb,decision->'vinculo_autenticacion_actor') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD177 categoría Aplicación no acreditada' USING ERRCODE='42501';
 END IF;
 RETURN jsonb_build_object('decision_ref',r.decision_ref,'efecto_ref',r.efecto_ref,
  'huella_efecto_sha256',r.huella_efecto_sha256,'consumo_huella_sha256',r.consumo_huella_sha256,
  'auditoria_ref',r.auditoria_ref,'consumida_en',r.consumida_en,
  'registrador_principal_ref',decision->>'principal_id','registrador_perfil_ref',decision->>'perfil_activo_ref',
  'operacion',decision->>'accion','finalidad',decision->>'finalidad',
  'decision_valida_hasta',decision->>'valida_hasta','proceso',r.proceso,'canal',r.canal);
END $f$;

-- Fachada de gobierno de perfil técnico fijo. L incorpora ese perfil al
-- núcleo; este archivo no amplía su lista positiva ni acepta perfiles libres.
-- CC7 se resuelve al invocar PL/pgSQL después de completar la cadena causal.
CREATE FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(
 p_material_exacto bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
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
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288 THEN
  RAISE EXCEPTION 'AD177 material de gobierno inválido' USING ERRCODE='22023'; END IF;
 BEGIN
  original:=convert_from(p_material_exacto,'UTF8')::json; m:=original::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD177 material de gobierno inválido' USING ERRCODE='22023';
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
  RAISE EXCEPTION 'AD177 material de gobierno inválido' USING ERRCODE='22023'; END IF;
 BEGIN catalogo:=convert_from(decode(m->>'catalogo_canonico_base64','base64'),'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD177 catálogo de gobierno inválido' USING ERRCODE='22023';
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
  RAISE EXCEPTION 'AD177 catálogo de gobierno divergente' USING ERRCODE='22023'; END IF;
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{},"atributos":{"estado":"'||estado||
  '","material_sha256":"'||material_sha||'","revision":"'||revision||'"}}','UTF8')),'hex');
 IF d->>'accion' IS DISTINCT FROM accion OR c->>'operacion' IS DISTINCT FROM accion
  OR d->>'recurso_ref' IS DISTINCT FROM recurso OR c->>'efecto_ref' IS DISTINCT FROM recurso
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
  OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM 'catalogo_configurable'
  OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1'
  OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD177 gobierno de plan denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'gobierno_plan_nominal_firma_ct',p_capacidad,p_decision,p_motivo,p_contexto,
  p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 comprobado:=vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(to_jsonb(consumo));
 IF (comprobado->>'decision_valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'AD177 gobierno de plan caducado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT resultado FROM vec_catalogos_configurables.confirmar_gobierno_plan_nominal_firma_v1(
  p_material_exacto,jsonb_build_object('consumo',to_jsonb(consumo),'actor_ref',comprobado->>'registrador_principal_ref',
   'perfil_ref',comprobado->>'registrador_perfil_ref','accion',comprobado->>'operacion',
   'finalidad',comprobado->>'finalidad','proceso',comprobado->>'proceso','canal',comprobado->>'canal'));
 IF (comprobado->>'decision_valida_hasta')::timestamptz<=clock_timestamp() THEN
  RAISE EXCEPTION 'AD177 gobierno de plan caducado' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('recibo',to_jsonb(resultado),'consumo',to_jsonb(consumo));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(
 bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
-- Sólo el runtime CT: el núcleo exige que session_user pertenezca a ese grupo.
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(
 bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_contratacion_temporal_ejecutor;

-- Una fachada nueva evita ampliar el ACL de AD167 ya instalada.
CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(p_consumo jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='15s' SET TimeZone='UTC'
AS $f$
BEGIN
 RETURN vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(p_consumo);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb),
 vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_catalogos_configurables_propietario;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb),
 vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb) TO vec_catalogos_configurables_propietario;
DO $acl$
DECLARE nombre text; f regprocedure; permiso record;
 propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 catalogos oid:='vec_catalogos_configurables_propietario'::regrole;
 runtime_ct oid:='vec_contratacion_temporal_ejecutor'::regrole;
 exterior boolean;
BEGIN
 FOREACH nombre IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(jsonb)',
  'vec_autorizacion_atestada_v3.comprobar_consumo_firma_plan_ct_v1(jsonb)'] LOOP
  f:=nombre::regprocedure;
  exterior:=nombre='vec_autorizacion_atestada_v3.registrar_y_confirmar_gobierno_plan_firma_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)';
  IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario THEN
   RAISE EXCEPTION 'AD177 clave=propietario_funcion observado=incompatible esperado=propietario_AD' USING ERRCODE='55000';
  END IF;
  -- Retira también concesiones heredadas de privilegios predeterminados.
  FOR permiso IN SELECT DISTINCT x.grantee FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x
   WHERE p.oid=f AND x.grantee NOT IN(propietario,CASE WHEN exterior THEN runtime_ct ELSE catalogos END) LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,
    CASE WHEN permiso.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(permiso.grantee)) END);
  END LOOP;
  IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f)<>2
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f
    AND (x.grantee NOT IN(propietario,CASE WHEN exterior THEN runtime_ct ELSE catalogos END) OR x.grantor<>propietario
      OR x.privilege_type<>'EXECUTE' OR x.is_grantable)) THEN
   RAISE EXCEPTION 'AD177 clave=acl_funcion observado=incompatible esperado=AD_y_CC_o_runtime_CT_EXECUTE_sin_grant_option' USING ERRCODE='55000';
  END IF;
 END LOOP;
END $acl$;

-- Las dos funciones siguientes son privadas para la fachada CT176.
-- No amplían el núcleo ni sustituyen AD170 instalada.
CREATE FUNCTION vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(p_decision_interior bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='10s' SET TimeZone='UTC' AS $f$
DECLARE d jsonb; r record; recibo jsonb; prueba jsonb;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR pg_is_in_recovery() OR p_decision_interior IS NULL OR octet_length(p_decision_interior) NOT BETWEEN 2 AND 524288 THEN
  RAISE EXCEPTION 'AD177 consumo interior no disponible' USING ERRCODE='42501'; END IF;
 d:=convert_from(p_decision_interior,'UTF8')::jsonb;
 IF jsonb_typeof(d) IS DISTINCT FROM 'object' OR d->>'decision_ref' IS NULL THEN
  RAISE EXCEPTION 'AD177 decisión interior no disponible' USING ERRCODE='42501'; END IF;
 SELECT a.decision_ref,a.efecto_ref,a.huella_efecto_sha256,a.consumo_huella_sha256,u.auditoria_ref,a.consumida_en
 INTO STRICT r FROM vec_autorizacion_atestada_v3.consumo_decision_v3 a
 JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 u
  ON u.decision_ref=a.decision_ref AND u.efecto_ref=a.efecto_ref AND u.huella_efecto_sha256=a.huella_efecto_sha256
 JOIN vec_autorizacion_atestada_v3.atestacion_decision_v3 t
  ON t.decision_ref=a.decision_ref AND t.efecto_ref=a.efecto_ref AND t.huella_efecto_sha256=a.huella_efecto_sha256
 WHERE a.decision_ref=d->>'decision_ref' AND t.decision_canonica=p_decision_interior FOR SHARE OF a,u,t;
 recibo:=jsonb_build_object('decision_ref',r.decision_ref,'efecto_ref',r.efecto_ref,
  'huella_efecto_sha256',r.huella_efecto_sha256,'consumo_huella_sha256',r.consumo_huella_sha256,
  'auditoria_ref',r.auditoria_ref,'consumida_en',r.consumida_en,'consumo_nuevo',true);
 -- AD167, actualizado por AD193 sin cambiar ABI, exige ambos sellos TopXID
 -- actuales de consumo y auditoría y relee la atestación,
 -- acción/audiencia y vigencia. Un recibo original de otra TX no sirve.
 prueba:=vec_autorizacion_atestada_v3.comprobar_consumo_firma_ct_v1(recibo);
 IF prueba->>'decision_ref' IS DISTINCT FROM r.decision_ref
  OR prueba->>'consumo_huella_sha256' IS DISTINCT FROM r.consumo_huella_sha256 THEN
  RAISE EXCEPTION 'AD177 consumo interior divergente' USING ERRCODE='42501'; END IF;
 RETURN recibo;
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'AD177 consumo interior no disponible' USING ERRCODE='42501';
WHEN data_exception THEN
 RAISE EXCEPTION 'AD177 decisión interior inválida' USING ERRCODE='22023';
END $f$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(
 p_solicitud text,p_envoltorio bytea,p_decision_interior bytea,
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea
) RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog SET row_security='on' SET lock_timeout='2s' SET statement_timeout='10s' SET TimeZone='UTC' AS $f$
DECLARE s jsonb; e jsonb; original json; c jsonb; d jsonb; interior jsonb; x record;
 k text; recurso text; accion text; audiencia text; tipo text; perfil text; material_sha text; plan_sha text; contexto_sha text;
BEGIN
 IF current_setting('transaction_isolation')<>'serializable' OR current_setting('transaction_read_only')<>'off'
  OR pg_is_in_recovery() OR p_solicitud IS NULL OR octet_length(p_solicitud) NOT BETWEEN 2 AND 65536
  OR p_envoltorio IS NULL OR octet_length(p_envoltorio) NOT BETWEEN 512 AND 65536
  OR p_decision_interior IS NULL OR octet_length(p_decision_interior) NOT BETWEEN 2 AND 524288
  OR p_decision IS NULL OR octet_length(p_decision) NOT BETWEEN 2 AND 524288
  OR p_capacidad IS NULL OR octet_length(p_capacidad) NOT BETWEEN 2 AND 65536 THEN
  RAISE EXCEPTION 'AD177 plan atestado no disponible' USING ERRCODE='42501'; END IF;
 -- Sólo el parseo queda en una subtransacción, antes de cualquier efecto.
 -- Evita que un DETAIL del analizador conserve tokens del material privado.
 BEGIN
  s:=p_solicitud::jsonb; original:=convert_from(p_envoltorio,'UTF8')::json; e:=original::jsonb;
  c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
  interior:=convert_from(p_decision_interior,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'AD177 material exterior inválido' USING ERRCODE='22023';
 END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object' OR jsonb_typeof(e) IS DISTINCT FROM 'object'
  OR jsonb_typeof(c) IS DISTINCT FROM 'object' OR jsonb_typeof(d) IS DISTINCT FROM 'object'
  OR jsonb_typeof(interior) IS DISTINCT FROM 'object'
  -- json conserva claves repetidas y jsonb no: distinto recuento delata duplicados.
  OR (SELECT count(*) FROM json_each(p_solicitud::json))<>(SELECT count(*) FROM jsonb_object_keys(s))
  OR (SELECT count(*) FROM json_each(original))<>4 OR (SELECT count(*) FROM jsonb_object_keys(e))<>4
  OR NOT(e ?& ARRAY['esquema','descriptor','plan','decision_interior_sha256'])
  OR e->>'esquema' IS DISTINCT FROM 'ct.plan-autorizado-firma.v2'
  OR jsonb_typeof(e->'descriptor') IS DISTINCT FROM 'object' OR jsonb_typeof(e->'plan') IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM json_each(original->'plan'))<>4 OR (SELECT count(*) FROM jsonb_object_keys(e->'plan'))<>4
  OR NOT((e->'plan') ?& ARRAY['catalogo_id','catalogo_version','catalogo_huella_sha256','entrada_clave'])
  OR jsonb_typeof(e->'decision_interior_sha256') IS DISTINCT FROM 'string'
  OR e->>'decision_interior_sha256' IS DISTINCT FROM encode(sha256(p_decision_interior),'hex')
  OR d->>'decision_ref' IS NULL OR d->>'decision_ref' IS NOT DISTINCT FROM interior->>'decision_ref'
  OR s->>'Via' IS NULL OR s->>'Via' NOT IN('certificado_vec','portafirmas_registro_rrhh')
  OR s->>'PoliticaVerificacion' IS DISTINCT FROM 'politica:vec:firma:verificacion-autonoma:v2'
  OR jsonb_typeof(s->'OrganizacionRef') IS DISTINCT FROM 'string' OR (s->>'OrganizacionRef' ~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$') IS NOT TRUE
  OR jsonb_typeof(s->'ClaveIdempotencia') IS DISTINCT FROM 'string' OR (s->>'ClaveIdempotencia' ~ '^[A-Za-z0-9][A-Za-z0-9._-]{15,63}$') IS NOT TRUE THEN
  RAISE EXCEPTION 'AD177 material exterior divergente' USING ERRCODE='42501'; END IF;
 FOREACH k IN ARRAY ARRAY['principal_id','perfil_activo_ref','version_rol_ref','accion','recurso_ref','modulo_id','tipo_recurso','finalidad'] LOOP
  IF d->>k IS NULL OR d->>k IS DISTINCT FROM interior->>k THEN
   RAISE EXCEPTION 'AD177 decisiones exterior e interior no ligadas' USING ERRCODE='42501'; END IF;
 END LOOP;
 IF s->>'Via'='certificado_vec' THEN
  recurso:='operacion-firma-vec-ct:'||(s->>'ClaveIdempotencia'); accion:='contratacion_temporal.documento.firma_vec.registrar';
  audiencia:='vec_contratacion_temporal.firma_vec.v2'; tipo:='firma_vec_documento_contratacion_temporal'; perfil:='firma_vec_documento_ct_v2';
 ELSE
  recurso:='operacion-firma-externa-ct:'||(s->>'ClaveIdempotencia'); accion:='contratacion_temporal.documento.firma_externa.registrar';
  audiencia:='vec_contratacion_temporal.firma_externa.v2'; tipo:='firma_externa_documento_contratacion_temporal'; perfil:='firma_externa_documento_ct_v2';
 END IF;
 material_sha:=encode(sha256(convert_to(p_solicitud,'UTF8')),'hex'); plan_sha:=encode(sha256(p_envoltorio),'hex');
 contexto_sha:=encode(sha256(convert_to('{"ambitos":{"organizacion_ref":"'||(s->>'OrganizacionRef')||
  '"},"atributos":{"material_sha256":"'||material_sha||'","plan_firma_sha256":"'||plan_sha||'"}}','UTF8')),'hex');
 IF c->>'operacion' IS DISTINCT FROM accion OR c->>'audiencia_consumo' IS DISTINCT FROM audiencia
  OR c->>'efecto_ref' IS DISTINCT FROM recurso OR c->>'huella_efecto_sha256' IS DISTINCT FROM contexto_sha
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
  OR d->>'accion' IS DISTINCT FROM accion OR d->>'recurso_ref' IS DISTINCT FROM recurso
  OR d->>'modulo_id' IS DISTINCT FROM 'contratacion_temporal' OR d->>'tipo_recurso' IS DISTINCT FROM tipo
  OR d->>'finalidad' IS DISTINCT FROM 'gestionar_contratacion_temporal'
  OR d->>'perfil_activo_ref' IS DISTINCT FROM s->>'PerfilActivoOperadorRef'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM contexto_sha
  OR d->'campos_permitidos' IS DISTINCT FROM '[]'::jsonb OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb THEN
  RAISE EXCEPTION 'AD177 autorización exterior divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  perfil,p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NOT TRUE OR x.efecto_ref IS DISTINCT FROM recurso OR x.huella_efecto_sha256 IS DISTINCT FROM contexto_sha THEN
  RAISE EXCEPTION 'AD177 consumo exterior divergente' USING ERRCODE='42501'; END IF;
 RETURN jsonb_build_object('decision_ref',x.decision_ref,'efecto_ref',x.efecto_ref,'huella_efecto_sha256',x.huella_efecto_sha256,
  'consumo_huella_sha256',x.consumo_huella_sha256,'auditoria_ref',x.auditoria_ref,'consumida_en',x.consumida_en,'consumo_nuevo',true);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(bytea) FROM PUBLIC;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(bytea),
 vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_contratacion_temporal_propietario;
DO $acl_plan$
DECLARE f regprocedure; x record; propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole; ct oid:='vec_contratacion_temporal_propietario'::regrole;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_autorizacion_atestada_v3.recuperar_consumo_firma_plan_ct_v1(bytea)'::regprocedure,
  'vec_autorizacion_atestada_v3.consumir_plan_firma_ct_v2_atestada(text,bytea,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure] LOOP
  FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND a.grantee NOT IN(propietario,ct) LOOP
   EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
  END LOOP;
  IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
   OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
    AND (a.grantee NOT IN(propietario,ct) OR a.grantor<>propietario OR a.privilege_type<>'EXECUTE' OR a.is_grantable)) THEN
   RAISE EXCEPTION 'AD177 clave=acl_plan observado=incompatible esperado=AD_CT_owner_EXECUTE' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl_plan$;
COMMIT;
