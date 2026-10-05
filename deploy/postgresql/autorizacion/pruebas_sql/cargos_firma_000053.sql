\set ON_ERROR_STOP on
-- Vector de AUT53. Se ejecuta como superusuario sobre un clon desechable con
-- AUT24, AUT49, AD196 y AUT53 instaladas. Todo ocurre en una transacción que
-- termina en ROLLBACK. Cada caso imprime «OK <caso>»; cualquier discrepancia
-- aborta con error.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SELECT secuencia AS aud0 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id \gset
SELECT count(*) AS adm0 FROM vec_autorizacion.rol_administrable_exacto_v1 \gset
SELECT count(*) AS rol0 FROM vec_autorizacion.version_rol \gset

CREATE FUNCTION pg_temp.fecha(i interval) RETURNS text LANGUAGE sql AS $f$
 SELECT to_char(date_trunc('second',now()+i) AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"')
$f$;
-- Cargo sin huella: el plan la calcula con el documento de AUT53 salvo que el
-- caso fije otra a propósito.
CREATE FUNCTION pg_temp.cargo(rol text,version int DEFAULT 1,anterior text DEFAULT '',ops jsonb DEFAULT '["consultar_r5","firmar_vec"]',
 comps jsonb DEFAULT '[{"accion":"contratacion_temporal.documento.firma_vec.registrar","tipo_recurso":"documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"}]',
 nombre text DEFAULT 'Cargo de firma de prueba',regla text DEFAULT 'lote_ordinario') RETURNS jsonb LANGUAGE sql AS $f$
 SELECT jsonb_build_object('rol_id',rol,'version',version,'nombre',nombre,'version_anterior_sha256',anterior,'operaciones_v2',ops,'competencias',comps,
  'version_rol_sha256','','regla_asignacion',regla,'organizacion_ref','organizacion:desarrollo:dipgra',
  'vigente_desde',pg_temp.fecha(interval '0'),'vigente_hasta',pg_temp.fecha(interval '365 days'),'duracion_propuesta_segundos',86400)
$f$;
CREATE FUNCTION pg_temp.plan(op text,cargos jsonb,preparado interval DEFAULT interval '1 minute',caduca interval DEFAULT interval '1 hour') RETURNS text LANGUAGE plpgsql AS $f$
DECLARE p jsonb;c jsonb;lista jsonb:='[]';
BEGIN
 p:=jsonb_build_object('esquema','vec.admin.cargos-firma.plan.v1','operacion_ref',op,'preparado_en',pg_temp.fecha(-preparado),
  'caduca_en',pg_temp.fecha(-preparado+caduca),'cargos','[]'::jsonb);
 FOR c IN SELECT value FROM jsonb_array_elements(cargos) LOOP
  IF c->>'version_rol_sha256'='' THEN
   -- Un cargo inválido a propósito no tiene documento: lleva una huella cualquiera.
   BEGIN
    c:=jsonb_set(c,'{version_rol_sha256}',to_jsonb(encode(sha256(convert_to(vec_autorizacion.canon_version_rol_admin_v1(vec_autorizacion.documento_rol_cargo_firma_v1(jsonb_set(c,'{version_rol_sha256}',to_jsonb(repeat('0',64))),p)),'UTF8')),'hex')));
   EXCEPTION WHEN OTHERS THEN c:=jsonb_set(c,'{version_rol_sha256}',to_jsonb(repeat('c',64)));
   END;
  END IF;
  lista:=lista||jsonb_build_array(c);
 END LOOP;
 RETURN jsonb_set(p,'{cargos}',lista)::text;
END $f$;
CREATE FUNCTION pg_temp.operador(login text,plan text,grupo text DEFAULT 'vec_admin_cargos_firma_ejecutor') RETURNS text LANGUAGE plpgsql AS $f$
BEGIN
 EXECUTE format('CREATE ROLE %I LOGIN',login);
 EXECUTE format('GRANT %I TO %I WITH INHERIT TRUE, SET FALSE, ADMIN FALSE',grupo,login);
 IF plan IS NOT NULL THEN
  INSERT INTO vec_autorizacion.config_cargos_firma_admin_v1 VALUES(login,encode(sha256(convert_to(plan,'UTF8')),'hex'),
   'aprobacion:prueba:aut53','0000000000000000000000000000000000000000000000000000000000000002','desarrollo',now()-interval '5 minutes',now()+interval '2 hours');
 END IF;
 RETURN encode(sha256(convert_to(coalesce(plan,''),'UTF8')),'hex');
END $f$;
CREATE FUNCTION pg_temp.comprobar(caso text,condicion boolean) RETURNS text LANGUAGE plpgsql AS $f$
BEGIN IF condicion IS NOT TRUE THEN RAISE EXCEPTION 'FALLO %',caso; END IF; RETURN 'OK '||caso; END $f$;

-- 1. Dos cargos: uno con firma V2 y consulta, otro sólo con la competencia.
SELECT pg_temp.plan('rpa_cf_prueba_aut53_positivo_01',jsonb_build_array(pg_temp.cargo('ct_prueba_direccion_rrhh'),
 pg_temp.cargo('ct_prueba_secretaria',ops=>'[]',nombre=>'Secretaría General (prueba)'))) AS plan_a \gset
SELECT pg_temp.operador('prueba_aut53_a',:'plan_a') AS sha_a \gset
SET SESSION AUTHORIZATION prueba_aut53_a;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_a',:'sha_a') AS r1 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('positivo',(:'r1'::jsonb)->>'estado'='permitido' AND ((:'r1'::jsonb)->>'replay')::boolean IS FALSE
 AND jsonb_array_length((:'r1'::jsonb)#>'{recibo,cargos}')=2
 AND (SELECT count(*) FROM vec_autorizacion.version_rol WHERE version_rol_ref IN('rol:ct_prueba_direccion_rrhh:v1','rol:ct_prueba_secretaria:v1'))=2
 AND (SELECT count(*) FROM vec_autorizacion.control_vigencia_version_rol_actual WHERE version_rol_ref IN('rol:ct_prueba_direccion_rrhh:v1','rol:ct_prueba_secretaria:v1') AND revision=1)=2
 AND (SELECT count(*) FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE clase='ordinario' AND unidad_requerida
      AND audiencia_administrativa='vec_autorizacion.administracion_perfiles.lote_ordinario.v1'
      AND ambitos_fijos='[{"clave":"organizacion_ref","valores":["organizacion:desarrollo:dipgra"]}]'
      AND version_rol_ref IN('rol:ct_prueba_direccion_rrhh:v1','rol:ct_prueba_secretaria:v1'))=2
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='perfiles_asignables_admin')=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='intento_perfiles_asignables_admin' AND resultado='permitido' AND motivo_ref='perfiles_asignables_registrado')=1
 AND (SELECT count(*) FROM vec_autorizacion.operacion_cargos_firma_admin_v1 o JOIN vec_autorizacion_atestada_v3.auditoria_consumo_v3 a USING(auditoria_ref) WHERE a.secuencia>:aud0)=2);
