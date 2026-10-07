\set ON_ERROR_STOP on
-- SÓLO CLON DESECHABLE, como superusuario, después de publicar Rol7 con la CLI.
-- Toma de una decisión V3 real de vec-admin la persona, el perfil, la asignación
-- actual en Rol7 y su vínculo de autenticación, y comprueba con ellos:
--  1. la puerta del lote acredita Rol7 (y no la versión anterior);
--  2. AD177 acepta el gobierno del plan con la categoría Aplicación real.
-- Las filas de consumo del punto 2 son sintéticas y se deshacen con ROLLBACK
-- (patrón de ad177_ad178_positivo_sintetico_clon.sql); la categoría no se sustituye.
BEGIN ISOLATION LEVEL SERIALIZABLE READ WRITE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL statement_timeout='30s';
CREATE TEMP TABLE real_admin(valor jsonb) ON COMMIT DROP;
INSERT INTO real_admin
SELECT convert_from(a.decision_canonica,'UTF8')::jsonb FROM vec_autorizacion_atestada_v3.atestacion_decision_v3 a
 WHERE convert_from(a.decision_canonica,'UTF8')::jsonb->>'version_rol_ref'='rol:administracion_perfiles:v7'
   AND convert_from(a.decision_canonica,'UTF8')::jsonb#>>'{vinculo_autenticacion_actor,superficie}'='administracion_privilegiada'
 ORDER BY a.registrada_en DESC LIMIT 1;
DO $p$
DECLARE d jsonb;
BEGIN
 SELECT valor INTO d FROM real_admin;
 IF d IS NULL THEN RAISE EXCEPTION 'AUT51: no hay decisión real de vec-admin con Rol7; recorrer antes la lista de usuarios'; END IF;
 IF vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles','[]',d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT51: la puerta del lote no acredita Rol7'; END IF;
 IF vec_autorizacion.acreditar_perfil_aplicacion_lote_ordinario_v1('rol:administracion_perfiles:v6',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'administracion.perfiles.aplicar_lote_ordinario','administracion','persona','gestion_perfiles','[]',d->'vinculo_autenticacion_actor') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT51: la puerta del lote acepta una versión no actual'; END IF;
 IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'vec.catalogos.publicar','contratacion_temporal','catalogo_configurable','gestionar_contratacion_temporal','[]',d->'vinculo_autenticacion_actor') IS NOT TRUE
 THEN RAISE EXCEPTION 'AUT51: la categoría Aplicación no acredita vec.catalogos.publicar en Rol7'; END IF;
 -- Fuera del plan: otro módulo, otro tipo o una acción general de catálogos no se acreditan.
 IF vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'vec.catalogos.publicar','catalogos','catalogo_configurable','gestionar_contratacion_temporal','[]',d->'vinculo_autenticacion_actor') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'vec.catalogos.publicar','contratacion_temporal','catalogo','gestionar_contratacion_temporal','[]',d->'vinculo_autenticacion_actor') IS NOT FALSE
 OR vec_autorizacion.acreditar_perfil_aplicacion_nominal_v1(d->>'version_rol_ref',d->>'asignacion_ref',d->>'principal_id',d->>'perfil_activo_ref',
   'vec.catalogos.borrar','contratacion_temporal','catalogo_configurable','gestionar_contratacion_temporal','[]',d->'vinculo_autenticacion_actor') IS NOT FALSE
 THEN RAISE EXCEPTION 'AUT51: la categoría acredita fuera del plan nominal de firma'; END IF;
END $p$;
-- AD177: siembra de consumo sintético con la decisión real adaptada al gobierno.
SET LOCAL session_replication_role=replica;
DO $preparar$
DECLARE r record;
BEGIN
 FOR r IN SELECT c.conrelid::regclass AS t,c.conname FROM pg_constraint c
  WHERE c.conrelid IN ('vec_autorizacion_atestada_v3.consumo_decision_v3'::regclass,
   'vec_autorizacion_atestada_v3.auditoria_consumo_v3'::regclass,
   'vec_autorizacion_atestada_v3.atestacion_decision_v3'::regclass) AND c.contype IN ('c','f') LOOP
  EXECUTE format('ALTER TABLE %s DROP CONSTRAINT %I',r.t,r.conname);
 END LOOP;
END $preparar$;
CREATE TEMP TABLE recibo_gobierno(valor jsonb) ON COMMIT DROP;
DO $sembrar$
DECLARE real jsonb; d jsonb; dc bytea; cc bytea; hd text; cap jsonb; ahora timestamptz(6):=clock_timestamp();
 contexto text:=repeat('e',64); hc text;
