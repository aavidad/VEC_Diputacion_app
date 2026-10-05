\set ON_ERROR_STOP on
-- AD205: publicación de certificados nominales de firmante desde vec-admin.
-- La fachada v3 de AD165 exige que la huella del recurso de la decisión sea el
-- SHA-256 del descriptor. El PDP común calcula siempre la huella canónica de
-- ámbitos y atributos, y además exige que el recurso tenga exactamente las
-- dimensiones de la asignación del administrador (organización y unidad). Por
-- eso ninguna decisión real podía cumplir la v3.
-- La v4 calcula la huella canónica:
--   {"ambitos":{"organizacion_ref":…,"unidad_ref":…},
--    "atributos":{"descriptor_sha256":…}}
-- con la organización y la unidad únicas de la asignación vigente de
-- Aplicación consumida, que lee una función nueva de solo lectura. El resto
-- de la v3 se conserva igual: consumidor, destino, ámbito, separación del
-- administrador y revalidación final. Además exige que la organización del
-- destino sea la de la asignación, y devuelve el recibo de CA25 junto al
-- consumo de este acceso. El grupo ejecutor pasa de la v3 a la v4.
-- Añade también el conjunto 4 de capacidades ADMIN (el 3 más la audiencia de
-- certificados nominales, tramo certificados:nominal) para que vec-admin
-- emita esas decisiones, y corrige destino_no_administrador_certificado_
-- nominal_v1 (AUT33), que fallaba siempre con un USING ambiguo. No toca el
-- núcleo ni el CHECK de audiencias.
-- Requiere AD165, AD198 y AD204. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion_atestada_v3:migracion:000205',0));
DO $pre$
DECLARE tres record;v3 regprocedure:=pg_catalog.to_regprocedure('vec_autorizacion.operar_certificado_nominal_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
BEGIN
 IF pg_catalog.current_setting('server_version_num')::integer NOT BETWEEN 180000 AND 189999
 OR NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 THEN RAISE EXCEPTION 'AD205: PARO clave=migrador_PG actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF v3 IS NULL OR pg_catalog.to_regrole('vec_autorizacion_certificado_nominal_ejecutor') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_certificado_nominal_ejecutor',v3,'EXECUTE')
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid=v3)<>'vec_autorizacion_propietario'::regrole
 OR pg_catalog.to_regprocedure('vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.asignacion_perfil_actual') IS NULL
 OR pg_catalog.to_regclass('vec_autorizacion.perfil_fijo_categoria_nominal_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.operar_certificado_nominal_v4(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.ambitos_asignacion_aplicacion_v1(text,text,text)') IS NOT NULL
 THEN RAISE EXCEPTION 'AD205: PARO clave=preimagen actual=incompatible esperado=AD165_sin_v4' USING ERRCODE='55000'; END IF;
 -- AUT33 dejó destino_no_administrador_certificado_nominal_v1 con un USING
 -- ambiguo (version_rol_ref aparece dos veces a la izquierda): cualquier
 -- destino ordinario acababa en error 42702. Se exige su definición exacta.
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(
   'vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text)'::regprocedure),'UTF8')),'hex')
   IS DISTINCT FROM '5e29d343d81975df7c86473c4c81ed681effb264e10d94f80161a05db69459bb'
 THEN RAISE EXCEPTION 'AD205: PARO clave=destino_no_administrador actual=distinto esperado=AUT33' USING ERRCODE='55000'; END IF;
 IF pg_catalog.to_regclass('vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1') IS NULL
 OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=4)
 THEN RAISE EXCEPTION 'AD205: PARO clave=conjuntos actual=incompatible esperado=AD198_sin_conjunto_4' USING ERRCODE='55000'; END IF;
 SELECT * INTO tres FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=3;
 IF NOT FOUND
 OR tres.audiencias IS DISTINCT FROM ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1']
 OR tres.segmentos IS DISTINCT FROM ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial']
 THEN RAISE EXCEPTION 'AD205: PARO clave=conjunto_3 actual=distinto esperado=AD204' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
   AND pg_catalog.strpos(pg_catalog.pg_get_constraintdef(c.oid,false),'''vec_contexto_actor.certificado_nominal.publicar.v1''::text')>0)
 THEN RAISE EXCEPTION 'AD205: PARO clave=AD165 actual=audiencia_no_admitida esperado=CHECK_con_certificado_nominal' USING ERRCODE='55000'; END IF;
END $pre$;

SET LOCAL ROLE vec_autorizacion_propietario;
-- Organización y unidad únicas de la asignación actual de Aplicación (rol
-- administracion_perfiles con metadatos de perfil fijo, como AUT48/AUT52),
-- activa, vigente y de la persona indicada. NULL si no es así o si alguna
-- dimensión falta o tiene más de un valor. No escribe nada.
CREATE FUNCTION vec_autorizacion.ambitos_asignacion_aplicacion_v1(version_ref text,p_asignacion_ref text,principal_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog AS $f$
DECLARE a record;ahora timestamptz;org text;unidad text;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR (version_ref ~ '^rol:administracion_perfiles:v[1-9][0-9]{0,8}$') IS NOT TRUE
  OR principal_ref IS NULL OR p_asignacion_ref IS NULL THEN RETURN NULL; END IF;
 IF NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r JOIN vec_autorizacion.perfil_fijo_categoria_nominal_v1 meta
  ON meta.version_rol_ref=r.version_rol_ref AND meta.fuente_ref=r.version_rol_ref AND meta.fuente_version=r.version AND meta.version_rol_huella_sha256=r.huella_sha256
  WHERE r.version_rol_ref=version_ref AND r.rol_id='administracion_perfiles' AND r.version_rol_ref='rol:administracion_perfiles:v'||r.version::text
  AND meta.categoria_administrativa='aplicacion' AND meta.tipo_perfil='fijo_sistema') THEN RETURN NULL; END IF;
 SELECT x.* INTO a FROM vec_autorizacion.asignacion_perfil_actual p
 JOIN vec_autorizacion.asignacion_perfil x USING(perfil_activo_ref,asignacion_ref)
 WHERE p.asignacion_ref=p_asignacion_ref FOR SHARE OF p,x;
 ahora:=pg_catalog.clock_timestamp();
 IF NOT FOUND OR a.version_rol_ref<>version_ref OR a.principal_id IS DISTINCT FROM principal_ref
  OR a.documento->>'estado'<>'activa'
  OR (ahora>=(a.documento->>'vigente_desde')::timestamptz AND ahora<(a.documento->>'vigente_hasta')::timestamptz) IS NOT TRUE
  OR pg_catalog.jsonb_typeof(a.documento->'ambitos')<>'array' OR pg_catalog.jsonb_array_length(a.documento->'ambitos')<>2 THEN RETURN NULL; END IF;
 SELECT e->'valores'->>0 INTO org FROM pg_catalog.jsonb_array_elements(a.documento->'ambitos') e
  WHERE e->>'clave'='organizacion_ref' AND pg_catalog.jsonb_typeof(e->'valores')='array' AND pg_catalog.jsonb_array_length(e->'valores')=1;
 SELECT e->'valores'->>0 INTO unidad FROM pg_catalog.jsonb_array_elements(a.documento->'ambitos') e
  WHERE e->>'clave'='unidad_ref' AND pg_catalog.jsonb_typeof(e->'valores')='array' AND pg_catalog.jsonb_array_length(e->'valores')=1;
 IF (org ~ '^org_[a-z0-9]{16,80}$') IS NOT TRUE OR (unidad ~ '^[A-Za-z0-9][A-Za-z0-9:._-]{2,159}$') IS NOT TRUE THEN RETURN NULL; END IF;
 RETURN pg_catalog.jsonb_build_object('organizacion_ref',org,'unidad_ref',unidad);
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.ambitos_asignacion_aplicacion_v1(text,text,text) FROM PUBLIC;

-- Huella canónica del recurso, idéntica a la del PDP: claves ordenadas,
-- valores JSON y sin espacios.
CREATE FUNCTION vec_autorizacion.huella_recurso_certificado_nominal_v1(amb jsonb,descriptor_sha text)
RETURNS text LANGUAGE sql IMMUTABLE SET search_path=pg_catalog AS $f$
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
  '{"ambitos":{"organizacion_ref":'||pg_catalog.to_jsonb(amb->>'organizacion_ref')::text||',"unidad_ref":'||pg_catalog.to_jsonb(amb->>'unidad_ref')::text||
  '},"atributos":{"descriptor_sha256":'||pg_catalog.to_jsonb(descriptor_sha)::text||'}}','UTF8')),'hex')
$f$;
REVOKE ALL ON FUNCTION vec_autorizacion.huella_recurso_certificado_nominal_v1(jsonb,text) FROM PUBLIC;

-- Corrección del USING ambiguo de AUT33: misma función, con el cruce de la
-- revisión de control escrito explícito. Conserva propietario y ACL.
CREATE OR REPLACE FUNCTION vec_autorizacion.destino_no_administrador_certificado_nominal_v1(cuenta text, persona text)
 RETURNS boolean
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog'
 SET lock_timeout TO '2s'
AS $function$
DECLARE privilegiada boolean;ahora timestamptz;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR cuenta IS NULL OR persona IS NULL THEN RETURN false; END IF;
 privilegiada:=vec_identidad_sesiones_v1.clasificar_cuenta_privilegiada_nominal_v1(cuenta);
 IF privilegiada IS DISTINCT FROM false THEN RETURN false; END IF;
 -- Impide una asignación administrativa concurrente, incluida una sobre
 -- otro perfil activo de la misma persona, hasta el COMMIT del certificado.
 LOCK TABLE vec_autorizacion.asignacion_perfil_actual IN SHARE MODE;
 LOCK TABLE vec_autorizacion.control_vigencia_version_rol_actual IN SHARE MODE;
 ahora:=pg_catalog.clock_timestamp();
 RETURN NOT EXISTS(
  SELECT 1 FROM vec_autorizacion.asignacion_perfil_actual x
  JOIN vec_autorizacion.asignacion_perfil a USING(perfil_activo_ref,asignacion_ref)
  JOIN vec_autorizacion.rol_sensible_exacto s ON s.version_rol_ref=a.version_rol_ref
  JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=s.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol_actual ca ON ca.version_rol_ref=r.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol cv ON cv.version_rol_ref=ca.version_rol_ref AND cv.revision=ca.revision
  WHERE a.principal_id=persona AND s.clase='administrador' AND s.huella_sha256=r.huella_sha256
   AND a.documento->>'estado'='activa' AND r.documento->>'estado'='publicada'
   AND cv.estado='habilitada'
   AND ahora>=(a.documento->>'vigente_desde')::timestamptz
   AND ahora<(a.documento->>'vigente_hasta')::timestamptz);
END $function$;

CREATE FUNCTION vec_autorizacion.operar_certificado_nominal_v4(
 p_descriptor bytea,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE d jsonb;c jsonb;a jsonb;s jsonb;org jsonb;amb jsonb;x record;resultado jsonb;accion text;sha text;huella text;
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable' OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR p_descriptor IS NULL OR pg_catalog.octet_length(p_descriptor) NOT BETWEEN 2 AND 16384
  OR p_capacidad IS NULL OR p_decision IS NULL
 THEN RAISE EXCEPTION 'AD205: operación denegada' USING ERRCODE='42501'; END IF;
 BEGIN
  d:=pg_catalog.convert_from(p_descriptor,'UTF8')::jsonb;
  c:=pg_catalog.convert_from(p_capacidad,'UTF8')::jsonb;
  a:=pg_catalog.convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN
  RAISE EXCEPTION 'AD205: material inválido' USING ERRCODE='22023';
 END;
 IF pg_catalog.jsonb_typeof(d)<>'object' THEN RAISE EXCEPTION 'AD205: material inválido' USING ERRCODE='22023'; END IF;
 accion:=CASE d->>'estado' WHEN 'vigente' THEN 'administracion.certificados.nominal.publicar'
   WHEN 'retirado' THEN 'administracion.certificados.nominal.retirar' ELSE NULL END;
 sha:=pg_catalog.encode(pg_catalog.sha256(p_descriptor),'hex');
 amb:=vec_autorizacion.ambitos_asignacion_aplicacion_v1(a->>'version_rol_ref',a->>'asignacion_ref',a->>'principal_id');
 IF amb IS NULL THEN RAISE EXCEPTION 'AD205: asignación sin ámbitos únicos' USING ERRCODE='42501'; END IF;
 huella:=vec_autorizacion.huella_recurso_certificado_nominal_v1(amb,sha);
 IF accion IS NULL OR c->>'operacion' IS DISTINCT FROM accion
  OR c->>'efecto_ref' IS DISTINCT FROM 'certificado-nominal:'||(d->>'certificado_der_sha256')
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM huella
  OR a->>'contexto_recurso_huella_sha256' IS DISTINCT FROM huella
  OR a->>'principal_id' IS NOT DISTINCT FROM d->>'persona_ref'
  OR d->>'organizacion_ref' IS DISTINCT FROM amb->>'organizacion_ref'
 THEN RAISE EXCEPTION 'AD205: recurso denegado' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_publicacion_certificado_nominal_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 s:=vec_contexto_actor_v1.fuentes_certificado_firmante_ct_v2(
  d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref');
 IF s IS NULL THEN RAISE EXCEPTION 'AD205: identidad destino no acreditada' USING ERRCODE='42501'; END IF;
 org:=vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
  d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref',
  (s#>>'{vinculo_cuenta_persona,version}')::numeric,d->>'organizacion_ref');
 IF org IS NULL OR org->>'organizacion_ref' IS DISTINCT FROM amb->>'organizacion_ref'
  OR vec_autorizacion.acreditar_ambito_certificado_nominal_v1(
   a->>'version_rol_ref',a->>'asignacion_ref',org->>'organizacion_ref') IS NOT TRUE
  OR vec_autorizacion.destino_no_administrador_certificado_nominal_v1(d->>'cuenta_ref',d->>'persona_ref') IS NOT TRUE
 THEN RAISE EXCEPTION 'AD205: destino denegado' USING ERRCODE='42501'; END IF;
 resultado:=vec_contexto_actor_v1.publicar_certificado_firmante_ct_v2(p_descriptor,x.decision_ref,x.auditoria_ref);
 IF resultado IS NULL OR vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(p_decision,p_motivo,p_persona_version,p_perfil_version) IS NULL
  OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(
   a->>'version_rol_ref',a->>'asignacion_ref',a->>'principal_id',a->>'perfil_activo_ref',
   accion,'administracion','vinculo_certificado_nominal','gestionar_certificados_firmantes','[]'::jsonb,a->'vinculo_autenticacion_actor') IS NOT TRUE
  OR vec_autorizacion.ambitos_asignacion_aplicacion_v1(a->>'version_rol_ref',a->>'asignacion_ref',a->>'principal_id') IS DISTINCT FROM amb
  OR vec_autorizacion.destino_no_administrador_certificado_nominal_v1(d->>'cuenta_ref',d->>'persona_ref') IS NOT TRUE
  OR vec_autorizacion.acreditar_ambito_certificado_nominal_v1(
   a->>'version_rol_ref',a->>'asignacion_ref',org->>'organizacion_ref') IS NOT TRUE
  OR vec_contexto_actor_v1.acreditar_organizacion_destino_certificado_v1(
   d->>'cuenta_ref',d->>'persona_ref',d->>'vinculo_cuenta_persona_ref',
   (s#>>'{vinculo_cuenta_persona,version}')::numeric,d->>'organizacion_ref') IS DISTINCT FROM org
 THEN RAISE EXCEPTION 'AD205: revalidación final denegada' USING ERRCODE='42501'; END IF;
 -- En un reintento CA25 devuelve el recibo original; el consumo de este
 -- acceso va aparte para que quien llama lo pueda acreditar.
 RETURN pg_catalog.jsonb_build_object('recibo',resultado,
  'consumo',pg_catalog.jsonb_build_object('decision_ref',x.decision_ref,'auditoria_ref',x.auditoria_ref));
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.operar_certificado_nominal_v4(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.operar_certificado_nominal_v4(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 TO vec_autorizacion_certificado_nominal_ejecutor;
-- La v3 deja de ser ejecutable: ninguna decisión real podía cumplirla.
REVOKE EXECUTE ON FUNCTION vec_autorizacion.operar_certificado_nominal_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM vec_autorizacion_certificado_nominal_ejecutor;
RESET ROLE;

SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES
 (4,ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
   'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1'],
  ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal']);
RESET ROLE;

DO $post$
DECLARE v4 regprocedure:='vec_autorizacion.operar_certificado_nominal_v4(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 v3 regprocedure:='vec_autorizacion.operar_certificado_nominal_v3(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 internas regprocedure[]:=ARRAY['vec_autorizacion.ambitos_asignacion_aplicacion_v1(text,text,text)',
  'vec_autorizacion.huella_recurso_certificado_nominal_v1(jsonb,text)']::regprocedure[];
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=v4 AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=v4 AND (x.grantee NOT IN(p.proowner,'vec_autorizacion_certificado_nominal_ejecutor'::regrole) OR x.is_grantable))
 OR (SELECT count(*) FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=v4 AND x.grantee='vec_autorizacion_certificado_nominal_ejecutor'::regrole AND x.privilege_type='EXECUTE')<>1
 OR pg_catalog.has_function_privilege('vec_autorizacion_certificado_nominal_ejecutor',v3,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=ANY(internas) AND x.grantee<>p.proowner)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=ANY(internas) AND proowner<>'vec_autorizacion_propietario'::regrole)
 THEN RAISE EXCEPTION 'AD205: PARO clave=acl_post actual=divergente esperado=v4_solo_grupo_y_v3_cerrada' USING ERRCODE='55000'; END IF;
 IF pg_catalog.strpos(pg_catalog.pg_get_functiondef('vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text)'::regprocedure),
   'USING(version_rol_ref,revision)')<>0
 OR (SELECT proowner FROM pg_catalog.pg_proc WHERE oid='vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text)'::regprocedure)
   <>'vec_autorizacion_propietario'::regrole
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid='vec_autorizacion.destino_no_administrador_certificado_nominal_v1(text,text)'::regprocedure
   AND x.grantee NOT IN(p.proowner,'vec_autorizacion_atestada_v3_propietario'::regrole))
 THEN RAISE EXCEPTION 'AD205: PARO clave=destino_no_administrador_post actual=divergente esperado=sin_USING_y_misma_ACL' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=4
   AND audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1',
    'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1','vec_personal.cargo_competencial.publicar.v1','vec_contexto_actor.certificado_nominal.publicar.v1']
   AND segmentos=ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote','catalogos:plan-firma','personal:cargo-competencial','certificados:nominal'])<>1
 THEN RAISE EXCEPTION 'AD205: PARO clave=postimagen esperado=conjunto_4_exacto actual=divergente' USING ERRCODE='55000'; END IF;
 -- La huella canónica coincide con un valor fijo calculado aparte (Go).
 IF vec_autorizacion.huella_recurso_certificado_nominal_v1('{"organizacion_ref":"org_aaaaaaaaaaaaaaaa","unidad_ref":"unidad:x"}'::jsonb,pg_catalog.repeat('b',64))
   IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   '{"ambitos":{"organizacion_ref":"org_aaaaaaaaaaaaaaaa","unidad_ref":"unidad:x"},"atributos":{"descriptor_sha256":"'||pg_catalog.repeat('b',64)||'"}}','UTF8')),'hex')
 THEN RAISE EXCEPTION 'AD205: PARO clave=huella_canonica actual=distinta esperado=PDP' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
