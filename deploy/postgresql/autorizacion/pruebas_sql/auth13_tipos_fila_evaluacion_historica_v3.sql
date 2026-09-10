\set ON_ERROR_STOP on
-- Auth13: catálogo real, sin negocio, permisos ni funciones persistentes.
-- Ejecutable antes/después de Auth13; deriva ambas imágenes exactas del catálogo.
BEGIN READ ONLY;
SET LOCAL search_path=pg_catalog;
SET LOCAL statement_timeout='10s';
DO $regresion_auth13$
DECLARE
 origen text:=$origen$            -- Sin ACL propia ni acceso a su esquema, no concede acceso a datos.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
              AND NOT pg_catalog.has_schema_privilege(login_oid,n.oid,'USAGE')
$origen$;
 destino text:=$destino$            -- Sin ACL propia, el tipo fila no concede acceso a datos;
            -- el guard anterior rechaza todo acceso a tablas y columnas.
            -- Los tipos independientes y cualquier concesión explícita siguen
            -- rechazados, también si pertenecen a otro esquema cerrado.
            AND NOT (
              t.typtype='c' AND t.typacl IS NULL
$destino$;
 anterior text; corregida text; guard_anterior text; guard_corregido text; consulta text;
 rechaza_anterior boolean; rechaza_corregida boolean; rechaza_negativa boolean;
 privilegio text; excepcion boolean; caso record;
BEGIN
 SELECT prosrc INTO STRICT anterior FROM pg_proc
  WHERE oid='vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)'::regprocedure;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex')='c59d048bdc170e7e28a3c56362e8abcb1131ccd22e54fc9228f18610392d3785' THEN
  anterior:=replace(anterior,destino,origen);
 END IF;
 IF encode(sha256(convert_to(anterior,'UTF8')),'hex') IS DISTINCT FROM '72932131b8bb7bc17cef6cc701870c68023d0f41428da07eca7a9cb89ccdc0ff' THEN
  RAISE EXCEPTION 'Auth13 regresión: preimagen inesperada'; END IF;
 corregida:=replace(anterior,origen,destino);
 IF encode(sha256(convert_to(corregida,'UTF8')),'hex') IS DISTINCT FROM 'c59d048bdc170e7e28a3c56362e8abcb1131ccd22e54fc9228f18610392d3785'
  OR replace(anterior,origen,'') IS DISTINCT FROM replace(corregida,destino,'') THEN
  RAISE EXCEPTION 'Auth13 regresión: otros guards modificados'; END IF;
 guard_anterior:='login_oid IS NULL'||split_part(split_part(anterior,'    IF login_oid IS NULL',2),
  E' THEN\n        RAISE EXCEPTION USING ERRCODE = ''42501''',1);
 guard_corregido:=replace(guard_anterior,origen,destino);
 consulta:=$consulta$WITH login AS (
 SELECT * FROM pg_roles WHERE rolname='vec_inc_v2_historia_evaluacion_20260910'), grupo AS (
 SELECT * FROM pg_roles WHERE rolname='vec_autorizacion_evaluacion_historica_lector'), contexto AS (
 SELECT login.oid AS login_oid,grupo.oid AS runtime_oid,'vec_autorizacion'::regnamespace::oid AS esquema_oid,
  (SELECT oid FROM pg_database WHERE datname=current_database()) AS base_oid,
  ARRAY['vec_autorizacion.leer_evaluacion_original_contexto_actor_v3(text,text,text)'::regprocedure::oid] AS funciones,
  (SELECT count(*) FROM pg_auth_members WHERE member=login.oid) AS membresias FROM login,grupo)
 SELECT (%s) FROM login,grupo,contexto$consulta$;
 EXECUTE format(consulta,guard_anterior) INTO STRICT rechaza_anterior;
 EXECUTE format(consulta,guard_corregido) INTO STRICT rechaza_corregida;
 IF rechaza_anterior IS DISTINCT FROM true OR rechaza_corregida IS DISTINCT FROM false THEN
  RAISE EXCEPTION 'Auth13 regresión: contraste del guard real inesperado'; END IF;
 -- Predicado de excepción exacto: sólo tipo fila sin ACL de una relación admitida.
 FOR caso IN SELECT * FROM (VALUES
  ('c',true,true,true), ('c',false,true,false), ('c',true,false,false),
  ('d',true,false,false), ('e',true,false,false), ('m',true,false,false), ('r',true,false,false)
 ) AS casos(tipo,acl_nula,relacion_admitida,esperada) LOOP
  excepcion:=caso.tipo='c' AND caso.acl_nula AND caso.relacion_admitida;
  IF excepcion IS DISTINCT FROM caso.esperada THEN
   RAISE EXCEPTION 'Auth13 regresión: excepción admite tipo independiente o ACL explícita'; END IF;
 END LOOP;
 -- Mantener el guard completo: simular sólo su booleano de privilegio, sin GRANT.
 -- La presencia de acceso de tabla o columna debe seguir denegando al LOGIN.
 FOREACH privilegio IN ARRAY ARRAY[
  'pg_catalog.has_table_privilege(login_oid,c.oid,''SELECT'')',
  'pg_catalog.has_any_column_privilege(login_oid,c.oid,''SELECT'')'] LOOP
  IF position(privilegio IN guard_corregido)=0 THEN
   RAISE EXCEPTION 'Auth13 regresión: falta guard tabla/columna'; END IF;
  EXECUTE format(consulta,replace(guard_corregido,privilegio,'true')) INTO STRICT rechaza_negativa;
  IF rechaza_negativa IS DISTINCT FROM true THEN
   RAISE EXCEPTION 'Auth13 regresión: privilegio tabla/columna no rechazado'; END IF;
 END LOOP;
END $regresion_auth13$;
ROLLBACK;
