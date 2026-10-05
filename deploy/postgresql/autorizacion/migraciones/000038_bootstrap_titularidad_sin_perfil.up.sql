\set ON_ERROR_STOP on
-- AUT38: sucesora mínima de la preimagen CA publicada por AUT37.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='60s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
DO $pre$
DECLARE actual text;
BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_roles WHERE rolname=current_user AND rolsuper) THEN
  RAISE EXCEPTION 'AUT38: PARO clave=migrador.superusuario actual=false esperado=true' USING ERRCODE='42501'; END IF;
 IF pg_catalog.to_regclass('vec_contexto_actor_v1.titularidad_cuenta_persona_v1') IS NULL
 OR pg_catalog.to_regclass('vec_contexto_actor_v1.fuentes_iniciales_admin_v1') IS NULL
 OR pg_catalog.to_regprocedure('vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(text,text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_autorizacion.registrar_bootstrap_central_admin_v3(text,text)') IS NULL
 OR pg_catalog.to_regprocedure('vec_personal.cotejar_unidad_bootstrap_admin_v1(text,jsonb,timestamptz)') IS NULL
 THEN RAISE EXCEPTION 'AUT38: PARO clave=dependencias actual=ausente esperado=AUT37_CA33_IS15_Personal31' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_catalog.pg_proc p
  WHERE p.oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(jsonb)')
   AND p.prorettype='jsonb'::regtype AND p.prolang=(SELECT oid FROM pg_catalog.pg_language WHERE lanname='plpgsql')
   AND pg_catalog.cardinality(p.proconfig)=2 AND 'search_path=pg_catalog'=ANY(p.proconfig)
   AND EXISTS(SELECT 1 FROM pg_catalog.unnest(p.proconfig) e WHERE pg_catalog.lower(pg_catalog.split_part(e,'=',1))='timezone' AND pg_catalog.split_part(e,'=',2)='UTC')
   AND pg_catalog.has_function_privilege('vec_autorizacion_propietario',p.oid,'EXECUTE')
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(COALESCE(p.proacl,pg_catalog.acldefault('f',p.proowner))) a
    WHERE a.grantee NOT IN('vec_contexto_actor_v1_propietario'::regrole,'vec_autorizacion_propietario'::regrole)
     OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 THEN RAISE EXCEPTION 'AUT38: PARO clave=AUT37.CA_preimagen.firma_config_ACL actual=divergente esperado=fachada_privada_original' USING ERRCODE='55000'; END IF;
 SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(prosrc,'UTF8')),'hex') INTO actual FROM pg_catalog.pg_proc
 WHERE oid=pg_catalog.to_regprocedure('vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(jsonb)')
 AND proowner='vec_contexto_actor_v1_propietario'::regrole AND prosecdef AND provolatile='v';
 IF actual IS DISTINCT FROM 'a2dfb507381b22a0d654083e3ca60582bba18a77aedc626493fe15f624ab873a' THEN
  RAISE EXCEPTION 'AUT38: PARO clave=AUT37.CA_preimagen.prosrc_sha256 actual=% esperado=a2dfb507381b22a0d654083e3ca60582bba18a77aedc626493fe15f624ab873a',COALESCE(actual,'ausente') USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $publicacion$
DECLARE previa jsonb; actual jsonb; funcion oid;
BEGIN
 funcion:=pg_catalog.to_regprocedure('vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(jsonb)');
 SELECT pg_catalog.to_jsonb(p)-'prosrc' INTO STRICT previa FROM pg_catalog.pg_proc p WHERE p.oid=funcion;
 -- Definición literal revisable: no reconstruye ni reemplaza texto del catálogo.
 EXECUTE $definicion$
