\set ON_ERROR_STOP on
-- Vector estructural y negativo de AD190. Ejecutar como superusuario en un clon
-- con AD190 instalada; todo termina en ROLLBACK. El consumo positivo exige una
-- decisión firmada por el emisor real y se acredita en el recorrido con vec-admin.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE FUNCTION pg_temp.comprobar(caso text,condicion boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;

-- 1. Fachada, función de LOGIN, núcleo y audiencia.
SELECT pg_temp.comprobar('acl_fachada',
 has_function_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_admin_perfiles_lote_ejecutor','vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')
 AND NOT has_function_privilege('vec_autorizacion_propietario','vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1()','EXECUTE')
 AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid='vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
  AND a.grantee=0));
SELECT pg_temp.comprobar('grupo_ejecutor',EXISTS(SELECT 1 FROM pg_roles WHERE rolname='vec_admin_perfiles_lote_ejecutor'
 AND NOT(rolcanlogin OR rolinherit OR rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls))
 AND has_database_privilege('vec_admin_perfiles_lote_ejecutor',current_database(),'CONNECT'));
SELECT pg_temp.comprobar('nucleo',(SELECT (length(p.prosrc)-length(replace(p.prosrc,'lote_perfiles_admin','')))/length('lote_perfiles_admin')=4
 AND strpos(p.prosrc,'acreditar_perfil_aplicacion_lote_ordinario_v1')>0 AND strpos(p.prosrc,'servicios_certificados_propios')>0
 AND strpos(p.prosrc,'resolver_origen_consumo_v1')>0 AND strpos(p.prosrc,'''recuperacion_firmas_r5_ct_v2''')>0
 AND strpos(p.prosrc,'''gobierno_plan_nominal_firma_ct''')>0
 FROM pg_proc p WHERE p.oid='vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure));
SELECT pg_temp.comprobar('audiencia',(SELECT count(*) FROM pg_constraint c WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
 AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.convalidated
 AND strpos(pg_get_constraintdef(c.oid,false),'vec_autorizacion.administracion_perfiles.lote_ordinario.v1')>0
 AND strpos(pg_get_constraintdef(c.oid,false),'vec_personal.servicios_certificados.v1')>0
 AND strpos(pg_get_constraintdef(c.oid,false),'vec_contratacion_temporal.firmas_r5.recuperar.v2')>0
 AND strpos(pg_get_constraintdef(c.oid,false),'vec_catalogos_configurables.plan_nominal_firma.gobierno.v1')>0)=1);

-- 2. Función de LOGIN: sólo un LOGIN mínimo, miembro único del grupo.
CREATE FUNCTION pg_temp.login_valido() RETURNS boolean LANGUAGE sql SECURITY DEFINER AS $f$
 SELECT vec_autorizacion_atestada_v3.login_lote_perfiles_admin_valido_v1() $f$;
CREATE FUNCTION pg_temp.consumir_basura() RETURNS text LANGUAGE plpgsql SECURITY DEFINER AS $f$
BEGIN
 PERFORM * FROM vec_autorizacion_atestada_v3.consumir_lote_perfiles_admin_v3_atestada(
  convert_to('{}','UTF8'),convert_to('{}','UTF8'),'\x00','\x00',1,1,'\x00','\x00','\x00',decode(repeat('00',44),'hex'));
 RETURN 'consumido';
EXCEPTION WHEN insufficient_privilege THEN RETURN '42501'; WHEN OTHERS THEN RETURN SQLSTATE;
END $f$;
CREATE ROLE prueba_ad190_bueno LOGIN;
GRANT vec_admin_perfiles_lote_ejecutor TO prueba_ad190_bueno WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_ad190_doble LOGIN;
GRANT vec_admin_perfiles_lote_ejecutor TO prueba_ad190_doble WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_admin_usuarios_lector TO prueba_ad190_doble WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_ad190_set LOGIN;
GRANT vec_admin_perfiles_lote_ejecutor TO prueba_ad190_set WITH INHERIT TRUE, SET TRUE, ADMIN FALSE;
CREATE ROLE prueba_ad190_ajeno LOGIN;
SELECT pg_temp.comprobar('login_superusuario_rechazado',pg_temp.login_valido() IS NOT TRUE);
SET SESSION AUTHORIZATION prueba_ad190_bueno;
SELECT pg_temp.login_valido() AS l_bueno \gset
SELECT pg_temp.consumir_basura() AS c_bueno \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad190_doble;
SELECT pg_temp.login_valido() AS l_doble \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad190_set;
SELECT pg_temp.login_valido() AS l_set \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad190_ajeno;
SELECT pg_temp.login_valido() AS l_ajeno \gset
SELECT pg_temp.consumir_basura() AS c_ajeno \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('login_bueno',:'l_bueno'::boolean);
SELECT pg_temp.comprobar('login_dos_grupos_rechazado',NOT :'l_doble'::boolean);
SELECT pg_temp.comprobar('login_con_set_rechazado',NOT :'l_set'::boolean);
SELECT pg_temp.comprobar('login_ajeno_rechazado',NOT :'l_ajeno'::boolean);

-- 3. Material inválido o sesión ajena: 42501 antes de entrar en el núcleo.
SELECT pg_temp.comprobar('material_invalido_42501',:'c_bueno'='42501');
SELECT pg_temp.comprobar('sesion_ajena_42501',:'c_ajeno'='42501');
SELECT pg_temp.comprobar('superusuario_42501',pg_temp.consumir_basura()='42501');

-- 4. Ningún intento negativo dejó claves de capacidad de la audiencia nueva.
SELECT pg_temp.comprobar('sin_consumos',NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version
 WHERE audiencia_consumo='vec_autorizacion.administracion_perfiles.lote_ordinario.v1'));
ROLLBACK;
