\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000075:registro:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
LOCK TABLE vec_contratacion_temporal.incorporacion_registro_v2,vec_contratacion_temporal.incorporacion_auditoria_v2,
 vec_contratacion_temporal.incorporacion_outbox_v2 IN ACCESS EXCLUSIVE MODE;
DO $retirada$
DECLARE nombre text; firma text; f oid; t oid; ids oid[]; tipos oid[]; funciones oid[]:='{}'; objetivo oid;
 columnas text[]; esperadas text[]; n int:=0; fk text; atributo text; destino text;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user AND NOT rolcanlogin AND NOT rolinherit AND NOT rolsuper
  AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=current_user::regrole) THEN
  RAISE EXCEPTION 'CT75: autoridad de retirada incompatible' USING ERRCODE='55000'; END IF;
 FOREACH nombre IN ARRAY ARRAY['incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2'] LOOP
  SELECT c.oid INTO t FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace
   WHERE ns.nspname='vec_contratacion_temporal' AND c.relname=nombre AND c.relkind='r' AND NOT c.relispartition
    AND c.relowner=current_user::regrole AND c.relrowsecurity AND c.relforcerowsecurity
    AND obj_description(c.oid,'pg_class')='CT75:registro-incorporacion-v2:inmutable';
  IF t IS NULL THEN RAISE EXCEPTION 'CT75: inventario de retirada incompatible' USING ERRCODE='55000'; END IF;
  n:=n+1;
  esperadas:=CASE n
   WHEN 1 THEN ARRAY['recibo_ref','seguimiento_ref','organizacion_ref','idempotencia_ref','solicitud_ref','expediente_ref','version_expediente','version_anterior','version_resultante','estado_anterior_sha256','estado_resultante_sha256','material_json','material_canonico','material_sha256','intencion_canonica','intencion_sha256','exportacion_ct','persona_version','perfil_version','recibo_json','auditoria_ref','outbox_ref','registrada_en','evidencia_orden_json']
   WHEN 2 THEN ARRAY['auditoria_ref','recibo_ref','recuperado','decision_ct_ref','consumo_ct_sha256','decision_lectura_ref','consumo_lectura_sha256','consumos_json','registrada_en']
   ELSE ARRAY['outbox_ref','recibo_ref','evento_json','estado_sha256','creada_en'] END;
  SELECT array_agg(attname::text ORDER BY attnum) INTO columnas FROM pg_attribute WHERE attrelid=t AND attnum>0;
  IF columnas IS DISTINCT FROM esperadas OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid=t AND attnum>0 AND (attisdropped OR attacl IS NOT NULL OR NOT attnotnull))
  OR EXISTS (SELECT 1 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a WHERE c.oid=t AND (a.grantee<>c.relowner OR a.grantor<>c.relowner))
  OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
  OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='propietario' AND polcmd='*' AND polpermissive
   AND polroles=ARRAY[current_user::regrole::oid] AND pg_get_expr(polqual,polrelid)='true' AND pg_get_expr(polwithcheck,polrelid)='true')
  OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal)<>1
  OR NOT EXISTS (SELECT 1 FROM pg_trigger g JOIN pg_proc p ON p.oid=g.tgfoid JOIN pg_namespace ns ON ns.oid=p.pronamespace
   WHERE g.tgrelid=t AND NOT g.tgisinternal AND g.tgname='historia_inmutable' AND g.tgtype=58 AND g.tgenabled='O'
    AND g.tgnargs=0 AND g.tgqual IS NULL AND ns.nspname='vec_contratacion_temporal' AND p.proname='rechazar_mutacion_historia_v1'
    AND p.pronargs=0 AND p.proowner=current_user::regrole AND p.prorettype='trigger'::regtype)
  OR EXISTS (SELECT 1 FROM pg_inherits WHERE inhparent=t OR inhrelid=t)
  OR EXISTS (SELECT 1 FROM pg_index i WHERE i.indrelid=t AND NOT EXISTS (SELECT 1 FROM pg_constraint c WHERE c.conrelid=t AND c.conindid=i.indexrelid AND c.contype IN ('p','u'))) THEN
   RAISE EXCEPTION 'CT75: ACL, policy o inventario alterados' USING ERRCODE='55000'; END IF;
 END LOOP;
 FOREACH firma IN ARRAY ARRAY[
  'vec_contratacion_temporal.incorporacion75_instante(timestamptz)',
  'vec_contratacion_temporal.incorporacion75_piezas(bytea[],numeric,numeric)',
  'vec_contratacion_temporal.incorporacion75_ventana(jsonb,jsonb,timestamptz)',
  'vec_contratacion_temporal.incorporacion75_autoridad(jsonb,bytea[],numeric,numeric,timestamptz)',
  'vec_contratacion_temporal.incorporacion75_solicitud_sha256(jsonb)',
  'vec_contratacion_temporal.incorporacion75_evidencia(jsonb,jsonb,bytea[],numeric,numeric,timestamptz)',
  'vec_contratacion_temporal.incorporacion75_recibo(jsonb,text,jsonb,jsonb,jsonb,jsonb,jsonb,text,text)',
  'vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb)'] LOOP
  f:=to_regprocedure(firma);
  IF f IS NULL OR NOT EXISTS (SELECT 1 FROM pg_proc WHERE oid=f AND proowner=current_user::regrole
   AND obj_description(oid,'pg_proc')='CT75:registro-incorporacion-v2' AND prokind='f' AND proconfig @> ARRAY['search_path=pg_catalog']
   AND prosecdef=(proname='registrar_incorporacion_ejercicio_v2')
   AND provolatile=(CASE WHEN proname='registrar_incorporacion_ejercicio_v2' THEN 'v' ELSE 'i' END)::"char")
  OR EXISTS (SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a
   WHERE p.oid=f AND (a.grantor<>p.proowner OR (a.grantee<>p.proowner AND NOT (
    p.proname='registrar_incorporacion_ejercicio_v2' AND a.grantee='vec_contratacion_temporal_ejecutor'::regrole
    AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)))) THEN
   RAISE EXCEPTION 'CT75: funciones de retirada incompatibles' USING ERRCODE='55000'; END IF;
  funciones:=array_append(funciones,f);
 END LOOP;
 IF (SELECT count(*) FROM pg_proc p JOIN pg_namespace ns ON ns.oid=p.pronamespace
  WHERE ns.nspname='vec_contratacion_temporal' AND (p.proname LIKE 'incorporacion75_%' OR p.proname='registrar_incorporacion_ejercicio_v2'))<>8 THEN
  RAISE EXCEPTION 'CT75: sobrecarga conservada' USING ERRCODE='55000'; END IF;
 SELECT array_agg(c.oid),array_agg(c.reltype) INTO ids,tipos FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace
  WHERE ns.nspname='vec_contratacion_temporal' AND c.relname IN ('incorporacion_registro_v2','incorporacion_auditoria_v2','incorporacion_outbox_v2');
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_registro_v2)
 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_auditoria_v2)
 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.incorporacion_outbox_v2) THEN
  RAISE EXCEPTION 'CT75: historia conservada' USING ERRCODE='55000'; END IF;
 IF EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid<>ALL(funciones) AND (
   strpos(p.prosrc,'registrar_incorporacion_ejercicio_v2')>0 OR strpos(p.prosrc,'incorporacion75_')>0
   OR strpos(p.prosrc,'incorporacion_registro_v2')>0 OR strpos(p.prosrc,'incorporacion_auditoria_v2')>0 OR strpos(p.prosrc,'incorporacion_outbox_v2')>0))
 OR EXISTS (SELECT 1 FROM pg_constraint WHERE confrelid=ANY(ids) AND conrelid<>ALL(ids))
 OR EXISTS (SELECT 1 FROM pg_depend d JOIN pg_rewrite r ON d.classid='pg_rewrite'::regclass AND r.oid=d.objid
  WHERE d.refclassid='pg_class'::regclass AND d.refobjid=ANY(ids) AND r.ev_class<>ALL(ids))
 OR EXISTS (SELECT 1 FROM pg_depend d WHERE
  (d.refclassid='pg_proc'::regclass AND d.refobjid=ANY(funciones) AND NOT(d.classid='pg_proc'::regclass AND d.objid=ANY(funciones)))
  OR (d.classid='pg_proc'::regclass AND d.objid<>ALL(funciones) AND (
   (d.refclassid='pg_class'::regclass AND d.refobjid=ANY(ids)) OR (d.refclassid='pg_type'::regclass AND d.refobjid=ANY(tipos))))) THEN
  RAISE EXCEPTION 'CT75: dependencia conservada' USING ERRCODE='55000'; END IF;
 -- Las únicas FK circulares que se quitan se identifican por columnas exactas.
 FOR fk,atributo,destino IN SELECT * FROM (VALUES
  ('incorporacion_registro_v2_auditoria_fk','auditoria_ref','incorporacion_auditoria_v2'),
  ('incorporacion_registro_v2_outbox_fk','outbox_ref','incorporacion_outbox_v2')) v LOOP
  SELECT c.oid INTO objetivo FROM pg_class c JOIN pg_namespace ns ON ns.oid=c.relnamespace WHERE ns.nspname='vec_contratacion_temporal' AND c.relname=destino;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint co WHERE co.conname=fk AND co.conrelid='vec_contratacion_temporal.incorporacion_registro_v2'::regclass
   AND co.confrelid=objetivo AND co.contype='f' AND co.condeferrable AND co.condeferred AND co.convalidated AND co.confmatchtype='s'
   AND co.confupdtype='a' AND co.confdeltype='a' AND co.conislocal AND co.coninhcount=0 AND co.conparentid=0 AND co.confdelsetcols IS NULL
   AND co.conkey=ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid=co.conrelid AND attname=atributo AND attnum>0 AND NOT attisdropped)]
   AND co.confkey=ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid=co.confrelid AND attname=atributo AND attnum>0 AND NOT attisdropped)]) THEN
   RAISE EXCEPTION 'CT75: FK circular alterada' USING ERRCODE='55000'; END IF;
 END LOOP;
