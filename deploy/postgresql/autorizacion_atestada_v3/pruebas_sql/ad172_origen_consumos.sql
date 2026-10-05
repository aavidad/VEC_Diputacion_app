\set ON_ERROR_STOP on
-- Sólo clon sintético desechable, después de AD172. Ningún consumo firmado
-- se simula: se comprueban contratos, configuración, familias y encuadre.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL statement_timeout='30s';
DO $contratos$
DECLARE r record; f oid;
BEGIN
 f:='vec_autorizacion_atestada_v3.resolver_origen_consumo_v1(text,text,text)'::regprocedure;
 IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND NOT p.prosecdef AND p.provolatile='s'
   AND p.proconfig=ARRAY['search_path=pg_catalog'])
 OR NOT EXISTS(SELECT 1 FROM pg_class c WHERE c.oid=
   'vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1'::regclass
   AND c.relrowsecurity AND c.relforcerowsecurity)
 THEN RAISE EXCEPTION 'AD172: PARO clave=contrato actual=divergente esperado=resolver_invocador_RLS_forzada'; END IF;
 FOR r IN SELECT rolname FROM pg_roles WHERE rolcanlogin AND NOT rolsuper LOOP
  IF has_function_privilege(r.rolname,f,'EXECUTE')
   OR has_table_privilege(r.rolname,'vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1','SELECT,INSERT,UPDATE,DELETE,TRUNCATE')
  THEN RAISE EXCEPTION 'AD172: PARO clave=ACL_login actual=amplia esperado=sin_acceso_directo'; END IF;
 END LOOP;
END $contratos$;

