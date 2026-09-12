\set ON_ERROR_STOP on
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_personal.dependencias.alta_ejercicio.v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000072:registro:v2',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;

-- Estabilizar también DDL/policies antes de inspeccionar el catálogo. Si falta
-- un objeto o cambió de propietario, LOCK falla sin retirar nada.
LOCK TABLE vec_contratacion_temporal.seguimiento_definicion_v2,
 vec_contratacion_temporal.seguimiento_raiz_v2,
 vec_contratacion_temporal.seguimiento_estado_v2 IN ACCESS EXCLUSIVE MODE;

-- No IF EXISTS: no retira una instalación parcial/ajena. Las guardas previas
-- a SELECT impiden que una policy alterada o un cambio de dueño oculte historia.
DO $inventario$
DECLARE tabla text; t oid; columnas integer; esperado integer; i integer:=0;
BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname=current_user
     AND NOT rolcanlogin AND NOT rolsuper AND NOT rolinherit AND NOT rolcreatedb
     AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal'
     AND nspowner=current_user::regrole) THEN
  RAISE EXCEPTION 'CT72: autoridad de retirada incompatible' USING ERRCODE='55000';
 END IF;
 FOREACH tabla IN ARRAY ARRAY['seguimiento_definicion_v2','seguimiento_raiz_v2','seguimiento_estado_v2'] LOOP
  i:=i+1; esperado:=(ARRAY[5,11,14])[i];
  SELECT c.oid INTO t FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
   WHERE n.nspname='vec_contratacion_temporal' AND c.relname=tabla
     AND c.relkind='r' AND c.relowner=current_user::regrole
     AND c.relrowsecurity AND c.relforcerowsecurity AND NOT c.relispartition
     AND obj_description(c.oid,'pg_class')='CT72:fundacion-seguimiento-v2;sin-api-registro;codec-dominio-pendiente';
  IF t IS NULL THEN
   RAISE EXCEPTION 'CT72: inventario de retirada incompatible' USING ERRCODE='55000';
  END IF;
  SELECT count(*) INTO columnas FROM pg_attribute WHERE attrelid=t AND attnum>0 AND NOT attisdropped;
  IF columnas<>esperado OR EXISTS (SELECT 1 FROM pg_attribute WHERE attrelid=t AND attnum>0
       AND (attisdropped OR attacl IS NOT NULL))
  OR EXISTS (SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(COALESCE(c.relacl,acldefault('r',c.relowner))) a
       WHERE c.oid=t AND (a.grantee<>c.relowner OR a.grantor<>c.relowner))
  OR (SELECT count(*) FROM pg_policy WHERE polrelid=t)<>1
  OR NOT EXISTS (SELECT 1 FROM pg_policy WHERE polrelid=t AND polname='propietario'
       AND polcmd='*' AND polpermissive AND polroles=ARRAY[current_user::regrole::oid]
       AND pg_get_expr(polqual,polrelid)='true' AND pg_get_expr(polwithcheck,polrelid)='true')
  OR (SELECT count(*) FROM pg_trigger WHERE tgrelid=t AND NOT tgisinternal)<>2
  OR NOT EXISTS (SELECT 1 FROM pg_trigger g JOIN pg_proc p ON p.oid=g.tgfoid
       JOIN pg_namespace n ON n.oid=p.pronamespace WHERE g.tgrelid=t
       AND NOT g.tgisinternal AND g.tgname='historia_inmutable' AND g.tgtype=27
       AND g.tgenabled='O' AND g.tgnargs=0 AND g.tgqual IS NULL
       AND n.nspname='vec_contratacion_temporal' AND p.proname='rechazar_mutacion_historia_v1'
       AND p.proowner=current_user::regrole AND p.pronargs=0 AND p.prorettype='trigger'::regtype)
  OR NOT EXISTS (SELECT 1 FROM pg_trigger g JOIN pg_proc p ON p.oid=g.tgfoid
       JOIN pg_namespace n ON n.oid=p.pronamespace WHERE g.tgrelid=t
       AND NOT g.tgisinternal AND g.tgname='historia_no_truncar' AND g.tgtype=34
       AND g.tgenabled='O' AND g.tgnargs=0 AND g.tgqual IS NULL
       AND n.nspname='vec_contratacion_temporal' AND p.proname='rechazar_mutacion_historia_v1'
       AND p.proowner=current_user::regrole AND p.pronargs=0 AND p.prorettype='trigger'::regtype)
  OR EXISTS (SELECT 1 FROM pg_inherits WHERE inhparent=t OR inhrelid=t)
  OR EXISTS (SELECT 1 FROM pg_index ix WHERE ix.indrelid=t AND NOT EXISTS
       (SELECT 1 FROM pg_constraint co WHERE co.conrelid=t AND co.conindid=ix.indexrelid AND co.contype IN ('p','u')))
  THEN RAISE EXCEPTION 'CT72: ACL, policy o estructura alterada' USING ERRCODE='55000'; END IF;
 END LOOP;
