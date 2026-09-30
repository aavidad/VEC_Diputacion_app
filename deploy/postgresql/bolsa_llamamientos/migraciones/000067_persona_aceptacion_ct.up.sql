\set ON_ERROR_STOP on
-- B67: persona de una aceptación, por lectura nominal y sin resolver HMAC
-- inversamente. AD3-128 y CTX18 preceden a esta fachada. No cambia avisos,
-- aceptaciones ni historia; consumo y auditoría centrales quedan en la TX.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000067',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.instantanea_orden_bolsa') IS NULL
    OR to_regprocedure('vec_contexto_actor_v1.acreditar_persona_candidato_incorporacion_v1(text)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_persona_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B67: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog, pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
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
END $f$;
DO $acl$
DECLARE f regprocedure:='vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
	 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
	    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
	    OR (SELECT array_agg(lower(split_part(k,'=',1))||'='||substr(k,strpos(k,'=')+1) ORDER BY n)
	        FROM pg_proc p CROSS JOIN LATERAL unnest(p.proconfig) WITH ORDINALITY cfg(k,n) WHERE p.oid=f)
	       IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=UTC','lock_timeout=2s']
	 THEN RAISE EXCEPTION 'B67: propietario o entorno incompatible' USING ERRCODE='55000'; END IF;
END $acl$;
COMMIT;
