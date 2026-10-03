\set ON_ERROR_STOP on
-- PREPARADA, NO EJECUTADA. Sólo en clon sintético autorizado después de cerrar
-- y ensayar AD180. No instala migraciones ni publica roles/perfiles/material.
-- No acredita que hoy exista consumidor ejecutable. No ejecutar en principal.
BEGIN;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
DO $prueba$
DECLARE
 f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_historia_servicios_propios_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 propietario oid:='vec_autorizacion_atestada_v3_propietario'::regrole;
 consumidor oid:='vec_personal_propietario'::regrole;
 x record; fuente text; d text;
BEGIN
 IF f IS NULL THEN RAISE EXCEPTION 'ad180.prueba: falta candidata cerrada/ensayada' USING ERRCODE='55000'; END IF;
 SELECT p.prosrc INTO STRICT fuente FROM pg_proc p WHERE p.oid=f;
 IF NOT EXISTS (SELECT 1 FROM pg_proc p WHERE p.oid=f AND p.proowner=propietario
   AND p.prokind='f' AND p.provolatile='v' AND p.proparallel='u' AND p.prosecdef
   AND p.proconfig=ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']
   AND p.pronargs=10
   AND p.proargnames=ARRAY['p_capacidad','p_decision','p_motivo','p_contexto',
      'p_persona_version','p_perfil_version','p_payload','p_sobre','p_evidencia','p_raiz',
      'decision_ref','efecto_ref','huella_efecto_sha256','consumo_huella_sha256',
      'auditoria_ref','consumida_en','consumo_nuevo']
   AND p.proargmodes=ARRAY['i','i','i','i','i','i','i','i','i','i','t','t','t','t','t','t','t']::"char"[])
 THEN RAISE EXCEPTION 'ad180.prueba: ABI/entorno divergente' USING ERRCODE='55000'; END IF;
 IF (SELECT count(*) FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f)<>2
    OR NOT has_function_privilege(consumidor,f,'EXECUTE')
    OR has_function_privilege('vec_personal_ejecutor',f,'EXECUTE')
    OR has_function_privilege('vec_personal_migrador',f,'EXECUTE') THEN
  RAISE EXCEPTION 'ad180.prueba: consumidor abierto/ausente' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.* FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF x.grantee NOT IN (propietario,consumidor) OR x.grantor<>propietario
     OR x.privilege_type<>'EXECUTE' OR x.is_grantable THEN
   RAISE EXCEPTION 'ad180.prueba: ACL no exclusiva' USING ERRCODE='55000'; END IF;
 END LOOP;
 IF strpos(fuente,'personal.registro_empleado.servicios.historia_propia.consultar')=0
    OR strpos(fuente,'vec_personal.registro_empleado.servicios.historia_propia.v1')=0
    OR strpos(fuente,'consultar_historia_servicios_propios')=0
    OR strpos(fuente,'["cobertura","corte","evidencia","revisiones"]')=0
    OR strpos(fuente,'''aud_v3_''||substr(x.consumo_huella_sha256,1,32)')=0
    OR strpos(fuente,'consumo_nuevo IS NOT TRUE')=0 THEN
  RAISE EXCEPTION 'ad180.prueba: contrato nominal/recibo divergente' USING ERRCODE='55000'; END IF;
 SELECT pg_get_constraintdef(c.oid,true) INTO STRICT d FROM pg_constraint c
 WHERE c.conrelid='vec_autorizacion_atestada_v3.clave_capacidad_version'::regclass
   AND c.conname='clave_capacidad_version_audiencia_consumo_check' AND c.contype='c' AND c.convalidated;
 IF strpos(d,'vec_personal.registro_empleado.servicios.historia_propia.v1')=0 THEN
  RAISE EXCEPTION 'ad180.prueba: audiencia no admitida' USING ERRCODE='55000'; END IF;
END $prueba$;
ROLLBACK;
