\set ON_ERROR_STOP on
-- Vector estructural y negativo de AD198. Ejecutar como superusuario en un clon
-- con AD198 instalada; todo termina en ROLLBACK. La publicación favorable la
-- acredita el ensayo con vec-gobierno-usuarios-admin (conjunto_capacidades 1).
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
CREATE FUNCTION pg_temp.comprobar(caso text,condicion boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;

-- 1. Conjunto 1 cerrado e inmutable.
SELECT pg_temp.comprobar('conjunto_1',(SELECT audiencias=ARRAY['vec.admin.usuarios.listar.v1','vec.admin.usuarios.consultar.v1','vec_autorizacion.administracion_perfiles.lote_ordinario.v1']
 AND segmentos=ARRAY['usuarios:listar','usuarios:consultar','perfiles:lote']
 FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 WHERE version=1)
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1)=1);
CREATE FUNCTION pg_temp.mutar_conjunto() RETURNS text LANGUAGE plpgsql AS $f$
BEGIN UPDATE vec_autorizacion_atestada_v3.conjunto_audiencias_capacidad_admin_v1 SET segmentos=segmentos WHERE version=1; RETURN 'mutado';
EXCEPTION WHEN OTHERS THEN RETURN 'rechazado'; END $f$;
SELECT pg_temp.comprobar('conjunto_inmutable',pg_temp.mutar_conjunto()='rechazado');

-- 2. ACL: el grupo sólo ejecuta aprovisionar; nada de tablas ni funciones privadas.
SELECT pg_temp.comprobar('acl',
 has_function_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)','EXECUTE')
 AND NOT has_function_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.efecto_gobierno_capacidades_admin_v1(text,text,text)','EXECUTE')
 AND NOT has_function_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(integer)','EXECUTE')
 AND NOT has_function_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.aprovisionar_gobierno_usuarios_admin_v1(text,text,text)','EXECUTE')
 AND NOT has_function_privilege('vec_gobierno_usuarios_admin_operador','vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)','EXECUTE')
 AND NOT has_table_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1','SELECT')
 AND NOT has_table_privilege('vec_gobierno_capacidades_admin_operador','vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT')
 AND NOT EXISTS(SELECT 1 FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE p.oid='vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1(text,text,text)'::regprocedure AND a.grantee=0));

-- 3. Preimagen del conjunto: tres audiencias y sus claves (sin secreto).
SELECT pg_temp.comprobar('preimagen',(SELECT x->>'conjunto_version'='1' AND jsonb_array_length(x->'audiencias')=3
 AND NOT jsonb_path_exists(x,'$.claves[*].secreto_hmac') AND x ? 'configuracion' AND x ? 'orden_claves'
 FROM (SELECT vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(1) x) s));
SELECT pg_temp.comprobar('preimagen_conjunto_inexistente',vec_autorizacion_atestada_v3.preimagen_gobierno_capacidades_admin_v1(99) IS NULL);

-- 4. Intentos denegados: sin configuración, con dos grupos y con plan ajeno.
CREATE ROLE prueba_ad198_bueno LOGIN;
GRANT vec_gobierno_capacidades_admin_operador TO prueba_ad198_bueno WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
CREATE ROLE prueba_ad198_doble LOGIN;
GRANT vec_gobierno_capacidades_admin_operador TO prueba_ad198_doble WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
GRANT vec_gobierno_usuarios_admin_operador TO prueba_ad198_doble WITH INHERIT TRUE, SET FALSE, ADMIN FALSE;
SELECT secuencia AS aud0 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id \gset
SELECT count(*) AS claves0 FROM vec_autorizacion_atestada_v3.clave_capacidad_version \gset
SELECT count(*) AS operaciones0 FROM vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1 \gset
SET SESSION AUTHORIZATION prueba_ad198_bueno;
SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1('{}',repeat('0',64),'{}')->>'estado' AS sin_config \gset
RESET SESSION AUTHORIZATION;
INSERT INTO vec_autorizacion_atestada_v3.config_gobierno_capacidades_admin_v1 VALUES
 ('prueba_ad198_bueno',1,encode(sha256(convert_to('{"version":1}','UTF8')),'hex'),repeat('a',64),encode(sha256(convert_to('{"claves":[]}','UTF8')),'hex'),now()-interval '1 minute',now()+interval '1 hour','desarrollo'),
 ('prueba_ad198_doble',1,repeat('b',64),repeat('a',64),repeat('c',64),now()-interval '1 minute',now()+interval '1 hour','desarrollo');
SET SESSION AUTHORIZATION prueba_ad198_bueno;
SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1('{"version":1}',encode(sha256(convert_to('{"version":1}','UTF8')),'hex'),'{"claves":[]}')->>'estado' AS plan_ajeno \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_ad198_doble;
SELECT vec_autorizacion_atestada_v3.aprovisionar_gobierno_capacidades_admin_v1('{}',repeat('b',64),'{}')->>'estado' AS dos_grupos \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('sin_configuracion_denegado',:'sin_config'='denegado');
SELECT pg_temp.comprobar('plan_ajeno_denegado',:'plan_ajeno'='denegado');
SELECT pg_temp.comprobar('dos_grupos_denegado',:'dos_grupos'='denegado');
SELECT pg_temp.comprobar('intentos_auditados',(SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3
 WHERE secuencia>:aud0 AND tipo_registro='intento_gobierno_usuarios_admin' AND resultado='denegado')=3
 AND NOT EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='gobierno_usuarios_admin'));
SELECT pg_temp.comprobar('sin_claves_nuevas',(SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version)=:claves0);
-- Cada intento denegado queda anotado como de AD198, ligado a su registro común.
SELECT pg_temp.comprobar('operaciones_ad198',(SELECT count(*) FROM vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1)=:operaciones0+3
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1 o
  JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a ON a.auditoria_ref=o.auditoria_ref
  WHERE a.secuencia>:aud0 AND o.tipo='intento' AND o.resultado='denegado' AND cardinality(o.clave_ids)=0
   AND o.funcion='aprovisionar_gobierno_capacidades_admin_v1' AND a.gobierno_usuarios_solicitud_sha256=o.solicitud_sha256)=3
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1 WHERE operador_login='prueba_ad198_bueno' AND conjunto_version IS NULL)=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1 WHERE operador_login IN('prueba_ad198_bueno','prueba_ad198_doble') AND conjunto_version=1)=2);
CREATE FUNCTION pg_temp.mutar_operacion() RETURNS text LANGUAGE plpgsql AS $f$
BEGIN DELETE FROM vec_autorizacion_atestada_v3.operacion_gobierno_capacidades_admin_v1; RETURN 'borrado';
EXCEPTION WHEN OTHERS THEN RETURN 'rechazado'; END $f$;
SELECT pg_temp.comprobar('operaciones_inmutables',pg_temp.mutar_operacion()='rechazado');
ROLLBACK;
