\set ON_ERROR_STOP on
-- B68: anclaje de la aceptación para CT, sin resolver persona ni crear efectos.
-- AD3-131 y B67 preceden a esta fachada. La huella de apertura procede
-- exclusivamente del registro original de Bolsa; consume y audita en la TX.
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path=pg_catalog;
SET LOCAL timezone='UTC';
SET LOCAL lock_timeout='5s';
SET LOCAL statement_timeout='30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000068',0));
DO $pre$
BEGIN
 IF current_user<>'vec_bolsa_llamamientos_propietario'
    OR to_regclass('vec_bolsa_llamamientos.integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.llamamiento_integracion_desarrollo') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.vinculo_candidato') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.constitucion_entrada') IS NULL
    OR to_regclass('vec_bolsa_llamamientos.instantanea_orden_bolsa') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_persona_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_autorizacion_atestada_v3.consumir_consulta_anclaje_aceptacion_ct_bolsa_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NULL
    OR to_regprocedure('vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') IS NOT NULL
 THEN RAISE EXCEPTION 'B68: preimagen incompatible' USING ERRCODE='55000'; END IF;
END $pre$;

CREATE FUNCTION vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(
 p_material text,p_capacidad bytea,p_decision bytea,p_motivo bytea,p_contexto bytea,
 p_persona_version numeric,p_perfil_version numeric,p_payload bytea,p_sobre bytea,p_evidencia bytea,p_raiz bytea)
RETURNS jsonb LANGUAGE plpgsql VOLATILE SECURITY DEFINER PARALLEL UNSAFE
SET search_path=pg_catalog, pg_temp SET row_security='on' SET timezone='UTC' SET lock_timeout='2s' AS $f$
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
END $f$;
REVOKE ALL ON FUNCTION vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) FROM PUBLIC;
DO $acl$
DECLARE f regprocedure:='vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure; x record;
BEGIN
 FOR x IN SELECT DISTINCT a.grantee FROM pg_proc p CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f AND a.grantee<>p.proowner LOOP
  EXECUTE format('REVOKE ALL ON FUNCTION %s FROM %s',f::text,CASE WHEN x.grantee=0 THEN 'PUBLIC' ELSE quote_ident(pg_get_userbyid(x.grantee)) END);
 END LOOP;
 GRANT EXECUTE ON FUNCTION vec_bolsa_llamamientos.consultar_anclaje_aceptacion_ct_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea) TO vec_bolsa_llamamientos_ejecutor;
 IF (SELECT proowner FROM pg_proc WHERE oid=f) IS DISTINCT FROM 'vec_bolsa_llamamientos_propietario'::regrole
    OR (SELECT prosecdef FROM pg_proc WHERE oid=f) IS NOT TRUE
    OR (SELECT array_agg(lower(split_part(k,'=',1))||'='||substr(k,strpos(k,'=')+1) ORDER BY n)
        FROM pg_proc p CROSS JOIN LATERAL unnest(p.proconfig) WITH ORDINALITY cfg(k,n) WHERE p.oid=f)
       IS DISTINCT FROM ARRAY['search_path=pg_catalog, pg_temp','row_security=on','timezone=UTC','lock_timeout=2s']
 THEN RAISE EXCEPTION 'B68: propietario o entorno incompatible' USING ERRCODE='55000'; END IF;
 FOR x IN SELECT a.grantee,a.privilege_type,a.is_grantable,p.proowner FROM pg_proc p
  CROSS JOIN LATERAL aclexplode(coalesce(p.proacl,acldefault('f',p.proowner))) a WHERE p.oid=f LOOP
  IF (x.grantee<>x.proowner AND x.grantee<>'vec_bolsa_llamamientos_ejecutor'::regrole::oid)
     OR x.privilege_type<>'EXECUTE' OR (x.grantee='vec_bolsa_llamamientos_ejecutor'::regrole::oid AND x.is_grantable)
  THEN RAISE EXCEPTION 'B68: ACL abierta' USING ERRCODE='55000'; END IF;
 END LOOP;
END $acl$;
COMMIT;
