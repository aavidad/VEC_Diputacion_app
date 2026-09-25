\set ON_ERROR_STOP on
-- PG18 desechable con Identidad 000006, BASE10 y Personal 000010a instaladas. Esta
-- transacción no deja política ni modifica historia de incorporación.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
DO $pre$
BEGIN
 IF to_regclass('vec_personal.org_nodo_historia') IS NULL
    OR to_regprocedure('vec_personal.consultar_organizacion_historica_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR EXISTS (SELECT 1 FROM vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1)
    OR has_table_privilege('vec_personal_propietario',
      'vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1','SELECT')
    OR has_function_privilege('vec_personal_ejecutor',
      'vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(jsonb,timestamptz)','EXECUTE')
    OR NOT has_function_privilege('vec_personal_propietario',
      'vec_identidad_sesiones_v1.admite_politica_certificado_personal_desarrollo_v1(text,text,timestamptz)','EXECUTE') THEN
   RAISE EXCEPTION 'ACL o preimagen Personal 000010a incompatible';
 END IF;
END $pre$;
SET LOCAL ROLE vec_personal_propietario;
DO $sin_politica$
DECLARE v jsonb:=jsonb_build_object('garantia_observada','sustancial','metodo_observado','certificado',
 'superficie','interna_corporativa','cuenta_privilegiada',false,
 'politica_garantia_ref','pga_0123456789abcdefghijkl','politica_garantia_huella_sha256',repeat('a',64));
BEGIN
 IF vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(v,clock_timestamp()) THEN
   RAISE EXCEPTION 'sin política admitida';
 END IF;
END $sin_politica$;
RESET ROLE;
INSERT INTO vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1
 (singleton,politica_ref,huella_sha256,retirar_en,activa,registrada_en)
 VALUES (true,'pga_0123456789abcdefghijkl',repeat('a',64),clock_timestamp()+interval '1 hour',true,clock_timestamp());
SET LOCAL ROLE vec_personal_propietario;
DO $casos$
DECLARE v jsonb:=jsonb_build_object('garantia_observada','sustancial','metodo_observado','certificado',
 'superficie','interna_corporativa','cuenta_privilegiada',false,
 'politica_garantia_ref','pga_0123456789abcdefghijkl','politica_garantia_huella_sha256',repeat('a',64));
 x jsonb;
BEGIN
 IF vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(v,clock_timestamp()) IS DISTINCT FROM true THEN
   RAISE EXCEPTION 'política exacta rechazada';
 END IF;
 FOR x IN SELECT dato FROM (VALUES
   (jsonb_set(v,'{politica_garantia_ref}','"pga_otra23456789abcdefghijkl"')),
   (jsonb_set(v,'{politica_garantia_huella_sha256}',to_jsonb(repeat('b',64)))),
   (jsonb_set(v,'{garantia_observada}','"bajo"')),
   (jsonb_set(v,'{metodo_observado}','"sso"')),
   (jsonb_set(v,'{superficie}','"administracion_privilegiada"')),
   (jsonb_set(v,'{cuenta_privilegiada}','true'))
 ) variantes(dato) LOOP
   IF vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(x,clock_timestamp()) THEN
     RAISE EXCEPTION 'variante ajena admitida';
   END IF;
 END LOOP;
 IF vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(v,
      clock_timestamp()+interval '2 hours') THEN
   RAISE EXCEPTION 'política caducada admitida';
 END IF;
END $casos$;
RESET ROLE;
UPDATE vec_identidad_sesiones_v1.politica_certificado_personal_desarrollo_v1 SET activa=false WHERE singleton=true;
SET LOCAL ROLE vec_personal_propietario;
DO $revocada$
DECLARE v jsonb:=jsonb_build_object('garantia_observada','sustancial','metodo_observado','certificado',
 'superficie','interna_corporativa','cuenta_privilegiada',false,
 'politica_garantia_ref','pga_0123456789abcdefghijkl','politica_garantia_huella_sha256',repeat('a',64));
BEGIN
 IF vec_personal.admite_lectura_incorporacion_certificado_desarrollo_v1(v,clock_timestamp()) THEN
   RAISE EXCEPTION 'política revocada admitida';
 END IF;
END $revocada$;
ROLLBACK;
