\set ON_ERROR_STOP on
-- CT-131 DOWN solo para ensayo sin historia. Las publicaciones, borradores,
-- recibos, consumos y eventos históricos no se eliminan en una base conservada.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:migracion:000131',0));
DO $vacia$
BEGIN
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_historia_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_outbox_v1)
    OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1)
 THEN RAISE EXCEPTION 'CT-131: DOWN denegado con historia' USING ERRCODE='55000'; END IF;
END $vacia$;
DROP FUNCTION vec_contratacion_temporal.operar_catalogo_plantillas_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea);
DROP FUNCTION vec_contratacion_temporal.comprobar_catalogo_plantillas_base_v1(text,bigint,text);
DROP FUNCTION vec_contratacion_temporal.provisionar_catalogo_plantillas_base_v1(jsonb,text,text,text);
DROP TABLE vec_contratacion_temporal.catalogo_plantillas_provision_auditoria_v1;
DROP TABLE vec_contratacion_temporal.catalogo_plantillas_outbox_v1;
DROP TABLE vec_contratacion_temporal.catalogo_plantillas_historia_v1;
-- CT-131 partió de ausencia de USAGE para el migrador. Si otra migración
-- añadió privilegios directos sobre objetos de este esquema, conservar USAGE.
DO $acl$
BEGIN
 IF NOT EXISTS (
  SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace,
   LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
  WHERE n.nspname='vec_contratacion_temporal'
    AND a.grantee='vec_contratacion_temporal_migrador'::regrole
 ) AND NOT EXISTS (
  SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace,
   LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE n.nspname='vec_contratacion_temporal'
    AND a.grantee='vec_contratacion_temporal_migrador'::regrole
 ) AND NOT EXISTS (
  SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace,
   LATERAL aclexplode(coalesce(t.typacl,acldefault('T',t.typowner))) a
  WHERE n.nspname='vec_contratacion_temporal'
    AND a.grantee='vec_contratacion_temporal_migrador'::regrole
 ) THEN
  REVOKE USAGE ON SCHEMA vec_contratacion_temporal FROM vec_contratacion_temporal_migrador;
 END IF;
END $acl$;
COMMIT;
