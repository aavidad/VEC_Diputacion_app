\set ON_ERROR_STOP on
-- RUM6: sólo lectura RRHH de hechos de la persona objetivo entregada por
-- el preparador autorizado de Bolsa. Sin decisión de acreditación ni cambios.
BEGIN;
SET LOCAL search_path=pg_catalog,pg_temp; SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_meritos:migracion:000006',0));
DO $pre$ BEGIN
 IF current_user<>'postgres' AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND rolsuper)
 OR to_regclass('vec_meritos.hecho_identidad') IS NULL
 OR to_regclass('vec_meritos.hecho_version') IS NULL
 OR to_regprocedure('vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(text,text,text,text,bytea,bytea,jsonb)') IS NULL
 OR to_regprocedure('vec_meritos.consultar_revision_rrhh_v1(bytea,bytea,jsonb,jsonb)') IS NOT NULL
 THEN RAISE EXCEPTION 'RUM6: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;
SET LOCAL ROLE vec_meritos_propietario;
CREATE FUNCTION vec_meritos.consultar_revision_rrhh_v1(
 p_contexto bytea,p_vinculo bytea,p_orden jsonb,p_captura_bolsa jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC'
SET lock_timeout='2s' SET statement_timeout='15s' AS $f$
DECLARE persona text; cursor text; limite integer; total integer; contador integer:=0;
 siguiente text:=''; ultimo text:=''; hechos jsonb:='[]'::jsonb; x record; acuse record;
 orden_auditoria jsonb; recibo jsonb; actor jsonb;
 claves constant text[]:=ARRAY['intento_ref','registro_contexto_ref','contexto_sha256',
  'procedencia_sha256','autenticacion_ref','sesion_ref','autenticacion_sha256',
  'accion','modulo_id','recurso_ref','finalidad_ref','proceso','canal',
  'correlacion_ref','lectura_revision_permisos','lectura_instantanea_sha256',
  'persona_ref','limite','cursor'];
BEGIN
 IF current_user<>'vec_meritos_propietario'
 OR session_user<>'vec_bolsa_inscripciones_rrhh_lector'
 OR current_setting('transaction_isolation')<>'serializable'
 OR current_setting('transaction_read_only')<>'off'
 OR current_setting('TimeZone')<>'UTC'
 OR jsonb_typeof(p_orden) IS DISTINCT FROM 'object'
 OR (SELECT count(*) FROM jsonb_object_keys(p_orden))<>array_length(claves,1)
 OR (p_orden ?& claves) IS NOT TRUE
 OR p_orden->>'accion' IS DISTINCT FROM 'meritos.hecho.revision_rrhh.consultar'
 OR p_orden->>'modulo_id' IS DISTINCT FROM 'meritos'
 OR p_orden->>'recurso_ref' !~ '^solicitud_inscripcion_[0-9a-f]{64}$'
 OR p_orden->>'finalidad_ref' IS DISTINCT FROM 'revision_meritos_inscripcion_rrhh'
 OR p_orden->>'proceso' IS DISTINCT FROM 'vec-server'
 OR p_orden->>'canal' IS DISTINCT FROM 'interna_corporativa'
 OR p_orden->>'persona_ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
 OR jsonb_typeof(p_orden->'limite') IS DISTINCT FROM 'number'
 OR coalesce(p_orden->>'limite','') !~ '^[1-9][0-9]*$'
 OR (p_orden->>'limite')::numeric NOT BETWEEN 1 AND 25
 OR jsonb_typeof(p_orden->'cursor') IS DISTINCT FROM 'string'
 THEN RAISE EXCEPTION 'RUM6: orden o sesión inválida' USING ERRCODE='42501'; END IF;
 persona:=p_orden->>'persona_ref';cursor:=p_orden->>'cursor';
 limite:=(p_orden->>'limite')::integer;
 orden_auditoria:=p_orden-'limite'-'cursor';
 BEGIN actor:=convert_from(p_contexto,'UTF8')::jsonb;
 EXCEPTION WHEN data_exception OR invalid_text_representation THEN
  RAISE EXCEPTION 'RUM6: contexto inválido' USING ERRCODE='42501'; END;
 IF vec_bolsa_llamamientos.cotejar_objetivo_revision_meritos_v1(
  p_orden->>'recurso_ref',persona,actor->>'principal_ref',actor->>'perfil_activo_ref',
  p_contexto,p_vinculo,p_captura_bolsa) IS NOT TRUE THEN
  SELECT * INTO STRICT acuse FROM vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(
   p_contexto,p_vinculo,orden_auditoria||jsonb_build_object(
    'resultado','denegado','motivo_ref','revision_meritos_rrhh_denegada'));
  RETURN jsonb_build_object('codigo','denegada','auditoria_ref',acuse.auditoria_ref);
 END IF;
 PERFORM set_config('vec_meritos.persona_ref',persona,true);
 IF cursor<>'' AND (vec_meritos.referencia_valida_v1(cursor) IS NOT TRUE OR
  NOT EXISTS(SELECT 1 FROM vec_meritos.hecho_identidad i
   WHERE i.persona_ref=persona AND i.hecho_ref=cursor)) THEN
  SELECT * INTO STRICT acuse FROM vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(
   p_contexto,p_vinculo,orden_auditoria||jsonb_build_object(
    'resultado','denegado','motivo_ref','revision_meritos_rrhh_denegada'));
  RETURN jsonb_build_object('codigo','denegada','auditoria_ref',acuse.auditoria_ref);
 END IF;
 -- El asiento se revierte junto a esta lectura si cualquier consulta falla.
 SELECT * INTO STRICT acuse FROM vec_autorizacion_atestada_v3.registrar_revision_rrhh_meritos_v1(
  p_contexto,p_vinculo,orden_auditoria||jsonb_build_object(
   'resultado','obtenida','motivo_ref','revision_meritos_rrhh_obtenida'));
 SELECT count(*) INTO total FROM vec_meritos.hecho_identidad i WHERE i.persona_ref=persona;
 FOR x IN
  SELECT i.hecho_ref,v.registro->'hecho' AS hecho
  FROM vec_meritos.hecho_identidad i
  JOIN LATERAL (SELECT hv.registro FROM vec_meritos.hecho_version hv
   WHERE hv.hecho_ref=i.hecho_ref AND hv.persona_ref=persona
   ORDER BY hv.version DESC LIMIT 1) v ON true
  WHERE i.persona_ref=persona AND (cursor='' OR i.hecho_ref>cursor)
  ORDER BY i.hecho_ref LIMIT limite+1
 LOOP
  contador:=contador+1;
  IF contador<=limite THEN
   hechos:=hechos||jsonb_build_array(x.hecho);
   ultimo:=x.hecho_ref;
  ELSE siguiente:=ultimo; END IF;
 END LOOP;
 recibo:=jsonb_build_object('auditoria_ref',acuse.auditoria_ref,
  'secuencia',acuse.secuencia::text,'huella_sha256',acuse.huella_sha256,
  'correlacion_ref',acuse.correlacion_ref,
  'registrada_en',to_char(acuse.registrada_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 RETURN jsonb_build_object('codigo','obtenida','esquema','vec.meritos.revision-rrhh.v1',
  'hechos',hechos,'total',total,'cursor_siguiente',siguiente,'recibo_acceso',recibo);
END $f$;
REVOKE ALL ON FUNCTION vec_meritos.consultar_revision_rrhh_v1(bytea,bytea,jsonb,jsonb) FROM PUBLIC;
GRANT USAGE ON SCHEMA vec_meritos TO vec_bolsa_llamamientos_lector_inscripciones_rrhh;
GRANT EXECUTE ON FUNCTION vec_meritos.consultar_revision_rrhh_v1(bytea,bytea,jsonb,jsonb)
 TO vec_bolsa_llamamientos_lector_inscripciones_rrhh;
COMMIT;
