\set ON_ERROR_STOP on
-- CT82: enlazar seguimiento con las referencias originales de Contratación.
-- No convierte identificadores, no reescribe historia ni modifica autoridad.
-- Migración de avance: conservar respaldo; no revertir sobre nuevas raíces.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal.dependencias.incorporacion_ejercicio.v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000073:seguimiento:v1',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000078:lectura_raices:v2',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_contratacion_temporal:000082:referencias-originales',0));
SET LOCAL ROLE vec_contratacion_temporal_propietario;
LOCK TABLE vec_contratacion_temporal.seguimiento_raiz_v2 IN ACCESS EXCLUSIVE MODE;

DO $ct82$
DECLARE
 propietario oid:='vec_contratacion_temporal_propietario'::regrole;
 r record; p pg_proc%ROWTYPE; f oid; antes jsonb; nuevo text; ddl text; c text;
 escalar_origen text:=$old$  WHEN 'hash' THEN s ~ '^[0-9a-f]{64}$' AND s<>repeat('0',64)$old$;
 escalar_destino text:=$new$  WHEN 'organizacion' THEN
   (s ~ '^ref:[0-9a-f]{64}$' AND s<>'ref:'||repeat('0',64))
   OR (octet_length(s) BETWEEN 14 AND 160 AND s ~ '^organizacion:[A-Za-z0-9._:/#-]+$')
  WHEN 'expediente' THEN
   (s ~ '^ref:[0-9a-f]{64}$' AND s<>'ref:'||repeat('0',64))
   OR (s ~ '^expediente:ct:[0-9a-f]{64}$' AND s<>'expediente:ct:'||repeat('0',64))
  WHEN 'hash' THEN s ~ '^[0-9a-f]{64}$' AND s<>repeat('0',64)$new$;
 selector_origen text:=$old$ FOREACH v_selector IN ARRAY ARRAY[p_organizacion_ref,p_expediente_ref,p_relacion_ref] LOOP
  IF v_selector IS NULL OR octet_length(v_selector)<>68
   OR v_selector !~ '^ref:[0-9a-f]{64}$' OR v_selector='ref:'||repeat('0',64)
  THEN RAISE EXCEPTION 'CT78: selector invalido' USING ERRCODE='22023'; END IF;
 END LOOP;$old$;
 selector_destino text:=$new$ IF (
  ((p_organizacion_ref ~ '^ref:[0-9a-f]{64}$' AND p_organizacion_ref<>'ref:'||repeat('0',64))
   OR (octet_length(p_organizacion_ref) BETWEEN 14 AND 160
    AND p_organizacion_ref ~ '^organizacion:[A-Za-z0-9._:/#-]+$'))
  AND ((p_expediente_ref ~ '^ref:[0-9a-f]{64}$' AND p_expediente_ref<>'ref:'||repeat('0',64))
   OR (p_expediente_ref ~ '^expediente:ct:[0-9a-f]{64}$'
    AND p_expediente_ref<>'expediente:ct:'||repeat('0',64)))
  AND (p_relacion_ref ~ '^ref:[0-9a-f]{64}$' AND p_relacion_ref<>'ref:'||repeat('0',64))
 ) IS NOT TRUE THEN
  RAISE EXCEPTION 'CT78: selector invalido' USING ERRCODE='22023';
 END IF;$new$;
