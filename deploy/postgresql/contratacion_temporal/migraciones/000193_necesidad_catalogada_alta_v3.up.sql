\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000193:necesidad-alta-v3', 0));

DO $prevalidacion$
BEGIN
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.confirmar_alta_atestada_v3(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.materializar_version_inicial_v1(text,numeric,bytea,text,numeric,text,text,text,timestamp with time zone)') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.periodo_previsto_estructural_v1(jsonb)') IS NULL
    OR pg_catalog.to_regclass('vec_contratacion_temporal.expediente_alta_version') IS NULL
    OR pg_catalog.to_regprocedure('vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb)') IS NOT NULL THEN
   RAISE EXCEPTION 'CT193: preimagen incompatible' USING ERRCODE='55000';
 END IF;
END $prevalidacion$;

-- Sólo acepta la instantánea de bytes incluida en el efecto atestado.
-- Su procedencia la acredita la aplicación confiable y el digest del recurso V3.
CREATE FUNCTION vec_contratacion_temporal.necesidad_alta_valida_v3(s jsonb)
RETURNS boolean LANGUAGE plpgsql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
DECLARE
 n jsonb;
 c jsonb;
 causa jsonb;
 raw bytea;
 b64 text;
 v_clave text;
 v_valor jsonb;
 v_grupo jsonb;
 v_fecha_fin text;
 v_inicio date;
 v_fin date;
 v_maximo integer;
 v_primer_destino date;
 v_siguiente_mes date;
 v_limite_exclusivo date;
 v_programa_fin date;
