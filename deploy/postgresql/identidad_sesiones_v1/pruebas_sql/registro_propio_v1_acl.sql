\set ON_ERROR_STOP on
SET search_path=pg_catalog;
DO $test$
DECLARE f regprocedure:='vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
 r record;
BEGIN
 IF current_setting('server_version_num')::int NOT BETWEEN 180000 AND 189999
 OR current_database() NOT LIKE 'vec_f2_%'
 OR to_regclass('vec_identidad_sesiones_v1.registro_propio_v1') IS NULL
 THEN RAISE EXCEPTION 'F2 registro: ensayo fuera de PG18 aislado' USING ERRCODE='55000'; END IF;
 SELECT p.proowner,p.prosecdef,p.provolatile,p.proconfig INTO STRICT r FROM pg_proc p WHERE p.oid=f;
 IF r.proowner IS DISTINCT FROM 'vec_autorizacion_atestada_v3_propietario'::regrole
 OR NOT r.prosecdef OR r.provolatile<>'v' OR r.proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog','lock_timeout=2s']
 OR EXISTS(SELECT 1 FROM pg_proc p,LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee=0)
 OR NOT has_function_privilege('vec_identidad_sesiones_v1_propietario',f,'EXECUTE')
 OR has_function_privilege('vec_contacto_usuario_owner',f,'EXECUTE')
 THEN RAISE EXCEPTION 'F2 registro: fachada V3/ACL divergente' USING ERRCODE='55000'; END IF;
 IF EXISTS(SELECT 1 FROM pg_class c WHERE c.oid IN (
 'vec_identidad_sesiones_v1.registro_propio_v1'::regclass,
 'vec_identidad_sesiones_v1.sujeto_persona_registro_propio_v1'::regclass,
 'vec_identidad_sesiones_v1.registro_propio_outbox_v1'::regclass)
 AND (NOT c.relrowsecurity OR NOT c.relforcerowsecurity))
 OR EXISTS(SELECT 1 FROM pg_class c,LATERAL aclexplode(coalesce(c.relacl,acldefault('r',c.relowner))) a
  WHERE c.oid='vec_identidad_sesiones_v1.registro_propio_v1'::regclass AND a.grantee=0)
 OR has_table_privilege('vec_identidad_sesiones_v1_provisionador','vec_identidad_sesiones_v1.registro_propio_v1','INSERT')
 THEN RAISE EXCEPTION 'F2 registro: RLS/ACL tabla divergente' USING ERRCODE='55000'; END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_identidad_sesiones_v1.registro_propio_v1'::regclass
 AND attname='auditoria_v3_ref' AND attnotnull)
 OR NOT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='vec_identidad_sesiones_v1.registro_propio_v1'::regclass
 AND attname='consumo_huella_sha256' AND attnotnull)
 THEN RAISE EXCEPTION 'F2 registro: recibo sin auditoría central' USING ERRCODE='55000'; END IF;
END $test$;

-- Alias rotado sin equivalencia de cuenta: la persona ya vinculada impide
-- aprovisionar otra cuenta. La historia sintética se revierte íntegra.
BEGIN;
SET LOCAL ROLE vec_contexto_actor_v1_propietario;
INSERT INTO vec_contexto_actor_v1.procedencias
 (procedencia_ref,procedencia_version,procedencia_huella_sha256,procedencia_autoridad)
 VALUES('prc_registropropioensayo000001',1,repeat('a',64),'autoridad_maestra_acreditada');
INSERT INTO vec_contexto_actor_v1.vinculo_contexto_versiones
 (vinculo_ref,version,cuenta_ref,perfil_ref,persona_ref,procedencia_ref,
  procedencia_version,procedencia_huella_sha256,procedencia_autoridad,estado,vigente_desde,vigente_hasta)
 VALUES('vca_registropropioensayo000001',1,'cta_registropropioensayo000001',
  'prf_registropropioensayo000001','per_registropropioensayo000001',
  'prc_registropropioensayo000001',1,repeat('a',64),'autoridad_maestra_acreditada',
  'activo','2026-01-01 UTC','2027-01-01 UTC');
RESET ROLE;
DO $test$
DECLARE cuentas bigint; definicion text;
BEGIN
 SELECT count(*) INTO cuentas FROM vec_identidad_sesiones_v1.cuenta;
 IF vec_contexto_actor_v1.persona_con_cuenta_registro_propio_v1('per_registropropioensayo000001') IS NOT TRUE
 OR vec_contexto_actor_v1.persona_con_cuenta_registro_propio_v1('per_registropropioensayo000002') IS NOT FALSE
 THEN RAISE EXCEPTION 'F2 registro: alias sin equivalencia crearía cuenta duplicada' USING ERRCODE='55000'; END IF;
 SELECT pg_get_functiondef('vec_identidad_sesiones_v1.registrar_propio_v1(bytea,bytea,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) INTO definicion;
 IF strpos(definicion,'persona_con_cuenta_registro_propio_v1')=0
 OR strpos(definicion,'persona_con_cuenta_registro_propio_v1')>strpos(definicion,'provisionar_cuenta_v1(')
 OR (SELECT count(*) FROM vec_identidad_sesiones_v1.cuenta)<>cuentas
 THEN RAISE EXCEPTION 'F2 registro: denegación fuera de la frontera del efecto' USING ERRCODE='55000'; END IF;
END $test$;
ROLLBACK;
-- Ni un superusuario de ensayo puede fabricar el consumo: la sesión nominal
-- exige un login técnico único y material V3 válido.
DO $test$ BEGIN
 BEGIN
  PERFORM vec_autorizacion_atestada_v3.registrar_y_consumir_registro_propio_v3_atestada(NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL,NULL);
  RAISE EXCEPTION 'F2 registro: consumo nulo aceptado' USING ERRCODE='55000';
 EXCEPTION WHEN insufficient_privilege THEN NULL; END;
END $test$;
