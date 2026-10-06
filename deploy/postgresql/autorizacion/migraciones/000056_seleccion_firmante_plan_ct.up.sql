\set ON_ERROR_STOP on
-- AUT56: 4c-4, corte 2. Con Personal38, el enlace de cargo guarda el TIPO de
-- recurso del paso del plan, no un documento concreto (decisión de dirección
-- del 05/10). Dos cambios:
-- 1. AUT35 (construir_contexto_nominal_firmante_ct_v1) coteja el recurso del
--    enlace con rec->>'tipo_recurso' en lugar de con el documento. La
--    decisión y el consumo de la firma siguen ligados al documento exacto: el
--    descriptor de CT172 fija recurso_autorizable_ref = documento_ref =
--    OriginalRef y su huella entra en la de contexto de la decisión V3. Se
--    sustituye una sola línea con marca única sobre la definición instalada
--    (huella exacta); propietario, ACL y configuración no cambian.
-- 2. Fachada seleccionar_firmante_plan_ct_v1 para el ejecutor CT antes del
--    PDP: persona del certificado (CA25), enlace vigente del cargo por tipo
--    (Personal38) y la única asignación activa y vigente de esa persona con el
--    rol del paso. Devuelve la selección, la cuenta y el vínculo del
--    certificado, y las versiones y huellas de asignación, rol y control; no escribe ni concede nada: AUT32/AUT35 y el
--    consumo V3 vuelven a comprobarlo todo en la transacción de la firma.
-- Requiere AUT35, CA25 y Personal38. Una sola vez; sin DOWN.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_autorizacion:migracion:000056',0));
DO $pre$
DECLARE f regprocedure:=pg_catalog.to_regprocedure('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)');
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper)
 OR pg_catalog.current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 THEN RAISE EXCEPTION 'AUT56: PARO clave=migrador actual=no_acreditado esperado=superusuario_PG18' USING ERRCODE='42501'; END IF;
 IF f IS NULL
 OR pg_catalog.to_regprocedure('vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text)') IS NULL
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario','vec_personal.localizar_enlace_cargo_ct_v1(text,text,text,text,text,text,text)','EXECUTE')
 OR NOT pg_catalog.has_function_privilege('vec_autorizacion_propietario','vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(text)','EXECUTE')
 OR pg_catalog.to_regprocedure('vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text)') IS NOT NULL
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname IN('vec_autorizacion_propietario','vec_contratacion_temporal_ejecutor') AND rolcanlogin)
 THEN RAISE EXCEPTION 'AUT56: PARO clave=preimagen actual=incompatible esperado=AUT35_CA25_Personal38_sin_AUT56' USING ERRCODE='55000'; END IF;
 IF pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(pg_catalog.pg_get_functiondef(f),'UTF8')),'hex')
   IS DISTINCT FROM '560cedb87481bb894abc3e5fcb259d2b4a68160df207e558bdd360cf426f0766'
 THEN RAISE EXCEPTION 'AUT56: PARO clave=AUT35 actual=distinta esperado=definicion_instalada' USING ERRCODE='55000'; END IF;
END $pre$;

-- 1. AUT35: el enlace se coteja por tipo. Marca única, reversible y sin
-- cambio de propietario, ACL ni configuración.
DO $aut35$
DECLARE f regprocedure:='vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)'::regprocedure;
 viejo text:=$m$  OR per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'recurso_autorizable_ref'$m$;
 nuevo text:=$m$  OR per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'tipo_recurso'$m$;
 original text;nueva text;meta jsonb;acl aclitem[];cfg text[];
