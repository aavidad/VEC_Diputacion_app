\set ON_ERROR_STOP on
-- Vector estructural y negativo de AD235. Ejecutar como superusuario en una
-- copia con AD235 instalada; todo termina en ROLLBACK. La equivalencia byte a
-- byte con las consultas anteriores y preparar/verificar con la CLI real los
-- acredita TestGobiernoLectorAD235PostgreSQLPrivado y el ensayo de la rama.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE FUNCTION pg_temp.comprobar(caso text,condicion boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;
CREATE FUNCTION pg_temp.sqlstate(sentencia text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN EXECUTE sentencia; RETURN 'ejecutada';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE; END $f$;

-- 1. Grupo NOLOGIN sin pertenencias, sin TEMP ni CREATE.
SELECT pg_temp.comprobar('grupo',(SELECT NOT rolcanlogin AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolbypassrls AND rolconfig IS NULL
 FROM pg_roles WHERE rolname='vec_autorizacion_atestada_v3_lector_gobierno')
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member='vec_autorizacion_atestada_v3_lector_gobierno'::regrole)
 AND NOT has_database_privilege('vec_autorizacion_atestada_v3_lector_gobierno',current_database(),'TEMP')
 AND NOT has_database_privilege('vec_autorizacion_atestada_v3_lector_gobierno',current_database(),'CREATE')
 AND NOT has_schema_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3','CREATE'));

-- 2. Solo las dos fachadas; preimágenes de AD188/AD198 siguen solo del propietario.
SELECT pg_temp.comprobar('acl',
 has_function_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer)','EXECUTE')
 AND has_function_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text)','EXECUTE')
 AND NOT has_function_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)','EXECUTE')
 AND NOT has_function_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()','EXECUTE')
 AND NOT has_function_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)','EXECUTE')
 AND NOT has_function_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer)','EXECUTE')
 AND NOT has_table_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')
 AND NOT has_table_privilege('vec_autorizacion_atestada_v3_lector_gobierno','vec_autorizacion_atestada_v3.auditoria_consumo_v3','SELECT')
 AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN('vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)'::regprocedure,'vec_autorizacion_atestada_v3.preimagen_gobierno_usuarios_admin_v1()'::regprocedure)
  AND a.grantee<>p.proowner)
 AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid IN('vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(integer)'::regprocedure,'vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(text)'::regprocedure)
  AND (a.grantee=0 OR a.is_grantable)));

-- 3. Un LOGIN lector como el del kit: lee por las fachadas y nada más.
SELECT max(version) AS ultimo FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 \gset
CREATE ROLE prueba_ad235_lector LOGIN INHERIT NOSUPERUSER;
GRANT vec_autorizacion_atestada_v3_lector_gobierno TO prueba_ad235_lector WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SET LOCAL ROLE prueba_ad235_lector;
SELECT pg_temp.comprobar('instantanea_ultimo_conjunto',(SELECT x->>'pre_sha' ~ '^[0-9a-f]{64}$' AND NOT x ? 'secreto_hmac' AND (SELECT count(*) FROM jsonb_object_keys(x))=12
 FROM (SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(:ultimo) x) s));
