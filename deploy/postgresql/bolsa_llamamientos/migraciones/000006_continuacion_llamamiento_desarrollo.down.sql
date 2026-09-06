\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_bolsa_llamamientos_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_advisory_xact_lock(hashtextextended('vec_bolsa_llamamientos:migracion:000006',0));
LOCK TABLE vec_bolsa_llamamientos.integracion_desarrollo,
    vec_bolsa_llamamientos.llamamiento_integracion_desarrollo,
    vec_bolsa_llamamientos.auditoria_integracion_desarrollo,
    vec_bolsa_llamamientos.outbox_integracion_desarrollo IN ACCESS EXCLUSIVE MODE;
DO $preservar$
BEGIN
    IF EXISTS (SELECT 1 FROM vec_bolsa_llamamientos.integracion_desarrollo
        WHERE terminal_anterior_ref IS NOT NULL OR continuacion_intencion_ref IS NOT NULL
           OR (convert_from(registro_canonico,'UTF8')::jsonb->'propuesta')?'continuacion') THEN
        RAISE EXCEPTION 'reversión denegada: historia de continuación Bolsa' USING ERRCODE='55000';
    END IF;
END
$preservar$;
DO $funcion$
DECLARE v_def text; v_acl aclitem[]; v_cambio record;
BEGIN
    SELECT pg_get_functiondef(p.oid),p.proacl INTO STRICT v_def,v_acl FROM pg_proc p
     WHERE p.oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure
       AND p.proowner='vec_bolsa_llamamientos_propietario'::regrole AND p.prosecdef;
    FOR v_cambio IN SELECT * FROM (VALUES
      ('variables',$antes$ v_auditoria_bytes bytea; v_evento_bytes bytea; v_secuencia bigint; v_anterior_hash text;$antes$,$despues$ v_auditoria_bytes bytea; v_evento_bytes bytea; v_secuencia bigint; v_anterior_hash text;
 v_cont jsonb; v_desde integer:=0; v_terminal_anterior record; v_propuesta_anterior record;
 v_terminal_json jsonb; v_propuesta_json jsonb; v_clave text;$despues$),
      ('continuacion',$antes$ v_resolucion:=r->'resolucion';$antes$,$despues$ v_resolucion:=r->'resolucion';
 v_cont:=p->'continuacion';
 IF p?'continuacion' AND jsonb_typeof(v_cont) IS DISTINCT FROM 'object' THEN
  RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='continuación no canónica';
 END IF;$despues$),
      ('accion',$antes$  WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrar' ELSE 'bolsa.llamamiento.abrir' END;$antes$,$despues$  WHEN 'renuncia_rrhh' THEN 'bolsa.llamamiento.renuncia_rrhh.registrar'
  ELSE CASE WHEN v_cont IS NOT NULL THEN 'bolsa.llamamiento.siguiente.abrir'
       ELSE 'bolsa.llamamiento.abrir' END END;$despues$),
      ('antecedente',$antes$  IF (convert_from(v_orden.registro_canonico,'UTF8')::jsonb->'instantanea'=i AND$antes$,$despues$  IF v_cont IS NOT NULL THEN
   IF (SELECT count(*) FROM jsonb_object_keys(v_cont))<>6 OR
      (SELECT count(*) FROM json_each(convert_from(p_registro,'UTF8')::json->'propuesta'->'continuacion'))<>6 OR
      NOT (v_cont ?& ARRAY['terminal_operacion_ref','terminal_sha256','propuesta_ref','propuesta_sha256','orden_anterior','intencion_ref']) THEN
    RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='campos de continuación Bolsa inválidos';
   END IF;
   FOREACH v_clave IN ARRAY ARRAY['terminal_operacion_ref','propuesta_ref','intencion_ref'] LOOP
    IF jsonb_typeof(v_cont->v_clave) IS DISTINCT FROM 'string'
       OR (v_cont->>v_clave)!~'^[A-Za-z0-9][A-Za-z0-9:._/-]{0,191}$' THEN
     RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='referencia de continuación Bolsa inválida';
    END IF;
   END LOOP;
   FOREACH v_clave IN ARRAY ARRAY['terminal_sha256','propuesta_sha256'] LOOP
    IF jsonb_typeof(v_cont->v_clave) IS DISTINCT FROM 'string'
       OR (v_cont->>v_clave)!~'^[0-9a-f]{64}$' OR v_cont->>v_clave=repeat('0',64) THEN
     RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='huella de continuación Bolsa inválida';
    END IF;
   END LOOP;
   IF jsonb_typeof(v_cont->'orden_anterior') IS DISTINCT FROM 'number'
      OR (v_cont->>'orden_anterior')!~'^[1-9][0-9]{0,2}$' THEN
    RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='posición anterior inválida';
   END IF;
   v_desde:=(v_cont->>'orden_anterior')::integer;
   SELECT * INTO STRICT v_terminal_anterior FROM vec_bolsa_llamamientos.integracion_desarrollo
    WHERE operacion_ref=v_cont->>'terminal_operacion_ref' AND tipo='renuncia_rrhh' FOR SHARE;
   SELECT * INTO STRICT v_propuesta_anterior FROM vec_bolsa_llamamientos.integracion_desarrollo
    WHERE operacion_ref=v_terminal_anterior.apertura_operacion_ref AND tipo='propuesta' FOR SHARE;
   v_terminal_json:=convert_from(v_terminal_anterior.registro_canonico,'UTF8')::jsonb;
   v_propuesta_json:=convert_from(v_propuesta_anterior.registro_canonico,'UTF8')::jsonb;
   -- Mismos bytes fuente/orden y propuesta original, renuncia realmente
   -- persistida. No convierte al renunciante en "no elegible".
   IF v_desde>=jsonb_array_length(i->'entradas') OR
      v_terminal_anterior.registro_huella_sha256 IS DISTINCT FROM v_cont->>'terminal_sha256' OR
      encode(sha256(v_terminal_anterior.registro_canonico),'hex') IS DISTINCT FROM v_cont->>'terminal_sha256' OR
      v_terminal_anterior.orden_operacion_ref IS DISTINCT FROM v_orden.operacion_ref OR
      v_propuesta_anterior.orden_operacion_ref IS DISTINCT FROM v_orden.operacion_ref OR
      v_terminal_json->>'estado_llamamiento' IS DISTINCT FROM 'renuncia' OR
      v_terminal_json->'llamamiento'->'Version' IS DISTINCT FROM '2'::jsonb OR
      v_propuesta_json->>'estado_llamamiento' IS DISTINCT FROM 'abierto' OR
      v_propuesta_json->'llamamiento'->'Version' IS DISTINCT FROM '1'::jsonb OR
      v_terminal_json->'propuesta' IS DISTINCT FROM v_propuesta_json->'propuesta' OR
      (v_terminal_json-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion']) IS DISTINCT FROM
       (v_propuesta_json-ARRAY['operacion_ref','tipo','estado_llamamiento','llamamiento','resolucion']) OR
      ((v_terminal_json->'llamamiento')-'Version') IS DISTINCT FROM ((v_propuesta_json->'llamamiento')-'Version') OR
      v_propuesta_json->'propuesta'->>'propuesta_ref' IS DISTINCT FROM v_cont->>'propuesta_ref' OR
      v_propuesta_json->'propuesta'->>'huella_contenido_sha256' IS DISTINCT FROM v_cont->>'propuesta_sha256' OR
      v_propuesta_json->'propuesta'->'orden_seleccionado' IS DISTINCT FROM v_cont->'orden_anterior' OR
      (v_propuesta_json-ARRAY['operacion_ref','propuesta','llamamiento']) IS DISTINCT FROM
       (r-ARRAY['operacion_ref','propuesta','llamamiento']) OR
      r->>'operacion_ref'=v_terminal_anterior.operacion_ref OR
      r->>'operacion_ref'=v_propuesta_anterior.operacion_ref OR
      l->>'LlamamientoRef'=v_propuesta_json->'llamamiento'->>'LlamamientoRef' OR
      p->>'propuesta_ref'=v_cont->>'propuesta_ref' OR
      (p->>'generada_en')::timestamptz<v_terminal_anterior.confirmada_en THEN
    RAISE EXCEPTION USING ERRCODE='22023',MESSAGE='continuación desligada de renuncia original';
   END IF;
  END IF;
  IF (convert_from(v_orden.registro_canonico,'UTF8')::jsonb->'instantanea'=i AND$despues$),
      ('longitud',$antes$   jsonb_array_length(p->'evaluaciones') BETWEEN 1 AND jsonb_array_length(i->'entradas') AND
   (p->>'orden_seleccionado')::integer=jsonb_array_length(p->'evaluaciones')) IS NOT TRUE THEN$antes$,$despues$   jsonb_array_length(p->'evaluaciones') BETWEEN 1 AND jsonb_array_length(i->'entradas')-v_desde AND
   (p->>'orden_seleccionado')::integer=v_desde+jsonb_array_length(p->'evaluaciones')) IS NOT TRUE THEN$despues$),
      ('entrada',$antes$   e:=p->'evaluaciones'->v_indice; entrada:=i->'entradas'->v_indice;$antes$,$despues$   e:=p->'evaluaciones'->v_indice; entrada:=i->'entradas'->(v_indice+v_desde);$despues$),
      ('orden',$antes$   IF ((e->>'orden')::integer=v_indice+1 AND (entrada->>'orden')::integer=v_indice+1 AND$antes$,$despues$   IF ((e->>'orden')::integer=v_indice+v_desde+1 AND (entrada->>'orden')::integer=v_indice+v_desde+1 AND$despues$),
      ('unicidad',$antes$  WHERE o.tipo=r->>'tipo' AND o.necesidad_ref=r->>'necesidad_ref' AND o.version_necesidad=(r->>'version_necesidad')::bigint$antes$,$despues$  WHERE o.tipo=r->>'tipo' AND o.necesidad_ref=r->>'necesidad_ref' AND o.version_necesidad=(r->>'version_necesidad')::bigint
   AND (r->>'tipo'='orden'
    OR (r->>'tipo'='propuesta' AND v_cont IS NULL AND o.terminal_anterior_ref IS NULL)
    OR (r->>'tipo'='propuesta' AND v_cont IS NOT NULL AND
     (o.terminal_anterior_ref=v_cont->>'terminal_operacion_ref' OR o.continuacion_intencion_ref=v_cont->>'intencion_ref')))$despues$),
      ('columnas',$antes$  registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en,apertura_operacion_ref$antes$,$despues$  registro_huella_sha256,contexto_huella_sha256,decision_ref,recibo_ref,confirmada_en,apertura_operacion_ref,
  terminal_anterior_ref,continuacion_intencion_ref$despues$),
      ('valores',$antes$  v_resolucion->>'apertura_operacion_ref');$antes$,$despues$  v_resolucion->>'apertura_operacion_ref',
  CASE WHEN r->>'tipo'='propuesta' THEN v_cont->>'terminal_operacion_ref' END,
  CASE WHEN r->>'tipo'='propuesta' THEN v_cont->>'intencion_ref' END);$despues$)
    ) AS cambios(nombre,anterior,nuevo) LOOP
        IF length(v_def)-length(replace(v_def,v_cambio.nuevo,''))<>length(v_cambio.nuevo) THEN
            RAISE EXCEPTION 'función Bolsa incompatible: %',v_cambio.nombre USING ERRCODE='55000';
        END IF;
        v_def:=replace(v_def,v_cambio.nuevo,v_cambio.anterior);
    END LOOP;
    EXECUTE v_def;
    IF (SELECT proacl FROM pg_proc WHERE oid='vec_bolsa_llamamientos.guardar_integracion_desarrollo_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)'::regprocedure) IS DISTINCT FROM v_acl THEN
        RAISE EXCEPTION 'continuación alteró permisos Bolsa' USING ERRCODE='55000';
    END IF;
END
$funcion$;
DROP INDEX vec_bolsa_llamamientos.integracion_primera_propuesta_unica;
DROP INDEX vec_bolsa_llamamientos.integracion_orden_necesidad_unica;
ALTER TABLE vec_bolsa_llamamientos.integracion_desarrollo
    DROP CONSTRAINT integracion_continuacion_ligada,
    DROP CONSTRAINT integracion_continuacion_intencion_unica,
    DROP CONSTRAINT integracion_continuacion_terminal_unico,
    DROP COLUMN continuacion_intencion_ref,
    DROP COLUMN terminal_anterior_ref,
    ADD UNIQUE (tipo,necesidad_ref,version_necesidad);
COMMIT;