BEGIN
 IF pg_catalog.jsonb_typeof(s) IS DISTINCT FROM 'object'
    OR NOT (s ? 'necesidad') THEN RETURN false; END IF;
 n := s->'necesidad';
 IF pg_catalog.jsonb_typeof(n) IS DISTINCT FROM 'object'
    OR (SELECT count(*) FROM pg_catalog.jsonb_object_keys(n)) <> 9
    OR NOT (n ?& ARRAY['esquema','catalogo_ref','catalogo_version',
                         'catalogo_huella_sha256','causa_clave','periodo',
                         'jornada_minutos','campos','catalogo_instantanea'])
    OR n->>'esquema' IS DISTINCT FROM 'vec.ct.necesidad_alta.v1'
    OR pg_catalog.jsonb_typeof(n->'catalogo_ref') IS DISTINCT FROM 'string'
    OR n->>'catalogo_ref' !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR pg_catalog.jsonb_typeof(n->'catalogo_version') IS DISTINCT FROM 'number'
    OR n->>'catalogo_version' !~ '^[1-9][0-9]{0,15}$'
    OR (n->>'catalogo_version')::numeric > 9007199254740991::numeric
    OR pg_catalog.jsonb_typeof(n->'catalogo_huella_sha256') IS DISTINCT FROM 'string'
    OR n->>'catalogo_huella_sha256' !~ '^[0-9a-f]{64}$'
    OR pg_catalog.jsonb_typeof(n->'causa_clave') IS DISTINCT FROM 'string'
    OR n->>'causa_clave' !~ '^[a-z][a-z0-9._-]{1,79}$'
    OR n->>'causa_clave' IS DISTINCT FROM s->>'motivo_clave'
    OR n->'periodo' IS DISTINCT FROM s->'periodo'
    OR vec_contratacion_temporal.periodo_previsto_estructural_v1(n->'periodo') IS NOT TRUE
    OR pg_catalog.jsonb_typeof(n->'jornada_minutos') IS DISTINCT FROM 'number'
    OR n->>'jornada_minutos' !~ '^[1-9][0-9]{0,4}$'
    OR (n->>'jornada_minutos')::integer NOT BETWEEN 1 AND 10080
    OR pg_catalog.jsonb_typeof(n->'campos') IS DISTINCT FROM 'object'
    OR pg_catalog.jsonb_typeof(n->'catalogo_instantanea') IS DISTINCT FROM 'string'
    OR pg_catalog.octet_length(n->>'catalogo_instantanea') NOT BETWEEN 4 AND 10924
    OR n->>'catalogo_instantanea' !~ '^[A-Za-z0-9+/]+={0,2}$' THEN
   RETURN false;
 END IF;
 FOR v_clave,v_valor IN SELECT key,value FROM pg_catalog.jsonb_each(n->'campos') LOOP
   IF v_clave !~ '^[a-z][a-z0-9_]{0,79}$'
      OR pg_catalog.jsonb_typeof(v_valor) IS DISTINCT FROM 'string' THEN
      RETURN false;
   END IF;
 END LOOP;
 IF (SELECT count(*) FROM pg_catalog.jsonb_object_keys(n->'campos')) > 32 THEN
   RETURN false;
 END IF;
 b64 := n->>'catalogo_instantanea';
 raw := pg_catalog.decode(b64,'base64');
 IF pg_catalog.octet_length(raw) NOT BETWEEN 1 AND 8192
    OR pg_catalog.replace(pg_catalog.encode(raw,'base64'), E'\n','') IS DISTINCT FROM b64
    OR pg_catalog.encode(pg_catalog.sha256(raw),'hex') IS DISTINCT FROM
       n->>'catalogo_huella_sha256' THEN
   RETURN false;
 END IF;
 c := pg_catalog.convert_from(raw,'UTF8')::jsonb;
 IF pg_catalog.jsonb_typeof(c) IS DISTINCT FROM 'object'
    OR c->>'esquema' IS DISTINCT FROM 'vec.ct.necesidades_alta.v1'
    OR c->>'referencia' IS DISTINCT FROM n->>'catalogo_ref'
    OR pg_catalog.jsonb_typeof(c->'version') IS DISTINCT FROM 'number'
    OR c->>'version' !~ '^[1-9][0-9]{0,15}$'
    OR c->'version' IS DISTINCT FROM n->'catalogo_version'
    OR pg_catalog.jsonb_typeof(c->'jornada_referencia_minutos') IS DISTINCT FROM 'number'
    OR c->>'jornada_referencia_minutos' !~ '^[1-9][0-9]{0,4}$'
    OR (c->>'jornada_referencia_minutos')::integer NOT BETWEEN 1 AND 10080
    OR pg_catalog.jsonb_typeof(c->'causas') IS DISTINCT FROM 'array' THEN
   RETURN false;
 END IF;
 IF (SELECT count(*) FROM pg_catalog.jsonb_array_elements(c->'causas') AS e(v)
      WHERE e.v->>'clave'=n->>'causa_clave') <> 1 THEN
   RETURN false;
 END IF;
 SELECT e.v INTO STRICT causa
   FROM pg_catalog.jsonb_array_elements(c->'causas') AS e(v)
  WHERE e.v->>'clave'=n->>'causa_clave';
 v_fecha_fin:=causa->>'fecha_fin';
 IF pg_catalog.jsonb_typeof(causa) IS DISTINCT FROM 'object'
    OR v_fecha_fin IS NULL
    OR v_fecha_fin NOT IN ('obligatoria','opcional','no_aplica')
    OR pg_catalog.jsonb_typeof(causa->'maximo_meses') IS DISTINCT FROM 'number'
    OR causa->>'maximo_meses' !~ '^[1-9][0-9]{0,2}$'
    OR (causa->>'maximo_meses')::integer NOT BETWEEN 1 AND 120
    OR pg_catalog.jsonb_typeof(causa->'campos_permitidos') IS DISTINCT FROM 'array'
    OR pg_catalog.jsonb_typeof(causa->'campos_obligatorios') IS DISTINCT FROM 'array'
    OR (causa ? 'uno_de' AND
        pg_catalog.jsonb_typeof(causa->'uno_de') IS DISTINCT FROM 'array')
    OR (v_fecha_fin='obligatoria' AND causa ? 'causa_fin')
    OR (v_fecha_fin<>'obligatoria' AND
        (pg_catalog.jsonb_typeof(causa->'causa_fin') IS DISTINCT FROM 'string'
         OR causa->>'causa_fin' !~ '^[a-z][a-z0-9._-]{1,79}$')) THEN
   RETURN false;
 END IF;
 IF EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(
       causa->'campos_permitidos') AS e(v)
       WHERE pg_catalog.jsonb_typeof(e.v)<>'string')
    OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(
       causa->'campos_obligatorios') AS e(v)
       WHERE pg_catalog.jsonb_typeof(e.v)<>'string') THEN
   RETURN false;
 END IF;
 FOR v_clave,v_valor IN SELECT key,value FROM pg_catalog.jsonb_each(n->'campos') LOOP
   IF NOT EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements_text(
          causa->'campos_permitidos') AS permitido(clave)
          WHERE permitido.clave=v_clave)
      OR v_valor#>>'{}'=''
      OR pg_catalog.btrim(v_valor#>>'{}') IS DISTINCT FROM v_valor#>>'{}' THEN
     RETURN false;
   END IF;
 END LOOP;
 -- El número viaja como texto canónico dentro de campos, igual que en Go.
 -- Una instantánea anterior puede no declararlo; la obligatoriedad la fija
 -- exclusivamente el catálogo sellado de esa alta.
 IF n->'campos' ? 'numero_personas' THEN
   v_clave:=n#>>'{campos,numero_personas}';
   IF v_clave !~ '^[1-9][0-9]{0,9}$' THEN RETURN false; END IF;
   IF v_clave::numeric > 4294967295::numeric THEN RETURN false; END IF;
 END IF;
 FOR v_clave IN SELECT permitido.clave FROM pg_catalog.jsonb_array_elements_text(
       causa->'campos_obligatorios') AS permitido(clave) LOOP
   IF NOT (n->'campos' ? v_clave)
      OR n->'campos'->>v_clave='' THEN RETURN false; END IF;
 END LOOP;
 FOR v_grupo IN SELECT e.v FROM pg_catalog.jsonb_array_elements(
      coalesce(causa->'uno_de','[]'::jsonb)) AS e(v) LOOP
   IF pg_catalog.jsonb_typeof(v_grupo) IS DISTINCT FROM 'array'
      OR pg_catalog.jsonb_array_length(v_grupo) NOT BETWEEN 2 AND 4
      OR EXISTS (SELECT 1 FROM pg_catalog.jsonb_array_elements(v_grupo) AS e(v)
                  WHERE pg_catalog.jsonb_typeof(e.v)<>'string')
      OR (SELECT count(*) FROM pg_catalog.jsonb_array_elements_text(v_grupo) AS e(clave)
           WHERE n->'campos' ? e.clave) <> 1 THEN
     RETURN false;
   END IF;
 END LOOP;
 IF n->'periodo' ? 'politica_fin' THEN
   IF n#>>'{periodo,politica_fin,regla_ref}' IS DISTINCT FROM causa->>'regla_ref'
      OR n#>'{periodo,politica_fin,catalogo_version}' IS DISTINCT FROM n->'catalogo_version'
      OR n#>>'{periodo,politica_fin,catalogo_huella_sha256}' IS DISTINCT FROM
         n->>'catalogo_huella_sha256'
      OR n#>>'{periodo,politica_fin,fecha_fin}' IS DISTINCT FROM v_fecha_fin
      OR n#>>'{periodo,politica_fin,causa_fin}' IS DISTINCT FROM causa->>'causa_fin' THEN
     RETURN false;
   END IF;
 END IF;
 v_inicio:=(n#>>'{periodo,inicio}')::date;
 IF pg_catalog.to_char(v_inicio,'YYYY-MM-DD') IS DISTINCT FROM
    n#>>'{periodo,inicio}' THEN RETURN false; END IF;
 IF n->'periodo' ? 'fin' THEN
   IF v_fecha_fin='no_aplica' THEN RETURN false; END IF;
   v_fin:=(n#>>'{periodo,fin}')::date;
   IF pg_catalog.to_char(v_fin,'YYYY-MM-DD') IS DISTINCT FROM n#>>'{periodo,fin}'
      OR v_fin<v_inicio THEN RETURN false; END IF;
   v_maximo:=(causa->>'maximo_meses')::integer;
   v_primer_destino:=(pg_catalog.date_trunc('month',v_inicio::timestamp)::date
                      +pg_catalog.make_interval(months=>v_maximo))::date;
   v_siguiente_mes:=(v_primer_destino+INTERVAL '1 month')::date;
   IF EXTRACT(day FROM v_inicio)>
      EXTRACT(day FROM v_siguiente_mes-1) THEN
     v_limite_exclusivo:=v_siguiente_mes;
   ELSE
     v_limite_exclusivo:=v_primer_destino+
       (EXTRACT(day FROM v_inicio)::integer-1);
   END IF;
   IF v_fin>=v_limite_exclusivo THEN RETURN false; END IF;
 ELSE
   IF v_fecha_fin='obligatoria'
      OR n#>>'{periodo,causa_fin}' IS DISTINCT FROM causa->>'causa_fin' THEN
     RETURN false;
   END IF;
 END IF;
 IF n->'campos' ? 'programa_fin' THEN
   IF n#>>'{campos,programa_fin}' !~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}$' THEN
     RETURN false;
   END IF;
   v_programa_fin:=(n#>>'{campos,programa_fin}')::date;
   IF pg_catalog.to_char(v_programa_fin,'YYYY-MM-DD') IS DISTINCT FROM
        n#>>'{campos,programa_fin}'
      OR v_programa_fin<v_inicio
      OR (v_fin IS NOT NULL AND v_programa_fin<v_fin) THEN
     RETURN false;
   END IF;
 END IF;
 RETURN true;
EXCEPTION WHEN OTHERS THEN
 RETURN false;
END $funcion$;

-- E3 reutiliza la serialización ya instalada para los diez campos E2.
CREATE FUNCTION vec_contratacion_temporal.reconstruir_periodo_necesidad_v3(p jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
 SELECT '{"inicio":' || vec_contratacion_temporal.texto_json_go_v1(p->>'inicio') ||
   (CASE WHEN p ? 'fin' THEN ',"fin":' || vec_contratacion_temporal.texto_json_go_v1(p->>'fin')
         ELSE ',"causa_fin":' || vec_contratacion_temporal.texto_json_go_v1(p->>'causa_fin') END) ||
   (CASE WHEN p ? 'politica_fin' THEN
     ',"politica_fin":{"regla_ref":' || vec_contratacion_temporal.texto_json_go_v1(p#>>'{politica_fin,regla_ref}') ||
     ',"catalogo_version":' || (p#>'{politica_fin,catalogo_version}')::text ||
     ',"catalogo_huella_sha256":' || vec_contratacion_temporal.texto_json_go_v1(p#>>'{politica_fin,catalogo_huella_sha256}') ||
     ',"fecha_fin":' || vec_contratacion_temporal.texto_json_go_v1(p#>>'{politica_fin,fecha_fin}') ||
     (CASE WHEN p#>'{politica_fin,causa_fin}' IS NOT NULL THEN
       ',"causa_fin":' || vec_contratacion_temporal.texto_json_go_v1(p#>>'{politica_fin,causa_fin}')
      ELSE '' END) || '}' ELSE '' END) || '}'
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(s jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
 SELECT pg_catalog.left(vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s),-1) ||
  ',"necesidad":{"esquema":' || vec_contratacion_temporal.texto_json_go_v1(s#>>'{necesidad,esquema}') ||
  ',"catalogo_ref":' || vec_contratacion_temporal.texto_json_go_v1(s#>>'{necesidad,catalogo_ref}') ||
  ',"catalogo_version":' || (s#>'{necesidad,catalogo_version}')::text ||
  ',"catalogo_huella_sha256":' || vec_contratacion_temporal.texto_json_go_v1(s#>>'{necesidad,catalogo_huella_sha256}') ||
  ',"causa_clave":' || vec_contratacion_temporal.texto_json_go_v1(s#>>'{necesidad,causa_clave}') ||
  ',"periodo":' || vec_contratacion_temporal.reconstruir_periodo_necesidad_v3(s#>'{necesidad,periodo}') ||
  ',"jornada_minutos":' || (s#>'{necesidad,jornada_minutos}')::text ||
  ',"campos":{' || coalesce((SELECT pg_catalog.string_agg(
       vec_contratacion_temporal.texto_json_go_v1(e.key) || ':' ||
       vec_contratacion_temporal.texto_json_go_v1(e.value),
       ',' ORDER BY e.key COLLATE "C")
       FROM pg_catalog.jsonb_each_text(s#>'{necesidad,campos}') e), '') || '}' ||
  ',"catalogo_instantanea":' || vec_contratacion_temporal.texto_json_go_v1(s#>>'{necesidad,catalogo_instantanea}') || '}}'
$funcion$;

-- La preimagen PG18 queda fijada antes de cualquier sustitución. Se retiene
-- toda la metadata de pg_proc salvo prosrc para comprobarla tras los DDL.
DO $preimagen_ct193$
DECLARE item record; f oid; metadatos jsonb := '{}'::jsonb;
 actual_huella text; actual_propietario pg_catalog.regrole;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.reconstruir_efecto_alta_v2(jsonb)','6143987ae2c1129902c3921c013eb23344a64009d448705010066d676f680149'),
  ('vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)','ea874f747173da65b05e62e8f4c78ed217799f58f44cd92f7fd094b6887d674c'),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)','ee55c83862ecf750f396e7e4eae7a71c9a6f00cdafda81035b19d56c49924472'),
  ('vec_contratacion_temporal.materializar_version_inicial_v1(text,numeric,bytea,text,numeric,text,text,text,timestamp with time zone)','0b1872b16ad62786eaccea72d4981fd76839adee57a9093b2dd6bed8b1108bd0')
 ) AS esperado(firma,huella) LOOP
   f:=pg_catalog.to_regprocedure(item.firma);
   IF f IS NULL THEN
     RAISE EXCEPTION 'CT193: preimagen ausente firma=% esperado=% actual=ausente',
       item.firma,item.huella USING ERRCODE='55000';
   END IF;
   SELECT p.proowner::pg_catalog.regrole INTO actual_propietario
     FROM pg_catalog.pg_proc p WHERE p.oid=f;
   IF actual_propietario IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::pg_catalog.regrole THEN
     RAISE EXCEPTION 'CT193: propietario firma=% esperado=% actual=%',
       item.firma,'vec_contratacion_temporal_propietario',actual_propietario USING ERRCODE='55000';
   END IF;
   actual_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
     pg_catalog.pg_get_functiondef(f),'UTF8')),'hex');
   IF actual_huella IS DISTINCT FROM item.huella THEN
     RAISE EXCEPTION 'CT193: preimagen firma=% esperado=% actual=%',
       item.firma,item.huella,actual_huella USING ERRCODE='55000';
   END IF;
   metadatos:=metadatos || pg_catalog.jsonb_build_object(
     item.firma,(SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f));
 END LOOP;
 IF pg_catalog.to_regprocedure('vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb)') IS NOT NULL THEN
   RAISE EXCEPTION 'CT193: reconstrucción E3 ya existente' USING ERRCODE='55000';
 END IF;
 PERFORM pg_catalog.set_config('vec_ct193.pre_meta',metadatos::text,true);
END $preimagen_ct193$;

-- Definición E3 explícita: posición y bytes E2 intactos salvo la solicitud V3.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.reconstruir_efecto_alta_v3(a jsonb)
 RETURNS bytea
 LANGUAGE sql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
    SELECT pg_catalog.convert_to(
      '{"esquema":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'esquema') ||
      ',"reserva_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'reserva_ref') ||
      ',"expediente_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'expediente_ref') ||
      ',"numero_visible":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'numero_visible') ||
      ',"recibo_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'recibo_ref') ||
      ',"organizacion_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'organizacion_ref') ||
      ',"actor_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'actor_ref') ||
      ',"perfil_ref":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'perfil_ref') ||
      ',"version":' || (a -> 'version')::text ||
      ',"flujo":{"definicion_ref":' ||
        vec_contratacion_temporal.texto_json_go_v1(a #>> '{flujo,definicion_ref}') ||
      ',"version":' || (a #> '{flujo,version}')::text ||
      ',"huella_sha256":' ||
        vec_contratacion_temporal.texto_json_go_v1(a #>> '{flujo,huella_sha256}') || '}' ||
      ',"fase_actual":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'fase_actual') ||
      ',"estado_actual":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'estado_actual') ||
      ',"solicitud":' ||
        vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(
            a -> 'solicitud'
        ) ||
      ',"creado_en":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'creado_en') ||
      ',"actualizado_en":' || vec_contratacion_temporal.texto_json_go_v1(a ->> 'actualizado_en') ||
      ',"actuacion":{"secuencia":' || (a #> '{actuacion,secuencia}')::text ||
      ',"version_expediente":' || (a #> '{actuacion,version_expediente}')::text ||
      ',"accion_clave":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,accion_clave}') ||
      ',"actor_ref":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,actor_ref}') ||
      ',"unidad_ref":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,unidad_ref}') ||
      ',"recibo_ref":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,recibo_ref}') ||
      ',"realizada_en":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,realizada_en}') ||
      ',"fase_origen":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,fase_origen}') ||
      ',"fase_destino":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,fase_destino}') ||
      ',"estado_origen":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,estado_origen}') ||
      ',"estado_destino":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,estado_destino}') ||
      ',"observaciones":' || vec_contratacion_temporal.texto_json_go_v1(a #>> '{actuacion,observaciones}') ||
      ',"documentos_ref":' ||
        vec_contratacion_temporal.lista_textos_json_v1(
            a #> '{actuacion,documentos_ref}'
        ) || '}}',
      'UTF8'
    )
$function$
;

CREATE FUNCTION vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(s jsonb)
RETURNS text LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
 SELECT CASE WHEN s ? 'necesidad' THEN
   vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(s)
 ELSE vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(s) END
$funcion$;

CREATE FUNCTION vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(a jsonb)
RETURNS bytea LANGUAGE sql IMMUTABLE STRICT SET search_path=pg_catalog
AS $funcion$
 SELECT CASE a->>'esquema'
  WHEN 'vec.contratacion-temporal.efecto-alta.v2' THEN
   vec_contratacion_temporal.reconstruir_efecto_alta_v2(a)
  WHEN 'vec.contratacion-temporal.efecto-alta.v3' THEN
   vec_contratacion_temporal.reconstruir_efecto_alta_v3(a)
 END
$funcion$;

-- Postimágenes explícitas capturadas de la misma preimagen validada.
CREATE OR REPLACE FUNCTION vec_contratacion_temporal.reconciliar_agregado_alta_v1(p_alta_canonica bytea, p_ambito_hmac text, p_huella_peticion_hmac text, p_decision_ref text, p_efecto_ref text, p_huella_efecto_sha256 text, p_consumo_huella_sha256 text)
 RETURNS TABLE(expediente_ref text, numero_visible text, version numeric, recibo_ref text, auditoria_ref text, evento_ref text, confirmada_en timestamp with time zone, recibo_huella_sha256 text)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE
    a jsonb;
    i record;
    ra record;
    rv record;
    e record;
    ev record;
    ac record;
    au record;
    o record;
    m record;
    c record;
    v_raiz text;
    v_huella_alias text;
    v_confirmacion_ref text;
    v_auditoria_ref text;
    v_evento_ref text;
    v_huella_alta text;
    v_huella_solicitud text;
    v_huella_actuacion text;
    v_huella_auditoria text;
    v_payload bytea;
    v_huella_payload text;
    v_huella_outbox text;
    v_huella_recibo text;
    v_huella_agregado text;
    v_incompleto boolean := false;
BEGIN
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'reconciliación de alta inválida';
    END;
    SELECT aa.ambito_raiz_hmac, ah.alias_hmac
      INTO v_raiz, v_huella_alias
      FROM vec_contratacion_temporal.alias_ambito_alta aa
      JOIN vec_contratacion_temporal.alias_huella_alta ah
        ON ah.ambito_raiz_hmac = aa.ambito_raiz_hmac
       AND ah.generacion = aa.generacion
     WHERE aa.alias_hmac = p_ambito_hmac;
    IF NOT FOUND
       OR v_huella_alias IS DISTINCT FROM p_huella_peticion_hmac THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    v_huella_alta := pg_catalog.encode(
        pg_catalog.sha256(p_alta_canonica), 'hex'
    );
    v_huella_solicitud := pg_catalog.encode(pg_catalog.sha256(
        pg_catalog.convert_to(
            vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(
                a -> 'solicitud'
            ), 'UTF8'
        )
    ), 'hex');
    v_huella_actuacion := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            (a #> '{actuacion}')::text
        )
    ), 'hex');
    v_confirmacion_ref := 'cnf_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(p_decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                p_consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    v_auditoria_ref := 'aud_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(p_decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            )
        ), 'hex'), 1, 32
    );
    v_evento_ref := 'evt_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                p_consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    SELECT * INTO i
      FROM vec_contratacion_temporal.identidad_reserva_alta ii
     WHERE ii.ambito_hmac = v_raiz;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO ra
      FROM vec_contratacion_temporal.reserva_alta_actual raa
     WHERE raa.ambito_hmac = v_raiz;
    IF NOT FOUND THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    SELECT * INTO rv
      FROM vec_contratacion_temporal.reserva_alta_version rav
     WHERE rav.ambito_hmac = v_raiz
       AND rav.revision = ra.revision;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO e
      FROM vec_contratacion_temporal.expediente_alta ea
     WHERE ea.expediente_ref = a ->> 'expediente_ref';
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO ev
      FROM vec_contratacion_temporal.expediente_alta_version eav
     WHERE eav.expediente_ref = a ->> 'expediente_ref'
       AND eav.version = 1;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO ac
      FROM vec_contratacion_temporal.actuacion_alta aa
     WHERE aa.expediente_ref = a ->> 'expediente_ref'
       AND aa.secuencia = 1;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO au
      FROM vec_contratacion_temporal.auditoria_alta aua
     WHERE aua.expediente_ref = a ->> 'expediente_ref';
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO o
      FROM vec_contratacion_temporal.outbox_alta oa
     WHERE oa.expediente_ref = a ->> 'expediente_ref';
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO m
      FROM vec_contratacion_temporal.confirmacion_agregado_alta caa
     WHERE caa.confirmacion_ref = v_confirmacion_ref;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    SELECT * INTO c
      FROM vec_contratacion_temporal.control_cadenas_alta cca
     WHERE cca.control_id;
    IF NOT FOUND THEN v_incompleto := true; END IF;
    IF v_incompleto THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(au.secuencia::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(au.anterior_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(p_decision_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            p_consumo_huella_sha256
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_alta)
    ), 'hex');
    v_payload := pg_catalog.convert_to(
        '{"esquema":"vec.contratacion-temporal.evento-expediente-registrado.v1"' ||
        ',"evento_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_evento_ref) ||
        ',"expediente_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'expediente_ref'
          ) ||
        ',"version":1,"ocurrido_en":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              vec_contratacion_temporal.instante_utc_v1(m.confirmada_en)
          ) || '}',
        'UTF8'
    );
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload), 'hex'
    );
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(o.secuencia::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(o.anterior_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_payload)
    ), 'hex');
    v_huella_recibo := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'numero_visible'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1('1') ||
        vec_contratacion_temporal.encuadrar_texto_v1(a ->> 'recibo_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(m.confirmada_en)
        )
    ), 'hex');
    v_huella_agregado :=
      vec_contratacion_temporal.huella_prueba_agregado_alta_v1(
        VARIADIC ARRAY[
          'vec.contratacion-temporal.confirmacion-agregado-alta.v1',
          v_confirmacion_ref, v_raiz, rv.revision::text,
          a ->> 'reserva_ref', a ->> 'expediente_ref',
          a ->> 'numero_visible', a ->> 'recibo_ref',
          p_decision_ref, p_efecto_ref, p_huella_efecto_sha256,
          p_consumo_huella_sha256, '1', v_huella_alta, '1',
          v_huella_actuacion, v_auditoria_ref, au.secuencia::text,
          au.anterior_sha256, v_huella_auditoria, v_evento_ref,
          o.secuencia::text, v_huella_payload, o.anterior_sha256,
          v_huella_outbox,
          vec_contratacion_temporal.instante_utc_v1(m.confirmada_en),
          v_huella_recibo
        ]
      );
    IF i.reserva_ref IS DISTINCT FROM a ->> 'reserva_ref'
       OR i.expediente_ref IS DISTINCT FROM a ->> 'expediente_ref'
       OR i.numero_visible IS DISTINCT FROM a ->> 'numero_visible'
       OR i.recibo_ref IS DISTINCT FROM a ->> 'recibo_ref'
       OR i.organizacion_ref IS DISTINCT FROM a ->> 'organizacion_ref'
       OR i.actor_ref IS DISTINCT FROM a ->> 'actor_ref'
       OR i.perfil_ref IS DISTINCT FROM a ->> 'perfil_ref'
       OR i.creada_en IS DISTINCT FROM
          (a #>> '{actuacion,realizada_en}')::timestamptz
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.alias_ambito_alta ia
             JOIN vec_contratacion_temporal.alias_huella_alta ih
               ON ih.ambito_raiz_hmac = ia.ambito_raiz_hmac
              AND ih.generacion = ia.generacion
            WHERE ia.alias_hmac = v_raiz
              AND ih.alias_hmac = i.huella_peticion_hmac
       )
       OR ra.revision IS DISTINCT FROM rv.revision
       OR rv.estado IS DISTINCT FROM 'confirmada'
       OR rv.version_expediente IS DISTINCT FROM 1
       OR rv.auditoria_ref IS DISTINCT FROM v_auditoria_ref
       OR rv.evento_ref IS DISTINCT FROM v_evento_ref
       OR rv.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR rv.confirmada_en IS DISTINCT FROM m.confirmada_en
       OR rv.registrada_en IS DISTINCT FROM m.confirmada_en
       OR e.reserva_ref IS DISTINCT FROM a ->> 'reserva_ref'
       OR e.numero_visible IS DISTINCT FROM a ->> 'numero_visible'
       OR e.organizacion_ref IS DISTINCT FROM a ->> 'organizacion_ref'
       OR e.actor_ref IS DISTINCT FROM a ->> 'actor_ref'
       OR e.perfil_ref IS DISTINCT FROM a ->> 'perfil_ref'
       OR e.decision_ref IS DISTINCT FROM p_decision_ref
       OR e.efecto_ref IS DISTINCT FROM p_efecto_ref
       OR e.huella_efecto_sha256 IS DISTINCT FROM p_huella_efecto_sha256
       OR e.creada_en IS DISTINCT FROM (a ->> 'creado_en')::timestamptz
       OR e.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR ev.alta_canonica IS DISTINCT FROM p_alta_canonica
       OR ev.huella_alta_sha256 IS DISTINCT FROM v_huella_alta
       OR ev.flujo_ref IS DISTINCT FROM a #>> '{flujo,definicion_ref}'
       OR ev.flujo_version IS DISTINCT FROM
          (a #>> '{flujo,version}')::numeric
       OR ev.flujo_huella_sha256 IS DISTINCT FROM
          a #>> '{flujo,huella_sha256}'
       OR ev.fase_clave IS DISTINCT FROM a ->> 'fase_actual'
       OR ev.estado IS DISTINCT FROM a ->> 'estado_actual'
       OR ev.solicitud_huella_sha256 IS DISTINCT FROM v_huella_solicitud
       OR ev.registrada_en IS DISTINCT FROM m.confirmada_en
       OR ev.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR ac.version_expediente IS DISTINCT FROM 1
       OR ac.accion_clave IS DISTINCT FROM a #>> '{actuacion,accion_clave}'
       OR ac.actor_ref IS DISTINCT FROM a #>> '{actuacion,actor_ref}'
       OR ac.unidad_ref IS DISTINCT FROM a #>> '{actuacion,unidad_ref}'
       OR ac.recibo_ref IS DISTINCT FROM a #>> '{actuacion,recibo_ref}'
       OR ac.fase_destino IS DISTINCT FROM a #>> '{actuacion,fase_destino}'
       OR ac.estado_destino IS DISTINCT FROM
          a #>> '{actuacion,estado_destino}'
       OR ac.realizada_en IS DISTINCT FROM
          (a #>> '{actuacion,realizada_en}')::timestamptz
       OR ac.huella_sha256 IS DISTINCT FROM v_huella_actuacion
       OR ac.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR au.auditoria_ref IS DISTINCT FROM v_auditoria_ref
       OR au.decision_ref IS DISTINCT FROM p_decision_ref
       OR au.consumo_huella_sha256 IS DISTINCT FROM
          p_consumo_huella_sha256
       OR au.anterior_sha256 IS NULL
       OR au.huella_sha256 IS DISTINCT FROM v_huella_auditoria
       OR au.registrada_en IS DISTINCT FROM m.confirmada_en
       OR au.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR o.evento_ref IS DISTINCT FROM v_evento_ref
       OR o.tipo_evento IS DISTINCT FROM
          'contratacion_temporal.expediente.registrado.v1'
       OR o.payload_canonico IS DISTINCT FROM v_payload
       OR o.payload_huella_sha256 IS DISTINCT FROM v_huella_payload
       OR o.anterior_sha256 IS NULL
       OR o.huella_sha256 IS DISTINCT FROM v_huella_outbox
       OR o.registrada_en IS DISTINCT FROM m.confirmada_en
       OR o.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR m.confirmacion_ref IS DISTINCT FROM v_confirmacion_ref
       OR m.agregado_huella_sha256 IS DISTINCT FROM v_huella_agregado
       OR m.ambito_hmac IS DISTINCT FROM v_raiz
       OR m.reserva_revision IS DISTINCT FROM rv.revision
       OR m.reserva_ref IS DISTINCT FROM a ->> 'reserva_ref'
       OR m.expediente_ref IS DISTINCT FROM a ->> 'expediente_ref'
       OR m.numero_visible IS DISTINCT FROM a ->> 'numero_visible'
       OR m.recibo_ref IS DISTINCT FROM a ->> 'recibo_ref'
       OR m.decision_ref IS DISTINCT FROM p_decision_ref
       OR m.efecto_ref IS DISTINCT FROM p_efecto_ref
       OR m.huella_efecto_sha256 IS DISTINCT FROM p_huella_efecto_sha256
       OR m.consumo_huella_sha256 IS DISTINCT FROM
          p_consumo_huella_sha256
       OR m.version_expediente IS DISTINCT FROM 1
       OR m.huella_alta_sha256 IS DISTINCT FROM v_huella_alta
       OR m.actuacion_secuencia IS DISTINCT FROM 1
       OR m.actuacion_huella_sha256 IS DISTINCT FROM v_huella_actuacion
       OR m.auditoria_ref IS DISTINCT FROM v_auditoria_ref
       OR m.auditoria_secuencia IS DISTINCT FROM au.secuencia
       OR m.auditoria_anterior_sha256 IS DISTINCT FROM au.anterior_sha256
       OR m.auditoria_huella_sha256 IS DISTINCT FROM v_huella_auditoria
       OR m.evento_ref IS DISTINCT FROM v_evento_ref
       OR m.outbox_secuencia IS DISTINCT FROM o.secuencia
       OR m.payload_huella_sha256 IS DISTINCT FROM v_huella_payload
       OR m.outbox_anterior_sha256 IS DISTINCT FROM o.anterior_sha256
       OR m.outbox_huella_sha256 IS DISTINCT FROM v_huella_outbox
       OR m.recibo_huella_sha256 IS DISTINCT FROM v_huella_recibo
       OR m.creada_en IS DISTINCT FROM m.confirmada_en THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    IF au.secuencia > c.secuencia_auditoria
       OR o.secuencia > c.secuencia_outbox
       OR (
           au.secuencia = 1
           AND au.anterior_sha256 <> pg_catalog.repeat('0', 64)
       )
       OR (
           au.secuencia > 1
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.auditoria_alta p
                WHERE p.secuencia = au.secuencia - 1
                  AND p.huella_sha256 = au.anterior_sha256
           )
       )
       OR (
           au.secuencia = c.secuencia_auditoria
           AND au.huella_sha256 <> c.cabeza_auditoria_sha256
       )
       OR (
           au.secuencia < c.secuencia_auditoria
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.auditoria_alta n
                WHERE n.secuencia = au.secuencia + 1
                  AND n.anterior_sha256 = au.huella_sha256
           )
       )
       OR NOT EXISTS (
           SELECT 1
             FROM vec_contratacion_temporal.auditoria_alta h
             JOIN vec_contratacion_temporal.expediente_alta he
               USING (expediente_ref)
             JOIN vec_contratacion_temporal.expediente_alta_version hv
               ON hv.expediente_ref = he.expediente_ref
              AND hv.version = 1
            WHERE h.secuencia = c.secuencia_auditoria
              AND h.huella_sha256 = c.cabeza_auditoria_sha256
              AND h.huella_sha256 = pg_catalog.encode(pg_catalog.sha256(
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.secuencia::text
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.anterior_sha256
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.auditoria_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.expediente_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.decision_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.consumo_huella_sha256
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      hv.huella_alta_sha256
                  )
              ), 'hex')
       )
       OR (
           o.secuencia = 1
           AND o.anterior_sha256 <> pg_catalog.repeat('0', 64)
       )
       OR (
           o.secuencia > 1
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.outbox_alta p
                WHERE p.secuencia = o.secuencia - 1
                  AND p.huella_sha256 = o.anterior_sha256
           )
       )
       OR (
           o.secuencia = c.secuencia_outbox
           AND o.huella_sha256 <> c.cabeza_outbox_sha256
       )
       OR (
           o.secuencia < c.secuencia_outbox
           AND NOT EXISTS (
               SELECT 1 FROM vec_contratacion_temporal.outbox_alta n
                WHERE n.secuencia = o.secuencia + 1
                  AND n.anterior_sha256 = o.huella_sha256
           )
       )
       OR NOT EXISTS (
           SELECT 1 FROM vec_contratacion_temporal.outbox_alta h
            WHERE h.secuencia = c.secuencia_outbox
              AND h.huella_sha256 = c.cabeza_outbox_sha256
              AND h.payload_huella_sha256 = pg_catalog.encode(
                  pg_catalog.sha256(h.payload_canonico), 'hex'
              )
              AND h.huella_sha256 = pg_catalog.encode(pg_catalog.sha256(
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.secuencia::text
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.anterior_sha256
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.evento_ref
                  ) ||
                  vec_contratacion_temporal.encuadrar_texto_v1(
                      h.payload_huella_sha256
                  )
              ), 'hex')
       ) THEN
        RAISE EXCEPTION USING
            ERRCODE = '55000',
            MESSAGE = 'integridad del agregado de alta no acreditada';
    END IF;
    RETURN QUERY SELECT
        m.expediente_ref, m.numero_visible, m.version_expediente,
        m.recibo_ref, m.auditoria_ref, m.evento_ref,
        m.confirmada_en, m.recibo_huella_sha256;
