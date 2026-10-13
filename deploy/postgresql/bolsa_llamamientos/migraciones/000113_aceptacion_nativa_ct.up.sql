\set ON_ERROR_STOP on
-- B113: aceptación firme B30 de la emisión CT B110 por las fachadas B67/B68
-- existentes. Reutiliza sus consumos AD128/AD131 y la acreditación CTX18;
-- la implementación nueva no recibe EXECUTE del LOGIN.
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
    OR to_regprocedure('vec_bolsa_llamamientos.consulta_aceptacion_nativa_ct_privada_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
    OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
       'vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
       IS DISTINCT FROM '292a66e7e87438a8caf3ae02878d878d84addddfd01fa877364ab9fbf4ce073b'
    OR encode(sha256(convert_to(pg_get_functiondef(to_regprocedure(
       'vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')),'UTF8')),'hex')
       IS DISTINCT FROM '66a3d6d857e9629f598760295e32e7dc43e41e89b9848a2a24910540d7dba078'
    OR to_regprocedure('vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)') IS NOT NULL
    OR to_regrole('vec_contratacion_temporal_propietario') IS NULL
    OR to_regrole('vec_contratacion_temporal_ejecutor') IS NULL
 THEN RAISE EXCEPTION 'B113: preimagen incompatible o falta unidad del origen B110' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consulta_aceptacion_nativa_ct_privada_v1(
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
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consulta_aceptacion_nativa_ct_privada_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)
 FROM PUBLIC,vec_bolsa_llamamientos_ejecutor,vec_contratacion_temporal_ejecutor;

-- B67/B68: definición instalada 13/10b, con una sola rama nativa cerrada.
CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '2s'
    AS $_$
DECLARE
 s jsonb; c jsonb; d jsonb; h text; material_hash text; canon text; consumo record;
 terminal record; apertura record; llamamiento record; t jsonb; a jsonb; i jsonb; p jsonb;
 total bigint;
 estado text:='no_encontrada'; anclaje jsonb:='null'::jsonb;
 claves constant text[]:=ARRAY['esquema','unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','aceptacion_registro_sha256','apertura_operacion_ref','llamamiento_ref','propuesta_ref'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B68: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096
 THEN RAISE EXCEPTION 'B68: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'B68: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B68: objeto requerido' USING ERRCODE='22023'; END IF;
 IF s->>'esquema' IS NOT DISTINCT FROM 'vec.bolsa.aceptacion-nativa-ct.anclaje.consulta.v1' THEN
  RETURN vec_bolsa_llamamientos.consulta_aceptacion_nativa_ct_privada_v1(
   p_material,'anclaje',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
   p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF (SELECT count(*) FROM json_each(p_material::json))<>9
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>9 OR NOT s ?& claves
    OR EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string')
    OR s->>'esquema' IS DISTINCT FROM 'vec.bolsa.anclaje-aceptacion-ct.consulta.v1'
    OR EXISTS (SELECT 1 FROM unnest(ARRAY['unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','apertura_operacion_ref','llamamiento_ref','propuesta_ref']) k
               WHERE octet_length(s->>k) NOT BETWEEN 1 AND 512 OR (s->>k)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
    OR EXISTS (SELECT 1 FROM unnest(ARRAY['aceptacion_registro_sha256']) k
               WHERE (s->>k)!~'^[0-9a-f]{64}$' OR s->>k=repeat('0',64))
 THEN RAISE EXCEPTION 'B68: campos inválidos' USING ERRCODE='22023'; END IF;
 SELECT '{'||string_agg(to_json(k)::text||':'||to_json(s->>k)::text,',' ORDER BY n)||'}'
 INTO canon FROM unnest(claves) WITH ORDINALITY x(k,n);
 IF canon IS DISTINCT FROM p_material
 THEN RAISE EXCEPTION 'B68: material no canónico' USING ERRCODE='22023'; END IF;
 material_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":'||to_json(s->>'unidad_ref')::text||
    '},"atributos":{"material_sha256":"'||material_hash||'"}}','UTF8')),'hex');
 IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
    OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144 OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576 OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR octet_length(p_raiz)<>44 OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_persona_version NOT BETWEEN 1 AND 9007199254740991 OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
    OR p_persona_version<>trunc(p_persona_version) OR p_perfil_version<>trunc(p_perfil_version)
 THEN RAISE EXCEPTION 'B68: autorización denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B68: autorización denegada' USING ERRCODE='42501'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.aceptacion_ct.anclaje.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.aceptacion_ct.anclaje.consultar'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'aceptacion_operacion_ref'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'anclaje_aceptacion_ct' OR d->>'finalidad' IS DISTINCT FROM 'preparar_incorporacion_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'aceptacion_operacion_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
    OR d->'campos_permitidos' IS DISTINCT FROM '["anclaje"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B68: autorización divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'aceptacion_operacion_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'B68: consumo divergente' USING ERRCODE='42501'; END IF;

 <<lectura>>
 BEGIN
  SELECT * INTO terminal FROM vec_bolsa_llamamientos.integracion_desarrollo
   WHERE operacion_ref=s->>'aceptacion_operacion_ref' AND tipo='aceptacion_rrhh';
  IF NOT FOUND THEN EXIT lectura; END IF;
  IF terminal.registro_huella_sha256 IS DISTINCT FROM encode(sha256(terminal.registro_canonico),'hex')
  THEN RAISE EXCEPTION 'B68: aceptación inconsistente' USING ERRCODE='P0685'; END IF;
  IF terminal.registro_huella_sha256 IS DISTINCT FROM s->>'aceptacion_registro_sha256'
     OR terminal.apertura_operacion_ref IS DISTINCT FROM s->>'apertura_operacion_ref'
  THEN EXIT lectura; END IF;
  SELECT * INTO apertura FROM vec_bolsa_llamamientos.integracion_desarrollo
   WHERE operacion_ref=terminal.apertura_operacion_ref AND tipo='propuesta';
  IF NOT FOUND THEN RAISE EXCEPTION 'B68: apertura inconsistente' USING ERRCODE='P0685'; END IF;
  IF apertura.registro_huella_sha256 IS DISTINCT FROM encode(sha256(apertura.registro_canonico),'hex')
  THEN RAISE EXCEPTION 'B68: apertura inconsistente' USING ERRCODE='P0685'; END IF;
  t:=convert_from(terminal.registro_canonico,'UTF8')::jsonb;
  a:=convert_from(apertura.registro_canonico,'UTF8')::jsonb;
  IF t->>'esquema' IS DISTINCT FROM 'vec.bolsa.integracion-llamamientos-desarrollo.v1'
     OR a->>'esquema' IS DISTINCT FROM 'vec.bolsa.integracion-llamamientos-desarrollo.v1'
     OR t->>'operacion_ref' IS DISTINCT FROM terminal.operacion_ref OR a->>'operacion_ref' IS DISTINCT FROM apertura.operacion_ref
     OR t->>'tipo' IS DISTINCT FROM 'aceptacion_rrhh' OR a->>'tipo' IS DISTINCT FROM 'propuesta'
     OR t->>'estado_llamamiento' IS DISTINCT FROM 'aceptacion' OR a->>'estado_llamamiento' IS DISTINCT FROM 'abierto'
     OR t#>>'{resolucion,apertura_operacion_ref}' IS DISTINCT FROM apertura.operacion_ref
     OR terminal.necesidad_ref IS DISTINCT FROM apertura.necesidad_ref OR terminal.version_necesidad IS DISTINCT FROM apertura.version_necesidad
     OR terminal.orden_operacion_ref IS DISTINCT FROM apertura.orden_operacion_ref
     OR (t-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion']) IS DISTINCT FROM (a-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion'])
     OR t#>'{llamamiento,Version}' IS DISTINCT FROM '2'::jsonb OR a#>'{llamamiento,Version}' IS DISTINCT FROM '1'::jsonb
     OR ((t->'llamamiento')-'Version') IS DISTINCT FROM ((a->'llamamiento')-'Version')
  THEN RAISE EXCEPTION 'B68: aceptación desligada de la apertura original' USING ERRCODE='P0685'; END IF;
  IF t->>'unidad_ref' IS DISTINCT FROM s->>'unidad_ref' OR t->>'categoria_ref' IS DISTINCT FROM s->>'categoria_ref'
     OR t->>'necesidad_ref' IS DISTINCT FROM s->>'necesidad_ref'
     OR terminal.necesidad_ref IS DISTINCT FROM s->>'necesidad_ref'
     OR a#>>'{llamamiento,LlamamientoRef}' IS DISTINCT FROM s->>'llamamiento_ref'
     OR a#>>'{llamamiento,PropuestaRef}' IS DISTINCT FROM s->>'propuesta_ref'
  THEN EXIT lectura; END IF;
  SELECT * INTO llamamiento FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo
   WHERE operacion_ref=apertura.operacion_ref;
  IF NOT FOUND OR llamamiento.llamamiento_ref IS DISTINCT FROM s->>'llamamiento_ref'
     OR llamamiento.propuesta_ref IS DISTINCT FROM s->>'propuesta_ref' OR llamamiento.necesidad_ref IS DISTINCT FROM terminal.necesidad_ref
     OR llamamiento.version IS DISTINCT FROM 1::bigint OR llamamiento.estado IS DISTINCT FROM 'abierto'
     OR llamamiento.datos_canonicos IS DISTINCT FROM a->'llamamiento'
  THEN RAISE EXCEPTION 'B68: llamamiento original inconsistente' USING ERRCODE='P0685'; END IF;
  i:=a->'instantanea'; p:=a->'propuesta';
  IF p->>'propuesta_ref' IS DISTINCT FROM s->>'propuesta_ref' OR p->>'necesidad_ref' IS DISTINCT FROM terminal.necesidad_ref
     OR p->>'instantanea_ref' IS DISTINCT FROM i->>'instantanea_ref' OR p->>'version_instantanea' IS DISTINCT FROM i->>'version'
     OR p->>'huella_instantanea_sha256' IS DISTINCT FROM i->>'huella_contenido_sha256'
     OR p->>'bolsa_ref' IS DISTINCT FROM i->>'bolsa_ref' OR p->>'version_bolsa' IS DISTINCT FROM i->>'version_bolsa'
     OR p->>'huella_bolsa_sha256' IS DISTINCT FROM i->>'huella_bolsa_sha256'
     OR llamamiento.bolsa_ref IS DISTINCT FROM i->>'bolsa_ref' OR a#>>'{llamamiento,BolsaRef}' IS DISTINCT FROM i->>'bolsa_ref'
     OR i->>'bolsa_ref' IS DISTINCT FROM a#>>'{fuente,datos,Bolsa,bolsa_ref}' OR i->>'version_bolsa' IS DISTINCT FROM a#>>'{fuente,datos,Bolsa,version}'
     OR i->'entradas' IS DISTINCT FROM a#>'{fuente,datos,Entradas}'
     OR a#>>'{fuente,datos,Necesidad,unidad_ref}' IS DISTINCT FROM s->>'unidad_ref'
     OR a#>>'{fuente,datos,Necesidad,categoria_ref}' IS DISTINCT FROM s->>'categoria_ref'
     OR p->>'participacion_seleccionada_ref' IS NULL
  THEN RAISE EXCEPTION 'B68: selección original inconsistente' USING ERRCODE='P0685'; END IF;
  estado:='pendiente';
  -- El vínculo B8 debe pertenecer a la selección y constitución originales.
  SELECT count(*) INTO total
  FROM vec_bolsa_llamamientos.vinculo_candidato v
  JOIN vec_bolsa_llamamientos.constitucion ct ON ct.acta_ref=v.acta_ref
   AND ct.instantanea_ref=v.instantanea_ref AND ct.version_instantanea=v.version_instantanea
  JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=v.instantanea_ref
   AND e.version_instantanea=v.version_instantanea AND e.participacion_ref=v.participacion_ref
  JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa io ON io.instantanea_ref=ct.instantanea_ref
   AND io.version=ct.version_instantanea AND io.huella_instantanea_sha256=ct.huella_instantanea_sha256
  WHERE v.participacion_ref=p->>'participacion_seleccionada_ref'
    AND ct.categoria_ref=s->>'categoria_ref' AND ct.bolsa_ref=i->>'bolsa_ref' AND ct.version_bolsa::text=i->>'version_bolsa'
    AND ct.huella_bolsa_sha256=i->>'huella_bolsa_sha256'
    AND ct.instantanea_ref=i->>'instantanea_ref' AND ct.version_instantanea::text=i->>'version'
    -- B7 guarda sha256 del JSON completo; el dominio calcula
    -- huella_contenido_sha256 sin ese campo. Son huellas diferentes.
    AND ct.huella_instantanea_sha256=encode(sha256(convert_to(
        (convert_from(apertura.registro_canonico,'UTF8')::json->'instantanea')::text,'UTF8')),'hex')
    AND e.orden::text=p->>'orden_seleccionado'
    AND encode(sha256(io.instantanea_canonica),'hex')=io.huella_instantanea_sha256
    AND convert_from(io.instantanea_canonica,'UTF8')=
        (convert_from(apertura.registro_canonico,'UTF8')::json->'instantanea')::text;
  IF total<>1 THEN EXIT lectura; END IF;
  anclaje:=jsonb_build_object(
   'unidad_ref',s->>'unidad_ref','categoria_ref',s->>'categoria_ref','necesidad_ref',s->>'necesidad_ref',
   'aceptacion_operacion_ref',terminal.operacion_ref,'aceptacion_registro_sha256',terminal.registro_huella_sha256,
   'apertura_operacion_ref',apertura.operacion_ref,'apertura_registro_sha256',apertura.registro_huella_sha256,
   'llamamiento_ref',llamamiento.llamamiento_ref,'propuesta_ref',llamamiento.propuesta_ref,
   'aceptacion_recibo_ref',terminal.recibo_ref);
  estado:='acreditado';
 EXCEPTION WHEN data_exception THEN
  RAISE EXCEPTION 'B68: registro original inconsistente' USING ERRCODE='P0685';
 END lectura;
 RETURN jsonb_build_object('estado',estado,'anclaje',anclaje,
  'evidencia',jsonb_build_object('decision_ref',consumo.decision_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,'consultada_en',to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
EXCEPTION WHEN deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'B68: consulta transitoria' USING ERRCODE='P0685';
END $_$;


CREATE OR REPLACE FUNCTION vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(p_material text, p_capacidad bytea, p_decision bytea, p_motivo bytea, p_contexto bytea, p_persona_version numeric, p_perfil_version numeric, p_payload bytea, p_sobre bytea, p_evidencia bytea, p_raiz bytea) RETURNS jsonb
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path TO 'pg_catalog', 'pg_temp'
    SET row_security TO 'on'
    SET "TimeZone" TO 'UTC'
    SET lock_timeout TO '2s'
    AS $_$
DECLARE
 s jsonb; c jsonb; d jsonb; h text; material_hash text; canon text; consumo record;
 terminal record; apertura record; llamamiento record; t jsonb; a jsonb; i jsonb; p jsonb;
 candidato text; total bigint; persona_ctx jsonb;
 estado text:='no_encontrada'; aceptacion jsonb:='null'::jsonb; persona jsonb:='null'::jsonb; vinculo jsonb:='null'::jsonb;
 claves constant text[]:=ARRAY['esquema','unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','aceptacion_registro_sha256','apertura_operacion_ref','apertura_registro_sha256','llamamiento_ref','propuesta_ref'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B67: sesión denegada' USING ERRCODE='42501'; END IF;
 IF p_material IS NULL OR octet_length(p_material) NOT BETWEEN 1 AND 4096
 THEN RAISE EXCEPTION 'B67: material inválido' USING ERRCODE='22023'; END IF;
 BEGIN s:=p_material::jsonb;
 EXCEPTION WHEN data_exception THEN RAISE EXCEPTION 'B67: material inválido' USING ERRCODE='22023'; END;
 IF jsonb_typeof(s) IS DISTINCT FROM 'object'
 THEN RAISE EXCEPTION 'B67: objeto requerido' USING ERRCODE='22023'; END IF;
 IF s->>'esquema' IS NOT DISTINCT FROM 'vec.bolsa.aceptacion-nativa-ct.persona.consulta.v1' THEN
  RETURN vec_bolsa_llamamientos.consulta_aceptacion_nativa_ct_privada_v1(
   p_material,'persona',p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,
   p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 END IF;
 IF (SELECT count(*) FROM json_each(p_material::json))<>10
    OR (SELECT count(*) FROM jsonb_object_keys(s))<>10 OR NOT s ?& claves
    OR EXISTS (SELECT 1 FROM jsonb_each(s) x WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string')
    OR s->>'esquema' IS DISTINCT FROM 'vec.bolsa.persona-aceptacion-ct.consulta.v1'
    OR EXISTS (SELECT 1 FROM unnest(ARRAY['unidad_ref','categoria_ref','necesidad_ref','aceptacion_operacion_ref','apertura_operacion_ref','llamamiento_ref','propuesta_ref']) k
               WHERE octet_length(s->>k) NOT BETWEEN 1 AND 512 OR (s->>k)!~'^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
    OR EXISTS (SELECT 1 FROM unnest(ARRAY['aceptacion_registro_sha256','apertura_registro_sha256']) k
               WHERE (s->>k)!~'^[0-9a-f]{64}$' OR s->>k=repeat('0',64))
 THEN RAISE EXCEPTION 'B67: campos inválidos' USING ERRCODE='22023'; END IF;
 SELECT '{'||string_agg(to_json(k)::text||':'||to_json(s->>k)::text,',' ORDER BY n)||'}'
 INTO canon FROM unnest(claves) WITH ORDINALITY x(k,n);
 IF canon IS DISTINCT FROM p_material
 THEN RAISE EXCEPTION 'B67: material no canónico' USING ERRCODE='22023'; END IF;
 material_hash:=encode(sha256(convert_to(p_material,'UTF8')),'hex');
 h:=encode(sha256(convert_to('{"ambitos":{"unidad_ref":'||to_json(s->>'unidad_ref')::text||
    '},"atributos":{"material_sha256":"'||material_hash||'"}}','UTF8')),'hex');
 IF p_capacidad IS NULL OR p_decision IS NULL OR p_motivo IS NULL OR p_contexto IS NULL
    OR p_payload IS NULL OR p_sobre IS NULL OR p_evidencia IS NULL OR p_raiz IS NULL
    OR octet_length(p_capacidad) NOT BETWEEN 512 AND 32768
    OR octet_length(p_decision) NOT BETWEEN 1 AND 524288 OR octet_length(p_motivo) NOT BETWEEN 1 AND 65536
    OR octet_length(p_contexto) NOT BETWEEN 1 AND 262144 OR octet_length(p_payload) NOT BETWEEN 1 AND 1048576
    OR octet_length(p_sobre) NOT BETWEEN 1 AND 1048576 OR octet_length(p_evidencia) NOT BETWEEN 1 AND 262144
    OR octet_length(p_raiz)<>44 OR p_persona_version IS NULL OR p_perfil_version IS NULL
    OR p_persona_version NOT BETWEEN 1 AND 9007199254740991 OR p_perfil_version NOT BETWEEN 1 AND 9007199254740991
    OR p_persona_version<>trunc(p_persona_version) OR p_perfil_version<>trunc(p_perfil_version)
 THEN RAISE EXCEPTION 'B67: autorización denegada' USING ERRCODE='42501'; END IF;
 BEGIN c:=convert_from(p_capacidad,'UTF8')::jsonb; d:=convert_from(p_decision,'UTF8')::jsonb;
 EXCEPTION WHEN others THEN RAISE EXCEPTION 'B67: autorización denegada' USING ERRCODE='42501'; END;
 IF c->>'audiencia_consumo' IS DISTINCT FROM 'vec_bolsa_llamamientos.aceptacion_ct.persona.v1'
    OR c->>'operacion' IS DISTINCT FROM 'bolsa.aceptacion_ct.persona.consultar'
    OR c->>'efecto_ref' IS DISTINCT FROM s->>'aceptacion_operacion_ref'
    OR c->>'huella_efecto_sha256' IS DISTINCT FROM h
    OR c->>'huella_decision_sha256' IS DISTINCT FROM encode(sha256(p_decision),'hex')
    OR d->>'accion' IS DISTINCT FROM c->>'operacion' OR d->>'modulo_id' IS DISTINCT FROM 'bolsa'
    OR d->>'tipo_recurso' IS DISTINCT FROM 'persona_aceptacion_ct' OR d->>'finalidad' IS DISTINCT FROM 'preparar_incorporacion_ct'
    OR d->>'recurso_ref' IS DISTINCT FROM s->>'aceptacion_operacion_ref' OR d->>'contexto_recurso_huella_sha256' IS DISTINCT FROM h
    OR d->'campos_permitidos' IS DISTINCT FROM '["aceptacion","persona","vinculo"]'::jsonb
    OR d->'obligaciones' IS DISTINCT FROM '[]'::jsonb
 THEN RAISE EXCEPTION 'B67: autorización divergente' USING ERRCODE='42501'; END IF;
 SELECT * INTO STRICT consumo FROM vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(
  p_capacidad,p_decision,p_motivo,p_contexto,p_persona_version,p_perfil_version,p_payload,p_sobre,p_evidencia,p_raiz);
 IF consumo.consumo_nuevo IS NOT TRUE OR consumo.efecto_ref IS DISTINCT FROM s->>'aceptacion_operacion_ref'
    OR consumo.huella_efecto_sha256 IS DISTINCT FROM h
 THEN RAISE EXCEPTION 'B67: consumo divergente' USING ERRCODE='42501'; END IF;

 <<lectura>>
 BEGIN
  SELECT * INTO terminal FROM vec_bolsa_llamamientos.integracion_desarrollo
   WHERE operacion_ref=s->>'aceptacion_operacion_ref' AND tipo='aceptacion_rrhh';
  IF NOT FOUND THEN EXIT lectura; END IF;
  IF terminal.registro_huella_sha256 IS DISTINCT FROM encode(sha256(terminal.registro_canonico),'hex')
  THEN RAISE EXCEPTION 'B67: aceptación inconsistente' USING ERRCODE='P0675'; END IF;
  IF terminal.registro_huella_sha256 IS DISTINCT FROM s->>'aceptacion_registro_sha256'
     OR terminal.apertura_operacion_ref IS DISTINCT FROM s->>'apertura_operacion_ref'
  THEN EXIT lectura; END IF;
  SELECT * INTO apertura FROM vec_bolsa_llamamientos.integracion_desarrollo
   WHERE operacion_ref=terminal.apertura_operacion_ref AND tipo='propuesta';
  IF NOT FOUND THEN RAISE EXCEPTION 'B67: apertura inconsistente' USING ERRCODE='P0675'; END IF;
  IF apertura.registro_huella_sha256 IS DISTINCT FROM encode(sha256(apertura.registro_canonico),'hex')
  THEN RAISE EXCEPTION 'B67: apertura inconsistente' USING ERRCODE='P0675'; END IF;
  IF apertura.registro_huella_sha256 IS DISTINCT FROM s->>'apertura_registro_sha256' THEN EXIT lectura; END IF;
  t:=convert_from(terminal.registro_canonico,'UTF8')::jsonb;
  a:=convert_from(apertura.registro_canonico,'UTF8')::jsonb;
  IF t->>'esquema' IS DISTINCT FROM 'vec.bolsa.integracion-llamamientos-desarrollo.v1'
     OR a->>'esquema' IS DISTINCT FROM 'vec.bolsa.integracion-llamamientos-desarrollo.v1'
     OR t->>'operacion_ref' IS DISTINCT FROM terminal.operacion_ref OR a->>'operacion_ref' IS DISTINCT FROM apertura.operacion_ref
     OR t->>'tipo' IS DISTINCT FROM 'aceptacion_rrhh' OR a->>'tipo' IS DISTINCT FROM 'propuesta'
     OR t->>'estado_llamamiento' IS DISTINCT FROM 'aceptacion' OR a->>'estado_llamamiento' IS DISTINCT FROM 'abierto'
     OR t#>>'{resolucion,apertura_operacion_ref}' IS DISTINCT FROM apertura.operacion_ref
     OR terminal.necesidad_ref IS DISTINCT FROM apertura.necesidad_ref OR terminal.version_necesidad IS DISTINCT FROM apertura.version_necesidad
     OR terminal.orden_operacion_ref IS DISTINCT FROM apertura.orden_operacion_ref
     OR (t-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion']) IS DISTINCT FROM (a-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion'])
     OR t#>'{llamamiento,Version}' IS DISTINCT FROM '2'::jsonb OR a#>'{llamamiento,Version}' IS DISTINCT FROM '1'::jsonb
     OR ((t->'llamamiento')-'Version') IS DISTINCT FROM ((a->'llamamiento')-'Version')
  THEN RAISE EXCEPTION 'B67: aceptación desligada de la apertura original' USING ERRCODE='P0675'; END IF;
  IF t->>'unidad_ref' IS DISTINCT FROM s->>'unidad_ref' OR t->>'categoria_ref' IS DISTINCT FROM s->>'categoria_ref'
     OR t->>'necesidad_ref' IS DISTINCT FROM s->>'necesidad_ref'
     OR terminal.necesidad_ref IS DISTINCT FROM s->>'necesidad_ref'
     OR a#>>'{llamamiento,LlamamientoRef}' IS DISTINCT FROM s->>'llamamiento_ref'
     OR a#>>'{llamamiento,PropuestaRef}' IS DISTINCT FROM s->>'propuesta_ref'
  THEN EXIT lectura; END IF;
  SELECT * INTO llamamiento FROM vec_bolsa_llamamientos.llamamiento_integracion_desarrollo
   WHERE operacion_ref=apertura.operacion_ref;
  IF NOT FOUND OR llamamiento.llamamiento_ref IS DISTINCT FROM s->>'llamamiento_ref'
     OR llamamiento.propuesta_ref IS DISTINCT FROM s->>'propuesta_ref' OR llamamiento.necesidad_ref IS DISTINCT FROM terminal.necesidad_ref
     OR llamamiento.version IS DISTINCT FROM 1::bigint OR llamamiento.estado IS DISTINCT FROM 'abierto'
     OR llamamiento.datos_canonicos IS DISTINCT FROM a->'llamamiento'
  THEN RAISE EXCEPTION 'B67: llamamiento original inconsistente' USING ERRCODE='P0675'; END IF;
  i:=a->'instantanea'; p:=a->'propuesta';
  IF p->>'propuesta_ref' IS DISTINCT FROM s->>'propuesta_ref' OR p->>'necesidad_ref' IS DISTINCT FROM terminal.necesidad_ref
     OR p->>'instantanea_ref' IS DISTINCT FROM i->>'instantanea_ref' OR p->>'version_instantanea' IS DISTINCT FROM i->>'version'
     OR p->>'huella_instantanea_sha256' IS DISTINCT FROM i->>'huella_contenido_sha256'
     OR p->>'bolsa_ref' IS DISTINCT FROM i->>'bolsa_ref' OR p->>'version_bolsa' IS DISTINCT FROM i->>'version_bolsa'
     OR p->>'huella_bolsa_sha256' IS DISTINCT FROM i->>'huella_bolsa_sha256'
     OR llamamiento.bolsa_ref IS DISTINCT FROM i->>'bolsa_ref' OR a#>>'{llamamiento,BolsaRef}' IS DISTINCT FROM i->>'bolsa_ref'
     OR i->>'bolsa_ref' IS DISTINCT FROM a#>>'{fuente,datos,Bolsa,bolsa_ref}' OR i->>'version_bolsa' IS DISTINCT FROM a#>>'{fuente,datos,Bolsa,version}'
     OR i->'entradas' IS DISTINCT FROM a#>'{fuente,datos,Entradas}'
     OR a#>>'{fuente,datos,Necesidad,unidad_ref}' IS DISTINCT FROM s->>'unidad_ref'
     OR a#>>'{fuente,datos,Necesidad,categoria_ref}' IS DISTINCT FROM s->>'categoria_ref'
     OR p->>'participacion_seleccionada_ref' IS NULL
  THEN RAISE EXCEPTION 'B67: selección original inconsistente' USING ERRCODE='P0675'; END IF;
  aceptacion:=jsonb_build_object('operacion_ref',terminal.operacion_ref,'recibo_ref',terminal.recibo_ref,
   'registro_sha256',terminal.registro_huella_sha256,'apertura_operacion_ref',apertura.operacion_ref,
   'apertura_registro_sha256',apertura.registro_huella_sha256,'llamamiento_ref',llamamiento.llamamiento_ref);
  estado:='pendiente';
  -- La participación proviene de la propuesta de ESTA apertura. No procede
  -- de la selección raíz de CT ni de una referencia enviada por el cliente.
  SELECT count(*),min(v.candidato_ref) INTO total,candidato
  FROM vec_bolsa_llamamientos.vinculo_candidato v
  JOIN vec_bolsa_llamamientos.constitucion ct ON ct.acta_ref=v.acta_ref
   AND ct.instantanea_ref=v.instantanea_ref AND ct.version_instantanea=v.version_instantanea
  JOIN vec_bolsa_llamamientos.constitucion_entrada e ON e.instantanea_ref=v.instantanea_ref
   AND e.version_instantanea=v.version_instantanea AND e.participacion_ref=v.participacion_ref
  JOIN vec_bolsa_llamamientos.instantanea_orden_bolsa io ON io.instantanea_ref=ct.instantanea_ref
   AND io.version=ct.version_instantanea AND io.huella_instantanea_sha256=ct.huella_instantanea_sha256
  WHERE v.participacion_ref=p->>'participacion_seleccionada_ref'
    AND ct.categoria_ref=s->>'categoria_ref' AND ct.bolsa_ref=i->>'bolsa_ref' AND ct.version_bolsa::text=i->>'version_bolsa'
    AND ct.huella_bolsa_sha256=i->>'huella_bolsa_sha256'
    AND ct.instantanea_ref=i->>'instantanea_ref' AND ct.version_instantanea::text=i->>'version'
    -- B7 guarda sha256 del JSON completo; el dominio calcula
    -- huella_contenido_sha256 sin ese campo. Son huellas diferentes.
    AND ct.huella_instantanea_sha256=encode(sha256(convert_to(
        (convert_from(apertura.registro_canonico,'UTF8')::json->'instantanea')::text,'UTF8')),'hex')
    AND e.orden::text=p->>'orden_seleccionado'
    AND encode(sha256(io.instantanea_canonica),'hex')=io.huella_instantanea_sha256
    AND convert_from(io.instantanea_canonica,'UTF8')=
        (convert_from(apertura.registro_canonico,'UTF8')::json->'instantanea')::text;
  IF total<>1 THEN EXIT lectura; END IF;
  persona_ctx:=vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(candidato);
  IF jsonb_typeof(persona_ctx) IS DISTINCT FROM 'object' OR persona_ctx->>'estado' IS NULL
     OR persona_ctx->>'estado' NOT IN ('pendiente','acreditado')
     OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(persona_ctx) k) IS DISTINCT FROM ARRAY['estado','persona','vinculo']
  THEN RAISE EXCEPTION 'B67: respuesta de persona incompatible' USING ERRCODE='P0675'; END IF;
  IF persona_ctx->>'estado'='pendiente' THEN
   IF persona_ctx->'persona' IS DISTINCT FROM 'null'::jsonb OR persona_ctx->'vinculo' IS DISTINCT FROM 'null'::jsonb
   THEN RAISE EXCEPTION 'B67: respuesta pendiente incompatible' USING ERRCODE='P0675'; END IF;
   EXIT lectura;
  END IF;
  -- Solo devuelve las referencias minimizadas de CTX18; ningún candidato,
  -- sujeto, participación, HMAC, nombre o documento de identidad cruza a CT.
  persona:=persona_ctx->'persona'; vinculo:=persona_ctx->'vinculo';
  IF jsonb_typeof(persona) IS DISTINCT FROM 'object' OR jsonb_typeof(vinculo) IS DISTINCT FROM 'object'
     OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(persona) k) IS DISTINCT FROM ARRAY['ref','version']
     OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(vinculo) k) IS DISTINCT FROM ARRAY['poblacion','procedencia_ref','procedencia_sha256','procedencia_version','ref','version','vigente_hasta']
     OR vinculo->>'poblacion' IS NULL OR vinculo->>'poblacion' NOT IN ('interna','externa')
     OR jsonb_typeof(persona->'ref') IS DISTINCT FROM 'string' OR (persona->>'ref')!~'^per_[A-Za-z0-9_-]{22,128}$'
     OR jsonb_typeof(persona->'version') IS DISTINCT FROM 'number' OR (persona->>'version')!~'^[1-9][0-9]{0,18}$'
     OR EXISTS (SELECT 1 FROM unnest(ARRAY['ref','procedencia_ref','procedencia_sha256','poblacion','vigente_hasta']) k WHERE jsonb_typeof(vinculo->k) IS DISTINCT FROM 'string')
     OR EXISTS (SELECT 1 FROM unnest(ARRAY['version','procedencia_version']) k WHERE jsonb_typeof(vinculo->k) IS DISTINCT FROM 'number' OR (vinculo->>k)!~'^[1-9][0-9]{0,18}$')
     OR (vinculo->>'procedencia_sha256')!~'^[0-9a-f]{64}$' OR vinculo->>'procedencia_sha256'=repeat('0',64)
     OR (vinculo->>'vigente_hasta')!~'^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
  THEN RAISE EXCEPTION 'B67: proyección de persona incompatible' USING ERRCODE='P0675'; END IF;
  IF (persona->>'version')::numeric>9223372036854775807
     OR (vinculo->>'version')::numeric>9223372036854775807 OR (vinculo->>'procedencia_version')::numeric>9223372036854775807
     OR (vinculo->>'vigente_hasta')::timestamptz<=clock_timestamp()
  THEN RAISE EXCEPTION 'B67: procedencia de persona no vigente' USING ERRCODE='P0675'; END IF;
  estado:='acreditado';
 END lectura;
 RETURN jsonb_build_object('estado',estado,'aceptacion',aceptacion,'persona',persona,'vinculo',vinculo,
  'evidencia',jsonb_build_object('decision_ref',consumo.decision_ref,'consumo_huella_sha256',consumo.consumo_huella_sha256,
   'auditoria_ref',consumo.auditoria_ref,'consultada_en',to_char(consumo.consumida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')));
EXCEPTION WHEN serialization_failure OR deadlock_detected OR lock_not_available THEN
 RAISE EXCEPTION 'B67: consulta transitoria' USING ERRCODE='P0675';
END $_$;


-- Puerto entre propietarios: CT lo invoca tras su ACT V3; el propietario
-- Bolsa lo reutiliza internamente desde B115 tras el ACT propio. No hay
-- concesión a LOGIN ni lector global de Bolsa.
CREATE FUNCTION vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(p_selector jsonb)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog,pg_temp SET row_security=on SET timezone='UTC' SET lock_timeout='2s' AS $v$
DECLARE o record; l record; r record; total bigint; persona_ctx jsonb; anclaje jsonb;
 claves constant text[]:=ARRAY['organizacion_ref','expediente_ref','clave_ct','unidad_ref',
  'categoria_ref','necesidad_ref','bolsa_ref','llamamiento_ref','recibo_emision_ref','recibo_respuesta_ref'];
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario' OR session_user=current_user
    OR NOT (pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
       OR pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER'))
    OR pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
    OR pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
    OR current_setting('transaction_isolation')<>'serializable'
    OR current_setting('transaction_read_only')<>'off'
 THEN RAISE EXCEPTION 'B113: verificador CT denegado' USING ERRCODE='42501'; END IF;
 IF jsonb_typeof(p_selector) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM jsonb_object_keys(p_selector))<>10
    OR NOT p_selector ?& claves
    OR EXISTS(SELECT 1 FROM jsonb_each(p_selector) x
      WHERE jsonb_typeof(x.value) IS DISTINCT FROM 'string'
        OR octet_length(x.value#>>'{}') NOT BETWEEN 1 AND 512
        OR x.value#>>'{}' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]*$')
 THEN RAISE EXCEPTION 'B113: selector CT inválido' USING ERRCODE='22023'; END IF;
 SELECT * INTO o FROM vec_bolsa_llamamientos.origen_llamamiento_ct_v1
  WHERE expediente_ref=p_selector->>'expediente_ref' AND clave_ct=p_selector->>'clave_ct';
 IF NOT FOUND THEN RETURN jsonb_build_object('estado','no_encontrada'); END IF;
 IF o.organizacion_ref IS DISTINCT FROM p_selector->>'organizacion_ref'
    OR o.unidad_ref IS DISTINCT FROM p_selector->>'unidad_ref'
    OR o.categoria_ref IS DISTINCT FROM p_selector->>'categoria_ref'
    OR o.necesidad_ref IS DISTINCT FROM p_selector->>'necesidad_ref'
    OR o.bolsa_ref IS DISTINCT FROM p_selector->>'bolsa_ref'
    OR o.llamamiento_ref IS DISTINCT FROM p_selector->>'llamamiento_ref'
    OR o.recibo_ref IS DISTINCT FROM p_selector->>'recibo_emision_ref'
    OR o.origen_canonico->>'unidad_ref' IS DISTINCT FROM o.unidad_ref
    OR o.origen_huella_sha256 IS DISTINCT FROM encode(sha256(convert_to(o.origen_canonico::text,'UTF8')),'hex')
 THEN RETURN jsonb_build_object('estado','no_encontrada'); END IF;
 SELECT * INTO l FROM vec_bolsa_llamamientos.llamamiento_emitido
  WHERE llamamiento_ref=o.llamamiento_ref AND recibo_ref=o.recibo_ref AND bolsa_ref=o.bolsa_ref;
 IF NOT FOUND OR l.decision_ref IS DISTINCT FROM o.decision_ref
    OR l.emitido_en IS DISTINCT FROM o.emitido_en
    OR l.participaciones IS DISTINCT FROM jsonb_build_array(o.participacion_ref)
    OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.origen_llamamiento_ct_outbox_v1 ev
      WHERE ev.evento_ref=o.evento_ref AND ev.llamamiento_ref=o.llamamiento_ref)
 THEN RAISE EXCEPTION 'B113: emisión CT divergente' USING ERRCODE='P1135'; END IF;
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
 IF total<>1 THEN RAISE EXCEPTION 'B113: selección CT divergente' USING ERRCODE='P1135'; END IF;
 SELECT * INTO r FROM vec_bolsa_llamamientos.respuesta_portal_llamamiento
  WHERE recibo_ref=p_selector->>'recibo_respuesta_ref';
 IF NOT FOUND OR r.llamamiento_ref IS DISTINCT FROM o.llamamiento_ref
    OR r.bolsa_ref IS DISTINCT FROM o.bolsa_ref
    OR r.participacion_ref IS DISTINCT FROM o.participacion_ref
    OR r.candidato_ref IS DISTINCT FROM o.candidato_ref
    OR r.respuesta IS DISTINCT FROM 'acepta' OR r.modo IS DISTINCT FROM 'firme'
    OR r.respondida_en<l.emitido_en
    OR NOT EXISTS(SELECT 1 FROM vec_bolsa_llamamientos.contacto_participacion cp
      WHERE cp.llamamiento_ref=r.llamamiento_ref AND cp.bolsa_ref=r.bolsa_ref
      AND cp.participacion_ref=r.participacion_ref AND cp.instante=r.contacto_en)
 THEN RETURN jsonb_build_object('estado','no_encontrada'); END IF;
 anclaje:=jsonb_build_object('organizacion_ref',o.organizacion_ref,
  'expediente_ref',o.expediente_ref,'clave_ct',o.clave_ct,'unidad_ref',o.unidad_ref,
  'categoria_ref',o.categoria_ref,'necesidad_ref',o.necesidad_ref,'bolsa_ref',o.bolsa_ref,
  'llamamiento_ref',o.llamamiento_ref,'recibo_emision_ref',o.recibo_ref,
  'recibo_respuesta_ref',r.recibo_ref,'version_expediente',o.version_expediente,
  'acta_ref',o.acta_ref,'version_bolsa',o.version_bolsa,
  'huella_bolsa_sha256',o.huella_bolsa_sha256,'instantanea_ref',o.instantanea_ref,
  'version_instantanea',o.version_instantanea,
  'huella_instantanea_sha256',o.huella_instantanea_sha256,
  'politica_ref',o.politica_ref,'version_politica',o.version_politica,
  'huella_politica_sha256',o.huella_politica_sha256,
  'participacion_ref',o.participacion_ref,'candidato_ref',o.candidato_ref,
  'orden_seleccionado',o.orden_seleccionado,'respuesta_ref',r.respuesta_ref,
  'respondida_en',to_char(r.respondida_en AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"'));
 persona_ctx:=vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(o.candidato_ref);
 IF jsonb_typeof(persona_ctx) IS DISTINCT FROM 'object'
    OR (SELECT array_agg(k ORDER BY k) FROM jsonb_object_keys(persona_ctx) k)
      IS DISTINCT FROM ARRAY['estado','persona','vinculo']
    OR persona_ctx->>'estado' NOT IN ('pendiente','acreditado')
 THEN RAISE EXCEPTION 'B113: persona CT divergente' USING ERRCODE='P1135'; END IF;
 IF persona_ctx->>'estado'='pendiente' THEN
  IF persona_ctx->'persona' IS DISTINCT FROM 'null'::jsonb
     OR persona_ctx->'vinculo' IS DISTINCT FROM 'null'::jsonb
  THEN RAISE EXCEPTION 'B113: persona pendiente divergente' USING ERRCODE='P1135'; END IF;
  RETURN jsonb_build_object('estado','pendiente','anclaje',anclaje,
   'persona','null'::jsonb,'vinculo','null'::jsonb);
 END IF;
 IF jsonb_typeof(persona_ctx->'persona') IS DISTINCT FROM 'object'
    OR jsonb_typeof(persona_ctx->'vinculo') IS DISTINCT FROM 'object'
    OR persona_ctx#>>'{persona,ref}' !~ '^per_[A-Za-z0-9_-]{22,128}$'
    OR persona_ctx#>>'{vinculo,procedencia_sha256}' !~ '^[0-9a-f]{64}$'
    OR (persona_ctx#>>'{vinculo,vigente_hasta}')::timestamptz<=clock_timestamp()
 THEN RAISE EXCEPTION 'B113: procedencia de persona divergente' USING ERRCODE='P1135'; END IF;
 RETURN jsonb_build_object('estado','acreditado','anclaje',anclaje,
  'persona',persona_ctx->'persona','vinculo',persona_ctx->'vinculo');
END $v$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.verificar_aceptacion_nativa_ct_para_ct_v1(jsonb)
 TO vec_contratacion_temporal_propietario;
DO $acl$
DECLARE f regprocedure; privada regprocedure:=
 'vec_bolsa_llamamientos.consulta_aceptacion_nativa_ct_privada_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure;
BEGIN
 FOREACH f IN ARRAY ARRAY[
  'vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure,
  'vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure]
 LOOP
  IF (SELECT proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR NOT prosecdef
       OR proacl IS DISTINCT FROM ARRAY[
        'vec_bolsa_llamamientos_propietario=X/vec_bolsa_llamamientos_propietario',
        'vec_bolsa_llamamientos_ejecutor=X/vec_bolsa_llamamientos_propietario']::aclitem[]
       OR proconfig IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp',
         'row_security=on','TimeZone=UTC','lock_timeout=2s']::text[]
      FROM pg_proc WHERE oid=f) IS NOT FALSE
  THEN RAISE EXCEPTION 'B113: fachada heredada cambió ACL o configuración' USING ERRCODE='42501'; END IF;
 END LOOP;
 IF (SELECT proowner<>'vec_bolsa_llamamientos_propietario'::regrole OR NOT prosecdef
      FROM pg_proc WHERE oid=privada) IS NOT FALSE
    OR has_function_privilege('public',privada,'EXECUTE')
    OR has_function_privilege('vec_bolsa_llamamientos_ejecutor',privada,'EXECUTE')
    OR has_function_privilege('vec_contratacion_temporal_ejecutor',privada,'EXECUTE')
 THEN RAISE EXCEPTION 'B113: lector privado expuesto' USING ERRCODE='42501'; END IF;
END $acl$;
COMMIT;
