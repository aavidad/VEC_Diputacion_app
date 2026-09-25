\set ON_ERROR_STOP on
-- Ejecutar como postgres sobre una base PG18 desechable VACÍA. Este esquema
-- sintético prueba sólo AD3-53a; no sustituye la instalación AD3 canónica.
DO $roles$
DECLARE nombre text;
BEGIN
  FOREACH nombre IN ARRAY ARRAY[
    'vec_autorizacion_atestada_v3_propietario',
    'vec_autorizacion_atestada_v3_emisor',
    'vec_autorizacion_atestada_v3_consumidor',
    'vec_autorizacion_atestada_v3_preflight_interno'] LOOP
    IF pg_catalog.to_regrole(nombre) IS NULL THEN
      EXECUTE pg_catalog.format('CREATE ROLE %I NOLOGIN',nombre);
    END IF;
  END LOOP;
  IF pg_catalog.to_regrole('vec_interno_preflight_v3_desarrollo') IS NULL THEN
    CREATE ROLE vec_interno_preflight_v3_desarrollo LOGIN;
  END IF;
END $roles$;
GRANT vec_autorizacion_atestada_v3_preflight_interno TO vec_interno_preflight_v3_desarrollo
  WITH ADMIN FALSE, INHERIT TRUE, SET FALSE;