-- El documento publicado lleva exactamente las concesiones que comprueban AD162 y AUT32.
SELECT pg_temp.comprobar('concesiones_exactas',
 (SELECT documento->'concesiones' FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:ct_prueba_direccion_rrhh:v1')
  = jsonb_build_array(vec_autorizacion.concesion_v2_cargo_firma_v1('consultar_r5'),vec_autorizacion.concesion_v2_cargo_firma_v1('firmar_vec'),
    '{"accion":"contratacion_temporal.documento.firma_vec.registrar","modulo_id":"contratacion_temporal","tipo_recurso":"documento_contratacion_temporal","finalidades":["gestionar_contratacion_temporal"],"garantia_minima":"alto","campos_permitidos":[],"obligaciones":[]}'::jsonb)
 AND (SELECT jsonb_array_length(documento->'concesiones') FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:ct_prueba_secretaria:v1')=1
 AND (SELECT documento->>'publicada_por' FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:ct_prueba_secretaria:v1')='operacion:cargos_firma:rpa_cf_prueba_aut53_positivo_01');
-- La consulta de AUT32 encuentra la concesión competencial sin campos ni obligaciones.
SELECT pg_temp.comprobar('competencia_aut32',EXISTS(SELECT 1 FROM vec_autorizacion.version_rol r, jsonb_array_elements(r.documento->'concesiones') q
 WHERE r.version_rol_ref='rol:ct_prueba_secretaria:v1' AND q->>'accion'='contratacion_temporal.documento.firma_vec.registrar' AND q->>'modulo_id'='contratacion_temporal'
 AND q->>'tipo_recurso'='documento_contratacion_temporal' AND q->'finalidades' ? 'gestionar_contratacion_temporal'
 AND coalesce(q->'obligaciones','[]'::jsonb)='[]'::jsonb AND coalesce(q->'campos_permitidos','[]'::jsonb)='[]'::jsonb));
SELECT pg_temp.comprobar('resolver_rol',vec_autorizacion.resolver_rol_administrable_v1('rol:ct_prueba_direccion_rrhh:v1')->>'clase'='ordinario'
 AND (vec_autorizacion.resolver_rol_administrable_v1('rol:ct_prueba_secretaria:v1')->>'unidad_requerida')::boolean);
SELECT pg_temp.comprobar('registro_plan',(SELECT plan=convert_to(:'plan_a','UTF8') AND login_nombre='prueba_aut53_a' FROM vec_autorizacion.registro_cargos_firma_admin_v1 WHERE operacion_ref='rpa_cf_prueba_aut53_positivo_01')
 AND (SELECT perfiles_asignables_detalle->>'aprobacion_sha256' FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='perfiles_asignables_admin')=repeat('0',63)||'2');

-- 2. Replay: mismo recibo, sin efecto nuevo, con intento auditado y anotado.
SET SESSION AUTHORIZATION prueba_aut53_a;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_a',:'sha_a') AS r2 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('replay',(:'r2'::jsonb)->>'estado'='permitido' AND ((:'r2'::jsonb)->>'replay')::boolean
 AND (:'r2'::jsonb)->'recibo'=(:'r1'::jsonb)->'recibo'
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND tipo_registro='perfiles_asignables_admin')=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND motivo_ref='perfiles_asignables_replay')=1
 AND (SELECT count(*) FROM vec_autorizacion.operacion_cargos_firma_admin_v1)=3);

-- 3. Huella aprobada distinta: denegado y auditado.
SET SESSION AUTHORIZATION prueba_aut53_a;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_a',repeat('0',64)) AS r3 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('huella_distinta',(:'r3'::jsonb)->>'estado'='denegado' AND (:'r3'::jsonb)->>'codigo'='cargos_firma_rechazado' AND (:'r3'::jsonb)->'recibo'='null'::jsonb
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND motivo_ref='perfiles_asignables_denegado')=1
 AND (SELECT count(*) FROM vec_autorizacion.operacion_cargos_firma_admin_v1 WHERE resultado='denegado')=1);

