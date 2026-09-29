\set ON_ERROR_STOP on
-- Ejecutar únicamente en el clon desechable de dirección, después de CTX15.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
CREATE ROLE prueba_ctx15_candidato LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
CREATE ROLE prueba_ctx15_usuarios LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE INHERIT NOREPLICATION NOBYPASSRLS;
GRANT vec_contexto_actor_v1_candidato_externo TO prueba_ctx15_candidato WITH INHERIT TRUE, SET FALSE;
GRANT vec_contexto_actor_v1_usuarios_externo TO prueba_ctx15_usuarios WITH INHERIT TRUE, SET FALSE;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $prueba$
DECLARE snapshot jsonb; componente jsonb; k text; prefijo text; h text; actual record; antes jsonb; despues jsonb; caso integer;
BEGIN
 SELECT jsonb_build_array((SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto)) INTO antes;
 FOR caso IN 1..2 LOOP
  snapshot:=jsonb_build_object('provision_ref',CASE WHEN caso=1 THEN 'pce_' ELSE 'pue_' END||'ctx15_sintetico_00000000000','poblacion',CASE WHEN caso=1 THEN 'candidato' ELSE 'usuarios' END,'estado','activo','vinculo_candidato',null);
  FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
   IF k='vinculo_candidato' AND caso=2 THEN CONTINUE; END IF;
   prefijo:=CASE k WHEN 'cuenta' THEN 'cta_' WHEN 'persona' THEN 'per_' WHEN 'perfil' THEN 'prf_' WHEN 'contexto' THEN 'vca_' ELSE 'vin_' END;
   componente:=jsonb_build_object('referencia',prefijo||'ctx15_sintetico_0000000000'||CASE WHEN k='persona' THEN '1' ELSE caso::text END,'version',1,'procedencia_ref','prc_ctx15_sintetico_000000000001','procedencia_version',1,'procedencia_huella_sha256',repeat('a',64),'procedencia_autoridad','autoridad_maestra_acreditada','estado','activo','vigente_desde','2020-01-01T00:00:00.000000Z','vigente_hasta','2099-01-01T00:00:00.000000Z');
   IF k='vinculo_candidato' THEN componente:=componente||jsonb_build_object('candidato_ref','can_ctx15_sintetico_000000000001'); END IF;
   snapshot:=snapshot||jsonb_build_object(k,componente);
  END LOOP;
  h:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(snapshot,1);
  BEGIN
   PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(snapshot,0,null,repeat('f',64));
   RAISE EXCEPTION 'aceptó huella no aprobada';
  EXCEPTION WHEN insufficient_privilege THEN NULL; END;
  SELECT * INTO actual FROM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(snapshot,0,null,h);
  IF actual.version<>1 OR actual.huella_sha256<>h THEN RAISE EXCEPTION 'recibo provisión divergente'; END IF;
  BEGIN
   PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(snapshot,0,null,h);
   RAISE EXCEPTION 'aceptó CAS antiguo';
  EXCEPTION WHEN serialization_failure THEN NULL; END;
  IF vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(snapshot||jsonb_build_object('empleado_ref','emp_prohibido')) THEN RAISE EXCEPTION 'aceptó campo corporativo'; END IF;
  IF vec_contexto_actor_v1.snapshot_contexto_externo_valido_v1(snapshot||jsonb_build_object('poblacion',null)) THEN RAISE EXCEPTION 'aceptó familia ausente'; END IF;
 END LOOP;
 SELECT jsonb_build_array((SELECT count(*) FROM vec_contexto_actor_v1.proyeccion_cuenta_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.persona_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.perfil_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.vinculo_contexto_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.vinculo_referencia_versiones),(SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto)) INTO despues;
 IF antes IS DISTINCT FROM despues THEN RAISE EXCEPTION 'provisión escribió historia compartida'; END IF;
 IF NOT vec_contexto_actor_v1.acreditar_candidato_externo_v1('per_ctx15_sintetico_00000000001','prf_ctx15_sintetico_00000000001','can_ctx15_sintetico_000000000001') THEN RAISE EXCEPTION 'AUT16 no acredita propia'; END IF;
 IF vec_contexto_actor_v1.acreditar_candidato_externo_v1('per_ctx15_sintetico_00000000001','prf_ctx15_sintetico_00000000001','can_ctx15_ajeno_000000000000001') THEN RAISE EXCEPTION 'AUT16 acredita candidato ajeno'; END IF;
 IF NOT vec_contexto_actor_v1.acreditar_perfil_usuarios_externo_v1('cta_ctx15_sintetico_00000000002','per_ctx15_sintetico_00000000001','prf_ctx15_sintetico_00000000002') THEN RAISE EXCEPTION 'AUT17 no acredita propia'; END IF;
END $prueba$;
RESET ROLE;
SET LOCAL SESSION AUTHORIZATION prueba_ctx15_candidato;
SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_candidato_externo_v1();
SELECT clock_timestamp() AS solicitado \gset
SELECT * FROM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1('oca_ctx15_operacion_000000000001','rca_ctx15_registro_0000000000001','cta_ctx15_sintetico_00000000001','prf_ctx15_sintetico_00000000001',:'solicitado');
SELECT * FROM vec_contexto_actor_v1.resolver_contexto_candidato_externo_v1('oca_ctx15_operacion_000000000001','rca_ctx15_registro_0000000000001','cta_ctx15_sintetico_00000000001','prf_ctx15_sintetico_00000000001',:'solicitado');
DO $denegacion$ BEGIN
 BEGIN
  PERFORM vec_contexto_actor_v1.resolver_y_registrar_contexto_actor_v2('oca_ctx15_operacion_000000000002','rca_ctx15_registro_0000000000002','cta_ctx15_sintetico_00000000001','prf_ctx15_sintetico_00000000001','certificado','alto',clock_timestamp(),'{}'::text[]);
  RAISE EXCEPTION 'externo alcanzó resolutor compartido';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
 BEGIN
  PERFORM 1 FROM vec_contexto_actor_v1.registros_contexto_externo_v2;
  RAISE EXCEPTION 'runtime lee tabla propia directamente';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $denegacion$;