DO $familias$
DECLARE condicion text; base jsonb; v jsonb; permitido boolean; campo text;
BEGIN
 SELECT substr(pg_get_constraintdef(oid,false),8,length(pg_get_constraintdef(oid,false))-8)
 INTO STRICT condicion FROM pg_constraint
 WHERE conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND conname='auditoria_tipo_disjunto_v3' AND convalidated;
 SELECT to_jsonb(a) INTO STRICT base
 FROM (SELECT * FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
       WHERE tipo_registro='consumo_confirmado' ORDER BY secuencia LIMIT 1) a;
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING base;
 IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD172: PARO clave=v1 actual=rechazado esperado=permitido'; END IF;
 v:=base||jsonb_build_object('tipo_registro','consumo_confirmado_v2','version_consumo',2,'proceso','rrhh_prueba','canal','interna_corporativa');
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v;
 IF permitido IS DISTINCT FROM true THEN RAISE EXCEPTION 'AD172: PARO clave=v2 actual=rechazado esperado=permitido'; END IF;
 FOREACH campo IN ARRAY ARRAY['proceso','canal','version_consumo','decision_ref','efecto_ref','huella_efecto_sha256'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object(campo,NULL);
  -- Un CHECK también permite NULL: se exige FALSE, no sólo «distinto de TRUE».
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD172: PARO clave=v2_sin_% actual=aceptable esperado=false',campo; END IF;
 END LOOP;
 FOREACH campo IN ARRAY ARRAY['actor_ref','perfil_activo_ref','registro_contexto_ref','contexto_sha256','procedencia_sha256','autenticacion_ref','sesion_ref','autenticacion_sha256','accion','modulo_id','recurso_ref','finalidad_ref','resultado','motivo_ref','correlacion_ref','vinculo_sha256','intento_ref','intento_material_sha256','evento_ref','evento_material_sha256','fuente_ref','fuente_sha256','operador_login','plan_sha256','aprobacion_ref'] LOOP
  EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING v||jsonb_build_object(campo,'mezcla');
  IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD172: PARO clave=v2_con_% actual=aceptable esperado=false',campo; END IF;
 END LOOP;
 EXECUTE 'SELECT ('||condicion||') FROM jsonb_populate_record(NULL::vec_autorizacion_atestada_v3.auditoria_consumo_v3,$1)' INTO permitido USING base||jsonb_build_object('version_consumo',2);
 IF permitido IS DISTINCT FROM false THEN RAISE EXCEPTION 'AD172: PARO clave=v1_version actual=aceptable esperado=false'; END IF;
END $familias$;

CREATE ROLE ad172_login_sintetico LOGIN INHERIT;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1
 (login_nombre,audiencia_consumo,operacion,proceso,canal_permitido)
 VALUES ('ad172_login_sintetico','audiencia:ad172:prueba','operacion:ad172:prueba','rrhh_prueba','interna_corporativa');
DO $inmutable$
DECLARE rechazados integer:=0;
BEGIN
 BEGIN UPDATE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 SET proceso='otro_proceso' WHERE login_nombre='ad172_login_sintetico';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazados:=rechazados+1; END;
 BEGIN DELETE FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1 WHERE login_nombre='ad172_login_sintetico';
 EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazados:=rechazados+1; END;
 BEGIN TRUNCATE vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1;
 EXCEPTION WHEN object_not_in_prerequisite_state THEN rechazados:=rechazados+1; END;
 IF rechazados<>3 THEN RAISE EXCEPTION 'AD172: PARO clave=inmutabilidad actual=% esperado=3',rechazados; END IF;
END $inmutable$;
RESET ROLE;
-- Fachada exclusivamente de prueba para observar SECURITY INVOKER dentro
-- del propietario. No reemplaza el núcleo ni acredita su consumo positivo.
CREATE SCHEMA ad172_ensayo AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION ad172_ensayo.comprobar_resolver() RETURNS boolean
 LANGUAGE sql SECURITY DEFINER SET search_path=pg_catalog AS $$
 SELECT vec_autorizacion_atestada_v3.resolver_origen_consumo_v1('audiencia:ad172:prueba','operacion:ad172:prueba','interna_corporativa')='rrhh_prueba'
 AND vec_autorizacion_atestada_v3.resolver_origen_consumo_v1('audiencia:ad172:otra','operacion:ad172:prueba','interna_corporativa') IS NULL
 AND vec_autorizacion_atestada_v3.resolver_origen_consumo_v1('audiencia:ad172:prueba','operacion:ad172:otra','interna_corporativa') IS NULL
 AND vec_autorizacion_atestada_v3.resolver_origen_consumo_v1('audiencia:ad172:prueba','operacion:ad172:prueba','externa_personal') IS NULL
 $$;
REVOKE ALL ON FUNCTION ad172_ensayo.comprobar_resolver() FROM PUBLIC;
GRANT USAGE ON SCHEMA ad172_ensayo TO ad172_login_sintetico;
GRANT EXECUTE ON FUNCTION ad172_ensayo.comprobar_resolver() TO ad172_login_sintetico;
RESET ROLE;
SET SESSION AUTHORIZATION ad172_login_sintetico;
DO $resolver$
BEGIN
 IF ad172_ensayo.comprobar_resolver() IS DISTINCT FROM true
 THEN RAISE EXCEPTION 'AD172: PARO clave=resolver actual=divergente esperado=terna_y_canal_exactos'; END IF;
 BEGIN
  PERFORM 1 FROM vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1;
  RAISE EXCEPTION 'AD172: PARO clave=lectura_directa actual=permitida esperado=denegada';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $resolver$;
SET ROLE ad172_login_sintetico;
DO $role_explicito$
BEGIN
 IF ad172_ensayo.comprobar_resolver() IS TRUE
 THEN RAISE EXCEPTION 'AD172: PARO clave=role_explicito actual=aceptado esperado=rechazado'; END IF;
END $role_explicito$;
RESET ROLE;
RESET SESSION AUTHORIZATION;
ALTER ROLE ad172_login_sintetico NOINHERIT;
SET SESSION AUTHORIZATION ad172_login_sintetico;
DO $noinherit$
BEGIN
 IF ad172_ensayo.comprobar_resolver() IS TRUE
 THEN RAISE EXCEPTION 'AD172: PARO clave=noinherit actual=aceptado esperado=rechazado'; END IF;
END $noinherit$;
RESET SESSION AUTHORIZATION;
ALTER ROLE ad172_login_sintetico INHERIT SUPERUSER;
SET SESSION AUTHORIZATION ad172_login_sintetico;
DO $superusuario$
BEGIN
 IF ad172_ensayo.comprobar_resolver() IS TRUE
 THEN RAISE EXCEPTION 'AD172: PARO clave=superusuario actual=aceptado esperado=rechazado'; END IF;
END $superusuario$;
RESET SESSION AUTHORIZATION;

DO $vector$
DECLARE huella text;
BEGIN
 SELECT encode(sha256(string_agg(vec_autorizacion_atestada_v3.encuadrar_mac(valor),''::bytea ORDER BY orden)),'hex') INTO huella
 FROM unnest(ARRAY['consumo_confirmado_v2','2','7',repeat('0',64),'decision:ad172:prueba','efecto:ad172:prueba',repeat('a',64),repeat('b',64),'rrhh_prueba','interna_corporativa']) WITH ORDINALITY v(valor,orden);
 IF huella<>'82d759097556f17a8c4549006433ba250c388bcdc7d1cc9a2a76c43f454fcec9'
 THEN RAISE EXCEPTION 'AD172: PARO clave=vector actual=% esperado=82d759097556f17a8c4549006433ba250c388bcdc7d1cc9a2a76c43f454fcec9',huella; END IF;
END $vector$;
ROLLBACK;
SELECT 'AD172-CONTRATOS-FAMILIAS-CONFIGURACION-VECTOR-OK';