-- 4-13. Negativos: ninguno deja efecto.
SELECT pg_temp.plan('rpa_cf_prueba_aut53_intervenc_1',jsonb_build_array(pg_temp.cargo('intervencion_firma_prueba'))) AS plan_b \gset
SELECT pg_temp.operador('prueba_aut53_b',:'plan_b') AS sha_b \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_nombre_int1',jsonb_build_array(pg_temp.cargo('ct_prueba_interventor',nombre=>'Intervención General'))) AS plan_c \gset
SELECT pg_temp.operador('prueba_aut53_c',:'plan_c') AS sha_c \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_casv1_0001',jsonb_build_array(pg_temp.cargo('ct_prueba_direccion_rrhh'))) AS plan_d \gset
SELECT pg_temp.operador('prueba_aut53_d',:'plan_d') AS sha_d \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_casv2_0001',jsonb_build_array(pg_temp.cargo('ct_prueba_direccion_rrhh',2,repeat('a',64)))) AS plan_e \gset
SELECT pg_temp.operador('prueba_aut53_e',:'plan_e') AS sha_e \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_huella_rol1',jsonb_build_array(jsonb_set(pg_temp.cargo('ct_prueba_huella'),'{version_rol_sha256}',to_jsonb(repeat('b',64))))) AS plan_f \gset
SELECT pg_temp.operador('prueba_aut53_f',:'plan_f') AS sha_f \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_tipo_v3_001',jsonb_build_array(pg_temp.cargo('ct_prueba_tipo',comps=>'[{"accion":"contratacion_temporal.documento.firma_externa.registrar","tipo_recurso":"firma_externa_documento_contratacion_temporal","finalidad":"gestionar_contratacion_temporal"}]'))) AS plan_g \gset
SELECT pg_temp.operador('prueba_aut53_g',:'plan_g') AS sha_g \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_admin_0001',jsonb_build_array(pg_temp.cargo('ct_prueba_admin',comps=>'[{"accion":"administracion.perfiles.otorgar","tipo_recurso":"documento_contratacion_temporal","finalidad":"gestion_perfiles"}]'))) AS plan_h \gset
SELECT pg_temp.operador('prueba_aut53_h',:'plan_h') AS sha_h \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_regla_00001',jsonb_build_array(pg_temp.cargo('ct_prueba_regla',regla=>'doble_control'))) AS plan_i \gset
SELECT pg_temp.operador('prueba_aut53_i',:'plan_i') AS sha_i \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_ops_000001',jsonb_build_array(pg_temp.cargo('ct_prueba_ops',ops=>'["firmar_vec","consultar_r5"]'))) AS plan_j \gset
SELECT pg_temp.operador('prueba_aut53_j',:'plan_j') AS sha_j \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_caducado01',jsonb_build_array(pg_temp.cargo('ct_prueba_caducado')),interval '3 hours',interval '1 hour') AS plan_k \gset
SELECT pg_temp.operador('prueba_aut53_k',:'plan_k') AS sha_k \gset
-- Un cargo bueno junto a uno sensible: la operación entera se deniega.
SELECT pg_temp.plan('rpa_cf_prueba_aut53_parcial_01',jsonb_build_array(pg_temp.cargo('ct_prueba_bueno'),pg_temp.cargo('operador_plataforma',2,(SELECT huella_sha256 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:operador_plataforma:v1')))) AS plan_l \gset
SELECT pg_temp.operador('prueba_aut53_l',:'plan_l') AS sha_l \gset
SELECT '{"esquema": "otro", '||substr(pg_temp.plan('rpa_cf_prueba_aut53_dobles_001',jsonb_build_array(pg_temp.cargo('ct_prueba_dobles'))),2) AS plan_m \gset
SELECT pg_temp.operador('prueba_aut53_m',:'plan_m') AS sha_m \gset
SELECT pg_temp.plan('rpa_cf_prueba_aut53_bootstrap1',jsonb_build_array(pg_temp.cargo('tecnico_rrhh_desarrollo'))) AS plan_n \gset
SELECT pg_temp.operador('prueba_aut53_n',:'plan_n') AS sha_n \gset
SET SESSION AUTHORIZATION prueba_aut53_b;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_b',:'sha_b')->>'estado' AS e_b \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_c;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_c',:'sha_c')->>'estado' AS e_c \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_d;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_d',:'sha_d')->>'estado' AS e_d \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_e;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_e',:'sha_e')->>'estado' AS e_e \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_f;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_f',:'sha_f')->>'estado' AS e_f \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_g;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_g',:'sha_g')->>'estado' AS e_g \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_h;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_h',:'sha_h')->>'estado' AS e_h \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_i;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_i',:'sha_i')->>'estado' AS e_i \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_j;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_j',:'sha_j')->>'estado' AS e_j \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_k;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_k',:'sha_k')->>'estado' AS e_k \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_l;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_l',:'sha_l')->>'estado' AS e_l \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_m;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_m',:'sha_m')->>'estado' AS e_m \gset
RESET SESSION AUTHORIZATION;
SET SESSION AUTHORIZATION prueba_aut53_n;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_n',:'sha_n')->>'estado' AS e_n \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('rol_intervencion_denegado',:'e_b'='denegado');
SELECT pg_temp.comprobar('nombre_intervencion_denegado',:'e_c'='denegado');
SELECT pg_temp.comprobar('cas_v1_existente_denegado',:'e_d'='denegado');
SELECT pg_temp.comprobar('cas_v2_huella_anterior_denegado',:'e_e'='denegado');
SELECT pg_temp.comprobar('huella_rol_divergente_denegado',:'e_f'='denegado');
SELECT pg_temp.comprobar('competencia_tipo_v3_denegado',:'e_g'='denegado');
SELECT pg_temp.comprobar('competencia_administracion_denegado',:'e_h'='denegado');
SELECT pg_temp.comprobar('regla_no_configurada_denegado',:'e_i'='denegado');
SELECT pg_temp.comprobar('operaciones_desordenadas_denegado',:'e_j'='denegado');
SELECT pg_temp.comprobar('caducado_denegado',:'e_k'='denegado');
SELECT pg_temp.comprobar('sistemas_en_plan_denegado',:'e_l'='denegado');
SELECT pg_temp.comprobar('claves_dobles_denegado',:'e_m'='denegado');
SELECT pg_temp.comprobar('rol_existente_v1_denegado',:'e_n'='denegado');
SELECT pg_temp.comprobar('sin_efecto_parcial',(SELECT count(*) FROM vec_autorizacion.version_rol)=:rol0+2
 AND (SELECT count(*) FROM vec_autorizacion.rol_administrable_exacto_v1)=:adm0+2
 AND (SELECT count(*) FROM vec_autorizacion.registro_cargos_firma_admin_v1)=1
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0 AND motivo_ref='perfiles_asignables_denegado')=14
 AND (SELECT count(*) FROM vec_autorizacion.operacion_cargos_firma_admin_v1 WHERE resultado='denegado')=14);

