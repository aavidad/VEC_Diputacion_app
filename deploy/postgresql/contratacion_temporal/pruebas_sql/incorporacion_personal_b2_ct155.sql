\set ON_ERROR_STOP on
-- Comprobación focal de estructura/ACL y codec, no un E2E Personal/RPT.
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path=pg_catalog;
DO $prueba$
DECLARE t text;col text;f record;
BEGIN
 FOREACH t IN ARRAY ARRAY['cese_nombramiento_v1','confirmacion_ginpix_v1','reincorporacion_titular_v1'] LOOP
 IF NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=('vec_contratacion_temporal.'||t)::regclass AND c.contype='f' AND c.confrelid='vec_contratacion_temporal.incorporacion_registro_v2'::regclass)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=('vec_contratacion_temporal.'||t)::regclass AND c.conname='origen_b2_ct155_fk' AND c.contype='f' AND c.confmatchtype='s' AND c.confrelid='vec_contratacion_temporal.origen_incorporacion_personal_b2_v1'::regclass AND c.convalidated)
 OR NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=('vec_contratacion_temporal.'||t)::regclass AND c.conname='origen_xor_ct155' AND c.contype='c' AND c.convalidated AND strpos(pg_get_constraintdef(c.oid),'IS NOT NULL')>0)
 THEN RAISE EXCEPTION 'CT155: FK histórica, B2 o XOR ausente: %',t;END IF;
 END LOOP;
 FOREACH t IN ARRAY ARRAY['plan_incorporacion_personal_b2_v1','origen_incorporacion_personal_b2_v1'] LOOP
 IF has_table_privilege('vec_contratacion_temporal_ejecutor','vec_contratacion_temporal.'||t,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR NOT EXISTS(SELECT 1 FROM pg_class WHERE oid=('vec_contratacion_temporal.'||t)::regclass AND relrowsecurity AND relforcerowsecurity)
 THEN RAISE EXCEPTION 'CT155: ACL o RLS incompleta: %',t;END IF;
 END LOOP;
 FOR f IN SELECT p.oid,p.proname,p.prosecdef,p.proconfig FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname='vec_contratacion_temporal' AND (p.proname LIKE '%ct155' OR p.proname IN ('registrar_plan_nominal_b2_v1','leer_plan_nominal_b2_v1','confirmar_origen_incorporacion_b2_v1','leer_origen_incorporacion_b2_v1','leer_antecedentes_plan_b2_v1')) LOOP
 IF EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) x WHERE p.oid=f.oid AND x.grantee=0) THEN RAISE EXCEPTION 'CT155: EXECUTE PUBLIC: %',f.proname;END IF;
	 IF f.prosecdef AND f.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on']
	 THEN RAISE EXCEPTION 'CT155: entorno SECURITY DEFINER incompatible: %',f.proname; END IF;
 IF f.proname LIKE '%ct155' AND has_function_privilege('vec_contratacion_temporal_ejecutor',f.oid,'EXECUTE') THEN RAISE EXCEPTION 'CT155: auxiliar alcanzable: %',f.proname;END IF;
 END LOOP;
 IF EXISTS (
  SELECT 1 FROM (VALUES
   ('confirmar_confirmacion_ginpix_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=utc','lock_timeout=2s']),
   ('leer_antecedente_reincorporacion_titular_atestada_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
    ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=utc','lock_timeout=2s'])
  ) v(firma,config) LEFT JOIN pg_proc p ON p.oid=to_regprocedure('vec_contratacion_temporal.'||v.firma)
  WHERE p.oid IS NULL OR NOT p.prosecdef OR p.proowner<>'vec_contratacion_temporal_propietario'::regrole
     OR ARRAY(SELECT lower(c) FROM unnest(p.proconfig) WITH ORDINALITY AS u(c,n) ORDER BY n)
        IS DISTINCT FROM v.config
 ) THEN RAISE EXCEPTION 'CT155: entorno heredado o metadatos de consumidor alterados'; END IF;
 IF vec_contratacion_temporal.canon_plan_personal_ct155('{"z":9007199254740991,"a":{"z":"<&>","a":"fecha"}}'::jsonb) IS DISTINCT FROM '{"a":{"a":"fecha","z":"\u003c\u0026\u003e"},"z":9007199254740991}' THEN RAISE EXCEPTION 'CT155: codec Go/SQL divergente';END IF;
 BEGIN PERFORM vec_contratacion_temporal.canon_plan_personal_ct155('{"z":9007199254740992}'::jsonb);RAISE EXCEPTION 'CT155: entero fuera de rango admitido';EXCEPTION WHEN invalid_parameter_value THEN NULL;END;
