\set ON_ERROR_STOP on
-- Vector de AUT49/AD196. Se ejecuta como superusuario sobre un clon con
-- AUT24, AD193, AD196 y AUT49 instaladas y roles ordinarios publicados de
-- desarrollo. Todo ocurre en una transacción que termina en ROLLBACK.
-- Cada caso imprime «OK <caso>»; cualquier discrepancia aborta con error.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT secuencia AS aud0 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id \gset

-- Plan con dos roles ordinarios publicados, tomados del propio clon.
CREATE FUNCTION pg_temp.perfil(ref text) RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('version_rol_ref',v.version_rol_ref,'version_rol_sha256',v.huella_sha256,
  'control_revision',c.revision,'control_sha256',c.huella_sha256,'unidad_requerida',false,
  'ambitos_fijos',jsonb_build_array(jsonb_build_object('clave','organizacion_ref','valores',jsonb_build_array('organizacion:desarrollo:dipgra'))),
  'vigente_desde',to_char(date_trunc('second',now()) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'vigente_hasta',to_char(date_trunc('second',now()+interval '365 days') AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'duracion_propuesta_segundos',86400)
 FROM vec_autorizacion.version_rol v JOIN vec_autorizacion.control_vigencia_version_rol_actual ca USING(version_rol_ref)
 JOIN vec_autorizacion.control_vigencia_version_rol c ON c.version_rol_ref=ca.version_rol_ref AND c.revision=ca.revision
 WHERE v.version_rol_ref=ref
$f$;
CREATE FUNCTION pg_temp.plan(op text,perfiles jsonb,preparado interval DEFAULT interval '1 minute',caduca interval DEFAULT interval '1 hour') RETURNS text LANGUAGE sql AS $f$
 SELECT jsonb_build_object('esquema','vec.admin.perfiles-asignables.plan.v1','operacion_ref',op,
  'preparado_en',to_char(date_trunc('second',now()-preparado) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'caduca_en',to_char(date_trunc('second',now()-preparado+caduca) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"'),
  'perfiles',perfiles)::text
$f$;
CREATE FUNCTION pg_temp.operador(login text,plan text) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE format('CREATE ROLE %I LOGIN',login);
 EXECUTE format('GRANT vec_admin_perfiles_asignables_ejecutor TO %I WITH INHERIT TRUE, SET FALSE, ADMIN FALSE',login);
 IF plan IS NOT NULL THEN
  INSERT INTO vec_autorizacion.config_perfiles_asignables_admin_v1 VALUES(login,encode(sha256(convert_to(plan,'UTF8')),'hex'),
   'aprobacion:prueba:aut49','0000000000000000000000000000000000000000000000000000000000000001','desarrollo',now()-interval '5 minutes',now()+interval '2 hours');
 END IF;
 RETURN encode(sha256(convert_to(coalesce(plan,''),'UTF8')),'hex');
END $f$;
CREATE FUNCTION pg_temp.comprobar(caso text,condicion boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;

-- 1. Registro de dos perfiles ordinarios.
SELECT pg_temp.plan('rpa_prueba_aut49_positivo_0001',jsonb_build_array(pg_temp.perfil('rol:tecnico_rrhh_desarrollo:v1'),pg_temp.perfil('rol:llamamiento_desarrollo:v1'))) AS plan_a \gset
SELECT pg_temp.operador('prueba_aut49_a',:'plan_a') AS sha_a \gset
SET SESSION AUTHORIZATION prueba_aut49_a;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_a',:'sha_a') AS r1 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('positivo',(:'r1'::jsonb)->>'estado'='permitido' AND ((:'r1'::jsonb)->>'replay')::boolean IS FALSE
 AND jsonb_array_length((:'r1'::jsonb)#>'{recibo,perfiles}')=2
 AND (SELECT count(*) FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE clase='ordinario' AND version_rol_ref IN('rol:tecnico_rrhh_desarrollo:v1','rol:llamamiento_desarrollo:v1'))=2
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='perfiles_asignables_admin')=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='intento_perfiles_asignables_admin' AND resultado='permitido' AND motivo_ref='perfiles_asignables_registrado')=1);
SELECT pg_temp.comprobar('resolver_rol',vec_autorizacion.resolver_rol_administrable_v1('rol:tecnico_rrhh_desarrollo:v1')->>'clase'='ordinario'
 AND (vec_autorizacion.resolver_rol_administrable_v1('rol:llamamiento_desarrollo:v1')->>'unidad_requerida')::boolean IS FALSE);
SELECT pg_temp.comprobar('aprobacion_en_auditoria',(SELECT perfiles_asignables_detalle->>'aprobacion_sha256' FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='perfiles_asignables_admin')=repeat('0',63)||'1'
 AND (:'r1'::jsonb)#>>'{recibo,aprobacion_ref}'='aprobacion:prueba:aut49');
SELECT pg_temp.comprobar('registro_plan',(SELECT plan=convert_to(:'plan_a','UTF8') AND login_nombre='prueba_aut49_a' FROM vec_autorizacion.registro_perfiles_asignables_admin_v1 WHERE operacion_ref='rpa_prueba_aut49_positivo_0001'));

-- 2. Replay: mismo recibo, sin efecto nuevo, con intento auditado.
SET SESSION AUTHORIZATION prueba_aut49_a;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_a',:'sha_a') AS r2 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('replay',(:'r2'::jsonb)->>'estado'='permitido' AND ((:'r2'::jsonb)->>'replay')::boolean
 AND (:'r2'::jsonb)->'recibo'=(:'r1'::jsonb)->'recibo'
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='perfiles_asignables_admin')=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND motivo_ref='perfiles_asignables_replay')=1);

-- 3. Huella aprobada distinta: denegado y auditado.
SET SESSION AUTHORIZATION prueba_aut49_a;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_a',repeat('0',64)) AS r3 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('huella_distinta',(:'r3'::jsonb)->>'estado'='denegado' AND (:'r3'::jsonb)->>'codigo'='perfiles_asignables_rechazado' AND (:'r3'::jsonb)->'recibo'='null'::jsonb
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND motivo_ref='perfiles_asignables_denegado')=1);