-- 14. Versión 2 con CAS correcto: se publica y se registra.
SELECT pg_temp.plan('rpa_cf_prueba_aut53_version2_01',jsonb_build_array(pg_temp.cargo('ct_prueba_secretaria',2,
 (SELECT huella_sha256 FROM vec_autorizacion.version_rol WHERE version_rol_ref='rol:ct_prueba_secretaria:v1'),ops=>'["consultar_r5"]',nombre=>'Secretaría General (prueba)'))) AS plan_v2 \gset
SELECT pg_temp.operador('prueba_aut53_v2',:'plan_v2') AS sha_v2 \gset
SET SESSION AUTHORIZATION prueba_aut53_v2;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_v2',:'sha_v2')->>'estado' AS e_v2 \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('version2_cas',:'e_v2'='permitido' AND EXISTS(SELECT 1 FROM vec_autorizacion.rol_administrable_exacto_v1 WHERE version_rol_ref='rol:ct_prueba_secretaria:v2' AND clase='ordinario'));

-- 15. LOGIN del grupo sin configuración aprobada: denegado y auditado.
SELECT pg_temp.operador('prueba_aut53_sin',NULL) AS sha_sin \gset
SET SESSION AUTHORIZATION prueba_aut53_sin;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_a',:'sha_a')->>'estado' AS e_sin \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('sin_configuracion_denegado',:'e_sin'='denegado');