END $prueba$;
-- Sintaxis Personal23/Go: este bloque solo llama al validador CT real.
-- Los ejemplos no acreditan pertenencia al catálogo propietario Personal.
DO $gramatica$
DECLARE p jsonb:='{}';s jsonb;k text;clase text;valor jsonb;
BEGIN
 FOREACH k IN ARRAY ARRAY['organizacion_ref','unidad_ct_ref','expediente_ref','analisis_recibo_ref','propuesta_recibo_ref','aceptacion_ref','aceptacion_recibo_ref','persona_ref','persona_recibo_bolsa_ref','organismo_ref','unidad_ref','version_plaza_ref','version_puesto_ref','puesto_ref','plaza_ref','catalogo_rpt_id','catalogo_rpt_modulo','categoria_ref','vinculo_recibo_ref','documento_ref'] LOOP
 p:=p||jsonb_build_object(k,'ref:gramatica');
 END LOOP;
 FOREACH k IN ARRAY ARRAY['version_expediente','analisis_version','persona_version','revision_plaza','revision_puesto','catalogo_rpt_version','vinculo_revision'] LOOP
 p:=p||jsonb_build_object(k,1);
 END LOOP;
 FOREACH k IN ARRAY ARRAY['analisis_sha256','catalogo_rpt_sha256','documento_sha256'] LOOP
 p:=p||jsonb_build_object(k,repeat('1',64));
 END LOOP;
 p:=p||jsonb_build_object(
 'persona_fuente',jsonb_build_object('ref','ref:gramatica','version',1,'sha256',repeat('1',64)),
 'fuente_organizacion',jsonb_build_object('ref','ref:gramatica','sha256',repeat('1',64)),
 'fuente_plantilla',jsonb_build_object('ref','11111111-1111-4111-8111-111111111111','revision',1,'fuente_ref','ref:gramatica','fuente_sha256',repeat('1',64)),
 'fuente_rpt',jsonb_build_object('ref','22222222-2222-4222-8222-222222222222','revision',1,'fuente_ref','ref:gramatica','fuente_sha256',repeat('1',64)),
 'regimen',jsonb_build_object('ref','catalogo:gramatica','version',1),
 'modalidad',jsonb_build_object('ref','catalogo:gramatica','version',1),
 'bolsa',jsonb_build_object('unidad_ref','ref:gramatica','categoria_ref','ref:gramatica','necesidad_ref','ref:gramatica','aceptacion_operacion_ref','ref:gramatica','aceptacion_registro_sha256',repeat('1',64),'apertura_operacion_ref','ref:gramatica','apertura_registro_sha256',repeat('1',64),'llamamiento_ref','ref:gramatica','propuesta_ref','ref:gramatica'),
 'desde','2026-09-01','hasta','2026-12-31','motivo_clave','motivo_prueba','ejercicio_sintetico',true);
 SELECT jsonb_object_agg(c,p->c) INTO s FROM unnest(ARRAY['organizacion_ref','expediente_ref','version_expediente','puesto_ref','plaza_ref','regimen','modalidad','desde','hasta','motivo_clave','documento_ref','documento_sha256']) c;
 s:=s||jsonb_build_object('clave_idempotencia','33333333-3333-4333-8333-333333333333','version_plantilla_ref',p#>>'{fuente_plantilla,ref}','version_rpt_ref',p#>>'{fuente_rpt,ref}');
 -- Límites y caracteres expresados como casos, sin repetir el regex.
 FOREACH clase IN ARRAY ARRAY['a','a0_','reserva',repeat('a',64)] LOOP
 PERFORM vec_contratacion_temporal.validar_plan_personal_ct155(p||jsonb_build_object('clase_ocupacion',clase),s||jsonb_build_object('clase_ocupacion',clase));
 END LOOP;
 FOR valor IN SELECT value FROM jsonb_array_elements(jsonb_build_array('','A','1a','a-b','a.b',repeat('a',65),7,true,NULL)) LOOP
 BEGIN
 PERFORM vec_contratacion_temporal.validar_plan_personal_ct155(p||jsonb_build_object('clase_ocupacion',valor),s||jsonb_build_object('clase_ocupacion',valor));
 RAISE EXCEPTION 'CT155: clase sintácticamente inválida admitida: %',valor;
 EXCEPTION WHEN invalid_parameter_value THEN NULL;
 END;
 END LOOP;
END $gramatica$;
-- CT155 positiva: plan y origen usan operaciones distintas aunque ambos
-- eventos conservan el mismo plan_ref. Ejercita el helper y la restricción
-- UNIQUE real del outbox, con rollback; no simula autorización Personal.
DO $outbox$
DECLARE e record; p text:='plan:prueba:ct155'; ahora timestamptz:=clock_timestamp();
BEGIN
 SELECT expediente_ref,version INTO STRICT e FROM vec_contratacion_temporal.expediente_version_integral ORDER BY expediente_ref,version LIMIT 1;
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155('org:prueba:ct155',e.expediente_ref,e.version,p,'evento:prueba:ct155:plan','ct.plan-incorporacion-personal.v1','recibo:prueba:ct155:plan',ahora);
 PERFORM vec_contratacion_temporal.evento_plan_personal_ct155('org:prueba:ct155',e.expediente_ref,e.version,p,'evento:prueba:ct155:origen','ct.incorporacion-personal.v1','recibo:prueba:ct155:origen',ahora);
 IF (SELECT count(DISTINCT operacion_ref) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE evento_ref IN ('evento:prueba:ct155:plan','evento:prueba:ct155:origen'))<>2
 OR (SELECT count(*) FROM vec_contratacion_temporal.outbox_expediente_integral WHERE evento_ref IN ('evento:prueba:ct155:plan','evento:prueba:ct155:origen') AND convert_from(payload_canonico,'UTF8')::jsonb->>'plan_ref'=p)<>2
 THEN RAISE EXCEPTION 'CT155: plan y origen colisionan o pierden vínculo'; END IF;
END $outbox$;
ROLLBACK;