CREATE OR REPLACE FUNCTION vec_contexto_actor_v1.preimagen_persona_bootstrap_central_admin_v3(p jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET timezone='UTC' AS $f$
DECLARE c jsonb;sub jsonb;sistema jsonb;sistemas jsonb:='[]'::jsonb;ambitos jsonb;organizaciones jsonb;legado boolean;
 titularidad jsonb;fuente jsonb;persona_fuente jsonb;is_real jsonb;salida jsonb;
BEGIN
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec:admin:continuidad:v1',0));
 -- Conservar el protocolo instalado de generación antes de observar las fuentes.
 PERFORM pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended('vec_contexto_actor_v1:mutacion_punteros_actuales:v2',0));
 c:=vec_contexto_actor_v1.preimagen_admin_interna_v1(p->>'cuenta_ref',p->>'persona_ref',p->>'perfil_ref',p->>'vinculo_ref');
 SELECT EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_actual a
  JOIN vec_contexto_actor_v1.vinculo_contexto_versiones v USING(vinculo_ref,version)
  WHERE v.cuenta_ref=p->>'cuenta_ref' AND v.persona_ref=p->>'persona_ref' AND v.estado='activo'
   AND v.procedencia_autoridad='autoridad_maestra_acreditada'
   AND pg_catalog.clock_timestamp()>=v.vigente_desde AND (p->>'vigente_hasta')::timestamptz<=v.vigente_hasta) INTO legado;
 IF NOT legado THEN
  SELECT pg_catalog.to_jsonb(t),pg_catalog.to_jsonb(f) INTO titularidad,fuente
  FROM vec_contexto_actor_v1.titularidad_cuenta_persona_v1 t
  JOIN vec_contexto_actor_v1.fuentes_iniciales_admin_v1 f USING(operacion_ref)
  JOIN vec_contexto_actor_v1.procedencias pr ON pr.procedencia_ref=t.fuente_ref
   AND pr.procedencia_version=t.fuente_version AND pr.procedencia_huella_sha256=t.fuente_sha256
   AND pr.procedencia_autoridad='autoridad_maestra_acreditada'
  WHERE t.cuenta_ref=p->>'cuenta_ref' AND t.persona_ref=p->>'persona_ref'
   AND t.version=1 AND t.version::text=p->>'cuenta_version' AND t.version::text=p->>'persona_version'
   AND t.alcance_fuente='sintetico_declarado' AND t.fuente_version=1
   AND pg_catalog.clock_timestamp()>=t.vigente_desde
   AND pg_catalog.clock_timestamp()<t.vigente_hasta AND (p->>'vigente_hasta')::timestamptz<=t.vigente_hasta
   AND f.plan->>'entorno'='desarrollo' AND f.plan->>'alcance_fuente'='sintetico_declarado'
   AND f.operacion_ref=f.plan->>'operacion_ref'
  FOR SHARE OF t,f,pr;
  IF titularidad IS NULL THEN
   RAISE EXCEPTION 'AUT38: PARO clave=titularidad actual=ausente_o_divergente esperado=fuente_CA33_vigente_sin_perfil' USING ERRCODE='40001';
  END IF;
  SELECT e.value INTO STRICT persona_fuente FROM pg_catalog.jsonb_array_elements(fuente#>'{plan,personas}') e
  WHERE e.value->>'persona_ref'=p->>'persona_ref';
  IF (titularidad->>'fuente_ref',titularidad->>'fuente_version',titularidad->>'fuente_sha256')
   IS DISTINCT FROM (persona_fuente#>>'{fuente_titularidad,referencia}',persona_fuente#>>'{fuente_titularidad,version}',persona_fuente#>>'{fuente_titularidad,huella_sha256}')
  OR c#>>'{cuenta,procedencia_ref}' IS DISTINCT FROM fuente#>>'{plan,procedencia,referencia}'
  OR c#>>'{cuenta,procedencia_version}' IS DISTINCT FROM fuente#>>'{plan,procedencia,version}'
  OR c#>>'{cuenta,procedencia_huella_sha256}' IS DISTINCT FROM fuente#>>'{plan,procedencia,huella_sha256}'
  OR c#>>'{persona,procedencia_ref}' IS DISTINCT FROM fuente#>>'{plan,procedencia,referencia}'
  OR c#>>'{persona,procedencia_version}' IS DISTINCT FROM fuente#>>'{plan,procedencia,version}'
  OR c#>>'{persona,procedencia_huella_sha256}' IS DISTINCT FROM fuente#>>'{plan,procedencia,huella_sha256}'
  OR fuente#>>'{recibo,operacion_ref}' IS DISTINCT FROM fuente->>'operacion_ref'
  OR fuente#>>'{recibo,plan_sha256}' IS DISTINCT FROM fuente->>'plan_sha256'
  OR fuente#>>'{recibo,aprobacion_ref}' IS DISTINCT FROM fuente->>'aprobacion_ref'
  OR fuente#>>'{recibo,huella_sha256}' IS DISTINCT FROM pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(((fuente->'recibo')-'huella_sha256')::text,'UTF8')),'hex')
  THEN RAISE EXCEPTION 'AUT38: PARO clave=fuente_titularidad actual=divergente esperado=procedencia_y_recibo_CA33_originales' USING ERRCODE='40001'; END IF;
  is_real:=vec_identidad_sesiones_v1.cotejar_recibo_fuentes_iniciales_admin_v1(fuente->>'operacion_ref',fuente->>'plan_sha256',fuente->>'aprobacion_ref');
  IF is_real IS DISTINCT FROM fuente->'recibo_is'
  OR NOT EXISTS(SELECT 1 FROM pg_catalog.jsonb_array_elements(is_real#>'{datos,personas}') e
   WHERE e.value->>'persona_ref'=p->>'persona_ref' AND p->>'cuenta_ref' IN(e.value->>'cuenta_ordinaria_ref',e.value->>'cuenta_privilegiada_ref'))
  THEN RAISE EXCEPTION 'AUT38: PARO clave=recibo_IS actual=divergente esperado=cuenta_Persona_originales' USING ERRCODE='40001'; END IF;
  titularidad:=titularidad||pg_catalog.jsonb_build_object('fuente',pg_catalog.jsonb_build_object(
   'operacion_ref',fuente->>'operacion_ref','plan_sha256',fuente->>'plan_sha256','aprobacion_ref',fuente->>'aprobacion_ref',
   'recibo_ca_ref',fuente#>>'{recibo,recibo_ref}','recibo_ca_sha256',fuente#>>'{recibo,huella_sha256}',
   'recibo_is_ref',is_real->>'recibo_ref','recibo_is_sha256',is_real->>'huella_sha256'));
 END IF;
 IF c#>>'{cuenta,version}' IS DISTINCT FROM p->>'cuenta_version'
 OR c#>>'{persona,version}' IS DISTINCT FROM p->>'persona_version'
 OR c->'perfil' IS DISTINCT FROM 'null'::jsonb OR c->'vinculo' IS DISTINCT FROM 'null'::jsonb
 OR c#>>'{persona,procedencia_ref}' IS DISTINCT FROM p#>>'{procedencia,referencia}'
 OR c#>>'{persona,procedencia_version}' IS DISTINCT FROM p#>>'{procedencia,version}'
 OR c#>>'{persona,procedencia_huella_sha256}' IS DISTINCT FROM p#>>'{procedencia,huella_sha256}'
 OR (p->>'vigente_hasta')::timestamptz>LEAST((c#>>'{persona,vigente_hasta}')::timestamptz,(c#>>'{cuenta,vigente_hasta}')::timestamptz)
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=p->>'perfil_ref')
 OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=p->>'vinculo_ref')
 OR (NOT legado AND titularidad IS NULL)
 THEN RAISE EXCEPTION 'AUT37: PARO clave=persona_preimagen actual=divergente esperado=fuente_viva_refs_nuevas' USING ERRCODE='40001'; END IF;
 ambitos:=vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(p->'ambitos',(p->>'vigente_hasta')::timestamptz);
 FOR sistema IN SELECT value FROM pg_catalog.jsonb_array_elements(p->'sistemas') LOOP
  sub:=vec_contexto_actor_v1.preimagen_admin_interna_v1(p->>'cuenta_ref',p->>'persona_ref',sistema->>'perfil_ref',sistema->>'vinculo_ref');
  IF sub->'perfil' IS DISTINCT FROM 'null'::jsonb OR sub->'vinculo' IS DISTINCT FROM 'null'::jsonb
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.perfil_versiones WHERE perfil_ref=sistema->>'perfil_ref')
  OR EXISTS(SELECT 1 FROM vec_contexto_actor_v1.vinculo_contexto_versiones WHERE vinculo_ref=sistema->>'vinculo_ref')
  OR (sistema->>'vigente_hasta')::timestamptz>(p->>'vigente_hasta')::timestamptz
  THEN RAISE EXCEPTION 'AUT37: PARO clave=sistemas_preimagen actual=divergente esperado=perfil_separado_nuevo' USING ERRCODE='40001'; END IF;
  sistemas:=sistemas||pg_catalog.jsonb_build_array(pg_catalog.jsonb_build_object('contexto',sub,
    'ambitos',vec_contexto_actor_v1.cotejar_ambitos_bootstrap_central_admin_v3(sistema->'ambitos',(sistema->>'vigente_hasta')::timestamptz)));
 END LOOP;
 SELECT pg_catalog.jsonb_agg(pg_catalog.to_jsonb(v) ORDER BY v.organizacion_ref) INTO organizaciones
 FROM vec_contexto_actor_v1.organizacion_actual a
 JOIN vec_contexto_actor_v1.organizacion_versiones v USING(organizacion_ref,version)
 WHERE v.organizacion_ref IN(SELECT x.value#>>'{valores,0}' FROM pg_catalog.jsonb_array_elements(p->'ambitos') x WHERE x.value->>'dimension'='organizacion_ref')
 OR v.organizacion_ref IN(SELECT x.value#>>'{valores,0}' FROM pg_catalog.jsonb_array_elements(p->'sistemas') sy,
   LATERAL pg_catalog.jsonb_array_elements(sy.value->'ambitos') x WHERE x.value->>'dimension'='organizacion_ref');
 salida:=pg_catalog.jsonb_build_object('esquema','vec.admin.bootstrap.persona.preimagen.v3','contexto',c,'sistemas',sistemas,'ambitos',ambitos,'organizaciones',organizaciones);
 IF NOT legado THEN salida:=salida||pg_catalog.jsonb_build_object('titularidad',titularidad); END IF;
 RETURN salida;
END $f$;

$definicion$;
 SELECT pg_catalog.to_jsonb(p)-'prosrc' INTO STRICT actual FROM pg_catalog.pg_proc p WHERE p.oid=funcion;
 IF actual IS DISTINCT FROM previa THEN
  RAISE EXCEPTION 'AUT38: PARO clave=OID_firma_propietario_config_ACL actual=divergente esperado=metadata_AUT37_original' USING ERRCODE='55000';
 END IF;
END $publicacion$;
COMMIT;