SET LOCAL quote_all_identifiers=on;
SELECT pg_temp.comprobar('huella_independiente_de_sesion',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(0)')='ejecutada');
SET LOCAL quote_all_identifiers=off;
SELECT pg_temp.comprobar('instantanea_0',(SELECT x->>'pre_sha' ~ '^[0-9a-f]{64}$' FROM (SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(0) x) s));
SELECT pg_temp.comprobar('instantanea_conjunto_inexistente',(SELECT x->'pre_sha'='null'::jsonb FROM (SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(999) x) s));
SELECT pg_temp.comprobar('cadena',(SELECT x->>'esquema'='vec.auditoria.verificacion.gobierno-usuarios-admin.v1' AND x->'manifiesto'->>'cadena_id'='cadena:comun:interna'
 FROM (SELECT vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1('vec.auditoria.verificacion.gobierno-usuarios-admin.v1') x) s));
SELECT pg_temp.comprobar('rango_conjunto',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(-1)')='22023');
SELECT pg_temp.comprobar('esquema_libre',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.cadena_gobierno_admin_lectura_v1(''Otro esquema'')')='22023');
SELECT pg_temp.comprobar('sin_preimagen',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(5)')='42501');
SELECT pg_temp.comprobar('sin_tablas',pg_temp.sqlstate('SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version LIMIT 1')='42501');
SELECT pg_temp.comprobar('sin_escritura',pg_temp.sqlstate('INSERT INTO vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1(version,audiencias,segmentos) VALUES(999,''{a}'',''{b}'')')='42501');
SELECT pg_temp.comprobar('sin_efecto',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(''{}'',''x'',''{}'')')='42501');
SELECT pg_temp.comprobar('sin_create',pg_temp.sqlstate('CREATE TABLE vec_autorizacion_atestada_v3.prueba_ad235(x integer)')='42501');
RESET ROLE;
SELECT pg_temp.comprobar('login_sin_temp',NOT has_database_privilege('prueba_ad235_lector',current_database(),'TEMP') AND NOT has_database_privilege('prueba_ad235_lector',current_database(),'CREATE'));

-- 4. Guarda de huella: si la preimagen envuelta cambia, la fachada para (55000).
-- Se cambia la definición de la preimagen dentro de este ROLLBACK con un ajuste
-- de configuración (sale en pg_get_functiondef), sin reconstruirla.
ALTER FUNCTION vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer) SET work_mem='64kB';
SELECT pg_temp.comprobar('huella_envuelta',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1('||:ultimo||')')='55000');
SELECT pg_temp.comprobar('huella_otra_intacta',pg_temp.sqlstate('SELECT vec_autorizacion_atestada_v3.instantanea_gobierno_admin_lectura_v1(0)')='ejecutada');

-- 5. Guarda de pg_auth_members de AD198: se llama a la función instalada,
-- exigir_operador_gobierno_capacidades_admin_v1, con cada LOGIN como
-- session_user (EXECUTE a PUBLIC solo dentro de este ROLLBACK). Un LOGIN del
-- operador que además sea lector para en clave=LOGIN; el que solo es operador
-- pasa esa guarda y para después, por falta de configuración aprobada.
CREATE ROLE prueba_ad235_mixto LOGIN INHERIT NOSUPERUSER;
GRANT vec_gobierno_capacidades_admin_operador TO prueba_ad235_mixto WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_autorizacion_atestada_v3_lector_gobierno TO prueba_ad235_mixto WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_ad235_operador LOGIN INHERIT NOSUPERUSER;
GRANT vec_gobierno_capacidades_admin_operador TO prueba_ad235_operador WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
-- A PUBLIC: una concesión nominal dejaría un pg_shdepend que la propia guarda rechaza.
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1() TO PUBLIC;
CREATE FUNCTION pg_temp.guarda_operador() RETURNS text LANGUAGE plpgsql AS $f$
BEGIN PERFORM vec_autorizacion_atestada_v3.exigir_operador_gobierno_capacidades_admin_v1(); RETURN 'admitido';
EXCEPTION WHEN OTHERS THEN RETURN SQLSTATE||' '||SQLERRM; END $f$;
SET SESSION AUTHORIZATION prueba_ad235_mixto;
SELECT pg_temp.guarda_operador() AS mixto \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad235_operador;
SELECT pg_temp.guarda_operador() AS operador \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('ad198_rechaza_mixto',:'mixto' LIKE '42501 %clave=LOGIN%');
SELECT pg_temp.comprobar('ad198_admite_operador',:'operador' LIKE '42501 %clave=configuracion%');
SELECT pg_temp.comprobar('lector_sin_aplicar',NOT has_function_privilege('prueba_ad235_lector','vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)','EXECUTE'));
ROLLBACK;