BEGIN
 SELECT pg_catalog.pg_get_functiondef(p.oid),to_jsonb(p)-'prosrc',p.proacl,p.proconfig INTO STRICT original,meta,acl,cfg
  FROM pg_catalog.pg_proc p WHERE p.oid=f;
 IF (pg_catalog.length(original)-pg_catalog.length(pg_catalog.replace(original,viejo,'')))/pg_catalog.length(viejo)<>1
  OR pg_catalog.strpos(original,nuevo)<>0
 THEN RAISE EXCEPTION 'AUT56: PARO clave=marca_AUT35 actual=no_unica esperado=una' USING ERRCODE='55000'; END IF;
 nueva:=pg_catalog.replace(original,viejo,nuevo);
 IF pg_catalog.replace(nueva,nuevo,viejo) IS DISTINCT FROM original
 THEN RAISE EXCEPTION 'AUT56: PARO clave=reversion_AUT35 actual=distinta esperado=original' USING ERRCODE='55000'; END IF;
 EXECUTE nueva;
 IF (SELECT to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT p.proacl FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM acl
  OR (SELECT p.proconfig FROM pg_catalog.pg_proc p WHERE p.oid=f) IS DISTINCT FROM cfg
  OR pg_catalog.pg_get_functiondef(f) IS DISTINCT FROM nueva
 THEN RAISE EXCEPTION 'AUT56: PARO clave=postimagen_AUT35 actual=distinta esperado=solo_una_linea' USING ERRCODE='55000'; END IF;
END $aut35$;

-- 2. Selección central del firmante para el paso del plan.
SET LOCAL ROLE vec_autorizacion_propietario;
CREATE FUNCTION vec_autorizacion.seleccionar_firmante_plan_ct_v1(
 p_certificado_der_sha256 text,p_cargo_ref text,p_rol_id text,p_accion text,p_tipo_recurso text,p_finalidad text,
 p_organizacion_ref text,p_unidad_ref text)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET lock_timeout='2s' SET TimeZone='UTC' AS $f$
DECLARE ca jsonb;en jsonb;a record;ahora timestamptz(6);
BEGIN
 IF pg_catalog.current_setting('transaction_isolation')<>'serializable'
  OR pg_catalog.current_setting('transaction_read_only')<>'off'
  OR pg_catalog.current_setting('role')<>'none'
  OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
  OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
  OR pg_catalog.pg_has_role(session_user,'vec_autorizacion_propietario','MEMBER')
  OR (p_certificado_der_sha256 ~ '^[0-9a-f]{64}$') IS NOT TRUE
  OR (p_rol_id ~ '^[a-z][a-z0-9_]{2,63}$') IS NOT TRUE
 THEN RAISE EXCEPTION 'seleccion_firmante_denegada' USING ERRCODE='42501'; END IF;
 -- Persona del certificado: CA25 revalida vínculo, cuenta, persona y
 -- organización de destino; un certificado retirado o ajeno no selecciona.
 ca:=vec_contexto_actor_v1.leer_revalidar_certificado_firmante_ct_v2(p_certificado_der_sha256);
 IF ca IS NULL OR ca->>'estado' IS DISTINCT FROM 'vigente'
  OR ca->>'certificado_der_sha256' IS DISTINCT FROM p_certificado_der_sha256
  OR ca#>>'{organizacion_destino,organizacion_ref}' IS DISTINCT FROM p_organizacion_ref
 THEN RAISE EXCEPTION 'seleccion_firmante_certificado_no_admitido' USING ERRCODE='42501'; END IF;
 -- Enlace vigente y único del cargo por tipo (Personal38 valida el resto).
 en:=vec_personal.localizar_enlace_cargo_ct_v1(p_cargo_ref,ca->>'persona_ref',p_accion,p_tipo_recurso,p_finalidad,
  p_organizacion_ref,p_unidad_ref);
 -- Única asignación activa y vigente de esa persona con el rol del paso, rol
 -- publicado y control habilitado. Más de una es ambigua: se deniega.
 ahora:=pg_catalog.clock_timestamp();
 SELECT x.asignacion_ref,x.version,x.huella_sha256,x.perfil_activo_ref,x.documento,
  r.version_rol_ref,r.huella_sha256 AS rol_huella,v.revision AS control_revision,v.huella_sha256 AS control_huella
  INTO STRICT a
  FROM vec_autorizacion.asignacion_perfil x
  JOIN vec_autorizacion.asignacion_perfil_actual p ON p.perfil_activo_ref=x.perfil_activo_ref AND p.asignacion_ref=x.asignacion_ref
  JOIN vec_autorizacion.version_rol r ON r.version_rol_ref=x.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol_actual c ON c.version_rol_ref=r.version_rol_ref
  JOIN vec_autorizacion.control_vigencia_version_rol v ON v.version_rol_ref=c.version_rol_ref AND v.revision=c.revision
  WHERE x.principal_id=ca->>'persona_ref' AND r.rol_id=p_rol_id
   AND x.documento->>'estado'='activa' AND r.documento->>'estado'='publicada' AND v.estado='habilitada'
   AND ahora>=(x.documento->>'vigente_desde')::timestamptz AND ahora<(x.documento->>'vigente_hasta')::timestamptz
   AND x.documento->'ambitos' @> pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('clave','organizacion_ref',
    'valores',pg_catalog.jsonb_build_array(p_organizacion_ref)))
   AND (NOT x.documento->'ambitos' @> '[{"clave":"unidad_ref"}]'::jsonb
    OR x.documento->'ambitos' @> pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('clave','unidad_ref',
     'valores',pg_catalog.jsonb_build_array(p_unidad_ref))))
  FOR SHARE OF x,p,r,c,v;
 RETURN pg_catalog.jsonb_build_object('esquema','vec.autorizacion.seleccion-firmante-plan.ct.v1',
  'persona_ref',ca->>'persona_ref','perfil_activo_ref',a.perfil_activo_ref,'rol_id',p_rol_id,
  'cuenta_ref',ca#>>'{vinculo_certificado,cuenta_ref}',
  'vinculo_certificado',pg_catalog.jsonb_build_object('referencia',ca#>>'{vinculo_certificado,referencia}',
   'version',ca#>'{vinculo_certificado,version}','huella_sha256',ca#>>'{vinculo_certificado,huella_sha256}'),
  'cargo_ref',p_cargo_ref,'enlace_ejercicio_ref',en#>>'{enlace,referencia}',
  'asignacion',pg_catalog.jsonb_build_object('referencia',a.asignacion_ref,'version',a.version,'huella_sha256',a.huella_sha256,
   'vigente_desde',a.documento->'vigente_desde','vigente_hasta',a.documento->'vigente_hasta'),
  'rol',pg_catalog.jsonb_build_object('referencia',a.version_rol_ref,'huella_sha256',a.rol_huella),
  'control_rol',pg_catalog.jsonb_build_object('referencia',a.version_rol_ref,'revision',a.control_revision,'huella_sha256',a.control_huella));
EXCEPTION WHEN no_data_found OR too_many_rows THEN
 RAISE EXCEPTION 'seleccion_firmante_ausente_o_ambigua' USING ERRCODE='42501';
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text) TO vec_contratacion_temporal_ejecutor;
RESET ROLE;

DO $post$
DECLARE s regprocedure:='vec_autorizacion.seleccionar_firmante_plan_ct_v1(text,text,text,text,text,text,text,text)'::regprocedure;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc WHERE oid=s AND proowner='vec_autorizacion_propietario'::regrole AND prosecdef)
 OR EXISTS(SELECT 1 FROM pg_catalog.pg_proc p CROSS JOIN LATERAL pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) x
   WHERE p.oid=s AND (x.grantee NOT IN(p.proowner,'vec_contratacion_temporal_ejecutor'::regrole) OR x.is_grantable))
 OR pg_catalog.strpos(pg_catalog.pg_get_functiondef('vec_autorizacion.construir_contexto_nominal_firmante_ct_v1(jsonb,jsonb,jsonb)'::regprocedure),
   $m$per->>'recurso_autorizable_ref' IS DISTINCT FROM rec->>'tipo_recurso'$m$)=0
 THEN RAISE EXCEPTION 'AUT56: PARO clave=postimagen actual=divergente esperado=AUT35_por_tipo_y_fachada_solo_CT' USING ERRCODE='55000'; END IF;
END $post$;
COMMIT;