RESET SESSION AUTHORIZATION;
SET LOCAL SESSION AUTHORIZATION prueba_ctx15_usuarios;
SELECT * FROM vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1();
SELECT * FROM vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1('oca_ctx15_operacion_000000000003','rca_ctx15_registro_0000000000003','cta_ctx15_sintetico_00000000002','prf_ctx15_sintetico_00000000002',clock_timestamp());
RESET SESSION AUTHORIZATION;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
DO $recibo_y_revocacion$
DECLARE r record; s record; nuevo jsonb; k text; h text; uso timestamptz;
BEGIN
 SELECT * INTO STRICT r FROM vec_contexto_actor_v1.registros_contexto_externo_v2 WHERE registro_contexto_ref='rca_ctx15_registro_0000000000001';
 SELECT * INTO STRICT s FROM vec_contexto_actor_v1.contexto_externo_versiones WHERE provision_ref=r.provision_ref AND version=r.provision_version;
 SELECT vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,r.manifiesto_procedencia_huella_sha256,'autoridad_maestra_acreditada',r.cuenta_ref,1,s.persona_ref,1,r.perfil_ref,1,s.contexto_ref,1,'certificado','alto',r.resuelto_en,clock_timestamp()+interval '1 minute') INTO uso;
 IF uso IS NULL THEN RAISE EXCEPTION 'despacho AD3 no acredita recibo propio'; END IF;
 IF (SELECT count(*) FROM vec_contexto_actor_v1.registros_contexto_externo_v2)<>2 THEN RAISE EXCEPTION 'replay duplicó recibo'; END IF;
 IF EXISTS(SELECT 1 FROM vec_contexto_actor_v1.registros_contexto WHERE cuenta_ref IN('cta_ctx15_sintetico_00000000001','cta_ctx15_sintetico_00000000002')) THEN RAISE EXCEPTION 'resolución escribió recibo compartido'; END IF;
 -- Inyección fuera de la API en un subbloque que se revierte: ni una
 -- coincidencia de recibo entre almacenes puede acreditar uso activo.
 BEGIN
  INSERT INTO vec_contexto_actor_v1.registros_contexto(operacion_ref,registro_contexto_ref,cuenta_ref,perfil_ref,metodo,garantia,solicitado_en,resuelto_en,representacion_canonica,huella_sha256,manifiesto_procedencia_canonico,manifiesto_procedencia_huella_sha256,autoridad_efectiva)
  VALUES(r.operacion_ref,r.registro_contexto_ref,r.cuenta_ref,r.perfil_ref,r.metodo,r.garantia,r.solicitado_en,r.resuelto_en,r.representacion_canonica,r.huella_sha256,r.manifiesto_procedencia_canonico,r.manifiesto_procedencia_huella_sha256,r.autoridad_efectiva);
  SELECT vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,r.manifiesto_procedencia_huella_sha256,'autoridad_maestra_acreditada',r.cuenta_ref,1,s.persona_ref,1,r.perfil_ref,1,s.contexto_ref,1,'certificado','alto',r.resuelto_en,clock_timestamp()+interval '1 minute') INTO uso;
  IF uso IS NOT NULL THEN RAISE EXCEPTION 'AD3 aceptó registro en ambas poblaciones'; END IF;
  RAISE EXCEPTION 'revertir inyección de prueba' USING ERRCODE='U0001';
 EXCEPTION WHEN SQLSTATE 'U0001' THEN NULL;
 END;
 nuevo:=s.snapshot||jsonb_build_object('estado','revocado');
 FOREACH k IN ARRAY ARRAY['cuenta','persona','perfil','contexto','vinculo_candidato'] LOOP
  nuevo:=jsonb_set(nuevo,ARRAY[k],(nuevo->k)||jsonb_build_object('estado','revocado','version',2));
 END LOOP;
 h:=vec_contexto_actor_v1.huella_snapshot_contexto_externo_v1(nuevo,2);
 PERFORM vec_contexto_actor_v1.publicar_snapshot_contexto_externo_v1(nuevo,1,s.huella_sha256,h);
 IF vec_contexto_actor_v1.acreditar_candidato_externo_v1(s.persona_ref,s.perfil_ref,'can_ctx15_sintetico_000000000001') THEN RAISE EXCEPTION 'AUT16 revocado positivo'; END IF;
 SELECT vec_contexto_actor_v1.acreditar_uso_registro_contexto_actor_v2(r.registro_contexto_ref,'vec.contexto-actor.vinculado.v2',r.huella_sha256,r.manifiesto_procedencia_huella_sha256,'autoridad_maestra_acreditada',r.cuenta_ref,1,s.persona_ref,1,r.perfil_ref,1,s.contexto_ref,1,'certificado','alto',r.resuelto_en,clock_timestamp()+interval '1 minute') INTO uso;
 IF uso IS NOT NULL THEN RAISE EXCEPTION 'AD3 revocado positivo'; END IF;
 IF (SELECT count(*) FROM vec_contexto_actor_v1.contexto_externo_versiones WHERE provision_ref=s.provision_ref)<>2 THEN RAISE EXCEPTION 'perdió historia externa'; END IF;
END $recibo_y_revocacion$;
RESET ROLE;
ROLLBACK;
\echo CTX15-PRUEBA-OK