CREATE SCHEMA vec_autorizacion_atestada_v3 AUTHORIZATION vec_autorizacion_atestada_v3_propietario;
GRANT USAGE ON SCHEMA vec_autorizacion_atestada_v3 TO vec_autorizacion_atestada_v3_preflight_interno;
SET ROLE vec_autorizacion_atestada_v3_propietario;
CREATE FUNCTION vec_autorizacion_atestada_v3.huella_sha256_valida(text)
RETURNS boolean LANGUAGE sql IMMUTABLE AS $$ SELECT $1 ~ '^[0-9a-f]{64}$' $$;
CREATE FUNCTION vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)
RETURNS boolean LANGUAGE sql AS $$ SELECT true $$;
CREATE TABLE vec_autorizacion_atestada_v3.checkpoint_gobierno
  (control_id boolean PRIMARY KEY, revision numeric, configuracion_secuencia_minima numeric, raiz_version_minima numeric);
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_confianza_version
  (revision text PRIMARY KEY, secuencia numeric, huella_configuracion_sha256 text,
   publicada_en timestamptz, expira_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.puntero_configuracion_actual
  (orden numeric PRIMARY KEY, configuracion_revision text, establecida_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.revocacion_configuracion
  (configuracion_revision text PRIMARY KEY, revocada_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.raiz_confianza_version
  (clave_id text, version numeric, clave_publica_spki bytea, huella_spki_sha256 text,
   audiencia_despliegue text, suite text, valida_desde timestamptz, valida_hasta timestamptz,
   PRIMARY KEY(clave_id,version));
CREATE TABLE vec_autorizacion_atestada_v3.configuracion_raiz
  (configuracion_revision text, raiz_clave_id text, raiz_version numeric);
CREATE TABLE vec_autorizacion_atestada_v3.revocacion_raiz
  (raiz_clave_id text, raiz_version numeric, revocada_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.clave_capacidad_version
  (clave_id text, version numeric, revision_gobierno numeric, huella_gobierno_sha256 text,
   huella_secreto_sha256 text, emisor_id text, audiencia_consumo text,
   valida_desde timestamptz, valida_hasta timestamptz, PRIMARY KEY(clave_id,version));
CREATE TABLE vec_autorizacion_atestada_v3.puntero_clave_emision
  (orden numeric PRIMARY KEY, clave_id text, version numeric, establecida_en timestamptz);
CREATE TABLE vec_autorizacion_atestada_v3.revocacion_clave_capacidad
  (clave_id text, version numeric, revocada_en timestamptz);
RESET ROLE;
-- Aplicar ahora, en la misma base, migraciones/000053a_lectura_configuracion_interna.up.sql.
-- El bloque siguiente es una transacción de prueba; termina en ROLLBACK.

BEGIN;
SET ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.checkpoint_gobierno VALUES (true,5,1,1);
INSERT INTO vec_autorizacion_atestada_v3.configuracion_confianza_version VALUES
 ('anterior',1,repeat('a',64),clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day'),
 ('actual',2,repeat('b',64),clock_timestamp()-interval '1 hour',clock_timestamp()+interval '1 day');
INSERT INTO vec_autorizacion_atestada_v3.puntero_configuracion_actual VALUES
 (1,'anterior',clock_timestamp()-interval '2 days'),
 (2,'actual',clock_timestamp()-interval '1 hour');
INSERT INTO vec_autorizacion_atestada_v3.raiz_confianza_version VALUES
 ('raiz',1,decode(repeat('01',44),'hex'),
  encode(sha256(decode(repeat('01',44),'hex')),'hex'),
  'vec:desarrollo:contratacion-temporal:atestacion:v3','VEC-AD-3-COSE-EDDSA-1',
  clock_timestamp()-interval '2 days',clock_timestamp()+interval '1 day');
INSERT INTO vec_autorizacion_atestada_v3.configuracion_raiz VALUES
 ('anterior','raiz',1),('actual','raiz',1);
WITH a(i,audiencia) AS (VALUES
 (1,'vec_personal.alta_ejercicio.v1'),
 (2,'vec_personal.lectura_incorporacion.v2'),
 (3,'vec_contratacion_temporal.incorporacion_ejercicio.v2'),
 (4,'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'),
 (5,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'))
INSERT INTO vec_autorizacion_atestada_v3.clave_capacidad_version
SELECT 'clave'||i,i,i,repeat('c',64),repeat('d',64),'emisor',audiencia,
       clock_timestamp()-interval '2 days',clock_timestamp()+interval '1 day' FROM a;
INSERT INTO vec_autorizacion_atestada_v3.puntero_clave_emision
SELECT i,'clave'||i,i,clock_timestamp()-interval '1 day' FROM generate_series(1,5) i;
RESET ROLE;
WITH a(i,audiencia) AS (VALUES
 (1,'vec_personal.alta_ejercicio.v1'),
 (2,'vec_personal.lectura_incorporacion.v2'),
 (3,'vec_contratacion_temporal.incorporacion_ejercicio.v2'),
 (4,'vec_contratacion_temporal.consultar_cuadro_rrhh_atestado.v1'),
 (5,'vec_contratacion_temporal.consultar_detalle_rrhh_atestado.v1'))
SELECT jsonb_build_object(
 'claves',(SELECT jsonb_agg(jsonb_build_object(
   'audiencia_consumo',audiencia,'clave_id','clave'||i,'version',i,
   'revision_gobierno',i,'huella_gobierno_sha256',repeat('c',64),
   'huella_secreto_sha256',repeat('d',64),'emisor_id','emisor') ORDER BY i) FROM a),
 'configuracion',jsonb_build_object('revision','anterior','secuencia',1,
   'huella_configuracion_sha256',repeat('a',64)),
 'raiz',jsonb_build_object('clave_id','raiz','version',1,
   'huella_spki_sha256',encode(sha256(decode(repeat('01',44),'hex')),'hex'),
   'audiencia_despliegue','vec:desarrollo:contratacion-temporal:atestacion:v3',
   'suite','VEC-AD-3-COSE-EDDSA-1'))::text AS material \gset
SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
SELECT format($fmt$DO $positivo$
DECLARE v jsonb;
BEGIN
    v := vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(%L::jsonb);
    IF v->>'revision' <> 'actual' OR v->>'secuencia' <> '2'
       OR v->>'huella_configuracion_sha256' <> repeat('b',64)
       OR (SELECT count(*) FROM jsonb_object_keys(v)) <> 5
       OR has_table_privilege(current_user,
          'vec_autorizacion_atestada_v3.configuracion_confianza_version','SELECT')
    THEN RAISE EXCEPTION 'lectura renovada o ACL incorrectas'; END IF;
END $positivo$;
$fmt$, :'material') \gexec
SELECT format($fmt$DO $negativo$ DECLARE p jsonb:=%L::jsonb; BEGIN
 p:=jsonb_set(p,'{raiz,huella_spki_sha256}',to_jsonb(repeat('e',64)));
 BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(p);
 RAISE EXCEPTION 'raíz cambiada aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $negativo$;$fmt$, :'material') \gexec
RESET SESSION AUTHORIZATION;
SET ROLE vec_autorizacion_atestada_v3_propietario;
INSERT INTO vec_autorizacion_atestada_v3.revocacion_configuracion
VALUES ('actual',clock_timestamp());
RESET ROLE;
SELECT format($fmt$SET SESSION AUTHORIZATION vec_interno_preflight_v3_desarrollo;
DO $negativo$ BEGIN
 BEGIN PERFORM vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(%L::jsonb);
 RAISE EXCEPTION 'configuración revocada aceptada';
 EXCEPTION WHEN SQLSTATE '42501' THEN NULL; END;
END $negativo$;
RESET SESSION AUTHORIZATION;$fmt$, :'material') \gexec
ROLLBACK;
