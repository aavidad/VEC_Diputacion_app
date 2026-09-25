\set ON_ERROR_STOP on
-- AD3-62: recuperación documental exacta; AD3-60 permanece intacta.
BEGIN;
SET LOCAL ROLE vec_autorizacion_atestada_v3_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_autorizacion_atestada_v3:migracion:000062',0));
DO $pre$
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario'
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR to_regprocedure('vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(bytea,bytea,numeric,numeric)') IS NULL
 THEN RAISE EXCEPTION 'AD3-62: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(
 p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,p_persona_version numeric,p_perfil_version numeric,
 p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS TABLE(decision_ref text,efecto_ref text,huella_efecto_sha256 text,consumo_huella_sha256 text,auditoria_ref text,consumida_en timestamptz,consumo_nuevo boolean)
LANGUAGE plpgsql VOLATILE SECURITY DEFINER SET search_path=pg_catalog SET lock_timeout='2s' AS $f$
DECLARE c jsonb; d jsonb; x record; k record; pk record; cfg record; raiz record;
 ahora timestamptz(6); viva timestamptz(6); accion text;
BEGIN
 IF current_user<>'vec_autorizacion_atestada_v3_propietario' OR session_user=current_user
 THEN RAISE EXCEPTION 'AD3-62: ejecutor inválido' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'AD3-62: material inválido' USING ERRCODE='22023'; END;
 accion:=d->>'accion';
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_documentos.operacion.v1'
    OR c->>'operacion' IS DISTINCT FROM accion
    OR d->>'modulo_id' IS DISTINCT FROM 'documentos'
    OR d->>'recurso_ref' IS DISTINCT FROM c->>'efecto_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM c->>'huella_efecto_sha256'
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
    OR NOT ((accion='documentos.generado.alta' AND d->>'tipo_recurso'='documento_generado'
      AND d->>'finalidad'='alta_documento_generado' AND d->'campos_permitidos'='["documento","recibo"]'::jsonb)
     OR (accion='documentos.expediente.listar' AND d->>'tipo_recurso'='expediente_documental'
      AND d->>'finalidad'='listar_documentos_expediente' AND d->'campos_permitidos'='["items","siguiente_cursor"]'::jsonb)
     OR (accion='documentos.original.descargar' AND d->>'tipo_recurso'='documento_original'
      AND d->>'finalidad'='descargar_documento_original' AND d->'campos_permitidos'='["contenido","documento"]'::jsonb)
     OR (accion='documentos.notificacion.preparar' AND d->>'tipo_recurso'='notificacion_preparada'
      AND d->>'finalidad'='preparar_notificacion' AND d->'campos_permitidos'='["preparacion","recibo"]'::jsonb))
 THEN RAISE EXCEPTION 'AD3-62: operación denegada' USING ERRCODE='42501'; END IF;
 -- El núcleo AD3 coteja las diez piezas byte a byte contra el consumo
 -- histórico y devuelve false solo para una repetición inequívoca.
 SELECT * INTO STRICT x FROM vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(
  'operacion_documentos_comunes',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,
  p_payload,p_sobre,p_evidencia,p_raiz);
 IF x.consumo_nuevo IS NULL OR x.decision_ref IS DISTINCT FROM c->>'decision_ref'
    OR x.efecto_ref IS DISTINCT FROM c->>'efecto_ref'
    OR x.huella_efecto_sha256 IS DISTINCT FROM c->>'huella_efecto_sha256'
 THEN RAISE EXCEPTION 'AD3-62: consumo incoherente' USING ERRCODE='42501'; END IF;
 IF x.consumo_nuevo IS FALSE THEN
  IF accion NOT IN ('documentos.generado.alta','documentos.notificacion.preparar')
  THEN RAISE EXCEPTION 'AD3-62: lectura requiere consumo nuevo' USING ERRCODE='42501'; END IF;
  -- El núcleo retorna replay antes de las comprobaciones vivas. Repetimos
  -- aquí la vigencia y bloqueamos el gobierno para que revocación concurrente
  -- no convierta un recibo antiguo en una autorización nueva.
  PERFORM 1 FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id FOR UPDATE;
  IF NOT FOUND THEN RAISE EXCEPTION 'AD3-62: gobierno no disponible' USING ERRCODE='55000'; END IF;
  ahora:=date_trunc('microseconds',clock_timestamp());
  SELECT * INTO k FROM vec_autorizacion_atestada_v3.clave_capacidad_version
   WHERE clave_id=c->>'clave_id' AND version=(c->>'clave_version')::numeric FOR SHARE;
  SELECT * INTO pk FROM vec_autorizacion_atestada_v3.puntero_clave_emision
   WHERE establecida_en<=ahora ORDER BY orden DESC LIMIT 1 FOR SHARE;
  SELECT cf.*, cp.configuracion_secuencia_minima,cp.raiz_version_minima INTO cfg
   FROM vec_autorizacion_atestada_v3.puntero_configuracion_actual p
   JOIN vec_autorizacion_atestada_v3.configuracion_confianza_version cf ON cf.revision=p.configuracion_revision
   CROSS JOIN vec_autorizacion_atestada_v3.checkpoint_gobierno cp
   WHERE p.establecida_en<=ahora ORDER BY p.orden DESC LIMIT 1 FOR SHARE OF p,cf;
  SELECT r.* INTO raiz FROM vec_autorizacion_atestada_v3.configuracion_raiz cr
   JOIN vec_autorizacion_atestada_v3.raiz_confianza_version r
     ON r.clave_id=cr.raiz_clave_id AND r.version=cr.raiz_version
   WHERE cr.configuracion_revision=cfg.revision AND r.clave_id=c->>'raiz_clave_id'
     AND r.version=(c->>'raiz_version')::numeric FOR SHARE OF r;
  IF k.clave_id IS NULL OR pk.clave_id IS NULL OR cfg.revision IS NULL OR raiz.clave_id IS NULL
     OR k.clave_id IS DISTINCT FROM pk.clave_id OR k.version IS DISTINCT FROM pk.version
     OR k.revision_gobierno IS DISTINCT FROM (c->>'revision_gobierno')::numeric
     OR k.huella_gobierno_sha256 IS DISTINCT FROM c->>'huella_gobierno_sha256'
     OR k.audiencia_consumo IS DISTINCT FROM 'vec_documentos.operacion.v1'
     OR k.emisor_id IS DISTINCT FROM c->>'emisor_id'
     OR ahora < k.valida_desde OR ahora >= k.valida_hasta
     OR cfg.revision IS DISTINCT FROM c->>'revision_confianza'
     OR cfg.secuencia IS DISTINCT FROM (c->>'configuracion_secuencia')::numeric
     OR cfg.secuencia < cfg.configuracion_secuencia_minima
     OR cfg.huella_configuracion_sha256 IS DISTINCT FROM c->>'huella_configuracion_sha256'
     OR cfg.expira_en <= ahora
     OR raiz.version < cfg.raiz_version_minima
     OR raiz.huella_spki_sha256 IS DISTINCT FROM c->>'huella_raiz_spki_sha256'
     OR raiz.clave_publica_spki IS DISTINCT FROM p_raiz
     OR raiz.suite IS DISTINCT FROM c->>'suite'
     OR raiz.audiencia_despliegue IS DISTINCT FROM c->>'audiencia_despliegue'
     OR ahora < raiz.valida_desde OR ahora >= raiz.valida_hasta
     OR ahora >= (c->>'expira_en')::timestamptz
     OR ahora >= (c->>'decision_valida_hasta')::timestamptz
     OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_clave_capacidad r
       WHERE r.clave_id=k.clave_id AND r.version=k.version AND r.revocada_en<=ahora)
     OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_configuracion r
       WHERE r.configuracion_revision=cfg.revision AND r.revocada_en<=ahora)
     OR EXISTS(SELECT 1 FROM vec_autorizacion_atestada_v3.revocacion_raiz r
       WHERE r.raiz_clave_id=raiz.clave_id AND r.raiz_version=raiz.version AND r.revocada_en<=ahora)
  THEN RAISE EXCEPTION 'AD3-62: replay sin vigencia' USING ERRCODE='42501'; END IF;
  viva:=vec_autorizacion.revalidar_decision_contexto_actor_v3_viva(
   p_decision,p_motivo,p_persona_version,p_perfil_version);
  IF viva IS NULL THEN RAISE EXCEPTION 'AD3-62: decisión retirada' USING ERRCODE='42501'; END IF;
 END IF;
 RETURN QUERY SELECT x.decision_ref,x.efecto_ref,x.huella_efecto_sha256,
  x.consumo_huella_sha256,x.auditoria_ref,x.consumida_en,x.consumo_nuevo;
END $f$;
REVOKE ALL ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_autorizacion_atestada_v3.consumir_operacion_documentos_replay_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_documentos_propietario;
COMMIT;