END
$function$
;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.confirmar_alta_atestada_v1(p_capacidad_canonica bytea, p_decision_canonica bytea, p_motivo_canonico bytea, p_contexto_actor_canonico bytea, p_persona_version numeric, p_perfil_version numeric, p_payload_vec_ad_3 bytea, p_sobre_cose_sign1 bytea, p_evidencia_verificacion bytea, p_raiz_publica_spki bytea, p_alta_canonica bytea, p_sellos_hmac_canonicos bytea)
 RETURNS TABLE(expediente_ref text, numero_visible text, version numeric, recibo_ref text, auditoria_ref text, evento_ref text, confirmada_en timestamp with time zone, recibo_huella_sha256 text)
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET lock_timeout TO '2s'
AS $function$
DECLARE
    a jsonb;
    s jsonb;
    d jsonb;
    v_consumo record;
    v_identidad record;
    v_par jsonb;
    v_pares jsonb;
    v_generaciones integer[];
    v_generaciones_politica integer[];
    v_aliases text[];
    v_raices text[];
    v_raiz text;
    v_activo_ambito text;
    v_activo_huella text;
    v_huella_alta text;
    v_huella_contexto_recurso text;
    v_contexto_recurso bytea;
    v_ahora timestamptz(6);
    v_revision bigint;
    v_auditoria_ref text;
    v_evento_ref text;
    v_anterior_auditoria text;
    v_anterior_outbox text;
    v_secuencia_auditoria numeric(20, 0);
    v_secuencia_outbox numeric(20, 0);
    v_huella_auditoria text;
    v_huella_outbox text;
    v_payload_outbox bytea;
    v_huella_payload text;
    v_confirmacion_ref text;
    v_huella_actuacion text;
    v_huella_agregado text;
    v_recibo_huella text;
    v_statement numeric;
    v_idle numeric;