-- 16. LOGIN ajeno (operador de AUT49) no puede ejecutar la fachada.
SELECT pg_temp.operador('prueba_aut53_ajeno',NULL,'vec_admin_perfiles_asignables_ejecutor') AS sha_ajeno \gset
SET SESSION AUTHORIZATION prueba_aut53_ajeno;
DO $d$
BEGIN
 BEGIN PERFORM vec_autorizacion.publicar_cargos_firma_admin_v1('{}','x'); RAISE EXCEPTION 'FALLO login_ajeno'; EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $d$;
RESET SESSION AUTHORIZATION;
SELECT 'OK login_ajeno_sin_execute';

-- 17. LOGIN con permisos de más no se acredita como operador.
SELECT pg_temp.plan('rpa_cf_prueba_aut53_ampliado_1',jsonb_build_array(pg_temp.cargo('ct_prueba_ampliado'))) AS plan_x \gset
SELECT pg_temp.operador('prueba_aut53_x',:'plan_x') AS sha_x \gset
GRANT USAGE ON SCHEMA vec_autorizacion TO prueba_aut53_x;
SET SESSION AUTHORIZATION prueba_aut53_x;
SELECT vec_autorizacion.publicar_cargos_firma_admin_v1(:'plan_x',:'sha_x')->>'estado' AS e_x \gset
RESET SESSION AUTHORIZATION;
SELECT pg_temp.comprobar('login_ampliado_denegado',:'e_x'='denegado' AND NOT EXISTS(SELECT 1 FROM vec_autorizacion.version_rol WHERE rol_id='ct_prueba_ampliado'));