END $inventario$;

DO $historia_dependencias$
DECLARE ids oid[]; tipos oid[]; nombres text[]:=ARRAY[
 'seguimiento_definicion_v2','seguimiento_raiz_v2','seguimiento_estado_v2'];
BEGIN
 SELECT array_agg(c.oid),array_agg(c.reltype) INTO ids,tipos
 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='vec_contratacion_temporal' AND c.relname=ANY(nombres) AND c.relkind='r';
 -- Incluso una publicación sin raíz constituye historia gobernada: no se borra.
 IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.seguimiento_definicion_v2)
 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.seguimiento_raiz_v2)
 OR EXISTS (SELECT 1 FROM vec_contratacion_temporal.seguimiento_estado_v2) THEN
  RAISE EXCEPTION 'CT72: historia conservada; retirada denegada' USING ERRCODE='55000';
 END IF;
 -- Todas las sobrecargas de un futuro writer y menciones por PL/pgSQL (que
 -- pueden no figurar en pg_depend). No resolver funciones en esquemas ajenos.
 IF EXISTS (SELECT 1 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
     WHERE (n.nspname='vec_contratacion_temporal' AND p.proname='registrar_incorporacion_ejercicio_v2')
        OR EXISTS (SELECT 1 FROM unnest(nombres) nombre WHERE strpos(p.prosrc,nombre)>0))
 OR EXISTS (SELECT 1 FROM pg_constraint WHERE confrelid=ANY(ids) AND conrelid<>ALL(ids))
 OR EXISTS (SELECT 1 FROM pg_depend d JOIN pg_rewrite r ON d.classid='pg_rewrite'::regclass AND r.oid=d.objid
     WHERE d.refclassid='pg_class'::regclass AND d.refobjid=ANY(ids) AND r.ev_class<>ALL(ids))
 OR EXISTS (SELECT 1 FROM pg_depend d WHERE d.classid='pg_proc'::regclass
     AND ((d.refclassid='pg_class'::regclass AND d.refobjid=ANY(ids))
       OR (d.refclassid='pg_type'::regclass AND d.refobjid=ANY(tipos))))
 THEN RAISE EXCEPTION 'CT72: dependencia conservada; retirada denegada' USING ERRCODE='55000'; END IF;
 -- No se retira ninguna constraint ajena para facilitar DROP. La única arista
 -- circular gestionada se comprueba antes de romperla, con ambas tablas vacías.
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='seguimiento_raiz_v2_inicial_fk'
     AND conrelid='vec_contratacion_temporal.seguimiento_raiz_v2'::regclass
     AND confrelid='vec_contratacion_temporal.seguimiento_estado_v2'::regclass
     AND contype='f' AND condeferrable AND condeferred AND convalidated
     AND confupdtype='a' AND confdeltype='a' AND confmatchtype='s'
     AND conislocal AND coninhcount=0 AND conparentid=0 AND confdelsetcols IS NULL
     AND conkey=(SELECT array_agg(a.attnum ORDER BY k.orden)
       FROM unnest(ARRAY['seguimiento_ref','version_inicial']) WITH ORDINALITY k(nombre,orden)
       JOIN pg_attribute a ON a.attrelid='vec_contratacion_temporal.seguimiento_raiz_v2'::regclass
         AND a.attname=k.nombre AND a.attnum>0 AND NOT a.attisdropped)
     AND confkey=(SELECT array_agg(a.attnum ORDER BY k.orden)
       FROM unnest(ARRAY['seguimiento_ref','version_seguimiento']) WITH ORDINALITY k(nombre,orden)
       JOIN pg_attribute a ON a.attrelid='vec_contratacion_temporal.seguimiento_estado_v2'::regclass
         AND a.attname=k.nombre AND a.attnum>0 AND NOT a.attisdropped)) THEN
  RAISE EXCEPTION 'CT72: ligadura inicial alterada' USING ERRCODE='55000';
 END IF;
END $historia_dependencias$;
ALTER TABLE vec_contratacion_temporal.seguimiento_raiz_v2 DROP CONSTRAINT seguimiento_raiz_v2_inicial_fk;
-- RESTRICT es la última barrera para dependencias catalogadas no enumeradas.
-- Ante cualquier error toda la transacción (incluido el ALTER) se revierte.
DROP TABLE vec_contratacion_temporal.seguimiento_estado_v2 RESTRICT;
DROP TABLE vec_contratacion_temporal.seguimiento_raiz_v2 RESTRICT;
DROP TABLE vec_contratacion_temporal.seguimiento_definicion_v2 RESTRICT;
COMMIT;
