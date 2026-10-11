\set ON_ERROR_STOP on
-- AD240: consumidor nominal de la carga directa documental D17.
-- Depende de AD238; AD239 continúa reservado a Méritos y no se simula.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_ad3:migracion:000240',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regrole('vec_documentos_propietario') IS NULL
 THEN RAISE EXCEPTION 'AD240: preimagen ausente' USING ERRCODE='55000'; END IF;
END $pre$;
DO $core$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; ancla text; extension text; propietario oid;
BEGIN
 SELECT pg_get_functiondef(f),p.proowner INTO original,propietario FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM
   '73432dc23428519f159864b4a4c572e37854573a807e55843175211e485e17f0'
  OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD240: núcleo divergente' USING ERRCODE='55000'; END IF;
 ancla:=$a$       AND d->'campos_permitidos' IS NOT DISTINCT FROM '["plan","estado"]'::jsonb)))$a$;
 extension:=$e$
           OR (
 p_perfil_mutacion IS NOT DISTINCT FROM 'operacion_documentos_comunes'
 AND c->>'audiencia_consumo' IS NOT DISTINCT FROM CASE d->>'accion'
  WHEN 'documentos.carga.preparar_reserva' THEN 'vec_documentos.carga.preparar_reserva.v1'
  WHEN 'documentos.carga.preparar_confirmacion' THEN 'vec_documentos.carga.preparar_confirmacion.v1'
  WHEN 'documentos.carga.preparar_respuesta' THEN 'vec_documentos.carga.preparar_respuesta.v1'
  WHEN 'documentos.carga.consultar_propia' THEN 'vec_documentos.carga.consultar_propia.v1'
  WHEN 'documentos.carga.conciliar' THEN 'vec_documentos.carga.conciliar.v1'
  WHEN 'documentos.carga.confirmar_intencion' THEN 'vec_documentos.carga.confirmar_intencion.v1'
  WHEN 'documentos.carga.confirmar_resultado' THEN 'vec_documentos.carga.confirmar_resultado.v1'
  WHEN 'vec.documentos.carga.analizar' THEN 'vec_documentos.carga.analizar.v1'
  WHEN 'documentos.carga.promover_intencion' THEN 'vec_documentos.carga.promover_intencion.v1'
  WHEN 'documentos.carga.promover_resultado' THEN 'vec_documentos.carga.promover_resultado.v1' END
 AND c->>'operacion' IS NOT DISTINCT FROM d->>'accion'
 AND d->>'modulo_id' IS NOT DISTINCT FROM 'documentos'
 AND d->>'finalidad' IS NOT DISTINCT FROM 'custodiar_evidencia_meritos'
 AND d->>'tipo_recurso' IS NOT DISTINCT FROM 'carga_documental'
 AND d->>'recurso_ref' ~ '^ref:[0-9a-f]{64}$'
 AND d->>'recurso_ref' IS DISTINCT FROM ('ref:'||repeat('0',64))
 AND d->>'recurso_ref' IS NOT DISTINCT FROM c->>'efecto_ref'
 AND d->>'contexto_recurso_huella_sha256' IS NOT DISTINCT FROM c->>'huella_efecto_sha256'
 AND d#>>'{vinculo_autenticacion_actor,superficie}' IS NOT DISTINCT FROM 'interna_corporativa'
 AND d->'obligaciones' IS NOT DISTINCT FROM '[]'::jsonb
 AND ((d->>'accion'='documentos.carga.preparar_reserva'
        AND d->'campos_permitidos'='["reserva","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.preparar_confirmacion'
        AND d->'campos_permitidos'='["manifiesto","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.preparar_respuesta'
        AND d->'campos_permitidos'='["recibo","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.consultar_propia'
        AND d->'campos_permitidos'='["carga","manifiesto"]'::jsonb)
   OR (d->>'accion'='documentos.carga.conciliar'
        AND d->'campos_permitidos'='["evidencia","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.confirmar_intencion'
        AND d->'campos_permitidos'='["intencion","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.confirmar_resultado'
        AND d->'campos_permitidos'='["contenido_cuarentena","estado"]'::jsonb)
   OR (d->>'accion'='vec.documentos.carga.analizar'
        AND d->'campos_permitidos'='["analisis_seguridad","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.promover_intencion'
        AND d->'campos_permitidos'='["intencion","estado"]'::jsonb)
   OR (d->>'accion'='documentos.carga.promover_resultado'
        AND d->'campos_permitidos'='["contenido_admitido","estado"]'::jsonb)))$e$;
 IF length(original)-length(replace(original,ancla,''))<>length(ancla)
 THEN RAISE EXCEPTION 'AD240: ancla núcleo no única' USING ERRCODE='55000'; END IF;
 nuevo:=replace(original,ancla,ancla||extension);
 EXECUTE nuevo;
END $core$;
DO $wrapper$
DECLARE f oid:=to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)');
 original text; nuevo text; a_audiencia text; a_acciones text; propietario oid;
