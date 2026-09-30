\set ON_ERROR_STOP on
-- AD3-113 (5.06): el consumidor documental con recuperación (AD3-62) admite
-- la custodia del documento firmado de un expediente:
-- documentos.firmado.custodiar, tipo documento_firmado, finalidad
-- custodiar_documento_firmado y campos exactos. Es un efecto: como el alta,
-- admite la recuperación exacta de un consumo ya hecho. Solo cambia el cuerpo
-- de esa función; propietario, ACL, configuración y dependencias se conservan.
-- Requiere AD3-62 exactamente como está instalada (huella de su definición).
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000113',0));
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:nucleo',0));
DO $m$
DECLARE
 f regprocedure;
 original text; nuevo text; actual text; meta jsonb; deps jsonb; acl aclitem[]; propietario oid; config text[]; definidora boolean;
 lista text:=$x$      AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb))$x$;
 lista_nueva text:=$x$      AND d->>'finalidad'='registrar_documento_externo' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (accion='documentos.firmado.custodiar' AND d->>'tipo_recurso'='documento_firmado'
      AND d->>'finalidad'='custodiar_documento_firmado' AND d->'campos_permitidos'='["documento_firmado.custodia","evidencia_custodia"]'::jsonb))$x$;
 replay text:=$x$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar')$x$;
 replay_nuevo text:=$x$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar')$x$;
BEGIN
 f:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR f IS NULL THEN
  RAISE EXCEPTION 'AD3-113: AD3-62 requerida' USING ERRCODE='55000';
 END IF;
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proacl,p.proowner,p.proconfig,p.prosecdef
   INTO STRICT original,meta,acl,propietario,config,definidora FROM pg_proc p WHERE p.oid=f;
 SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
   INTO deps FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f;
 IF md5(original)<>'2f9f7fe8b8508a6b80f95a56942cade5'
    OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole OR NOT definidora
    OR length(original)-length(replace(original,lista,''))<>length(lista)
    OR length(original)-length(replace(original,replay,''))<>length(replay)
    OR strpos(original,'documentos.firmado.custodiar')<>0
 THEN RAISE EXCEPTION 'AD3-113: consumidor documental distinto del esperado' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(original,lista,lista_nueva),replay,replay_nuevo);
 EXECUTE nuevo;
 SELECT pg_get_functiondef(f) INTO STRICT actual;
 IF actual IS DISTINCT FROM nuevo
    OR replace(replace(actual,lista_nueva,lista),replay_nuevo,replay) IS DISTINCT FROM original
    OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
    OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
    OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
    OR (SELECT proconfig FROM pg_proc WHERE oid=f) IS DISTINCT FROM config
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS DISTINCT FROM definidora
    OR (SELECT coalesce(jsonb_agg(to_jsonb(d) ORDER BY d.classid,d.objid,d.objsubid,d.refclassid,d.refobjid,d.refobjsubid,d.deptype),'[]'::jsonb)
         FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid=f) IS DISTINCT FROM deps
 THEN RAISE EXCEPTION 'AD3-113: consumidor alterado fuera del contrato' USING ERRCODE='55000'; END IF;
END $m$;
COMMIT;
