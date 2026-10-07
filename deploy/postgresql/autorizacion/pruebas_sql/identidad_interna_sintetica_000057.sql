\set ON_ERROR_STOP on
-- Ejecutar sólo en un clon desechable conectado como el LOGIN técnico exclusivo
-- configurado por el DBA. El operador carga en privado vec.ensayo.plan_canonico;
-- el ensayo revierte la provisión y todos los intentos al finalizar.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='60s';
DO $ensayo$
DECLARE plan text:=pg_catalog.current_setting('vec.ensayo.plan_canonico');
 sha text;operacion text;primero jsonb;replay jsonb;recuperado jsonb;rechazo jsonb;ausente jsonb;
BEGIN
 IF current_user<>session_user OR plan IS NULL OR pg_catalog.octet_length(plan)>65536 THEN
  RAISE EXCEPTION 'AUT57 ensayo: LOGIN o plan privado ausente'; END IF;
 sha:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(plan,'UTF8')),'hex');
 operacion:=plan::jsonb->>'operacion_ref';
 primero:=vec_autorizacion.provisionar_identidad_interna_sintetica_v1(plan,sha);
 IF primero->>'estado' IS DISTINCT FROM 'permitido' OR primero->>'replay' IS DISTINCT FROM 'false'
 OR primero#>>'{recibo,operacion_ref}' IS DISTINCT FROM operacion
 OR primero#>>'{recibo,plan_sha256}' IS DISTINCT FROM sha
 OR primero#>>'{recibo,is,datos,cuenta_ordinaria_ref}' IS DISTINCT FROM primero#>>'{recibo,ca,datos,cuenta_ordinaria_ref}'
 OR primero#>>'{recibo,ca,datos,persona_ref}' IS DISTINCT FROM plan::jsonb#>>'{persona,persona_ref}'
 OR primero::text ~ '(sujeto_id_hmac|cuenta_privilegiada|empleado_ref|perfil_ref|material_hmac)'
 THEN RAISE EXCEPTION 'AUT57 ensayo: efecto/recibo incorrecto'; END IF;
 replay:=vec_autorizacion.provisionar_identidad_interna_sintetica_v1(plan,sha);
 IF replay->>'estado' IS DISTINCT FROM 'permitido' OR replay->>'replay' IS DISTINCT FROM 'true'
 OR replay->'recibo' IS DISTINCT FROM primero->'recibo'
 OR replay#>>'{auditoria_intento,auditoria_ref}' IS NOT DISTINCT FROM primero#>>'{auditoria_intento,auditoria_ref}'
 THEN RAISE EXCEPTION 'AUT57 ensayo: replay duplicó efecto o intento'; END IF;
 rechazo:=vec_autorizacion.provisionar_identidad_interna_sintetica_v1(plan,pg_catalog.repeat('0',64));
 IF rechazo->>'estado' IS DISTINCT FROM 'denegado' OR rechazo->'recibo' IS DISTINCT FROM 'null'::jsonb
 OR rechazo#>>'{auditoria_intento,auditoria_ref}' IS NULL
 OR rechazo#>>'{auditoria_intento,auditoria_ref}' IN(primero#>>'{auditoria_intento,auditoria_ref}',replay#>>'{auditoria_intento,auditoria_ref}')
 THEN RAISE EXCEPTION 'AUT57 ensayo: rechazo no auditado o con recibo'; END IF;
 recuperado:=vec_autorizacion.recuperar_identidad_interna_sintetica_v1(operacion,sha);
 IF recuperado->>'estado' IS DISTINCT FROM 'permitido' OR recuperado->>'replay' IS DISTINCT FROM 'true'
 OR recuperado->'recibo' IS DISTINCT FROM primero->'recibo'
 OR recuperado#>>'{auditoria_intento,auditoria_ref}' IN(primero#>>'{auditoria_intento,auditoria_ref}',replay#>>'{auditoria_intento,auditoria_ref}',rechazo#>>'{auditoria_intento,auditoria_ref}')
 THEN RAISE EXCEPTION 'AUT57 ensayo: recuperación divergente o sin auditoría nueva'; END IF;
 ausente:=vec_autorizacion.recuperar_identidad_interna_sintetica_v1('piis_'||pg_catalog.repeat('z',30),sha);
 IF ausente->>'estado' IS DISTINCT FROM 'denegado' OR ausente->'recibo' IS DISTINCT FROM 'null'::jsonb
 OR ausente#>>'{auditoria_intento,auditoria_ref}' IS NULL
 THEN RAISE EXCEPTION 'AUT57 ensayo: lectura ajena disponible'; END IF;
END $ensayo$;
ROLLBACK;