BEGIN
 SELECT pg_get_functiondef(f),p.proowner INTO original,propietario FROM pg_proc p WHERE p.oid=f;
 IF encode(sha256(convert_to(original,'UTF8')),'hex') IS DISTINCT FROM
   'f43b5c42452d250b44c092719d705555a5341805e867a24bb6c243796a794600'
  OR propietario<>'vec_autorizacion_atestada_v3_propietario'::regrole
 THEN RAISE EXCEPTION 'AD240: wrapper divergente' USING ERRCODE='55000'; END IF;
 a_audiencia:=$a$c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.operacion.v1'$a$;
 nuevo:=replace(original,a_audiencia,$b$NOT ((c->>'audiencia_consumo'='vec_documentos.operacion.v1'
  AND accion NOT IN ('documentos.carga.preparar_reserva','documentos.carga.preparar_confirmacion',
   'documentos.carga.preparar_respuesta','documentos.carga.consultar_propia','documentos.carga.conciliar',
   'documentos.carga.confirmar_intencion','documentos.carga.confirmar_resultado','vec.documentos.carga.analizar','documentos.carga.promover_intencion','documentos.carga.promover_resultado'))
  OR c->>'audiencia_consumo'=CASE accion
   WHEN 'documentos.carga.preparar_reserva' THEN 'vec_documentos.carga.preparar_reserva.v1'
   WHEN 'documentos.carga.preparar_confirmacion' THEN 'vec_documentos.carga.preparar_confirmacion.v1'
   WHEN 'documentos.carga.preparar_respuesta' THEN 'vec_documentos.carga.preparar_respuesta.v1'
   WHEN 'documentos.carga.consultar_propia' THEN 'vec_documentos.carga.consultar_propia.v1'
   WHEN 'documentos.carga.conciliar' THEN 'vec_documentos.carga.conciliar.v1'
   WHEN 'documentos.carga.confirmar_intencion' THEN 'vec_documentos.carga.confirmar_intencion.v1'
  WHEN 'documentos.carga.confirmar_resultado' THEN 'vec_documentos.carga.confirmar_resultado.v1'
   WHEN 'vec.documentos.carga.analizar' THEN 'vec_documentos.carga.analizar.v1'
   WHEN 'documentos.carga.promover_intencion' THEN 'vec_documentos.carga.promover_intencion.v1'
  WHEN 'documentos.carga.promover_resultado' THEN 'vec_documentos.carga.promover_resultado.v1' END)$b$);
 a_acciones:=$a$     OR (accion='documentos.ingesta_mayan.consultar' AND d->>'tipo_recurso'='plan_ingesta_mayan'
      AND d->>'finalidad'='custodiar_evidencia_meritos' AND d->'campos_permitidos'='["plan","estado"]'::jsonb))$a$;
 nuevo:=replace(nuevo,a_acciones,left(a_acciones,length(a_acciones)-1)||$b$
     OR (accion IN ('documentos.carga.preparar_reserva','documentos.carga.preparar_confirmacion',
       'documentos.carga.preparar_respuesta','documentos.carga.consultar_propia','documentos.carga.conciliar',
       'documentos.carga.confirmar_intencion','documentos.carga.confirmar_resultado','vec.documentos.carga.analizar','documentos.carga.promover_intencion','documentos.carga.promover_resultado')
       AND d->>'tipo_recurso'='carga_documental' AND d->>'finalidad'='custodiar_evidencia_meritos'
       AND d->'campos_permitidos'=CASE accion
        WHEN 'documentos.carga.preparar_reserva' THEN '["reserva","estado"]'::jsonb
        WHEN 'documentos.carga.preparar_confirmacion' THEN '["manifiesto","estado"]'::jsonb
        WHEN 'documentos.carga.preparar_respuesta' THEN '["recibo","estado"]'::jsonb
        WHEN 'documentos.carga.consultar_propia' THEN '["carga","manifiesto"]'::jsonb
        WHEN 'documentos.carga.conciliar' THEN '["evidencia","estado"]'::jsonb
        WHEN 'documentos.carga.confirmar_intencion' THEN '["intencion","estado"]'::jsonb
        WHEN 'documentos.carga.confirmar_resultado' THEN '["contenido_cuarentena","estado"]'::jsonb
        WHEN 'vec.documentos.carga.analizar' THEN '["analisis_seguridad","estado"]'::jsonb
        WHEN 'documentos.carga.promover_intencion' THEN '["intencion","estado"]'::jsonb
        WHEN 'documentos.carga.promover_resultado' THEN '["contenido_admitido","estado"]'::jsonb END))$b$);
 IF length(original)-length(replace(original,a_audiencia,''))<>length(a_audiencia)
  OR length(original)-length(replace(original,a_acciones,''))<>length(a_acciones)
 THEN RAISE EXCEPTION 'AD240: ancla wrapper no única' USING ERRCODE='55000'; END IF;
 EXECUTE nuevo;
END $wrapper$;
COMMIT;