-- 18. El operador no puede llamar a las funciones internas ni leer tablas.
SET SESSION AUTHORIZATION prueba_aut53_a;
DO $d$
DECLARE n int:=0;
BEGIN
 BEGIN PERFORM vec_autorizacion.aplicar_cargos_firma_admin_v1('{}','x'); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM vec_autorizacion.exigir_operador_cargos_firma_admin_v1(); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM vec_autorizacion.documento_rol_cargo_firma_v1('{}','{}'); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM 1 FROM vec_autorizacion.registro_cargos_firma_admin_v1; EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM 1 FROM vec_autorizacion.config_cargos_firma_admin_v1; EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM 1 FROM vec_autorizacion.regla_asignacion_cargo_firma_v1; EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 BEGIN PERFORM vec_autorizacion.validar_perfil_asignable_admin_v1('{}',true); EXCEPTION WHEN insufficient_privilege THEN n:=n+1; END;
 IF n<>7 THEN RAISE EXCEPTION 'FALLO acl_operador %',n; END IF;
END $d$;
RESET SESSION AUTHORIZATION;
SELECT 'OK acl_operador';

-- 19. Inmutabilidad de la historia, el registro y las reglas.
DO $d$
DECLARE n int:=0;
BEGIN
 BEGIN UPDATE vec_autorizacion.registro_cargos_firma_admin_v1 SET login_nombre='x'; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN DELETE FROM vec_autorizacion.operacion_cargos_firma_admin_v1; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN UPDATE vec_autorizacion.regla_asignacion_cargo_firma_v1 SET audiencia_administrativa='otra'; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 BEGIN DELETE FROM vec_autorizacion.version_rol WHERE rol_id='ct_prueba_secretaria'; EXCEPTION WHEN OTHERS THEN n:=n+1; END;
 IF n<>4 THEN RAISE EXCEPTION 'FALLO inmutable %',n; END IF;
END $d$;
SELECT 'OK inmutable';

-- 20. La cadena común sigue enlazada.
SELECT pg_temp.comprobar('cadena',NOT EXISTS(
 SELECT 1 FROM (SELECT secuencia,anterior_sha256,lag(huella_sha256) OVER (ORDER BY secuencia) AS previa FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>=:aud0) x
 WHERE secuencia>:aud0 AND anterior_sha256 IS DISTINCT FROM previa)
 AND (SELECT cabeza_sha256 FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id)=(SELECT huella_sha256 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 ORDER BY secuencia DESC LIMIT 1)
 AND (SELECT count(*) FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE secuencia>:aud0)=(SELECT count(*) FROM vec_autorizacion.operacion_cargos_firma_admin_v1));
ROLLBACK;