-- 4-8. Clases prohibidas, CAS, duplicados y caducidad: denegados sin efecto.
SELECT pg_temp.plan('rpa_prueba_aut49_admin_000001',jsonb_build_array(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1'),pg_temp.perfil('rol:administracion_perfiles:v4'))) AS plan_b \gset
SELECT pg_temp.operador('prueba_aut49_b',:'plan_b') AS sha_b \gset
SELECT pg_temp.plan('rpa_prueba_aut49_fiscal_00001',jsonb_build_array(pg_temp.perfil('rol:intervencion_fiscalizacion_desarrollo:v1'))) AS plan_c \gset
SELECT pg_temp.operador('prueba_aut49_c',:'plan_c') AS sha_c \gset
SELECT pg_temp.plan('rpa_prueba_aut49_control_0001',jsonb_build_array(jsonb_set(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1'),'{control_revision}','7'))) AS plan_d \gset
SELECT pg_temp.operador('prueba_aut49_d',:'plan_d') AS sha_d \gset
SELECT pg_temp.plan('rpa_prueba_aut49_repetido_001',jsonb_build_array(pg_temp.perfil('rol:tecnico_rrhh_desarrollo:v1'))) AS plan_e \gset
SELECT pg_temp.operador('prueba_aut49_e',:'plan_e') AS sha_e \gset
SELECT pg_temp.plan('rpa_prueba_aut49_caducado_001',jsonb_build_array(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1')),interval '3 hours',interval '1 hour') AS plan_f \gset
SELECT pg_temp.operador('prueba_aut49_f',:'plan_f') AS sha_f \gset
SELECT pg_temp.plan('rpa_prueba_aut49_doble_00001',jsonb_build_array(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1'),pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1'))) AS plan_g \gset
SELECT pg_temp.operador('prueba_aut49_g',:'plan_g') AS sha_g \gset
SELECT pg_temp.plan('rpa_prueba_aut49_sistemas_001',jsonb_build_array(pg_temp.perfil('rol:operador_plataforma:v1'))) AS plan_h \gset
SELECT pg_temp.operador('prueba_aut49_h',:'plan_h') AS sha_h \gset
SELECT jsonb_set(jsonb_set(pg_temp.plan('rpa_prueba_aut49_fechasnulas01',jsonb_build_array(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1')))::jsonb,'{preparado_en}','null'),'{caduca_en}','null')::text AS plan_j \gset
SELECT pg_temp.operador('prueba_aut49_j',:'plan_j') AS sha_j \gset
SELECT '{"esquema": "otro", '||substr(pg_temp.plan('rpa_prueba_aut49_clavesdobles1',jsonb_build_array(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1'))),2) AS plan_k \gset
SELECT pg_temp.operador('prueba_aut49_k',:'plan_k') AS sha_k \gset
SELECT pg_temp.plan('rpa_prueba_aut49_candidato_001',jsonb_build_array(pg_temp.perfil('rol:candidato_bolsa_consulta_propia_desarrollo:v1'))) AS plan_l \gset
SELECT pg_temp.operador('prueba_aut49_l',:'plan_l') AS sha_l \gset
SELECT pg_temp.plan('rpa_prueba_aut49_externa_0001',jsonb_build_array(pg_temp.perfil('rol:usuarios_preferencias_externa_h3:v2'),pg_temp.perfil('rol:candidato_bolsa_portal_propio_desarrollo:v1'))) AS plan_m \gset
SELECT pg_temp.plan('rpa_prueba_aut49_firmasinterv1',jsonb_build_array(pg_temp.perfil('rol:intervencion_firmas_lector_desarrollo:v1'))) AS plan_n \gset
SELECT pg_temp.operador('prueba_aut49_n',:'plan_n') AS sha_n \gset
SELECT pg_temp.operador('prueba_aut49_m',:'plan_m') AS sha_m \gset
SET SESSION AUTHORIZATION prueba_aut49_j;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_j',:'sha_j')->>'estado' AS e_j \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_k;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_k',:'sha_k')->>'estado' AS e_k \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_l;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_l',:'sha_l')->>'estado' AS e_l \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_m;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_m',:'sha_m')->>'estado' AS e_m \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_n;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_n',:'sha_n')->>'estado' AS e_n \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_b;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_b',:'sha_b')->>'estado' AS e_b \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_c;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_c',:'sha_c')->>'estado' AS e_c \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_d;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_d',:'sha_d')->>'estado' AS e_d \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_e;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_e',:'sha_e')->>'estado' AS e_e \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_f;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_f',:'sha_f')->>'estado' AS e_f \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_g;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_g',:'sha_g')->>'estado' AS e_g \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut49_h;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_h',:'sha_h')->>'estado' AS e_h \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('administracion_denegada',:'e_b'='denegado');
SELECT pg_temp.comprobar('intervencion_denegada',:'e_c'='denegado');
SELECT pg_temp.comprobar('control_cas_denegado',:'e_d'='denegado');
SELECT pg_temp.comprobar('ya_registrado_denegado',:'e_e'='denegado');
SELECT pg_temp.comprobar('caducado_denegado',:'e_f'='denegado');
SELECT pg_temp.comprobar('duplicado_denegado',:'e_g'='denegado');
SELECT pg_temp.comprobar('sistemas_denegado',:'e_h'='denegado');
SELECT pg_temp.comprobar('fechas_nulas_denegado',:'e_j'='denegado');
SELECT pg_temp.comprobar('claves_dobles_denegado',:'e_k'='denegado');
SELECT pg_temp.comprobar('candidato_denegado',:'e_l'='denegado');
SELECT pg_temp.comprobar('externa_denegado',:'e_m'='denegado');
SELECT pg_temp.comprobar('lector_firmas_intervencion_denegado',:'e_n'='denegado');
SELECT pg_temp.comprobar('sin_efecto_parcial',NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref IN('rol:respuesta_recibida_desarrollo:v1','rol:intervencion_fiscalizacion_desarrollo:v1'))
 AND (SELECT count(*) FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE clase='ordinario')=2
 AND (SELECT count(*) FROM vec_autorizacion.registro_perfiles_asignables_admin_v1)=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND motivo_ref='perfiles_asignables_denegado')=13);

-- 9. LOGIN del grupo sin configuración aprobada.
SELECT pg_temp.operador('prueba_aut49_sin',NULL) AS sha_sin \gset
SET SESSION AUTHORIZATION prueba_aut49_sin;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_a',:'sha_a')->>'estado' AS e_sin \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('sin_configuracion_denegado',:'e_sin'='denegado');

-- 10. El operador no puede llamar a las funciones internas ni leer tablas.
SET SESSION AUTHORIZATION prueba_aut49_a;
DO $d$
DECLARE n int:=0;
BEGIN
 BEGIN PERFORM vec_autorizacion.aplicar_perfiles_asignables_admin_v1('{}','x'); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM vec_autorizacion.exigir_operador_perfiles_asignables_admin_v1(); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM 1 FROM vec_autorizacion.registro_perfiles_asignables_admin_v1; EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM 1 FROM vec_autorizacion.config_perfiles_asignables_admin_v1; EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM vec_autorizacion_atestada_v3.registrar_intento_perfiles_asignables_admin_v1('{}'); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 IF n<>5 THEN RAISE EXCEPTION 'FALLO acl_operador %',n; END IF;
END $d$;
RESET SESSION AUTHORIZATION;
SELECT 'OK acl_operador';

-- 11. Un LOGIN con permisos de más no se acredita como operador.
SELECT pg_temp.plan('rpa_prueba_aut49_ampliado_001',jsonb_build_array(pg_temp.perfil('rol:respuesta_recibida_desarrollo:v1'))) AS plan_i \gset
SELECT pg_temp.operador('prueba_aut49_i',:'plan_i') AS sha_i \gset
GRANT USAGE ON SCHEMA vec_autorizacion TO prueba_aut49_i;
SET SESSION AUTHORIZATION prueba_aut49_i;
SELECT vec_autorizacion.registrar_perfiles_asignables_admin_v1(:'plan_i',:'sha_i')->>'estado' AS e_i \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('login_ampliado_denegado',:'e_i'='denegado' AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref='rol:respuesta_recibida_desarrollo:v1'));

-- 12. Inmutabilidad de la historia y del registro.
DO $d$
DECLARE n int:=0;
BEGIN
 BEGIN UPDATE vec_autorizacion.registro_perfiles_asignables_admin_v1 SET login_nombre='x'; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN DELETE FROM vec_autorizacion.config_perfiles_asignables_admin_v1; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN DELETE FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE clase='ordinario'; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 IF n<>3 THEN RAISE EXCEPTION 'FALLO inmutable %',n; END IF;
END $d$;
SELECT 'OK inmutable';

-- 13. La cadena común sigue enlazada y los tipos nuevos cumplen su formato.
SELECT pg_temp.comprobar('cadena',NOT EXISTS(
 SELECT 1 FROM (SELECT secuencia,anterior_sha256,lag(huella_sha256) OVER (ORDER BY secuencia) AS previa FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>=:aud0) x
 WHERE secuencia>:aud0 AND anterior_sha256 IS DISTINCT FROM previa)
 AND (SELECT cabeza_sha256 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)=(SELECT huella_sha256 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 ORDER BY secuencia DESC LIMIT 1)
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0)=18);
ROLLBACK;
