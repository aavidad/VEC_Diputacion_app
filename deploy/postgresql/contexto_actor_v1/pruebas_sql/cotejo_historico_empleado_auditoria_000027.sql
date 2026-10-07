\set ON_ERROR_STOP on
-- Prueba exclusiva del checkpoint sintético RPT27; nunca en principal.
\if :{?ca27_ensayo_autorizado}
\else
\echo 'PARO CA27: falta activación de ensayo sintético'
\quit 2
\endif
\if :ca27_ensayo_autorizado
\else
\echo 'PARO CA27: ensayo no autorizado'
\quit 2
\endif
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='15s';
CREATE TEMP TABLE ca27_test_originales AS
 SELECT * FROM vec_contexto_actor_v1.registros_contexto
 WHERE registro_contexto_ref LIKE 'rca_rpt27_%';
CREATE TEMP TABLE ca27_test_legacy AS
 SELECT * FROM vec_contexto_actor_v1.registros_contexto
 WHERE autoridad_efectiva='autoridad_maestra_acreditada'
   AND convert_from(representacion_canonica,'UTF8')::jsonb->>'esquema'='vec.contexto-actor.vinculado.v2'
   AND jsonb_array_length(convert_from(representacion_canonica,'UTF8')::jsonb->'vinculos')>0
   AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(
       convert_from(representacion_canonica,'UTF8')::jsonb->'vinculos') j
       WHERE j->>'vinculo_ref' NOT LIKE 'vin_%')
 ORDER BY resuelto_en DESC LIMIT 1;
CREATE TEMP TABLE ca27_test_empleado AS
 SELECT r.registro_contexto_ref,r.huella_sha256,r.manifiesto_procedencia_huella_sha256,
        r.representacion_canonica,r.resuelto_en,p.persona_ref,pe.*
 FROM ca27_test_originales r
 JOIN vec_contexto_actor_v1.perfil_versiones p ON p.perfil_ref=r.perfil_ref
     AND p.version=(convert_from(r.representacion_canonica,'UTF8')::jsonb->>'perfil_version')::numeric
 CROSS JOIN LATERAL vec_contexto_actor_v1.proyeccion_empleado_personal_v2(p.persona_ref,r.resuelto_en) pe
 ORDER BY r.registro_contexto_ref LIMIT 1;
DO $cantidad$
BEGIN
 IF (SELECT count(*) FROM ca27_test_originales)<>3
    OR (SELECT count(*) FROM ca27_test_empleado WHERE resultado='empleado')<>1
    OR (SELECT count(*) FROM ca27_test_legacy)<>1
    OR (SELECT count(*) FROM vec_personal.recibo_relacion_para_rpt)<>3
    OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='intento_nominal') THEN
  RAISE EXCEPTION 'CA27 necesita checkpoint sintético PARCIAL de tres positivos y cero intentos' USING ERRCODE='55000';
 END IF;
END $cantidad$;
GRANT SELECT ON ca27_test_originales,ca27_test_empleado,ca27_test_legacy TO vec_autorizacion_atestada_v3_propietario,vec_personal_propietario;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $originales$
DECLARE r record; legado record; bytes bytea; rechazado boolean;
BEGIN
 SELECT * INTO STRICT legado FROM pg_temp.ca27_test_legacy;
 IF vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
     legado.registro_contexto_ref,legado.huella_sha256,
     legado.manifiesto_procedencia_huella_sha256,legado.representacion_canonica) IS NOT TRUE THEN
  RAISE EXCEPTION 'CA27 perdió el cotejo legacy vin_';
 END IF;
 FOR r IN SELECT * FROM pg_temp.ca27_test_originales LOOP
  IF vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
     r.registro_contexto_ref,r.huella_sha256,r.manifiesto_procedencia_huella_sha256,r.representacion_canonica) IS NOT TRUE THEN
   RAISE EXCEPTION 'CA27 original PEP no acreditado';
  END IF;
  -- Un byte ajeno, aunque sea espacio JSON, no tiene el canon registrado.
  bytes:=r.representacion_canonica||convert_to(' ','UTF8'); rechazado:=false;
  BEGIN
   PERFORM vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
      r.registro_contexto_ref,r.huella_sha256,r.manifiesto_procedencia_huella_sha256,bytes);
  EXCEPTION WHEN SQLSTATE '22023' THEN rechazado:=true; END;
  IF NOT rechazado THEN RAISE EXCEPTION 'CA27 aceptó canon modificado'; END IF;
  rechazado:=false;
  BEGIN
   PERFORM vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
      r.registro_contexto_ref,r.huella_sha256,repeat('0',64),r.representacion_canonica);
  EXCEPTION WHEN SQLSTATE 'P0002' THEN rechazado:=true; END;
  IF NOT rechazado THEN RAISE EXCEPTION 'CA27 aceptó procedencia ajena'; END IF;
 END LOOP;
END $originales$;
-- Revisión posterior: Personal conserva la versión1 conocida en resuelto_en.
SET LOCAL ROLE vec_personal_propietario;
DO $posterior$
DECLARE e record;
BEGIN
 SELECT * INTO STRICT e FROM pg_temp.ca27_test_empleado;
 PERFORM vec_personal.publicar_proyeccion_empleado_persona_v1(
   e.proyeccion_ref,e.version::bigint+1,e.persona_ref,e.empleado_ref,'revocada',
   e.vigente_desde,e.vigente_hasta,'baja',e.procedencia_ref,
   e.procedencia_version::bigint+1,e.procedencia_huella_sha256);
END $posterior$;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
DO $historico$
DECLARE e record;
BEGIN
 SELECT * INTO STRICT e FROM pg_temp.ca27_test_empleado;
 IF vec_contexto_actor_v1.cotejar_contexto_historico_auditoria_v1(
      e.registro_contexto_ref,e.huella_sha256,e.manifiesto_procedencia_huella_sha256,e.representacion_canonica) IS NOT TRUE THEN
  RAISE EXCEPTION 'CA27 perdió el original después de la revisión posterior';
 END IF;
END $historico$;
RESET ROLE;
DO $preservacion$
BEGIN
 IF (SELECT count(*) FROM vec_personal.recibo_relacion_para_rpt)<>3
    OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.auditoria_consumo_v3 WHERE tipo_registro='intento_nominal')
    OR EXISTS(SELECT 1 FROM pg_temp.ca27_test_originales o FULL JOIN
        (SELECT * FROM vec_contexto_actor_v1.registros_contexto WHERE registro_contexto_ref LIKE 'rca_rpt27_%') r
        USING(registro_contexto_ref) WHERE to_jsonb(o) IS DISTINCT FROM to_jsonb(r)) THEN
  RAISE EXCEPTION 'CA27 ensayo alteró recibos, intentos o contextos originales';
 END IF;
END $preservacion$;
ROLLBACK;
\echo 'CA27 SQL focal: originales PEP y legacy vin_, canon/procedencia alterados, revisión posterior y conservación OK; todo ROLLBACK'
