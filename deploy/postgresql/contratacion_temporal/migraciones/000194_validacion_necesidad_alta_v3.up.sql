\set ON_ERROR_STOP on
BEGIN;
SET LOCAL ROLE vec_contratacion_temporal_propietario;
SET LOCAL search_path = pg_catalog;
SET LOCAL timezone = 'UTC';
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '30s';
SELECT pg_catalog.pg_advisory_xact_lock(pg_catalog.hashtextextended(
    'vec_contratacion_temporal:000194:validacion-necesidad-alta-v3', 0));

-- Ambas definiciones se capturaron mediante pg_get_functiondef sobre la
-- instalación postHX + HZ + B85 + B86 + CT193 + B87. No se reconstruye CT193.
DO $preimagen_ct194$
DECLARE
 item record;
 f oid;
 actual_sha text;
 actual_owner text;
 actual_acl text;
 actual_config text;
BEGIN
 FOR item IN SELECT * FROM (VALUES
    ('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)', 'ad2f412578a38d113f61103c4af107a1e5695a8ebdd8b8ee688dd7cf843e5c71', 'vec_contratacion_temporal_propietario', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}', '{search_path=pg_catalog}'),
    ('vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)', '675854b375f8e9b32aa4bf687944b8fb2219f74defacf187f709302f457fc5d5', 'vec_contratacion_temporal_propietario', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}', '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,statement_timeout=15s}')
 ) AS v(firma,huella,propietario,acl,configuracion) LOOP
   f:=pg_catalog.to_regprocedure(item.firma);
   IF f IS NULL THEN
     RAISE EXCEPTION 'CT194: preimagen firma=% esperado=instalada actual=ausente',
       item.firma USING ERRCODE='55000';
   END IF;
   SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            pg_catalog.pg_get_functiondef(f),'UTF8')),'hex'),
          pg_catalog.pg_get_userbyid(p.proowner),p.proacl::text,p.proconfig::text
     INTO actual_sha,actual_owner,actual_acl,actual_config
     FROM pg_catalog.pg_proc p WHERE p.oid=f;
   IF actual_sha IS DISTINCT FROM item.huella THEN
     RAISE EXCEPTION 'CT194: preimagen firma=% esperado_sha=% actual_sha=%',
       item.firma,item.huella,actual_sha USING ERRCODE='55000';
   END IF;
   IF actual_owner IS DISTINCT FROM item.propietario
      OR actual_acl IS DISTINCT FROM item.acl
      OR actual_config IS DISTINCT FROM item.configuracion THEN
     RAISE EXCEPTION 'CT194: metadatos firma=% owner esperado=% actual=% acl esperado=% actual=% config esperado=% actual=%',
       item.firma,item.propietario,actual_owner,item.acl,actual_acl,
       item.configuracion,actual_config USING ERRCODE='55000';
   END IF;
 END LOOP;
END $preimagen_ct194$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.necesidad_alta_valida_v3(s jsonb)
 RETURNS boolean
 LANGUAGE plpgsql
 IMMUTABLE STRICT
 SET search_path TO 'pg_catalog'
AS $function$
DECLARE
 n jsonb;
 c jsonb;
 causa jsonb;
 raw bytea;
 b64 text;
 v_clave text;
 v_valor jsonb;
 v_texto text;
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
   v_texto:=v_valor#>>'{}';
   IF v_clave IN ('titular_ref','financiacion_ref','rc_ref',
                  'intervencion_ref','vacancia_fuente_ref','rpt_catalogo_ref') THEN
     IF v_texto !~ '^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$' THEN
       RETURN false;
     END IF;
   ELSIF v_clave='rpt_catalogo_huella_sha256' THEN
     IF v_texto !~ '^[a-f0-9]{64}$'
        OR v_texto=pg_catalog.repeat('0',64) THEN RETURN false; END IF;
   ELSIF v_clave='porcentaje_financiacion' THEN
     IF v_texto !~ '^([1-9]|[1-9][0-9]|100)$' THEN RETURN false; END IF;
   ELSIF v_clave IN ('justificacion_temporal','programa_denominacion') THEN
     IF pg_catalog.char_length(v_texto)>4000
        OR normalize(v_texto,NFC) IS DISTINCT FROM v_texto
        OR v_texto ~ '(^[[:space:]]|[[:space:]]$)'
        OR pg_catalog.translate(v_texto,E'\t\n','') ~ '[[:cntrl:]]' THEN
       RETURN false;
     END IF;
   ELSIF v_clave NOT IN ('numero_personas','programa_fin') THEN
     IF v_texto !~ '^[A-Za-z0-9][A-Za-z0-9._/-]{0,79}$' THEN
       RETURN false;
     END IF;
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
END $function$;

