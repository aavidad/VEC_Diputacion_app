\set ON_ERROR_STOP on
-- AD238. Acciones nominales del plan Mayan de Documentos D16.
-- Preimagen exacta medida en clon PG18 desechable con AD236 y D15.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_ad3:migracion:000238',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
  OR to_regrole('vec_documentos_propietario') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
 THEN RAISE EXCEPTION 'AD238: preimagen ausente' USING ERRCODE='55000'; END IF;
END $pre$;

DO $core$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; ancla text; extension text; meta jsonb; propietario oid; acl aclitem[];
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proowner,p.proacl INTO original,meta,propietario,acl FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM '124674b93f72c8fb32f3476b9075cb2690ca365cdeb87581f0b7ccec796945e4'
   OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD238: núcleo V3 divergente' USING ERRCODE='55000'; END IF;
 ancla:=$a$       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["documento","recibo"]'::jsonb)))$a$;
 extension:=$e$
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM 'vec_documentos.operacion.v1'
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_evidencia_meritos'
 AND nullif(d->>'version_rol_ref','') IS NOT NULL
 AND d->>'recurso_ref' ~ '^(ref:[0-9a-f]{64}|[a-z][a-z0-9_]{1,31}:[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})$'
 AND d->>'recurso_ref' IS DISTINCT FROM ('ref:'||repeat('0',64))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion' IN ('documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva','documentos.ingesta_mayan.abandonar_recurso')
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'reserva_recurso_documental'
       AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["reserva","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.preparar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["plan","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.despachar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["intento","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.resolver'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["evidencia_custodia","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.conciliar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["evidencia_custodia","estado"]'::jsonb)
   OR (d->>'accion' IS NOT DISTINCT FROM 'documentos.ingesta_mayan.consultar'
       AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'plan_ingesta_mayan'
       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["plan","estado"]'::jsonb)))$e$;
 IF length(original)-length(replace(original,ancla,''))<>length(ancla) THEN
  RAISE EXCEPTION 'AD238: ancla núcleo no única' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,ancla,ancla||extension);
 EXECUTE nuevo;
 IF pg_get_functiondef(f) IS DISTINCT FROM nuevo
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
  OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 THEN RAISE EXCEPTION 'AD238: postimagen núcleo divergente' USING ERRCODE='55000'; END IF;
END $core$;

DO $wrapper$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; ancla text; ampliada text; replay text; replay_nuevo text;
 meta jsonb; propietario oid; acl aclitem[];
BEGIN
 SELECT pg_get_functiondef(f),to_jsonb(p)-'prosrc',p.proowner,p.proacl INTO original,meta,propietario,acl FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM 'ad37b692d166b26803e6c9d4bf7952bbd65c27b1aa0942f930309c78a2a99ff2'
  OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD238: fachada Documentos divergente' USING ERRCODE='55000'; END IF;
 ancla:=$a$     OR (accion='documentos.original_firmable.confirmar' AND d->>'tipo_recurso'='documento_original_firmable'
      AND d->>'finalidad'='custodiar_original_firmable' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb))$a$;
 ampliada:=$b$     OR (accion='documentos.original_firmable.confirmar' AND d->>'tipo_recurso'='documento_original_firmable'
      AND d->>'finalidad'='custodiar_original_firmable' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (accion IN ('documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.consultar_reserva','documentos.ingesta_mayan.abandonar_recurso')
      AND d->>'tipo_recurso'='reserva_recurso_documental' AND d->>'finalidad'='custodiar_evidencia_meritos'
      AND d->'campos_permitidos'='["reserva","estado"]'::jsonb)
     OR (accion='documentos.ingesta_mayan.preparar' AND d->>'tipo_recurso'='plan_ingesta_mayan'
      AND d->>'finalidad'='custodiar_evidencia_meritos' AND d->'campos_permitidos'='["plan","estado"]'::jsonb)
     OR (accion='documentos.ingesta_mayan.despachar' AND d->>'tipo_recurso'='plan_ingesta_mayan'
      AND d->>'finalidad'='custodiar_evidencia_meritos' AND d->'campos_permitidos'='["intento","estado"]'::jsonb)
     OR (accion IN ('documentos.ingesta_mayan.resolver','documentos.ingesta_mayan.conciliar')
      AND d->>'tipo_recurso'='plan_ingesta_mayan' AND d->>'finalidad'='custodiar_evidencia_meritos'
      AND d->'campos_permitidos'='["evidencia_custodia","estado"]'::jsonb)
     OR (accion='documentos.ingesta_mayan.consultar' AND d->>'tipo_recurso'='plan_ingesta_mayan'
      AND d->>'finalidad'='custodiar_evidencia_meritos' AND d->'campos_permitidos'='["plan","estado"]'::jsonb))$b$;
 replay:=$r$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar','documentos.original_firmable.reservar','documentos.original_firmable.confirmar')$r$;
 replay_nuevo:=$rn$  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar','documentos.externo.registrar','documentos.firmado.custodiar','documentos.original_firmable.reservar','documentos.original_firmable.confirmar','documentos.ingesta_mayan.reservar_recurso','documentos.ingesta_mayan.preparar','documentos.ingesta_mayan.despachar')$rn$;
 IF length(original)-length(replace(original,ancla,''))<>length(ancla)
  OR length(original)-length(replace(original,replay,''))<>length(replay)
 THEN RAISE EXCEPTION 'AD238: ancla fachada no única' USING ERRCODE='55000'; END IF;
 nuevo:=replace(replace(original,ancla,ampliada),replay,replay_nuevo);
 EXECUTE nuevo;
 IF pg_get_functiondef(f) IS DISTINCT FROM nuevo
  OR (SELECT to_jsonb(p)-'prosrc' FROM pg_proc p WHERE p.oid=f) IS DISTINCT FROM meta
  OR (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM propietario
  OR (SELECT proacl FROM pg_proc WHERE oid=f) IS DISTINCT FROM acl
 THEN RAISE EXCEPTION 'AD238: postimagen fachada divergente' USING ERRCODE='55000'; END IF;
END $wrapper$;
COMMIT;