BEGIN
    IF pg_catalog.current_setting('transaction_isolation') <> 'serializable'
       OR pg_catalog.current_setting('transaction_read_only') <> 'off'
       OR pg_catalog.current_setting('TimeZone') <> 'UTC'
       OR session_user = current_user
       OR NOT pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_ejecutor', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_migrador', 'MEMBER')
       OR pg_catalog.pg_has_role(
           session_user, 'vec_contratacion_temporal_propietario', 'MEMBER') THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'confirmación de alta rechazada';
    END IF;
    SELECT setting::numeric INTO v_statement
      FROM pg_catalog.pg_settings
     WHERE name = 'statement_timeout' AND unit = 'ms';
    SELECT setting::numeric INTO v_idle
      FROM pg_catalog.pg_settings
     WHERE name = 'idle_in_transaction_session_timeout' AND unit = 'ms';
    IF v_statement IS NULL OR v_statement NOT BETWEEN 1 AND 15000
       OR v_idle IS NULL OR v_idle NOT BETWEEN 1 AND 20000
       OR pg_catalog.octet_length(p_alta_canonica) NOT BETWEEN 256 AND 32768
       OR pg_catalog.octet_length(p_sellos_hmac_canonicos)
          NOT BETWEEN 256 AND 8192 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
    END IF;
    BEGIN
        a := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
        s := pg_catalog.convert_from(p_sellos_hmac_canonicos, 'UTF8')::jsonb;
        d := pg_catalog.convert_from(p_decision_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN data_exception OR invalid_text_representation
          OR character_not_in_repertoire OR untranslatable_character THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'entrada de alta inválida';
    END;
    IF pg_catalog.jsonb_typeof(a) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a)) <> 16
       OR NOT (a ?& ARRAY[
           'esquema', 'reserva_ref', 'expediente_ref', 'numero_visible',
           'recibo_ref', 'organizacion_ref', 'actor_ref', 'perfil_ref',
           'version', 'flujo', 'fase_actual', 'estado_actual',
           'solicitud', 'creado_en', 'actualizado_en', 'actuacion'
       ])
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a) AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'version', 'flujo', 'solicitud', 'actuacion') THEN false
            ELSE pg_catalog.jsonb_typeof(a -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a -> 'version') <> 'number'
       OR pg_catalog.jsonb_typeof(a -> 'flujo') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'flujo')) <> 3
       OR NOT ((a -> 'flujo') ?& ARRAY[
           'definicion_ref', 'version', 'huella_sha256'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{flujo,definicion_ref}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{flujo,version}') <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{flujo,huella_sha256}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a -> 'solicitud') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'solicitud')) <>
       (CASE WHEN a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
             THEN 11 ELSE 10 END)
       OR NOT ((a -> 'solicitud') ?& ARRAY[
           'centro_ref', 'contacto_ref', 'categoria_ref',
           'grupo_subgrupo', 'motivo_clave', 'detalle', 'periodo', 'rc',
           'documentos_adjuntos', 'observaciones'
       ])
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a -> 'solicitud') AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'periodo', 'rc', 'documentos_adjuntos', 'necesidad') THEN false
            ELSE pg_catalog.jsonb_typeof(
                (a -> 'solicitud') -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,periodo}') <> 'object'
       OR NOT vec_contratacion_temporal.periodo_previsto_estructural_v1(a #> '{solicitud,periodo}')
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc}') <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,rc}')) <> 5
       OR NOT ((a #> '{solicitud,rc}') ?& ARRAY[
           'existe', 'numero', 'fecha', 'importe', 'documento_ref'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,existe}')
          <> 'boolean'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,numero}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,fecha}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,documento_ref}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,importe}')
          <> 'object'
       OR (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(
               a #> '{solicitud,rc,importe}')) <> 2
       OR NOT ((a #> '{solicitud,rc,importe}') ?&
               ARRAY['centimos', 'moneda'])
       OR pg_catalog.jsonb_typeof(
           a #> '{solicitud,rc,importe,centimos}') <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{solicitud,rc,importe,moneda}')
          <> 'string'
       OR pg_catalog.jsonb_typeof(
           a #> '{solicitud,documentos_adjuntos}') <> 'array'
       OR pg_catalog.jsonb_array_length(
           a #> '{solicitud,documentos_adjuntos}') > 100
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(
               a #> '{solicitud,documentos_adjuntos}') AS e(valor)
           WHERE pg_catalog.jsonb_typeof(e.valor) <> 'string'
              OR e.valor #>> '{}' !~
                 '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR pg_catalog.jsonb_typeof(a -> 'actuacion') <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(a -> 'actuacion')) <> 13
       OR NOT ((a -> 'actuacion') ?& ARRAY[
           'secuencia', 'version_expediente', 'accion_clave', 'actor_ref',
           'unidad_ref', 'recibo_ref', 'realizada_en', 'fase_origen',
           'fase_destino', 'estado_origen', 'estado_destino',
           'observaciones', 'documentos_ref'
       ])
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,secuencia}')
          <> 'number'
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,version_expediente}')
          <> 'number'
       OR EXISTS (
           SELECT 1
             FROM pg_catalog.jsonb_object_keys(a -> 'actuacion') AS k(clave)
            WHERE CASE WHEN k.clave IN (
                'secuencia', 'version_expediente', 'documentos_ref') THEN false
            ELSE pg_catalog.jsonb_typeof(
                (a -> 'actuacion') -> k.clave) <> 'string'
            END)
       OR pg_catalog.jsonb_typeof(a #> '{actuacion,documentos_ref}')
          <> 'array'
       OR pg_catalog.jsonb_array_length(
           a #> '{actuacion,documentos_ref}') > 100
       OR EXISTS (
           SELECT 1 FROM pg_catalog.jsonb_array_elements(
               a #> '{actuacion,documentos_ref}') AS e(valor)
           WHERE pg_catalog.jsonb_typeof(e.valor) <> 'string'
              OR e.valor #>> '{}' !~
                 '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(a)
          IS DISTINCT FROM p_alta_canonica
       OR a ->> 'esquema' NOT IN (
          'vec.contratacion-temporal.efecto-alta.v2',
          'vec.contratacion-temporal.efecto-alta.v3')
       OR (a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
           AND vec_contratacion_temporal.necesidad_alta_valida_v3(a -> 'solicitud') IS NOT TRUE)
       OR (a ->> 'version') !~ '^[1-9][0-9]{0,15}$'
       OR (a ->> 'version')::numeric <> 1
       OR (a #>> '{flujo,version}') !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{flujo,version}')::numeric >
          9007199254740991::numeric
       OR (a #>> '{actuacion,secuencia}') !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{actuacion,secuencia}')::numeric <> 1
       OR (a #>> '{actuacion,version_expediente}')
          !~ '^[1-9][0-9]{0,15}$'
       OR (a #>> '{actuacion,version_expediente}')::numeric <> 1
       OR (a #>> '{solicitud,rc,importe,centimos}')
          !~ '^(0|[1-9][0-9]{0,15})$'
       OR (a #>> '{solicitud,rc,importe,centimos}')::numeric >
          9007199254740991::numeric
       OR a ->> 'estado_actual' <> 'en_curso'
       OR a ->> 'numero_visible' !~
          '^[0-9]{4}/[A-Za-z0-9._-]{1,40}$'
       OR EXISTS (
           SELECT 1 FROM pg_catalog.unnest(ARRAY[
               a ->> 'reserva_ref', a ->> 'expediente_ref',
               a ->> 'recibo_ref', a ->> 'organizacion_ref',
               a #>> '{solicitud,centro_ref}',
               a #>> '{solicitud,categoria_ref}',
               a ->> 'actor_ref', a ->> 'perfil_ref',
               a #>> '{flujo,definicion_ref}', a ->> 'fase_actual',
               a #>> '{actuacion,accion_clave}',
               a #>> '{actuacion,unidad_ref}'
           ]) AS r(valor)
           WHERE r.valor !~
             '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$')
       OR a #>> '{flujo,huella_sha256}' !~ '^[0-9a-f]{64}$'
       OR a ->> 'creado_en' !~
          '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}[.][0-9]{6}Z$'
       OR a ->> 'actualizado_en' <> a ->> 'creado_en'
       OR a #>> '{actuacion,realizada_en}' <> a ->> 'creado_en'
       OR a #>> '{actuacion,actor_ref}' <> a ->> 'actor_ref'
       OR a #>> '{actuacion,recibo_ref}' <> a ->> 'recibo_ref'
       OR a #>> '{actuacion,fase_origen}' <> ''
       OR a #>> '{actuacion,fase_destino}' <> a ->> 'fase_actual'
       OR a #>> '{actuacion,estado_origen}' <> 'pendiente'
       OR a #>> '{actuacion,estado_destino}' <> a ->> 'estado_actual'
       OR a ->> 'actor_ref' <> d ->> 'principal_id'
       OR a ->> 'perfil_ref' <> d ->> 'perfil_activo_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'proyección de alta inválida';
    END IF;
    IF pg_catalog.jsonb_typeof(s) <> 'object'
       OR (SELECT pg_catalog.count(*)
             FROM pg_catalog.jsonb_object_keys(s)) <> 3
       OR NOT (s ?& ARRAY['esquema', 'activo', 'retenidos'])
       OR s ->> 'esquema' <>
          'vec.contratacion-temporal.sellos-hmac.v1'
       OR pg_catalog.jsonb_typeof(s -> 'activo') <> 'object'
       OR pg_catalog.jsonb_typeof(s -> 'retenidos') <> 'array'
       OR pg_catalog.jsonb_array_length(s -> 'retenidos') > 3
       OR vec_contratacion_temporal.reconstruir_sellos_hmac_v1(s)
          IS DISTINCT FROM p_sellos_hmac_canonicos THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    v_pares := pg_catalog.jsonb_build_array(s -> 'activo') ||
               (s -> 'retenidos');
    IF EXISTS (
        SELECT 1
          FROM pg_catalog.jsonb_array_elements(v_pares) AS e(valor)
         WHERE pg_catalog.jsonb_typeof(e.valor) <> 'object'
            OR (SELECT pg_catalog.count(*)
                  FROM pg_catalog.jsonb_object_keys(e.valor)) <> 3
            OR NOT (e.valor ?& ARRAY[
                'generacion', 'ambito_hmac', 'huella_hmac'
            ])
            OR pg_catalog.jsonb_typeof(e.valor -> 'generacion')
               <> 'number'
            OR pg_catalog.jsonb_typeof(e.valor -> 'ambito_hmac')
               <> 'string'
            OR pg_catalog.jsonb_typeof(e.valor -> 'huella_hmac')
               <> 'string'
            OR e.valor ->> 'generacion' !~ '^[1-9][0-9]{0,8}$'
            OR (e.valor ->> 'generacion')::numeric >
               9007199254740991::numeric
            OR e.valor ->> 'ambito_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR e.valor ->> 'huella_hmac' !~
               (
                 '^hmac-sha256:vec[.]contratacion-temporal[.]'
                 || 'huella-peticion/v[1-9][0-9]{0,8}:[a-f0-9]{64}$')
            OR pg_catalog.right(e.valor ->> 'ambito_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR pg_catalog.right(e.valor ->> 'huella_hmac', 64) =
               pg_catalog.repeat('0', 64)
            OR substring(
                 e.valor ->> 'ambito_hmac'
                 FROM '/v([1-9][0-9]{0,8}):')::integer <> (e.valor ->> 'generacion')::integer
            OR substring(
                 e.valor ->> 'huella_hmac'
                 FROM '/v([1-9][0-9]{0,8}):')::integer <> (e.valor ->> 'generacion')::integer) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'sellos HMAC inválidos';
    END IF;
    SELECT pg_catalog.array_agg(
               (e.valor ->> 'generacion')::integer ORDER BY e.orden),
           pg_catalog.array_agg(
               e.valor ->> 'ambito_hmac' ORDER BY e.orden)
      INTO v_generaciones, v_aliases
      FROM pg_catalog.jsonb_array_elements(v_pares)
           WITH ORDINALITY AS e(valor, orden);
    SELECT pg_catalog.array_agg(
               generacion ORDER BY posicion)
      INTO v_generaciones_politica
      FROM vec_contratacion_temporal.politica_generaciones_hmac_alta;
    IF v_generaciones IS DISTINCT FROM v_generaciones_politica
       OR pg_catalog.cardinality(v_generaciones) <>
          pg_catalog.cardinality(
              ARRAY(SELECT DISTINCT x FROM pg_catalog.unnest(
                  v_generaciones) AS u(x))) THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'política HMAC no satisfecha';
    END IF;
    v_activo_ambito := s #>> '{activo,ambito_hmac}';
    v_activo_huella := s #>> '{activo,huella_hmac}';
    v_huella_alta := pg_catalog.encode(
        pg_catalog.sha256(p_alta_canonica), 'hex');
    -- Cierra la ligadura completa del recurso que originó la decisión V3.
    v_contexto_recurso := pg_catalog.convert_to(
        '{"ambitos":{"categoria_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{solicitud,categoria_ref}') ||
        ',"centro_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{solicitud,centro_ref}') ||
        ',"organizacion_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'organizacion_ref') ||
        '},"atributos":{"efecto_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_huella_alta) ||
        ',"flujo_huella_sha256":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,huella_sha256}') ||
        ',"flujo_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,definicion_ref}') ||
        ',"flujo_version":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a #>> '{flujo,version}') ||
        ',"huella_peticion_hmac_activa":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_activo_huella) ||
        '}}',
        'UTF8');
    v_huella_contexto_recurso := pg_catalog.encode(
        pg_catalog.sha256(v_contexto_recurso), 'hex');
    IF d ->> 'recurso_ref' <> v_activo_ambito
       OR d ->> 'modulo_id' <> 'contratacion_temporal'
       OR d ->> 'tipo_recurso' <> 'expediente_contratacion_temporal'
       OR d ->> 'accion' <> 'contratacion_temporal.solicitud.crear'
       OR d ->> 'finalidad' <> 'tramitar_necesidad_personal_temporal'
       OR d ->> 'contexto_recurso_huella_sha256' <>
          v_huella_contexto_recurso THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'efecto de alta no autorizado';
    END IF;
    SELECT * INTO STRICT v_consumo
      FROM vec_autorizacion_atestada_v3.
           registrar_y_consumir_decision_v3_atestada(
               p_capacidad_canonica, p_decision_canonica,
               p_motivo_canonico, p_contexto_actor_canonico,
               p_persona_version, p_perfil_version,
               p_payload_vec_ad_3, p_sobre_cose_sign1,
               p_evidencia_verificacion, p_raiz_publica_spki);
    IF v_consumo.efecto_ref <> v_activo_ambito
       OR v_consumo.huella_efecto_sha256 <>
          v_huella_contexto_recurso
       OR v_consumo.consumo_nuevo IS NULL THEN
        RAISE EXCEPTION USING
            ERRCODE = '42501',
            MESSAGE = 'consumo de alta incoherente';
    END IF;
    v_confirmacion_ref := 'cnf_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.decision_ref
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref'
            ) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.consumo_huella_sha256
            )
        ), 'hex'), 1, 32
    );
    IF v_consumo.consumo_nuevo IS FALSE THEN
        RETURN QUERY
        SELECT *
          FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
              p_alta_canonica, v_activo_ambito, v_activo_huella,
              v_consumo.decision_ref, v_consumo.efecto_ref,
              v_consumo.huella_efecto_sha256,
              v_consumo.consumo_huella_sha256
          );
        RETURN;
    END IF;
    -- Orden total de locks de alias para que todas las sesiones converjan.
    PERFORM pg_catalog.pg_advisory_xact_lock(
        pg_catalog.hashtextextended('vec_ct:alias:' || alias, 0))
      FROM pg_catalog.unnest(v_aliases) AS u(alias)
     ORDER BY alias COLLATE "C";
    SELECT pg_catalog.array_agg(
               DISTINCT ambito_raiz_hmac ORDER BY ambito_raiz_hmac)
      INTO v_raices
      FROM vec_contratacion_temporal.alias_ambito_alta
     WHERE alias_hmac = ANY (v_aliases);
    IF pg_catalog.cardinality(v_raices) > 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'alias HMAC divergentes';
    END IF;
    IF pg_catalog.cardinality(v_raices) = 1 THEN
        v_raiz := v_raices[1];
    ELSE
        v_raiz := v_activo_ambito;
        INSERT INTO vec_contratacion_temporal.identidad_reserva_alta (
            ambito_hmac, reserva_ref, expediente_ref, numero_visible,
            recibo_ref, huella_peticion_hmac, organizacion_ref,
            actor_ref, perfil_ref, creada_en) VALUES (
            v_raiz, a ->> 'reserva_ref', a ->> 'expediente_ref',
            a ->> 'numero_visible', a ->> 'recibo_ref',
            v_activo_huella, a ->> 'organizacion_ref',
            a ->> 'actor_ref', a ->> 'perfil_ref',
            (a #>> '{actuacion,realizada_en}')::timestamptz);
        INSERT INTO vec_contratacion_temporal.reserva_alta_version (
            ambito_hmac, revision, estado, registrada_en) VALUES (v_raiz, 1, 'reservada', clock_timestamp());
        INSERT INTO vec_contratacion_temporal.reserva_alta_actual
            VALUES (v_raiz, 1);
    END IF;
    SELECT * INTO STRICT v_identidad
      FROM vec_contratacion_temporal.identidad_reserva_alta
     WHERE ambito_hmac = v_raiz;
    IF v_identidad.reserva_ref <> a ->> 'reserva_ref'
       OR v_identidad.expediente_ref <> a ->> 'expediente_ref'
       OR v_identidad.numero_visible <> a ->> 'numero_visible'
       OR v_identidad.recibo_ref <> a ->> 'recibo_ref'
       OR v_identidad.organizacion_ref <> a ->> 'organizacion_ref'
       OR v_identidad.actor_ref <> a ->> 'actor_ref'
       OR v_identidad.perfil_ref <> a ->> 'perfil_ref' THEN
        RAISE EXCEPTION USING
            ERRCODE = '23505',
            MESSAGE = 'reserva de alta en conflicto';
    END IF;
    FOR v_par IN
        SELECT e.valor
          FROM pg_catalog.jsonb_array_elements(v_pares)
               WITH ORDINALITY AS e(valor, orden)
         ORDER BY e.orden
    LOOP
        IF EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_ambito_alta x
             WHERE x.alias_hmac = v_par ->> 'ambito_hmac'
               AND x.ambito_raiz_hmac <> v_raiz) OR EXISTS (
            SELECT 1
              FROM vec_contratacion_temporal.alias_huella_alta x
             WHERE x.ambito_raiz_hmac = v_raiz
               AND x.generacion =
                   (v_par ->> 'generacion')::integer
               AND x.alias_hmac <> v_par ->> 'huella_hmac') THEN
            RAISE EXCEPTION USING
                ERRCODE = '23505',
                MESSAGE = 'par HMAC en conflicto';
        END IF;
        INSERT INTO vec_contratacion_temporal.alias_ambito_alta (
            alias_hmac, ambito_raiz_hmac, generacion, registrada_en) VALUES (
            v_par ->> 'ambito_hmac', v_raiz,
            (v_par ->> 'generacion')::integer, clock_timestamp()) ON CONFLICT DO NOTHING;
        INSERT INTO vec_contratacion_temporal.alias_huella_alta (
            ambito_raiz_hmac, generacion, alias_hmac, registrada_en) VALUES (
            v_raiz, (v_par ->> 'generacion')::integer,
            v_par ->> 'huella_hmac', clock_timestamp()) ON CONFLICT DO NOTHING;
    END LOOP;
    IF EXISTS (
        SELECT 1 FROM vec_contratacion_temporal.expediente_alta e
         WHERE e.expediente_ref = a ->> 'expediente_ref'
    ) OR EXISTS (
        SELECT 1
          FROM vec_contratacion_temporal.confirmacion_agregado_alta m
         WHERE m.confirmacion_ref = v_confirmacion_ref
    ) THEN
        RAISE EXCEPTION USING
            ERRCODE = 'V2070',
            MESSAGE = 'estado previo de alta incoherente';
    END IF;
    v_ahora := pg_catalog.date_trunc(
        'microseconds', clock_timestamp());
    IF v_ahora < (a #>> '{actuacion,realizada_en}')::timestamptz THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'instante de alta inválido';
    END IF;
    v_auditoria_ref := 'aud_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.decision_ref) ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref')), 'hex'), 1, 32);
    v_evento_ref := 'evt_ct_' || pg_catalog.substr(
        pg_catalog.encode(pg_catalog.sha256(
            vec_contratacion_temporal.encuadrar_texto_v1(
                a ->> 'expediente_ref') ||
            vec_contratacion_temporal.encuadrar_texto_v1(
                v_consumo.consumo_huella_sha256)), 'hex'), 1, 32);
    INSERT INTO vec_contratacion_temporal.expediente_alta (
        expediente_ref, reserva_ref, numero_visible, organizacion_ref,
        actor_ref, perfil_ref, decision_ref, efecto_ref,
        huella_efecto_sha256, creada_en, confirmacion_ref) VALUES (
        a ->> 'expediente_ref', a ->> 'reserva_ref',
        a ->> 'numero_visible', a ->> 'organizacion_ref',
        a ->> 'actor_ref', a ->> 'perfil_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        (a ->> 'creado_en')::timestamptz, v_confirmacion_ref);
    INSERT INTO vec_contratacion_temporal.expediente_alta_version (
        expediente_ref, version, alta_canonica, huella_alta_sha256,
        flujo_ref, flujo_version, flujo_huella_sha256, fase_clave,
        estado, solicitud_huella_sha256, registrada_en,
        confirmacion_ref) VALUES (
        a ->> 'expediente_ref', 1, p_alta_canonica, v_huella_alta,
        a #>> '{flujo,definicion_ref}',
        (a #>> '{flujo,version}')::numeric,
        a #>> '{flujo,huella_sha256}', a ->> 'fase_actual',
        a ->> 'estado_actual',
        pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(
                a -> 'solicitud'), 'UTF8')), 'hex'),
        v_ahora, v_confirmacion_ref);
    v_huella_actuacion := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref'
        ) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            (a #> '{actuacion}')::text
        )
    ), 'hex');
    INSERT INTO vec_contratacion_temporal.actuacion_alta (
        expediente_ref, secuencia, version_expediente, accion_clave,
        actor_ref, unidad_ref, recibo_ref, fase_destino,
        estado_destino, realizada_en, huella_sha256,
        confirmacion_ref) VALUES (
        a ->> 'expediente_ref', 1, 1,
        a #>> '{actuacion,accion_clave}',
        a #>> '{actuacion,actor_ref}',
        a #>> '{actuacion,unidad_ref}',
        a #>> '{actuacion,recibo_ref}',
        a #>> '{actuacion,fase_destino}',
        a #>> '{actuacion,estado_destino}',
        (a #>> '{actuacion,realizada_en}')::timestamptz,
        v_huella_actuacion, v_confirmacion_ref);
    SELECT secuencia_auditoria, cabeza_auditoria_sha256,
           secuencia_outbox, cabeza_outbox_sha256
      INTO STRICT v_secuencia_auditoria, v_anterior_auditoria,
                  v_secuencia_outbox, v_anterior_outbox
     FROM vec_contratacion_temporal.control_cadenas_alta
     WHERE control_id
     FOR UPDATE;
    IF v_secuencia_auditoria >= 9007199254740991::numeric
       OR v_secuencia_outbox >= 9007199254740991::numeric THEN
        RAISE EXCEPTION USING
            ERRCODE = '22003',
            MESSAGE = 'límite de secuencia alcanzado';
    END IF;
    v_secuencia_auditoria := v_secuencia_auditoria + 1;
    v_secuencia_outbox := v_secuencia_outbox + 1;
    v_huella_auditoria := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_secuencia_auditoria::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_anterior_auditoria) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_consumo.decision_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_consumo.consumo_huella_sha256) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_alta)), 'hex');
    INSERT INTO vec_contratacion_temporal.auditoria_alta (
        auditoria_ref, secuencia, expediente_ref, decision_ref,
        consumo_huella_sha256, anterior_sha256, huella_sha256,
        registrada_en, confirmacion_ref) VALUES (
        v_auditoria_ref, v_secuencia_auditoria,
        a ->> 'expediente_ref', v_consumo.decision_ref,
        v_consumo.consumo_huella_sha256, v_anterior_auditoria,
        v_huella_auditoria, v_ahora, v_confirmacion_ref);
    v_payload_outbox := pg_catalog.convert_to(
        '{"esquema":"vec.contratacion-temporal.evento-expediente-registrado.v1"' ||
        ',"evento_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(v_evento_ref) ||
        ',"expediente_ref":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              a ->> 'expediente_ref') ||
        ',"version":1,"ocurrido_en":' ||
          vec_contratacion_temporal.texto_json_go_v1(
              vec_contratacion_temporal.instante_utc_v1(v_ahora)) || '}',
        'UTF8');
    v_huella_payload := pg_catalog.encode(
        pg_catalog.sha256(v_payload_outbox), 'hex');
    v_huella_outbox := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_secuencia_outbox::text) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            v_anterior_outbox) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_huella_payload)), 'hex');
    INSERT INTO vec_contratacion_temporal.outbox_alta (
        evento_ref, secuencia, expediente_ref, tipo_evento,
        payload_canonico, payload_huella_sha256, anterior_sha256,
        huella_sha256, registrada_en, confirmacion_ref) VALUES (
        v_evento_ref, v_secuencia_outbox, a ->> 'expediente_ref',
        'contratacion_temporal.expediente.registrado.v1',
        v_payload_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora, v_confirmacion_ref);
    UPDATE vec_contratacion_temporal.control_cadenas_alta
       SET secuencia_auditoria = v_secuencia_auditoria,
           cabeza_auditoria_sha256 = v_huella_auditoria,
           secuencia_outbox = v_secuencia_outbox,
           cabeza_outbox_sha256 = v_huella_outbox,
           actualizada_en = v_ahora
     WHERE control_id;
    SELECT revision INTO STRICT v_revision
      FROM vec_contratacion_temporal.reserva_alta_actual
     WHERE ambito_hmac = v_raiz
     FOR UPDATE;
    v_revision := v_revision + 1;
    INSERT INTO vec_contratacion_temporal.reserva_alta_version (
        ambito_hmac, revision, estado, version_expediente,
        auditoria_ref, evento_ref, confirmada_en, registrada_en,
        confirmacion_ref) VALUES (
        v_raiz, v_revision, 'confirmada', 1,
        v_auditoria_ref, v_evento_ref, v_ahora, v_ahora,
        v_confirmacion_ref);
    UPDATE vec_contratacion_temporal.reserva_alta_actual
       SET revision = v_revision
     WHERE ambito_hmac = v_raiz;
    v_recibo_huella := pg_catalog.encode(pg_catalog.sha256(
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'expediente_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            a ->> 'numero_visible') ||
        vec_contratacion_temporal.encuadrar_texto_v1('1') ||
        vec_contratacion_temporal.encuadrar_texto_v1(a ->> 'recibo_ref') ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_auditoria_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(v_evento_ref) ||
        vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(v_ahora))), 'hex');
    v_huella_agregado :=
      vec_contratacion_temporal.huella_prueba_agregado_alta_v1(
        VARIADIC ARRAY[
          'vec.contratacion-temporal.confirmacion-agregado-alta.v1',
          v_confirmacion_ref, v_raiz, v_revision::text,
          a ->> 'reserva_ref', a ->> 'expediente_ref',
          a ->> 'numero_visible', a ->> 'recibo_ref',
          v_consumo.decision_ref, v_consumo.efecto_ref,
          v_consumo.huella_efecto_sha256,
          v_consumo.consumo_huella_sha256, '1', v_huella_alta, '1',
          v_huella_actuacion, v_auditoria_ref,
          v_secuencia_auditoria::text, v_anterior_auditoria,
          v_huella_auditoria, v_evento_ref, v_secuencia_outbox::text,
          v_huella_payload, v_anterior_outbox, v_huella_outbox,
          vec_contratacion_temporal.instante_utc_v1(v_ahora),
          v_recibo_huella
        ]
      );
    INSERT INTO vec_contratacion_temporal.confirmacion_agregado_alta (
        confirmacion_ref, agregado_huella_sha256, ambito_hmac,
        reserva_revision, reserva_ref, expediente_ref, numero_visible,
        recibo_ref, decision_ref, efecto_ref, huella_efecto_sha256,
        consumo_huella_sha256, version_expediente, huella_alta_sha256,
        actuacion_secuencia, actuacion_huella_sha256, auditoria_ref,
        auditoria_secuencia, auditoria_anterior_sha256,
        auditoria_huella_sha256, evento_ref, outbox_secuencia,
        payload_huella_sha256, outbox_anterior_sha256,
        outbox_huella_sha256, confirmada_en, recibo_huella_sha256,
        creada_en
    ) VALUES (
        v_confirmacion_ref, v_huella_agregado, v_raiz, v_revision,
        a ->> 'reserva_ref', a ->> 'expediente_ref',
        a ->> 'numero_visible', a ->> 'recibo_ref',
        v_consumo.decision_ref, v_consumo.efecto_ref,
        v_consumo.huella_efecto_sha256,
        v_consumo.consumo_huella_sha256, 1, v_huella_alta, 1,
        v_huella_actuacion, v_auditoria_ref, v_secuencia_auditoria,
        v_anterior_auditoria, v_huella_auditoria, v_evento_ref,
        v_secuencia_outbox, v_huella_payload, v_anterior_outbox,
        v_huella_outbox, v_ahora, v_recibo_huella, v_ahora
    );
    RETURN QUERY
    SELECT *
      FROM vec_contratacion_temporal.reconciliar_agregado_alta_v1(
          p_alta_canonica, v_activo_ambito, v_activo_huella,
          v_consumo.decision_ref, v_consumo.efecto_ref,
          v_consumo.huella_efecto_sha256,
          v_consumo.consumo_huella_sha256
      );