BEGIN
 SELECT valor INTO real FROM real_admin;
 d:=jsonb_build_object('decision_ref','decision:aut51-gobierno','concedida',true,'codigo','concedida',
  'accion','vec.catalogos.publicar','modulo_id','contratacion_temporal','tipo_recurso','catalogo_configurable',
  'finalidad','gestionar_contratacion_temporal','recurso_ref','ct.plan.firma:1','contexto_recurso_huella_sha256',contexto,
  'vinculo_autenticacion_actor',real->'vinculo_autenticacion_actor',
  'campos_permitidos','[]'::jsonb,'obligaciones','[]'::jsonb,'principal_id',real->>'principal_id','perfil_activo_ref',real->>'perfil_activo_ref',
  'version_rol_ref',real->>'version_rol_ref','asignacion_ref',real->>'asignacion_ref',
  'valida_hasta',to_char(clock_timestamp()+interval '10 minutes','YYYY-MM-DD"T"HH24:MI:SS"Z"'));
 dc:=convert_to(d::text,'UTF8'); hd:=encode(sha256(dc),'hex');
 cap:=jsonb_build_object('operacion','vec.catalogos.publicar','audiencia_consumo','vec_catalogos_configurables.plan_nominal_firma.gobierno.v1',
  'efecto_ref','ct.plan.firma:1','huella_efecto_sha256',contexto,'huella_decision_sha256',hd);
 cc:=convert_to(cap::text,'UTF8'); hc:=encode(sha256(convert_to('consumo:aut51','UTF8')),'hex');
 INSERT INTO vec_autorizacion_atestada_v3.atestacion_decision_v3(decision_ref,huella_decision_sha256,decision_canonica,
  motivo_canonico,contexto_actor_canonico,payload_vec_ad_3,sobre_cose_sign1,evidencia_verificacion,raiz_publica_spki,
  capacidad_canonica,huella_capacidad_sha256,efecto_ref,huella_efecto_sha256,registrada_en)
 VALUES('decision:aut51-gobierno',hd,dc,'\x00','\x00','\x00','\x00','\x00','\x00',cc,encode(sha256(cc),'hex'),'ct.plan.firma:1',contexto,ahora);
 INSERT INTO vec_autorizacion_atestada_v3.consumo_decision_v3(decision_ref,huella_decision_sha256,nonce,efecto_ref,
  huella_efecto_sha256,consumo_huella_sha256,consumida_en,transaccion_origen)
 VALUES('decision:aut51-gobierno',hd,'n-aut51','ct.plan.firma:1',contexto,hc,ahora,pg_current_xact_id());
 INSERT INTO vec_autorizacion_atestada_v3.auditoria_consumo_v3(auditoria_ref,secuencia,decision_ref,efecto_ref,
  huella_efecto_sha256,anterior_sha256,huella_sha256,registrada_en,tipo_registro,version_consumo,actor_ref,
  perfil_activo_ref,finalidad_ref,proceso,canal,transaccion_origen)
 VALUES('aud-aut51',999999001,'decision:aut51-gobierno','ct.plan.firma:1',contexto,repeat('0',64),encode(sha256(convert_to('auditoria:aut51','UTF8')),'hex'),ahora,'consumo_confirmado_v4',4,
  d->>'principal_id',d->>'perfil_activo_ref',d->>'finalidad','prueba','sql',pg_current_xact_id());
 INSERT INTO recibo_gobierno VALUES(jsonb_build_object('decision_ref','decision:aut51-gobierno','efecto_ref','ct.plan.firma:1','huella_efecto_sha256',contexto,
  'consumo_huella_sha256',hc,'auditoria_ref','aud-aut51','consumida_en',ahora,'consumo_nuevo',true));
END $sembrar$;
GRANT SELECT ON recibo_gobierno TO vec_catalogos_configurables_propietario;
SET LOCAL ROLE vec_catalogos_configurables_propietario;
DO $gobierno$
DECLARE r jsonb; x jsonb;
BEGIN
 SELECT valor INTO r FROM recibo_gobierno;
 x:=vec_autorizacion_atestada_v3.comprobar_consumo_gobierno_plan_firma_v1(r);
 IF x->>'operacion'<>'vec.catalogos.publicar' OR x->>'decision_ref'<>'decision:aut51-gobierno' THEN
  RAISE EXCEPTION 'AUT51: AD177 devolvió otro resultado: %',x; END IF;
END $gobierno$;
RESET ROLE;
ROLLBACK;
SELECT 'AUT51-POSITIVO-CLON-OK; consumo sintético deshecho, categoría Aplicación real';
