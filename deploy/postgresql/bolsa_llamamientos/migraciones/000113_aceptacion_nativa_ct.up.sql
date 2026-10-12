\set ON_ERROR_STOP on
-- B113: aceptación firme B30 de la emisión CT B110, con consumo nominal
-- AD128/AD131 y acreditación CTX18. No modifica hechos de Bolsa ni de CT.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog,pg_temp;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000113',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.origen_llamamiento_ct_v1') IS NULL
    OR NOT EXISTS(SELECT 1 FROM information_schema.columns
      WHERE table_schema='vec_bolsa_llamamientos' AND table_name='origen_llamamiento_ct_v1' AND column_name='unidad_ref')
    OR to_regclass('vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.respuesta_portal_llamamiento') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B113: preimagen incompatible o falta unidad del origen B110' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1(
 p_material text,p_tipo text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $f$
DECLARE
 s jsonb; c jsonb; d jsonb; canon text; h text; material_hash text; consumo record;
 o record; l record; r record; total bigint; persona_ctx jsonb;
 anclaje jsonb:='null'::jsonb; persona jsonb:='null'::jsonb; vinculo jsonb:='null'::jsonb;
 estado text:='no_encontrada'; accion text; audiencia text; tipo_recurso text; campos jsonb;
 claves constant text[]:=ARRAY['esquema','organizacion_ref','expediente_ref','clave_ct','unidad_ref','categoria_ref','necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B113: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_tipo NOT IN ('anclaje','persona') OR p_material IS NULL
    OR octet_length(p_material) NOT BETWEEN 1 AND 4096
 THEN RAISE EXCEPTION 'B113: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'B113: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM json_each(p_material::json))<>11
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>11 OR NOT s ?& claves
    OR EXISTS(SELECT 1 FROM jsonb_each(s) x WHERE jsonb_typeof(x.value)<>'string')
    OR s->>'esquema' IS DISTINCT FROM 'vec.bolsa.aceptacion-nativa-ct.'||p_tipo||'.consulta.v1'
    OR EXISTS(SELECT 1 FROM unnest(claves[2:11]) k
      WHERE octet_length(s->>k) NOT BETWEEN 1 AND 512
        OR (s->>k)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
 THEN RAISE EXCEPTION 'B113: selector inválido' USING ERRCODE='22023'; END IF;
 SELECT '{'||string_agg(to_json(k)::text||':'||to_json(s->>k)::text,',' ORDER BY n)||'}'
 INTO canon FROM unnest(claves) WITH ORDINALITY x(k,n);
 IF canon IS DISTINCT FROM p_material
 THEN RAISE EXCEPTION 'B113: material no canónico' USING ERRCODE='22023'; END IF;
 material_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":'||to_json(s->>'unidad_ref')::text||
  '},"atributos":{"material_sha256":"'||material_hash||'"}}','UTF8')),'hex');
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
 THEN RAISE EXCEPTION 'B113: autorización denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B113: autorización denegada' USING ERRCODE='42501'; END;
 IF p_tipo='anclaje' THEN
  accion:='bolsa.aceptacion_ct.anclaje.consultar';
  audiencia:='vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1';
  tipo_recurso:='anclaje_aceptacion_ct'; campos:='["anclaje"]'::jsonb;
 ELSE
  accion:='bolsa.aceptacion_ct.persona.consultar';
  audiencia:='vec_bolsa_llamamientos.aceptacion_ct.persona.v1';
  tipo_recurso:='persona_aceptacion_ct'; campos:='["aceptacion","persona","vinculo"]'::jsonb;
 END IF;
 IF c->>'audiencia_consumo' IS DISTINCT FROM audiencia OR c->>'operacion' IS DISTINCT FROM accion
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'recibo_respuesta_ref'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->>'accion' IS DISTINCT FROM accion OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM tipo_recurso
    OR d->>'finalidad' IS DISTINCT FROM 'preparar_incorporacion_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'recibo_respuesta_ref'
    OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
    OR d->'campos_permitidos' IS DISTINCT FROM campos OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B113: autorización divergente' USING ERRCODE='42501'; END IF;
 IF p_tipo='anclaje' THEN
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 ELSE
  SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(
   p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF consumo.consumo_nuevo IS NOT TRUE
    OR consumo.efecto_ref IS DISTINCT FROM s->>'recibo_respuesta_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'B113: consumo divergente' USING ERRCODE='42501'; END IF;

 <<lectura>>
 BEGIN
  SELECT * INTO o FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
   WHERE expediente_ref=s->>'expediente_ref' AND clave_ct=s->>'clave_ct';
  IF NOT FOUND THEN EXIT lectura; END IF;
  IF o.organizacion_ref IS DISTINCT FROM s->>'organizacion_ref'
     OR o.unidad_ref IS DISTINCT FROM s->>'unidad_ref'
     OR o.categoria_ref IS DISTINCT FROM s->>'categoria_ref'
     OR o.necesidad_ref IS DISTINCT FROM s->>'necesidad_ref'
     OR o.bolsa_ref IS DISTINCT FROM s->>'bolsa_ref'
     OR o.llamamiento_ref IS DISTINCT FROM s->>'llamamiento_ref'
     OR o.recibo_ref IS DISTINCT FROM s->>'recibo_emision_ref'
  THEN EXIT lectura; END IF;
  IF o.origen_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(o.origen_canonico::text,'UTF8')),'hex')
     OR o.origen_canonico->>'unidad_ref' IS DISTINCT FROM o.unidad_ref
  THEN RAISE EXCEPTION 'B113: origen divergente' USING ERRCODE='P1135'; END IF;
  SELECT * INTO l FROM vec_bolsa_llamamientos.llamamiento_emitido
   WHERE llamamiento_ref=o.llamamiento_ref AND recibo_ref=o.recibo_ref AND bolsa_ref=o.bolsa_ref;
  IF NOT FOUND OR l.decision_ref IS DISTINCT FROM o.decision_ref
     OR l.emitido_en IS DISTINCT FROM o.emitido_en
     OR l.participaciones IS DISTINCT FROM jsonb_build_array(o.participacion_ref)
     OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ev
       WHERE ev.evento_ref=o.evento_ref AND ev.llamamiento_ref=o.llamamiento_ref)
  THEN RAISE EXCEPTION 'B113: emisión original divergente' USING ERRCODE='P1135'; END IF;
  SELECT count(*) INTO total FROM vec_bolsa_llamamientos.constitucion ct
   JOIN vec_bolsa_llamamientos.constitucion_entrada ce
     ON ce.instantanea_ref=ct.instantanea_ref AND ce.version_instantanea=ct.version_instantanea
   JOIN vec_bolsa_llamamientos.vinculo_candidato vc
     ON vc.participacion_ref=ce.participacion_ref AND vc.acta_ref=ct.acta_ref
    AND vc.instantanea_ref=ct.instantanea_ref AND vc.version_instantanea=ct.version_instantanea
   WHERE ct.acta_ref=o.acta_ref AND ct.bolsa_ref=o.bolsa_ref
    AND ct.version_bolsa=o.version_bolsa AND ct.huella_bolsa_sha256=o.huella_bolsa_sha256
    AND ct.categoria_ref=o.categoria_ref AND ct.instantanea_ref=o.instantanea_ref
    AND ct.version_instantanea=o.version_instantanea
    AND ct.huella_instantanea_sha256=o.huella_instantanea_sha256
    AND ce.participacion_ref=o.participacion_ref AND ce.orden=o.orden_seleccionado
    AND vc.candidato_ref=o.candidato_ref;
  IF total<>1 THEN RAISE EXCEPTION 'B113: selección original divergente' USING ERRCODE='P1135'; END IF;
  SELECT * INTO r FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento
   WHERE recibo_ref=s->>'recibo_respuesta_ref';
  IF NOT FOUND THEN EXIT lectura; END IF;
  IF r.llamamiento_ref IS DISTINCT FROM o.llamamiento_ref
     OR r.bolsa_ref IS DISTINCT FROM o.bolsa_ref
     OR r.participacion_ref IS DISTINCT FROM o.participacion_ref
     OR r.candidato_ref IS DISTINCT FROM o.candidato_ref
     OR r.respuesta IS DISTINCT FROM 'acepta' OR r.modo IS DISTINCT FROM 'firme'
     OR r.respondida_en<l.emitido_en
     OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion cp
       WHERE cp.llamamiento_ref=r.llamamiento_ref AND cp.bolsa_ref=r.bolsa_ref
         AND cp.participacion_ref=r.participacion_ref AND cp.instante=r.contacto_en)
  THEN EXIT lectura; END IF;
  anclaje:=jsonb_build_object(
   'organizacion_ref',o.organizacion_ref,'expediente_ref',o.expediente_ref,
   'clave_ct',o.clave_ct,'unidad_ref',o.unidad_ref,'categoria_ref',o.categoria_ref,
   'necesidad_ref',o.necesidad_ref,'bolsa_ref',o.bolsa_ref,
   'llamamiento_ref',o.llamamiento_ref,'recibo_emision_ref',o.recibo_ref,
   'recibo_respuesta_ref',r.recibo_ref,'version_expediente',o.version_expediente,
   'acta_ref',o.acta_ref,'version_bolsa',o.version_bolsa,
   'huella_bolsa_sha256',o.huella_bolsa_sha256,
   'instantanea_ref',o.instantanea_ref,'version_instantanea',o.version_instantanea,
   'huella_instantanea_sha256',o.huella_instantanea_sha256,
   'politica_ref',o.politica_ref,'version_politica',o.version_politica,
   'huella_politica_sha256',o.huella_politica_sha256,
   'participacion_ref',o.participacion_ref,'candidato_ref',o.candidato_ref,
   'orden_seleccionado',o.orden_seleccionado,'respuesta_ref',r.respuesta_ref,
   'respondida_en',to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
  estado:='acreditado';
  IF p_tipo='persona' THEN
   persona_ctx:=vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(o.candidato_ref);
   IF jsonb_typeof(persona_ctx) IS DISTINCT FROM 'object'
      OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(persona_ctx) k)
         IS DISTINCT FROM ARRAY['estado','persona','vinculo']
      OR persona_ctx->>'estado' NOT IN ('pendiente','acreditado')
   THEN RAISE EXCEPTION 'B113: acreditación de persona incompatible' USING ERRCODE='P1135'; END IF;
   IF persona_ctx->>'estado'='pendiente' THEN
    IF persona_ctx->'persona' IS DISTINCT FROM 'null'::jsonb
       OR persona_ctx->'vinculo' IS DISTINCT FROM 'null'::jsonb
    THEN RAISE EXCEPTION 'B113: persona pendiente incompatible' USING ERRCODE='P1135'; END IF;
    estado:='pendiente';
   ELSE
    persona:=persona_ctx->'persona'; vinculo:=persona_ctx->'vinculo';
    IF jsonb_typeof(persona)<>'object' OR jsonb_typeof(vinculo)<>'object'
       OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(persona) k)
          IS DISTINCT FROM ARRAY['ref','version']
       OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(vinculo) k)
          IS DISTINCT FROM ARRAY['poblacion','procedencia_ref','procedencia_sha256','procedencia_version','ref','version','vigente_hasta']
       OR jsonb_typeof(persona->'ref') IS DISTINCT FROM 'string'
       OR persona->>'ref' !~ '^per_[A-Za-z0-9_-]{22,128}$'
       OR jsonb_typeof(persona->'version') IS DISTINCT FROM 'number'
       OR persona->>'version' !~ '^[1-9][0-9]{0,18}$'
       OR EXISTS(SELECT 1 FROM unnest(ARRAY['ref','procedencia_ref','procedencia_sha256','poblacion','vigente_hasta']) k
         WHERE jsonb_typeof(vinculo->k) IS DISTINCT FROM 'string'
           OR octet_length(vinculo->>k) NOT BETWEEN 1 AND 512)
       OR EXISTS(SELECT 1 FROM unnest(ARRAY['version','procedencia_version']) k
         WHERE jsonb_typeof(vinculo->k) IS DISTINCT FROM 'number'
           OR vinculo->>k !~ '^[1-9][0-9]{0,18}$')
       OR vinculo->>'poblacion' NOT IN ('interna','externa')
       OR vinculo->>'procedencia_sha256' !~ '^[0-9a-f]{64}$'
       OR vinculo->>'procedencia_sha256'=repeat('0',64)
       OR vinculo->>'vigente_hasta' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR (persona->>'version')::numeric>9223372036854775807
       OR (vinculo->>'version')::numeric>9223372036854775807
       OR (vinculo->>'procedencia_version')::numeric>9223372036854775807
       OR (vinculo->>'vigente_hasta')::timestamptz<=clock_timestamp()
    THEN RAISE EXCEPTION 'B113: procedencia de persona incompatible' USING ERRCODE='P1135'; END IF;
   END IF;
  END IF;
 END lectura;
 IF p_tipo='anclaje' THEN
  RETURN jsonb_build_object('estado',estado,'anclaje',anclaje,'evidencia',
   jsonb_build_object('decision_ref',consumo.decision_ref,
   'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
 END IF;
 RETURN jsonb_build_object('estado',estado,'aceptacion',anclaje,'persona',persona,
  'vinculo',vinculo,'evidencia',jsonb_build_object('decision_ref',consumo.decision_ref,
   'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,
   'consultada_en',to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_aceptacion_nativa_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
COMMIT;