CREATE OR REPLACE FUNCTION vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(p_ambito_hmac text, p_organizacion_ref text, p_actor_ref text, p_perfil_ref text)
 RETURNS TABLE(estado text, instantanea bytea)
 LANGUAGE plpgsql
 STABLE SECURITY DEFINER
 SET search_path TO 'pg_catalog', 'pg_temp'
 SET row_security TO 'on'
 SET "TimeZone" TO 'UTC'
 SET statement_timeout TO '15s'
AS $function$
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
 EXCEPTION
   WHEN SQLSTATE '40001' THEN RAISE;
   WHEN SQLSTATE '40P01' THEN RAISE;
   WHEN OTHERS THEN
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
 EXCEPTION
   WHEN SQLSTATE '40001' THEN RAISE;
   WHEN SQLSTATE '40P01' THEN RAISE;
   WHEN OTHERS THEN
   RAISE EXCEPTION 'integridad de instantánea no acreditada' USING ERRCODE='55000';
 END;
END $function$;

-- CREATE OR REPLACE mantiene OID, autoría y ACL. La postimagen comprueba
-- explícitamente esos metadatos y el path del lector SECURITY DEFINER.
DO $postimagen_ct194$
DECLARE
 item record;
 f oid;
 actual_sha text;
 actual_owner text;
 actual_acl text;
 actual_config text;
BEGIN
 FOR item IN SELECT * FROM (VALUES
    ('vec_contratacion_temporal.necesidad_alta_valida_v3(jsonb)', 'db5185f959d6fe36f85ff87cc14268dbcc6e36fb14e0e549cf3ed160a0d856d3', 'vec_contratacion_temporal_propietario', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario}', '{search_path=pg_catalog}'),
    ('vec_contratacion_temporal.leer_instantanea_necesidad_alta_v3(text,text,text,text)', 'f2896f419c9248a3c891103092b9284a06d68e0bdf11e22bcabffdee71d09f81', 'vec_contratacion_temporal_propietario', '{vec_contratacion_temporal_propietario=X/vec_contratacion_temporal_propietario,vec_contratacion_temporal_ejecutor=X/vec_contratacion_temporal_propietario}', '{"search_path=pg_catalog, pg_temp",row_security=on,TimeZone=UTC,statement_timeout=15s}')
 ) AS v(firma,huella,propietario,acl,configuracion) LOOP
   f:=pg_catalog.to_regprocedure(item.firma);
   SELECT pg_catalog.encode(pg_catalog.sha256(pg_catalog.convert_to(
            pg_catalog.pg_get_functiondef(f),'UTF8')),'hex'),
          pg_catalog.pg_get_userbyid(p.proowner),p.proacl::text,p.proconfig::text
     INTO actual_sha,actual_owner,actual_acl,actual_config
     FROM pg_catalog.pg_proc p WHERE p.oid=f;
   IF actual_sha IS DISTINCT FROM item.huella THEN
     RAISE EXCEPTION 'CT194: postimagen firma=% esperado_sha=% actual_sha=%',
       item.firma,item.huella,actual_sha USING ERRCODE='55000';
   END IF;
   IF actual_owner IS DISTINCT FROM item.propietario
      OR actual_acl IS DISTINCT FROM item.acl
      OR actual_config IS DISTINCT FROM item.configuracion THEN
     RAISE EXCEPTION 'CT194: postimagen firma=% owner esperado=% actual=% acl esperado=% actual=% config esperado=% actual=%',
       item.firma,item.propietario,actual_owner,item.acl,actual_acl,
       item.configuracion,actual_config USING ERRCODE='55000';
   END IF;
 END LOOP;
END $postimagen_ct194$;
COMMIT;
