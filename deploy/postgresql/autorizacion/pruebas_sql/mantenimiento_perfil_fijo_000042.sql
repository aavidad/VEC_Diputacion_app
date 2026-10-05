\set ON_ERROR_STOP on
-- Operador real configurado, documentos aprobados por Dirección fuera de Git.
-- GUC de sesión vec.ensayo.plan_mantenimiento contiene bytes exactos del plan.
-- Sólo clon; ROLLBACK final. No siembra configuración ni permisos favorables.
BEGIN ISOLATION LEVEL SERIALIZABLE;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='2s';
SET LOCAL statement_timeout='30s';
DO $pruebas$
DECLARE p text:=current_setting('vec.ensayo.plan_mantenimiento');sha text:=encode(pg_catalog.sha256(convert_to(p,'UTF8')),'hex');r jsonb;s jsonb;n jsonb;
BEGIN
 r:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,sha);
 IF r->>'estado' IS DISTINCT FROM 'permitido' OR r->>'replay' IS DISTINCT FROM 'false' OR r#>>'{recibo,rol_destino_ref}' IS DISTINCT FROM 'rol:administracion_perfiles:v5' OR jsonb_array_length(r#>'{recibo,asignaciones}')<>2 OR r#>>'{recibo,auditoria_ref}' IS NULL OR r#>>'{auditoria_intento,auditoria_ref}' IS NULL THEN RAISE EXCEPTION 'AUT42 prueba: mantenimiento positivo no demostrado';END IF;
 s:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,sha);
 IF s->>'estado' IS DISTINCT FROM 'permitido' OR s->>'replay' IS DISTINCT FROM 'true' OR s->'recibo' IS DISTINCT FROM r->'recibo' OR s#>>'{auditoria_intento,auditoria_ref}'=r#>>'{auditoria_intento,auditoria_ref}' THEN RAISE EXCEPTION 'AUT42 prueba: replay divergente o sin intento nuevo';END IF;
 n:=vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,repeat('0',64));
 IF n->>'estado' IS DISTINCT FROM 'denegado' OR n->'recibo' IS DISTINCT FROM 'null'::jsonb OR n->>'replay' IS DISTINCT FROM 'false' OR n#>>'{auditoria_intento,auditoria_ref}' IS NULL THEN RAISE EXCEPTION 'AUT42 prueba: aprobación ajena sin denegación auditada';END IF;
 IF vec_autorizacion.mantener_version_perfil_fijo_admin_v1(p,sha)->'recibo' IS DISTINCT FROM r->'recibo' THEN RAISE EXCEPTION 'AUT42 prueba: rechazo alteró recibo original';END IF;
END $pruebas$;
ROLLBACK;
