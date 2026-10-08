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

-- Clonar sólo la estructura superior E2; la diferencia es el serializador de
-- solicitud. Ningún byte E2 ni su función instalada se modifica.
DO $clonar_efecto$
DECLARE
 f oid := pg_catalog.to_regprocedure('vec_contratacion_temporal.reconstruir_efecto_alta_v2(jsonb)');
 d text;
 n text;
 m1 text := 'vec_contratacion_temporal.reconstruir_efecto_alta_v2(';
 m2 text := 'vec_contratacion_temporal.reconstruir_solicitud_efecto_v2(';
BEGIN
 SELECT pg_catalog.pg_get_functiondef(f) INTO STRICT d;
 IF (pg_catalog.length(d)-pg_catalog.length(pg_catalog.replace(d,m1,'')))<>pg_catalog.length(m1)
    OR (pg_catalog.length(d)-pg_catalog.length(pg_catalog.replace(d,m2,'')))<>pg_catalog.length(m2) THEN
   RAISE EXCEPTION 'CT193: marcas del efecto E2 incompatibles' USING ERRCODE='55000';
 END IF;
 n:=pg_catalog.replace(pg_catalog.replace(d,m1,
   'vec_contratacion_temporal.reconstruir_efecto_alta_v3('),m2,
   'vec_contratacion_temporal.reconstruir_solicitud_efecto_v3(');
 EXECUTE n;
END $clonar_efecto$;

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

-- Sustituciones unívocas sobre la definición REAL instalada. La transacción
-- revierte entera si una marca falta, está duplicada o cambia la ACL.
DO $parchar$
DECLARE
 item record;
 f oid;
 d text;
 n text;
 m text;
 r text;
 meta jsonb;
BEGIN
 FOR item IN SELECT * FROM (VALUES
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
   $m$FROM pg_catalog.jsonb_object_keys(a -> 'solicitud')) <> 10$m$,
   $r$FROM pg_catalog.jsonb_object_keys(a -> 'solicitud')) <>
       (CASE WHEN a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
             THEN 11 ELSE 10 END)$r$),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
   $m$'periodo', 'rc', 'documentos_adjuntos') THEN false$m$,
   $r$'periodo', 'rc', 'documentos_adjuntos', 'necesidad') THEN false$r$),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
   $m$OR vec_contratacion_temporal.reconstruir_efecto_alta_v2(a)$m$,
   $r$OR vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(a)$r$),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
   $m$OR a ->> 'esquema' <>
          'vec.contratacion-temporal.efecto-alta.v2'$m$,
   $r$OR a ->> 'esquema' NOT IN (
          'vec.contratacion-temporal.efecto-alta.v2',
          'vec.contratacion-temporal.efecto-alta.v3')
       OR (a ->> 'esquema' = 'vec.contratacion-temporal.efecto-alta.v3'
           AND vec_contratacion_temporal.necesidad_alta_valida_v3(a -> 'solicitud') IS NOT TRUE)$r$),
  ('vec_contratacion_temporal.confirmar_alta_atestada_v1(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea,bytea,bytea)',
   $m$vec_contratacion_temporal.reconstruir_solicitud_efecto_v2($m$,
   $r$vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3($r$),
  ('vec_contratacion_temporal.reconciliar_agregado_alta_v1(bytea,text,text,text,text,text,text)',
   $m$vec_contratacion_temporal.reconstruir_solicitud_efecto_v2($m$,
   $r$vec_contratacion_temporal.reconstruir_solicitud_segun_esquema_v3($r$),
  ('vec_contratacion_temporal.materializar_version_inicial_v1(text,numeric,bytea,text,numeric,text,text,text,timestamp with time zone)',
   $m$vec_contratacion_temporal.reconstruir_efecto_alta_v2(v_efecto)$m$,
   $r$vec_contratacion_temporal.reconstruir_efecto_segun_esquema_v3(v_efecto)$r$)
 ) AS p(firma,marca,reemplazo) LOOP
   f:=pg_catalog.to_regprocedure(item.firma);
   IF f IS NULL THEN RAISE EXCEPTION 'CT193: función ausente %',item.firma USING ERRCODE='55000'; END IF;
   d:=pg_catalog.pg_get_functiondef(f); m:=item.marca; r:=item.reemplazo;
   SELECT pg_catalog.to_jsonb(p)-'prosrc' INTO STRICT meta
     FROM pg_catalog.pg_proc p WHERE p.oid=f;
   IF (SELECT p.proowner FROM pg_catalog.pg_proc p WHERE p.oid=f)
       IS DISTINCT FROM 'vec_contratacion_temporal_propietario'::regrole THEN
     RAISE EXCEPTION 'CT193: propietario incompatible %',item.firma USING ERRCODE='55000';
   END IF;
   IF (pg_catalog.length(d)-pg_catalog.length(pg_catalog.replace(d,m,'')))<>pg_catalog.length(m)
      OR pg_catalog.strpos(d,r)<>0 THEN
     RAISE EXCEPTION 'CT193: marca incompatible %',item.firma USING ERRCODE='55000';
   END IF;
   n:=pg_catalog.replace(d,m,r);
   EXECUTE n;
   IF pg_catalog.pg_get_functiondef(f) IS DISTINCT FROM n THEN
     RAISE EXCEPTION 'CT193: postimagen incompatible %',item.firma USING ERRCODE='55000';
   END IF;
   IF (SELECT pg_catalog.to_jsonb(p)-'prosrc' FROM pg_catalog.pg_proc p WHERE p.oid=f)
      IS DISTINCT FROM meta THEN
     RAISE EXCEPTION 'CT193: metadatos incompatibles %',item.firma USING ERRCODE='55000';
   END IF;
 END LOOP;
END $parchar$;

-- Lectura para replay de un alta E3 ya confirmada. El ámbito HMAC es opaco;
-- los tres identificadores se cotejan con la identidad persistida. El rol
-- técnico sólo debe invocarla tras resolver la identidad en la frontera.
CREATE FUNCTION vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(
 p_ambito_hmac text, p_organizacion_ref text, p_actor_ref text,
 p_perfil_ref text
) RETURNS TABLE(estado text, instantanea bytea)
LANGUAGE plpgsql STABLE SECURITY DEFINER
SET search_path=pg_catalog SET row_security=on SET TimeZone='UTC'
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