EXCEPTION
    WHEN invalid_text_representation OR datetime_field_overflow
      OR numeric_value_out_of_range THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'entrada de alta inválida';
END
$function$
;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.materializar_version_inicial_v1(p_expediente_ref text, p_version numeric, p_alta_canonica bytea, p_flujo_ref text, p_flujo_version numeric, p_flujo_huella_sha256 text, p_fase_clave text, p_estado text, p_registrada_en timestamp with time zone)
 RETURNS void
 LANGUAGE plpgsql
 SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
AS $function$
DECLARE
    v_efecto jsonb;
    v_agregado jsonb;
    v_agregado_huella text;
    v_prueba bytea;
    v_operacion_ref text;
    v_organizacion_ref text;
    v_numero_visible text;
BEGIN
    IF p_version <> 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'versión inicial incompatible';
    END IF;
    BEGIN
        v_efecto := pg_catalog.convert_from(p_alta_canonica, 'UTF8')::jsonb;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE EXCEPTION USING
                ERRCODE = '22023',
                MESSAGE = 'alta canónica no interpretable';
    END;
    IF vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(v_efecto)
           IS DISTINCT FROM p_alta_canonica
       OR v_efecto ->> 'expediente_ref' IS DISTINCT FROM p_expediente_ref
       OR (v_efecto ->> 'version')::numeric IS DISTINCT FROM p_version
       OR v_efecto #>> '{flujo,definicion_ref}' IS DISTINCT FROM p_flujo_ref
       OR (v_efecto #>> '{flujo,version}')::numeric
            IS DISTINCT FROM p_flujo_version
       OR v_efecto #>> '{flujo,huella_sha256}'
            IS DISTINCT FROM p_flujo_huella_sha256
       OR v_efecto ->> 'fase_actual' IS DISTINCT FROM p_fase_clave
       OR v_efecto ->> 'estado_actual' IS DISTINCT FROM p_estado
       OR (v_efecto #>> '{actuacion,secuencia}')::numeric <> 1
       OR (v_efecto #>> '{actuacion,version_expediente}')::numeric <> 1 THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'alta no ligada al expediente integral';
    END IF;

    SELECT e.organizacion_ref, e.numero_visible
      INTO STRICT v_organizacion_ref, v_numero_visible
      FROM vec_contratacion_temporal.expediente_alta e
     WHERE e.expediente_ref = p_expediente_ref;
    IF v_efecto ->> 'organizacion_ref'
           IS DISTINCT FROM v_organizacion_ref
       OR v_efecto ->> 'numero_visible'
           IS DISTINCT FROM v_numero_visible THEN
        RAISE EXCEPTION USING
            ERRCODE = '22023',
            MESSAGE = 'identidad de alta incompatible';
    END IF;

    v_agregado := pg_catalog.jsonb_build_object(
        'referencia', p_expediente_ref,
        'organizacion_ref', v_organizacion_ref,
        'numero_visible', v_numero_visible,
        'version', p_version,
        'flujo', v_efecto -> 'flujo',
        'fase_actual', p_fase_clave,
        'estado_actual', p_estado,
        'solicitud', v_efecto -> 'solicitud',
        'creado_en', v_efecto -> 'creado_en',
        'actualizado_en', v_efecto -> 'actualizado_en',
        'actuaciones', pg_catalog.jsonb_build_array(
            v_efecto -> 'actuacion'
        )
    );
    v_agregado_huella := pg_catalog.encode(
        pg_catalog.sha256(
            pg_catalog.convert_to(v_agregado::text, 'UTF8')
        ),
        'hex'
    );
    v_operacion_ref := 'alta:' || (v_efecto ->> 'recibo_ref');
    v_prueba :=
        pg_catalog.convert_to(
            'VEC-CT-EXPEDIENTE-INTEGRAL-V1' || chr(10), 'UTF8'
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_expediente_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_version::text)
        || vec_contratacion_temporal.encuadrar_texto_v1(v_agregado_huella)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_flujo_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_flujo_version::text
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(
            p_flujo_huella_sha256
        )
        || vec_contratacion_temporal.encuadrar_texto_v1(p_fase_clave)
        || vec_contratacion_temporal.encuadrar_texto_v1(p_estado)
        || vec_contratacion_temporal.encuadrar_texto_v1('alta_o2')
        || vec_contratacion_temporal.encuadrar_texto_v1(v_operacion_ref)
        || vec_contratacion_temporal.encuadrar_texto_v1(
            vec_contratacion_temporal.instante_utc_v1(p_registrada_en)
        );

    INSERT INTO vec_contratacion_temporal.expediente_version_integral (
        expediente_ref, version, agregado_json,
        agregado_json_huella_sha256, prueba_canonica,
        prueba_huella_sha256, flujo_ref, flujo_version,
        flujo_huella_sha256, fase_clave, estado, origen_version,
        operacion_ref, registrada_en
    ) VALUES (
        p_expediente_ref, p_version, v_agregado,
        v_agregado_huella, v_prueba,
        pg_catalog.encode(pg_catalog.sha256(v_prueba), 'hex'),
        p_flujo_ref, p_flujo_version, p_flujo_huella_sha256,
        p_fase_clave, p_estado, 'alta_o2', v_operacion_ref,
        p_registrada_en
    );
    INSERT INTO vec_contratacion_temporal.expediente_integral_actual (
        expediente_ref, version, actualizada_en, operacion_ref
    ) VALUES (
        p_expediente_ref, p_version, p_registrada_en, v_operacion_ref
    );
END
$function$
;

-- Las tres funciones históricas conservan firma, owner y ACL; sólo prosrc y
-- search_path (añadiendo pg_temp al final) pueden cambiar.
DO $postimagen_ct193$
DECLARE item record; f oid; metadatos jsonb; actual_meta jsonb;
 actual_config text[]; actual_huella text;
BEGIN
 metadatos:=pg_catalog.current_setting('vec_ct193.pre_meta',true)::jsonb;
 IF pg_catalog.jsonb_typeof(metadatos) IS DISTINCT FROM 'object' OR
    (SELECT pg_catalog.count(*) FROM pg_catalog.jsonb_object_keys(metadatos))<>4 THEN
   RAISE EXCEPTION 'CT193: falta preimagen de metadatos' USING ERRCODE='55000';
 END IF;
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)','c14f8bf5628d1b24c87bd31b0d560159a043718132796f7ca6ce32d3a524489b',ARRAY['search_path=pg_catalog, pg_temp']::text[]),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea,bytea)','35d99b5e553f95ca06a67c3ae11854eb842482c201b315edab3c6081ffc9a218',ARRAY['search_path=pg_catalog, pg_temp','lock_timeout=2s']::text[]),
  ('vec_contratacion_temporal.materializar_version_inicial_v1(text,numeric,bytea,text,numeric,text,text,text,timestamp with time zone)','d6670161f8dcae0b0da0625569e35174c910cd237b502b4a793219194f768414',ARRAY['search_path=pg_catalog, pg_temp']::text[]),
  ('vec_contratacion_temporal.reconstruir_efecto_alta_v2(jsonb)','6143987ae2c1129902c3921c013eb23344a64009d448705010066d676f680149',ARRAY['search_path=pg_catalog']::text[]),
  ('vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb)','2bf5d8591ccace7beb46d1004be617285e6274912d2383e7d28f1d333d61fece',NULL::text[])
 ) AS esperado(firma,huella,configuracion) LOOP
   f:=pg_catalog.to_regprocedure(item.firma);
   IF f IS NULL THEN
     RAISE EXCEPTION 'CT193: postimagen ausente firma=% esperado=% actual=ausente',
       item.firma,item.huella USING ERRCODE='55000';
   END IF;
   actual_huella:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
       pg_catalog.pg_get_functiondef(f),'UTF8')),'hex');
   IF actual_huella IS DISTINCT FROM item.huella THEN
     RAISE EXCEPTION 'CT193: postimagen firma=% esperado=% actual=%',
       item.firma,item.huella,actual_huella USING ERRCODE='55000';
   END IF;
   IF item.firma <> 'vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb)' THEN
     SELECT pg_catalog.to_jsonb(p)-'prosrc',p.proconfig
       INTO actual_meta,actual_config FROM pg_catalog.pg_proc p WHERE p.oid=f;
     IF actual_config IS DISTINCT FROM item.configuracion THEN
       RAISE EXCEPTION 'CT193: configuración firma=% esperado=% actual=%',
         item.firma,item.configuracion,actual_config USING ERRCODE='55000';
     END IF;
     IF actual_meta-'proconfig' IS DISTINCT FROM (metadatos->item.firma)-'proconfig' THEN
       RAISE EXCEPTION 'CT193: metadatos firma=% esperado_sha=% actual_sha=%',
         item.firma,
         pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           ((metadatos->item.firma)-'proconfig')::text,'UTF8')),'hex'),
         pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
           (actual_meta-'proconfig')::text,'UTF8')),'hex') USING ERRCODE='55000';
     END IF;
   END IF;
 END LOOP;
