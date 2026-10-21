\set ON_ERROR_STOP on
-- B149: fuente de la identidad documental de una aceptación nativa CT.
-- Bolsa solo entrega procedencia técnica de la fila; el servicio Bolsa
-- recupera el lote protegido y proyecta los cuatro campos autorizados.
-- Requiere AD306. Consulta auditada, sin DOWN ni datos personales en SQL.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000149',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
  OR to_regclass('vec_bolsa_llamamientos.origen_llamamiento_ct_v1') IS NULL
  OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
  OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
  OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_identidad_documento_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
  OR to_regprocedure('vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B149: preimagen incompatible' USING ERRCODE='55000';END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 s jsonb;c jsonb;d jsonb;canon text;h text;consumo record;
 o record;l record;r record;io record;b record;total bigint;fila integer;
 instantanea jsonb;bolsa jsonb;listado text;
 estado text:='no_encontrada';fuente jsonb:='null'::jsonb;
 claves constant text[]:=ARRAY['esquema','organizacion_ref','expediente_ref','clave_ct','unidad_ref',
  'categoria_ref','necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
  OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
  OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
  OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
  OR current_setting('transaction_isolation')<>'serializable'
  OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B149: sesión denegada' USING ERRCODE='42501';END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096
 THEN RAISE EXCEPTION 'B149: selector inválido' USING ERRCODE='22023';END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'B149: selector inválido' USING ERRCODE='22023';END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
  OR (SELECT count(*) FROM json_each(p_material::json))<>11
  OR (SELECT count(*) FROM jsonb_object_keys(s))<>11 OR NOT s ?& claves
  OR EXISTS(SELECT 1 FROM jsonb_each(s) x WHERE jsonb_typeof(x.value)<>'string')
  OR s->>'esquema' IS DISTINCT FROM 'vec.bolsa.aceptacion-nativa-ct.identidad-documento.consulta.v1'
  OR EXISTS(SELECT 1 FROM unnest(claves[2:11]) k
    WHERE octet_length(s->>k) NOT BETWEEN 1 AND 512
      OR (s->>k)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
 THEN RAISE EXCEPTION 'B149: selector inválido' USING ERRCODE='22023';END IF;
 SELECT '{'||string_agg(to_json(k)::text||':'||to_json(s->>k)::text,',' ORDER BY n)||'}'
 INTO canon FROM unnest(claves) WITH ORDINALITY x(k,n);
 IF canon IS DISTINCT FROM p_material
 THEN RAISE EXCEPTION 'B149: selector no canónico' USING ERRCODE='22023';END IF;
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":'||to_json(s->>'unidad_ref')::text||
  '},"atributos":{"material_sha256":"'||encode(sha256(convert_to(p_material,'UTF8')),'hex')||'"}}','UTF8')),'hex');
 IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
  OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
  OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
  OR octet_length(p_decision) NOT BETWEEN 1 AND 524288
  OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
  OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144
  OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
  OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576
  OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144 OR octet_length(p_raiz)<>44
  OR p_persona_version IS NULL OR p_perfil_version IS NULL
  OR p_persona_version NOT BETWEEN 1 AND 9007199254740991
  OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
  OR p_persona_version<>trunc(p_persona_version)
  OR p_perfil_version<>trunc(p_perfil_version)
 THEN RAISE EXCEPTION 'B149: autorización denegada' USING ERRCODE='42501';END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb;d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B149: autorización denegada' USING ERRCODE='42501';END;
 IF c->>'operacion' IS DISTINCT FROM 'bolsa.aceptacion_ct.identidad_documento.consultar'
  OR c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.aceptacion_ct.identidad_documento.v1'
  OR c->>'efecto_ref' IS DISTINCT FROM s->>'recibo_respuesta_ref'
  OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
  OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
  OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
  OR d->>'tipo_recurso' IS DISTINCT FROM 'identidad_documento_aceptacion_ct'
  OR d->>'finalidad' IS DISTINCT FROM 'redactar_documento_ct'
  OR d->>'recurso_ref' IS DISTINCT FROM s->>'recibo_respuesta_ref'
  OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
  OR d->'campos_permitidos' IS DISTINCT FROM '["apellidos","documento_enmascarado","nombre","version_fuente"]'::jsonb
  OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B149: autorización divergente' USING ERRCODE='42501';END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_identidad_documento_aceptacion_ct_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'recibo_respuesta_ref'
  OR consumo.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'B149: consumo divergente' USING ERRCODE='42501';END IF;

 <<lectura>>
 BEGIN
  SELECT * INTO o FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
   WHERE expediente_ref=s->>'expediente_ref' AND clave_ct=s->>'clave_ct';
  IF NOT FOUND THEN EXIT lectura;END IF;
  IF o.organizacion_ref IS DISTINCT FROM s->>'organizacion_ref'
   OR o.unidad_ref IS DISTINCT FROM s->>'unidad_ref'
   OR o.categoria_ref IS DISTINCT FROM s->>'categoria_ref'
   OR o.necesidad_ref IS DISTINCT FROM s->>'necesidad_ref'
   OR o.bolsa_ref IS DISTINCT FROM s->>'bolsa_ref'
   OR o.llamamiento_ref IS DISTINCT FROM s->>'llamamiento_ref'
   OR o.recibo_ref IS DISTINCT FROM s->>'recibo_emision_ref'
  THEN EXIT lectura;END IF;
  IF o.origen_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(o.origen_canonico::text,'UTF8')),'hex')
   OR o.origen_canonico->>'unidad_ref' IS DISTINCT FROM o.unidad_ref
  THEN RAISE EXCEPTION 'B149: origen divergente' USING ERRCODE='P1495';END IF;
  SELECT * INTO l FROM vec_bolsa_llamamientos.llamamiento_emitido
   WHERE llamamiento_ref=o.llamamiento_ref AND recibo_ref=o.recibo_ref AND bolsa_ref=o.bolsa_ref;
  IF NOT FOUND OR l.decision_ref IS DISTINCT FROM o.decision_ref
   OR l.emitido_en IS DISTINCT FROM o.emitido_en
   OR l.participaciones IS DISTINCT FROM jsonb_build_array(o.participacion_ref)
   OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ev
     WHERE ev.evento_ref=o.evento_ref AND ev.llamamiento_ref=o.llamamiento_ref)
  THEN RAISE EXCEPTION 'B149: emisión original divergente' USING ERRCODE='P1495';END IF;
  SELECT * INTO r FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento
   WHERE recibo_ref=s->>'recibo_respuesta_ref';
  IF NOT FOUND OR r.llamamiento_ref IS DISTINCT FROM o.llamamiento_ref
   OR r.bolsa_ref IS DISTINCT FROM o.bolsa_ref
   OR r.participacion_ref IS DISTINCT FROM o.participacion_ref
   OR r.candidato_ref IS DISTINCT FROM o.candidato_ref
   OR r.respuesta IS DISTINCT FROM 'acepta' OR r.modo IS DISTINCT FROM 'firme'
   OR r.respondida_en<l.emitido_en
   OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion cp
     WHERE cp.llamamiento_ref=r.llamamiento_ref AND cp.bolsa_ref=r.bolsa_ref
      AND cp.participacion_ref=r.participacion_ref AND cp.instante=r.contacto_en)
  THEN EXIT lectura;END IF;
  SELECT count(*),min(ce.fila_numero) INTO total,fila
   FROM vec_bolsa_llamamientos.constitucion ct
   JOIN vec_bolsa_llamamientos.constitucion_entrada ce
    ON ce.instantanea_ref=ct.instantanea_ref AND ce.version_instantanea=ct.version_instantanea
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc
    ON vc.participacion_ref=ce.participacion_ref AND vc.acta_ref=ct.acta_ref
    AND vc.instantanea_ref=ct.instantanea_ref AND vc.version_instantanea=ct.version_instantanea
   JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa io
    ON io.instantanea_ref=ct.instantanea_ref AND io.version=ct.version_instantanea
    AND io.huella_instantanea_sha256=ct.huella_instantanea_sha256
   WHERE ct.acta_ref=o.acta_ref AND ct.bolsa_ref=o.bolsa_ref
    AND ct.version_bolsa=o.version_bolsa AND ct.huella_bolsa_sha256=o.huella_bolsa_sha256
    AND ct.categoria_ref=o.categoria_ref AND ct.instantanea_ref=o.instantanea_ref
    AND ct.version_instantanea=o.version_instantanea
    AND ct.huella_instantanea_sha256=o.huella_instantanea_sha256
    AND ce.participacion_ref=o.participacion_ref AND ce.orden=o.orden_seleccionado
    AND vc.candidato_ref=o.candidato_ref
    AND encode(sha256(io.instantanea_canonica),'hex')=io.huella_instantanea_sha256;
  IF total<>1 OR fila IS NULL THEN estado:='pendiente';EXIT lectura;END IF;
  SELECT * INTO io FROM vec_bolsa_llamamientos.instantanea_orden_bolsa
   WHERE instantanea_ref=o.instantanea_ref AND version=o.version_instantanea
    AND huella_instantanea_sha256=o.huella_instantanea_sha256;
  IF NOT FOUND THEN RAISE EXCEPTION 'B149: instantánea original ausente' USING ERRCODE='P1495';END IF;
  SELECT * INTO b FROM vec_bolsa_llamamientos.bolsa_constituida
   WHERE bolsa_ref=o.bolsa_ref AND version=o.version_bolsa
    AND huella_bolsa_sha256=o.huella_bolsa_sha256;
  IF NOT FOUND THEN RAISE EXCEPTION 'B149: bolsa original ausente' USING ERRCODE='P1495';END IF;
  IF io.bolsa_ref IS DISTINCT FROM o.bolsa_ref
   OR io.version_bolsa IS DISTINCT FROM o.version_bolsa
   OR io.huella_bolsa_sha256 IS DISTINCT FROM o.huella_bolsa_sha256
   OR b.categoria_ref IS DISTINCT FROM o.categoria_ref
   OR encode(sha256(io.instantanea_canonica),'hex') IS DISTINCT FROM io.huella_instantanea_sha256
   OR encode(sha256(b.bolsa_canonica),'hex') IS DISTINCT FROM b.huella_bolsa_sha256
  THEN RAISE EXCEPTION 'B149: fuente original divergente' USING ERRCODE='P1495';END IF;
  BEGIN
   instantanea:=convert_from(io.instantanea_canonica,'UTF8')::jsonb;
   bolsa:=convert_from(b.bolsa_canonica,'UTF8')::jsonb;
  EXCEPTION WHEN data_exception THEN
   RAISE EXCEPTION 'B149: fuente original inválida' USING ERRCODE='P1495';
  END;
  listado:=instantanea->>'huella_listado_sha256';
  IF listado IS NULL OR listado!~'^[0-9a-f]{64}$'
   OR bolsa->>'huella_listado_sha256' IS DISTINCT FROM listado
   OR instantanea->>'instantanea_ref' IS DISTINCT FROM o.instantanea_ref
   OR instantanea->>'version' IS DISTINCT FROM o.version_instantanea::text
   OR instantanea->>'bolsa_ref' IS DISTINCT FROM o.bolsa_ref
   OR instantanea->>'version_bolsa' IS DISTINCT FROM o.version_bolsa::text
   OR instantanea->>'huella_bolsa_sha256' IS DISTINCT FROM o.huella_bolsa_sha256
   OR bolsa->>'bolsa_ref' IS DISTINCT FROM o.bolsa_ref
   OR bolsa->>'version' IS DISTINCT FROM o.version_bolsa::text
   OR bolsa->>'categoria_ref' IS DISTINCT FROM o.categoria_ref
  THEN RAISE EXCEPTION 'B149: listado original divergente' USING ERRCODE='P1495';END IF;
  fuente:=jsonb_build_object('acta_ref',o.acta_ref,'fila_numero',fila,
   'instantanea_ref',o.instantanea_ref,'version_instantanea',o.version_instantanea,
   'huella_instantanea_sha256',o.huella_instantanea_sha256,'bolsa_ref',o.bolsa_ref,
   'huella_listado_sha256',listado,
   'recibo_emision_ref',o.recibo_ref,'recibo_respuesta_ref',r.recibo_ref);
  estado:='acreditado';
 END lectura;
 RETURN jsonb_build_object('estado',estado,'fuente',fuente,'evidencia',
  jsonb_build_object('decision_ref',consumo.decision_ref,
   'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_identidad_documental_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
