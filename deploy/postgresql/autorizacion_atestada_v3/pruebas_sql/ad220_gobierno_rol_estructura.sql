\set ON_ERROR_STOP on
-- Ejecutar tras AD219, AUT59 y AD220 en un clon PG18 aislado. Solo lectura.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='10s';
DO $prueba$
DECLARE
 nucleo oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 consumidor oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_gobierno_rol_nuevo_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 intento oid:=to_regprocedure('vec_autorizacion_atestada_v3.registrar_intento_gobierno_rol_nuevo_v1(jsonb)');
 sesion oid:=to_regprocedure('vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1()');
 cuerpo text;audiencias text;familia text;
 f oid;
BEGIN
 IF nucleo IS NULL OR consumidor IS NULL OR intento IS NULL OR sesion IS NULL
 OR to_regrole('vec_admin_gobierno_roles_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'AD220 prueba: funciones o grupo ausentes'; END IF;
 SELECT pg_get_functiondef(nucleo) INTO STRICT cuerpo;
 IF (length(cuerpo)-length(replace(cuerpo,'p_perfil_mutacion IS NOT DISTINCT FROM ''gobierno_rol_nuevo''','')))/
    length('p_perfil_mutacion IS NOT DISTINCT FROM ''gobierno_rol_nuevo''')<>2
 OR strpos(cuerpo,'vec_autorizacion_atestada_v3.login_gobierno_rol_nuevo_valido_v1() IS TRUE')=0
 OR strpos(cuerpo,'vec_autorizacion.acreditar_gobierno_rol_nuevo_v1(d) IS TRUE')=0
 OR strpos(cuerpo,'p_perfil_mutacion IS DISTINCT FROM ''gobierno_rol_nuevo''')=0
 THEN RAISE EXCEPTION 'AD220 prueba: tres guardas nominales del núcleo divergentes'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT audiencias FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(audiencias,'vec_autorizacion.gobierno_rol_nuevo.propuesta.v1')=0
 OR strpos(audiencias,'vec_autorizacion.gobierno_rol_nuevo.cierre.v1')=0
 THEN RAISE EXCEPTION 'AD220 prueba: audiencias ausentes'; END IF;
 SELECT pg_get_constraintdef(c.oid,false) INTO STRICT familia FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass
 AND c.conname='auditoria_tipo_disjunto_v4' AND c.contype='c' AND c.convalidated;
 IF strpos(familia,'intento_gobierno_rol_nuevo')=0
 OR strpos(familia,'gobierno_rol_nuevo_solicitud_sha256 IS NOT NULL')=0
 OR strpos(familia,'resultado IS NOT NULL')=0
 OR strpos(familia,'catalogo_acciones_detalle IS NULL')=0
 THEN RAISE EXCEPTION 'AD220 prueba: familia de auditoría no disjunta'; END IF;
 FOREACH f IN ARRAY ARRAY[consumidor,intento] LOOP
  IF NOT EXISTS(SELECT 1 FROM pg_proc p WHERE p.oid=f
   AND p.proowner='vec_autorizacion_atestada_v3_propietario'::regrole
   AND p.prosecdef AND p.provolatile='v' AND p.proparallel='u')
  OR (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
  OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
    aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f
    AND (a.grantee NOT IN(p.proowner,'vec_autorizacion_propietario'::regrole)
      OR a.grantor<>p.proowner OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
  OR has_function_privilege('vec_admin_gobierno_roles_ejecutor',f,'EXECUTE')
  THEN RAISE EXCEPTION 'AD220 prueba: ACL o propietario divergente para %',f; END IF;
 END LOOP;
 IF has_function_privilege('vec_admin_gobierno_roles_ejecutor',sesion,'EXECUTE')
 OR EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL
   aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=sesion
   AND (a.grantee<>p.proowner OR a.grantor<>p.proowner
     OR a.privilege_type<>'EXECUTE' OR a.is_grantable))
 OR NOT EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname='vec_admin_gobierno_roles_ejecutor'
   AND NOT(r.rolcanlogin OR r.rolinherit OR r.rolsuper OR r.rolcreatedb
    OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls))
 OR EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_admin_gobierno_roles_ejecutor'::regrole)
 THEN RAISE EXCEPTION 'AD220 prueba: grupo con EXECUTE directo o herencia'; END IF;
END $prueba$;
ROLLBACK;
\echo AD220_ESTRUCTURA_OK