END $postimagen_ct193$;

-- Lectura para replay de un alta E3 ya confirmada. El ámbito HMAC es opaco;
-- los tres identificadores se cotejan con la identidad persistida. El rol
-- técnico sólo debe invocarla tras resolver la identidad en la frontera.
CREATE FUNCTION vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
 p_ambito_hmac text, p_organizacion_ref text, p_actor_ref text,
 p_perfil_ref text
) RETURNS TABLE(estado text, instantanea bytea)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog,pg_temp SET row_security=on SET TimeZone='UTC'
SET statement_timeout='15s' AS $funcion$
DECLARE
 v_raiz text;
 v_identidad record;
 v_confirmacion record;
 v_expediente record;
 v_version record;
 v_efecto jsonb;
 v_huella_recibo text;
 v_huella_solicitud text;
BEGIN
 IF session_user=current_user
    OR NOT pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_ejecutor','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_migrador','MEMBER')
    OR pg_catalog.pg_has_role(session_user,'vec_contratacion_temporal_propietario','MEMBER')
    OR pg_catalog.current_setting('transaction_isolation')<>'serializable'
    OR p_ambito_hmac IS NULL
    OR p_ambito_hmac !~ '^hmac-sha256:vec[.]contratacion-temporal[.]ambito-idempotencia/v[1-9][0-9]{0,8}:[a-f0-9]{64}$'
    OR pg_catalog.right(p_ambito_hmac,64)=pg_catalog.repeat('0',64)
    OR p_organizacion_ref IS NULL
    OR p_organizacion_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_actor_ref IS NULL
    OR p_actor_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$'
    OR p_perfil_ref IS NULL
    OR p_perfil_ref !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
   RAISE EXCEPTION 'lectura de instantánea denegada' USING ERRCODE='42501';
 END IF;
 SELECT a.ambito_raiz_hmac INTO v_raiz
   FROM vec_contratacion_temporal.alias_ambito_alta a
  WHERE a.alias_hmac=p_ambito_hmac;
 IF NOT FOUND THEN
   RETURN QUERY SELECT 'ausente'::text,NULL::bytea;
   RETURN;
 END IF;
 SELECT * INTO v_identidad
   FROM vec_contratacion_temporal.identidad_reserva_alta i
  WHERE i.ambito_hmac=v_raiz;
 IF NOT FOUND THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 IF v_identidad.organizacion_ref IS DISTINCT FROM p_organizacion_ref
    OR v_identidad.actor_ref IS DISTINCT FROM p_actor_ref
    OR v_identidad.perfil_ref IS DISTINCT FROM p_perfil_ref THEN
   RAISE EXCEPTION 'lectura de instantánea denegada' USING ERRCODE='42501';
 END IF;
 SELECT * INTO v_confirmacion
   FROM vec_contratacion_temporal.confirmacion_agregado_alta m
  WHERE m.ambito_hmac=v_raiz;
 IF NOT FOUND THEN
   IF EXISTS (SELECT 1 FROM vec_contratacion_temporal.expediente_alta e
               WHERE e.expediente_ref=v_identidad.expediente_ref) THEN
     RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
   END IF;
   RETURN QUERY SELECT 'ausente'::text,NULL::bytea;
   RETURN;
 END IF;
 SELECT * INTO v_expediente
   FROM vec_contratacion_temporal.expediente_alta e
  WHERE e.expediente_ref=v_confirmacion.expediente_ref;
 IF NOT FOUND THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 SELECT * INTO v_version
   FROM vec_contratacion_temporal.expediente_alta_version v
  WHERE v.expediente_ref=v_confirmacion.expediente_ref AND v.version=1;
 IF NOT FOUND THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 v_huella_recibo:=pg_catalog.encode(pg_catalog.sha256(
   vec_contratacion_temporal.encuadrar_texto_v1(v_confirmacion.expediente_ref) ||
   vec_contratacion_temporal.encuadrar_texto_v1(v_confirmacion.numero_visible) ||
   vec_contratacion_temporal.encuadrar_texto_v1('1') ||
   vec_contratacion_temporal.encuadrar_texto_v1(v_confirmacion.recibo_ref) ||
   vec_contratacion_temporal.encuadrar_texto_v1(v_confirmacion.auditoria_ref) ||
   vec_contratacion_temporal.encuadrar_texto_v1(v_confirmacion.evento_ref) ||
   vec_contratacion_temporal.encuadrar_texto_v1(
     vec_contratacion_temporal.instante_utc_v1(v_confirmacion.confirmada_en))
 ),'hex');
 IF v_confirmacion.expediente_ref IS DISTINCT FROM v_identidad.expediente_ref
    OR v_confirmacion.reserva_ref IS DISTINCT FROM v_identidad.reserva_ref
    OR v_confirmacion.numero_visible IS DISTINCT FROM v_identidad.numero_visible
    OR v_confirmacion.recibo_ref IS DISTINCT FROM v_identidad.recibo_ref
    OR v_confirmacion.version_expediente IS DISTINCT FROM 1
    OR v_confirmacion.recibo_huella_sha256 IS DISTINCT FROM v_huella_recibo
    OR v_expediente.reserva_ref IS DISTINCT FROM v_confirmacion.reserva_ref
    OR v_expediente.numero_visible IS DISTINCT FROM v_confirmacion.numero_visible
    OR v_expediente.organizacion_ref IS DISTINCT FROM v_identidad.organizacion_ref
    OR v_expediente.actor_ref IS DISTINCT FROM v_identidad.actor_ref
    OR v_expediente.perfil_ref IS DISTINCT FROM v_identidad.perfil_ref
    OR v_expediente.confirmacion_ref IS DISTINCT FROM v_confirmacion.confirmacion_ref
    OR v_version.confirmacion_ref IS DISTINCT FROM v_confirmacion.confirmacion_ref
    OR v_version.huella_alta_sha256 IS DISTINCT FROM v_confirmacion.huella_alta_sha256
    OR v_version.huella_alta_sha256 IS DISTINCT FROM
       pg_catalog.encode(pg_catalog.sha256(v_version.alta_canonica),'hex') THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 BEGIN
   v_efecto:=pg_catalog.convert_from(v_version.alta_canonica,'UTF8')::jsonb;
 EXCEPTION WHEN OTHERS THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END;
 BEGIN
 IF v_efecto->>'reserva_ref' IS DISTINCT FROM v_confirmacion.reserva_ref
    OR v_efecto->>'expediente_ref' IS DISTINCT FROM v_confirmacion.expediente_ref
    OR v_efecto->>'numero_visible' IS DISTINCT FROM v_confirmacion.numero_visible
    OR v_efecto->>'recibo_ref' IS DISTINCT FROM v_confirmacion.recibo_ref
    OR v_efecto->>'organizacion_ref' IS DISTINCT FROM v_identidad.organizacion_ref
    OR v_efecto->>'actor_ref' IS DISTINCT FROM v_identidad.actor_ref
    OR v_efecto->>'perfil_ref' IS DISTINCT FROM v_identidad.perfil_ref
    OR v_efecto#>>'{flujo,definicion_ref}' IS DISTINCT FROM v_version.flujo_ref
    OR (v_efecto#>>'{flujo,version}')::numeric IS DISTINCT FROM v_version.flujo_version
    OR v_efecto#>>'{flujo,huella_sha256}' IS DISTINCT FROM v_version.flujo_huella_sha256
    OR v_efecto->>'fase_actual' IS DISTINCT FROM v_version.fase_clave
    OR v_efecto->>'estado_actual' IS DISTINCT FROM v_version.estado
    OR (v_efecto->>'creado_en')::timestamptz IS DISTINCT FROM v_expediente.creada_en
    OR v_version.registrada_en IS DISTINCT FROM v_confirmacion.confirmada_en THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 v_huella_solicitud:=pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
   vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(
      v_efecto->'solicitud'),'UTF8')),'hex');
 IF v_version.solicitud_huella_sha256 IS DISTINCT FROM v_huella_solicitud THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 IF v_efecto->>'esquema'='vec.contratacion-temporal.efecto-alta.v2' THEN
   IF vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_efecto)
      IS DISTINCT FROM v_version.alta_canonica THEN
     RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
   END IF;
   RETURN QUERY SELECT 'legado_v2'::text,NULL::bytea;
   RETURN;
 END IF;
 IF v_efecto->>'esquema' IS DISTINCT FROM 'vec.contratacion-temporal.efecto-alta.v3'
    OR vec_contratacion_temporal.reconstruir_efecto_alta_v3(v_efecto)
       IS DISTINCT FROM v_version.alta_canonica
    OR vec_contratacion_temporal.necesidad_alta_valida_v3(v_efecto->'solicitud') IS NOT TRUE THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END IF;
 RETURN QUERY SELECT 'confirmada_v3'::text,
   pg_catalog.decode(v_efecto#>>'{solicitud,necesidad,catalogo_instantanea}','base64');
 EXCEPTION WHEN OTHERS THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END;
END $funcion$;

REVOKE ALL ON FUNCTION
 vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb),
 vec_contratacion_temporal.reconstruir_periodo_necesidad_v3(jsonb),
 vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(jsonb),
 vec_contratacion_temporal.reconstruir_efecto_alta_v3(jsonb),
 vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3(jsonb),
 vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(jsonb),
 vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)
 FROM PUBLIC, vec_contratacion_temporal_ejecutor;
GRANT EXECUTE ON FUNCTION
 vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)
 TO vec_contratacion_temporal_ejecutor;
COMMIT;