BEGIN
 IF getdatabaseencoding()<>'UTF8' OR current_user<>'vec_contratacion_temporal_propietario'
 OR NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname='vec_contratacion_temporal' AND nspowner=propietario)
 OR NOT EXISTS (SELECT 1 FROM pg_roles WHERE oid=propietario AND NOT rolcanlogin AND NOT rolinherit
  AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolreplication AND NOT rolbypassrls)
 THEN RAISE EXCEPTION 'CT82: propietario incompatible' USING ERRCODE='55000'; END IF;

 -- Las dos restricciones originales deben seguir presentes y validadas.
 FOREACH c IN ARRAY ARRAY['organizacion_ref','expediente_ref'] LOOP
  IF NOT EXISTS (
   SELECT 1 FROM pg_constraint
   WHERE conrelid='vec_contratacion_temporal.seguimiento_raiz_v2'::regclass
    AND conname='seguimiento_raiz_v2_'||c||'_check' AND contype='c' AND convalidated
    AND pg_get_constraintdef(oid)=format(
     'CHECK (((%s ~ ''^ref:[0-9a-f]{64}$''::text) AND (%s <> (''ref:''::text || repeat(''0''::text, 64)))))',c,c)
  ) THEN RAISE EXCEPTION 'CT82: restricción previa incompatible' USING ERRCODE='55000'; END IF;
 END LOOP;

 -- Sustituciones acotadas sobre los mismos OID, firmas, atributos y ACL.
 FOR r IN SELECT * FROM (VALUES
  ('seguimiento73_escalar(jsonb,text)','dfedf16246c9170b83eec2c017c9f3c75427758007f3258d9bd1f436215f2dc5',false),
  ('seguimiento73_nodo(jsonb,text)','1c2829bdf85638105a76252df5491d10216f9f6ee17a05f0ac176dd14d1e6e4b',false),
  ('leer_raices_incorporacion_ejercicio_v2(text,text,text)','27bdd1f5f03621744f3f8721b6eaa87b199d2f92f9a0e4ff42cbfdc14a7d5c60',true)
 ) AS cambios(firma,sha256,definidor) LOOP
  f:=to_regprocedure('vec_contratacion_temporal.'||r.firma);
  SELECT * INTO STRICT p FROM pg_proc WHERE oid=f;
  IF p.proowner<>propietario OR p.prosecdef IS DISTINCT FROM r.definidor
   OR p.pronamespace<>'vec_contratacion_temporal'::regnamespace
   OR encode(sha256(convert_to(p.prosrc,'UTF8')),'hex') IS DISTINCT FROM r.sha256
  THEN RAISE EXCEPTION 'CT82: función previa incompatible' USING ERRCODE='55000'; END IF;
  antes:=to_jsonb(p)-'prosrc';
  IF p.proname='seguimiento73_escalar' THEN
   IF (length(p.prosrc)-length(replace(p.prosrc,escalar_origen,'')))/length(escalar_origen)<>1
   THEN RAISE EXCEPTION 'CT82: escalar no unívoco' USING ERRCODE='55000'; END IF;
   nuevo:=replace(p.prosrc,escalar_origen,escalar_destino);
  ELSIF p.proname='seguimiento73_nodo' THEN
   IF (length(p.prosrc)-length(replace(p.prosrc,'organizacion_ref:ref|expediente_ref:ref','')))/length('organizacion_ref:ref|expediente_ref:ref')<>2
    OR (length(p.prosrc)-length(replace(p.prosrc,'''u1'',''ref'',''hash''','')))/length('''u1'',''ref'',''hash''')<>1
   THEN RAISE EXCEPTION 'CT82: nodo no unívoco' USING ERRCODE='55000'; END IF;
   nuevo:=replace(p.prosrc,'organizacion_ref:ref|expediente_ref:ref','organizacion_ref:organizacion|expediente_ref:expediente');
   nuevo:=replace(nuevo,'''u1'',''ref'',''hash''','''u1'',''ref'',''organizacion'',''expediente'',''hash''');
  ELSE
   IF (length(p.prosrc)-length(replace(p.prosrc,selector_origen,'')))/length(selector_origen)<>1
   THEN RAISE EXCEPTION 'CT82: selector no unívoco' USING ERRCODE='55000'; END IF;
   nuevo:=replace(p.prosrc,selector_origen,selector_destino);
  END IF;
  ddl:=pg_get_functiondef(f);
  IF (length(ddl)-length(replace(ddl,p.prosrc,'')))/length(p.prosrc)<>1
  THEN RAISE EXCEPTION 'CT82: cuerpo no unívoco' USING ERRCODE='55000'; END IF;
  EXECUTE replace(ddl,p.prosrc,nuevo);
  IF (SELECT to_jsonb(z)-'prosrc' FROM pg_proc z WHERE z.oid=f) IS DISTINCT FROM antes
   OR (SELECT prosrc FROM pg_proc WHERE oid=f) IS DISTINCT FROM nuevo
  THEN RAISE EXCEPTION 'CT82: atributos o postimagen modificados' USING ERRCODE='55000'; END IF;
 END LOOP;
END $ct82$;

-- Se amplía la sintaxis sólo de estos campos. La FK al expediente y todas las
-- referencias propias de seguimiento conservan sus restricciones originales.
ALTER TABLE vec_contratacion_temporal.seguimiento_raiz_v2
 DROP CONSTRAINT seguimiento_raiz_v2_organizacion_ref_check,
 DROP CONSTRAINT seguimiento_raiz_v2_expediente_ref_check,
 ADD CONSTRAINT seguimiento_raiz_v2_organizacion_ref_check CHECK (
  (organizacion_ref ~ '^ref:[0-9a-f]{64}$' AND organizacion_ref<>'ref:'||repeat('0',64))
  OR (octet_length(organizacion_ref) BETWEEN 14 AND 160 AND organizacion_ref ~ '^organizacion:[A-Za-z0-9._:/#-]+$')),
 ADD CONSTRAINT seguimiento_raiz_v2_expediente_ref_check CHECK (
  (expediente_ref ~ '^ref:[0-9a-f]{64}$' AND expediente_ref<>'ref:'||repeat('0',64))
  OR (expediente_ref ~ '^expediente:ct:[0-9a-f]{64}$' AND expediente_ref<>'expediente:ct:'||repeat('0',64)));
COMMIT;