END $retirada$;
DROP FUNCTION vec_contratacion_temporal.registrar_incorporacion_ejercicio_v2(jsonb,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bytea,bigint,bigint,bytea,bytea,bytea,bytea,jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_evidencia(jsonb,jsonb,bytea[],numeric,numeric,timestamptz) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_solicitud_sha256(jsonb) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_recibo(jsonb,text,jsonb,jsonb,jsonb,jsonb,jsonb,text,text) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_autoridad(jsonb,bytea[],numeric,numeric,timestamptz) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_ventana(jsonb,jsonb,timestamptz) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_piezas(bytea[],numeric,numeric) RESTRICT;
DROP FUNCTION vec_contratacion_temporal.incorporacion75_instante(timestamptz) RESTRICT;
ALTER TABLE vec_contratacion_temporal.incorporacion_registro_v2 DROP CONSTRAINT incorporacion_registro_v2_auditoria_fk;
ALTER TABLE vec_contratacion_temporal.incorporacion_registro_v2 DROP CONSTRAINT incorporacion_registro_v2_outbox_fk;
DROP TABLE vec_contratacion_temporal.incorporacion_outbox_v2 RESTRICT;
DROP TABLE vec_contratacion_temporal.incorporacion_auditoria_v2 RESTRICT;
DROP TABLE vec_contratacion_temporal.incorporacion_registro_v2 RESTRICT;
COMMIT;
