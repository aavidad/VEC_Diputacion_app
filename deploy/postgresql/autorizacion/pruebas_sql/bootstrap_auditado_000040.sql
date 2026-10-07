\set ON_ERROR_STOP on
-- Requiere un bootstrap real ya confirmado en el clon. No fabrica perfiles.
-- Dirección configura vec.ensayo.plan_bootstrap / vec.ensayo.sha_aprobado y
-- conecta como el LOGIN técnico exclusivo aprobado. Todo termina en ROLLBACK.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL timezone='UTC';
SET LOCAL search_path=pg_catalog;
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $vectores$
DECLARE p text:=current_setting('vec.ensayo.plan_bootstrap');sha text:=current_setting('vec.ensayo.sha_aprobado');
 r jsonb;repetido jsonb;n jsonb;a text;b text;
BEGIN
 IF has_function_privilege(session_user,'vec_autorizacion.aplicar_efecto_bootstrap_central_admin_v3(text,text)','EXECUTE') THEN
  RAISE EXCEPTION 'AUT40 prueba: helper privado accesible al LOGIN'; END IF;
 r:=vec_autorizacion.registrar_bootstrap_central_admin_v3(p,sha);
 repetido:=vec_autorizacion.registrar_bootstrap_central_admin_v3(p,sha);
 IF r->>'estado'<>'permitido' OR r->>'replay'<>'true' OR r->'recibo' IS NULL
 OR repetido->>'estado'<>'permitido' OR r->'recibo' IS DISTINCT FROM repetido->'recibo'
 OR r#>>'{auditoria_intento,auditoria_ref}'=repetido#>>'{auditoria_intento,auditoria_ref}' THEN
  RAISE EXCEPTION 'AUT40 prueba: replay/recibo/intento no conservados'; END IF;
 a:=r#>>'{auditoria_intento,auditoria_ref}';b:=repetido#>>'{auditoria_intento,auditoria_ref}';
 IF a !~ '^aud_v3_bi_[0-9a-f]{32}$' OR b !~ '^aud_v3_bi_[0-9a-f]{32}$' THEN
  RAISE EXCEPTION 'AUT40 prueba: familia de intento cruzada'; END IF;
 n:=vec_autorizacion.registrar_bootstrap_central_admin_v3(p,repeat('0',64));
 IF n->>'estado'<>'denegado' OR n->>'codigo'<>'bootstrap_rechazado'
 OR n->'recibo' IS DISTINCT FROM 'null'::jsonb OR n->>'replay'<>'false'
 OR n#>>'{auditoria_intento,auditoria_ref}' IS NULL THEN
  RAISE EXCEPTION 'AUT40 prueba: rechazo sin intento o con efecto ficticio'; END IF;
END $vectores$;
ROLLBACK;
