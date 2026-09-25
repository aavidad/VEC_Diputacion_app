\set ON_ERROR_STOP on
-- Ejecutar con psql como DBA solo en una base PostgreSQL 18 desechable,
-- con AD3-29 y AD3-50a instaladas, sin filas de gobierno. Todo el material
-- sintético y el LOGIN nominal de esta prueba se revierten.
BEGIN;
DO $preimagen$
BEGIN
    IF pg_catalog.to_regprocedure(
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)') IS NULL
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.clave_capacidad_version)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.configuracion_confianza_version)
       OR EXISTS (SELECT 1 FROM vec_autorizacion_atestada_v3.raiz_confianza_version)
       OR pg_catalog.to_regrole('vec_interno_preflight_v3_desarrollo') IS NOT NULL
    THEN
        RAISE EXCEPTION 'AD3-50a: prueba requiere base desechable vacía'
            USING ERRCODE = '55000';
    END IF;
END $preimagen$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno
    (control_id,revision,configuracion_secuencia_minima,raiz_version_minima,actualizada_en)
VALUES (true,0,0,0,clock_timestamp());
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version
    (revision,secuencia,huella_configuracion_sha256,publicada_en,expira_en,acto_ref)
VALUES ('c1',1,repeat('b',64),clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day','acto_config_1');
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version
    (clave_id,version,clave_publica_spki,huella_spki_sha256,valida_desde,valida_hasta,suite,audiencia_despliegue,acto_ref)
SELECT 'r1',1,spki,encode(sha256(spki),'hex'),clock_timestamp()-interval '1 day',
       clock_timestamp()+interval '1 day','VEC-AD-3-COSE-EDDSA-1',
       'vec:desarrollo:contratacion-temporal:atestacion:v3','acto_raiz_1'
FROM (SELECT decode(repeat('01',44),'hex') AS spki) AS x;
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES ('c1','r1',1);
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual
    (orden,configuracion_revision,establecida_en,acto_ref)
VALUES (1,'c1',clock_timestamp()-interval '1 hour','acto_puntero_config_1');
WITH aud(i,nombre) AS (VALUES
 (1,'vec_personal.alta_ejercicio.v1'),
 (2,'vec_personal.lectura_incorporacion.v2'),
 (3,'vec_contratacion_temporal.incorporacion_ejercicio.v2'),
 (4,'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'),
 (5,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'))
INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
    (clave_id,version,revision_gobierno,huella_gobierno_sha256,secreto_hmac,huella_secreto_sha256,
     emisor_id,audiencia_consumo,valida_desde,valida_hasta,acto_ref)
SELECT 'k'||i,i,i,repeat('a',64),secreto,encode(sha256(secreto),'hex'),
       'e1',nombre,clock_timestamp()-interval '1 day',clock_timestamp()+interval '1 day','acto_key_'||i
FROM aud CROSS JOIN LATERAL (SELECT decode(repeat(lpad(to_hex(i),2,'0'),32),'hex') AS secreto) AS s;
INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
    (orden,clave_id,version,establecida_en,acto_ref)
SELECT i,'k'||i,i,clock_timestamp()-interval '1 hour','acto_pointer_'||i
FROM generate_series(1,5) AS i;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno SET revision=5 WHERE control_id;
RESET ROLE;
CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
WITH aud(i,nombre) AS (VALUES
 (1,'vec_personal.alta_ejercicio.v1'),
 (2,'vec_personal.lectura_incorporacion.v2'),
 (3,'vec_contratacion_temporal.incorporacion_ejercicio.v2'),
 (4,'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'),
 (5,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1')),
material AS (
 SELECT jsonb_build_object(
  'claves', (SELECT jsonb_agg(jsonb_build_object(
   'audiencia_consumo',nombre,'clave_id','k'||i,'version',i,
   'revision_gobierno',i,'huella_gobierno_sha256',repeat('a',64),
   'huella_secreto_sha256',encode(sha256(decode(repeat(lpad(to_hex(i),2,'0'),32),'hex')),'hex'),
   'emisor_id','e1') ORDER BY i) FROM aud),
  'configuracion', jsonb_build_object('revision','c1','secuencia',1,'huella_configuracion_sha256',repeat('b',64)),
  'raiz', jsonb_build_object('clave_id','r1','version',1,
   'huella_spki_sha256',encode(sha256(decode(repeat('01',44),'hex')),'hex'),
   'audiencia_despliegue','vec:desarrollo:contratacion-temporal:atestacion:v3',
   'suite','VEC-AD-3-COSE-EDDSA-1')) AS material)
SELECT material::text AS material FROM material \gset
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT session_user,current_user,
       vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(:'material'::jsonb) AS valido,
       has_function_privilege(current_user,
         'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)','EXECUTE') AS ejecuta,
       has_table_privilege(current_user,
         'vec_autorizacion_atestada_v3.clave_capacidad_version','SELECT') AS lee_clave,
       has_table_privilege(current_user,
         'vec_autorizacion_atestada_v3.raiz_confianza_version','SELECT') AS lee_raiz;
SELECT pg_catalog.format($formato$
DO $aserciones$
BEGIN
    IF vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(%L::jsonb)
       IS DISTINCT FROM true
       OR NOT pg_catalog.has_function_privilege(current_user,
           'vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)',
           'EXECUTE')
       OR pg_catalog.has_table_privilege(current_user,
           'vec_autorizacion_atestada_v3.clave_capacidad_version', 'SELECT')
       OR pg_catalog.has_table_privilege(current_user,
           'vec_autorizacion_atestada_v3.raiz_confianza_version', 'SELECT')
    THEN
        RAISE EXCEPTION 'AD3-50a: positivo o ACL incompatibles'
            USING ERRCODE = '55000';
    END IF;
END $aserciones$;
$formato$, :'material') \gexec
SELECT format($fmt$DO $negativo$
DECLARE p jsonb := %L::jsonb; v boolean;
BEGIN
    p := jsonb_set(p,'{claves,3,huella_secreto_sha256}',to_jsonb(repeat('e',64)));
    BEGIN
        v := vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(p);
        RAISE EXCEPTION 'la huella incorrecta fue aceptada';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        RAISE NOTICE 'huella incorrecta rechazada 42501';
    END;
END $negativo$;$fmt$, :'material') \gexec
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.revocacion_clave_capacidad
    (clave_id,version,revocada_en,motivo_catalogado_ref,acto_ref)
VALUES ('k4',4,clock_timestamp(),'motivo_sintetico','acto_revoke_4');
RESET ROLE;
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $negativo$
DECLARE p jsonb := %L::jsonb; v boolean;
BEGIN
    BEGIN
        v := vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(p);
        RAISE EXCEPTION 'la clave revocada fue aceptada';
    EXCEPTION WHEN SQLSTATE '42501' THEN
        RAISE NOTICE 'clave revocada rechazada 42501';
    END;
END $negativo$;$fmt$, :'material') \gexec
RESET SESSION AUTHORIZATION;
ROLLBACK;
